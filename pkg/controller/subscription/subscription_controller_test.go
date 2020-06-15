package subscription

import (
	"context"
	"testing"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	operatorNamespace = "openshift-operators"
)

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	err := v1alpha1.SchemeBuilder.AddToScheme(scheme)
	return scheme, err
}

func TestSubscriptionReconciler(t *testing.T) {
	scenarios := []struct {
		Name            string
		Request         reconcile.Request
		APISubscription *v1alpha1.Subscription
		Verify          func(client k8sclient.Client, res reconcile.Result, err error, t *testing.T)
	}{
		{
			Name: "subscription controller changes integreatly Subscription from automatic to manual",
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: operatorNamespace,
					Name:      IntegreatlyPackage,
				},
			},
			APISubscription: &v1alpha1.Subscription{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: operatorNamespace,
					Name:      IntegreatlyPackage,
				},
				Spec: &v1alpha1.SubscriptionSpec{
					InstallPlanApproval: v1alpha1.ApprovalAutomatic,
				},
			},
			Verify: func(c k8sclient.Client, res reconcile.Result, err error, t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
				sub := &v1alpha1.Subscription{}
				c.Get(context.TODO(), k8sclient.ObjectKey{Name: IntegreatlyPackage, Namespace: operatorNamespace}, sub)
				if sub.Spec.InstallPlanApproval != v1alpha1.ApprovalManual {
					t.Fatalf("expected Manual but got %s", sub.Spec.InstallPlanApproval)
				}
			},
		},
		{
			Name: "subscription controller doesn't change subscription in different namespace",
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "other-ns",
					Name:      IntegreatlyPackage,
				},
			},
			APISubscription: &v1alpha1.Subscription{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "other-ns",
					Name:      IntegreatlyPackage,
				},
				Spec: &v1alpha1.SubscriptionSpec{
					InstallPlanApproval: v1alpha1.ApprovalAutomatic,
				},
			},
			Verify: func(c k8sclient.Client, res reconcile.Result, err error, t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
				sub := &v1alpha1.Subscription{}
				c.Get(context.TODO(), k8sclient.ObjectKey{Name: IntegreatlyPackage, Namespace: "other-ns"}, sub)
				if sub.Spec.InstallPlanApproval != v1alpha1.ApprovalAutomatic {
					t.Fatalf("expected Automatic but got %s", sub.Spec.InstallPlanApproval)
				}
			},
		},
		{
			Name: "subscription controller doesn't change other subscription in the same namespace",
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: operatorNamespace,
					Name:      "other-package",
				},
			},
			APISubscription: &v1alpha1.Subscription{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: operatorNamespace,
					Name:      "other-package",
				},
				Spec: &v1alpha1.SubscriptionSpec{
					InstallPlanApproval: v1alpha1.ApprovalAutomatic,
				},
			},
			Verify: func(c k8sclient.Client, res reconcile.Result, err error, t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
				sub := &v1alpha1.Subscription{}
				c.Get(context.TODO(), k8sclient.ObjectKey{Name: "other-package", Namespace: operatorNamespace}, sub)
				if sub.Spec.InstallPlanApproval != v1alpha1.ApprovalAutomatic {
					t.Fatalf("expected Automatic but got %s", sub.Spec.InstallPlanApproval)
				}
			},
		},
		{
			Name: "subscription controller handles when subscription is missing",
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: operatorNamespace,
					Name:      IntegreatlyPackage,
				},
			},
			APISubscription: &v1alpha1.Subscription{},
			Verify: func(c k8sclient.Client, res reconcile.Result, err error, t *testing.T) {
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
				client:            client,
				scheme:            scheme,
				operatorNamespace: operatorNamespace,
			}
			res, err := reconciler.Reconcile(scenario.Request)
			scenario.Verify(client, res, err, t)
		})
	}
}
