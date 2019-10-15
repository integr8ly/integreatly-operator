package mobilesecurityserviceapp

import (
	"context"

	"github.com/aerogear/mobile-security-service/pkg/models"

	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/aerogear/mobile-security-service-operator/pkg/service"
	"github.com/aerogear/mobile-security-service-operator/pkg/utils"
	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_mobilesecurityserviceapp")

// Add creates a new MobileSecurityServiceApp Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileMobileSecurityServiceApp{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("mobilesecurityserviceapp-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource MobileSecurityServiceApp
	err = c.Watch(&source.Kind{Type: &mobilesecurityservicev1alpha1.MobileSecurityServiceApp{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileMobileSecurityServiceApp{}

// Update the factory object and requeue
func (r *ReconcileMobileSecurityServiceApp) delete(obj runtime.Object, reqLogger logr.Logger) error {
	err := r.client.Delete(context.TODO(), obj)
	if err != nil {
		reqLogger.Error(err, "Failed to delete obj", "obj:", obj)
		return err
	}
	reqLogger.Info("Deleted successfully", "obj:", obj)
	return nil
}

// ReconcileMobileSecurityServiceApp reconciles a MobileSecurityServiceApp object
type ReconcileMobileSecurityServiceApp struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a MobileSecurityServiceApp object and makes changes based on the state read
// and what is in the MobileSecurityServiceApp.Spec
// Note:
// The Controller will create the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileMobileSecurityServiceApp) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling MobileSecurityServiceApp")

	//Fetch the MobileSecurityService App mssApp
	mssApp := &mobilesecurityservicev1alpha1.MobileSecurityServiceApp{}
	mssApp, err := r.fetchMssAppInstance(reqLogger, request)
	if err != nil {
		reqLogger.Error(err, "Failed to get Mobile Security Service App")
		return reconcile.Result{}, err
	}

	// Ensure that the APP CR will be installed and performed just in the namespaces setup in the ENV VAR APP NAMESPACES
	if isValidNamespace, err := utils.IsValidAppNamespace(mssApp.Namespace); err != nil || isValidNamespace == false {
		// Stop reconcile
		envVar, _ := utils.GetAppNamespaces()
		reqLogger.Error(err, "Unable to reconcile Mobile Security Service App", "mssApp.Namespace", mssApp.Namespace, "isValidNamespace", isValidNamespace, "EnvVar.APP_NAMESPACES", envVar)

		//Update status with Invalid Namespace
		if err := r.updateBindStatusWithInvalidNamespace(reqLogger, request); err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	reqLogger.Info("Valid namespace for MobileSecurityServiceApp", "Namespace", request.Namespace)
	reqLogger.Info("Checking for MobileSecurityService instance ...")

	operatorNamespace, err := k8sutil.GetOperatorNamespace()

	// Check if it is a local env or an unit test
	if err == k8sutil.ErrNoNamespace {
		operatorNamespace = utils.OperatorNamespaceForLocalEnv
	}

	// Fetch MSS Instance
	mss := r.fetchMssInstance(reqLogger, operatorNamespace, request)

	// Check if has Conditionals to be deleted and perform the actions required to allow it.
	if hasConditionsToBeDeleted(mssApp, mss) {
		// Try to fetch MSS and check if the Service was deleted and/or marked to be deleted
		if r.isMobileSecurityServiceDeleted(operatorNamespace, mss) {
			reqLogger.Info("Mobile Security Service mssApp resource not found. Mobile Security Service Application is required to create the application")

			if err := r.removeFinalizerFromCR(mssApp); err != nil {
				reqLogger.Error(err, "Failed to update MobileSecurityService App CR with finalizer")
				return reconcile.Result{}, err
			}

			//Stop the reconcile
			return reconcile.Result{}, nil
		}

		// Get the REST Service Endpoint
		serviceAPI := utils.GetServiceAPIURL(mss)

		// If the CR was marked to be deleted before it finalizes the app need to be deleted from the Service Side
		// Do request to get the app.ID to delete app
		app, err := fetchBindAppRestServiceByAppID(serviceAPI, mssApp, reqLogger)
		if err != nil {
			return reconcile.Result{}, err
		}

		// If the request works with success and the app was found then
		// Do request to delete it from the service
		if app.ID != "" {
			if err := service.DeleteAppFromServiceByRestAPI(serviceAPI, app.ID, reqLogger); err != nil {
				reqLogger.Error(err, "Unable to delete app from Service", "App.ID", app.ID)
				return reconcile.Result{}, err
			}
			reqLogger.Info("Successfully delete app ...")
		}

		// Check if the finalizer criteria is met and remove finalizer from the CR
		if err := r.handleFinalizer(serviceAPI, reqLogger, request); err != nil {
			return reconcile.Result{}, err
		}

		//Stop the reconcile
		return reconcile.Result{}, nil
	}

	if !hasMandatorySpecs(mssApp, reqLogger) {
		//Stop reconcile since it has not the mandatory specs
		return reconcile.Result{}, nil
	}

	// Add finalizer for this CR
	if err := r.addFinalizer(reqLogger, mssApp, request); err != nil {
		return reconcile.Result{}, err
	}

	// Get the route in order to obtain the public Service URL API
	reqLogger.Info("Checking if the route already exists ...")
	route := &routev1.Route{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: mss.Spec.RouteName, Namespace: operatorNamespace}, route); err != nil {
		return reconcile.Result{}, err
	}

	// Get the REST Service Endpoint
	serviceAPI := utils.GetServiceAPIURL(mss)

	// Fetch app
	app, err := fetchBindAppRestServiceByAppID(serviceAPI, mssApp, reqLogger)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Bind App in the Service by the REST API
	// NOTE: If the app was soft deleted before it will make the required job as well
	if app.ID == "" {
		newApp := models.NewAppByNameAndAppID(mssApp.Spec.AppName, mssApp.Spec.AppId)
		if err := service.CreateAppByRestAPI(serviceAPI, newApp, reqLogger); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Update the app name if it was changed.
	if app.AppName != mssApp.Spec.AppName {

		// Re-fetch the app to get the app.ID since now it was created in the Service
		if app.ID == "" {
			app, err = fetchBindAppRestServiceByAppID(serviceAPI, mssApp, reqLogger)
			if err != nil {
				return reconcile.Result{}, err
			}
		}

		// Update the name by the REST API when exists the app
		app.AppName = mssApp.Spec.AppName

		//Check if App was update with success
		if err := service.UpdateAppNameByRestAPI(serviceAPI, app, reqLogger); err != nil {
			return reconcile.Result{}, err
		}
	}

	//Update status for BindStatus
	if err := r.updateBindStatus(serviceAPI, reqLogger, mssApp, request); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
