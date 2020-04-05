package rhssouser

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/sirupsen/logrus"
	controllerruntime "sigs.k8s.io/controller-runtime"

	keycloakCommon "github.com/integr8ly/keycloak-client/pkg/common"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	threescalev1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	kafkav1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis-products/kafka.strimzi.io/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"

	oauthv1 "github.com/openshift/api/oauth/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	fakeoauthClient "github.com/openshift/client-go/oauth/clientset/versioned/fake"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"

	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	croTypes "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	defaultOperatorNamespace = "integreatly-operator"
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
			Namespace: defaultRhssoNamespace,
		},
		Data: map[string][]byte{},
		Type: corev1.SecretTypeOpaque,
	}
}

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := threescalev1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = keycloak.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = operatorsv1alpha1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = marketplacev1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = corev1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = coreosv1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = kafkav1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = usersv1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = oauthv1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = routev1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = projectv1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = crov1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = monitoringv1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	return scheme, err
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
	}{
		{
			Name:            "test error on failed config",
			ExpectedStatus:  integreatlyv1alpha1.PhaseFailed,
			ExpectError:     true,
			ExpectedError:   "could not read rhsso config",
			Installation:    &integreatlyv1alpha1.RHMI{},
			FakeClient:      fakeclient.NewFakeClient(),
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

			status, err := testReconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.FakeClient)
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
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	kc := &keycloak.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: defaultRhssoNamespace,
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
	})

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

	//completed postgres that points at the secret croPostgresSecret
	croPostgres := &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhssouser-postgres-",
			Namespace: defaultOperatorNamespace,
		},
		Status: crov1.PostgresStatus{
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
	}{
		{
			Name:            "Test reconcile custom resource returns completed when successful created",
			FakeClient:      fakeclient.NewFakeClientWithScheme(scheme, oauthClientSecrets, githubOauthSecret, kcr, kc, group, croPostgres, croPostgresSecret),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:      basicConfigMock(),
			Installation: &integreatlyv1alpha1.RHMI{
				TypeMeta: metav1.TypeMeta{
					Kind:       integreatlyv1alpha1.SchemaGroupVersionKind.Kind,
					APIVersion: integreatlyv1alpha1.SchemeGroupVersion.String(),
				},
				ObjectMeta: controllerruntime.ObjectMeta{
					Namespace: defaultOperatorNamespace,
				},
			},
			ExpectedStatus:        integreatlyv1alpha1.PhaseCompleted,
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
		},
		{
			Name: "Test reconcile custom resource returns failed on unsuccessful create",
			FakeClient: &moqclient.SigsClientInterfaceMock{
				CreateFunc: func(ctx context.Context, obj runtime.Object, opts ...k8sclient.CreateOption) error {
					return errors.New("failed to create keycloak custom resource")
				},
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return k8serr.NewNotFound(schema.GroupResource{}, "keycloak")
				},
			},
			FakeConfig: basicConfigMock(),
			Installation: &integreatlyv1alpha1.RHMI{
				TypeMeta: metav1.TypeMeta{
					Kind:       integreatlyv1alpha1.SchemaGroupVersionKind.Kind,
					APIVersion: integreatlyv1alpha1.SchemeGroupVersion.String(),
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
			)
			if err != nil {
				t.Fatal("unexpected err ", err)
			}
			phase, err := reconciler.reconcileComponents(context.TODO(), tc.Installation, tc.FakeClient)
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

func TestReconciler_handleProgress(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	kc := &keycloak.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: defaultRhssoNamespace,
		},
	}

	kcr := getKcr(keycloak.KeycloakRealmStatus{
		Phase: keycloak.PhaseReconciling,
	})

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: defaultRhssoNamespace,
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
		Recorder              record.EventRecorder
		ApiUrl                string
		KeycloakClientFactory keycloakCommon.KeycloakClientFactory
	}{
		{
			Name:                  "test ready kcr returns phase complete",
			ExpectedStatus:        integreatlyv1alpha1.PhaseCompleted,
			FakeClient:            moqclient.NewSigsClientMoqWithScheme(scheme, secret, kc, kcr, githubOauthSecret, oauthClientSecrets),
			FakeOauthClient:       fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:            basicConfigMock(),
			Installation:          &integreatlyv1alpha1.RHMI{},
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
		},
		{
			Name:                  "test unready kcr cr returns phase in progress",
			ExpectedStatus:        integreatlyv1alpha1.PhaseInProgress,
			FakeClient:            moqclient.NewSigsClientMoqWithScheme(scheme, kc, secret, getKcr(keycloak.KeycloakRealmStatus{Phase: keycloak.PhaseFailing}), githubOauthSecret, oauthClientSecrets),
			FakeOauthClient:       fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:            basicConfigMock(),
			Installation:          &integreatlyv1alpha1.RHMI{},
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
		},
		{
			Name:                  "test missing kc cr returns phase failed",
			ExpectedStatus:        integreatlyv1alpha1.PhaseFailed,
			ExpectError:           true,
			FakeClient:            moqclient.NewSigsClientMoqWithScheme(scheme, secret, kcr, githubOauthSecret, oauthClientSecrets),
			FakeOauthClient:       fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:            basicConfigMock(),
			Installation:          &integreatlyv1alpha1.RHMI{},
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
		},
		{
			Name:                  "test missing kcr cr returns phase failed",
			ExpectedStatus:        integreatlyv1alpha1.PhaseFailed,
			ExpectError:           true,
			FakeClient:            moqclient.NewSigsClientMoqWithScheme(scheme, secret, kc, githubOauthSecret, oauthClientSecrets),
			FakeOauthClient:       fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:            basicConfigMock(),
			Installation:          &integreatlyv1alpha1.RHMI{},
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
		},
		{
			Name:            "test failed config write",
			ExpectedStatus:  integreatlyv1alpha1.PhaseFailed,
			ExpectError:     true,
			FakeClient:      moqclient.NewSigsClientMoqWithScheme(scheme, secret, kc, kcr, githubOauthSecret, oauthClientSecrets),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadRHSSOUserFunc: func() (*config.RHSSOUser, error) {
					return config.NewRHSSOUser(config.ProductConfig{
						"NAMESPACE": "user-sso",
						"REALM":     "openshift",
						"URL":       "rhsso.openshift-cluster.com",
					}), nil
				},
				WriteConfigFunc: func(config config.ConfigReadable) error {
					return errors.New("error writing config")
				},
			},
			Installation:          &integreatlyv1alpha1.RHMI{},
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
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
			)
			if err != nil && err.Error() != tc.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
			}

			status, err := testReconciler.handleProgressPhase(context.TODO(), tc.FakeClient)

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

func TestReconciler_fullReconcile(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	installation := &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "installation",
			Namespace:  defaultRhssoNamespace,
			Finalizers: []string{"finalizer.user-sso.integreatly.org"},
			UID:        types.UID("xyz"),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       integreatlyv1alpha1.SchemaGroupVersionKind.Kind,
			APIVersion: integreatlyv1alpha1.SchemeGroupVersion.String(),
		},
		Status: integreatlyv1alpha1.RHMIStatus{
			Stages: map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus{
				"codeready-stage": {
					Name: "codeready-stage",
					Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
						integreatlyv1alpha1.ProductCodeReadyWorkspaces: {
							Name:   integreatlyv1alpha1.ProductCodeReadyWorkspaces,
							Status: integreatlyv1alpha1.PhaseCreatingComponents,
						},
					},
				},
			},
		},
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultRhssoNamespace,
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
			Name: defaultRhssoNamespace + "-operator",
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
			Namespace: defaultRhssoNamespace,
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: defaultRhssoNamespace,
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
			Namespace: defaultRhssoNamespace,
		},
		Status: crov1.PostgresStatus{
			Phase: croTypes.PhaseComplete,
			SecretRef: &croTypes.SecretRef{
				Name:      "test",
				Namespace: defaultRhssoNamespace,
			},
		},
	}

	//secret created by the cloud resource operator postgres reconciler
	croPostgresSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: defaultRhssoNamespace,
		},
		Data: map[string][]byte{},
		Type: corev1.SecretTypeOpaque,
	}

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
	}{
		{
			Name:            "test successful reconcile",
			ExpectedStatus:  integreatlyv1alpha1.PhaseCompleted,
			FakeClient:      moqclient.NewSigsClientMoqWithScheme(scheme, getKcr(keycloak.KeycloakRealmStatus{Phase: keycloak.PhaseReconciling}), kc, secret, ns, operatorNS, githubOauthSecret, oauthClientSecrets, installation, edgeRoute, group, croPostgresSecret, croPostgres, getRHSSOCredentialSeed()),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:      basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, owner ownerutil.Owner, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {

					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlanList, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlanList{
							Items: []operatorsv1alpha1.InstallPlan{
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "codeready-install-plan",
									},
									Status: operatorsv1alpha1.InstallPlanStatus{
										Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
									},
								},
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
			)
			if err != nil && err.Error() != tc.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
			}

			status, err := testReconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.FakeClient)

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

func TestReconciler_reconcileCloudResources(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	installation := &integreatlyv1alpha1.RHMI{
		ObjectMeta: controllerruntime.ObjectMeta{
			Name:      "test",
			Namespace: defaultRhssoNamespace,
		},
	}

	//completed postgres that points at the secret croPostgresSecret
	croPostgres := &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("rhssouser-postgres-%s", installation.Name),
			Namespace: defaultRhssoNamespace,
		},
		Status: crov1.PostgresStatus{
			Phase: croTypes.PhaseComplete,
			SecretRef: &croTypes.SecretRef{
				Name:      "test",
				Namespace: defaultRhssoNamespace,
			},
		},
	}

	//secret created by the cloud resource operator postgres reconciler
	croPostgresSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: defaultRhssoNamespace,
		},
		Data: map[string][]byte{},
		Type: corev1.SecretTypeOpaque,
	}

	tests := []struct {
		name         string
		installation *integreatlyv1alpha1.RHMI
		fakeClient   func() k8sclient.Client
		want         integreatlyv1alpha1.StatusPhase
		wantErr      bool
	}{
		{
			name:         "error creating postgres cr causes state failed",
			installation: &integreatlyv1alpha1.RHMI{},
			fakeClient: func() k8sclient.Client {
				mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, croPostgres, croPostgresSecret)
				mockClient.GetFunc = func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return errors.New("test error")
				}
				return mockClient
			},
			wantErr: true,
			want:    integreatlyv1alpha1.PhaseFailed,
		},
		{
			name:         "nil secret causes state awaiting",
			installation: installation,
			fakeClient: func() k8sclient.Client {
				pendingCroPostgres := croPostgres.DeepCopy()
				pendingCroPostgres.Status.Phase = croTypes.PhaseInProgress
				return moqclient.NewSigsClientMoqWithScheme(scheme, croPostgresSecret, pendingCroPostgres)
			},
			want: integreatlyv1alpha1.PhaseAwaitingCloudResources,
		},
		{
			name:         "defined secret causes state completed",
			installation: installation,
			fakeClient: func() k8sclient.Client {
				return moqclient.NewSigsClientMoqWithScheme(scheme, croPostgres, croPostgresSecret)
			},
			want: integreatlyv1alpha1.PhaseCompleted,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				logger: logrus.NewEntry(logrus.StandardLogger()),
				Config: &config.RHSSOUser{
					Config: map[string]string{
						"NAMESPACE": defaultRhssoNamespace,
					},
				},
			}
			got, err := r.reconcileCloudResources(context.TODO(), tt.installation, tt.fakeClient())
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileCloudResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("reconcileCloudResources() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_reconcileFirstLoginAuthFlow(t *testing.T) {
	keycloakClientFactory, mockContext := createKeycloakClientFactoryMock()

	r := &Reconciler{
		logger: logrus.NewEntry(logrus.StandardLogger()),
		Config: &config.RHSSOUser{
			Config: map[string]string{
				"NAMESPACE": defaultRhssoNamespace,
			},
		},
		keycloakClientFactory: keycloakClientFactory,
	}

	kc := &keycloak.Keycloak{}
	statusPhase, err := r.reconcileFirstLoginAuthFlow(kc)

	if statusPhase != integreatlyv1alpha1.PhaseCompleted {
		t.Errorf("Expected phase to be completed, got %s", statusPhase)
	}
	if err != nil {
		t.Errorf("Unexpected error occurred: %s", err)
	}

	executions := mockContext.AuthenticationFlowsExecutions[firstBrokerLoginFlowAlias]

	// Iterate through the executions for the first broker login flow and, for
	// the "review profile config" execution, assert that it's disabled after
	// the reconciliation finished
	for _, execution := range executions {
		if execution.Alias != reviewProfileExecutionAlias {
			continue
		}

		if execution.Requirement != "DISABLED" {
			t.Errorf("Expected execution %s to be DISABLED, got %s", execution.Alias, execution.Requirement)
		}
	}
}

func TestReconciler_reconcileDevelopersGroup(t *testing.T) {
	keycloakClientFactory, mockContext := createKeycloakClientFactoryMock()

	r := &Reconciler{
		logger: logrus.NewEntry(logrus.StandardLogger()),
		Config: &config.RHSSOUser{
			Config: map[string]string{
				"NAMESPACE": defaultRhssoNamespace,
			},
		},
		ConfigManager:         basicConfigMock(),
		keycloakClientFactory: keycloakClientFactory,
	}

	kc := &keycloak.Keycloak{}
	statusPhase, err := r.reconcileDevelopersGroup(kc)

	if statusPhase != integreatlyv1alpha1.PhaseCompleted {
		t.Errorf("Expected phase to be completed, got %s", statusPhase)
	}
	if err != nil {
		t.Errorf("Unexpected error occurred: %s", err)
	}

	// Assert that the group `rhmi-developers` was created
	foundGroup := false
	var groupID string
	for _, group := range mockContext.Groups {
		if group.Name == developersGroupName {
			foundGroup = true
			groupID = group.ID
			break
		}
	}

	if !foundGroup {
		t.Errorf("Group %s not found in mock Keycloak interface", developersGroupName)
	}

	// Assert that the group `rhmi-developers` is among the default groups
	foundDefaultGroup := false
	for _, group := range mockContext.DefaultGroups {
		if group.Name == developersGroupName {
			foundDefaultGroup = true
			break
		}
	}

	if !foundDefaultGroup {
		t.Errorf("Group %s not found among default groups in mock Keycloak interface", developersGroupName)
	}

	// Assert that the `view-realm` role is mapped to the group
	groupRoles, ok := mockContext.ClientRoles[groupID]
	foundRole := false
	if !ok {
		t.Errorf("Group %s not found in client role mappings", developersGroupName)
	}

	for _, role := range groupRoles {
		if role.Name == viewRealmRoleName {
			foundRole = true
			break
		}
	}

	if !foundRole {
		t.Errorf("Role %s not found in client role mappings for group %s", viewRealmRoleName, developersGroupName)
	}

	// Assert that the `create-realm` role is mapped to the group
	clientRoles, ok := mockContext.RealmRoles[groupID]
	foundRole = false
	if !ok {
		t.Errorf("Group %s not found in realm role mappings", developersGroupName)
	}

	for _, role := range clientRoles {
		if role.Name == createRealmRoleName {
			foundRole = true
			break
		}
	}

	if !foundRole {
		t.Errorf("Role %s not found in realm role mappings for group %s", createRealmRoleName, developersGroupName)
	}
}

func TestReconciler_reconcileDedicatedAdminsGroup(t *testing.T) {
	keycloakClientFactory, mockContext := createKeycloakClientFactoryMock()

	r := &Reconciler{
		logger: logrus.NewEntry(logrus.StandardLogger()),
		Config: &config.RHSSOUser{
			Config: map[string]string{
				"NAMESPACE": defaultRhssoNamespace,
			},
		},
		keycloakClientFactory: keycloakClientFactory,
	}

	kc := &keycloak.Keycloak{}
	statusPhase, err := r.reconcileDedicatedAdminsGroup(kc)

	if statusPhase != integreatlyv1alpha1.PhaseCompleted {
		t.Errorf("Expected phase to be completed, got %s", statusPhase)
	}
	if err != nil {
		t.Errorf("Unexpected error occurred: %s", err)
	}

	foundDedicatedAdminsGroupID := ""
	foundRealmManagersGroupID := ""
	for _, group := range mockContext.Groups {
		if group.Name != dedicatedAdminsGroupName {
			continue
		}

		foundDedicatedAdminsGroupID = group.ID
		for _, childGroup := range group.SubGroups {
			if childGroup.Name == realmManagersGroupName {
				foundRealmManagersGroupID = childGroup.ID
			}
		}
	}

	if foundDedicatedAdminsGroupID == "" {
		t.Fatal("dedicated-admins group not found")
	}

	if foundRealmManagersGroupID == "" {
		t.Fatal("realm-managers group not found")
	}

	hasManageUsersRole := false
	hasViewRealmRole := false
	for _, clientRole := range mockContext.ClientRoles[foundDedicatedAdminsGroupID] {
		if clientRole.Name == manageUsersRoleName {
			hasManageUsersRole = true
		}
		if clientRole.Name == viewRealmRoleName {
			hasViewRealmRole = true
		}
	}

	if !hasManageUsersRole {
		t.Fatal("manage-users role not found for dedicated-admins group")
	}
	if !hasViewRealmRole {
		t.Fatal("view-realm role not found for dedicated-admins group")
	}

	for _, expectedRole := range realmManagersClientRoles {
		hasRole := false

		for _, mappedRole := range mockContext.ClientRoles[foundRealmManagersGroupID] {
			if mappedRole.Name == expectedRole {
				hasRole = true
				break
			}
		}

		if !hasRole {
			t.Errorf("Expected client role %s mapped to realm-managers group not found", expectedRole)
		}
	}
}

func getKcr(status keycloak.KeycloakRealmStatus) *keycloak.KeycloakRealm {
	return &keycloak.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      masterRealmName,
			Namespace: defaultRhssoNamespace,
		},
		Spec: keycloak.KeycloakRealmSpec{
			Realm: &keycloak.KeycloakAPIRealm{
				ID:          masterRealmName,
				Realm:       masterRealmName,
				DisplayName: masterRealmName,
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

func createKeycloakClientFactoryMock() (keycloakCommon.KeycloakClientFactory, *mockClientContext) {
	keycloakInterfaceMock, ctx := createKeycloakInterfaceMock()

	return &keycloakCommon.KeycloakClientFactoryMock{
		AuthenticatedClientFunc: func(_ keycloak.Keycloak) (keycloakCommon.KeycloakInterface, error) {
			return keycloakInterfaceMock, nil
		},
	}, ctx
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
	}, &context
}
