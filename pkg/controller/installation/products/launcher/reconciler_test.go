package launcher

import (
	"context"
	"testing"

	launcherv1alpha2 "github.com/fabric8-launcher/launcher-operator/pkg/apis/launcher/v1alpha2"
	keycloak "github.com/integr8ly/integreatly-operator/pkg/apis/keycloak/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	oauthv1 "github.com/openshift/api/oauth/v1"
	routev1 "github.com/openshift/api/route/v1"
	fakeappsv1Client "github.com/openshift/client-go/apps/clientset/versioned/fake"
	appsv1Client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	operatorsv1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	integreatlyOperatorNamespace = "integreatly-operator-namespace"
)

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := keycloak.SchemeBuilder.AddToScheme(scheme)
	err = integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	err = operatorsv1alpha1.AddToScheme(scheme)
	err = marketplacev1.SchemeBuilder.AddToScheme(scheme)
	err = corev1.SchemeBuilder.AddToScheme(scheme)
	err = coreosv1.SchemeBuilder.AddToScheme(scheme)
	err = launcherv1alpha2.SchemeBuilder.AddToScheme(scheme)
	err = routev1.AddToScheme(scheme)
	err = oauthv1.AddToScheme(scheme)
	return scheme, err
}

type LauncherTestScenario struct {
	Name             string
	Installation     *integreatlyv1alpha1.Installation
	FakeClient       client.Client
	FakeAppsV1Client appsv1Client.AppsV1Interface
	FakeMPM          *marketplace.MarketplaceInterfaceMock
	ExpectedStatus   integreatlyv1alpha1.StatusPhase
	Assert           AssertFunc
	Product          *v1alpha1.InstallationProductStatus
}

func TestLauncher(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("Error creating build scheme")
	}

	installation := &v1alpha1.Installation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-installation",
			Namespace: integreatlyOperatorNamespace,
			UID:       types.UID("xyz"),
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultInstallationNamespace,
			Labels: map[string]string{
				resources.OwnerLabelKey: string(installation.GetUID()),
			},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}

	scenarios := []LauncherTestScenario{
		{
			Name:             "Test successful installation without errors",
			FakeClient:       moqclient.NewSigsClientMoqWithScheme(scheme, append(getClusterPreReqObjects(integreatlyOperatorNamespace), installation, ns)...),
			FakeAppsV1Client: fakeappsv1Client.NewSimpleClientset(launcherDeploymentConfigs...).AppsV1(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient client.Client, owner ownerutil.Owner, os operatorsv1.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {
					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient client.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlanList, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlanList{
							Items: []operatorsv1alpha1.InstallPlan{
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "launcher-install-plan",
									},
									Status: operatorsv1alpha1.InstallPlanStatus{
										Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
									},
								},
							},
						}, &operatorsv1alpha1.Subscription{
							Status: operatorsv1alpha1.SubscriptionStatus{
								Install: &operatorsv1alpha1.InstallPlanReference{
									Name: "launcher-install-plan",
								},
							},
						}, nil
				},
			},
			Installation:   installation,
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Assert:         assertInstallationSuccessfullyReconciled,
			Product:        &v1alpha1.InstallationProductStatus{},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			ctx := context.TODO()
			configManager, err := config.NewManager(context.TODO(), scenario.FakeClient, configManagerConfigMap.Namespace, configManagerConfigMap.Name, scenario.Installation)
			if err != nil {
				t.Fatalf("Error creating config manager")
			}

			testReconciler, err := NewReconciler(configManager, scenario.Installation, scenario.FakeAppsV1Client, scenario.FakeMPM)
			status, err := testReconciler.Reconcile(ctx, scenario.Installation, scenario.Product, scenario.FakeClient)
			if err != nil {
				t.Fatalf("Error reconciling %s: %v", defaultLauncherName, err)
			}

			if status != scenario.ExpectedStatus {
				t.Fatalf("unexpected status: %v, expected: %v", status, scenario.ExpectedStatus)
			}

			err = scenario.Assert(scenario, configManager, scenario.FakeMPM)
			if err != nil {
				t.Fatal(err.Error())
			}
		})
	}
}
