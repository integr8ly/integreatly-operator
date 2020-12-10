package codeready

import (
	"context"
	"errors"
	"fmt"
	"testing"

	prometheusmonitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"

	chev1 "github.com/eclipse/che-operator/pkg/apis/org/v1"

	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	types2 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"

	monitoring "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	kafkav1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis-products/kafka.strimzi.io/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"

	projectv1 "github.com/openshift/api/project/v1"

	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"

	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/integr8ly/integreatly-operator/pkg/config"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
)

var testKeycloakClient = &keycloak.KeycloakClient{
	Spec: keycloak.KeycloakClientSpec{
		Client: &keycloak.KeycloakAPIClient{
			Name: defaultClientName,
		},
	},
}

var testKeycloakRealm = &keycloak.KeycloakRealm{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "openshift",
		Namespace: "rhsso",
	},
	Spec: keycloak.KeycloakRealmSpec{
		Realm: &keycloak.KeycloakAPIRealm{
			Realm: defaultClientName,
		},
	},
}

var testCheCluster = chev1.CheCluster{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: defaultInstallationNamespace,
		Name:      defaultCheClusterName,
	},
}

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
				"HOST":      "rhsso.openshift-cluster.com",
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
		GetBackupsSecretNameFunc: func() string {
			return "backups-s3-credentials"
		},
	}
}

func backupsSecretMock() *corev1.Secret {
	config := basicConfigMock()
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.GetBackupsSecretNameFunc(),
			Namespace: config.GetOperatorNamespace(),
		},
		Data: map[string][]byte{},
	}
}

func buildScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	chev1.SchemeBuilder.AddToScheme(scheme)
	keycloak.SchemeBuilder.AddToScheme(scheme)
	integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	operatorsv1alpha1.AddToScheme(scheme)
	marketplacev1.SchemeBuilder.AddToScheme(scheme)
	kafkav1alpha1.SchemeBuilder.AddToScheme(scheme)
	corev1.SchemeBuilder.AddToScheme(scheme)
	rbacv1.SchemeBuilder.AddToScheme(scheme)
	batchv1beta1.SchemeBuilder.AddToScheme(scheme)
	appsv1.SchemeBuilder.AddToScheme(scheme)
	monitoring.SchemeBuilder.AddToScheme(scheme)
	prometheusmonitoringv1.SchemeBuilder.AddToScheme(scheme)
	crov1.SchemeBuilder.AddToScheme(scheme)
	projectv1.AddToScheme(scheme)
	return scheme
}

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
}

func TestReconciler_config(t *testing.T) {

	cases := []struct {
		Name           string
		ExpectError    bool
		ExpectedStatus integreatlyv1alpha1.StatusPhase
		ExpectedError  string
		FakeConfig     *config.ConfigReadWriterMock
		FakeClient     k8sclient.Client
		FakeMPM        *marketplace.MarketplaceInterfaceMock
		Installation   *integreatlyv1alpha1.RHMI
		Product        *integreatlyv1alpha1.RHMIProductStatus
		Recorder       record.EventRecorder
		Logger         l.Logger
	}{
		{
			Name:           "test error on failed config",
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			ExpectError:    true,
			ExpectedError:  "could not retrieve che config: could not read che config",
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeClient:     fakeclient.NewFakeClient(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadCodeReadyFunc: func() (ready *config.CodeReady, e error) {
					return nil, errors.New("could not read che config")
				},
			},
			Product:  &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder: setupRecorder(),
			Logger:   l.NewLogger(),
		},
		{
			Name:           "test subscription phase with error from mpm",
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			ExpectError:    true,
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {

					return errors.New("dummy error")
				},
			},
			FakeClient: fakeclient.NewFakeClient(),
			FakeConfig: basicConfigMock(),
			Product:    &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:   setupRecorder(),
			Logger:     l.NewLogger(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			testReconciler, err := NewReconciler(
				tc.FakeConfig,
				tc.Installation,
				tc.FakeMPM,
				tc.Recorder,
				tc.Logger,
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

func TestCodeready_reconcileCluster(t *testing.T) {

	pg := &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codeready-postgres-",
			Namespace: "codeready-workspaces",
		},
		Spec: crov1.PostgresSpec{},
		Status: crov1.PostgresStatus{
			Phase: types2.PhaseComplete,
			SecretRef: &types2.SecretRef{
				Name: "codeready-postgres-",
			},
		},
	}

	scenarios := []struct {
		Name           string
		ExpectedStatus integreatlyv1alpha1.StatusPhase
		ExpectError    bool
		ExpectedError  string
		Installation   *integreatlyv1alpha1.RHMI
		FakeConfig     *config.ConfigReadWriterMock
		FakeClient     k8sclient.Client
		FakeMPM        *marketplace.MarketplaceInterfaceMock
		Recorder       record.EventRecorder
		Logger         l.Logger
	}{
		{
			Name:           "test phase in progress when che cluster is missing",
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: defaultInstallationNamespace,
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       integreatlyv1alpha1.SchemaGroupVersionKind.Kind,
					APIVersion: integreatlyv1alpha1.SchemeGroupVersion.String(),
				},
			},
			FakeClient: fakeclient.NewFakeClientWithScheme(buildScheme(), testKeycloakClient, testKeycloakRealm, pg, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "codeready-postgres-",
				},
			}),
			FakeConfig: basicConfigMock(),
			Recorder:   setupRecorder(),
			Logger:     l.NewLogger(),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			testReconciler, err := NewReconciler(
				scenario.FakeConfig,
				scenario.Installation,
				scenario.FakeMPM,
				scenario.Recorder,
				scenario.Logger,
			)
			if err != nil && err.Error() != scenario.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, scenario.ExpectedError)
			}

			status, err := testReconciler.reconcileCheCluster(context.TODO(), scenario.FakeClient)
			if err != nil && err.Error() != scenario.ExpectedError {
				t.Fatalf("unexpected error: %v, expected: %v", err, scenario.ExpectedError)
			}

			if err == nil && scenario.ExpectedError != "" {
				t.Fatalf("expected error '%v' and got nil", scenario.ExpectedError)
			}

			if status != scenario.ExpectedStatus {
				t.Fatalf("Expected status: '%v', got: '%v'", scenario.ExpectedStatus, status)
			}
		})
	}
}

func TestCodeready_reconcileClient(t *testing.T) {
	scenarios := []struct {
		Name                string
		ExpectedStatus      integreatlyv1alpha1.StatusPhase
		ExpectedError       string
		ExpectedCreateError string
		Installation        *integreatlyv1alpha1.RHMI
		FakeConfig          *config.ConfigReadWriterMock
		FakeClient          k8sclient.Client
		FakeMPM             *marketplace.MarketplaceInterfaceMock
		Recorder            record.EventRecorder
		Logger              l.Logger
	}{
		{
			Name:           "test creating components phase missing cluster expect p",
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			Installation: &integreatlyv1alpha1.RHMI{
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
									Status: integreatlyv1alpha1.PhaseCreatingSubscription,
								},
							},
						},
					},
				},
			},
			ExpectedError: fmt.Sprintf("could not retrieve checluster for keycloak client update: checlusters.org.eclipse.che \"%s\" not found", defaultCheClusterName),
			FakeClient:    fakeclient.NewFakeClientWithScheme(buildScheme(), testKeycloakClient, testKeycloakRealm),
			FakeConfig:    basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {

					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlanList, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlanList{}, &operatorsv1alpha1.Subscription{}, nil
				},
			},
			Recorder: setupRecorder(),
			Logger:   l.NewLogger(),
		},
		{
			Name:           "test creating components returns in progress",
			ExpectedStatus: integreatlyv1alpha1.PhaseInProgress,
			Installation: &integreatlyv1alpha1.RHMI{
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
									Status: integreatlyv1alpha1.PhaseCreatingSubscription,
								},
							},
						},
					},
				},
			},
			FakeClient: fakeclient.NewFakeClientWithScheme(buildScheme(), testKeycloakClient, testKeycloakRealm, &testCheCluster),
			FakeConfig: basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {

					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlanList, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlanList{}, &operatorsv1alpha1.Subscription{}, nil
				},
			},
			Recorder: setupRecorder(),
			Logger:   l.NewLogger(),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			testReconciler, err := NewReconciler(
				scenario.FakeConfig,
				scenario.Installation,
				scenario.FakeMPM,
				scenario.Recorder,
				scenario.Logger,
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

			status, err := testReconciler.reconcileKeycloakClient(context.TODO(), scenario.FakeClient)
			if err != nil && err.Error() != scenario.ExpectedError {
				t.Fatalf("unexpected error: %v, expected: %v", err, scenario.ExpectedError)
			}

			if err == nil && scenario.ExpectedError != "" {
				t.Fatalf("expected error '%v' and got nil", scenario.ExpectedError)
			}

			if status != scenario.ExpectedStatus {
				t.Fatalf("Expected status: '%v', got: '%v'", scenario.ExpectedStatus, status)
			}
		})
	}
}

func TestCodeready_reconcileProgress(t *testing.T) {
	scenarios := []struct {
		Name           string
		ExpectedStatus integreatlyv1alpha1.StatusPhase
		ExpectedError  string
		ExpectError    bool
		Installation   *integreatlyv1alpha1.RHMI
		FakeConfig     *config.ConfigReadWriterMock
		FakeClient     k8sclient.Client
		FakeMPM        *marketplace.MarketplaceInterfaceMock
		Recorder       record.EventRecorder
		Logger         l.Logger
	}{
		{
			Name:           "test che cluster creating returns phase in progress",
			ExpectedStatus: integreatlyv1alpha1.PhaseInProgress,
			Installation: &integreatlyv1alpha1.RHMI{
				TypeMeta: metav1.TypeMeta{
					Kind:       integreatlyv1alpha1.SchemaGroupVersionKind.Kind,
					APIVersion: integreatlyv1alpha1.SchemeGroupVersion.String(),
				},
			},
			FakeClient: fakeclient.NewFakeClientWithScheme(buildScheme(), testKeycloakClient, testKeycloakRealm, &testCheCluster),
			FakeConfig: basicConfigMock(),
			Recorder:   setupRecorder(),
			Logger:     l.NewLogger(),
		},
		{
			Name:           "test che cluster create error returns phase failed",
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			Installation: &integreatlyv1alpha1.RHMI{
				TypeMeta: metav1.TypeMeta{
					Kind:       integreatlyv1alpha1.SchemaGroupVersionKind.Kind,
					APIVersion: integreatlyv1alpha1.SchemeGroupVersion.String(),
				},
			},
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return errors.New("get request failed")
				},
			},
			ExpectError:   true,
			ExpectedError: "could not retrieve checluster for keycloak client update: get request failed",
			FakeConfig:    basicConfigMock(),
			Recorder:      setupRecorder(),
			Logger:        l.NewLogger(),
		},
		{
			Name:           "test che cluster available returns phase complete",
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Installation: &integreatlyv1alpha1.RHMI{
				TypeMeta: metav1.TypeMeta{
					Kind:       integreatlyv1alpha1.SchemaGroupVersionKind.Kind,
					APIVersion: integreatlyv1alpha1.SchemeGroupVersion.String(),
				},
			},
			FakeClient: moqclient.NewSigsClientMoqWithScheme(buildScheme(), &chev1.CheCluster{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: defaultInstallationNamespace,
					Name:      defaultCheClusterName,
				},
				Status: chev1.CheClusterStatus{
					CheClusterRunning: "Available",
				},
			}),
			FakeConfig: basicConfigMock(),
			Recorder:   setupRecorder(),
			Logger:     l.NewLogger(),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			testReconciler, err := NewReconciler(
				scenario.FakeConfig,
				scenario.Installation,
				scenario.FakeMPM,
				scenario.Recorder,
				scenario.Logger,
			)
			if err != nil && err.Error() != scenario.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, scenario.ExpectedError)
			}

			status, err := testReconciler.handleProgressPhase(context.TODO(), scenario.FakeClient)
			if err != nil && err.Error() != scenario.ExpectedError {
				t.Fatalf("unexpected error: %v, expected: %v", err, scenario.ExpectedError)
			}

			if err == nil && scenario.ExpectedError != "" {
				t.Fatalf("expected error '%v' and got nil", scenario.ExpectedError)
			}

			if status != scenario.ExpectedStatus {
				t.Fatalf("Expected status: '%v', got: '%v'", scenario.ExpectedStatus, status)
			}
		})
	}
}

func TestCodeready_fullReconcile(t *testing.T) {
	installation := &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "installation",
			Namespace: defaultInstallationNamespace,
			UID:       types.UID("xyz"),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       integreatlyv1alpha1.SchemaGroupVersionKind.Kind,
			APIVersion: integreatlyv1alpha1.SchemeGroupVersion.String(),
		},
	}
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultInstallationNamespace,
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
			Name: defaultInstallationNamespace + "-operator",
			Labels: map[string]string{
				resources.OwnerLabelKey: string(installation.GetUID()),
			},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}

	cluster := &chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defaultInstallationNamespace,
			Name:      defaultCheClusterName,
		},
		Spec: chev1.CheClusterSpec{},
		Status: chev1.CheClusterStatus{
			CheURL: "https://test.com",
		},
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "integreatly-operator",
			Name:      "codeready-postgres-installation",
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
	}

	pg := &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codeready-postgres-installation",
			Namespace: "codeready-workspaces",
		},
		Spec: crov1.PostgresSpec{},
		Status: crov1.PostgresStatus{
			Phase: types2.PhaseComplete,
			SecretRef: &types2.SecretRef{
				Name:      "test",
				Namespace: "test",
			},
		},
	}

	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}

	scenarios := []struct {
		Name                string
		ExpectedStatus      integreatlyv1alpha1.StatusPhase
		ExpectedError       string
		ExpectedCreateError string
		Installation        *integreatlyv1alpha1.RHMI
		FakeConfig          *config.ConfigReadWriterMock
		FakeClient          k8sclient.Client
		FakeMPM             *marketplace.MarketplaceInterfaceMock
		ValidateCallCounts  func(mockConfig *config.ConfigReadWriterMock, mockMPM *marketplace.MarketplaceInterfaceMock, t *testing.T)
		Product             *integreatlyv1alpha1.RHMIProductStatus
		Recorder            record.EventRecorder
		Logger              l.Logger
	}{
		{
			Name:           "test successful installation without errors",
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Installation:   installation,
			FakeClient:     fakeclient.NewFakeClientWithScheme(buildScheme(), testKeycloakClient, testKeycloakRealm, dep, ns, operatorNS, cluster, installation, pg, sec, backupsSecretMock()),
			FakeConfig:     basicConfigMock(),
			ValidateCallCounts: func(mockConfig *config.ConfigReadWriterMock, mockMPM *marketplace.MarketplaceInterfaceMock, t *testing.T) {
				if len(mockConfig.ReadCodeReadyCalls()) != 1 {
					t.Fatalf("expected 1 call to readCodeReady config, got: %d", len(mockConfig.ReadCodeReadyCalls()))
				}
				if len(mockConfig.ReadRHSSOCalls()) != 2 {
					t.Fatalf("expected 2 calls to readRHSSO config, got: %d", len(mockConfig.ReadRHSSOCalls()))
				}
				if mockMPM == nil {
					t.Fatalf("expected MPM not to be nil but it was nil ")
				}
				if len(mockMPM.InstallOperatorCalls()) != 1 {
					t.Fatalf("expected CreateSubscriptionCalls to be 1 bug got %d", len(mockMPM.InstallOperatorCalls()))
				}
				if len(mockMPM.GetSubscriptionInstallPlansCalls()) != 1 {
					t.Fatalf("expected GetSubscriptionInstallPlansCalls to be 1 bug got %d", len(mockMPM.GetSubscriptionInstallPlansCalls()))
				}
			},
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {

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
			Product:  &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder: setupRecorder(),
			Logger:   l.NewLogger(),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			testReconciler, err := NewReconciler(
				scenario.FakeConfig,
				scenario.Installation,
				scenario.FakeMPM,
				scenario.Recorder,
				scenario.Logger,
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

			status, err := testReconciler.Reconcile(context.TODO(), scenario.Installation, scenario.Product, scenario.FakeClient)
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

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.ProductCodeReadyWorkspaces})
}
