package rhsso

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"github.com/integr8ly/integreatly-operator/utils"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	keycloakCommon "github.com/integr8ly/keycloak-client/pkg/common"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"
	keycloak "github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"
	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	fakeoauthClient "github.com/openshift/client-go/oauth/clientset/versioned/fake"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	crov1 "github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1"
	croTypes "github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	defaultOperatorNamespace = "integreatly-operator"
	localProductDeclaration  = marketplace.LocalProductDeclaration("integreatly-rhsso")
)

func basicConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		GetOperatorNamespaceFunc: func() string {
			return defaultOperatorNamespace
		},
		ReadRHSSOFunc: func() (*config.RHSSO, error) {
			return config.NewRHSSO(config.ProductConfig{
				"NAMESPACE": "rhsso",
				"REALM":     "openshift",
				"URL":       "rhsso.openshift-cluster.com",
				"HOST":      "edge/route",
			}), nil
		},
		WriteConfigFunc: func(config config.ConfigReadable) error {
			return nil
		},
		GetOauthClientsSecretNameFunc: func() string {
			return "oauth-client-secrets"
		},
		GetGHOauthClientsSecretNameFunc: func() string {
			return "github-oauth-secret"
		},
	}
}

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
}

func getRHSSOCredentialSeed() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      adminCredentialSecretName,
			Namespace: defaultOperandNamespace,
		},
		Data: map[string][]byte{},
		Type: corev1.SecretTypeOpaque,
	}
}

func TestReconciler_config(t *testing.T) {
	cases := []struct {
		Name                  string
		ExpectError           bool
		ExpectedStatus        integreatlyv1alpha1.StatusPhase
		ExpectedError         string
		FakeConfig            *config.ConfigReadWriterMock
		FakeClient            k8sclient.Client
		FakeOauthClient       oauthClient.OauthV1Interface
		FakeMPM               *marketplace.MarketplaceInterfaceMock
		Installation          *integreatlyv1alpha1.RHMI
		Product               *integreatlyv1alpha1.RHMIProductStatus
		Recorder              record.EventRecorder
		APIURL                string
		KeycloakClientFactory keycloakCommon.KeycloakClientFactory
		Uninstall             bool
	}{
		{
			Name:            "test error on failed config",
			ExpectedStatus:  integreatlyv1alpha1.PhaseFailed,
			ExpectError:     true,
			ExpectedError:   "could not read rhsso config",
			Installation:    &integreatlyv1alpha1.RHMI{},
			FakeClient:      utils.NewTestClient(runtime.NewScheme()),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadRHSSOFunc: func() (ready *config.RHSSO, e error) {
					return nil, errors.New("could not read rhsso config")
				},
			},
			Product:               &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:              setupRecorder(),
			APIURL:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
			Uninstall:             false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			testReconciler, err := NewReconciler(
				tc.FakeConfig,
				tc.Installation,
				tc.FakeOauthClient,
				tc.FakeMPM,
				tc.Recorder,
				tc.APIURL,
				tc.KeycloakClientFactory,
				getLogger(),
				localProductDeclaration,
			)
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

			status, err := testReconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.FakeClient, &quota.ProductConfigMock{}, tc.Uninstall)
			if err != nil && !tc.ExpectError {
				t.Fatalf("expected no errors, but got one: %v", err)
			}

			if err == nil && tc.ExpectError {
				t.Fatal("expected error but got none")
			}

			if status != tc.ExpectedStatus {
				t.Fatalf("Expected status: '%v', got: '%v'", tc.ExpectedStatus, status)
			}
		})
	}
}

func TestReconciler_reconcileComponents(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	kc := &keycloak.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: defaultOperandNamespace,
		},
	}

	kcr := getKcr(keycloak.KeycloakRealmStatus{
		Phase: keycloak.PhaseReconciling,
	})

	credentialRhsso := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      adminCredentialSecretName,
			Namespace: "rhsso",
		},
		Data: map[string][]byte{
			"rhsso": bytes.NewBufferString("test").Bytes(),
		},
	}

	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oauth-client-secrets",
			Namespace: defaultOperatorNamespace,
		},
		Data: map[string][]byte{
			"rhsso": bytes.NewBufferString("test").Bytes(),
		},
	}

	githubOauthSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "github-oauth-secret",
			Namespace: defaultOperatorNamespace,
		},
		Data: map[string][]byte{
			"clientId": bytes.NewBufferString("dummy").Bytes(),
			"secret":   bytes.NewBufferString("dummy").Bytes(),
		},
	}

	//completed postgres that points at the secret croPostgresSecret
	croPostgres := &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhsso-postgres-",
			Namespace: defaultOperatorNamespace,
		},
		Status: croTypes.ResourceTypeStatus{
			Phase: croTypes.PhaseComplete,
			SecretRef: &croTypes.SecretRef{
				Name:      "test",
				Namespace: defaultOperatorNamespace,
			},
		},
	}

	infrastructureAws := &configv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Status: configv1.InfrastructureStatus{
			PlatformStatus: &configv1.PlatformStatus{
				Type: configv1.AWSPlatformType,
			},
		},
	}

	//secret created by the cloud resource operator postgres reconciler
	croPostgresSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: defaultOperatorNamespace,
		},
		Data: map[string][]byte{},
		Type: corev1.SecretTypeOpaque,
	}

	installation := integreatlyv1alpha1.RHMI{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RHMI",
			APIVersion: integreatlyv1alpha1.GroupVersion.String(),
		},
		ObjectMeta: controllerruntime.ObjectMeta{
			Namespace: defaultOperatorNamespace,
		},
	}

	cases := []struct {
		Name                  string
		FakeClient            k8sclient.Client
		FakeOauthClient       oauthClient.OauthV1Interface
		FakeConfig            *config.ConfigReadWriterMock
		Installation          *integreatlyv1alpha1.RHMI
		ExpectError           bool
		ExpectedError         string
		ExpectedStatus        integreatlyv1alpha1.StatusPhase
		FakeMPM               *marketplace.MarketplaceInterfaceMock
		Recorder              record.EventRecorder
		ApiUrl                string
		KeycloakClientFactory keycloakCommon.KeycloakClientFactory
		ProductDeclaration    *marketplace.ProductDeclaration
	}{
		{
			Name:                  "Test reconcile custom resource returns completed when successful created",
			FakeClient:            utils.NewTestClient(scheme, oauthClientSecrets, githubOauthSecret, kc, croPostgres, croPostgresSecret, kcr, credentialRhsso, infrastructureAws),
			FakeOauthClient:       fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:            basicConfigMock(),
			Installation:          &installation,
			ExpectedStatus:        integreatlyv1alpha1.PhaseCompleted,
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
			ProductDeclaration:    localProductDeclaration,
		},
		{
			Name:                  "No product declaration found for RHSSO",
			FakeClient:            utils.NewTestClient(scheme, oauthClientSecrets, githubOauthSecret, kc, croPostgres, croPostgresSecret, kcr, credentialRhsso, infrastructureAws),
			FakeOauthClient:       fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:            basicConfigMock(),
			Installation:          &installation,
			ExpectedStatus:        integreatlyv1alpha1.PhaseCompleted,
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
			ProductDeclaration:    nil,
		},
		{
			Name: "Test reconcile custom resource returns completed when successful created (on upgrade, use rolling strategy)",
			FakeClient: utils.NewTestClient(scheme, oauthClientSecrets, githubOauthSecret, &keycloak.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      keycloakName,
					Namespace: defaultOperandNamespace,
				},
				Status: keycloak.KeycloakStatus{
					Ready:   true,
					Version: string(integreatlyv1alpha1.OperatorVersionRHSSO),
				},
			}, croPostgres, croPostgresSecret, kcr, credentialRhsso, infrastructureAws),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig: func() *config.ConfigReadWriterMock {
				basicConfig := basicConfigMock()
				basicConfig.ReadRHSSOFunc = func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
						"REALM":     "openshift",
						"URL":       "rhsso.openshift-cluster.com",
						"HOST":      "edge/route",
						"VERSION":   string(integreatlyv1alpha1.VersionRHSSO),
					}), nil
				}
				return basicConfig
			}(),
			Installation:          &installation,
			ExpectedStatus:        integreatlyv1alpha1.PhaseCompleted,
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
			ProductDeclaration:    localProductDeclaration,
		},
		{
			Name:                  "Failed to add ownerReference admin credentials secret",
			FakeClient:            moqclient.NewSigsClientMoqWithScheme(scheme, infrastructureAws),
			FakeOauthClient:       fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:            basicConfigMock(),
			Installation:          &installation,
			ExpectedStatus:        integreatlyv1alpha1.PhaseFailed,
			ExpectError:           true,
			ExpectedError:         fmt.Sprintf(` "%s" not found`, adminCredentialSecretName),
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
			ProductDeclaration:    localProductDeclaration,
		},
		{
			Name:            "URL for Keycloak not yet available",
			FakeClient:      utils.NewTestClient(scheme, oauthClientSecrets, githubOauthSecret, kc, croPostgres, croPostgresSecret, kcr, credentialRhsso, infrastructureAws),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig: func() *config.ConfigReadWriterMock {
				basicConfig := basicConfigMock()
				basicConfig.ReadRHSSOFunc = func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
						"REALM":     "openshift",
						"URL":       "rhsso.openshift-cluster.com",
						"HOST":      "",
						"VERSION":   string(integreatlyv1alpha1.VersionRHSSO),
					}), nil
				}
				return basicConfig
			}(),
			Installation:          &installation,
			ExpectedStatus:        integreatlyv1alpha1.PhaseAwaitingComponents,
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
			ProductDeclaration:    localProductDeclaration,
		},
		{
			Name:               "Failed to setup Openshift IDP",
			FakeClient:         utils.NewTestClient(scheme, kc, croPostgres, croPostgresSecret, kcr, credentialRhsso, infrastructureAws),
			FakeOauthClient:    fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:         basicConfigMock(),
			Installation:       &installation,
			ExpectedStatus:     integreatlyv1alpha1.PhaseFailed,
			ExpectError:        true,
			ExpectedError:      `secrets "oauth-client-secrets" not found`,
			Recorder:           setupRecorder(),
			ApiUrl:             "https://serverurl",
			ProductDeclaration: localProductDeclaration,
		},
		{
			Name:               "Failed to setup Github IDP",
			FakeClient:         utils.NewTestClient(scheme, oauthClientSecrets, kc, croPostgres, croPostgresSecret, kcr, credentialRhsso, infrastructureAws),
			FakeOauthClient:    fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:         basicConfigMock(),
			Installation:       &installation,
			ExpectedStatus:     integreatlyv1alpha1.PhaseFailed,
			ExpectError:        true,
			ExpectedError:      `secrets "github-oauth-secret" not found`,
			Recorder:           setupRecorder(),
			ApiUrl:             "https://serverurl",
			ProductDeclaration: localProductDeclaration,
		},
		{
			Name:               "Failed to authenticate client in keycloak api",
			FakeClient:         utils.NewTestClient(scheme, oauthClientSecrets, githubOauthSecret, kc, croPostgres, croPostgresSecret, kcr, credentialRhsso, infrastructureAws),
			FakeOauthClient:    fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:         basicConfigMock(),
			Installation:       &installation,
			ExpectedStatus:     integreatlyv1alpha1.PhaseFailed,
			ExpectError:        true,
			ExpectedError:      "failed to authenticate client in keycloak api",
			Recorder:           setupRecorder(),
			ApiUrl:             "https://serverurl",
			ProductDeclaration: localProductDeclaration,
			KeycloakClientFactory: func() keycloakCommon.KeycloakClientFactory {
				return &keycloakCommon.KeycloakClientFactoryMock{
					AuthenticatedClientFunc: func(kc keycloak.Keycloak) (keycloakInterface keycloakCommon.KeycloakInterface, err error) {
						return nil, errors.New("failed to authenticate client in keycloak api")
					},
				}
			}(),
		},
		{
			Name:               "Failed to create and add keycloak authentication flow",
			FakeClient:         utils.NewTestClient(scheme, oauthClientSecrets, githubOauthSecret, kc, croPostgres, croPostgresSecret, kcr, credentialRhsso, infrastructureAws),
			FakeOauthClient:    fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:         basicConfigMock(),
			Installation:       &installation,
			ExpectedStatus:     integreatlyv1alpha1.PhaseFailed,
			ExpectError:        true,
			ExpectedError:      "failed to find keycloak authentication flow",
			Recorder:           setupRecorder(),
			ApiUrl:             "https://serverurl",
			ProductDeclaration: localProductDeclaration,
			KeycloakClientFactory: func() keycloakCommon.KeycloakClientFactory {
				return &keycloakCommon.KeycloakClientFactoryMock{
					AuthenticatedClientFunc: func(kc keycloak.Keycloak) (keycloakInterface keycloakCommon.KeycloakInterface, err error) {
						return &keycloakCommon.KeycloakInterfaceMock{
							FindAuthenticationFlowByAliasFunc: func(flowAlias string, realmName string) (*keycloakCommon.AuthenticationFlow, error) {
								return nil, errors.New("failed to find keycloak authentication flow")
							},
						}, nil
					},
				}
			}(),
		},
		{
			Name:               "Failed to sync openshift idp client secret",
			FakeClient:         utils.NewTestClient(scheme, oauthClientSecrets, githubOauthSecret, kc, croPostgres, croPostgresSecret, kcr, credentialRhsso, infrastructureAws),
			FakeOauthClient:    fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:         basicConfigMock(),
			Installation:       &installation,
			ExpectedStatus:     integreatlyv1alpha1.PhaseFailed,
			ExpectError:        true,
			ExpectedError:      "Unable to update Identity Provider",
			Recorder:           setupRecorder(),
			ApiUrl:             "https://serverurl",
			ProductDeclaration: localProductDeclaration,
			KeycloakClientFactory: func() keycloakCommon.KeycloakClientFactory {
				keycloakInterfaceMock, _ := createKeycloakInterfaceMock()
				return &keycloakCommon.KeycloakClientFactoryMock{
					AuthenticatedClientFunc: func(kc keycloak.Keycloak) (keycloakInterface keycloakCommon.KeycloakInterface, err error) {
						return &keycloakCommon.KeycloakInterfaceMock{
							FindAuthenticationFlowByAliasFunc:      keycloakInterfaceMock.FindAuthenticationFlowByAlias,
							CreateAuthenticationFlowFunc:           keycloakInterfaceMock.CreateAuthenticationFlow,
							FindAuthenticationExecutionForFlowFunc: keycloakInterfaceMock.FindAuthenticationExecutionForFlow,
							AddExecutionToAuthenticatonFlowFunc:    keycloakInterfaceMock.AddExecutionToAuthenticatonFlow,
							GetIdentityProviderFunc: func(alias string, realmName string) (provider *keycloak.KeycloakIdentityProvider, err error) {
								return &keycloak.KeycloakIdentityProvider{
									Alias:                     "babla",
									FirstBrokerLoginFlowAlias: "authdelay",
									Config:                    map[string]string{},
								}, nil
							},
							UpdateIdentityProviderFunc: func(specIdentityProvider *keycloak.KeycloakIdentityProvider, realmName string) error {
								return errors.New("generic")
							},
						}, nil
					},
				}
			}(),
		},
		{
			Name: "Test reconcile custom resource returns failed on unsuccessful create",
			FakeClient: func() k8sclient.Client {
				client := moqclient.NewSigsClientMoqWithScheme(scheme, infrastructureAws)
				client.CreateFunc = func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.CreateOption) error {
					return errors.New("failed to create keycloak custom resource")
				}
				return client
			}(),
			FakeConfig:            basicConfigMock(),
			Installation:          &installation,
			ExpectError:           true,
			ExpectedError:         "failed to create/update keycloak custom resource: failed to create keycloak custom resource",
			ExpectedStatus:        integreatlyv1alpha1.PhaseFailed,
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
			ProductDeclaration:    localProductDeclaration,
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(
				tc.FakeConfig,
				tc.Installation,
				tc.FakeOauthClient,
				tc.FakeMPM,
				tc.Recorder,
				tc.ApiUrl,
				tc.KeycloakClientFactory,
				getLogger(),
				tc.ProductDeclaration,
			)
			if err != nil {
				if errorContains(err, "no product declaration found for RHSSO") {
					return
				}
				t.Fatal("unexpected err ", err)
			}
			phase, err := reconciler.reconcileComponents(context.TODO(), tc.Installation, tc.FakeClient)
			if tc.ExpectError && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectError && err != nil {
				t.Fatal("expected no error but got one ", err)
			}
			if !errorContains(err, tc.ExpectedError) {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
			}
			if tc.ExpectedStatus != phase {
				t.Fatal("expected phase ", tc.ExpectedStatus, " but got ", phase)
			}
		})
	}
}

func TestReconciler_fullReconcile(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	installation := &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "installation",
			Namespace:  defaultOperandNamespace,
			Finalizers: []string{"finalizer.rhsso.integreatly.org"},
			UID:        types.UID("xyz"),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "RHMI",
			APIVersion: integreatlyv1alpha1.GroupVersion.String(),
		},
		Status: integreatlyv1alpha1.RHMIStatus{
			Stages: map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus{
				"threeScale-stage": {
					Name: "threeScale-stage",
					Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
						integreatlyv1alpha1.Product3Scale: {
							Name:  integreatlyv1alpha1.Product3Scale,
							Phase: integreatlyv1alpha1.PhaseCreatingComponents,
						},
					},
				},
			},
		},
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultOperandNamespace,
			Labels: map[string]string{
				resources.OwnerLabelKey: string(installation.GetUID()),
			},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}

	operatorNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultOperandNamespace + "-operator",
			Labels: map[string]string{
				resources.OwnerLabelKey: string(installation.GetUID()),
			},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}

	kc := &keycloak.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: defaultOperandNamespace,
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: defaultOperandNamespace,
		},
	}

	githubOauthSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "github-oauth-secret",
			Namespace: defaultOperatorNamespace,
		},
		Data: map[string][]byte{
			"clientId": bytes.NewBufferString("dummy").Bytes(),
			"secret":   bytes.NewBufferString("dummy").Bytes(),
		},
	}

	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oauth-client-secrets",
			Namespace: defaultOperatorNamespace,
		},
		Data: map[string][]byte{
			"rhsso":  bytes.NewBufferString("test").Bytes(),
			"3scale": bytes.NewBufferString("test").Bytes(),
		},
	}

	edgeRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak-edge",
			Namespace: defaultOperandNamespace,
		},
		Spec: routev1.RouteSpec{
			Host: "sampleHost",
		},
	}

	normalRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak",
			Namespace: defaultOperandNamespace,
		},
		Spec: routev1.RouteSpec{
			Host: "sampleHost",
		},
	}

	//completed postgres that points at the secret croPostgresSecret
	croPostgres := &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("rhsso-postgres-%s", installation.Name),
			Namespace: defaultOperandNamespace,
		},
		Status: croTypes.ResourceTypeStatus{
			Phase: croTypes.PhaseComplete,
			SecretRef: &croTypes.SecretRef{
				Name:      "test",
				Namespace: defaultOperandNamespace,
			},
		},
	}

	//secret created by the cloud resource operator postgres reconciler
	croPostgresSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: defaultOperandNamespace,
		},
		Data: map[string][]byte{},
		Type: corev1.SecretTypeOpaque,
	}

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak",
			Namespace: defaultOperandNamespace,
			Labels: map[string]string{
				"app": "keycloak",
			},
		},
	}

	csv := &operatorsv1alpha1.ClusterServiceVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhsso-operator.7.6.1-opr-001",
			Namespace: defaultOperandNamespace + "-operator",
		},
		Spec: operatorsv1alpha1.ClusterServiceVersionSpec{
			InstallStrategy: operatorsv1alpha1.NamedInstallStrategy{
				StrategySpec: operatorsv1alpha1.StrategyDetailsDeployment{
					DeploymentSpecs: []operatorsv1alpha1.StrategyDeploymentSpec{
						{
							Name: "rhsso-operator",
							Spec: appsv1.DeploymentSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Env: []corev1.EnvVar{},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	prometheusRules := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak",
			Namespace: "rhsso",
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name:     "general.rules",
					Interval: "",
					Rules: []monitoringv1.Rule{
						{Alert: "Some Rule"},
					},
				},
			},
		},
	}

	infrastructureAws := &configv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Status: configv1.InfrastructureStatus{
			PlatformStatus: &configv1.PlatformStatus{
				Type: configv1.AWSPlatformType,
			},
		},
	}

	cases := []struct {
		Name                  string
		ExpectError           bool
		ExpectedStatus        integreatlyv1alpha1.StatusPhase
		ExpectedError         string
		FakeConfig            *config.ConfigReadWriterMock
		FakeClient            k8sclient.Client
		FakeMPM               *marketplace.MarketplaceInterfaceMock
		Installation          *integreatlyv1alpha1.RHMI
		Product               *integreatlyv1alpha1.RHMIProductStatus
		Recorder              record.EventRecorder
		ApiUrl                string
		KeycloakClientFactory keycloakCommon.KeycloakClientFactory
		Uninstall             bool
	}{
		{
			Name:           "test successful reconcile",
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			FakeClient: func() k8sclient.Client {
				mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, getKcr(keycloak.KeycloakRealmStatus{Phase: keycloak.PhaseReconciling}), kc, secret, ns, operatorNS, githubOauthSecret, oauthClientSecrets, installation, edgeRoute, normalRoute, croPostgres, croPostgresSecret, getRHSSOCredentialSeed(), statefulSet, csv /*dashboard,*/, prometheusRules, infrastructureAws)
				mockClient.PatchFunc = func(ctx context.Context, obj k8sclient.Object, patch k8sclient.Patch, opts ...k8sclient.PatchOption) error {
					return nil
				}
				return mockClient
			}(),
			FakeConfig: basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {

					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlan{
							ObjectMeta: metav1.ObjectMeta{
								Name: "3scale-install-plan",
							},
							Status: operatorsv1alpha1.InstallPlanStatus{
								Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
							},
						}, &operatorsv1alpha1.Subscription{
							Status: operatorsv1alpha1.SubscriptionStatus{
								Install: &operatorsv1alpha1.InstallPlanReference{
									Name: "3scale-install-plan",
								},
							},
						}, nil
				},
			},
			Installation:          installation,
			Product:               &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
			Uninstall:             false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			server := configureTestServer(t)
			defer server.Close()

			oauthv1Client, err := oauthClient.NewForConfig(&rest.Config{Host: server.URL})
			if err != nil {
				t.Errorf("Failed to configure oauthclient")
			}
			testReconciler, err := NewReconciler(
				tc.FakeConfig,
				tc.Installation,
				oauthv1Client,
				tc.FakeMPM,
				tc.Recorder,
				tc.ApiUrl,
				tc.KeycloakClientFactory,
				getLogger(),
				localProductDeclaration,
			)
			if err != nil && err.Error() != tc.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
			}

			status, err := testReconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.FakeClient, &quota.ProductConfigMock{}, tc.Uninstall)

			if err != nil && !tc.ExpectError {
				t.Fatalf("expected no errors, but got one: %v", err)
			}

			if err == nil && tc.ExpectError {
				t.Fatal("expected error but got none")
			}

			if status != tc.ExpectedStatus {
				t.Fatalf("Expected status: '%v', got: '%v'", tc.ExpectedStatus, status)
			}
		})
	}
}

func getKcr(status keycloak.KeycloakRealmStatus) *keycloak.KeycloakRealm {
	return &keycloak.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakRealmName,
			Namespace: defaultOperandNamespace,
		},
		Spec: keycloak.KeycloakRealmSpec{
			Realm: &keycloak.KeycloakAPIRealm{
				ID:          keycloakRealmName,
				Realm:       keycloakRealmName,
				DisplayName: keycloakRealmName,
				Enabled:     true,
				EventsListeners: []string{
					"metrics-listener",
				},
			},
		},
		Status: status,
	}
}

func getMoqKeycloakClientFactory() keycloakCommon.KeycloakClientFactory {

	keycloakInterfaceMock, _ := createKeycloakInterfaceMock()

	return &keycloakCommon.KeycloakClientFactoryMock{
		AuthenticatedClientFunc: func(kc keycloak.Keycloak) (keycloakInterface keycloakCommon.KeycloakInterface, err error) {
			return &keycloakCommon.KeycloakInterfaceMock{
				CreateIdentityProviderFunc: func(identityProvider *keycloak.KeycloakIdentityProvider, realmName string) (string, error) {
					return "", nil
				}, GetIdentityProviderFunc: func(alias string, realmName string) (provider *keycloak.KeycloakIdentityProvider, err error) {
					return &keycloak.KeycloakIdentityProvider{
						Alias:                     "babla",
						FirstBrokerLoginFlowAlias: "authdelay",
						Config:                    map[string]string{},
					}, nil
				}, CreateAuthenticatorConfigFunc: func(authenticatorConfig *keycloak.AuthenticatorConfig, realmName string, executionID string) (string, error) {
					return "", nil
				},
				UpdateIdentityProviderFunc:               keycloakInterfaceMock.UpdateIdentityProvider,
				AddExecutionToAuthenticatonFlowFunc:      keycloakInterfaceMock.AddExecutionToAuthenticatonFlow,
				CreateAuthenticationFlowFunc:             keycloakInterfaceMock.CreateAuthenticationFlow,
				FindAuthenticationFlowByAliasFunc:        keycloakInterfaceMock.FindAuthenticationFlowByAlias,
				ListAuthenticationFlowsFunc:              keycloakInterfaceMock.ListAuthenticationFlows,
				FindAuthenticationExecutionForFlowFunc:   keycloakInterfaceMock.FindAuthenticationExecutionForFlow,
				ListAuthenticationExecutionsForFlowFunc:  keycloakInterfaceMock.ListAuthenticationExecutionsForFlow,
				UpdateAuthenticationExecutionForFlowFunc: keycloakInterfaceMock.UpdateAuthenticationExecutionForFlow,
			}, nil
		}}
}

// Mock context of the Keycloak interface. Allows to check that the operations
// performed by the client were correct
type mockClientContext struct {
	AuthenticationFlow            map[string][]*keycloakCommon.AuthenticationFlow
	AuthenticationFlowsExecutions map[string][]*keycloak.AuthenticationExecutionInfo
}

// Create a mock of the `KeycloakClientFactory` that creates a `KeycloakInterface` mock that
// manages groups and their client roles ignoring realm or client parameters. This mock is
// implemented to test the `reconcileDevelopersGroup` phase
func createKeycloakInterfaceMock() (keycloakCommon.KeycloakInterface, *mockClientContext) {
	context := mockClientContext{
		AuthenticationFlow: map[string][]*keycloakCommon.AuthenticationFlow{
			keycloakName: []*keycloakCommon.AuthenticationFlow{
				&keycloakCommon.AuthenticationFlow{
					ID:    "bkabka",
					Alias: "aaa",
				},
			},
		},
		AuthenticationFlowsExecutions: map[string][]*keycloak.AuthenticationExecutionInfo{
			authFlowAlias: []*keycloak.AuthenticationExecutionInfo{
				&keycloak.AuthenticationExecutionInfo{
					Requirement: "REQUIRED",
					Alias:       authFlowAlias,
				},
				// dummy ones
				&keycloak.AuthenticationExecutionInfo{
					Requirement: "REQUIRED",
					Alias:       "dummy execution",
				},
			},
		},
	}

	createAuthenticationFlowFunc := func(authFlow keycloakCommon.AuthenticationFlow, realmName string) (string, error) {
		if len(context.AuthenticationFlow) <= 0 {
			context.AuthenticationFlow = make(map[string][]*keycloakCommon.AuthenticationFlow)
		}

		for _, af := range context.AuthenticationFlow[realmName] {
			if af.ID == authFlow.ID {
				return "", errors.New("Authentication flow already exist")
			}
		}

		context.AuthenticationFlow[realmName] = append(context.AuthenticationFlow[realmName], &authFlow)
		return "dummy-id", nil
	}

	listAuthenticationFlowsFunc := func(realmName string) ([]*keycloakCommon.AuthenticationFlow, error) {
		if len(context.AuthenticationFlow) <= 0 {
			return []*keycloakCommon.AuthenticationFlow{}, nil
		}
		return context.AuthenticationFlow[realmName], nil
	}

	updateIdentityProviderFunc := func(specIdentityProvider *v1alpha1.KeycloakIdentityProvider, realmName string) error {
		return nil
	}

	findAuthenticationFlowByAliasFunc := func(flowAlias, realmName string) (*keycloakCommon.AuthenticationFlow, error) {
		authenticationFlows, err := listAuthenticationFlowsFunc(realmName)

		if err != nil {
			return nil, err
		}

		for _, authFlow := range authenticationFlows {
			if authFlow.Alias == flowAlias {
				return authFlow, nil
			}
		}

		return nil, nil
	}

	listAuthenticationExecutionsForFlowFunc := func(flowAlias, realmName string) ([]*keycloak.AuthenticationExecutionInfo, error) {
		if len(context.AuthenticationFlowsExecutions) <= 0 {
			return []*keycloak.AuthenticationExecutionInfo{}, nil
		}

		executions, ok := context.AuthenticationFlowsExecutions[flowAlias]

		if !ok {
			return nil, errors.New("Authentication flow not found")
		}

		return executions, nil
	}

	findAuthenticationExecutionForFlowFunc := func(flowAlias, realmName string, predicate func(*keycloak.AuthenticationExecutionInfo) bool) (*keycloak.AuthenticationExecutionInfo, error) {
		executions, err := listAuthenticationExecutionsForFlowFunc(flowAlias, realmName)

		if err != nil {
			return nil, err
		}

		for _, execution := range executions {
			if predicate(execution) {
				return execution, nil
			}
		}

		return nil, nil
	}

	updateAuthenticationExecutionForFlowFunc := func(flowAlias, realmName string, execution *keycloak.AuthenticationExecutionInfo) error {
		executions, ok := context.AuthenticationFlowsExecutions[flowAlias]

		if !ok {
			return fmt.Errorf("Authentication flow %s not found", flowAlias)
		}

		for i, currentExecution := range executions {
			if currentExecution.Alias != execution.Alias {
				continue
			}

			context.AuthenticationFlowsExecutions[flowAlias][i] = execution
			break
		}

		return nil
	}

	addExecutionToAuthenticatonFlowFunc := func(flowAlias, realmName string, providerID string, requirement keycloakCommon.Requirement) error {
		execution := keycloak.AuthenticationExecutionInfo{
			Alias:      flowAlias,
			ProviderID: providerID,
		}

		context.AuthenticationFlowsExecutions[flowAlias] = append(context.AuthenticationFlowsExecutions[flowAlias], &execution)

		if requirement != "" {
			execution, err := findAuthenticationExecutionForFlowFunc(flowAlias, realmName, func(execution *v1alpha1.AuthenticationExecutionInfo) bool {
				return execution.ProviderID == providerID
			})

			if err != nil {
				return fmt.Errorf("error finding Authentication Execution %s", providerID)
			}
			execution.Requirement = string(requirement)

			err = updateAuthenticationExecutionForFlowFunc(flowAlias, realmName, execution)

			if err != nil {
				return fmt.Errorf("error updating Authentication Execution %s", providerID)
			}
		}
		return nil
	}

	return &keycloakCommon.KeycloakInterfaceMock{
		CreateAuthenticationFlowFunc:             createAuthenticationFlowFunc,
		FindAuthenticationFlowByAliasFunc:        findAuthenticationFlowByAliasFunc,
		ListAuthenticationExecutionsForFlowFunc:  listAuthenticationExecutionsForFlowFunc,
		AddExecutionToAuthenticatonFlowFunc:      addExecutionToAuthenticatonFlowFunc,
		FindAuthenticationExecutionForFlowFunc:   findAuthenticationExecutionForFlowFunc,
		UpdateAuthenticationExecutionForFlowFunc: updateAuthenticationExecutionForFlowFunc,
		ListAuthenticationFlowsFunc:              listAuthenticationFlowsFunc,
		UpdateIdentityProviderFunc:               updateIdentityProviderFunc,
	}, &context
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.ProductRHSSO})
}

func errorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}

func configureTestServer(t *testing.T /*, apiList *metav1.APIResourceList*/) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var list interface{}
		switch req.URL.Path {
		case "/api/v1":
			list = &metav1.APIResourceList{
				GroupVersion: "v1",
				APIResources: []metav1.APIResource{},
			}
		case "/api":
			list = &metav1.APIVersions{
				Versions: []string{
					"v1",
				},
			}
		default:
			t.Logf("unexpected request: %s", req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		output, err := json.Marshal(list)
		if err != nil {
			t.Errorf("unexpected encoding error: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(output)
		if err != nil {
			t.Fatal(err)
		}
	}))
	return server
}
