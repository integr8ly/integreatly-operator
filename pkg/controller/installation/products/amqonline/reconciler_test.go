package amqonline

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	chev1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	enmassev1 "github.com/enmasseproject/enmasse/pkg/apis/admin/v1beta1"
	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta1"
	enmassev1beta1 "github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta2"
	enmassev1beta2 "github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta2"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	kafkav1 "github.com/integr8ly/integreatly-operator/pkg/apis/kafka.strimzi.io/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	defaultNamespace = "amq-online"
)

func buildScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	chev1.SchemeBuilder.AddToScheme(scheme)
	aerogearv1.SchemeBuilder.AddToScheme(scheme)
	v1alpha1.SchemeBuilder.AddToScheme(scheme)
	operatorsv1alpha1.AddToScheme(scheme)
	marketplacev1.SchemeBuilder.AddToScheme(scheme)
	kafkav1.SchemeBuilder.AddToScheme(scheme)
	corev1.SchemeBuilder.AddToScheme(scheme)
	enmassev1.SchemeBuilder.AddToScheme(scheme)
	enmassev1beta1.SchemeBuilder.AddToScheme(scheme)
	enmassev1beta2.SchemeBuilder.AddToScheme(scheme)
	return scheme
}

func basicResourceSetServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		b := new(bytes.Buffer)
		json.NewEncoder(b).Encode(&resourceSet{
			AddrPlans: []*enmassev1beta2.AddressPlan{
				{
					ObjectMeta: v12.ObjectMeta{
						Name: "standard-large-multicast",
					},
				},
				{
					ObjectMeta: v12.ObjectMeta{
						Name: "standard-large-anycast",
					},
				},
			},
			AddrSpacePlans: []*enmassev1beta2.AddressSpacePlan{
				{
					ObjectMeta: v12.ObjectMeta{
						Name: "brokered-single-broker",
					},
				},
			},
			StdInfraConfigs: []*enmassev1beta1.StandardInfraConfig{
				{
					ObjectMeta: v12.ObjectMeta{
						Name: "default",
					},
				},
			},
			BrokeredInfraConfigs: []*enmassev1beta1.BrokeredInfraConfig{},
			AuthServices: []*enmassev1.AuthenticationService{
				{
					ObjectMeta: v12.ObjectMeta{
						Name: "standard-authservice",
					},
				},
				{
					ObjectMeta: v12.ObjectMeta{
						Name: "none-authservice",
					},
				},
				{
					ObjectMeta: v12.ObjectMeta{
						Name: "test-authservice",
					},
				},
			},
		})
		rw.Write(b.Bytes())
	}))
}

func basicAddressPlanList(ns string) []*enmassev1beta2.AddressPlan {
	return []*v1beta2.AddressPlan{
		{
			ObjectMeta: v12.ObjectMeta{
				Name:      "brokered-topic",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Brokered Topic",
				DisplayOrder:     0,
				ShortDescription: "Creates a topic on a broker.",
				LongDescription:  "Creates a topic on a broker.",
				AddressType:      "topic",
				Resources: v1beta2.AddressPlanResources{
					Broker: 0.0,
				},
			},
		},
		{
			ObjectMeta: v12.ObjectMeta{
				Name:      "brokered-queue",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Brokered Queue",
				DisplayOrder:     0,
				ShortDescription: "Creates a queue on a broker.",
				LongDescription:  "Creates a queue on a broker.",
				AddressType:      "queue",
				Resources: v1beta2.AddressPlanResources{
					Broker: 0.0,
				},
			},
		},
	}
}

func basicAuthServiceList(ns string) []*enmassev1.AuthenticationService {
	return []*enmassev1.AuthenticationService{
		{
			ObjectMeta: v12.ObjectMeta{
				Name:      "standard-authservice",
				Namespace: ns,
			},
			Spec: enmassev1.AuthenticationServiceSpec{
				Type: "standard",
			},
		},
		{
			ObjectMeta: v12.ObjectMeta{
				Name:      "none-authservice",
				Namespace: ns,
			},
			Spec: enmassev1.AuthenticationServiceSpec{
				Type: "none",
			},
		},
	}
}

func basicBrokeredInfraConfigList(ns string) []*v1beta1.BrokeredInfraConfig {
	return []*v1beta1.BrokeredInfraConfig{
		{
			ObjectMeta: v12.ObjectMeta{
				Name:      "default",
				Namespace: ns,
			},
			Spec: v1beta1.BrokeredInfraConfigSpec{
				Admin: v1beta1.InfraConfigAdmin{
					Resources: v1beta1.InfraConfigResources{
						Memory: "512Mi",
					},
				},
				Broker: v1beta1.InfraConfigBroker{
					Resources: v1beta1.InfraConfigResources{
						Memory:  "512Mi",
						Storage: "5Gi",
					},
					AddressFullPolicy: "FAIL",
				},
			},
		},
	}
}

func basicStandardInfraConfigList(ns string) []*v1beta1.StandardInfraConfig {
	return []*v1beta1.StandardInfraConfig{
		{
			ObjectMeta: v12.ObjectMeta{
				Name: "default-minimal",
			},
			Spec: v1beta1.StandardInfraConfigSpec{
				Admin: v1beta1.InfraConfigAdmin{
					Resources: v1beta1.InfraConfigResources{
						Memory: "512Mi",
					},
				},
				Broker: v1beta1.InfraConfigBroker{
					Resources: v1beta1.InfraConfigResources{
						Memory:  "512Mi",
						Storage: "2Gi",
					},
					AddressFullPolicy: "FAIL",
				},
				Router: v1beta1.InfraConfigRouter{
					MinReplicas: 1,
					Resources: v1beta1.InfraConfigResources{
						Memory: "256Mi",
					},
					LinkCapacity: 250,
				},
			},
		},
	}
}

func basicAddressSpacePlanList(ns string) []*v1beta2.AddressSpacePlan {
	return []*v1beta2.AddressSpacePlan{
		{
			ObjectMeta: v12.ObjectMeta{
				Name:      "brokered-single-broker",
				Namespace: ns,
			},
			Spec: v1beta2.AddressSpacePlanSpec{
				DisplayName:      "Single Broker",
				DisplayOrder:     0,
				InfraConfigRef:   "default",
				ShortDescription: "Single Broker instance",
				LongDescription:  "Single Broker plan where you can create an infinite number of queues until the system falls over.",
				AddressSpaceType: "brokered",
				ResourceLimits: v1beta2.AddressSpacePlanResourceLimits{
					Broker: 1.9,
				},
				AddressPlans: []string{
					"brokered-queue",
					"brokered-topic",
				},
			},
		},
	}
}

func basicConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
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
	}
}

func TestReconcile_reconcileNamespace(t *testing.T) {
	defaultInstallation := &v1alpha1.Installation{ObjectMeta: v12.ObjectMeta{Name: "install"}, TypeMeta: v12.TypeMeta{APIVersion: v1alpha1.SchemeGroupVersion.String()}}
	scenarios := []struct {
		Name           string
		FakeConfig     *config.ConfigReadWriterMock
		Installation   *v1alpha1.Installation
		ExpectErr      bool
		ExpectedStatus v1alpha1.StatusPhase
		FakeMPM        *marketplace.MarketplaceInterfaceMock
		FakeNSR        *resources.NamespaceReconcilerMock
	}{
		{
			Name:           "Test returns awaiting status when namespace is terminating",
			FakeConfig:     basicConfigMock(),
			Installation:   defaultInstallation,
			ExpectedStatus: v1alpha1.PhaseAwaitingNS,
			FakeNSR: &resources.NamespaceReconcilerMock{
				ReconcileFunc: func(ctx context.Context, ns *v1.Namespace, owner *v1alpha1.Installation) (*v1.Namespace, error) {
					return &v1.Namespace{
						ObjectMeta: v12.ObjectMeta{
							Name: "amq-online",
						},
						Status: v1.NamespaceStatus{
							Phase: v1.NamespaceTerminating,
						},
					}, nil
				},
			},
		},
		{
			Name:           "Test returns none phase when namespace is active",
			FakeConfig:     basicConfigMock(),
			Installation:   defaultInstallation,
			ExpectedStatus: v1alpha1.PhaseNone,
			FakeNSR: &resources.NamespaceReconcilerMock{
				ReconcileFunc: func(ctx context.Context, ns *v1.Namespace, owner *v1alpha1.Installation) (*v1.Namespace, error) {
					return &v1.Namespace{
						ObjectMeta: v12.ObjectMeta{
							Name: "amq-online",
						},
						Status: v1.NamespaceStatus{
							Phase: v1.NamespaceActive,
						},
					}, nil
				},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			r, err := NewReconciler(scenario.FakeConfig, scenario.Installation, scenario.FakeMPM, scenario.FakeNSR)
			if err != nil {
				t.Fatalf("could not create reconciler %v", err)
			}
			phase, err := r.reconcileNamespace(context.TODO(), scenario.Installation)
			if err != nil {
				t.Fatalf("could not reconcile namespace %v", err)
			}
			if phase != scenario.ExpectedStatus {
				t.Fatalf("expected status %s but got %s", scenario.ExpectedStatus, phase)
			}
		})
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
		FakeNSR        *resources.NamespaceReconcilerMock
	}{
		{
			Name:           "Test returns none phase if successfully creating new auth services",
			Client:         fake.NewFakeClientWithScheme(buildScheme()),
			FakeConfig:     basicConfigMock(),
			AuthServices:   basicAuthServiceList(defaultNamespace),
			ExpectedStatus: v1alpha1.PhaseNone,
		},
		{
			Name:           "Test returns none phase if trying to create existing auth services",
			Client:         fake.NewFakeClientWithScheme(buildScheme(), basicAuthServiceList(defaultSubscriptionName)[0]),
			AuthServices:   basicAuthServiceList(defaultNamespace),
			FakeConfig:     basicConfigMock(),
			ExpectedStatus: v1alpha1.PhaseNone,
		},
	}

	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			r, err := NewReconciler(s.FakeConfig, s.Installation, s.FakeMPM, s.FakeNSR)
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
		FakeNSR              *resources.NamespaceReconcilerMock
	}{
		{
			Name:                 "Test returns none phase if successfully creating new address space plans",
			Client:               fake.NewFakeClientWithScheme(buildScheme()),
			FakeConfig:           basicConfigMock(),
			BrokeredInfraConfigs: basicBrokeredInfraConfigList(defaultNamespace),
			StandardInfraConfigs: basicStandardInfraConfigList(defaultNamespace),
			ExpectedStatus:       v1alpha1.PhaseNone,
		},
		{
			Name:                 "Test returns none phase if trying to create existing address space plans",
			Client:               fake.NewFakeClientWithScheme(buildScheme(), basicAuthServiceList(defaultSubscriptionName)[0]),
			BrokeredInfraConfigs: basicBrokeredInfraConfigList(defaultNamespace),
			StandardInfraConfigs: basicStandardInfraConfigList(defaultNamespace),
			FakeConfig:           basicConfigMock(),
			ExpectedStatus:       v1alpha1.PhaseNone,
		},
	}

	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			r, err := NewReconciler(s.FakeConfig, s.Installation, s.FakeMPM, s.FakeNSR)
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
		FakeNSR        *resources.NamespaceReconcilerMock
	}{
		{
			Name:           "Test returns none phase if successfully creating new address space plans",
			Client:         fake.NewFakeClientWithScheme(buildScheme()),
			FakeConfig:     basicConfigMock(),
			AddressPlans:   basicAddressPlanList(defaultNamespace),
			ExpectedStatus: v1alpha1.PhaseNone,
		},
		{
			Name:           "Test returns none phase if trying to create existing address space plans",
			Client:         fake.NewFakeClientWithScheme(buildScheme(), basicAuthServiceList(defaultSubscriptionName)[0]),
			AddressPlans:   basicAddressPlanList(defaultNamespace),
			FakeConfig:     basicConfigMock(),
			ExpectedStatus: v1alpha1.PhaseNone,
		},
	}

	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			r, err := NewReconciler(s.FakeConfig, s.Installation, s.FakeMPM, s.FakeNSR)
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
		FakeNSR           *resources.NamespaceReconcilerMock
	}{
		{
			Name:              "Test returns none phase if successfully creating new address space plans",
			Client:            fake.NewFakeClientWithScheme(buildScheme()),
			FakeConfig:        basicConfigMock(),
			AddressSpacePlans: basicAddressSpacePlanList(defaultNamespace),
			ExpectedStatus:    v1alpha1.PhaseNone,
		},
		{
			Name:              "Test returns none phase if trying to create existing address space plans",
			Client:            fake.NewFakeClientWithScheme(buildScheme(), basicAuthServiceList(defaultSubscriptionName)[0]),
			AddressSpacePlans: basicAddressSpacePlanList(defaultNamespace),
			FakeConfig:        basicConfigMock(),
			ExpectedStatus:    v1alpha1.PhaseNone,
		},
	}

	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			r, err := NewReconciler(s.FakeConfig, s.Installation, s.FakeMPM, s.FakeNSR)
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
				ObjectMeta: v12.ObjectMeta{
					Name:      defaultConsoleSvcName,
					Namespace: defaultNamespace,
				},
				Status: enmassev1.ConsoleServiceStatus{
					Host: defaultHost,
					Port: 0,
				},
			}),
			FakeConfig:     basicConfigMock(),
			ExpectedStatus: v1alpha1.PhaseNone,
			ValidateCallCounts: func(t *testing.T, cfgMock *config.ConfigReadWriterMock) {
				if len(cfgMock.WriteConfigCalls()) != 0 {
					t.Fatal("config written once or more")
				}
			},
		},
		{
			Name: "Test doesn't set host when the host is undefined or empty",
			Client: fake.NewFakeClientWithScheme(buildScheme(), &enmassev1.ConsoleService{
				ObjectMeta: v12.ObjectMeta{
					Name:      defaultConsoleSvcName,
					Namespace: defaultNamespace,
				},
				Status: enmassev1.ConsoleServiceStatus{
					Host: "",
					Port: 443,
				},
			}),
			FakeConfig:     basicConfigMock(),
			ExpectedStatus: v1alpha1.PhaseNone,
			ValidateCallCounts: func(t *testing.T, cfgMock *config.ConfigReadWriterMock) {
				if len(cfgMock.WriteConfigCalls()) != 0 {
					t.Fatal("config written once or more")
				}
			},
		},
		{
			Name: "Test successfully setting host when port and host are defined properly",
			Client: fake.NewFakeClientWithScheme(buildScheme(), &enmassev1.ConsoleService{
				ObjectMeta: v12.ObjectMeta{
					Name:      defaultConsoleSvcName,
					Namespace: defaultNamespace,
				},
				Status: enmassev1.ConsoleServiceStatus{
					Host: defaultHost,
					Port: 443,
				},
			}),
			FakeConfig:     basicConfigMock(),
			ExpectedStatus: v1alpha1.PhaseNone,
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
			ExpectedStatus: v1alpha1.PhaseNone,
			ValidateCallCounts: func(t *testing.T, cfgMock *config.ConfigReadWriterMock) {
				if len(cfgMock.WriteConfigCalls()) != 0 {
					t.Fatal("config called once or more")
				}
			},
		},
		{
			Name: "Test fails with error when failing to write config",
			Client: fake.NewFakeClientWithScheme(buildScheme(), &enmassev1.ConsoleService{
				ObjectMeta: v12.ObjectMeta{
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
			r, err := NewReconciler(s.FakeConfig, nil, nil, nil)
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

func TestReconcile_getResourceSetFromURL(t *testing.T) {
	scenarios := []struct {
		Server                      *httptest.Server
		ExpectError                 bool
		ExpectedAddrPlanCount       int
		ExpectedAddrSpacePlanCount  int
		ExpectedAuthServiceCount    int
		ExpectedStdInfraConfigCount int
		ExpectedBrokeredConfigCount int
	}{
		{
			Server:                      basicResourceSetServer(),
			ExpectedAddrPlanCount:       2,
			ExpectedBrokeredConfigCount: 0,
			ExpectedStdInfraConfigCount: 1,
			ExpectedAuthServiceCount:    3,
			ExpectedAddrSpacePlanCount:  1,
		},
		{
			Server: httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				rw.WriteHeader(500)
			})),
			ExpectError: true,
		},
		{
			Server: httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				rw.Write([]byte("[{ \"testKey\": \"testVal\" }]"))
			})),
			ExpectError: true,
		},
	}
	for _, s := range scenarios {
		defer s.Server.Close()
		rs, err := getResourceSetFromURL(s.Server.URL, s.Server.Client())
		if err != nil && !s.ExpectError {
			t.Fatal("error getting resource set from URL", err)
		}
		if err == nil && s.ExpectError {
			t.Fatal("expected error but received none")
		}
		if err == nil {
			if len(rs.AddrPlans) != s.ExpectedAddrPlanCount {
				t.Fatalf("incorrect address plan count, expected %d but got %d", s.ExpectedAddrPlanCount, len(rs.AddrPlans))
			}
			if len(rs.AddrSpacePlans) != s.ExpectedAddrSpacePlanCount {
				t.Fatalf("incorrect address space plan count, expected %d but got %d", s.ExpectedAddrSpacePlanCount, len(rs.AddrSpacePlans))
			}
			if len(rs.AuthServices) != s.ExpectedAuthServiceCount {
				t.Fatalf("incorrect auth service count, expected %d but got %d", s.ExpectedAuthServiceCount, len(rs.AuthServices))
			}
			if len(rs.StdInfraConfigs) != s.ExpectedStdInfraConfigCount {
				t.Fatalf("incorrect standard infra config count, expected %d but got %d", s.ExpectedStdInfraConfigCount, len(rs.StdInfraConfigs))
			}
			if len(rs.BrokeredInfraConfigs) != s.ExpectedBrokeredConfigCount {
				t.Fatalf("incorrect brokered infra config count, expected %d but got %d", s.ExpectedBrokeredConfigCount, len(rs.BrokeredInfraConfigs))
			}
		}
	}
}

func TestReconcile_getResourceSetFromURLList(t *testing.T) {
	scenario := []struct {
		Servers                     []*httptest.Server
		ExpectError                 bool
		ExpectedAddrPlanCount       int
		ExpectedAddrSpacePlanCount  int
		ExpectedAuthServiceCount    int
		ExpectedStdInfraConfigCount int
		ExpectedBrokeredConfigCount int
	}{
		{
			Servers: []*httptest.Server{
				basicResourceSetServer(),
				basicResourceSetServer(),
			},
			ExpectedAddrPlanCount:       4,
			ExpectedBrokeredConfigCount: 0,
			ExpectedStdInfraConfigCount: 2,
			ExpectedAuthServiceCount:    6,
			ExpectedAddrSpacePlanCount:  2,
		},
		{
			Servers: []*httptest.Server{
				basicResourceSetServer(),
				httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					rw.WriteHeader(500)
				})),
			},
			ExpectError: true,
		},
	}
	for _, s := range scenario {
		serverURLs := []string{}
		for _, server := range s.Servers {
			serverURLs = append(serverURLs, server.URL)
		}
		rs, err := getResourceSetFromURLList(serverURLs, s.Servers[0].Client())
		if err != nil && !s.ExpectError {
			t.Fatal("error getting resource set from URL", err)
		}
		if err == nil && s.ExpectError {
			t.Fatal("expected error but received none")
		}

		if err == nil {
			if len(rs.AddrPlans) != s.ExpectedAddrPlanCount {
				t.Fatalf("incorrect address plan count, expected %d but got %d", s.ExpectedAddrPlanCount, len(rs.AddrPlans))
			}
			if len(rs.AddrSpacePlans) != s.ExpectedAddrSpacePlanCount {
				t.Fatalf("incorrect address space plan count, expected %d but got %d", s.ExpectedAddrSpacePlanCount, len(rs.AddrSpacePlans))
			}
			if len(rs.AuthServices) != s.ExpectedAuthServiceCount {
				t.Fatalf("incorrect auth service count, expected %d but got %d", s.ExpectedAuthServiceCount, len(rs.AuthServices))
			}
			if len(rs.StdInfraConfigs) != s.ExpectedStdInfraConfigCount {
				t.Fatalf("incorrect standard infra config count, expected %d but got %d", s.ExpectedStdInfraConfigCount, len(rs.StdInfraConfigs))
			}
			if len(rs.BrokeredInfraConfigs) != s.ExpectedBrokeredConfigCount {
				t.Fatalf("incorrect brokered infra config count, expected %d but got %d", s.ExpectedBrokeredConfigCount, len(rs.BrokeredInfraConfigs))
			}
		}
	}
}
