package rhsso

import (
	"bytes"
	"context"
	"errors"
	"testing"

	threescalev1 "github.com/integr8ly/integreatly-operator/pkg/apis/3scale/v1alpha1"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	kafkav1 "github.com/integr8ly/integreatly-operator/pkg/apis/kafka.strimzi.io/v1alpha1"
	monitoring "github.com/integr8ly/integreatly-operator/pkg/apis/monitoring/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	oauthv1 "github.com/openshift/api/oauth/v1"
	usersv1 "github.com/openshift/api/user/v1"
	fakeoauthClient "github.com/openshift/client-go/oauth/clientset/versioned/fake"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"

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
	}
}

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := threescalev1.SchemeBuilder.AddToScheme(scheme)
	err = keycloak.SchemeBuilder.AddToScheme(scheme)
	err = integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	err = operatorsv1alpha1.AddToScheme(scheme)
	err = marketplacev1.SchemeBuilder.AddToScheme(scheme)
	err = corev1.SchemeBuilder.AddToScheme(scheme)
	err = coreosv1.SchemeBuilder.AddToScheme(scheme)
	err = kafkav1.SchemeBuilder.AddToScheme(scheme)
	err = usersv1.AddToScheme(scheme)
	err = oauthv1.AddToScheme(scheme)
	err = monitoring.SchemeBuilder.AddToScheme(scheme)
	err = routev1.AddToScheme(scheme)
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
		Installation    *integreatlyv1alpha1.Installation
		Product         *integreatlyv1alpha1.InstallationProductStatus
		Recorder        record.EventRecorder
		APIURL          string
	}{
		{
			Name:            "test error on failed config",
			ExpectedStatus:  integreatlyv1alpha1.PhaseFailed,
			ExpectError:     true,
			ExpectedError:   "could not read rhsso config",
			Installation:    &integreatlyv1alpha1.Installation{},
			FakeClient:      fakeclient.NewFakeClient(),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadRHSSOFunc: func() (ready *config.RHSSO, e error) {
					return nil, errors.New("could not read rhsso config")
				},
			},
			Product:  &integreatlyv1alpha1.InstallationProductStatus{},
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
	}

	cases := []struct {
		Name            string
		FakeClient      k8sclient.Client
		FakeOauthClient oauthClient.OauthV1Interface
		FakeConfig      *config.ConfigReadWriterMock
		Installation    *integreatlyv1alpha1.Installation
		ExpectError     bool
		ExpectedError   string
		ExpectedStatus  integreatlyv1alpha1.StatusPhase
		FakeMPM         *marketplace.MarketplaceInterfaceMock
		Recorder        record.EventRecorder
		APIURL          string
	}{
		{
			Name:            "Test reconcile custom resource returns completed when successful created",
			FakeClient:      fakeclient.NewFakeClientWithScheme(scheme, oauthClientSecrets, githubOauthSecret),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:      basicConfigMock(),
			Installation: &integreatlyv1alpha1.Installation{
				TypeMeta: metav1.TypeMeta{
					Kind:       integreatlyv1alpha1.SchemaGroupVersionKind.Kind,
					APIVersion: integreatlyv1alpha1.SchemeGroupVersion.String(),
				},
			},
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Recorder:       setupRecorder(),
			APIURL:         "https://serverurl",
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
			Installation: &integreatlyv1alpha1.Installation{
				TypeMeta: metav1.TypeMeta{
					Kind:       integreatlyv1alpha1.SchemaGroupVersionKind.Kind,
					APIVersion: integreatlyv1alpha1.SchemeGroupVersion.String(),
				},
			},
			ExpectError:    true,
			ExpectedError:  "failed to create/update keycloak custom resource: failed to create keycloak custom resource",
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			Recorder:       setupRecorder(),
			APIURL:         "https://serverurl",
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
				tc.APIURL,
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
		Installation    *integreatlyv1alpha1.Installation
		Recorder        record.EventRecorder
		APIURL          string
	}{
		{
			Name:            "test ready kcr returns phase complete",
			ExpectedStatus:  integreatlyv1alpha1.PhaseCompleted,
			FakeClient:      moqclient.NewSigsClientMoqWithScheme(scheme, secret, kc, kcr, githubOauthSecret, oauthClientSecrets),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:      basicConfigMock(),
			Installation:    &integreatlyv1alpha1.Installation{},
			Recorder:        setupRecorder(),
			APIURL:          "https://serverurl",
		},
		{
			Name:            "test unready kcr cr returns phase in progress",
			ExpectedStatus:  integreatlyv1alpha1.PhaseInProgress,
			FakeClient:      moqclient.NewSigsClientMoqWithScheme(scheme, kc, secret, getKcr(keycloak.KeycloakRealmStatus{Phase: keycloak.PhaseFailing}), githubOauthSecret, oauthClientSecrets),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:      basicConfigMock(),
			Installation:    &integreatlyv1alpha1.Installation{},
			Recorder:        setupRecorder(),
			APIURL:          "https://serverurl",
		},
		{
			Name:            "test missing kc cr returns phase failed",
			ExpectedStatus:  integreatlyv1alpha1.PhaseFailed,
			ExpectError:     true,
			FakeClient:      moqclient.NewSigsClientMoqWithScheme(scheme, secret, kcr, githubOauthSecret, oauthClientSecrets),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:      basicConfigMock(),
			Installation:    &integreatlyv1alpha1.Installation{},
			Recorder:        setupRecorder(),
			APIURL:          "https://serverurl",
		},
		{
			Name:            "test missing kcr cr returns phase failed",
			ExpectedStatus:  integreatlyv1alpha1.PhaseFailed,
			ExpectError:     true,
			FakeClient:      moqclient.NewSigsClientMoqWithScheme(scheme, secret, kc, githubOauthSecret, oauthClientSecrets),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:      basicConfigMock(),
			Installation:    &integreatlyv1alpha1.Installation{},
			Recorder:        setupRecorder(),
			APIURL:          "https://serverurl",
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
			Installation: &integreatlyv1alpha1.Installation{},
			Recorder:     setupRecorder(),
			APIURL:       "https://serverurl",
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

	installation := &integreatlyv1alpha1.Installation{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "installation",
			Namespace:  defaultRhssoNamespace,
			Finalizers: []string{"finalizer.rhsso.integreatly.org"},
			UID:        types.UID("xyz"),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       integreatlyv1alpha1.SchemaGroupVersionKind.Kind,
			APIVersion: integreatlyv1alpha1.SchemeGroupVersion.String(),
		},
		Status: integreatlyv1alpha1.InstallationStatus{
			Stages: map[integreatlyv1alpha1.StageName]*integreatlyv1alpha1.InstallationStageStatus{
				"codeready-stage": {
					Name: "codeready-stage",
					Products: map[integreatlyv1alpha1.ProductName]*integreatlyv1alpha1.InstallationProductStatus{
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
			"rhsso":  bytes.NewBufferString("test").Bytes(),
			"3scale": bytes.NewBufferString("test").Bytes(),
		},
	}

	edgeRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak-edge",
			Namespace: defaultRhssoNamespace,
		},
		Spec: routev1.RouteSpec{
			Host: "sampleHost",
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
		Installation    *integreatlyv1alpha1.Installation
		Product         *integreatlyv1alpha1.InstallationProductStatus
		Recorder        record.EventRecorder
		APIURL          string
	}{
		{
			Name:            "test successful reconcile",
			ExpectedStatus:  integreatlyv1alpha1.PhaseCompleted,
			FakeClient:      moqclient.NewSigsClientMoqWithScheme(scheme, getKcr(keycloak.KeycloakRealmStatus{Phase: keycloak.PhaseReconciling}), kc, secret, ns, githubOauthSecret, oauthClientSecrets, installation, edgeRoute),
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
			Product:      &integreatlyv1alpha1.InstallationProductStatus{},
			Recorder:     setupRecorder(),
			APIURL:       "https://serverurl",
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

func getKcr(status keycloak.KeycloakRealmStatus) *keycloak.KeycloakRealm {
	return &keycloak.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakRealmName,
			Namespace: defaultRhssoNamespace,
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
