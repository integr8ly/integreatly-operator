package installation

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	pkgerr "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

const (
	defaultInstallationConfigMapName = "integreatly-installation-config"
)

var log = logf.Log.WithName("Installation Controller")

// Add creates a new Installation Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	restConfig := controllerruntime.GetConfigOrDie()
	coreClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		logrus.Infof("error creating core client: %v", err)
		return &ReconcileInstallation{client: mgr.GetClient(), scheme: mgr.GetScheme()}
	}
	return &ReconcileInstallation{client: mgr.GetClient(), scheme: mgr.GetScheme(), coreClient: coreClient, restConfig: restConfig}
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
	client     client.Client
	scheme     *runtime.Scheme
	coreClient *kubernetes.Clientset
	restConfig *rest.Config
}

// Reconcile reads that state of the cluster for a Installation object and makes changes based on the state read
// and what is in the Installation.Spec
func (r *ReconcileInstallation) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	instance := &v1alpha1.Installation{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if instance.Status.Stages == nil {
		instance.Status.Stages = map[int]string{}
	}

	err, installType := InstallationTypeFactory(instance.Spec.Type)
	if err != nil {
		return reconcile.Result{}, err
	}
	installationCfgMap := os.Getenv("INSTALLATION_CONFIG_MAP")
	if installationCfgMap == "" {
		installationCfgMap = instance.Spec.NamespacePrefix + defaultInstallationConfigMapName
	}

	configManager, err := config.NewManager(r.client, request.NamespacedName.Namespace, installationCfgMap)
	if err != nil {
		return reconcile.Result{}, err
	}
	for stage, installProducts := range installType.GetProductOrder() {
		// if the stage has a status phase already, check it's value
		val, ok := instance.Status.Stages[stage]
		if ok {
			//if this stage failed we need to abort the install, so return an error
			if val == string(v1alpha1.PhaseFailed) {
				return reconcile.Result{}, pkgerr.New(fmt.Sprintf("installation failed on stage %d", stage))
			}
		}
		logrus.Infof("processing stage: %s", val)
		//found an incomplete stage, so process it and log it's status
		phase, err := r.processStage(instance, installProducts, configManager)
		instance.Status.Stages[stage] = string(phase)
		if err != nil {
			return reconcile.Result{}, err
		}
		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			// The 'Update' function can error if the resource has been updated by another process and the versions are not correct.
			if k8serr.IsConflict(err) {
				// If there is a conflict, requeue the resource and retry Update
				log.Info("Error updating Installation resource. Requeue and retry.")
				return reconcile.Result{
					Requeue: true,
				}, nil
			}

			log.Error(err, "Error reconciling installation instance status")
			return reconcile.Result{}, err
		}
		err = r.client.Update(context.TODO(), instance)
		if err != nil {
			// The 'Update' function can error if the resource has been updated by another process and the versions are not correct.
			if k8serr.IsConflict(err) {
				// If there is a conflict, requeue the resource and retry Update
				log.Info("Error updating Installation resource. Requeue and retry.")
				return reconcile.Result{
					Requeue: true,
				}, nil
			}

			log.Error(err, "error reconciling installation instance")
			return reconcile.Result{}, err
		}

		// if phase is still in progress, ensure it's requeued until its completed.
		if string(phase) == string(v1alpha1.PhaseInProgress) {
			return reconcile.Result{
				Requeue:      true,
				RequeueAfter: time.Second * 10,
			}, nil
		}
		//don't move to next stage until current stage is finished
		if !(val == string(v1alpha1.PhaseCompleted)) {
			break
		}
	}
	return reconcile.Result{
		Requeue:      true,
		RequeueAfter: time.Second * 10,
	}, nil
}

func (r *ReconcileInstallation) processStage(instance *v1alpha1.Installation, prods []v1alpha1.ProductName, configManager config.ConfigReadWriter) (v1alpha1.StatusPhase, error) {
	incompleteStage := false
	if instance.Status.ProductStatus == nil {
		instance.Status.ProductStatus = map[v1alpha1.ProductName]string{}
	}
	for _, product := range prods {
		logrus.Infof("checking product: %s", product)
		phase := ""
		//check current phase of this product installation
		if val, ok := instance.Status.ProductStatus[product]; ok {
			phase = val
			//product failed to install, return error and failed phase for stage
			if phase == string(v1alpha1.PhaseFailed) {
				//found a failed product
				incompleteStage = true
			}
		}
		//found an incomplete product
		if !(phase == string(v1alpha1.PhaseCompleted)) {
			incompleteStage = true
		}
		reconciler, err := products.NewReconciler(v1alpha1.ProductName(product), r.client, r.restConfig, r.coreClient, configManager, instance)
		if err != nil {
			return v1alpha1.PhaseFailed, pkgerr.Wrapf(err, "failed installation of %s", product)
		}
		logrus.Infof("reconciling product: %s", product)
		newPhase, err := reconciler.Reconcile(instance)
		instance.Status.ProductStatus[product] = string(newPhase)
		if err != nil {
			return v1alpha1.PhaseFailed, pkgerr.Wrapf(err, "failed installation of %s", product)
		}
	}

	//some products in this stage have not installed successfully yet
	if incompleteStage {
		return v1alpha1.PhaseInProgress, nil
	}
	return v1alpha1.PhaseCompleted, nil
}
