package rhssouser

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"testing"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	grafanav1alpha1 "github.com/integr8ly/grafana-operator/v3/pkg/apis/integreatly/v1alpha1"

	controllerruntime "sigs.k8s.io/controller-runtime"

	keycloakCommon "github.com/integr8ly/keycloak-client/pkg/common"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	threescalev1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	kafkav1alpha1 "github.com/integr8ly/integreatly-operator/apis-products/kafka.strimzi.io/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	consolev1 "github.com/openshift/api/console/v1"

	oauthv1 "github.com/openshift/api/oauth/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	fakeoauthClient "github.com/openshift/client-go/oauth/clientset/versioned/fake"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"

	crov1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	croTypes "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
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
		ReadMonitoringFunc: func() (*config.Monitoring, error) {
			return config.NewMonitoring(config.ProductConfig{
				"NAMESPACE": "middleware-monitoring",
			}), nil
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
	err = corev1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = appsv1.SchemeBuilder.AddToScheme(scheme)
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
	if err := consolev1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	err = grafanav1alpha1.SchemeBuilder.AddToScheme(scheme)
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

			status, err := testReconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.FakeClient, &quota.ProductConfigMock{})
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
			FakeClient:      fakeclient.NewFakeClientWithScheme(scheme, oauthClientSecrets, githubOauthSecret, kcr, kc, group, croPostgres, croPostgresSecret),
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
	scheme, err := getBuildScheme()
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
		ProductConfig         *quota.ProductConfigMock
	}{
		{
			Name:            "test successful reconcile",
			ExpectedStatus:  integreatlyv1alpha1.PhaseCompleted,
			FakeClient:      moqclient.NewSigsClientMoqWithScheme(scheme, getKcr(keycloak.KeycloakRealmStatus{Phase: keycloak.PhaseReconciling}), kc, secret, ns, operatorNS, githubOauthSecret, oauthClientSecrets, installation, edgeRoute, group, croPostgresSecret, croPostgres, getRHSSOCredentialSeed(), statefulSet),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:      basicConfigMock(),
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

			status, err := testReconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.FakeClient, tc.ProductConfig)

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
	scheme, err := getBuildScheme()
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
						integreatlyv1alpha1.ProductCodeReadyWorkspaces: {
							Name:   integreatlyv1alpha1.ProductRHSSO,
							Status: integreatlyv1alpha1.PhaseCreatingComponents,
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
		ProductConfig         *quota.ProductConfigMock
	}{
		{
			Name:            "RHOAM - test successful reconcile",
			ExpectedStatus:  integreatlyv1alpha1.PhaseCompleted,
			FakeClient:      moqclient.NewSigsClientMoqWithScheme(scheme, getKcr(keycloak.KeycloakRealmStatus{Phase: keycloak.PhaseReconciling}), kc, secret, ns, operatorNS, githubOauthSecret, oauthClientSecrets, installation, edgeRoute, group, croPostgresSecret, croPostgres, getRHSSOCredentialSeed(), statefulSet),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:      basicConfigMock(),
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

			status, err := testReconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.FakeClient, tc.ProductConfig)

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
			Name:      masterRealmName,
			Namespace: defaultNamespace,
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

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.ProductRHSSO})
}
