package controllers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/controllers/subscription/csvlocator"
	"github.com/integr8ly/integreatly-operator/controllers/subscription/webapp"

	catalogsourceClient "github.com/integr8ly/integreatly-operator/pkg/resources/catalogsource"
	v1 "k8s.io/api/core/v1"
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
	err := v1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return scheme, err
	}
	return scheme, integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
}

func intPtr(val int) *int    { return &val }
func boolPtr(val bool) *bool { return &val }

func TestSubscriptionReconciler(t *testing.T) {

	csv := &v1alpha1.ClusterServiceVersion{
		Spec: v1alpha1.ClusterServiceVersionSpec{
			Replaces: "123",
		},
	}
	csvStringfied, err := json.Marshal(csv)
	if err != nil {
		panic(err)
	}

	installPlan := &olmv1alpha1.InstallPlan{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "installplan",
			Namespace: operatorNamespace,
		},
		Status: olmv1alpha1.InstallPlanStatus{
			Plan: []*olmv1alpha1.Step{
				{
					Resource: olmv1alpha1.StepResource{
						Kind:     olmv1alpha1.ClusterServiceVersionKind,
						Manifest: string(csvStringfied),
					},
				},
			},
		},
	}

	rhmiConfig := &integreatlyv1alpha1.RHMIConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhmi-config",
			Namespace: operatorNamespace,
		},
		Spec: integreatlyv1alpha1.RHMIConfigSpec{
			Upgrade: integreatlyv1alpha1.Upgrade{
				NotBeforeDays:      intPtr(10),
				WaitForMaintenance: boolPtr(true),
			},
			Maintenance: integreatlyv1alpha1.Maintenance{
				ApplyFrom: "Thu 00:00",
			},
		},
	}

	rhmiCR := &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhmi",
			Namespace: operatorNamespace,
		},
	}

	scenarios := []struct {
		Name                string
		Request             reconcile.Request
		APISubscription     *v1alpha1.Subscription
		catalogsourceClient catalogsourceClient.CatalogSourceClientInterface
		Verify              func(client k8sclient.Client, res reconcile.Result, err error, t *testing.T)
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
				err = c.Get(context.TODO(), k8sclient.ObjectKey{Name: IntegreatlyPackage, Namespace: operatorNamespace}, sub)
				if err != nil {
					t.Fatalf("unexpected error getting subscription: %s", err.Error())
				}
				if sub.Spec.InstallPlanApproval != v1alpha1.ApprovalManual {
					t.Fatalf("expected Manual but got %s", sub.Spec.InstallPlanApproval)
				}
			},
			catalogsourceClient: getCatalogSourceClient(""),
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
				err = c.Get(context.TODO(), k8sclient.ObjectKey{Name: IntegreatlyPackage, Namespace: "other-ns"}, sub)
				if err != nil {
					t.Fatalf("unexpected error getting subscription : %s", err.Error())
				}
				if sub.Spec.InstallPlanApproval != v1alpha1.ApprovalAutomatic {
					t.Fatalf("expected Automatic but got %s", sub.Spec.InstallPlanApproval)
				}
			},
			catalogsourceClient: getCatalogSourceClient(""),
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
				err = c.Get(context.TODO(), k8sclient.ObjectKey{Name: "other-package", Namespace: operatorNamespace}, sub)
				if err != nil {
					t.Fatalf("unexpected error getting subscription: %s", err.Error())
				}
				if sub.Spec.InstallPlanApproval != v1alpha1.ApprovalAutomatic {
					t.Fatalf("expected Automatic but got %s", sub.Spec.InstallPlanApproval)
				}
			},
			catalogsourceClient: getCatalogSourceClient(""),
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
			catalogsourceClient: getCatalogSourceClient(""),
		},
		{
			Name: "subscription controller changes the subscription status block to trigger the recreation of a installplan",
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
				Spec: &olmv1alpha1.SubscriptionSpec{
					InstallPlanApproval: olmv1alpha1.ApprovalManual,
				},
				Status: v1alpha1.SubscriptionStatus{
					InstallPlanRef: &v1.ObjectReference{
						Name:      installPlan.Name,
						Namespace: installPlan.Namespace,
					},
					InstalledCSV: "123",
					CurrentCSV:   "124",
				},
			},
			Verify: func(c k8sclient.Client, res reconcile.Result, err error, t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				sub := &v1alpha1.Subscription{}
				err = c.Get(context.TODO(), k8sclient.ObjectKey{Name: IntegreatlyPackage, Namespace: operatorNamespace}, sub)
				if err != nil {
					t.Fatalf("unexpected error getting sublscription: %s", err.Error())
				}
				if sub.Status.InstalledCSV != sub.Status.CurrentCSV {
					t.Fatalf("expected installedCSV %s to be the same as currentCSV  %s", sub.Status.InstalledCSV, sub.Status.CurrentCSV)
				}
			},
			catalogsourceClient: getCatalogSourceClient(""),
		},
	}

	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("failed to build scheme: %s", err.Error())
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			APIObject := scenario.APISubscription
			client := fakeclient.NewFakeClientWithScheme(scheme, APIObject, installPlan, rhmiConfig, rhmiCR)
			reconciler := SubscriptionReconciler{
				Client:              client,
				Scheme:              scheme,
				catalogSourceClient: scenario.catalogsourceClient,
				operatorNamespace:   operatorNamespace,
				webbappNotifier:     &webapp.NoOp{},
				csvLocator:          &csvlocator.EmbeddedCSVLocator{},
			}
			res, err := reconciler.Reconcile(scenario.Request)
			scenario.Verify(client, res, err, t)
		})
	}
}

func TestShouldReconcileSubscription(t *testing.T) {
	scenarios := []struct {
		Name           string
		Namespace      string
		Request        reconcile.Request
		ExpectedResult bool
	}{
		{
			Name:      "Non matching namespace",
			Namespace: "testing-namespaces-operator",
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "integreatly",
					Namespace: "another",
				},
			},
			ExpectedResult: false,
		},
		{
			Name:      "Not in reconcile name list",
			Namespace: "testing-namespaces-operator",
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "another",
					Namespace: "testing-namespaces-operator",
				},
			},
			ExpectedResult: false,
		},
		{
			Name:      "\"integreatly\" subscription",
			Namespace: "testing-namespaces-operator",
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "integreatly",
					Namespace: "testing-namespaces-operator",
				},
			},
			ExpectedResult: true,
		},
		{
			Name:      "RHMI Addon subscription",
			Namespace: "testing-namespaces-operator",
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "addon-rhmi",
					Namespace: "testing-namespaces-operator",
				},
			},
			ExpectedResult: true,
		},
		{
			Name:      "Managed API Addon subscription",
			Namespace: "testing-namespaces-operator",
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "addon-managed-api-service",
					Namespace: "testing-namespaces-operator",
				},
			},
			ExpectedResult: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			reconciler := &SubscriptionReconciler{
				operatorNamespace: scenario.Namespace,
			}

			result := reconciler.shouldReconcileSubscription(scenario.Request)

			if result != scenario.ExpectedResult {
				t.Errorf("Unexpected result. Expected %v, got %v", scenario.ExpectedResult, result)
			}
		})
	}
}

func getCatalogSourceClient(replaces string) catalogsourceClient.CatalogSourceClientInterface {
	return &catalogsourceClient.CatalogSourceClientInterfaceMock{
		GetLatestCSVFunc: func(catalogSourceKey types.NamespacedName, packageName, channelName string) (*v1alpha1.ClusterServiceVersion, error) {
			return &v1alpha1.ClusterServiceVersion{
				Spec: v1alpha1.ClusterServiceVersionSpec{
					Replaces: replaces,
				},
			}, nil
		},
	}
}
