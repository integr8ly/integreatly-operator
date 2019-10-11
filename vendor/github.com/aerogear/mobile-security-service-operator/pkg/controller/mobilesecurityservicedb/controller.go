package mobilesecurityservicedb

import (
	"context"
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

var log = logf.Log.WithName("controller_mobilesecurityservicedb")

const (
	DEEPLOYMENT = "Deployment"
	PVC         = "PersistentVolumeClaim"
	SERVICE     = "Service"
)

// Add creates a new MobileSecurityServiceDB Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileMobileSecurityServiceDB{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("mobilesecurityservicedb-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
	// Watch for changes to primary resource MobileSecurityServiceDB
	err = c.Watch(&source.Kind{Type: &mobilesecurityservicev1alpha1.MobileSecurityServiceDB{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	/** Watch for changes to secondary resources and create the owner MobileSecurityService **/

	//Deployment
	if err := watchDeployment(c); err != nil {
		return err
	}

	//Service
	if err := watchService(c); err != nil {
		return err
	}

	//PersistenceVolume
	if err := watchPersistenceVolumeClaim(c); err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileMobileSecurityServiceDB{}

// ReconcileMobileSecurityServiceDB reconciles a MobileSecurityServiceDB object
type ReconcileMobileSecurityServiceDB struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

//Update the object and reconcile it
func (r *ReconcileMobileSecurityServiceDB) update( obj runtime.Object, reqLogger logr.Logger) (reconcile.Result, error) {
	err := r.client.Update(context.TODO(), obj)
	if err != nil {
		reqLogger.Error(err, "Failed to update Spec")
		return reconcile.Result{}, err
	}
	reqLogger.Info("Spec updated - return and create")
	return reconcile.Result{Requeue: true}, nil
}

//Create the object and reconcile it
func (r *ReconcileMobileSecurityServiceDB) create(instance *mobilesecurityservicev1alpha1.MobileSecurityServiceDB, reqLogger logr.Logger, kind string, err error) (reconcile.Result, error) {
	obj, errBuildObject := r.buildFactory(reqLogger, instance, kind)
	if errBuildObject != nil {
		return reconcile.Result{}, errBuildObject
	}
	if errors.IsNotFound(err) {
		reqLogger.Info("Creating a new ", "kind", kind, "Namespace", instance.Namespace)
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			reqLogger.Error(err, "Failed to create new ", "kind", kind, "Namespace", instance.Namespace)
			return reconcile.Result{}, err
		}
		reqLogger.Info("Created successfully - return and create", "kind", kind, "Namespace", instance.Namespace)
		return reconcile.Result{Requeue: true}, nil
	}
	reqLogger.Error(err, "Failed to get", "kind", kind, "Namespace", instance.Namespace)
	return reconcile.Result{}, err
}

//buildFactory will return the resource according to the kind defined
func (r *ReconcileMobileSecurityServiceDB) buildFactory(reqLogger logr.Logger, instance *mobilesecurityservicev1alpha1.MobileSecurityServiceDB, kind string) (runtime.Object, error) {
	reqLogger.Info("Check "+kind, "into the namespace", instance.Namespace)
	switch kind {
	case PVC:
		return r.buildPVCForDB(instance), nil
	case DEEPLOYMENT:
		return r.buildDBDeployment(instance), nil
	case SERVICE:
		return r.buildDBService(instance), nil
	default:
		msg := "Failed to recognize type of object" + kind + " into the Namespace " + instance.Namespace
		panic(msg)
	}
}



// Reconcile reads that state of the cluster for a MobileSecurityServiceDB object and makes changes based on the state read
// and what is in the MobileSecurityServiceDB.Spec
// Note:
// The Controller will create the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileMobileSecurityServiceDB) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Mobile Security Service Database")

	instance := &mobilesecurityservicev1alpha1.MobileSecurityServiceDB{}

	//Fetch the MobileSecurityService instance
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		return fetch(r, reqLogger, err)
	}

	//Check if Deployment for the app exist, if not create one
	deployment, err := r.fetchDBDeployment(reqLogger, instance)
	if err != nil {
		// To give time for the mobile security service CRD controller create the configMap which will be used for both.
		// If the configMap be not found it will created with the default values specified in its CR for the env variables
		time.Sleep(10 * time.Second)
		return r.create(instance, reqLogger, DEEPLOYMENT, err)
	}

	//Ensure the deployment size is the same as the spec
	reqLogger.Info("Ensuring the Mobile Security Service Database deployment size is the same as the spec")
	size := instance.Spec.Size
	if *deployment.Spec.Replicas != size {
		deployment.Spec.Replicas = &size
		return r.update(deployment, reqLogger)
	}

	//Check if Service for the app exist, if not create one
	if _, err := r.fetchDBService(reqLogger, instance); err != nil {
		return r.create(instance, reqLogger, SERVICE, err)
	}

	//Check if PersistentVolumeClaim for the app exist, if not create one
	if _, err := r.fetchDBPersistentVolumeClaim(reqLogger, instance); err != nil {
		return r.create(instance, reqLogger, PVC, err)
	}

	//Update status for deployment
	deploymentStatus, err := r.updateDeploymentStatus(reqLogger,instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	//Update status for Service
	serviceStatus, err := r.updateServiceStatus(reqLogger, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	//Update status for PVC
	pvcStatus, err := r.updatePvcStatus(reqLogger, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	//Update status for DB
	if err:= r.updateDBStatus(reqLogger, deploymentStatus, serviceStatus, pvcStatus, instance); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
