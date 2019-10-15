package mobilesecurityservicedb

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

var log = logf.Log.WithName("controller_mobilesecurityservicedb")

const (
	Deployment = "Deployment"
	PVC        = "PersistentVolumeClaim"
	Service    = "Service"
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

	// Deployment
	if err := watchDeployment(c); err != nil {
		return err
	}

	// Service
	if err := watchService(c); err != nil {
		return err
	}

	// PersistenceVolume
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

// Update the object and reconcile it
func (r *ReconcileMobileSecurityServiceDB) update(obj runtime.Object, reqLogger logr.Logger) error {
	err := r.client.Update(context.TODO(), obj)
	if err != nil {
		reqLogger.Error(err, "Failed to update Object", "obj:", obj)
		return err
	}
	reqLogger.Info("Object updated", "obj:", obj)
	return nil
}

// Create the object and reconcile it
func (r *ReconcileMobileSecurityServiceDB) create(db *mobilesecurityservicev1alpha1.MobileSecurityServiceDB, kind string, reqLogger logr.Logger) error {
	obj := r.buildFactory(db, kind, reqLogger)
	reqLogger.Info("Creating a new ", "kind", kind, "Namespace", db.Namespace)
	err := r.client.Create(context.TODO(), obj)
	if err != nil {
		reqLogger.Error(err, "Failed to create new ", "kind", kind, "Namespace", db.Namespace)
		return err
	}
	reqLogger.Info("Created successfully", "kind", kind, "Namespace", db.Namespace)
	return nil
}

// buildFactory will return the resource according to the kind defined
func (r *ReconcileMobileSecurityServiceDB) buildFactory(db *mobilesecurityservicev1alpha1.MobileSecurityServiceDB, kind string, reqLogger logr.Logger) runtime.Object {
	reqLogger.Info("Check "+kind, "into the namespace", db.Namespace)
	switch kind {
	case PVC:
		return r.buildPVCForDB(db)
	case Deployment:
		// If we decide to export the DB to be reused it for other operators (E.g create an operator for the PostgreSQL) then it should not be made here
		// The Service should find the DB deployment and update its ENV VARs instead of we looking for it here for we are able to extract it and use it with any service
		serviceConfigMapName := r.getMssConfigMapName(db)
		return r.buildDBDeployment(db, serviceConfigMapName)
	case Service:
		return r.buildDBService(db)
	default:
		msg := "Failed to recognize type of object" + kind + " into the Namespace " + db.Namespace
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

	// Fetch the MobileSecurityService DB
	db := &mobilesecurityservicev1alpha1.MobileSecurityServiceDB{}
	db, err := r.fetchDBInstance(reqLogger, request)
	if err != nil {
		reqLogger.Error(err, "Failed to get Mobile Security Service DB")
		return reconcile.Result{}, err
	}

	// Check if the DB CR was applied in the same namespace of the operator
	if isValidNamespace, err := utils.IsValidOperatorNamespace(db.Namespace); err != nil || isValidNamespace == false {
		operatorNamespace, _ := k8sutil.GetOperatorNamespace()
		reqLogger.Error(err, "Unable to reconcile Mobile Security Service Database", "db.Namespace", db.Namespace, "isValidNamespace", isValidNamespace, "Operator.Namespace", operatorNamespace)

		//Update status with Invalid Namespace
		if err := r.updateStatusWithInvalidNamespace(reqLogger, request); err != nil {
			return reconcile.Result{}, err
		}

		// Stop reconcile
		return reconcile.Result{}, nil
	}
	reqLogger.Info("Valid namespace for Mobile Security Service DB", "Namespace", request.Namespace)

	// Add const values for mandatory specs
	addMandatorySpecsDefinitions(db)

	// Check if Deployment for the app exist, if not create one
	deployment, err := r.fetchDBDeployment(reqLogger, db)
	if err != nil {
		// Create the deployment
		if err := r.create(db, Deployment, reqLogger); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	// Check if Service for the app exist, if not create one
	if _, err := r.fetchDBService(reqLogger, db); err != nil {
		if err := r.create(db, Service, reqLogger); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Check if PersistentVolumeClaim for the app exist, if not create one
	if _, err := r.fetchDBPersistentVolumeClaim(reqLogger, db); err != nil {
		if err := r.create(db, PVC, reqLogger); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Ensure the deployment size is the same as the spec
	reqLogger.Info("Ensuring the Mobile Security Service Database deployment size is the same as the spec")
	size := db.Spec.Size
	if *deployment.Spec.Replicas != size {
		// Set the number of Replicas spec in the CR
		deployment.Spec.Replicas = &size
		// Update
		if err := r.update(deployment, reqLogger); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Update status for deployment
	deploymentStatus, err := r.updateDeploymentStatus(reqLogger, request)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Update status for Service
	serviceStatus, err := r.updateServiceStatus(reqLogger, request)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Update status for PVC
	pvcStatus, err := r.updatePvcStatus(reqLogger, request)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Update status for DB
	if err := r.updateDBStatus(reqLogger, deploymentStatus, serviceStatus, pvcStatus, request); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
