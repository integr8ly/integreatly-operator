package codeready

import (
	"context"
	"testing"

	chev1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"

	kafkav1 "github.com/integr8ly/integreatly-operator/pkg/apis/kafka.strimzi.io/v1alpha1"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	operatorsv1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	corev1 "k8s.io/api/core/v1"

	rbacv1 "k8s.io/api/rbac/v1"

	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
)

var testKeycloakRealm = aerogearv1.KeycloakRealm{
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
		WriteConfigFunc: func(config config.ConfigReadable) error {
			return nil
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

func TestReconciler_config(t *testing.T) {

	cases := []struct {
		Name           string
		ExpectError    bool
		ExpectedStatus v1alpha1.StatusPhase
		ExpectedError  string
		FakeConfig     *config.ConfigReadWriterMock
		FakeClient     pkgclient.Client
		FakeMPM        *marketplace.MarketplaceInterfaceMock
		Installation   *v1alpha1.Installation
		Product        *v1alpha1.InstallationProductStatus
	}{
		{
			Name:           "test error on failed config",
			ExpectedStatus: v1alpha1.PhaseFailed,
			ExpectError:    true,
			ExpectedError:  "could not retrieve che config: could not read che config",
			Installation:   &v1alpha1.Installation{},
			FakeClient:     fakeclient.NewFakeClient(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadCodeReadyFunc: func() (ready *config.CodeReady, e error) {
					return nil, errors.New("could not read che config")
				},
			},
			Product: &v1alpha1.InstallationProductStatus{},
		},
		{
			Name:           "test subscription phase with error from mpm",
			ExpectedStatus: v1alpha1.PhaseFailed,
			ExpectError:    true,
			Installation:   &v1alpha1.Installation{},
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, os operatorsv1.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {

					return errors.New("dummy error")
				},
			},
			FakeClient: fakeclient.NewFakeClient(),
			FakeConfig: basicConfigMock(),
			Product:    &v1alpha1.InstallationProductStatus{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			testReconciler, err := NewReconciler(
				tc.FakeConfig,
				tc.Installation,
				tc.FakeMPM,
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
	scenarios := []struct {
		Name           string
		ExpectedStatus v1alpha1.StatusPhase
		ExpectError    bool
		ExpectedError  string
		Installation   *v1alpha1.Installation
		FakeConfig     *config.ConfigReadWriterMock
		FakeClient     client.Client
		FakeMPM        *marketplace.MarketplaceInterfaceMock
	}{
		{
			Name:           "test phase in progress when che cluster is missing",
			ExpectedStatus: v1alpha1.PhaseInProgress,
			Installation: &v1alpha1.Installation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "installation",
					APIVersion: "integreatly.org/v1alpha1",
				},
			},
			FakeClient: fakeclient.NewFakeClientWithScheme(buildScheme(), &testKeycloakRealm),
			FakeConfig: basicConfigMock(),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			testReconciler, err := NewReconciler(
				scenario.FakeConfig,
				scenario.Installation,
				scenario.FakeMPM,
			)
			if err != nil && err.Error() != scenario.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, scenario.ExpectedError)
			}

			status, err := testReconciler.reconcileCheCluster(context.TODO(), scenario.Installation, scenario.FakeClient)
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
		ExpectedStatus      v1alpha1.StatusPhase
		ExpectedError       string
		ExpectedCreateError string
		Installation        *v1alpha1.Installation
		FakeConfig          *config.ConfigReadWriterMock
		FakeClient          client.Client
		FakeMPM             *marketplace.MarketplaceInterfaceMock
	}{
		{
			Name:           "test creating components phase missing cluster expect p",
			ExpectedStatus: v1alpha1.PhaseFailed,
			Installation: &v1alpha1.Installation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "installation",
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
				},
				Status: v1alpha1.InstallationStatus{
					Stages: map[v1alpha1.StageName]*v1alpha1.InstallationStageStatus{
						"codeready-stage": {
							Name: "codeready-stage",
							Products: map[v1alpha1.ProductName]*v1alpha1.InstallationProductStatus{
								v1alpha1.ProductCodeReadyWorkspaces: {
									Name:   v1alpha1.ProductCodeReadyWorkspaces,
									Status: v1alpha1.PhaseCreatingSubscription,
								},
							},
						},
					},
				},
			},
			ExpectedError: "could not retrieve checluster for keycloak client update: checlusters.org.eclipse.che \"integreatly-cluster\" not found",
			FakeClient:    fakeclient.NewFakeClientWithScheme(buildScheme(), &testKeycloakRealm),
			FakeConfig:    basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, os operatorsv1.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {

					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient client.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlanList, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlanList{}, &operatorsv1alpha1.Subscription{}, nil
				},
			},
		},
		{
			Name:           "test creating components returns in progress",
			ExpectedStatus: v1alpha1.PhaseInProgress,
			Installation: &v1alpha1.Installation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "installation",
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
				},
				Status: v1alpha1.InstallationStatus{
					Stages: map[v1alpha1.StageName]*v1alpha1.InstallationStageStatus{
						"codeready-stage": {
							Name: "codeready-stage",
							Products: map[v1alpha1.ProductName]*v1alpha1.InstallationProductStatus{
								v1alpha1.ProductCodeReadyWorkspaces: {
									Name:   v1alpha1.ProductCodeReadyWorkspaces,
									Status: v1alpha1.PhaseCreatingSubscription,
								},
							},
						},
					},
				},
			},
			FakeClient: fakeclient.NewFakeClientWithScheme(buildScheme(), &testKeycloakRealm, &testCheCluster),
			FakeConfig: basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, os operatorsv1.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {

					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient client.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlanList, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlanList{}, &operatorsv1alpha1.Subscription{}, nil
				},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			testReconciler, err := NewReconciler(
				scenario.FakeConfig,
				scenario.Installation,
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
		ExpectedStatus v1alpha1.StatusPhase
		ExpectedError  string
		ExpectError    bool
		Installation   *v1alpha1.Installation
		FakeConfig     *config.ConfigReadWriterMock
		FakeClient     client.Client
		FakeMPM        *marketplace.MarketplaceInterfaceMock
	}{
		{
			Name:           "test che cluster creating returns phase in progress",
			ExpectedStatus: v1alpha1.PhaseInProgress,
			Installation: &v1alpha1.Installation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "installation",
					APIVersion: "integreatly.org/v1alpha1",
				},
			},
			FakeClient: fakeclient.NewFakeClientWithScheme(buildScheme(), &testKeycloakRealm, &testCheCluster),
			FakeConfig: basicConfigMock(),
		},
		{
			Name:           "test che cluster create error returns phase failed",
			ExpectedStatus: v1alpha1.PhaseFailed,
			Installation: &v1alpha1.Installation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "installation",
					APIVersion: "integreatly.org/v1alpha1",
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
		},
		{
			Name:           "test che cluster available returns phase complete",
			ExpectedStatus: v1alpha1.PhaseCompleted,
			Installation: &v1alpha1.Installation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "installation",
					APIVersion: "integreatly.org/v1alpha1",
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
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			testReconciler, err := NewReconciler(
				scenario.FakeConfig,
				scenario.Installation,
				scenario.FakeMPM,
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
	installation := &v1alpha1.Installation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "installation",
			Namespace: defaultInstallationNamespace,
			UID:       types.UID("xyz"),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "installation",
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
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
	}

	scenarios := []struct {
		Name                string
		ExpectedStatus      v1alpha1.StatusPhase
		ExpectedError       string
		ExpectedCreateError string
		Installation        *v1alpha1.Installation
		FakeConfig          *config.ConfigReadWriterMock
		FakeClient          client.Client
		FakeMPM             *marketplace.MarketplaceInterfaceMock
		ValidateCallCounts  func(mockConfig *config.ConfigReadWriterMock, mockMPM *marketplace.MarketplaceInterfaceMock, t *testing.T)
		Product             *v1alpha1.InstallationProductStatus
	}{
		{
			Name:           "test successful installation without errors",
			ExpectedStatus: v1alpha1.PhaseCompleted,
			Installation:   installation,
			FakeClient:     fakeclient.NewFakeClientWithScheme(buildScheme(), &testKeycloakRealm, dep, ns, cluster, installation),
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
				InstallOperatorFunc: func(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, os operatorsv1.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {

					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient client.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlanList, subscription *operatorsv1alpha1.Subscription, e error) {
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
			Product: &v1alpha1.InstallationProductStatus{},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			testReconciler, err := NewReconciler(
				scenario.FakeConfig,
				scenario.Installation,
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

// Generate a fake k8s client with a fuse custom resource in a specific phase
func getFakeClient(scheme *runtime.Scheme, objs []runtime.Object) *moqclient.SigsClientInterfaceMock {
	sigsFakeClient := moqclient.NewSigsClientMoqWithScheme(scheme, objs...)
	sigsFakeClient.CreateFunc = func(ctx context.Context, obj runtime.Object) error {
		switch obj := obj.(type) {
		case *chev1.CheCluster:
			obj.Status.CheURL = "https://test.com"
			return sigsFakeClient.GetSigsClient().Create(ctx, obj)
		}

		return sigsFakeClient.GetSigsClient().Create(ctx, obj)
	}

	return sigsFakeClient
}
