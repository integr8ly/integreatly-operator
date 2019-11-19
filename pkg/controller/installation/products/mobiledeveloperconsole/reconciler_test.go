package mobiledeveloperconsole

import (
	"bytes"
	"context"
	"testing"

	routev1 "github.com/openshift/api/route/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"

	mdc "github.com/aerogear/mobile-developer-console-operator/pkg/apis/mdc/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	appsv1 "github.com/openshift/api/apps/v1"
	v1 "github.com/openshift/api/apps/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	v1Route "github.com/openshift/api/route/v1"
	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	defaultOperatorNamespace = "integreatly-mdc"
	oauthClientName          = "oauth-client-secrets"
)

func basicConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		ReadMobileDeveloperConsoleFunc: func() (ready *config.MobileDeveloperConsole, e error) {
			return config.NewMobileDeveloperConsole(config.ProductConfig{}), nil
		},
		WriteConfigFunc: func(config config.ConfigReadable) error {
			return nil
		},
		GetOauthClientsSecretNameFunc: func() string {
			return oauthClientName
		},
		GetOperatorNamespaceFunc: func() string {
			return defaultOperatorNamespace
		},
	}
}

func basicInstallation() *v1alpha1.Installation {
	return &v1alpha1.Installation{
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
}

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	err = operatorsv1alpha1.AddToScheme(scheme)
	err = marketplacev1.SchemeBuilder.AddToScheme(scheme)
	err = corev1.SchemeBuilder.AddToScheme(scheme)
	err = coreosv1.SchemeBuilder.AddToScheme(scheme)
	err = mdc.SchemeBuilder.AddToScheme(scheme)
	err = v1Route.AddToScheme(scheme)
	err = appsv1.AddToScheme(scheme)
	err = oauthv1.AddToScheme(scheme)
	return scheme, err
}

func TestReconciler_config(t *testing.T) {
	cases := []struct {
		Name           string
		ExpectError    bool
		ExpectedError  string
		ExpectedStatus v1alpha1.StatusPhase
		FakeConfig     *config.ConfigReadWriterMock
		FakeClient     pkgclient.Client
		FakeMPM        *marketplace.MarketplaceInterfaceMock
		Installation   *v1alpha1.Installation
	}{
		{
			Name:           "test error on failed config",
			ExpectError:    true,
			ExpectedError:  "could not read mobile developer console: could not read mdc config",
			ExpectedStatus: v1alpha1.PhaseFailed,
			Installation:   &v1alpha1.Installation{},
			FakeClient:     fakeclient.NewFakeClient(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadMobileDeveloperConsoleFunc: func() (ready *config.MobileDeveloperConsole, e error) {
					return nil, errors.New("could not read mdc config")
				},
			},
		},
		{
			Name:         "test namespace is set without fail",
			Installation: &v1alpha1.Installation{},
			FakeClient:   fakeclient.NewFakeClient(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadMobileDeveloperConsoleFunc: func() (ready *config.MobileDeveloperConsole, e error) {
					return config.NewMobileDeveloperConsole(config.ProductConfig{
						"NAMESPACE": "",
					}), nil
				},
			},
		},
		{
			Name:           "test subscription phase with error from mpm",
			ExpectedStatus: v1alpha1.PhaseFailed,
			ExpectError:    true,
			Installation:   &v1alpha1.Installation{},
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, os marketplacev1.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {
					return errors.New("dummy error")
				},
			},
			FakeClient: fakeclient.NewFakeClient(),
			FakeConfig: basicConfigMock(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM)
			if err != nil && err.Error() != tc.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
			}
			if err == nil && tc.ExpectedError != "" {
				t.Fatalf("expected error '%v' and got nil", tc.ExpectedError)
			}
		})
	}
}

func TestReconciler_reconcileComponents(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		Name           string
		FakeClient     pkgclient.Client
		FakeConfig     *config.ConfigReadWriterMock
		Installation   *v1alpha1.Installation
		ExpectError    bool
		ExpectedStatus v1alpha1.StatusPhase
		FakeMPM        *marketplace.MarketplaceInterfaceMock
	}{
		{
			Name:           "Test reconcile custom resource returns in progress when successful created",
			FakeClient:     fakeclient.NewFakeClientWithScheme(scheme, getOauthClientSecret(), getOperatorDC(), getRoute()),
			FakeConfig:     basicConfigMock(),
			Installation:   &v1alpha1.Installation{},
			ExpectedStatus: v1alpha1.PhaseCompleted,
		},
		{
			Name: "Test reconcile custom resource returns failed when cr status is failed",
			FakeClient: &moqclient.SigsClientInterfaceMock{
				CreateFunc: func(ctx context.Context, obj runtime.Object, opts ...pkgclient.CreateOption) error {
					return errors.New("dummy create error")
				},
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return k8serr.NewNotFound(schema.GroupResource{
						Group:    mdc.SchemeBuilder.GroupVersion.Group,
						Resource: "MobileDeveloperConsole",
					}, key.Name)
				},
			},
			FakeConfig:     basicConfigMock(),
			Installation:   &v1alpha1.Installation{},
			ExpectedStatus: v1alpha1.PhaseFailed,
			ExpectError:    true,
		},
		{
			Name:           "Test reconcile custom resource returns in progress when DC is not available",
			FakeClient:     fakeclient.NewFakeClientWithScheme(scheme, getOauthClientSecret(), getRoute()),
			FakeConfig:     basicConfigMock(),
			Installation:   &v1alpha1.Installation{},
			ExpectedStatus: v1alpha1.PhaseInProgress,
			ExpectError:    false,
		},
		{
			Name:           "Test reconcile custom resource returns in progress when route CR is not available",
			FakeClient:     fakeclient.NewFakeClientWithScheme(scheme, getOauthClientSecret(), getOperatorDC()),
			FakeConfig:     basicConfigMock(),
			Installation:   &v1alpha1.Installation{},
			ExpectedStatus: v1alpha1.PhaseInProgress,
			ExpectError:    false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM)
			if err != nil {
				t.Fatal("unexpected err ", err)
			}

			phase, err := reconciler.reconcileComponents(context.TODO(), tc.FakeClient, tc.Installation)
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

func TestReconciler_handleProgress(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	mdcCrUp := &mdc.MobileDeveloperConsole{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defaultInstallationNamespace,
			Name:      resourceName,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: mdc.SchemeGroupVersion.String(),
			Kind:       "MobileDeveloperConsole",
		}, Status: mdc.MobileDeveloperConsoleStatus{
			Phase: mdc.PhaseComplete,
		},
	}

	mdcCrProvisioning := &mdc.MobileDeveloperConsole{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defaultInstallationNamespace,
			Name:      resourceName,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: mdc.SchemeGroupVersion.String(),
			Kind:       "MobileDeveloperConsole",
		}, Status: mdc.MobileDeveloperConsoleStatus{
			Phase: mdc.PhaseProvision,
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
	}{
		{
			Name:           "test failure to get mdc install",
			ExpectedStatus: v1alpha1.PhaseFailed,
			ExpectedError:  "failed to get mdc cr while reconciling custom resource",
			ExpectError:    true,
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme),
			FakeConfig:     basicConfigMock(),
			Installation:   &v1alpha1.Installation{},
		},
		{
			Name:           "test mdc cr provisioning",
			ExpectedStatus: v1alpha1.PhaseInProgress,
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, mdcCrProvisioning),
			FakeConfig:     basicConfigMock(),
			Installation:   &v1alpha1.Installation{},
		},
		{
			Name:           "test mdc cr complete",
			ExpectedStatus: v1alpha1.PhaseCompleted,
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, mdcCrUp),
			FakeConfig:     basicConfigMock(),
			Installation:   &v1alpha1.Installation{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM)
			if err != nil && err.Error() != tc.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
			}

			status, err := reconciler.handleProgress(context.TODO(), tc.FakeClient)
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

	// initialise runtime objects
	objs := []runtime.Object{}
	objs = append(objs, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultInstallationNamespace,
			Labels: map[string]string{
				resources.OwnerLabelKey: string(basicInstallation().GetUID()),
			},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}, &v1Route.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Route",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defaultInstallationNamespace,
			Name:      resourceName,
		},
	}, &mdc.MobileDeveloperConsole{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defaultInstallationNamespace,
			Name:      resourceName,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: mdc.SchemeGroupVersion.String(),
			Kind:       "MobileDeveloperConsole",
		}, Status: mdc.MobileDeveloperConsoleStatus{
			Phase: mdc.PhaseComplete,
		},
	}, getOauthClientSecret(), getOperatorDC(), getRoute(), basicInstallation())

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
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, objs...),
			FakeConfig:     basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, os marketplacev1.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {
					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient client.Client, subName string, ns string) (plan *operatorsv1alpha1.InstallPlanList, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlanList{
							Items: []operatorsv1alpha1.InstallPlan{
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "mdc-install-plan",
									},
									Status: operatorsv1alpha1.InstallPlanStatus{
										Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
									},
								},
							},
						}, &operatorsv1alpha1.Subscription{
							Status: operatorsv1alpha1.SubscriptionStatus{
								Install: &operatorsv1alpha1.InstallPlanReference{
									Name: "mdc-install-plan",
								},
							},
						}, nil
				},
			},
			Installation: basicInstallation(),
			Product:      &v1alpha1.InstallationProductStatus{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM)
			if err != nil && err.Error() != tc.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
			}

			status, err := reconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.FakeClient)
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

func TestReconciler_testPhases(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		Name           string
		ExpectedStatus v1alpha1.StatusPhase
		FakeConfig     *config.ConfigReadWriterMock
		FakeClient     pkgclient.Client
		FakeMPM        *marketplace.MarketplaceInterfaceMock
		Installation   *v1alpha1.Installation
		Product        *v1alpha1.InstallationProductStatus
	}{
		{
			Name:           "test namespace terminating returns phase in progress",
			ExpectedStatus: v1alpha1.PhaseInProgress,
			Installation:   basicInstallation(),
			FakeClient: moqclient.NewSigsClientMoqWithScheme(scheme, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: defaultInstallationNamespace,
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceTerminating,
				},
			}, basicInstallation()),
			FakeConfig: basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, os marketplacev1.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {
					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient client.Client, subName string, ns string) (plan *operatorsv1alpha1.InstallPlanList, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlanList{}, &operatorsv1alpha1.Subscription{}, nil
				},
			},
			Product: &v1alpha1.InstallationProductStatus{},
		},
		{
			Name:           "test subscription creating returns phase in progress",
			ExpectedStatus: v1alpha1.PhaseInProgress,
			Installation:   basicInstallation(),
			FakeClient:     fakeclient.NewFakeClientWithScheme(scheme, basicInstallation()),
			FakeConfig:     basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, os marketplacev1.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {
					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient client.Client, subName string, ns string) (plan *operatorsv1alpha1.InstallPlanList, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlanList{}, &operatorsv1alpha1.Subscription{}, nil
				},
			},
			Product: &v1alpha1.InstallationProductStatus{},
		},
		{
			Name:           "test components creating returns phase in progress",
			ExpectedStatus: v1alpha1.PhaseInProgress,
			Installation:   basicInstallation(),
			FakeClient:     fakeclient.NewFakeClientWithScheme(scheme, basicInstallation(), getOauthClientSecret()),
			FakeConfig:     basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, os marketplacev1.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {
					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient client.Client, subName string, ns string) (plan *operatorsv1alpha1.InstallPlanList, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlanList{
							Items: []operatorsv1alpha1.InstallPlan{
								{
									ObjectMeta: metav1.ObjectMeta{
										Name:      "mdc-install-plan",
										Namespace: defaultInstallationNamespace,
									},
									Status: operatorsv1alpha1.InstallPlanStatus{
										Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
									},
								},
							},
						}, &operatorsv1alpha1.Subscription{
							Status: operatorsv1alpha1.SubscriptionStatus{
								Install: &operatorsv1alpha1.InstallPlanReference{
									Name: "mdc-install-plan",
								},
							},
						}, nil
				},
			},
			Product: &v1alpha1.InstallationProductStatus{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM)
			if err != nil {
				t.Fatalf("unexpected error : '%v'", err)
			}

			status, err := reconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.FakeClient)
			if err != nil {
				t.Fatalf("expected no error but got one: %v", err)
			}
			if status != tc.ExpectedStatus {
				t.Fatalf("Expected status: '%v', got: '%v'", tc.ExpectedStatus, status)
			}
		})
	}
}

func getOauthClientSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oauth-client-secrets",
			Namespace: defaultOperatorNamespace,
		},
		Data: map[string][]byte{
			"mdc": bytes.NewBufferString("test").Bytes(),
		},
	}
}

func getOperatorDC() *appsv1.DeploymentConfig {
	return &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defaultInstallationNamespace,
			Name:      resourceName,
		},
		Spec: v1.DeploymentConfigSpec{
			Template: &corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Env: []corev1.EnvVar{
								corev1.EnvVar{
									Name:  "OPENSHIFT_HOST",
									Value: "URL",
								},
							},
						},
					},
				},
			},
		},
	}
}

func getRoute() *routev1.Route {
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeResourceName,
			Namespace: defaultInstallationNamespace,
		},
	}
}
