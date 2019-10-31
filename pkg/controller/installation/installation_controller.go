package installation

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
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
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	defaultInstallationConfigMapName = "integreatly-installation-config"
)

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
		instance.Status.Stages = map[v1alpha1.StageName]*v1alpha1.InstallationStageStatus{}
	}

	// If the CR is being deleted, cancel the current context
	// and attempt to clean up the products with finalizers
	if instance.DeletionTimestamp != nil {
		// Cancel this context to kill all ongoing requests to the API
		// and use a new context to handle deletion logic
		r.cancel()

		// Clean up the products which have finalizers associated to them
		merr := &multiErr{}
		for _, productFinalizer := range instance.Finalizers {
			if !strings.Contains(productFinalizer, "integreatly") {
				continue
			}
			productName := strings.Split(productFinalizer, ".")[1]
			product := instance.GetProductStatusObject(v1alpha1.ProductName(productName))
			reconciler, err := products.NewReconciler(product.Name, r.restConfig, configManager, instance)
			if err != nil {
				merr.Add(pkgerr.Wrapf(err, "Failed to build reconciler for product %s", product.Name))
			}
			serverClient, err := client.New(r.restConfig, client.Options{})
			if err != nil {
				merr.Add(pkgerr.Wrapf(err, "Failed to create server client for %s", product.Name))
			}
			_, err = reconciler.Reconcile(context.TODO(), instance, product, serverClient)
			if err != nil {
				merr.Add(pkgerr.Wrapf(err, "Failed to reconcile product %s", product.Name))
			}
		}

		if len(merr.errors) == 0 {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, merr
	}

	for _, stage := range installType.GetStages() {
		var err error
		var stagePhase v1alpha1.StatusPhase
		if stage.Name == v1alpha1.BootstrapStage {
			stagePhase, err = r.bootstrapStage(instance, configManager)
		} else {
			stagePhase, err = r.processStage(instance, &stage, configManager)
		}

		if instance.Status.Stages == nil {
			instance.Status.Stages = make(map[v1alpha1.StageName]*v1alpha1.InstallationStageStatus)
		}
		instance.Status.Stages[stage.Name] = &v1alpha1.InstallationStageStatus{
			Name:     stage.Name,
			Phase:    stagePhase,
			Products: stage.Products,
		}
		if err != nil {
			return reconcile.Result{}, err
		}
		//don't move to next stage until current stage is complete
		if stagePhase != v1alpha1.PhaseCompleted {
			break
		}
	}

	// UPDATE STATUS
	err = r.client.Status().Update(r.context, instance)
	if err != nil {
		// The 'Update' function can error if the resource has been updated by another process and the versions are not correct.
		if k8serr.IsConflict(err) {
			// If there is a conflict, requeue the resource and retry Update
			logrus.Info("Error updating Installation resource status. Requeue and retry.", err)
			return reconcile.Result{
				Requeue:      true,
				RequeueAfter: time.Second * 10,
			}, nil
		}

		logrus.Error(err, "error reconciling installation instance")
		return reconcile.Result{}, err
	}

	// UPDATE OBJECT
	err = r.client.Update(r.context, instance)
	if err != nil {
		// The 'Update' function can error if the resource has been updated by another process and the versions are not correct.
		if k8serr.IsConflict(err) {
			// If there is a conflict, requeue the resource and retry Update
			logrus.Info("Error updating Installation resource. Requeue and retry.", err)
			return reconcile.Result{
				Requeue:      true,
				RequeueAfter: time.Second * 10,
			}, nil
		}

		logrus.Error(err, "error reconciling installation instance")
		return reconcile.Result{}, err
	}

	return reconcile.Result{
		Requeue:      true,
		RequeueAfter: time.Second * 10,
	}, nil
}

func (r *ReconcileInstallation) preflightChecks(installation *v1alpha1.Installation, installationType *Type, configManager *config.Manager) (reconcile.Result, error) {
	logrus.Info("Running preflight checks..")

	result := reconcile.Result{
		Requeue:      true,
		RequeueAfter: time.Second * 10,
	}
	requiredSecrets := []string{"s3-credentials", "s3-bucket", "github-oauth-secret"}
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
			preflightMessage := fmt.Sprintf("Could not find %s secret in integreatly-operator namespace: %s", secret.Name, installation.Namespace)
			installation.Status.PreflightStatus = v1alpha1.PreflightFail
			installation.Status.PreflightMessage = preflightMessage
			logrus.Info(preflightMessage)
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

func (r *ReconcileInstallation) bootstrapStage(instance *v1alpha1.Installation, configManager config.ConfigReadWriter) (v1alpha1.StatusPhase, error) {
	mpm := marketplace.NewManager()

	reconciler, err := NewBootstrapReconciler(configManager, instance, mpm)
	if err != nil {
		return v1alpha1.PhaseFailed, pkgerr.Wrapf(err, "failed to build a reconciler for Bootstrap")
	}
	serverClient, err := client.New(r.restConfig, client.Options{})
	if err != nil {
		return v1alpha1.PhaseFailed, pkgerr.Wrap(err, "could not create server client")
	}
	phase, err := reconciler.Reconcile(r.context, instance, serverClient)
	if err != nil || phase == v1alpha1.PhaseFailed {
		return v1alpha1.PhaseFailed, pkgerr.Wrap(err, "Bootstrap stage reconcile failed")
	}

	return phase, nil
}

func (r *ReconcileInstallation) processStage(instance *v1alpha1.Installation, stage *Stage, configManager config.ConfigReadWriter) (v1alpha1.StatusPhase, error) {
	incompleteStage := false
	var mErr error
	for _, product := range stage.Products {
		reconciler, err := products.NewReconciler(product.Name, r.restConfig, configManager, instance)
		if err != nil {
			return v1alpha1.PhaseFailed, pkgerr.Wrapf(err, "failed to build a reconciler for %s", product.Name)
		}
		serverClient, err := client.New(r.restConfig, client.Options{})
		if err != nil {
			return v1alpha1.PhaseFailed, pkgerr.Wrap(err, "could not create server client")
		}
		product.Status, err = reconciler.Reconcile(r.context, instance, product, serverClient)
		if err != nil {
			if mErr == nil {
				mErr = &multiErr{}
			}
			mErr.(*multiErr).Add(pkgerr.Wrapf(err, "failed installation of %s", product.Name))
		}
		//found an incomplete product
		if !(product.Status == v1alpha1.PhaseCompleted) {
			incompleteStage = true
		}
	}

	//some products in this stage have not installed successfully yet
	if incompleteStage {
		return v1alpha1.PhaseInProgress, mErr
	}
	return v1alpha1.PhaseCompleted, mErr
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
