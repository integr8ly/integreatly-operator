package subscription

import (
	"context"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// IntegreatlyPackage - package name is used for Subsription name
	IntegreatlyPackage = "integreatly"
)

// Add creates a new Subscription Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, _ []string) error {
	return add(mgr, newReconciler(mgr))
}

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	operatorNs, _ := k8sutil.GetOperatorNamespace()
	return &ReconcileSubscription{client: mgr.GetClient(), scheme: mgr.GetScheme(), operatorNamespace: operatorNs}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("subscription-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &v1alpha1.Subscription{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	return nil
}

// blank assignment to verify that ReconcileSubscription implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileSubscription{}

type ReconcileSubscription struct {
	client            k8sclient.Client
	scheme            *runtime.Scheme
	operatorNamespace string
}

// Reconcile will ensure that that Subscription object(s) have Manual approval for the upgrades
// In a namespaced installation of integreatly operator it will only reconcile Subscription of the integreatly operator itself
func (r *ReconcileSubscription) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// skip any Subscriptions that are not integreatly operator
	if request.Namespace != r.operatorNamespace ||
		(request.Name != IntegreatlyPackage && request.Name != "addon-rhmi") {
		return reconcile.Result{}, nil
	}

	instance := &v1alpha1.Subscription{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request. Return and don't requeue
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if instance.Spec.InstallPlanApproval != v1alpha1.ApprovalManual {
		instance.Spec.InstallPlanApproval = v1alpha1.ApprovalManual
		err = r.client.Update(context.TODO(), instance)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}
