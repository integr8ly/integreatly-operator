package fuse

import (
	"context"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/resources"

	v1 "github.com/openshift/api/apps/v1"

	threescalev1 "github.com/integr8ly/integreatly-operator/pkg/apis/3scale/v1alpha1"
	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	usersv1 "github.com/openshift/api/user/v1"
	"github.com/pkg/errors"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	routev1 "github.com/openshift/api/route/v1"
	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"

	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	syn "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
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
		WriteConfigFunc: func(config config.ConfigReadable) error {
			return nil
		},
	}
}

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := threescalev1.SchemeBuilder.AddToScheme(scheme)
	err = aerogearv1.SchemeBuilder.AddToScheme(scheme)
	err = v1alpha1.SchemeBuilder.AddToScheme(scheme)
	err = operatorsv1alpha1.AddToScheme(scheme)
	err = marketplacev1.SchemeBuilder.AddToScheme(scheme)
	err = corev1.SchemeBuilder.AddToScheme(scheme)
	err = coreosv1.SchemeBuilder.AddToScheme(scheme)
	err = syn.SchemeBuilder.AddToScheme(scheme)
	err = routev1.AddToScheme(scheme)
	err = v1.AddToScheme(scheme)
	err = usersv1.AddToScheme(scheme)
	err = rbacv1.SchemeBuilder.AddToScheme(scheme)
	return scheme, err
}

func TestReconciler_config(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

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
			ExpectedError:  "could not retrieve fuse config: could not read fuse config",
			Installation:   &v1alpha1.Installation{},
			FakeClient:     fakeclient.NewFakeClient(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadFuseFunc: func() (ready *config.Fuse, e error) {
					return nil, errors.New("could not read fuse config")
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
				InstallOperatorFunc: func(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {

					return errors.New("dummy error")
				},
			},
			FakeClient: moqclient.NewSigsClientMoqWithScheme(scheme, &v1alpha1.Installation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "installation",
					Namespace: defaultInstallationNamespace,
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "installation",
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
				},
			}),
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

	cases := []struct {
		Name           string
		FakeClient     client.Client
		FakeConfig     *config.ConfigReadWriterMock
		Installation   *v1alpha1.Installation
		ExpectError    bool
		ExpectedStatus v1alpha1.StatusPhase
		FakeMPM        *marketplace.MarketplaceInterfaceMock
	}{
		{
			Name:       "Test reconcile custom resource returns in progress when successful created",
			FakeClient: fakeclient.NewFakeClientWithScheme(scheme, secret),
			FakeConfig: basicConfigMock(),
			Installation: &v1alpha1.Installation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "installation",
					APIVersion: "integreatly.org/v1alpha1",
				},
			},
			ExpectedStatus: v1alpha1.PhaseInProgress,
		},
		{
			Name:       "Test reconcile custom resource returns failed when cr status is failed",
			FakeClient: fakeclient.NewFakeClientWithScheme(scheme, getFuseCr(syn.SyndesisPhaseStartupFailed)),
			FakeConfig: basicConfigMock(),
			Installation: &v1alpha1.Installation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "installation",
					APIVersion: "integreatly.org/v1alpha1",
				},
			},
			ExpectedStatus: v1alpha1.PhaseFailed,
			ExpectError:    true,
		},
		{
			Name:       "Test reconcile custom resource returns phase complete when cr status is installed",
			FakeClient: fakeclient.NewFakeClientWithScheme(scheme, getFuseCr(syn.SyndesisPhaseInstalled), route, secret),
			FakeConfig: basicConfigMock(),
			Installation: &v1alpha1.Installation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "installation",
					APIVersion: "integreatly.org/v1alpha1",
				},
			},
			ExpectedStatus: v1alpha1.PhaseCompleted,
		},
		{
			Name:       "Test reconcile custom resource returns phase in progress when cr status is installing",
			FakeClient: fakeclient.NewFakeClientWithScheme(scheme, getFuseCr(syn.SyndesisPhaseInstalling), secret),
			FakeConfig: basicConfigMock(),
			Installation: &v1alpha1.Installation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "installation",
					APIVersion: "integreatly.org/v1alpha1",
				},
			},
			ExpectedStatus: v1alpha1.PhaseInProgress,
		},
		{
			Name: "Test reconcile custom resource returns failed on unsuccessful create",
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return errors.New("dummy get error")
				},
				CreateFunc: func(ctx context.Context, obj runtime.Object, opts ...pkgclient.CreateOption) error {
					return errors.New("dummy create error")
				},
			},
			FakeConfig: basicConfigMock(),
			Installation: &v1alpha1.Installation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "installation",
					APIVersion: "integreatly.org/v1alpha1",
				},
			},
			ExpectedStatus: v1alpha1.PhaseFailed,
			ExpectError:    true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(
				tc.FakeConfig,
				tc.Installation,
				tc.FakeMPM,
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
			Name:      resources.DefaultOriginPullSecretName,
			Namespace: resources.DefaultOriginPullSecretNamespace,
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

	cases := []struct {
		Name           string
		ExpectError    bool
		ExpectedStatus v1alpha1.StatusPhase
		ExpectedError  string
		FakeConfig     *config.ConfigReadWriterMock
		FakeClient     client.Client
		FakeMPM        *marketplace.MarketplaceInterfaceMock
		Installation   *v1alpha1.Installation
		Product        *v1alpha1.InstallationProductStatus
	}{
		{
			Name:           "test successful reconcile",
			ExpectedStatus: v1alpha1.PhaseCompleted,
			FakeClient:     fakeclient.NewFakeClientWithScheme(scheme, getFuseCr(syn.SyndesisPhaseInstalled), getFuseDC(ns.Name), ns, route, secret, test1User, rhmiDevelopersGroup, pullSecret, installation),
			FakeConfig:     basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {

					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient client.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlanList, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlanList{
							Items: []operatorsv1alpha1.InstallPlan{
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "fuse-install-plan",
									},
									Status: operatorsv1alpha1.InstallPlanStatus{
										Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
									},
								},
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
			Product:      &v1alpha1.InstallationProductStatus{},
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

// Return a fuse custom resource in a specific phase
func getFuseCr(phase syn.SyndesisPhase) *syn.Syndesis {
	intLimit := -1
	return &syn.Syndesis{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defaultInstallationNamespace,
			Name:      "integreatly",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Syndesis",
			APIVersion: syn.SchemeGroupVersion.String(),
		},
		Spec: syn.SyndesisSpec{
			Integration: syn.IntegrationSpec{
				Limit: &intLimit,
			},
			Components: syn.ComponentsSpec{
				Server: syn.ServerConfiguration{
					Features: syn.ServerFeatures{
						ExposeVia3Scale: true,
					},
				},
			},
		},
		Status: syn.SyndesisStatus{
			Phase: phase,
		},
	}
}

func getFuseDC(ns string) *v1.DeploymentConfig {
	return &v1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      "syndesis-oauthproxy",
		},
		Spec: v1.DeploymentConfigSpec{
			Template: &corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Args: []string{"--openshift-sar={}"},
						},
					},
				},
			},
		},
	}
}
