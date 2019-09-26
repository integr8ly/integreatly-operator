package solutionexplorer

import (
	"context"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/apis"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	fakeoauthClient "github.com/openshift/client-go/oauth/clientset/versioned/fake"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	operatorsv1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SolutionExplorerScenario struct {
	Name            string
	ExpectErr       bool
	ExpectedError   string
	ExpectedStatus  v1alpha1.StatusPhase
	client          client.Client
	FakeConfig      *config.ConfigReadWriterMock
	FakeMPM         *marketplace.MarketplaceInterfaceMock
	FakeOauthClient oauthClient.OauthV1Interface
	Installation    *v1alpha1.Installation
	OauthResolver   func() OauthResolver
	Validate        func(t *testing.T, mock interface{})
	Product         *v1alpha1.InstallationProductStatus
}

func basicConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		ReadSolutionExplorerFunc: func() (explorer *config.SolutionExplorer, e error) {
			return config.NewSolutionExplorer(config.ProductConfig{}), nil
		},
		ReadRHSSOFunc: func() (*config.RHSSO, error) {
			return config.NewRHSSO(config.ProductConfig{
				"NAMESPACE": "fuse",
				"URL":       "fuse.openshift-cluster.com",
			}), nil
		},
		WriteConfigFunc: func(config config.ConfigReadable) error {
			return nil
		},
	}
}

var oauthResolver = func() OauthResolver {
	return &OauthResolverMock{
		GetOauthEndPointFunc: func() (serverConfig *resources.OauthServerConfig, e error) {
			return &resources.OauthServerConfig{AuthorizationEndpoint: "http://test.com"}, nil
		},
	}
}

func TestReconciler_ReconcileCustomResource(t *testing.T) {
	// Initialize scheme so that types required by the scenarios are available
	scheme := scheme.Scheme
	if err := apis.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to initialize scheme: %s", err)
	}

	cases := []SolutionExplorerScenario{
		{
			Name:            " test custom resource is reconciled and phase complete returned",
			OauthResolver:   oauthResolver,
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			ExpectedStatus:  v1alpha1.PhaseCompleted,
			FakeMPM:         &marketplace.MarketplaceInterfaceMock{},
			Installation:    installation,
			FakeConfig:      basicConfigMock(),
			client:          fake.NewFakeClient(webappCR),
			Validate: func(t *testing.T, mock interface{}) {
				m := mock.(*OauthResolverMock)
				if len(m.GetOauthEndPointCalls()) != 1 {
					t.Fatal("expected 1 call to GetOauthEndPointCalls but got  ", len(m.GetOauthEndPointCalls()))
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			mockResolver := tc.OauthResolver()
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeOauthClient, tc.FakeMPM, mockResolver)
			if err != nil {
				t.Fatal("unexpected err settin up reconciler ", err)
			}
			status, err := reconciler.ReconcileCustomResource(context.TODO(), tc.Installation, tc.client)
			if tc.ExpectErr && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectErr && err != nil {
				t.Fatal("expected no error but got one ", err)
			}
			if tc.ExpectedStatus != status {
				t.Fatal("expected phase ", tc.ExpectedStatus, " but got ", status)
			}
			if tc.Validate != nil {
				tc.Validate(t, mockResolver)
			}
		})
	}
}

func TestSolutionExplorer(t *testing.T) {
	// Initialize scheme so that types required by the scenarios are available
	scheme := scheme.Scheme
	if err := apis.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to initialize scheme: %s", err)
	}

	cases := []SolutionExplorerScenario{
		{
			Name:            "test full reconcile",
			OauthResolver:   oauthResolver,
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			ExpectedStatus:  v1alpha1.PhaseCompleted,
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient client.Client, owner ownerutil.Owner, os operatorsv1.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient client.Client, subName string, ns string) (plan *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlan{
							ObjectMeta: v1.ObjectMeta{
								Name: "solution-explorer-install-plan",
							},
							Status: operatorsv1alpha1.InstallPlanStatus{
								Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
							},
						}, &operatorsv1alpha1.Subscription{
							Status: operatorsv1alpha1.SubscriptionStatus{
								Install: &operatorsv1alpha1.InstallPlanReference{
									Name: "solution-explorer-install-plan",
								},
							},
						}, nil
				},
			},
			Installation: installation,
			FakeConfig:   basicConfigMock(),
			client:       fake.NewFakeClient(webappNs, webappCR, installation, webappRoute),
			Product:      &v1alpha1.InstallationProductStatus{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeOauthClient, tc.FakeMPM, tc.OauthResolver())
			if err != nil && err.Error() != tc.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
			}

			if err == nil && tc.ExpectedError != "" {
				t.Fatalf("expected error '%v' and got nil", tc.ExpectedError)
			}

			// if we expect errors creating the reconciler, don't try to use it
			if tc.ExpectedError != "" {
				return
			}

			status, err := reconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.client)
			if err != nil && !tc.ExpectErr {
				t.Fatalf("expected error but got one: %v", err)
			}

			if err == nil && tc.ExpectErr {
				t.Fatal("expected error but got none")
			}

			if status != tc.ExpectedStatus {
				t.Fatalf("Expected status: '%v', got: '%v'", tc.ExpectedStatus, status)
			}
		})
	}
}
