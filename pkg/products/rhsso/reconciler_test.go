package rhsso

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	controllerruntime "sigs.k8s.io/controller-runtime"

	threescalev1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"

	monitoring "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	kafkav1 "github.com/integr8ly/integreatly-operator/pkg/apis/kafka.strimzi.io/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	oauthv1 "github.com/openshift/api/oauth/v1"
	projectv1 "github.com/openshift/api/project/v1"
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
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
		ReadRHSSOFunc: func() (*config.RHSSO, error) {
			return config.NewRHSSO(config.ProductConfig{
				"NAMESPACE": "rhsso",
				"REALM":     "openshift",
				"URL":       "rhsso.openshift-cluster.com",
				"HOST":      "edge/route",
			}), nil
		},
		ReadMonitoringFunc: func() (*config.Monitoring, error) {
			return config.NewMonitoring(config.ProductConfig{
				"NAMESPACE": "middleware-monitoring",
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
	err = kafkav1.SchemeBuilder.AddToScheme(scheme)
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
	err = monitoring.SchemeBuilder.AddToScheme(scheme)
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
	return scheme, err
}

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
}

func TestReconciler_config(t *testing.T) {
	cases := []struct {
		Name            string
		ExpectError     bool
		ExpectedStatus  integreatlyv1alpha1.StatusPhase
		ExpectedError   string
		FakeConfig      *config.ConfigReadWriterMock
		FakeClient      k8sclient.Client
		FakeOauthClient oauthClient.OauthV1Interface
		FakeMPM         *marketplace.MarketplaceInterfaceMock
		Installation    *integreatlyv1alpha1.RHMI
		Product         *integreatlyv1alpha1.RHMIProductStatus
		Recorder        record.EventRecorder
		APIURL          string
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
				ReadRHSSOFunc: func() (ready *config.RHSSO, e error) {
					return nil, errors.New("could not read rhsso config")
				},
			},
			Product:  &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder: setupRecorder(),
			APIURL:   "https://serverurl",
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
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	kc := &keycloak.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: defaultOperandNamespace,
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
		Name            string
		FakeClient      k8sclient.Client
		FakeOauthClient oauthClient.OauthV1Interface
		FakeConfig      *config.ConfigReadWriterMock
		Installation    *integreatlyv1alpha1.RHMI
		ExpectError     bool
		ExpectedError   string
		ExpectedStatus  integreatlyv1alpha1.StatusPhase
		FakeMPM         *marketplace.MarketplaceInterfaceMock
		Recorder        record.EventRecorder
		ApiUrl          string
	}{
		{
			Name:            "Test reconcile custom resource returns completed when successful created",
			FakeClient:      fakeclient.NewFakeClientWithScheme(scheme, oauthClientSecrets, githubOauthSecret, kc, croPostgres, croPostgresSecret),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:      basicConfigMock(),
			Installation: &integreatlyv1alpha1.RHMI{
				TypeMeta: metav1.TypeMeta{
					Kind:       integreatlyv1alpha1.SchemaGroupVersionKind.Kind,
					APIVersion: integreatlyv1alpha1.SchemeGroupVersion.String(),
				},
			},
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Recorder:       setupRecorder(),
			ApiUrl:         "https://serverurl",
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
			},
			ExpectError:    true,
			ExpectedError:  "failed to create/update keycloak custom resource: failed to create keycloak custom resource",
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			Recorder:       setupRecorder(),
			ApiUrl:         "https://serverurl",
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
			Namespace: defaultOperandNamespace,
		},
	}

	kcr := getKcr(keycloak.KeycloakRealmStatus{
		Phase: keycloak.PhaseReconciling,
	})

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
			"rhsso": bytes.NewBufferString("test").Bytes(),
		},
	}

	cases := []struct {
		Name            string
		ExpectError     bool
		ExpectedStatus  integreatlyv1alpha1.StatusPhase
		ExpectedError   string
		FakeConfig      *config.ConfigReadWriterMock
		FakeClient      k8sclient.Client
		FakeOauthClient oauthClient.OauthV1Interface
		FakeMPM         *marketplace.MarketplaceInterfaceMock
		Installation    *integreatlyv1alpha1.RHMI
		Recorder        record.EventRecorder
		ApiUrl          string
	}{
		{
			Name:            "test ready kcr returns phase complete",
			ExpectedStatus:  integreatlyv1alpha1.PhaseCompleted,
			FakeClient:      moqclient.NewSigsClientMoqWithScheme(scheme, secret, kc, kcr, githubOauthSecret, oauthClientSecrets),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:      basicConfigMock(),
			Installation:    &integreatlyv1alpha1.RHMI{},
			Recorder:        setupRecorder(),
			ApiUrl:          "https://serverurl",
		},
		{
			Name:            "test unready kcr cr returns phase in progress",
			ExpectedStatus:  integreatlyv1alpha1.PhaseInProgress,
			FakeClient:      moqclient.NewSigsClientMoqWithScheme(scheme, kc, secret, getKcr(keycloak.KeycloakRealmStatus{Phase: keycloak.PhaseFailing}), githubOauthSecret, oauthClientSecrets),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:      basicConfigMock(),
			Installation:    &integreatlyv1alpha1.RHMI{},
			Recorder:        setupRecorder(),
			ApiUrl:          "https://serverurl",
		},
		{
			Name:            "test missing kc cr returns phase failed",
			ExpectedStatus:  integreatlyv1alpha1.PhaseFailed,
			ExpectError:     true,
			FakeClient:      moqclient.NewSigsClientMoqWithScheme(scheme, secret, kcr, githubOauthSecret, oauthClientSecrets),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:      basicConfigMock(),
			Installation:    &integreatlyv1alpha1.RHMI{},
			Recorder:        setupRecorder(),
			ApiUrl:          "https://serverurl",
		},
		{
			Name:            "test missing github secret cr returns phase failed",
			ExpectedStatus:  integreatlyv1alpha1.PhaseFailed,
			ExpectError:     true,
			FakeClient:      moqclient.NewSigsClientMoqWithScheme(scheme, secret, kcr, oauthClientSecrets),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:      basicConfigMock(),
			Installation:    &integreatlyv1alpha1.RHMI{},
			Recorder:        setupRecorder(),
			ApiUrl:          "https://serverurl",
		},
		{
			Name:            "test missing kcr cr returns phase failed",
			ExpectedStatus:  integreatlyv1alpha1.PhaseFailed,
			ExpectError:     true,
			FakeClient:      moqclient.NewSigsClientMoqWithScheme(scheme, secret, kc, githubOauthSecret, oauthClientSecrets),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:      basicConfigMock(),
			Installation:    &integreatlyv1alpha1.RHMI{},
			Recorder:        setupRecorder(),
			ApiUrl:          "https://serverurl",
		},
		{
			Name:            "test failed config write",
			ExpectedStatus:  integreatlyv1alpha1.PhaseFailed,
			ExpectError:     true,
			FakeClient:      moqclient.NewSigsClientMoqWithScheme(scheme, secret, kc, kcr, githubOauthSecret, oauthClientSecrets),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
						"REALM":     "openshift",
						"URL":       "rhsso.openshift-cluster.com",
					}), nil
				},
				WriteConfigFunc: func(config config.ConfigReadable) error {
					return errors.New("error writing config")
				},
				GetOauthClientsSecretNameFunc: func() string {
					return "oauth-client-secrets"
				},
			},
			Installation: &integreatlyv1alpha1.RHMI{},
			Recorder:     setupRecorder(),
			ApiUrl:       "https://serverurl",
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
			Namespace:  defaultOperandNamespace,
			Finalizers: []string{"finalizer.rhsso.integreatly.org"},
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

	//completed postgres that points at the secret croPostgresSecret
	croPostgres := &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("rhsso-postgres-%s", installation.Name),
			Namespace: defaultOperandNamespace,
		},
		Status: crov1.PostgresStatus{
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

	cases := []struct {
		Name            string
		ExpectError     bool
		ExpectedStatus  integreatlyv1alpha1.StatusPhase
		ExpectedError   string
		FakeConfig      *config.ConfigReadWriterMock
		FakeClient      k8sclient.Client
		FakeOauthClient oauthClient.OauthV1Interface
		FakeMPM         *marketplace.MarketplaceInterfaceMock
		Installation    *integreatlyv1alpha1.RHMI
		Product         *integreatlyv1alpha1.RHMIProductStatus
		Recorder        record.EventRecorder
		ApiUrl          string
	}{
		{
			Name:            "test successful reconcile",
			ExpectedStatus:  integreatlyv1alpha1.PhaseCompleted,
			FakeClient:      moqclient.NewSigsClientMoqWithScheme(scheme, getKcr(keycloak.KeycloakRealmStatus{Phase: keycloak.PhaseReconciling}), kc, secret, ns, operatorNS, githubOauthSecret, oauthClientSecrets, installation, edgeRoute, croPostgres, croPostgresSecret),
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
			Installation: installation,
			Product:      &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:     setupRecorder(),
			ApiUrl:       "https://serverurl",
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
			Namespace: defaultOperatorNamespace,
		},
	}

	//completed postgres that points at the secret croPostgresSecret
	croPostgres := &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("rhsso-postgres-%s", installation.Name),
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
				Config: &config.RHSSO{
					Config: map[string]string{
						"NAMESPACE": defaultOperatorNamespace,
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
