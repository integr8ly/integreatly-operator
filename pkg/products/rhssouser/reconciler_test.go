package rhssouser

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/integr8ly/integreatly-operator/utils"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"

	croTypes "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"k8s.io/apimachinery/pkg/types"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	controllerruntime "sigs.k8s.io/controller-runtime"

	crotypes "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	keycloak "github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"
	keycloakCommon "github.com/integr8ly/keycloak-client/pkg/common"
	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	fakeoauthClient "github.com/openshift/client-go/oauth/clientset/versioned/fake"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	appsv1 "k8s.io/api/apps/v1"

	crov1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
		ReadRHSSOUserFunc: func() (*config.RHSSOUser, error) {
			return config.NewRHSSOUser(config.ProductConfig{
				"NAMESPACE": "user-sso",
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
	}
}

func getRHSSOCredentialSeed() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      adminCredentialSecretName,
			Namespace: defaultNamespace,
		},
		Data: map[string][]byte{},
		Type: corev1.SecretTypeOpaque,
	}
}

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
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
		ApiUrl                string
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
				ReadRHSSOUserFunc: func() (ready *config.RHSSOUser, e error) {
					return nil, errors.New("could not read rhsso config")
				},
			},
			Recorder:              setupRecorder(),
			Product:               &integreatlyv1alpha1.RHMIProductStatus{},
			ApiUrl:                "https://serverurl",
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
				tc.ApiUrl,
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
				t.Fatalf("expected error but got one: %v", err)
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
			Namespace: defaultNamespace,
		},
	}

	group := &usersv1.Group{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dedicated-admins",
		},
		Users: nil,
	}

	kcr := getKcr(keycloak.KeycloakRealmStatus{
		Phase: keycloak.PhaseReconciling,
	}, masterRealmName, "user-sso")

	githubOauthSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "github-oauth-secret",
			Namespace: defaultOperatorNamespace,
		},
	}

	credentialRhsso := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      adminCredentialSecretName,
			Namespace: "user-sso",
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
			"rhssouser": bytes.NewBufferString("test").Bytes(),
		},
	}

	//completed postgres that points at the secret croPostgresSecret
	croPostgres := &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhssouser-postgres-",
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

	//secret created by the cloud resource operator postgres reconciler
	croPostgresSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: defaultOperatorNamespace,
		},
		Data: map[string][]byte{},
		Type: corev1.SecretTypeOpaque,
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
		ProductConfig         *quota.ProductConfigMock
	}{
		{
			Name:            "Test reconcile custom resource returns completed when successful created",
			FakeClient:      utils.NewTestClient(scheme, oauthClientSecrets, githubOauthSecret, kcr, kc, group, croPostgres, croPostgresSecret, credentialRhsso, infrastructureAws),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:      basicConfigMock(),
			Installation: &integreatlyv1alpha1.RHMI{
				TypeMeta: metav1.TypeMeta{
					Kind:       "RHMI",
					APIVersion: integreatlyv1alpha1.GroupVersion.String(),
				},
				ObjectMeta: controllerruntime.ObjectMeta{
					Namespace: defaultOperatorNamespace,
				},
			},
			ExpectedStatus:        integreatlyv1alpha1.PhaseCompleted,
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
			ProductConfig: &quota.ProductConfigMock{
				ConfigureFunc: func(obj metav1.Object) error {
					return nil
				},
			},
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
			FakeConfig: basicConfigMock(),
			Installation: &integreatlyv1alpha1.RHMI{
				TypeMeta: metav1.TypeMeta{
					Kind:       "RHMI",
					APIVersion: integreatlyv1alpha1.GroupVersion.String(),
				},
				ObjectMeta: controllerruntime.ObjectMeta{
					Namespace: defaultOperatorNamespace,
				},
			},
			ExpectError:           true,
			ExpectedError:         "failed to create/update keycloak custom resource: failed to create keycloak custom resource",
			ExpectedStatus:        integreatlyv1alpha1.PhaseFailed,
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
			ProductConfig: &quota.ProductConfigMock{
				ConfigureFunc: func(obj metav1.Object) error {
					return nil
				},
			},
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
				localProductDeclaration,
			)
			if err != nil {
				t.Fatal("unexpected err ", err)
			}
			phase, err := reconciler.reconcileComponents(context.TODO(), tc.Installation, tc.FakeClient, tc.ProductConfig)
			if tc.ExpectError && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectError && err != nil {
				t.Fatal("expected no error but got one ", err)
			}
			if err != nil && err.Error() != tc.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
			}
			if tc.ExpectedStatus != phase {
				t.Fatal("expected phase ", tc.ExpectedStatus, " but got ", phase)
			}

		})
	}
}

func TestReconciler_full_RHMI_Reconcile(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	installation := &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "installation",
			Namespace:  defaultNamespace,
			Finalizers: []string{"finalizer.user-sso.integreatly.org"},
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
			Name: defaultNamespace,
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
			Name: defaultNamespace + "-operator",
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
			Namespace: defaultNamespace,
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: defaultNamespace,
		},
	}

	githubOauthSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "github-oauth-secret",
			Namespace: defaultOperatorNamespace,
		},
	}

	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oauth-client-secrets",
			Namespace: defaultOperatorNamespace,
		},
		Data: map[string][]byte{
			"rhssouser": bytes.NewBufferString("test").Bytes(),
		},
	}

	edgeRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak-edge",
			Namespace: "user-sso",
		},
		Spec: routev1.RouteSpec{
			Host: "sampleHost",
		},
	}

	normalRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak",
			Namespace: "user-sso",
		},
		Spec: routev1.RouteSpec{
			Host: "sampleHost",
		},
	}

	group := &usersv1.Group{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dedicated-admins",
		},
		Users: nil,
	}

	//completed postgres that points at the secret croPostgresSecret
	croPostgres := &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("rhssouser-postgres-%s", installation.Name),
			Namespace: defaultNamespace,
		},
		Status: croTypes.ResourceTypeStatus{
			Phase: croTypes.PhaseComplete,
			SecretRef: &croTypes.SecretRef{
				Name:      "test",
				Namespace: defaultNamespace,
			},
		},
	}

	//secret created by the cloud resource operator postgres reconciler
	croPostgresSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: defaultNamespace,
		},
		Data: map[string][]byte{},
		Type: corev1.SecretTypeOpaque,
	}

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak",
			Namespace: defaultNamespace,
			Labels: map[string]string{
				"app": "keycloak",
			},
		},
	}

	csv := &operatorsv1alpha1.ClusterServiceVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhsso-operator.7.6.1-opr-001",
			Namespace: defaultNamespace + "-operator",
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

	rhssoPostgres := &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s%s", constants.RHSSOPostgresPrefix, "installation"),
			Namespace: defaultOperatorNamespace,
		},
		Status: crotypes.ResourceTypeStatus{Phase: crotypes.PhaseComplete},
	}

	rhssoPostgresInProgress := &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s%s", constants.RHSSOPostgresPrefix, "installation"),
			Namespace: defaultOperatorNamespace,
		},
		Status: crotypes.ResourceTypeStatus{Phase: crotypes.PhaseInProgress},
	}

	prometheusRules := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak",
			Namespace: "user-sso",
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{Name: "general.rules"},
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
		ProductConfig         *quota.ProductConfigMock
		Uninstall             bool
	}{
		{
			Name:           "test successful reconcile",
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			FakeClient: func() k8sclient.Client {
				mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, getKcr(keycloak.KeycloakRealmStatus{Phase: keycloak.PhaseReconciling}, masterRealmName, "user-sso"), kc, secret, ns, operatorNS, githubOauthSecret, oauthClientSecrets, installation, edgeRoute, group, croPostgresSecret, croPostgres, getRHSSOCredentialSeed(), statefulSet, csv, rhssoPostgres, normalRoute, prometheusRules, infrastructureAws)
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
								Name: "codeready-install-plan",
							},
							Status: operatorsv1alpha1.InstallPlanStatus{
								Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
							},
						}, &operatorsv1alpha1.Subscription{
							Status: operatorsv1alpha1.SubscriptionStatus{
								Install: &operatorsv1alpha1.InstallPlanReference{
									Name: "codeready-install-plan",
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
			ProductConfig: &quota.ProductConfigMock{
				ConfigureFunc: func(obj metav1.Object) error {
					return nil
				},
			},
			Uninstall: false,
		},
		{
			Name:           "test waiting for RHSSO postgres",
			ExpectedStatus: integreatlyv1alpha1.PhaseInProgress,
			FakeClient: func() k8sclient.Client {
				mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, getKcr(keycloak.KeycloakRealmStatus{Phase: keycloak.PhaseReconciling}, masterRealmName, "user-sso"), kc, secret, ns, operatorNS, githubOauthSecret, oauthClientSecrets, installation, edgeRoute, group, croPostgresSecret, croPostgres, getRHSSOCredentialSeed(), statefulSet, csv, rhssoPostgresInProgress, infrastructureAws)
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
								Name: "codeready-install-plan",
							},
							Status: operatorsv1alpha1.InstallPlanStatus{
								Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
							},
						}, &operatorsv1alpha1.Subscription{
							Status: operatorsv1alpha1.SubscriptionStatus{
								Install: &operatorsv1alpha1.InstallPlanReference{
									Name: "codeready-install-plan",
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
			ProductConfig: &quota.ProductConfigMock{
				ConfigureFunc: func(obj metav1.Object) error {
					return nil
				},
			},
			Uninstall: false,
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

			status, err := testReconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.FakeClient, tc.ProductConfig, tc.Uninstall)

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

func TestReconciler_full_RHOAM_Reconcile(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	installation := &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "installation",
			Namespace:  defaultNamespace,
			Finalizers: []string{"finalizer.user-sso.integreatly.org"},
			UID:        types.UID("xyz"),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "RHMI",
			APIVersion: integreatlyv1alpha1.GroupVersion.String(),
		},
		Status: integreatlyv1alpha1.RHMIStatus{
			Stages: map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus{
				"rhsso-stage": {
					Name: "rhsso-stage",
					Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
						integreatlyv1alpha1.Product3Scale: {
							Name:  integreatlyv1alpha1.ProductRHSSO,
							Phase: integreatlyv1alpha1.PhaseCreatingComponents,
						},
					},
				},
			},
		},
		Spec: integreatlyv1alpha1.RHMISpec{
			Type: string(integreatlyv1alpha1.InstallationTypeManagedApi),
		},
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultNamespace,
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
			Name: defaultNamespace + "-operator",
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
			Namespace: defaultNamespace,
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: defaultNamespace,
		},
	}

	githubOauthSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "github-oauth-secret",
			Namespace: defaultOperatorNamespace,
		},
	}

	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oauth-client-secrets",
			Namespace: defaultOperatorNamespace,
		},
		Data: map[string][]byte{
			"rhssouser": bytes.NewBufferString("test").Bytes(),
		},
	}

	edgeRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak",
			Namespace: "user-sso",
		},
		Spec: routev1.RouteSpec{
			Host: "sampleHost",
		},
	}

	group := &usersv1.Group{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dedicated-admins",
		},
		Users: nil,
	}

	//completed postgres that points at the secret croPostgresSecret
	croPostgres := &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("rhssouser-postgres-%s", installation.Name),
			Namespace: defaultNamespace,
		},
		Status: croTypes.ResourceTypeStatus{
			Phase: croTypes.PhaseComplete,
			SecretRef: &croTypes.SecretRef{
				Name:      "test",
				Namespace: defaultNamespace,
			},
		},
	}

	//secret created by the cloud resource operator postgres reconciler
	croPostgresSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: defaultNamespace,
		},
		Data: map[string][]byte{},
		Type: corev1.SecretTypeOpaque,
	}

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak",
			Namespace: defaultNamespace,
			Labels: map[string]string{
				"app": "keycloak",
			},
		},
	}

	// prometheus rule created by the SSO operator that is exported to the observability namespace
	ssoAlert := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak",
			Namespace: defaultNamespace,
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name: "rhsso.rules",
				},
			},
		},
	}

	ssoAlertNoGroups := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak",
			Namespace: defaultNamespace,
		},
	}

	csv := &operatorsv1alpha1.ClusterServiceVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhsso-operator.7.6.1-opr-001",
			Namespace: defaultNamespace + "-operator",
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

	rhssoPostgres := &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s%s", constants.RHSSOPostgresPrefix, "installation"),
			Namespace: defaultOperatorNamespace,
		},
		Status: crotypes.ResourceTypeStatus{Phase: crotypes.PhaseComplete},
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
		ProductConfig         *quota.ProductConfigMock
		Uninstall             bool
	}{
		{
			Name:           "RHOAM - test successful reconcile",
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			FakeClient: func() k8sclient.Client {
				mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, getKcr(keycloak.KeycloakRealmStatus{Phase: keycloak.PhaseReconciling}, masterRealmName, "user-sso"), kc, secret, ns, operatorNS, githubOauthSecret, oauthClientSecrets, installation, edgeRoute, group, croPostgresSecret, croPostgres, getRHSSOCredentialSeed(), statefulSet, ssoAlert, csv, rhssoPostgres, infrastructureAws)
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
								Name: "codeready-install-plan",
							},
							Status: operatorsv1alpha1.InstallPlanStatus{
								Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
							},
						}, &operatorsv1alpha1.Subscription{
							Status: operatorsv1alpha1.SubscriptionStatus{
								Install: &operatorsv1alpha1.InstallPlanReference{
									Name: "codeready-install-plan",
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
			ProductConfig: &quota.ProductConfigMock{
				ConfigureFunc: func(obj metav1.Object) error {
					return nil
				},
			},
			Uninstall: false,
		},
		{
			Name:           "RHOAM - test in progress if no rhsso prom rules are present",
			ExpectedStatus: integreatlyv1alpha1.PhaseAwaitingComponents,
			FakeClient: func() k8sclient.Client {
				mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, getKcr(keycloak.KeycloakRealmStatus{Phase: keycloak.PhaseReconciling}, masterRealmName, "user-sso"), kc, secret, ns, operatorNS, githubOauthSecret, oauthClientSecrets, installation, edgeRoute, group, croPostgresSecret, croPostgres, getRHSSOCredentialSeed(), statefulSet, ssoAlertNoGroups, csv, infrastructureAws)
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
								Name: "codeready-install-plan",
							},
							Status: operatorsv1alpha1.InstallPlanStatus{
								Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
							},
						}, &operatorsv1alpha1.Subscription{
							Status: operatorsv1alpha1.SubscriptionStatus{
								Install: &operatorsv1alpha1.InstallPlanReference{
									Name: "codeready-install-plan",
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
			ProductConfig: &quota.ProductConfigMock{
				ConfigureFunc: func(obj metav1.Object) error {
					return nil
				},
			},
			Uninstall: false,
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

			status, err := testReconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.FakeClient, tc.ProductConfig, tc.Uninstall)

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

func getKcr(status keycloak.KeycloakRealmStatus, name string, ns string) *keycloak.KeycloakRealm {
	return &keycloak.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: keycloak.KeycloakRealmSpec{
			Realm: &keycloak.KeycloakAPIRealm{
				ID:          name,
				Realm:       name,
				DisplayName: name,
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
	exInfo := []*keycloak.AuthenticationExecutionInfo{
		{
			ProviderID: "identity-provider-redirector",
			ID:         "123-123-123",
		},
	}

	keycloakInterfaceMock, context := createKeycloakInterfaceMock()

	// Add the browser flow execution mock to the context in order to test
	// the reconcileComponents phase
	context.AuthenticationFlowsExecutions["browser"] = exInfo

	return &keycloakCommon.KeycloakClientFactoryMock{AuthenticatedClientFunc: func(kc keycloak.Keycloak) (keycloakInterface keycloakCommon.KeycloakInterface, err error) {
		return &keycloakCommon.KeycloakInterfaceMock{CreateIdentityProviderFunc: func(identityProvider *keycloak.KeycloakIdentityProvider, realmName string) (string, error) {
			return "", nil
		}, GetIdentityProviderFunc: func(alias string, realmName string) (provider *keycloak.KeycloakIdentityProvider, err error) {
			return nil, nil
		}, CreateAuthenticatorConfigFunc: func(authenticatorConfig *keycloak.AuthenticatorConfig, realmName string, executionID string) (string, error) {
			return "", nil
		},
			ListRealmsFunc:                           keycloakInterfaceMock.ListRealms,
			FindGroupByNameFunc:                      keycloakInterfaceMock.FindGroupByName,
			CreateGroupFunc:                          keycloakInterfaceMock.CreateGroup,
			SetGroupChildFunc:                        keycloakInterfaceMock.SetGroupChild,
			MakeGroupDefaultFunc:                     keycloakInterfaceMock.MakeGroupDefault,
			ListUsersInGroupFunc:                     keycloakInterfaceMock.ListUsersInGroup,
			ListDefaultGroupsFunc:                    keycloakInterfaceMock.ListDefaultGroups,
			CreateGroupClientRoleFunc:                keycloakInterfaceMock.CreateGroupClientRole,
			ListGroupClientRolesFunc:                 keycloakInterfaceMock.ListGroupClientRoles,
			FindGroupClientRoleFunc:                  keycloakInterfaceMock.FindGroupClientRole,
			ListAvailableGroupClientRolesFunc:        keycloakInterfaceMock.ListAvailableGroupClientRoles,
			FindAvailableGroupClientRoleFunc:         keycloakInterfaceMock.FindAvailableGroupClientRole,
			ListGroupRealmRolesFunc:                  keycloakInterfaceMock.ListGroupRealmRoles,
			ListAvailableGroupRealmRolesFunc:         keycloakInterfaceMock.ListAvailableGroupRealmRoles,
			CreateGroupRealmRoleFunc:                 keycloakInterfaceMock.CreateGroupRealmRole,
			ListAuthenticationExecutionsForFlowFunc:  keycloakInterfaceMock.ListAuthenticationExecutionsForFlow,
			FindAuthenticationExecutionForFlowFunc:   keycloakInterfaceMock.FindAuthenticationExecutionForFlow,
			UpdateAuthenticationExecutionForFlowFunc: keycloakInterfaceMock.UpdateAuthenticationExecutionForFlow,
			ListClientsFunc:                          keycloakInterfaceMock.ListClients,
			ListOfActivesUsersPerRealmFunc:           keycloakInterfaceMock.ListOfActivesUsersPerRealm,
		}, nil
	}}
}

// Mock context of the Keycloak interface. Allows to check that the operations
// performed by the client were correct
type mockClientContext struct {
	Groups                        []*keycloakCommon.Group
	DefaultGroups                 []*keycloakCommon.Group
	ClientRoles                   map[string][]*keycloak.KeycloakUserRole
	RealmRoles                    map[string][]*keycloak.KeycloakUserRole
	AuthenticationFlowsExecutions map[string][]*keycloak.AuthenticationExecutionInfo
}

// Create a mock of the `KeycloakClientFactory` that creates a `KeycloakInterface` mock that
// manages groups and their client roles ignoring realm or client parameters. This mock is
// implemented to test the `reconcileDevelopersGroup` phase
func createKeycloakInterfaceMock() (keycloakCommon.KeycloakInterface, *mockClientContext) {
	context := mockClientContext{
		Groups:        []*keycloakCommon.Group{},
		DefaultGroups: []*keycloakCommon.Group{},
		ClientRoles:   map[string][]*keycloak.KeycloakUserRole{},
		RealmRoles:    map[string][]*keycloak.KeycloakUserRole{},
		AuthenticationFlowsExecutions: map[string][]*keycloak.AuthenticationExecutionInfo{
			firstBrokerLoginFlowAlias: []*keycloak.AuthenticationExecutionInfo{
				&keycloak.AuthenticationExecutionInfo{
					Requirement: "REQUIRED",
					Alias:       reviewProfileExecutionAlias,
				},
				// dummy ones
				&keycloak.AuthenticationExecutionInfo{
					Requirement: "REQUIRED",
					Alias:       "dummy execution",
				},
			},
		},
	}

	availableGroupClientRoles := []*keycloak.KeycloakUserRole{
		&keycloak.KeycloakUserRole{
			ID:   "create-client",
			Name: "create-client",
		},
		&keycloak.KeycloakUserRole{
			ID:   "manage-authorization",
			Name: "manage-authorization",
		},
		&keycloak.KeycloakUserRole{
			ID:   "manage-clients",
			Name: "manage-clients",
		},
		&keycloak.KeycloakUserRole{
			ID:   "manage-events",
			Name: "manage-events",
		},
		&keycloak.KeycloakUserRole{
			ID:   "manage-identity-providers",
			Name: "manage-identity-providers",
		},
		&keycloak.KeycloakUserRole{
			ID:   "manage-realm",
			Name: "manage-realm",
		},
		&keycloak.KeycloakUserRole{
			ID:   "manage-users",
			Name: "manage-users",
		},
		&keycloak.KeycloakUserRole{
			ID:   "query-clients",
			Name: "query-clients",
		},
		&keycloak.KeycloakUserRole{
			ID:   "query-groups",
			Name: "query-groups",
		},
		&keycloak.KeycloakUserRole{
			ID:   "query-realms",
			Name: "query-realms",
		},
		&keycloak.KeycloakUserRole{
			ID:   "query-users",
			Name: "query-users",
		},
		&keycloak.KeycloakUserRole{
			ID:   "view-authorization",
			Name: "view-authorization",
		},
		&keycloak.KeycloakUserRole{
			ID:   "view-clients",
			Name: "view-clients",
		},
		&keycloak.KeycloakUserRole{
			ID:   "view-events",
			Name: "view-events",
		},
		&keycloak.KeycloakUserRole{
			ID:   "view-identity-providers",
			Name: "view-identity-providers",
		},
		&keycloak.KeycloakUserRole{
			ID:   "view-realm",
			Name: "view-realm",
		},
		&keycloak.KeycloakUserRole{
			ID:   "view-users",
			Name: "view-users",
		},
	}

	availableGroupRealmRoles := []*keycloak.KeycloakUserRole{
		&keycloak.KeycloakUserRole{
			ID:   "mock-role-3",
			Name: "mock-role-3",
		},
		&keycloak.KeycloakUserRole{
			ID:   "create-realm",
			Name: "create-realm",
		},
		&keycloak.KeycloakUserRole{
			ID:   "mock-role-4",
			Name: "mock-role-4",
		},
	}

	listRealmsFunc := func() ([]*keycloak.KeycloakAPIRealm, error) {
		return []*keycloak.KeycloakAPIRealm{
			&keycloak.KeycloakAPIRealm{
				Realm: "master",
			},
			&keycloak.KeycloakAPIRealm{
				Realm: "test",
			},
		}, nil
	}

	findGroupByNameFunc := func(groupName string, realmName string) (*keycloakCommon.Group, error) {
		for _, group := range context.Groups {
			if group.Name == groupName {
				return group, nil
			}
		}

		return nil, nil
	}

	createGroupFunc := func(groupName string, realmName string) (string, error) {
		nextID := fmt.Sprintf("group-%d", len(context.Groups))

		newGroup := &keycloakCommon.Group{
			ID:   string(nextID),
			Name: groupName,
		}

		context.Groups = append(context.Groups, newGroup)

		context.ClientRoles[nextID] = []*keycloak.KeycloakUserRole{}
		context.RealmRoles[nextID] = []*keycloak.KeycloakUserRole{}

		return nextID, nil
	}

	setGroupChildFunc := func(groupID, realmName string, childGroup *keycloakCommon.Group) error {
		var childGroupToAppend *keycloakCommon.Group
		for _, group := range context.Groups {
			if group.ID == childGroup.ID {
				childGroupToAppend = group
			}
		}

		if childGroupToAppend == nil {
			childGroupToAppend = childGroup
		}

		found := false
		for _, group := range context.Groups {
			if group.ID == groupID {
				group.SubGroups = append(group.SubGroups, childGroupToAppend)
				found = true
			}
		}

		if !found {
			return fmt.Errorf("Group %s not found", groupID)
		}

		return nil
	}

	listUsersInGroupFunc := func(realmName, groupID string) ([]*keycloak.KeycloakAPIUser, error) {
		return []*keycloak.KeycloakAPIUser{}, nil
	}

	makeGroupDefaultFunc := func(groupID string, realmName string) error {
		var group *keycloakCommon.Group

		for _, existingGroup := range context.Groups {
			if existingGroup.ID == groupID {
				group = existingGroup
				break
			}
		}

		if group == nil {
			return fmt.Errorf("Referenced group not found")
		}

		context.DefaultGroups = append(context.DefaultGroups, group)
		return nil
	}

	listDefaultGroupsFunc := func(realmName string) ([]*keycloakCommon.Group, error) {
		return context.DefaultGroups, nil
	}

	createGroupClientRoleFunc := func(role *keycloak.KeycloakUserRole, realmName, clientID, groupID string) (string, error) {
		groupClientRoles, ok := context.ClientRoles[groupID]

		if !ok {
			return "", fmt.Errorf("Referenced group not found")
		}

		context.ClientRoles[groupID] = append(groupClientRoles, role)
		return "dummy-group-client-role-id", nil
	}

	listGroupClientRolesFunc := func(realmName, clientID, groupID string) ([]*keycloak.KeycloakUserRole, error) {
		groupRoles, ok := context.ClientRoles[groupID]

		if !ok {
			return nil, fmt.Errorf("Referenced group not found")
		}

		return groupRoles, nil
	}

	listAvailableGroupClientRolesFunc := func(realmName, clientID, groupID string) ([]*keycloak.KeycloakUserRole, error) {
		_, ok := context.ClientRoles[groupID]

		if !ok {
			return nil, fmt.Errorf("Referenced group not found")
		}

		return availableGroupClientRoles, nil
	}

	findGroupClientRoleFunc := func(realmName, clientID, groupID string, predicate func(*keycloak.KeycloakUserRole) bool) (*keycloak.KeycloakUserRole, error) {
		all, err := listGroupClientRolesFunc(realmName, clientID, groupID)

		if err != nil {
			return nil, err
		}

		for _, role := range all {
			if predicate(role) {
				return role, nil
			}
		}

		return nil, nil
	}

	findAvailableGroupClientRoleFunc := func(realmName, clientID, groupID string, predicate func(*keycloak.KeycloakUserRole) bool) (*keycloak.KeycloakUserRole, error) {
		all, err := listAvailableGroupClientRolesFunc(realmName, clientID, groupID)

		if err != nil {
			return nil, err
		}

		for _, role := range all {
			if predicate(role) {
				return role, nil
			}
		}

		return nil, nil
	}

	listGroupRealmRolesFunc := func(realmName, groupID string) ([]*keycloak.KeycloakUserRole, error) {
		groupRoles, ok := context.RealmRoles[groupID]

		if !ok {
			return nil, fmt.Errorf("Referenced group not found")
		}

		return groupRoles, nil
	}

	listAvailableGroupRealmRolesFunc := func(realmName, groupID string) ([]*keycloak.KeycloakUserRole, error) {
		_, ok := context.RealmRoles[groupID]

		if !ok {
			return nil, fmt.Errorf("Referenced group not found")
		}

		return availableGroupRealmRoles, nil
	}

	createGroupRealmRoleFunc := func(role *keycloak.KeycloakUserRole, realmName, groupID string) (string, error) {
		groupRealmRoles, ok := context.RealmRoles[groupID]

		if !ok {
			return "", fmt.Errorf("Referenced group not found")
		}

		context.RealmRoles[groupID] = append(groupRealmRoles, role)
		return "dummy-group-realm-role-id", nil
	}

	listClientsFunc := func(realmName string) ([]*keycloak.KeycloakAPIClient, error) {
		return []*keycloak.KeycloakAPIClient{
			&keycloak.KeycloakAPIClient{
				ClientID: "test-realm",
				ID:       "test-realm",
				Name:     "test-realm",
			},
			&keycloak.KeycloakAPIClient{
				ClientID: "master-realm",
				ID:       "master-realm",
				Name:     "master-realm",
			},
		}, nil
	}

	listAuthenticationExecutionsForFlowFunc := func(flowAlias, realmName string) ([]*keycloak.AuthenticationExecutionInfo, error) {
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

	listOfActivesUsersPerRealmFunc := func(realmName, dateFrom string, max int) ([]keycloakCommon.Users, error) {
		users := []keycloakCommon.Users{
			{
				UserID: "1",
			},
			{
				UserID: "2",
			},
			{
				UserID: "3",
			},
		}

		return users, nil
	}

	return &keycloakCommon.KeycloakInterfaceMock{
		ListRealmsFunc:                           listRealmsFunc,
		FindGroupByNameFunc:                      findGroupByNameFunc,
		CreateGroupFunc:                          createGroupFunc,
		SetGroupChildFunc:                        setGroupChildFunc,
		ListUsersInGroupFunc:                     listUsersInGroupFunc,
		MakeGroupDefaultFunc:                     makeGroupDefaultFunc,
		ListDefaultGroupsFunc:                    listDefaultGroupsFunc,
		CreateGroupClientRoleFunc:                createGroupClientRoleFunc,
		ListGroupClientRolesFunc:                 listGroupClientRolesFunc,
		ListAvailableGroupClientRolesFunc:        listAvailableGroupClientRolesFunc,
		FindGroupClientRoleFunc:                  findGroupClientRoleFunc,
		FindAvailableGroupClientRoleFunc:         findAvailableGroupClientRoleFunc,
		ListGroupRealmRolesFunc:                  listGroupRealmRolesFunc,
		ListAvailableGroupRealmRolesFunc:         listAvailableGroupRealmRolesFunc,
		CreateGroupRealmRoleFunc:                 createGroupRealmRoleFunc,
		ListAuthenticationExecutionsForFlowFunc:  listAuthenticationExecutionsForFlowFunc,
		FindAuthenticationExecutionForFlowFunc:   findAuthenticationExecutionForFlowFunc,
		UpdateAuthenticationExecutionForFlowFunc: updateAuthenticationExecutionForFlowFunc,
		ListClientsFunc:                          listClientsFunc,
		ListOfActivesUsersPerRealmFunc:           listOfActivesUsersPerRealmFunc,
	}, &context
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.ProductRHSSOUser})
}

func configureTestServer(t *testing.T) *httptest.Server {
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
