package fuse

import (
	"context"
	"errors"
	"fmt"
	"testing"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"

	appsv1 "k8s.io/api/apps/v1"

	crov1alpha1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	croTypes "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"

	syndesisv1beta1 "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1beta1"

	threescalev1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/resources"

	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"

	projectv1 "github.com/openshift/api/project/v1"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"

	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func basicConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		ReadFuseFunc: func() (ready *config.Fuse, e error) {
			return config.NewFuse(config.ProductConfig{}), nil
		},
		ReadRHSSOFunc: func() (*config.RHSSO, error) {
			return config.NewRHSSO(config.ProductConfig{
				"NAMESPACE": "fuse",
				"HOST":      "fuse.openshift-cluster.com",
			}), nil
		},
		ReadThreeScaleFunc: func() (*config.ThreeScale, error) {
			return config.NewThreeScale(config.ProductConfig{
				"NAMESPACE": "threescale",
				"HOST":      "threescale.openshift-cluster.com",
			}), nil
		},
		WriteConfigFunc: func(config config.ConfigReadable) error {
			return nil
		},
		ReadMonitoringFunc: func() (*config.Monitoring, error) {
			return config.NewMonitoring(config.ProductConfig{
				"NAMESPACE": "middleware-monitoring",
			}), nil
		},
	}
}

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := threescalev1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := keycloak.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := operatorsv1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := corev1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := coreosv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := syndesisv1beta1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := routev1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := usersv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := rbacv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := projectv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := crov1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := monitoringv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	return scheme, nil
}

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
}

func TestReconciler_config(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

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
	}{
		{
			Name:           "test error on failed config",
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			ExpectError:    true,
			ExpectedError:  "could not retrieve fuse config: could not read fuse config",
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeClient:     fakeclient.NewFakeClient(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadFuseFunc: func() (ready *config.Fuse, e error) {
					return nil, errors.New("could not read fuse config")
				},
			},
			Product:  &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder: setupRecorder(),
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
			FakeClient: moqclient.NewSigsClientMoqWithScheme(scheme, &integreatlyv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "installation",
					Namespace: defaultInstallationNamespace,
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "installation",
					APIVersion: integreatlyv1alpha1.GroupVersion.String(),
				},
			}),
			FakeConfig: basicConfigMock(),
			Product:    &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:   setupRecorder(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			testReconciler, err := NewReconciler(
				tc.FakeConfig,
				tc.Installation,
				tc.FakeMPM,
				tc.Recorder,
				getLogger(),
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

func TestReconciler_reconcileCustomResource(t *testing.T) {
	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "syndesis",
			Namespace: defaultInstallationNamespace,
		},
	}

	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "syndesis-global-config",
			Namespace: defaultInstallationNamespace,
		},
	}

	testRHMINamespace := "test"
	croPostgresSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.FusePostgresPrefix,
			Namespace: testRHMINamespace,
		},
		Data: map[string][]byte{},
		Type: corev1.SecretTypeOpaque,
	}
	cases := []struct {
		Name           string
		FakeClient     k8sclient.Client
		FakeConfig     *config.ConfigReadWriterMock
		Installation   *integreatlyv1alpha1.RHMI
		ExpectError    bool
		ExpectedStatus integreatlyv1alpha1.StatusPhase
		FakeMPM        *marketplace.MarketplaceInterfaceMock
		Recorder       record.EventRecorder
	}{
		{
			Name:       "Test reconcile custom resource returns in progress when successful created",
			FakeClient: fakeclient.NewFakeClientWithScheme(scheme, secret, croPostgresSecret),
			FakeConfig: basicConfigMock(),
			Installation: &integreatlyv1alpha1.RHMI{
				TypeMeta: metav1.TypeMeta{
					Kind:       "RHMI",
					APIVersion: integreatlyv1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testRHMINamespace,
				},
			},
			ExpectedStatus: integreatlyv1alpha1.PhaseInProgress,
			Recorder:       setupRecorder(),
		},
		{
			Name:       "Test reconcile custom resource returns failed when cr status is failed",
			FakeClient: fakeclient.NewFakeClientWithScheme(scheme, getFuseCr(syndesisv1beta1.SyndesisPhaseStartupFailed), croPostgresSecret),
			FakeConfig: basicConfigMock(),
			Installation: &integreatlyv1alpha1.RHMI{
				TypeMeta: metav1.TypeMeta{
					Kind:       "RHMI",
					APIVersion: integreatlyv1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testRHMINamespace,
				},
			},
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			ExpectError:    true,
			Recorder:       setupRecorder(),
		},
		{
			Name:       "Test reconcile custom resource returns phase complete when cr status is installed",
			FakeClient: fakeclient.NewFakeClientWithScheme(scheme, getFuseCr(syndesisv1beta1.SyndesisPhaseInstalled), route, secret, croPostgresSecret),
			FakeConfig: basicConfigMock(),
			Installation: &integreatlyv1alpha1.RHMI{
				TypeMeta: metav1.TypeMeta{
					Kind:       "RHMI",
					APIVersion: integreatlyv1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testRHMINamespace,
				},
			},
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Recorder:       setupRecorder(),
		},
		{
			Name:       "Test reconcile custom resource returns phase in progress when cr status is installing",
			FakeClient: fakeclient.NewFakeClientWithScheme(scheme, getFuseCr(syndesisv1beta1.SyndesisPhaseInstalling), secret, croPostgresSecret),
			FakeConfig: basicConfigMock(),
			Installation: &integreatlyv1alpha1.RHMI{
				TypeMeta: metav1.TypeMeta{
					Kind:       "RHMI",
					APIVersion: integreatlyv1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testRHMINamespace,
				},
			},
			ExpectedStatus: integreatlyv1alpha1.PhaseInProgress,
			Recorder:       setupRecorder(),
		},
		{
			Name: "Test reconcile custom resource returns failed on unsuccessful create",
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return errors.New("dummy get error")
				},
				CreateFunc: func(ctx context.Context, obj runtime.Object, opts ...k8sclient.CreateOption) error {
					return errors.New("dummy create error")
				},
			},
			FakeConfig: basicConfigMock(),
			Installation: &integreatlyv1alpha1.RHMI{
				TypeMeta: metav1.TypeMeta{
					Kind:       "RHMI",
					APIVersion: integreatlyv1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testRHMINamespace,
				},
			},
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			ExpectError:    true,
			Recorder:       setupRecorder(),
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(
				tc.FakeConfig,
				tc.Installation,
				tc.FakeMPM,
				tc.Recorder,
				getLogger(),
			)
			if err != nil {
				t.Fatal("unexpected err ", err)
			}
			phase, err := reconciler.reconcileCustomResource(context.TODO(), tc.Installation, tc.FakeClient)
			if tc.ExpectError && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectError && err != nil {
				t.Fatal("expected no error but got one ", err)
			}
			if tc.ExpectedStatus != phase {
				t.Fatal("expected phase ", tc.ExpectedStatus, " but got ", phase)
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
			Name:      "installation",
			Namespace: defaultInstallationNamespace,
			UID:       types.UID("xyz"),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "RHMI",
			APIVersion: integreatlyv1alpha1.GroupVersion.String(),
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

	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "syndesis",
			Namespace: defaultInstallationNamespace,
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "syndesis-global-config",
			Namespace: defaultInstallationNamespace,
		},
	}

	pullSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      integreatlyv1alpha1.DefaultOriginPullSecretName,
			Namespace: integreatlyv1alpha1.DefaultOriginPullSecretNamespace,
		},
		Data: map[string][]byte{
			"test": {'t', 'e', 's', 't'},
		},
	}

	test1User := &usersv1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test1",
		},
	}
	rhmiDevelopersGroup := &usersv1.Group{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rhmi-developers",
		},
		Users: []string{
			test1User.Name,
		},
	}
	operatorDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "syndesis-operator",
			Namespace: "fuse-operator",
		},
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
	}
	//completed postgres that points at the secret croPostgresSecret
	postgresName := fmt.Sprintf("%s%s", constants.FusePostgresPrefix, installation.Name)
	croPostgres := &crov1alpha1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      postgresName,
			Namespace: installation.Namespace,
		},
		Status: crov1alpha1.PostgresStatus{
			Phase: croTypes.PhaseComplete,
			SecretRef: &croTypes.SecretRef{
				Name:      postgresName,
				Namespace: installation.Namespace,
			},
		},
	}

	//secret created by the cloud resource operator postgres reconciler
	croPostgresSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      postgresName,
			Namespace: installation.Namespace,
		},
		Data: map[string][]byte{},
		Type: corev1.SecretTypeOpaque,
	}

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
	}{
		{
			Name:           "test successful reconcile",
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			FakeClient:     fakeclient.NewFakeClientWithScheme(scheme, getFuseCr(syndesisv1beta1.SyndesisPhaseInstalled), ns, operatorNS, route, secret, test1User, rhmiDevelopersGroup, pullSecret, installation, operatorDeployment, croPostgres, croPostgresSecret),
			FakeConfig:     basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlan{
							ObjectMeta: metav1.ObjectMeta{
								Name: "fuse-install-plan",
							},
							Status: operatorsv1alpha1.InstallPlanStatus{
								Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
							},
						}, &operatorsv1alpha1.Subscription{
							Status: operatorsv1alpha1.SubscriptionStatus{
								Install: &operatorsv1alpha1.InstallPlanReference{
									Name: "fuse-install-plan",
								},
							},
						}, nil
				},
			},
			Installation: installation,
			Product:      &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:     setupRecorder(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			testReconciler, err := NewReconciler(
				tc.FakeConfig,
				tc.Installation,
				tc.FakeMPM,
				tc.Recorder,
				getLogger(),
			)
			if err != nil && err.Error() != tc.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
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

			pvccr := &v1.PersistentVolumeClaim{}
			err = tc.FakeClient.Get(context.TODO(), k8sclient.ObjectKey{Name: syndesisPrometheus, Namespace: defaultInstallationNamespace}, pvccr)
			if err != nil {
				t.Fatalf("expected no error but got one: %v", err)
			}
			if pvccr.Spec.Resources.Requests[v1.ResourceStorage] != resource.MustParse(syndesisPrometheusPVC) {
				t.Fatalf("syndesis-prometheus pvc not set to 10Gi")
			}
		})
	}
}

// Return a fuse custom resource in a specific phase
func getFuseCr(phase syndesisv1beta1.SyndesisPhase) *syndesisv1beta1.Syndesis {
	intLimit := -1
	return &syndesisv1beta1.Syndesis{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defaultInstallationNamespace,
			Name:      "integreatly",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Syndesis",
			APIVersion: syndesisv1beta1.SchemeGroupVersion.String(),
		},
		Spec: syndesisv1beta1.SyndesisSpec{
			Components: syndesisv1beta1.ComponentsSpec{
				Server: syndesisv1beta1.ServerConfiguration{
					Features: syndesisv1beta1.ServerFeatures{
						IntegrationLimit:       intLimit,
						ManagementURLFor3scale: "https://3scale-admin.dummmy",
					},
				},
			},
		},
		Status: syndesisv1beta1.SyndesisStatus{
			Phase: phase,
		},
	}
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.ProductDataSync})
}
