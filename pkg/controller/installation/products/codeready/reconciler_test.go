package codeready

import (
	"errors"
	chev1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	operatorsv1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	k8sclient "k8s.io/client-go/kubernetes/fake"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	kafkav1 "github.com/integr8ly/integreatly-operator/pkg/apis/kafka.strimzi.io/v1alpha1"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	corev1 "k8s.io/api/core/v1"
)

func basicConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		ReadCodeReadyFunc: func() (ready *config.CodeReady, e error) {
			return config.NewCodeReady(config.ProductConfig{}), nil
		},
		ReadRHSSOFunc: func() (*config.RHSSO, error) {
			return config.NewRHSSO(config.ProductConfig{
				"NAMESPACE": "rhsso",
				"REALM":     "openshift",
				"URL":       "rhsso.openshift-cluster.com",
			}), nil
		},
	}
}

func buildScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	chev1.SchemeBuilder.AddToScheme(scheme)
	aerogearv1.SchemeBuilder.AddToScheme(scheme)
	v1alpha1.SchemeBuilder.AddToScheme(scheme)
	operatorsv1alpha1.AddToScheme(scheme)
	marketplacev1.SchemeBuilder.AddToScheme(scheme)
	kafkav1.SchemeBuilder.AddToScheme(scheme)
	corev1.SchemeBuilder.AddToScheme(scheme)
	return scheme
}

func TestCodeready(t *testing.T) {
	scenarios := []struct {
		Name                 string
		ExpectedStatus       v1alpha1.StatusPhase
		ExpectedError        string
		ExpectedCreateError  string
		Object               *v1alpha1.Installation
		FakeConfig           *config.ConfigReadWriterMock
		FakeK8sClient        *k8sclient.Clientset
		FakeControllerClient client.Client
		FakeMPM              *marketplace.MarketplaceInterfaceMock
		ValidateCallCounts   func(mockConfig *config.ConfigReadWriterMock, mockMPM *marketplace.MarketplaceInterfaceMock, t *testing.T)
	}{
		{
			Name:                 "test no phase without errors",
			ExpectedStatus:       v1alpha1.PhaseAwaitingNS,
			Object:               &v1alpha1.Installation{},
			FakeControllerClient: pkgclient.NewFakeClient(),
			FakeConfig:           basicConfigMock(),
			ValidateCallCounts: func(mockConfig *config.ConfigReadWriterMock, mockMPM *marketplace.MarketplaceInterfaceMock, t *testing.T) {
				if len(mockConfig.ReadCodeReadyCalls()) != 1 {
					t.Fatalf("expected 1 call to readCodeReady config, got: %d", len(mockConfig.ReadCodeReadyCalls()))
				}
				if len(mockConfig.ReadRHSSOCalls()) != 1 {
					t.Fatalf("expected 1 call to readRHSSO config, got: %d", len(mockConfig.ReadCodeReadyCalls()))
				}
				if mockMPM != nil {
					t.Fatalf("expected MPM to be nil, got: %+v", mockMPM)
				}
			},
		},
		{
			Name:                 "test error on bad codeready config",
			ExpectedStatus:       v1alpha1.PhaseNone,
			ExpectedCreateError:  "could not retrieve che config: could not load codeready config",
			Object:               &v1alpha1.Installation{},
			FakeControllerClient: pkgclient.NewFakeClient(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadCodeReadyFunc: func() (ready *config.CodeReady, e error) {
					return nil, errors.New("could not load codeready config")
				},
			},
		},
		{
			Name:                 "test error on bad RHSSO config",
			ExpectedStatus:       v1alpha1.PhaseNone,
			ExpectedCreateError:  "keycloak config is not valid: config realm is not defined",
			Object:               &v1alpha1.Installation{},
			FakeControllerClient: pkgclient.NewFakeClient(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadCodeReadyFunc: func() (ready *config.CodeReady, e error) {
					return config.NewCodeReady(config.ProductConfig{}), nil
				},
				ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{}), nil
				},
			},
		},
		{
			Name:                 "test awaiting subscrition - failed RHSSO config",
			ExpectedStatus:       v1alpha1.PhaseNone,
			ExpectedCreateError:  "could not retrieve keycloak config: could not load keycloak config",
			Object:               &v1alpha1.Installation{},
			FakeControllerClient: pkgclient.NewFakeClient(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadCodeReadyFunc: func() (ready *config.CodeReady, e error) {
					return config.NewCodeReady(config.ProductConfig{}), nil
				},
				ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return nil, errors.New("could not load keycloak config")
				},
			},
		},
		{
			Name:                 "test error on failed RHSSO config",
			ExpectedStatus:       v1alpha1.PhaseNone,
			ExpectedCreateError:  "could not retrieve keycloak config: could not load keycloak config",
			Object:               &v1alpha1.Installation{},
			FakeControllerClient: pkgclient.NewFakeClient(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadCodeReadyFunc: func() (ready *config.CodeReady, e error) {
					return config.NewCodeReady(config.ProductConfig{}), nil
				},
				ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return nil, errors.New("could not load keycloak config")
				},
			},
		},
		{
			Name:                 "test no phase with creatNamespaces",
			ExpectedStatus:       v1alpha1.PhaseAwaitingNS,
			Object:               &v1alpha1.Installation{},
			FakeControllerClient: pkgclient.NewFakeClient(),
			FakeConfig:           basicConfigMock(),
		},
		{
			Name:           "test subscription phase",
			ExpectedStatus: v1alpha1.PhaseAwaitingOperator,
			Object: &v1alpha1.Installation{
				Status: v1alpha1.InstallationStatus{
					ProductStatus: map[v1alpha1.ProductName]string{
						v1alpha1.ProductCodeReadyWorkspaces: string(v1alpha1.PhaseCreatingSubscription),
					},
				},
			},
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				CreateSubscriptionFunc: func(os operatorsv1.OperatorSource, ns string, pkg string, channel string, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {
					return nil
				},
			},
			FakeControllerClient: pkgclient.NewFakeClient(),
			FakeConfig:           basicConfigMock(),
			ValidateCallCounts: func(mockConfig *config.ConfigReadWriterMock, mockMPM *marketplace.MarketplaceInterfaceMock, t *testing.T) {
				if mockMPM == nil {
					t.Fatalf("expected MPM to not be nil, got: %+v", mockMPM)
				}
				if len(mockMPM.CreateSubscriptionCalls()) != 1 {
					t.Fatalf("expected 1 call to mockMPM.CreateSubscription, got: %d", len(mockMPM.CreateSubscriptionCalls()))
				}
			},
		},
		{
			Name:           "test subscription phase with error from mpm",
			ExpectedStatus: v1alpha1.PhaseFailed,
			ExpectedError:  "could not create subscription in namespace: codeready-workspaces: dummy error",
			Object: &v1alpha1.Installation{
				Status: v1alpha1.InstallationStatus{
					ProductStatus: map[v1alpha1.ProductName]string{
						v1alpha1.ProductCodeReadyWorkspaces: string(v1alpha1.PhaseCreatingSubscription),
					},
				},
			},
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				CreateSubscriptionFunc: func(os operatorsv1.OperatorSource, ns string, pkg string, channel string, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {
					return errors.New("dummy error")
				},
			},
			FakeControllerClient: pkgclient.NewFakeClient(),
			FakeConfig:           basicConfigMock(),
		},
		{
			Name:           "test creating components phase",
			ExpectedStatus: v1alpha1.PhaseInProgress,
			Object: &v1alpha1.Installation{
				Status: v1alpha1.InstallationStatus{
					ProductStatus: map[v1alpha1.ProductName]string{
						v1alpha1.ProductCodeReadyWorkspaces: string(v1alpha1.PhaseCreatingComponents),
					},
				},
			},
			FakeControllerClient: pkgclient.NewFakeClientWithScheme(buildScheme(), &aerogearv1.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "openshift",
					Namespace: "rhsso",
				},
			}, &chev1.CheCluster{ObjectMeta: metav1.ObjectMeta{Name: defaultCheClusterName, Namespace: defaultInstallationNamespace}}),
			FakeConfig: basicConfigMock(),
		},
		{
			Name:           "test creating components phase missing cluster",
			ExpectedStatus: v1alpha1.PhaseInProgress,
			Object: &v1alpha1.Installation{
				Status: v1alpha1.InstallationStatus{
					ProductStatus: map[v1alpha1.ProductName]string{
						v1alpha1.ProductCodeReadyWorkspaces: string(v1alpha1.PhaseCreatingComponents),
					},
				},
			},
			FakeControllerClient: pkgclient.NewFakeClientWithScheme(buildScheme(), &aerogearv1.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "openshift",
					Namespace: "rhsso",
				},
			}),
			FakeConfig: basicConfigMock(),
		},
		{
			Name:           "test awaiting operator phase, not yet ready",
			ExpectedStatus: v1alpha1.PhaseAwaitingOperator,
			Object: &v1alpha1.Installation{
				Status: v1alpha1.InstallationStatus{
					ProductStatus: map[v1alpha1.ProductName]string{
						v1alpha1.ProductCodeReadyWorkspaces: string(v1alpha1.PhaseAwaitingOperator),
					},
				},
			},
			FakeControllerClient: pkgclient.NewFakeClientWithScheme(buildScheme()),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				GetSubscriptionInstallPlanFunc: func(subName string, ns string) (plan *operatorsv1alpha1.InstallPlan, sub *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlan{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: ns,
							Name:      subName,
						},
					}, nil, nil
				},
			},
			FakeConfig: basicConfigMock(),
			ValidateCallCounts: func(mockConfig *config.ConfigReadWriterMock, mockMPM *marketplace.MarketplaceInterfaceMock, t *testing.T) {
				if mockMPM == nil {
					t.Fatalf("expected MPM to not be nil, got: %+v", mockMPM)
				}
				if len(mockMPM.GetSubscriptionInstallPlanCalls()) != 1 {
					t.Fatalf("expected 1 call to mockMPM.GetSubscriptionInstallPlan, got: %d", len(mockMPM.GetSubscriptionInstallPlanCalls()))
				}
			},
		},
		{
			Name:           "test awaiting operator phase, ready",
			ExpectedStatus: v1alpha1.PhaseCreatingComponents,
			Object: &v1alpha1.Installation{
				Status: v1alpha1.InstallationStatus{
					ProductStatus: map[v1alpha1.ProductName]string{
						v1alpha1.ProductCodeReadyWorkspaces: string(v1alpha1.PhaseAwaitingOperator),
					},
				},
			},
			FakeControllerClient: pkgclient.NewFakeClientWithScheme(buildScheme()),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				GetSubscriptionInstallPlanFunc: func(subName string, ns string) (plan *operatorsv1alpha1.InstallPlan, sub *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlan{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: ns,
							Name:      subName,
						},
						Status: operatorsv1alpha1.InstallPlanStatus{
							Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
						},
					}, nil, nil
				},
			},
			FakeConfig: basicConfigMock(),
			ValidateCallCounts: func(mockConfig *config.ConfigReadWriterMock, mockMPM *marketplace.MarketplaceInterfaceMock, t *testing.T) {
				if mockMPM == nil {
					t.Fatalf("expected MPM to not be nil, got: %+v", mockMPM)
				}
				if len(mockMPM.GetSubscriptionInstallPlanCalls()) != 1 {
					t.Fatalf("expected 1 call to mockMPM.GetSubscriptionInstallPlan, got: %d", len(mockMPM.GetSubscriptionInstallPlanCalls()))
				}
			},
		},
		{
			Name:           "test in progress phase, not ready",
			ExpectedStatus: v1alpha1.PhaseInProgress,
			Object: &v1alpha1.Installation{
				Status: v1alpha1.InstallationStatus{
					ProductStatus: map[v1alpha1.ProductName]string{
						v1alpha1.ProductCodeReadyWorkspaces: string(v1alpha1.PhaseInProgress),
					},
				},
			},
			FakeControllerClient: pkgclient.NewFakeClientWithScheme(buildScheme(), &aerogearv1.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "openshift",
					Namespace: "rhsso",
				},
			}, &chev1.CheCluster{ObjectMeta: metav1.ObjectMeta{Name: defaultCheClusterName, Namespace: defaultInstallationNamespace}}),
			FakeConfig: basicConfigMock(),
		},
		{
			Name:           "test in progress phase, ready",
			ExpectedStatus: v1alpha1.PhaseCompleted,
			Object: &v1alpha1.Installation{
				Status: v1alpha1.InstallationStatus{
					ProductStatus: map[v1alpha1.ProductName]string{
						v1alpha1.ProductCodeReadyWorkspaces: string(v1alpha1.PhaseInProgress),
					},
				},
			},
			FakeControllerClient: pkgclient.NewFakeClientWithScheme(buildScheme(), &aerogearv1.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "openshift",
					Namespace: "rhsso",
				},
			}, &chev1.CheCluster{
				ObjectMeta: metav1.ObjectMeta{Name: defaultCheClusterName, Namespace: defaultInstallationNamespace},
				Status: chev1.CheClusterStatus{
					CheClusterRunning: "Available",
				},
			}),
			FakeConfig: basicConfigMock(),
		},
		{
			Name:           "test completed phase",
			ExpectedStatus: v1alpha1.PhaseCompleted,
			Object: &v1alpha1.Installation{
				Status: v1alpha1.InstallationStatus{
					ProductStatus: map[v1alpha1.ProductName]string{
						v1alpha1.ProductCodeReadyWorkspaces: string(v1alpha1.PhaseCompleted),
					},
				},
			},
			FakeControllerClient: pkgclient.NewFakeClientWithScheme(buildScheme(), &aerogearv1.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "openshift",
					Namespace: "rhsso",
				},
				Spec: aerogearv1.KeycloakRealmSpec{
					KeycloakApiRealm: &aerogearv1.KeycloakApiRealm{
						Clients: []*aerogearv1.KeycloakClient{
							{
								KeycloakApiClient: &aerogearv1.KeycloakApiClient{
									ClientID: defaultClientName,
								},
							},
						},
					},
				},
			}, &chev1.CheCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultCheClusterName,
					Namespace: defaultInstallationNamespace,
				},
				Status: chev1.CheClusterStatus{
					CheURL: "some.url",
				},
			}),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				CreateSubscriptionFunc: func(os marketplacev1.OperatorSource, ns string, pkg string, channel string, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {
					return nil
				},
			},
			FakeConfig: basicConfigMock(),
			ValidateCallCounts: func(mockConfig *config.ConfigReadWriterMock, mockMPM *marketplace.MarketplaceInterfaceMock, t *testing.T) {
				if mockMPM == nil {
					t.Fatalf("expected MPM to not be nil, got: %+v", mockMPM)
				}
				if len(mockMPM.CreateSubscriptionCalls()) != 1 {
					t.Fatalf("expected 1 call to mockMPM.CreateSubscription, got: %d", len(mockMPM.CreateSubscriptionCalls()))
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(*testing.T) {
			testReconciler, err := NewReconciler(
				scenario.FakeConfig,
				scenario.Object,
				scenario.FakeMPM,
			)
			if err != nil && err.Error() != scenario.ExpectedCreateError {
				t.Fatalf("unexpected error creating reconciler: '%v', expected: '%v'", err, scenario.ExpectedCreateError)
			}

			if err == nil && scenario.ExpectedCreateError != "" {
				t.Fatalf("expected error '%v' and got nil", scenario.ExpectedCreateError)
			}

			// if we expect errors creating the reconciler, don't try to use it
			if scenario.ExpectedCreateError != "" {
				return
			}

			status, err := testReconciler.Reconcile(scenario.Object, scenario.FakeControllerClient)
			if err != nil && err.Error() != scenario.ExpectedError {
				t.Fatalf("unexpected error: %v, expected: %v", err, scenario.ExpectedError)
			}

			if err == nil && scenario.ExpectedError != "" {
				t.Fatalf("expected error '%v' and got nil", scenario.ExpectedError)
			}

			if status != scenario.ExpectedStatus {
				t.Fatalf("Expected status: '%v', got: '%v'", scenario.ExpectedStatus, status)
			}

			if scenario.ValidateCallCounts != nil {
				scenario.ValidateCallCounts(scenario.FakeConfig, scenario.FakeMPM, t)
			}
		})
	}
}
