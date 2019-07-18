package codeready

import (
	"context"
	"errors"
	chev1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	operatorsv1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	kafkav1 "github.com/integr8ly/integreatly-operator/pkg/apis/kafka.strimzi.io/v1alpha1"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	corev1 "k8s.io/api/core/v1"

	rbacv1 "k8s.io/api/rbac/v1"

	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
)

func basicConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		GetOperatorNamespaceFunc: func() string {
			return "integreatly-operator"
		},
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
	aerogearv1.SchemeBuilder.AddToScheme(scheme)
	rbacv1.SchemeBuilder.AddToScheme(scheme)
	batchv1beta1.SchemeBuilder.AddToScheme(scheme)
	appsv1.SchemeBuilder.AddToScheme(scheme)
	return scheme
}

func TestCodeready(t *testing.T) {
	testKeycloakRealm := aerogearv1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "openshift",
			Namespace: "rhsso",
		},
		Spec: aerogearv1.KeycloakRealmSpec{
			CreateOnly: false,
			KeycloakApiRealm: &aerogearv1.KeycloakApiRealm{
				Clients: []*aerogearv1.KeycloakClient{
					&aerogearv1.KeycloakClient{
						KeycloakApiClient: &aerogearv1.KeycloakApiClient{
							Name: defaultClientName,
						},
					},
				},
			},
		},
	}

	testCheCluster := chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defaultInstallationNamespace,
			Name:      defaultCheClusterName,
		},
		Spec: chev1.CheClusterSpec{},
		Status: chev1.CheClusterStatus{
			CheURL: "https://test.com",
		},
	}

	scenarios := []struct {
		Name                 string
		ExpectedStatus       v1alpha1.StatusPhase
		ExpectedError        string
		ExpectedCreateError  string
		Object               *v1alpha1.Installation
		FakeConfig           *config.ConfigReadWriterMock
		FakeControllerClient client.Client
		FakeMPM              *marketplace.MarketplaceInterfaceMock
		ValidateCallCounts   func(mockConfig *config.ConfigReadWriterMock, mockMPM *marketplace.MarketplaceInterfaceMock, t *testing.T)
	}{
		{
			Name:           "test successful installation without errors",
			ExpectedStatus: v1alpha1.PhaseCompleted,
			Object: &v1alpha1.Installation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "installation",
					APIVersion: "integreatly.org/v1alpha1",
				},
			},
			FakeControllerClient: pkgclient.NewFakeClientWithScheme(buildScheme(), &testKeycloakRealm, &testCheCluster, &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: defaultInstallationNamespace,
					Name:      "postgres",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name:  "POSTGRES_USERNAME",
											Value: "username",
										},
										{
											Name:  "POSTGRES_PASSWORD",
											Value: "password",
										},
										{
											Name:  "POSTGRES_DATABASE",
											Value: "database",
										},
									},
								},
							},
						},
					},
				},
			}),
			FakeConfig: basicConfigMock(),
			ValidateCallCounts: func(mockConfig *config.ConfigReadWriterMock, mockMPM *marketplace.MarketplaceInterfaceMock, t *testing.T) {
				if len(mockConfig.ReadCodeReadyCalls()) != 1 {
					t.Fatalf("expected 1 call to readCodeReady config, got: %d", len(mockConfig.ReadCodeReadyCalls()))
				}
				if len(mockConfig.ReadRHSSOCalls()) != 1 {
					t.Fatalf("expected 1 call to readRHSSO config, got: %d", len(mockConfig.ReadCodeReadyCalls()))
				}
				if mockMPM == nil {
					t.Fatalf("expected MPM not to be nil but it was nil ")
				}
				if len(mockMPM.CreateSubscriptionCalls()) != 1 {
					t.Fatalf("expected CreateSubscriptionCalls to be 1 bug got %d", len(mockMPM.CreateSubscriptionCalls()))
				}
				if len(mockMPM.GetSubscriptionInstallPlanCalls()) != 1 {
					t.Fatalf("expected GetSubscriptionInstallPlanCalls to be 1 bug got %d", len(mockMPM.GetSubscriptionInstallPlanCalls()))
				}
			},
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				CreateSubscriptionFunc: func(ctx context.Context, serverClient client.Client, owner ownerutil.Owner, os marketplacev1.OperatorSource, ns string, pkg string, channel string, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient client.Client, subName string, ns string) (plan *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlan{}, &operatorsv1alpha1.Subscription{}, nil
				},
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
			Name:           "test subscription phase with error from mpm",
			ExpectedStatus: v1alpha1.PhaseFailed,
			ExpectedError:  "could not reconcile subscription: could not create subscription in namespace: codeready-workspaces: dummy error",
			Object: &v1alpha1.Installation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "installation",
					Namespace: "installation-namespace",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "installation",
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
				},
				Status: v1alpha1.InstallationStatus{
					ProductStatus: map[v1alpha1.ProductName]string{
						v1alpha1.ProductCodeReadyWorkspaces: string(v1alpha1.PhaseCreatingSubscription),
					},
				},
			},
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				CreateSubscriptionFunc: func(ctx context.Context, serverClient client.Client, owner ownerutil.Owner, os operatorsv1.OperatorSource, ns string, pkg string, channel string, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {
					return errors.New("dummy error")
				},
			},
			FakeControllerClient: pkgclient.NewFakeClient(
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: defaultInstallationNamespace,
						OwnerReferences: []metav1.OwnerReference{
							{
								Name:       "installation",
								APIVersion: v1alpha1.SchemeGroupVersion.String(),
							},
						},
					},
					Status: corev1.NamespaceStatus{},
				}),
			FakeConfig: basicConfigMock(),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			testReconciler, err := NewReconciler(
				scenario.FakeConfig,
				scenario.Object,
				scenario.FakeMPM,
			)
			if err != nil && err.Error() != scenario.ExpectedCreateError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, scenario.ExpectedCreateError)
			}

			if err == nil && scenario.ExpectedCreateError != "" {
				t.Fatalf("expected error '%v' and got nil", scenario.ExpectedCreateError)
			}

			// if we expect errors creating the reconciler, don't try to use it
			if scenario.ExpectedCreateError != "" {
				return
			}

			status, err := testReconciler.Reconcile(context.TODO(), scenario.Object, scenario.FakeControllerClient)
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
