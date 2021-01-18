package amqonline

import (
	"context"
	"errors"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"testing"

	prometheusmonitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"

	chev1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"

	monitoring "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	crotypes "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	enmassev1 "github.com/integr8ly/integreatly-operator/pkg/apis-products/enmasse/admin/v1beta1"
	enmassev1beta1 "github.com/integr8ly/integreatly-operator/pkg/apis-products/enmasse/v1beta1"
	enmassev1beta2 "github.com/integr8ly/integreatly-operator/pkg/apis-products/enmasse/v1beta2"
	kafkav1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis-products/kafka.strimzi.io/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	projectv1 "github.com/openshift/api/project/v1"

	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"

	crov1alpha1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func buildScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	chev1.SchemeBuilder.AddToScheme(scheme)
	keycloak.SchemeBuilder.AddToScheme(scheme)
	integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	operatorsv1alpha1.AddToScheme(scheme)
	marketplacev1.SchemeBuilder.AddToScheme(scheme)
	kafkav1alpha1.SchemeBuilder.AddToScheme(scheme)
	corev1.SchemeBuilder.AddToScheme(scheme)
	enmassev1.SchemeBuilder.AddToScheme(scheme)
	enmassev1beta1.SchemeBuilder.AddToScheme(scheme)
	enmassev1beta2.SchemeBuilder.AddToScheme(scheme)
	rbacv1.SchemeBuilder.AddToScheme(scheme)
	batchv1beta1.SchemeBuilder.AddToScheme(scheme)
	appsv1.SchemeBuilder.AddToScheme(scheme)
	monitoring.SchemeBuilder.AddToScheme(scheme)
	prometheusmonitoringv1.SchemeBuilder.AddToScheme(scheme)
	projectv1.AddToScheme(scheme)
	crov1alpha1.SchemeBuilder.AddToScheme(scheme)
	return scheme
}

const (
	defaultNamespace = "amq-online"
)

func basicInstallation() *integreatlyv1alpha1.RHMI {
	return &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "integreatly",
			Namespace: "integreatly",
		},
	}
}

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

func authServiceSecretMock() *corev1.Secret {
	config := basicConfigMock()
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "standard-authservice-postgresql",
			Namespace: config.GetOperatorNamespace(),
		},
		Data: map[string][]byte{},
	}
}

func croPostgresSecretMock(installationNamespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "standard-authservice-postgresql",
			Namespace: installationNamespace,
		},
		Data: map[string][]byte{
			"host":     []byte("dummy"),
			"port":     []byte("5432"),
			"database": []byte("dummy"),
			"username": []byte("dummy"),
			"password": []byte("dummy"),
		},
	}
}

func serviceAdminRoleMock(installationNamespace string) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "enmasse.io:service-admin",
			Namespace: installationNamespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"dummy"},
				Resources: []string{"dummy"},
				Verbs:     []string{"dummy"},
			},
		},
	}
}

func serviceAdminRoleBindingMock(installationNamespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dedicated-admins-service-admin",
			Namespace: installationNamespace,
		},
		RoleRef: rbacv1.RoleRef{
			Name: "dummy",
			Kind: "Role",
		},
		Subjects: []rbacv1.Subject{
			{
				Name: "dummy",
				Kind: "Group",
			},
		},
	}
}

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
}

func TestReconcile_reconcileAuthServices(t *testing.T) {

	postgres := &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "standard-authservice-postgresql",
			Namespace: "test-namespace",
		},
		Spec: crov1.PostgresSpec{},
		Status: crov1.PostgresStatus{
			Phase: crotypes.PhaseComplete,
			SecretRef: &crotypes.SecretRef{
				Name:      "enmasse-postgres-secret",
				Namespace: "test-namespace",
			},
		},
	}

	backupSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "enmasse-postgres-secret",
			Namespace: "test-namespace",
		},
	}

	scenarios := []struct {
		Name           string
		Client         k8sclient.Client
		FakeConfig     *config.ConfigReadWriterMock
		Installation   *integreatlyv1alpha1.RHMI
		ExpectedStatus integreatlyv1alpha1.StatusPhase
		AuthServices   []*enmassev1.AuthenticationService
		FakeMPM        *marketplace.MarketplaceInterfaceMock
		Recorder       record.EventRecorder
	}{
		{
			Name:           "Test returns completed phase if successfully creating new auth services",
			Client:         fake.NewFakeClientWithScheme(buildScheme(), croPostgresSecretMock("test-namespace"), postgres, backupSecret),
			FakeConfig:     basicConfigMock(),
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-installation",
					Namespace: "test-namespace",
				},
			},
			Recorder: setupRecorder(),
		},
		{
			Name:           "Test returns completed phase if trying to create existing auth services",
			Client:         fake.NewFakeClientWithScheme(buildScheme(), croPostgresSecretMock("test-namespace"), postgres, backupSecret),
			FakeConfig:     basicConfigMock(),
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-installation",
					Namespace: "test-namespace",
				},
			},
			Recorder: setupRecorder(),
		},
	}

	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			r, err := NewReconciler(s.FakeConfig, s.Installation, s.FakeMPM, s.Recorder, getLogger())
			if err != nil {
				t.Fatalf("could not create reconciler %v", err)
			}
			phase, err := r.reconcileStandardAuthenticationService(context.TODO(), s.Client)
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if phase != s.ExpectedStatus {
				t.Fatalf("expected status %s but got %s", s.ExpectedStatus, phase)
			}
		})
	}
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.ProductAMQOnline})
}

func TestReconcile_reconcileInfraConfigs(t *testing.T) {
	scenarios := []struct {
		Name                 string
		Client               k8sclient.Client
		FakeConfig           *config.ConfigReadWriterMock
		Installation         *integreatlyv1alpha1.RHMI
		ExpectedStatus       integreatlyv1alpha1.StatusPhase
		BrokeredInfraConfigs []*enmassev1beta1.BrokeredInfraConfig
		StandardInfraConfigs []*enmassev1beta1.StandardInfraConfig
		FakeMPM              *marketplace.MarketplaceInterfaceMock
		Recorder             record.EventRecorder
	}{
		{
			Name:                 "Test returns completed phase if successfully creating new infra configs",
			Client:               fake.NewFakeClientWithScheme(buildScheme()),
			FakeConfig:           basicConfigMock(),
			BrokeredInfraConfigs: GetDefaultBrokeredInfraConfigs(defaultNamespace),
			StandardInfraConfigs: GetDefaultStandardInfraConfigs(defaultNamespace),
			ExpectedStatus:       integreatlyv1alpha1.PhaseCompleted,
			Installation:         basicInstallation(),
			Recorder:             setupRecorder(),
		},
		{
			Name:                 "Test returns completed phase if trying to create existing infra configs",
			Client:               fake.NewFakeClientWithScheme(buildScheme(), GetDefaultBrokeredInfraConfigs(defaultNamespace)[0], GetDefaultStandardInfraConfigs(defaultNamespace)[0]),
			BrokeredInfraConfigs: GetDefaultBrokeredInfraConfigs(defaultNamespace),
			StandardInfraConfigs: GetDefaultStandardInfraConfigs(defaultNamespace),
			FakeConfig:           basicConfigMock(),
			ExpectedStatus:       integreatlyv1alpha1.PhaseCompleted,
			Installation:         basicInstallation(),
			Recorder:             setupRecorder(),
		},
	}

	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			r, err := NewReconciler(s.FakeConfig, s.Installation, s.FakeMPM, s.Recorder, getLogger())
			if err != nil {
				t.Fatalf("could not create reconciler %v", err)
			}
			phase, err := r.reconcileInfraConfigs(context.TODO(), s.Client, s.BrokeredInfraConfigs, s.StandardInfraConfigs)
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
		Client         k8sclient.Client
		FakeConfig     *config.ConfigReadWriterMock
		Installation   *integreatlyv1alpha1.RHMI
		ExpectedStatus integreatlyv1alpha1.StatusPhase
		AddressPlans   []*enmassev1beta2.AddressPlan
		FakeMPM        *marketplace.MarketplaceInterfaceMock
		Recorder       record.EventRecorder
	}{
		{
			Name:           "Test returns completed phase if successfully creating new address plans",
			Client:         fake.NewFakeClientWithScheme(buildScheme()),
			FakeConfig:     basicConfigMock(),
			AddressPlans:   GetDefaultAddressPlans(defaultNamespace),
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Installation:   basicInstallation(),
			Recorder:       setupRecorder(),
		},
		{
			Name:           "Test returns completed phase if trying to create existing address plans",
			Client:         fake.NewFakeClientWithScheme(buildScheme(), GetDefaultAddressPlans(defaultNamespace)[0]),
			AddressPlans:   GetDefaultAddressPlans(defaultNamespace),
			FakeConfig:     basicConfigMock(),
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Installation:   basicInstallation(),
			Recorder:       setupRecorder(),
		},
	}

	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			r, err := NewReconciler(s.FakeConfig, s.Installation, s.FakeMPM, s.Recorder, getLogger())
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
		Client            k8sclient.Client
		FakeConfig        *config.ConfigReadWriterMock
		Installation      *integreatlyv1alpha1.RHMI
		ExpectedStatus    integreatlyv1alpha1.StatusPhase
		AddressSpacePlans []*enmassev1beta2.AddressSpacePlan
		FakeMPM           *marketplace.MarketplaceInterfaceMock
		Recorder          record.EventRecorder
	}{
		{
			Name:              "Test returns completed phase if successfully creating new address space plans",
			Client:            fake.NewFakeClientWithScheme(buildScheme()),
			FakeConfig:        basicConfigMock(),
			AddressSpacePlans: GetDefaultAddressSpacePlans(defaultNamespace),
			ExpectedStatus:    integreatlyv1alpha1.PhaseCompleted,
			Installation:      basicInstallation(),
			Recorder:          setupRecorder(),
		},
		{
			Name:              "Test returns completed phase if trying to create existing address space plans",
			Client:            fake.NewFakeClientWithScheme(buildScheme(), GetDefaultAddressSpacePlans(defaultNamespace)[0]),
			AddressSpacePlans: GetDefaultAddressSpacePlans(defaultNamespace),
			FakeConfig:        basicConfigMock(),
			ExpectedStatus:    integreatlyv1alpha1.PhaseCompleted,
			Installation:      basicInstallation(),
			Recorder:          setupRecorder(),
		},
	}

	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			r, err := NewReconciler(s.FakeConfig, s.Installation, s.FakeMPM, s.Recorder, getLogger())
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

func TestReconcile_reconcileServiceAdmin(t *testing.T) {
	scenarios := []struct {
		Name             string
		Client           k8sclient.Client
		FakeConfig       *config.ConfigReadWriterMock
		Installation     *integreatlyv1alpha1.RHMI
		ExpectedStatus   integreatlyv1alpha1.StatusPhase
		ServiceAdminRole *rbacv1.Role
		FakeMPM          *marketplace.MarketplaceInterfaceMock
		Recorder         record.EventRecorder
	}{
		{
			Name:           "Test returns completed phase if successfully creating amq online service admin role and role binding",
			Client:         fake.NewFakeClientWithScheme(buildScheme()),
			FakeConfig:     basicConfigMock(),
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Installation:   basicInstallation(),
			Recorder:       setupRecorder(),
		},
		{
			Name:           "Test returns completed phase if trying to create existing amq online service admin role and role binding",
			Client:         fake.NewFakeClientWithScheme(buildScheme(), serviceAdminRoleMock(defaultNamespace), serviceAdminRoleBindingMock(defaultNamespace)),
			FakeConfig:     basicConfigMock(),
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Installation:   basicInstallation(),
			Recorder:       setupRecorder(),
		},
	}

	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			r, err := NewReconciler(s.FakeConfig, s.Installation, s.FakeMPM, s.Recorder, getLogger())
			if err != nil {
				t.Fatalf("could not create reconciler %v", err)
			}
			phase, err := r.reconcileServiceAdmin(context.TODO(), s.Client)
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
		Client             k8sclient.Client
		ExpectedStatus     integreatlyv1alpha1.StatusPhase
		FakeConfig         *config.ConfigReadWriterMock
		ExpectError        bool
		ValidateCallCounts func(t *testing.T, cfgMock *config.ConfigReadWriterMock)
		Recorder           record.EventRecorder
	}{
		{
			Name: "Test doesn't set host when the port is not 443",
			Client: fake.NewFakeClientWithScheme(buildScheme(), &enmassev1.ConsoleService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultConsoleSvcName,
					Namespace: defaultNamespace,
				},
				Status: enmassev1.ConsoleServiceStatus{
					Host: defaultHost,
					Port: 0,
				},
			}),
			FakeConfig:     basicConfigMock(),
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			ValidateCallCounts: func(t *testing.T, cfgMock *config.ConfigReadWriterMock) {
				if len(cfgMock.WriteConfigCalls()) != 0 {
					t.Fatal("config written once or more")
				}
			},
			Recorder: setupRecorder(),
		},
		{
			Name: "Test doesn't set host when the host is undefined or empty",
			Client: fake.NewFakeClientWithScheme(buildScheme(), &enmassev1.ConsoleService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultConsoleSvcName,
					Namespace: defaultNamespace,
				},
				Status: enmassev1.ConsoleServiceStatus{
					Host: "",
					Port: 443,
				},
			}),
			FakeConfig:     basicConfigMock(),
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			ValidateCallCounts: func(t *testing.T, cfgMock *config.ConfigReadWriterMock) {
				if len(cfgMock.WriteConfigCalls()) != 0 {
					t.Fatal("config written once or more")
				}
			},
			Recorder: setupRecorder(),
		},
		{
			Name: "Test successfully setting host when port and host are defined properly",
			Client: fake.NewFakeClientWithScheme(buildScheme(), &enmassev1.ConsoleService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultConsoleSvcName,
					Namespace: defaultNamespace,
				},
				Status: enmassev1.ConsoleServiceStatus{
					Host: defaultHost,
					Port: 443,
				},
			}),
			FakeConfig:     basicConfigMock(),
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
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
			Recorder: setupRecorder(),
		},
		{
			Name:           "Test continues when console it not found",
			Client:         fake.NewFakeClientWithScheme(buildScheme()),
			FakeConfig:     basicConfigMock(),
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			ExpectError:    true,
			ValidateCallCounts: func(t *testing.T, cfgMock *config.ConfigReadWriterMock) {
				if len(cfgMock.WriteConfigCalls()) != 0 {
					t.Fatal("config called once or more")
				}
			},
			Recorder: setupRecorder(),
		},
		{
			Name: "Test fails with error when failing to write config",
			Client: fake.NewFakeClientWithScheme(buildScheme(), &enmassev1.ConsoleService{
				ObjectMeta: metav1.ObjectMeta{
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
			ExpectedStatus:     integreatlyv1alpha1.PhaseFailed,
			ExpectError:        true,
			ValidateCallCounts: func(t *testing.T, cfgMock *config.ConfigReadWriterMock) {},
			Recorder:           setupRecorder(),
		},
	}
	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			r, err := NewReconciler(s.FakeConfig, nil, nil, s.Recorder, getLogger())
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
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultConsoleSvcName,
			Namespace: defaultInstallationNamespace,
		},
	}

	postgres := &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "standard-authservice-postgresql",
			Namespace: defaultInstallationNamespace,
		},
		Spec: crov1.PostgresSpec{},
		Status: crov1.PostgresStatus{
			Phase: crotypes.PhaseComplete,
			SecretRef: &crotypes.SecretRef{
				Name:      "enmasse-postgres-secret",
				Namespace: defaultInstallationNamespace,
			},
		},
	}

	backupSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defaultInstallationNamespace,
			Name:      "enmasse-postgres-secret",
		},
	}

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

	operatorDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "enmasse-operator",
			Namespace: "amq-online-operator",
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
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(buildScheme(), ns, operatorNS, consoleSvc, installation, operatorDeployment, backupsSecretMock(), croPostgresSecretMock(installation.Namespace), postgres, backupSecret),
			FakeConfig:     basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {

					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlanList, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlanList{
							Items: []operatorsv1alpha1.InstallPlan{
								{
									ObjectMeta: metav1.ObjectMeta{
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
				t.Fatalf("unexpected error: %v", err)
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
