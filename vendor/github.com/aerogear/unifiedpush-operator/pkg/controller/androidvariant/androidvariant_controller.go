package androidvariant

import (
	"context"
	"time"

	pushv1alpha1 "github.com/aerogear/unifiedpush-operator/pkg/apis/push/v1alpha1"
	"github.com/aerogear/unifiedpush-operator/pkg/controller/util"
	"github.com/aerogear/unifiedpush-operator/pkg/nspredicate"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_androidvariant")

// Add creates a new AndroidVariant Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileAndroidVariant{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("androidvariant-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource AndroidVariant
	onlyEnqueueForValidNamespaces, err := nspredicate.NewFromEnvVar("APP_NAMESPACES")
	if err != nil {
		return err
	}
	err = c.Watch(
		&source.Kind{Type: &pushv1alpha1.AndroidVariant{}},
		&handler.EnqueueRequestForObject{},
		onlyEnqueueForValidNamespaces,
	)
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileAndroidVariant{}

// ReconcileAndroidVariant reconciles a AndroidVariant object
type ReconcileAndroidVariant struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a AndroidVariant
// object and makes changes based on the state read and what is in the
// AndroidVariant.Spec
// Note:
// The Controller will requeue the Request to be processed again if
// the returned error is non-nil or Result.Requeue is true, otherwise
// upon completion it will remove the work from the queue.
func (r *ReconcileAndroidVariant) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling AndroidVariant")

	// Fetch the AndroidVariant instance
	instance := &pushv1alpha1.AndroidVariant{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Get a UPS Client for interactions with the UPS service
	unifiedpushClient, err := util.UnifiedpushClient(r.client, reqLogger)
	if err != nil {
		reqLogger.Error(err, "Error getting a UPS Client.", "AndroidVariant.Name", instance.Name)
		return reconcile.Result{RequeueAfter: time.Second * 5}, nil
	}

	// Check if the CR was marked to be deleted
	if instance.GetDeletionTimestamp() != nil {
		// First delete from UPS
		err := unifiedpushClient.DeleteAndroidVariant(instance)
		if err != nil {
			reqLogger.Error(err, "Failed to delete AndroidVariant from UPS", "AndroidVaraint.Name", instance.Name)
			return reconcile.Result{}, err
		}

		// Then unset finalizers
		instance.SetFinalizers(nil)
		err = r.client.Update(context.TODO(), instance)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	foundVariant, err := unifiedpushClient.GetAndroidVariant(instance)
	if err != nil {
		// this doesn't denote a 404. it is a 500
		reqLogger.Error(err, "Error getting the existing Android variant.", "AndroidVariant.Name", instance.Name)
		return reconcile.Result{}, err
	}

	if foundVariant.VariantId == "" {
		createdVariant, err := unifiedpushClient.CreateAndroidVariant(instance)
		if err != nil {
			reqLogger.Error(err, "Error creating Android variant in UPS.", "AndroidVariant.Name", instance.Name)
			return reconcile.Result{RequeueAfter: time.Second * 5}, nil
		}

		if instance.ObjectMeta.Annotations == nil {
			instance.ObjectMeta.Annotations = make(map[string]string)
		}
		instance.ObjectMeta.Annotations["variantId"] = createdVariant.VariantId
		err = r.client.Update(context.TODO(), instance)
		if err != nil {
			return reconcile.Result{}, err
		}

		reqLogger.Info("Android Variant created", "Name", instance.Name, "VariantId", createdVariant.VariantId)
		return reconcile.Result{Requeue: true}, nil

	}

	if instance.ObjectMeta.Annotations["variantId"] != foundVariant.VariantId {
		instance.ObjectMeta.Annotations["variantId"] = foundVariant.VariantId
		err = r.client.Update(context.TODO(), instance)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// If either of these have diverged, fix both, so there's just a single status update
	if instance.Status.VariantId != foundVariant.VariantId || instance.Status.Secret != foundVariant.Secret || instance.Status.Ready != true {
		instance.Status.VariantId = foundVariant.VariantId
		instance.Status.Secret = foundVariant.Secret
		instance.Status.Ready = true
		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Error updating AndroidVariant status", "Name", instance.Name)
			return reconcile.Result{}, err
		}
	}

	if err := util.AddFinalizer(r.client, reqLogger, instance); err != nil {
		return reconcile.Result{}, err
	}

	reqLogger.Info("Android Variant reconciled", "Name", instance.Name, "VariantId", foundVariant.VariantId)
	return reconcile.Result{}, nil
}
