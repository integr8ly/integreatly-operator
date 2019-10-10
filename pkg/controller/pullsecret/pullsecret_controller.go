package pullsecret

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	// controllerruntime "sigs.k8s.io/controller-runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/integr8ly/integreatly-operator/pkg/resources"
)

var log = logf.Log.WithName("controller_pullsecret")

const (
	WebAppLabel = "webapp"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new PullSecret Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, products []string) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {

	// new client to avoid caching issues
	restConfig := controllerruntime.GetConfigOrDie()
	newClient, _ := client.New(restConfig, client.Options{})
	return &ReconcilePullSecret{client: newClient, scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("pullsecret-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource
	err = c.Watch(&source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcilePullSecret implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcilePullSecret{}

// ReconcilePullSecret reconciles a PullSecret object
type ReconcilePullSecret struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile will ensure namespaces with the WebApp label has a default pull secret copied and assigned to the namespace
func (r *ReconcilePullSecret) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Checking namespace for webapp label for pull secret reconciling")

	// Fetch the Namespace instance
	instance := &corev1.Namespace{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Only for namespaces with the webapp label, copy the default pull secret and set as the default pull secret for namespace
	if _, ok := instance.ObjectMeta.Labels[WebAppLabel]; ok {
		reqLogger.Info("Found namespace with webapp label")

		err = resources.CopyDefaultPullSecretToNameSpace(request.Name, resources.DefaultOriginPullSecretName, r.client, context.TODO())

		if err != nil {
			return reconcile.Result{}, err
		}

		reqLogger.Info("Successfully updated namespace with default pull secret")
	}

	return reconcile.Result{}, nil
}
