package mobilesecurityservice

import (
	"context"

	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/aerogear/mobile-security-service-operator/pkg/utils"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ConfigMap          = "ConfigMap"
	Deployment         = "Deployment"
	ProxyService       = "Proxy Service"
	Route              = "Route"
	ApplicationService = "Application Service"
	ServiceAccount     = "ServiceAccount"
)

var log = logf.Log.WithName("controller_mobilesecurityservice")

// Add creates a new MobileSecurityService Controller and adds it to the Manager.
// The Manager will set fields on the Controller and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// Returns the a new Reconciler for this operator and controller
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileMobileSecurityService{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// Add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("mobilesecurityservice-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource MobileSecurityService
	err = c.Watch(&source.Kind{Type: &mobilesecurityservicev1alpha1.MobileSecurityService{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	/** Watch for changes to secondary resources and create the owner MobileSecurityService **/

	// ConfigMap
	if err := watchConfigMap(c); err != nil {
		return err
	}

	// Deployment
	if err := watchDeployment(c); err != nil {
		return err
	}

	// Service
	if err := watchService(c); err != nil {
		return err
	}

	// Route
	if err := watchRoute(c); err != nil {
		return err
	}

	// ServiceAccount
	if err := watchServiceAccount(c); err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileMobileSecurityService{}

// ReconcileMobileSecurityService reconciles a MobileSecurityService object
type ReconcileMobileSecurityService struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Update the factory object and requeue
func (r *ReconcileMobileSecurityService) update(obj runtime.Object, reqLogger logr.Logger) error {
	err := r.client.Update(context.TODO(), obj)
	if err != nil {
		reqLogger.Error(err, "Failed to update Object", "obj:", obj)
		return err
	}
	reqLogger.Info("Updated successfully", "obj:", obj)
	return nil
}

// Create the factory object and requeue
func (r *ReconcileMobileSecurityService) create(mss *mobilesecurityservicev1alpha1.MobileSecurityService, reqLogger logr.Logger, kind string) error {
	obj := r.buildFactory(reqLogger, mss, kind)
	reqLogger.Info("Creating a new ", "kind", kind, "Namespace", mss.Namespace)
	err := r.client.Create(context.TODO(), obj)
	if err != nil {
		reqLogger.Error(err, "Failed to create new ", "kind", kind, "Namespace", mss.Namespace)
	}
	reqLogger.Info("Created successfully", "kind", kind, "Namespace", mss.Namespace)
	return err
}

// buildFactory will return the resource according to the resource defined
func (r *ReconcileMobileSecurityService) buildFactory(reqLogger logr.Logger, mss *mobilesecurityservicev1alpha1.MobileSecurityService, resource string) runtime.Object {
	reqLogger.Info("Check "+resource, "into the namespace", mss.Namespace)
	switch resource {
	case ConfigMap:
		return r.buildConfigMap(mss)
	case Deployment:
		return r.buildDeployment(mss)
	case ProxyService:
		return r.buildProxyService(mss)
	case ApplicationService:
		return r.buildApplicationService(mss)
	case Route:
		return r.buildRoute(mss)
	case ServiceAccount:
		return r.buildServiceAccount(mss)
	default:
		msg := "Failed to recognize type of object" + resource + " into the Namespace " + mss.Namespace
		panic(msg)
	}
}

// Reconcile reads that state of the cluster for a MobileSecurityService object and makes changes based on the state read
// and what is in the MobileSecurityService.Spec
// Note:
// The Controller will create the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileMobileSecurityService) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Mobile Security Service ...")

	// Fetch the MobileSecurityService
	mss := &mobilesecurityservicev1alpha1.MobileSecurityService{}
	mss, err := r.fetchMssInstance(reqLogger, request)
	if err != nil {
		reqLogger.Error(err, "Failed to get Mobile Security Service ")
		return reconcile.Result{}, err
	}

	// Check if the Service CR was applied in the same namespace of the operator
	if isValidNamespace, err := utils.IsValidOperatorNamespace(mss.Namespace); err != nil || isValidNamespace == false {
		// Get Operator Namespace
		operatorNamespace, _ := k8sutil.GetOperatorNamespace()
		reqLogger.Error(err, "Unable to reconcile Mobile Security Service", "mss.Namespace", mss.Namespace, "isValidNamespace", isValidNamespace, "Operator.Namespace", operatorNamespace)

		// Update CR status with info of Invalid Namespace
		if err := r.updateStatusWithInvalidNamespace(reqLogger, request); err != nil {
			return reconcile.Result{}, err
		}

		// Stop Reconcile
		return reconcile.Result{}, nil
	}
	reqLogger.Info("Valid namespace for Mobile Security Service", "Namespace", request.Namespace)

	// Add const values for mandatory specs
	addMandatorySpecsDefinitions(mss)

	// Check if ConfigMap for the app exist, if not create one.
	if _, err := r.fetchConfigMap(reqLogger, mss); err != nil {
		if err := r.create(mss, reqLogger, ConfigMap); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Check if Deployment for the app exist, if not create one
	deployment, err := r.fetchDeployment(reqLogger, mss)
	if err != nil {
		if err := r.create(mss, reqLogger, Deployment); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	reqLogger.Info("Ensuring the Mobile Security Service deployment size is the same as the spec")
	size := mss.Spec.Size
	if *deployment.Spec.Replicas != size {
		//Set size of the deployment spec
		deployment.Spec.Replicas = &size
		if err := r.update(deployment, reqLogger); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Check if Service for the app exist, if not create one
	if _, err := r.fetchService(reqLogger, mss, utils.ApplicationServiceInstanceName); err != nil {
		if err := r.create(mss, reqLogger, ApplicationService); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Check if Proxy Service for the app exist, if not create one
	if _, err := r.fetchService(reqLogger, mss, utils.ProxyServiceInstanceName); err != nil {
		if err := r.create(mss, reqLogger, ProxyService); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Check if Route for the Service exist, if not create one
	if _, err := r.fetchRoute(reqLogger, mss); err != nil {
		if err := r.create(mss, reqLogger, Route); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Check if ServiceAccount for the app exist, if not create one
	if _, err := r.fetchServiceAccount(reqLogger, mss); err != nil {
		if err := r.create(mss, reqLogger, ServiceAccount); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Update status for ConfigMap
	configMapStatus, err := r.updateConfigMapStatus(reqLogger, request)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Update status for deployment
	deploymentStatus, err := r.updateDeploymentStatus(reqLogger, request)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Update status for Proxy Service
	proxyServiceStatus, err := r.updateProxyServiceStatus(reqLogger, request)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Update status for Application Service
	applicationServiceStatus, err := r.updateAppServiceStatus(reqLogger, request)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Update status for Route
	routeStatus, err := r.updateRouteStatus(reqLogger, request)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Update status for App
	if err := r.updateStatus(reqLogger, configMapStatus, deploymentStatus, proxyServiceStatus, applicationServiceStatus, routeStatus, request); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
