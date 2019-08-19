package installation

import (
	"context"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	pkgerr "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"os"
	"sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"time"
)

const (
	defaultInstallationConfigMapName = "integreatly-installation-config"
)

var log = logf.Log.WithName("Installation Controller")

// Add creates a new Installation Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, products []string) error {
	return add(mgr, newReconciler(mgr, products))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, products []string) reconcile.Reconciler {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	restConfig := controllerruntime.GetConfigOrDie()
	return &ReconcileInstallation{
		client:            mgr.GetClient(),
		scheme:            mgr.GetScheme(),
		restConfig:        restConfig,
		productsToInstall: products,
		context:           ctx,
		cancel:            cancel,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("installation-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Installation
	err = c.Watch(&source.Kind{Type: &v1alpha1.Installation{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner Installation
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &v1alpha1.Installation{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileInstallation{}

// ReconcileInstallation reconciles a Installation object
type ReconcileInstallation struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client            client.Client
	scheme            *runtime.Scheme
	restConfig        *rest.Config
	productsToInstall []string
	context           context.Context
	cancel            context.CancelFunc
}

// Reconcile reads that state of the cluster for a Installation object and makes changes based on the state read
// and what is in the Installation.Spec
func (r *ReconcileInstallation) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	instance := &v1alpha1.Installation{}
	err := r.client.Get(r.context, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// if the CR is being deleted,
	// cancel this context to kill all
	// ongoing requests to the API and exit
	if instance.DeletionTimestamp != nil {
		r.cancel() //cancel context
		return reconcile.Result{}, nil
	}

	err, installType := InstallationTypeFactory(instance.Spec.Type, r.productsToInstall)
	if err != nil {
		return reconcile.Result{}, err
	}
	installationCfgMap := os.Getenv("INSTALLATION_CONFIG_MAP")
	if installationCfgMap == "" {
		installationCfgMap = instance.Spec.NamespacePrefix + defaultInstallationConfigMapName
	}

	configManager, err := config.NewManager(r.context, r.client, request.NamespacedName.Namespace, installationCfgMap)
	if err != nil {
		return reconcile.Result{}, err
	}

	// either not checked, or rechecking preflight checks
	if instance.Status.PreflightStatus == v1alpha1.PreflightInProgress ||
		instance.Status.PreflightStatus == v1alpha1.PreflightFail {
		return r.preflightChecks(instance, installType, configManager)
	}

	if instance.Status.Stages == nil {
		instance.Status.Stages = map[string]*v1alpha1.InstallationStageStatus{}
	}

	for _, stage := range installType.GetStages() {
		stageStatus, ok := instance.Status.Stages[stage.Name]
		if !ok {
			//initialise the stage
			stageStatus = &v1alpha1.InstallationStageStatus{
				Phase:    "unprocessed",
				Name:     stage.Name,
				Products: stage.Products,
			}
		}

		err := r.processStage(instance, stageStatus, configManager)
		instance.Status.Stages[stage.Name] = stageStatus
		if err != nil {
			return reconcile.Result{}, err
		}
		//don't move to next stage until current stage is complete
		if stageStatus.Phase != v1alpha1.PhaseCompleted {
			break
		}
	}

	//UPDATE STATUS
	err = r.client.Status().Update(r.context, instance)
	if err != nil {
		// The 'Update' function can error if the resource has been updated by another process and the versions are not correct.
		if k8serr.IsConflict(err) {
			// If there is a conflict, requeue the resource and retry Update
			log.Info("Error updating Installation resource status. Requeue and retry.")
			return reconcile.Result{
				Requeue:      true,
				RequeueAfter: time.Second * 10,
			}, nil
		}

		log.Error(err, "error reconciling installation instance")
		return reconcile.Result{}, err
	}

	//UPDATE OBJECT
	err = r.client.Update(r.context, instance)
	if err != nil {
		// The 'Update' function can error if the resource has been updated by another process and the versions are not correct.
		if k8serr.IsConflict(err) {
			// If there is a conflict, requeue the resource and retry Update
			log.Info("Error updating Installation resource. Requeue and retry.")
			return reconcile.Result{
				Requeue:      true,
				RequeueAfter: time.Second * 10,
			}, nil
		}

		log.Error(err, "error reconciling installation instance")
		return reconcile.Result{}, err
	}

	return reconcile.Result{
		Requeue:      true,
		RequeueAfter: time.Second * 10,
	}, nil
}

func (r *ReconcileInstallation) preflightChecks(installation *v1alpha1.Installation, installationType *Type, configManager *config.Manager) (reconcile.Result, error) {
	result := reconcile.Result{
		Requeue:      true,
		RequeueAfter: time.Second * 10,
	}
	requiredSecrets := []string{"s3-credentials", "s3-bucket"}
	for _, secretName := range requiredSecrets {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: installation.Namespace,
			},
		}
		if exists, err := resources.Exists(r.context, r.client, secret); err != nil {
			return result, err
		} else if !exists {
			installation.Status.PreflightStatus = v1alpha1.PreflightFail
			installation.Status.PreflightMessage = "Could not find " + secret.Name + " secret in integreatly-operator namespace: " + installation.Namespace
			_ = r.client.Status().Update(r.context, installation)
			return result, err
		}
	}

	namespaces := &corev1.NamespaceList{}
	err := r.client.List(r.context, &client.ListOptions{}, namespaces)
	if err != nil {
		// could not list namespaces, keep trying
		logrus.Infof("error listing namespaces, will retry")
		return result, err
	}

	for _, ns := range namespaces.Items {
		products, err := r.checkNamespaceForProducts(ns, installation, installationType, configManager)
		if err != nil {
			// error searching for existing products, keep trying
			logrus.Infof("error looking for existing deployments, will retry")
			return result, err
		}
		if len(products) != 0 {
			//found one or more conflicting products
			installation.Status.PreflightStatus = v1alpha1.PreflightFail
			installation.Status.PreflightMessage = "found conflicting packages: " + strings.Join(products, ", ") + ", in namespace: " + ns.GetName()
			logrus.Infof("found conflicting packages: " + strings.Join(products, ", ") + ", in namespace: " + ns.GetName())
			_ = r.client.Status().Update(r.context, installation)
			return result, err
		}
	}

	installation.Status.PreflightStatus = v1alpha1.PreflightSuccess
	installation.Status.PreflightMessage = "preflight checks passed"
	_ = r.client.Status().Update(r.context, installation)
	return result, nil
}

func (r *ReconcileInstallation) checkNamespaceForProducts(ns corev1.Namespace, installation *v1alpha1.Installation, installationType *Type, configManager *config.Manager) ([]string, error) {
	foundProducts := []string{}
	// new client to avoid caching issues
	serverClient, _ := client.New(r.restConfig, client.Options{})
	for _, stage := range installationType.Stages {
		for _, product := range stage.Products {
			reconciler, err := products.NewReconciler(product.Name, r.restConfig, configManager, installation)
			if err != nil {
				return foundProducts, err
			}
			search := reconciler.GetPreflightObject(ns.Name)
			if search == nil {
				continue
			}
			exists, err := resources.Exists(r.context, serverClient, search)
			if err != nil {
				return foundProducts, err
			} else if exists {
				foundProducts = append(foundProducts, string(product.Name))
			}
		}
	}
	return foundProducts, nil
}
func (r *ReconcileInstallation) processStage(instance *v1alpha1.Installation, stage *v1alpha1.InstallationStageStatus, configManager config.ConfigReadWriter) error {
	incompleteStage := false
	var merr error
	for _, product := range stage.Products {
		reconciler, err := products.NewReconciler(product.Name, r.restConfig, configManager, instance)
		if err != nil {
			stage.Phase = v1alpha1.PhaseFailed
			return pkgerr.Wrapf(err, "failed to build a reconciler for %s", product.Name)
		}
		serverClient, err := client.New(r.restConfig, client.Options{})
		if err != nil {
			stage.Phase = v1alpha1.PhaseFailed
			return pkgerr.Wrap(err, "could not create server client")
		}
		product.Status, err = reconciler.Reconcile(r.context, instance, product, serverClient)
		if err != nil {
			if merr == nil {
				merr = &multiErr{}
			}
			merr.(*multiErr).Add(pkgerr.Wrapf(err, "failed installation of %s", product.Name))
		}
		//found an incomplete product
		if !(product.Status == v1alpha1.PhaseCompleted) {
			incompleteStage = true
		}
	}

	//some products in this stage have not installed successfully yet
	if incompleteStage {
		stage.Phase = v1alpha1.PhaseInProgress
	} else {
		stage.Phase = v1alpha1.PhaseCompleted
	}
	return merr
}

type multiErr struct {
	errors []string
}

func (mer *multiErr) Error() string {
	return "product installation errors : " + strings.Join(mer.errors, ":")
}

func (mer *multiErr) Add(err error) {
	if mer.errors == nil {
		mer.errors = []string{}
	}
	mer.errors = append(mer.errors, err.Error())
}
