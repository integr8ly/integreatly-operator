package subscription

import (
	"context"
	"testing"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := v1alpha1.SchemeBuilder.AddToScheme(scheme)
	return scheme, err
}

func TestSubscriptionReconciler(t *testing.T) {
	scenarios := []struct {
		Name            string
		Request         reconcile.Request
		APISubscription *v1alpha1.Subscription
		Verify          func(client client.Client, res reconcile.Result, err error, t *testing.T)
	}{
		{
			Name: "subscription controller changes automatic to manual",
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "test-namespace",
					Name:      "test-subscription",
				},
			},
			APISubscription: &v1alpha1.Subscription{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "test-namespace",
					Name:      "test-subscription",
				},
				Spec: &v1alpha1.SubscriptionSpec{
					InstallPlanApproval: v1alpha1.ApprovalAutomatic,
				},
			},
			Verify: func(c client.Client, res reconcile.Result, err error, t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
				sub := &v1alpha1.Subscription{}
				c.Get(context.TODO(), client.ObjectKey{Name: "test-subscription", Namespace: "test-namespace"}, sub)
				if sub.Spec.InstallPlanApproval != v1alpha1.ApprovalManual {
					t.Fatalf("expected Manual but got %s", sub.Spec.InstallPlanApproval)
				}
			},
		},
		{
			Name: "subscription controller handles when subscription is missing",
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "test-namespace",
					Name:      "test-subscription",
				},
			},
			APISubscription: &v1alpha1.Subscription{},
			Verify: func(c client.Client, res reconcile.Result, err error, t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			},
		},
	}

	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("failed to build scheme: %s", err.Error())
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			APIObject := scenario.APISubscription
			client := fakeclient.NewFakeClientWithScheme(scheme, APIObject)
			reconciler := ReconcileSubscription{
				client: client,
				scheme: scheme,
			}
			res, err := reconciler.Reconcile(scenario.Request)
			scenario.Verify(client, res, err, t)
		})
	}
}
