package amqonline

import (
	"context"
	"errors"
	"fmt"
	"testing"

	chev1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	enmassev1 "github.com/enmasseproject/enmasse/pkg/apis/admin/v1beta1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta1"

	enmassev1beta1 "github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta2"
	enmassev1beta2 "github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta2"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	kafkav1 "github.com/integr8ly/integreatly-operator/pkg/apis/kafka.strimzi.io/v1alpha1"
	keycloak "github.com/integr8ly/integreatly-operator/pkg/apis/keycloak/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	monitoring "github.com/integr8ly/integreatly-operator/pkg/apis/monitoring/v1alpha1"
)

func buildScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	chev1.SchemeBuilder.AddToScheme(scheme)
	keycloak.SchemeBuilder.AddToScheme(scheme)
	v1alpha1.SchemeBuilder.AddToScheme(scheme)
	operatorsv1alpha1.AddToScheme(scheme)
	marketplacev1.SchemeBuilder.AddToScheme(scheme)
	kafkav1.SchemeBuilder.AddToScheme(scheme)
	corev1.SchemeBuilder.AddToScheme(scheme)
	enmassev1.SchemeBuilder.AddToScheme(scheme)
	enmassev1beta1.SchemeBuilder.AddToScheme(scheme)
	enmassev1beta2.SchemeBuilder.AddToScheme(scheme)
	rbacv1.SchemeBuilder.AddToScheme(scheme)
	batchv1beta1.SchemeBuilder.AddToScheme(scheme)
	appsv1.SchemeBuilder.AddToScheme(scheme)
	monitoring.SchemeBuilder.AddToScheme(scheme)
	return scheme
}

const (
	defaultNamespace = "amq-online"
)

func basicConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		GetOperatorNamespaceFunc: func() string {
			return "integreatly-operator"
		},
		ReadAMQOnlineFunc: func() (ready *config.AMQOnline, e error) {
			return config.NewAMQOnline(config.ProductConfig{
				"NAMESPACE": defaultNamespace,
			}), nil
		},
		ReadRHSSOFunc: func() (*config.RHSSO, error) {
			return config.NewRHSSO(config.ProductConfig{
				"NAMESPACE": defaultNamespace,
				"URL":       "sso.openshift-cluster.com",
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

func TestReconcile_reconcileAuthServices(t *testing.T) {
	scenarios := []struct {
		Name           string
		Client         client.Client
		FakeConfig     *config.ConfigReadWriterMock
		Installation   *v1alpha1.Installation
		ExpectedStatus v1alpha1.StatusPhase
		AuthServices   []*enmassev1.AuthenticationService
		FakeMPM        *marketplace.MarketplaceInterfaceMock
	}{
		{
			Name:           "Test returns none phase if successfully creating new auth services",
			Client:         fake.NewFakeClientWithScheme(buildScheme()),
			FakeConfig:     basicConfigMock(),
			AuthServices:   GetDefaultAuthServices(defaultNamespace),
			ExpectedStatus: v1alpha1.PhaseCompleted,
		},
		{
			Name:           "Test returns none phase if trying to create existing auth services",
			Client:         fake.NewFakeClientWithScheme(buildScheme(), GetDefaultAuthServices(defaultSubscriptionName)[0]),
			AuthServices:   GetDefaultAuthServices(defaultNamespace),
			FakeConfig:     basicConfigMock(),
			ExpectedStatus: v1alpha1.PhaseCompleted,
		},
	}

	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			r, err := NewReconciler(s.FakeConfig, s.Installation, s.FakeMPM)
			if err != nil {
				t.Fatalf("could not create reconciler %v", err)
			}
			phase, err := r.reconcileAuthServices(context.TODO(), s.Client, s.AuthServices)
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if phase != s.ExpectedStatus {
				t.Fatalf("expected status %s but got %s", s.ExpectedStatus, phase)
			}
		})
	}
}

func TestReconcile_reconcileBrokerConfigs(t *testing.T) {
	scenarios := []struct {
		Name                 string
		Client               client.Client
		FakeConfig           *config.ConfigReadWriterMock
		Installation         *v1alpha1.Installation
		ExpectedStatus       v1alpha1.StatusPhase
		BrokeredInfraConfigs []*v1beta1.BrokeredInfraConfig
		StandardInfraConfigs []*v1beta1.StandardInfraConfig
		FakeMPM              *marketplace.MarketplaceInterfaceMock
	}{
		{
			Name:                 "Test returns none phase if successfully creating new address space plans",
			Client:               fake.NewFakeClientWithScheme(buildScheme()),
			FakeConfig:           basicConfigMock(),
			BrokeredInfraConfigs: GetDefaultBrokeredInfraConfigs(defaultNamespace),
			StandardInfraConfigs: GetDefaultStandardInfraConfigs(defaultNamespace),
			ExpectedStatus:       v1alpha1.PhaseCompleted,
		},
		{
			Name:                 "Test returns none phase if trying to create existing address space plans",
			Client:               fake.NewFakeClientWithScheme(buildScheme(), GetDefaultAuthServices(defaultSubscriptionName)[0]),
			BrokeredInfraConfigs: GetDefaultBrokeredInfraConfigs(defaultNamespace),
			StandardInfraConfigs: GetDefaultStandardInfraConfigs(defaultNamespace),
			FakeConfig:           basicConfigMock(),
			ExpectedStatus:       v1alpha1.PhaseCompleted,
		},
	}

	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			r, err := NewReconciler(s.FakeConfig, s.Installation, s.FakeMPM)
			if err != nil {
				t.Fatalf("could not create reconciler %v", err)
			}
			phase, err := r.reconcileBrokerConfigs(context.TODO(), s.Client, s.BrokeredInfraConfigs, s.StandardInfraConfigs)
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if phase != s.ExpectedStatus {
				t.Fatalf("expected status %s but got %s", s.ExpectedStatus, phase)
			}
		})
	}
}

func TestReconcile_reconcileAddressPlans(t *testing.T) {
	scenarios := []struct {
		Name           string
		Client         client.Client
		FakeConfig     *config.ConfigReadWriterMock
		Installation   *v1alpha1.Installation
		ExpectedStatus v1alpha1.StatusPhase
		AddressPlans   []*v1beta2.AddressPlan
		FakeMPM        *marketplace.MarketplaceInterfaceMock
	}{
		{
			Name:           "Test returns none phase if successfully creating new address space plans",
			Client:         fake.NewFakeClientWithScheme(buildScheme()),
			FakeConfig:     basicConfigMock(),
			AddressPlans:   GetDefaultAddressPlans(defaultNamespace),
			ExpectedStatus: v1alpha1.PhaseCompleted,
		},
		{
			Name:           "Test returns none phase if trying to create existing address space plans",
			Client:         fake.NewFakeClientWithScheme(buildScheme(), GetDefaultAuthServices(defaultSubscriptionName)[0]),
			AddressPlans:   GetDefaultAddressPlans(defaultNamespace),
			FakeConfig:     basicConfigMock(),
			ExpectedStatus: v1alpha1.PhaseCompleted,
		},
	}

	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			r, err := NewReconciler(s.FakeConfig, s.Installation, s.FakeMPM)
			if err != nil {
				t.Fatalf("could not create reconciler %v", err)
			}
			phase, err := r.reconcileAddressPlans(context.TODO(), s.Client, s.AddressPlans)
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if phase != s.ExpectedStatus {
				t.Fatalf("expected status %s but got %s", s.ExpectedStatus, phase)
			}
		})
	}
}

func TestReconcile_reconcileAddressSpacePlans(t *testing.T) {
	scenarios := []struct {
		Name              string
		Client            client.Client
		FakeConfig        *config.ConfigReadWriterMock
		Installation      *v1alpha1.Installation
		ExpectedStatus    v1alpha1.StatusPhase
		AddressSpacePlans []*v1beta2.AddressSpacePlan
		FakeMPM           *marketplace.MarketplaceInterfaceMock
	}{
		{
			Name:              "Test returns none phase if successfully creating new address space plans",
			Client:            fake.NewFakeClientWithScheme(buildScheme()),
			FakeConfig:        basicConfigMock(),
			AddressSpacePlans: GetDefaultAddressSpacePlans(defaultNamespace),
			ExpectedStatus:    v1alpha1.PhaseCompleted,
		},
		{
			Name:              "Test returns none phase if trying to create existing address space plans",
			Client:            fake.NewFakeClientWithScheme(buildScheme(), GetDefaultAuthServices(defaultSubscriptionName)[0]),
			AddressSpacePlans: GetDefaultAddressSpacePlans(defaultNamespace),
			FakeConfig:        basicConfigMock(),
			ExpectedStatus:    v1alpha1.PhaseCompleted,
		},
	}

	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			r, err := NewReconciler(s.FakeConfig, s.Installation, s.FakeMPM)
			if err != nil {
				t.Fatalf("could not create reconciler %v", err)
			}
			phase, err := r.reconcileAddressSpacePlans(context.TODO(), s.Client, s.AddressSpacePlans)
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if phase != s.ExpectedStatus {
				t.Fatalf("expected status %s but got %s", s.ExpectedStatus, phase)
			}
		})
	}
}

func TestReconcile_reconcileConfig(t *testing.T) {
	defaultHost := "https://example.host.com"
	scenarios := []struct {
		Name               string
		Client             client.Client
		ExpectedStatus     v1alpha1.StatusPhase
		FakeConfig         *config.ConfigReadWriterMock
		ExpectError        bool
		ValidateCallCounts func(t *testing.T, cfgMock *config.ConfigReadWriterMock)
	}{
		{
			Name: "Test doesn't set host when the port is not 443",
			Client: fake.NewFakeClientWithScheme(buildScheme(), &enmassev1.ConsoleService{
				ObjectMeta: v1.ObjectMeta{
					Name:      defaultConsoleSvcName,
					Namespace: defaultNamespace,
				},
				Status: enmassev1.ConsoleServiceStatus{
					Host: defaultHost,
					Port: 0,
				},
			}),
			FakeConfig:     basicConfigMock(),
			ExpectedStatus: v1alpha1.PhaseCompleted,
			ValidateCallCounts: func(t *testing.T, cfgMock *config.ConfigReadWriterMock) {
				if len(cfgMock.WriteConfigCalls()) != 0 {
					t.Fatal("config written once or more")
				}
			},
		},
		{
			Name: "Test doesn't set host when the host is undefined or empty",
			Client: fake.NewFakeClientWithScheme(buildScheme(), &enmassev1.ConsoleService{
				ObjectMeta: v1.ObjectMeta{
					Name:      defaultConsoleSvcName,
					Namespace: defaultNamespace,
				},
				Status: enmassev1.ConsoleServiceStatus{
					Host: "",
					Port: 443,
				},
			}),
			FakeConfig:     basicConfigMock(),
			ExpectedStatus: v1alpha1.PhaseCompleted,
			ValidateCallCounts: func(t *testing.T, cfgMock *config.ConfigReadWriterMock) {
				if len(cfgMock.WriteConfigCalls()) != 0 {
					t.Fatal("config written once or more")
				}
			},
		},
		{
			Name: "Test successfully setting host when port and host are defined properly",
			Client: fake.NewFakeClientWithScheme(buildScheme(), &enmassev1.ConsoleService{
				ObjectMeta: v1.ObjectMeta{
					Name:      defaultConsoleSvcName,
					Namespace: defaultNamespace,
				},
				Status: enmassev1.ConsoleServiceStatus{
					Host: defaultHost,
					Port: 443,
				},
			}),
			FakeConfig:     basicConfigMock(),
			ExpectedStatus: v1alpha1.PhaseCompleted,
			ValidateCallCounts: func(t *testing.T, cfgMock *config.ConfigReadWriterMock) {
				expectedHost := fmt.Sprintf("https://%s", defaultHost)
				if len(cfgMock.WriteConfigCalls()) != 1 {
					t.Fatal("config not called once")
				}
				cfg := config.NewAMQOnline(cfgMock.WriteConfigCalls()[0].Config.Read())
				if cfg.GetHost() != expectedHost {
					t.Fatalf("incorrect host, expected %s but got %s", expectedHost, cfg.GetHost())
				}
			},
		},
		{
			Name:           "Test continues when console it not found",
			Client:         fake.NewFakeClientWithScheme(buildScheme()),
			FakeConfig:     basicConfigMock(),
			ExpectedStatus: v1alpha1.PhaseFailed,
			ExpectError:    true,
			ValidateCallCounts: func(t *testing.T, cfgMock *config.ConfigReadWriterMock) {
				if len(cfgMock.WriteConfigCalls()) != 0 {
					t.Fatal("config called once or more")
				}
			},
		},
		{
			Name: "Test fails with error when failing to write config",
			Client: fake.NewFakeClientWithScheme(buildScheme(), &enmassev1.ConsoleService{
				ObjectMeta: v1.ObjectMeta{
					Name:      defaultConsoleSvcName,
					Namespace: defaultNamespace,
				},
				Status: enmassev1.ConsoleServiceStatus{
					Host: defaultHost,
					Port: 443,
				},
			}),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadAMQOnlineFunc: func() (ready *config.AMQOnline, e error) {
					return config.NewAMQOnline(config.ProductConfig{
						"NAMESPACE": defaultNamespace,
					}), nil
				},
				ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": defaultNamespace,
						"URL":       "sso.openshift-cluster.com",
					}), nil
				},
				WriteConfigFunc: func(config config.ConfigReadable) error {
					return errors.New("test error")
				},
			},
			ExpectedStatus:     v1alpha1.PhaseFailed,
			ExpectError:        true,
			ValidateCallCounts: func(t *testing.T, cfgMock *config.ConfigReadWriterMock) {},
		},
	}
	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			r, err := NewReconciler(s.FakeConfig, nil, nil)
			if err != nil {
				t.Fatal("could not create reconciler", err)
			}
			phase, err := r.reconcileConfig(context.TODO(), s.Client)
			if err != nil && !s.ExpectError {
				t.Fatal("failed to reconcile config", err)
			}
			if err == nil && s.ExpectError {
				t.Fatal("expected error but received nil")
			}
			if phase != s.ExpectedStatus {
				t.Fatalf("expected status %s but got %s", s.ExpectedStatus, phase)
			}
			s.ValidateCallCounts(t, s.FakeConfig)
		})

	}
}

func TestReconciler_fullReconcile(t *testing.T) {
	consoleSvc := &enmassev1.ConsoleService{
		ObjectMeta: v1.ObjectMeta{
			Name:      defaultConsoleSvcName,
			Namespace: defaultInstallationNamespace,
		},
	}

	installation := &v1alpha1.Installation{
		ObjectMeta: v1.ObjectMeta{
			Name:      "installation",
			Namespace: defaultInstallationNamespace,
			UID:       types.UID("xyz"),
		},
		TypeMeta: v1.TypeMeta{
			Kind:       "installation",
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
	}

	ns := &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: defaultInstallationNamespace,
			Labels: map[string]string{
				resources.OwnerLabelKey: string(installation.GetUID()),
			},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
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
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(buildScheme(), ns, consoleSvc, installation),
			FakeConfig:     basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient client.Client, owner ownerutil.Owner, os marketplacev1.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {

					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient client.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlanList, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlanList{
							Items: []operatorsv1alpha1.InstallPlan{
								{
									ObjectMeta: v1.ObjectMeta{
										Name: "amqonline-install-plan",
									},
									Status: operatorsv1alpha1.InstallPlanStatus{
										Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
									},
								},
							},
						}, &operatorsv1alpha1.Subscription{
							Status: operatorsv1alpha1.SubscriptionStatus{
								Install: &operatorsv1alpha1.InstallPlanReference{
									Name: "amqonline-install-plan",
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
