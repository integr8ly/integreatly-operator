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
	oauthv1 "github.com/openshift/api/oauth/v1"
	usersv1 "github.com/openshift/api/user/v1"
	appsv1Client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	fakeoauthClient "github.com/openshift/client-go/oauth/clientset/versioned/fake"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"
	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	marketplacev2 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v2"
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
	err = marketplacev2.SchemeBuilder.AddToScheme(scheme)
	err = corev1.SchemeBuilder.AddToScheme(scheme)
	err = coreosv1.SchemeBuilder.AddToScheme(scheme)
	err = usersv1.AddToScheme(scheme)
	err = oauthv1.AddToScheme(scheme)
	return scheme, err
}

type ThreeScaleTestScenario struct {
	Name                 string
	Installation         *integreatlyv1alpha1.Installation
	FakeSigsClient       client.Client
	FakeAppsV1Client     appsv1Client.AppsV1Interface
	FakeOauthClient      oauthClient.OauthV1Interface
	FakeThreeScaleClient *ThreeScaleInterfaceMock
	ExpectedStatus       integreatlyv1alpha1.StatusPhase
	Assert               AssertFunc
	MPM                  marketplace.MarketplaceInterface
	Product              *v1alpha1.InstallationProductStatus
}

func TestThreeScale(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("Error creating build scheme")
	}

	scenarios := []ThreeScaleTestScenario{
		{
			Name:                 "Test successful installation without errors",
			FakeSigsClient:       getSigClient(getSuccessfullTestPreReqs(integreatlyOperatorNamespace, defaultInstallationNamespace), scheme),
			FakeAppsV1Client:     getAppsV1Client(successfulTestAppsV1Objects),
			FakeOauthClient:      fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeThreeScaleClient: getThreeScaleClient(),
			Assert:               assertInstallationSuccessfull,
			Installation: &v1alpha1.Installation{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-installation",
					Namespace:  "integreatly-operator-namespace",
					Finalizers: []string{"finalizer.3scale.integreatly.org"},
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
				},
				Spec: v1alpha1.InstallationSpec{
					MasterURL:        "https://console.apps.example.com",
					RoutingSubdomain: "apps.example.com",
				},
			},
			MPM:            marketplace.NewManager(),
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Product:        &v1alpha1.InstallationProductStatus{},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			ctx := context.TODO()
			configManager, err := config.NewManager(ctx, scenario.FakeSigsClient, configManagerConfigMap.Namespace, configManagerConfigMap.Name, scenario.Installation)
			if err != nil {
				t.Fatalf("Error creating config manager")
			}

			tsReconciler, err := NewReconciler(configManager, scenario.Installation, scenario.FakeAppsV1Client, scenario.FakeOauthClient, scenario.FakeThreeScaleClient, scenario.MPM)
			if err != nil {
				t.Fatalf("Error creating new reconciler %s: %v", packageName, err)
			}
			status, err := tsReconciler.Reconcile(ctx, scenario.Installation, scenario.Product, scenario.FakeSigsClient)
			if err != nil {
				t.Fatalf("Error reconciling %s: %v", packageName, err)
			}

			if status != scenario.ExpectedStatus {
				t.Fatalf("unexpected status: %v, expected: %v", status, scenario.ExpectedStatus)
			}

			err = scenario.Assert(scenario, configManager)
			if err != nil {
				t.Fatal(err.Error())
			}
		})
	}

}
