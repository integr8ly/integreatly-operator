package threescale

import (
	"context"
	"testing"

	threescalev1 "github.com/integr8ly/integreatly-operator/pkg/apis/3scale/v1alpha1"
	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	appsv1Client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	fakeoauthClient "github.com/openshift/client-go/oauth/clientset/versioned/fake"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"
	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	integreatlyOperatorNamespace = "integreatly-operator-namespace"
)

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := threescalev1.SchemeBuilder.AddToScheme(scheme)
	err = aerogearv1.SchemeBuilder.AddToScheme(scheme)
	err = integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	err = operatorsv1alpha1.AddToScheme(scheme)
	err = marketplacev1.SchemeBuilder.AddToScheme(scheme)
	err = corev1.SchemeBuilder.AddToScheme(scheme)
	err = coreosv1.SchemeBuilder.AddToScheme(scheme)
	return scheme, err
}

func TestThreeScale(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("Error creating build scheme")
	}

	scenarios := []struct {
		Name                 string
		Installation         *integreatlyv1alpha1.Installation
		FakeSigsClient       client.Client
		FakeAppsV1Client     appsv1Client.AppsV1Interface
		FakeOauthClient      oauthClient.OauthV1Interface
		FakeThreeScaleClient *ThreeScaleInterfaceMock
		ExpectedStatus       integreatlyv1alpha1.StatusPhase
		AssertFunc           AssertFunc
		FakeMPM              *marketplace.MarketplaceInterfaceMock
	}{
		{
			Name:                 "Test successful installation without errors",
			FakeSigsClient:       getSigClient(getSuccessfullTestPreReqs(integreatlyOperatorNamespace, defaultInstallationNamespace), scheme),
			FakeAppsV1Client:     getAppsV1Client(successfulTestAppsV1Objects),
			FakeOauthClient:      fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeThreeScaleClient: getThreeScaleClient(),
			AssertFunc:           assertInstallationSuccessfull,
			Installation: &v1alpha1.Installation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-installation",
					Namespace: "integreatly-operator-namespace",
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
				},
				Spec: v1alpha1.InstallationSpec{
					MasterURL:        "https://console.apps.example.com",
					RoutingSubdomain: "apps.example.com",
				},
			},
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient client.Client, subName string, ns string) (plan *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlan{Status: operatorsv1alpha1.InstallPlanStatus{Phase: operatorsv1alpha1.InstallPlanPhaseComplete}}, nil, nil
				},
				InstallOperatorFunc: func(ctx context.Context, serverClient client.Client, owner ownerutil.Owner, os marketplacev1.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {

					return nil
				},
			},
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			ctx := context.TODO()
			configManager, err := config.NewManager(context.TODO(), scenario.FakeSigsClient, configManagerConfigMap.Namespace, configManagerConfigMap.Name)
			if err != nil {
				t.Fatalf("Error creating config manager")
			}

			testReconciler, err := NewReconciler(configManager, scenario.Installation, scenario.FakeAppsV1Client, scenario.FakeOauthClient, scenario.FakeThreeScaleClient, scenario.FakeMPM)
			if err != nil {
				t.Fatalf("Error creating new reconciler %s: %v", packageName, err)
			}
			status, err := testReconciler.Reconcile(ctx, scenario.Installation, scenario.FakeSigsClient)
			if err != nil {
				t.Fatalf("Error reconciling %s: %v", packageName, err)
			}

			if status != scenario.ExpectedStatus {
				t.Fatalf("unexpected status: %v, expected: %v", status, scenario.ExpectedStatus)
			}

			err = scenario.AssertFunc(scenario.Installation, configManager, scenario.FakeSigsClient, scenario.FakeThreeScaleClient, scenario.FakeAppsV1Client, scenario.FakeOauthClient, scenario.FakeMPM)
			if err != nil {
				t.Fatal(err.Error())
			}
		})
	}

}
