package solutionexplorer

import (
	"context"
	"testing"

	v1alpha12 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/tutorial-web-app-operator/pkg/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	fakeoauthClient "github.com/openshift/client-go/oauth/clientset/versioned/fake"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

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
	}
}

func TestReconciler_ReconcileCustomResource(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha12.AddToScheme(scheme)
	cases := []struct {
		Name            string
		client          client.Client
		FakeConfig      *config.ConfigReadWriterMock
		Installation    *v1alpha1.Installation
		ExpectErr       bool
		ExpectedStatus  v1alpha1.StatusPhase
		OauthResolver   func() OauthResolver
		FakeMPM         *marketplace.MarketplaceInterfaceMock
		FakeOauthClient oauthClient.OauthV1Interface
		Validate        func(t *testing.T, mock interface{})
	}{
		{
			Name: " test custom resource is reconciled and phase complete returned",
			OauthResolver: func() OauthResolver {
				return &OauthResolverMock{
					GetOauthEndPointFunc: func() (serverConfig *resources.OauthServerConfig, e error) {
						return &resources.OauthServerConfig{AuthorizationEndpoint: "http://test.com"}, nil
					},
				}
			},
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			ExpectedStatus:  v1alpha1.PhaseCompleted,
			FakeMPM:         &marketplace.MarketplaceInterfaceMock{},
			Installation: &v1alpha1.Installation{
				TypeMeta: v1.TypeMeta{
					Kind:       "Installation",
					APIVersion: "v1alpha1",
				},
			},
			FakeConfig: basicConfigMock(),
			client: fake.NewFakeClientWithScheme(scheme, &v1alpha12.WebApp{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "solution-explorer",
					Name:      "solution-explorer",
				},
				Status: v1alpha12.WebAppStatus{
					Message: "OK",
				},
			}),
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
