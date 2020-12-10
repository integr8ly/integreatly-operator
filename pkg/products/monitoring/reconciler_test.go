package monitoring

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	configv1 "github.com/openshift/api/config/v1"
	"testing"

	grafanav1alpha1 "github.com/integr8ly/grafana-operator/v3/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	v1 "github.com/openshift/api/route/v1"

	prometheusmonitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"

	monitoringv1 "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	projectv1 "github.com/openshift/api/project/v1"

	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"

	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	mockSMTPSecretName               = "test-smtp"
	mockPagerdutySecretName          = "test-pd"
	mockDMSSecretName                = "test-dms"
	mockCustomerAlertingEmailAddress = "noreply-customer-test@rhmi-redhat.com"
	mockAlertingEmailAddress         = "noreply-test@rhmi-redhat.com"
	mockBUAlertingEmailAddress       = "noreply-bu-test@rhmi-redhat.com"
	mockIsMultiAZCluster             = true
	mockAlertFromAddress             = "noreply-alert@devshift.org"
)

func basicInstallation() *integreatlyv1alpha1.RHMI {
	return &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "installation",
			Namespace: defaultInstallationNamespace,
			UID:       types.UID("xyz"),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       integreatlyv1alpha1.SchemaGroupVersionKind.Kind,
			APIVersion: integreatlyv1alpha1.SchemeGroupVersion.String(),
		},
		Spec: integreatlyv1alpha1.RHMISpec{
			SMTPSecret:           mockSMTPSecretName,
			PagerDutySecret:      mockPagerdutySecretName,
			DeadMansSnitchSecret: mockDMSSecretName,
			Type:                 string(integreatlyv1alpha1.InstallationTypeManaged),
		},
	}
}

func basicInstallationWithAlertEmailAddress() *integreatlyv1alpha1.RHMI {
	installation := basicInstallation()
	installation.Spec.AlertFromAddress = mockAlertFromAddress
	installation.Spec.AlertingEmailAddress = mockCustomerAlertingEmailAddress
	installation.Spec.AlertingEmailAddresses.CSSRE = mockAlertingEmailAddress
	installation.Spec.AlertingEmailAddresses.BusinessUnit = mockBUAlertingEmailAddress
	return installation
}

func basicConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		ReadMonitoringFunc: func() (ready *config.Monitoring, e error) {
			return config.NewMonitoring(config.ProductConfig{}), nil
		},
		WriteConfigFunc: func(config config.ConfigReadable) error {
			return nil
		},
	}
}

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := monitoringv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := operatorsv1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := marketplacev1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := corev1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := coreosv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := prometheusmonitoringv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := projectv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := v1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := rbac.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := grafanav1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := configv1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	return scheme, nil
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
		Recorder       record.EventRecorder
	}{
		{
			Name:           "test error on failed config",
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			ExpectError:    true,
			ExpectedError:  "could not read monitoring config",
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeClient:     fakeclient.NewFakeClient(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadMonitoringFunc: func() (ready *config.Monitoring, e error) {
					return nil, errors.New("could not read monitoring config")
				},
			},
			Recorder: setupRecorder(),
		},
		{
			Name:         "test namespace is set without fail",
			Installation: &integreatlyv1alpha1.RHMI{},
			FakeClient:   fakeclient.NewFakeClient(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadMonitoringFunc: func() (ready *config.Monitoring, e error) {
					return config.NewMonitoring(config.ProductConfig{
						"NAMESPACE": "",
					}), nil
				},
			},
			Recorder: setupRecorder(),
		},
		{
			Name:           "test subscription phase with error from mpm",
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			ExpectError:    true,
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {
					return errors.New("dummy")
				},
			},
			FakeClient: fakeclient.NewFakeClient(),
			FakeConfig: basicConfigMock(),
			Recorder:   setupRecorder(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM, tc.Recorder, getLogger())
			if err != nil && err.Error() != tc.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
			}
			if err == nil && tc.ExpectedError != "" {
				t.Fatalf("expected error '%v' and got nil", tc.ExpectedError)
			}
		})
	}

}

func TestReconciler_reconcileCustomResource(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		Name           string
		FakeClient     k8sclient.Client
		FakeConfig     *config.ConfigReadWriterMock
		Installation   *integreatlyv1alpha1.RHMI
		ExpectError    bool
		ExpectedError  string
		ExpectedStatus integreatlyv1alpha1.StatusPhase
		FakeMPM        *marketplace.MarketplaceInterfaceMock
		Recorder       record.EventRecorder
	}{
		{
			Name:           "Test reconcile custom resource returns success on successful create",
			FakeClient:     fakeclient.NewFakeClientWithScheme(scheme),
			FakeConfig:     basicConfigMock(),
			Installation:   &integreatlyv1alpha1.RHMI{},
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Recorder:       setupRecorder(),
		},
		{
			Name: "Test reconcile custom resource returns failed on unsuccessful create",
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return k8serr.NewNotFound(schema.GroupResource{
						Group:    monitoringv1.SchemeBuilder.GroupVersion.Group,
						Resource: "ApplicationMonitoring",
					}, key.Name)
				},
				CreateFunc: func(ctx context.Context, obj runtime.Object, opts ...k8sclient.CreateOption) error {
					return errors.New("dummy create error")
				},
			},
			FakeConfig:     basicConfigMock(),
			Installation:   &integreatlyv1alpha1.RHMI{},
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			ExpectError:    true,
			ExpectedError:  "failed to create/update applicationmonitoring custom resource",
			Recorder:       setupRecorder(),
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM, tc.Recorder, getLogger())
			if err != nil {
				t.Fatal("unexpected err ", err)
			}

			phase, err := reconciler.reconcileComponents(context.TODO(), tc.FakeClient, mockIsMultiAZCluster)
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

	// initialise runtime objects
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultInstallationNamespace,
			Labels: map[string]string{
				resources.OwnerLabelKey: string(basicInstallation().GetUID()),
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
				resources.OwnerLabelKey: string(basicInstallation().GetUID()),
			},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}
	grafanadatasourcesecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-datasources",
			Namespace: "openshift-monitoring",
		},
		Data: map[string][]byte{
			"prometheus.yaml": []byte("{\"datasources\":[{\"basicAuthUser\":\"testuser\",\"basicAuthPassword\":\"testpass\"}]}"),
		},
	}
	federationNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultInstallationNamespace + "-federate",
			Labels: map[string]string{
				resources.OwnerLabelKey:           string(basicInstallation().GetUID()),
				"openshift.io/cluster-monitoring": "true",
			},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}

	grafana := &grafanav1alpha1.Grafana{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana",
			Namespace: defaultInstallationNamespace,
		},
	}

	installation := basicInstallation()

	smtpSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockSMTPSecretName,
			Namespace: installation.Namespace,
		},
		Data: map[string][]byte{
			"host":     []byte("smtp.sendgrid.com"),
			"port":     []byte("587"),
			"username": []byte("test"),
			"password": []byte("test"),
		},
		Type: corev1.SecretTypeOpaque,
	}

	pagerdutySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockPagerdutySecretName,
			Namespace: installation.Namespace,
		},
		Data: map[string][]byte{
			"PAGERDUTY_KEY": []byte("test"),
		},
		Type: corev1.SecretTypeOpaque,
	}

	dmsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockDMSSecretName,
			Namespace: installation.Namespace,
		},
		Data: map[string][]byte{
			"SNITCH_URL": []byte("https://example.com"),
		},
		Type: corev1.SecretTypeOpaque,
	}

	alertmanagerRoute := &v1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      alertManagerRouteName,
			Namespace: defaultInstallationNamespace,
		},
		Spec: v1.RouteSpec{
			Host: "example.com",
		},
	}

	clusterInfra := &configv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterInfraName,
		},
		Status: configv1.InfrastructureStatus{
			InfrastructureName: "cluster-infra",
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
			FakeClient: moqclient.NewSigsClientMoqWithScheme(scheme, ns, operatorNS, federationNs,
				grafanadatasourcesecret, installation, smtpSecret, pagerdutySecret,
				dmsSecret, alertmanagerRoute, grafana, clusterInfra),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadMonitoringFunc: func() (ready *config.Monitoring, e error) {
					return config.NewMonitoring(config.ProductConfig{
						"NAMESPACE":          "",
						"OPERATOR_NAMESPACE": defaultInstallationNamespace,
					}), nil
				},
				ReadThreeScaleFunc: func() (ready *config.ThreeScale, e error) {
					return config.NewThreeScale(config.ProductConfig{
						"NAMESPACE": "",
					}), nil
				},
				WriteConfigFunc: func(config config.ConfigReadable) error {
					return nil
				},
			},
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plan *operatorsv1alpha1.InstallPlanList, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlanList{
							TypeMeta: metav1.TypeMeta{
								Kind:       "ApplicationMonitoring",
								APIVersion: monitoringv1.SchemeGroupVersion.String(),
							},
							Items: []operatorsv1alpha1.InstallPlan{
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "monitoring-install-plan",
									},
									Status: operatorsv1alpha1.InstallPlanStatus{
										Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
									},
								},
							},
							ListMeta: metav1.ListMeta{},
						}, &operatorsv1alpha1.Subscription{
							Status: operatorsv1alpha1.SubscriptionStatus{
								Install: &operatorsv1alpha1.InstallPlanReference{
									Name: "monitoring-install-plan",
								},
							},
						}, nil
				},
			},
			Installation: basicInstallation(),
			Product:      &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:     setupRecorder(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM, tc.Recorder, getLogger())
			if err != nil && err.Error() != tc.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
			}

			status, err := reconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.FakeClient)
			if err != nil && !tc.ExpectError {
				t.Fatalf("expected no error but got one: %v", err)
			}
			if err == nil && tc.ExpectError {
				t.Fatal("expected error but got none")
			}
			if status != tc.ExpectedStatus {
				t.Fatalf("Expected status: '%v', got: '%v'", tc.ExpectedStatus, status)
			}

			//Verify that grafana dashboards are created
			for _, dashboard := range reconciler.Config.GetDashboards(integreatlyv1alpha1.InstallationType(installation.Spec.Type)) {
				grafanaDB := &grafanav1alpha1.GrafanaDashboard{}
				err = tc.FakeClient.Get(context.TODO(), k8sclient.ObjectKey{Name: dashboard, Namespace: defaultInstallationNamespace}, grafanaDB)
				if err != nil {
					t.Fatalf("expected no error but got one: %v", err)
				}
			}
		})
	}
}

func TestReconciler_testPhases(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultInstallationNamespace,
			Labels: map[string]string{
				resources.OwnerLabelKey: string(basicInstallation().GetUID()),
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
				resources.OwnerLabelKey: string(basicInstallation().GetUID()),
			},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}

	federationNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultInstallationNamespace + "-federate",
			Labels: map[string]string{
				resources.OwnerLabelKey:           string(basicInstallation().GetUID()),
				"openshift.io/cluster-monitoring": "true",
			},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}

	cases := []struct {
		Name           string
		ExpectedStatus integreatlyv1alpha1.StatusPhase
		FakeConfig     *config.ConfigReadWriterMock
		FakeClient     k8sclient.Client
		FakeMPM        *marketplace.MarketplaceInterfaceMock
		Installation   *integreatlyv1alpha1.RHMI
		Product        *integreatlyv1alpha1.RHMIProductStatus
		Recorder       record.EventRecorder
	}{
		{
			Name:           "test namespace terminating returns phase in progress",
			ExpectedStatus: integreatlyv1alpha1.PhaseInProgress,
			Installation:   basicInstallation(),
			FakeClient: moqclient.NewSigsClientMoqWithScheme(scheme, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: defaultInstallationNamespace,
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceTerminating,
				},
			}, operatorNS, federationNs, basicInstallation()),
			FakeConfig: basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plan *operatorsv1alpha1.InstallPlanList, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlanList{}, &operatorsv1alpha1.Subscription{}, nil
				},
			},
			Product:  &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder: setupRecorder(),
		},
		{
			Name:           "test subscription creating returns phase in progress",
			ExpectedStatus: integreatlyv1alpha1.PhaseInProgress,
			Installation:   basicInstallation(),
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, ns, operatorNS, federationNs, basicInstallation()),
			FakeConfig:     basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (*operatorsv1alpha1.InstallPlanList, *operatorsv1alpha1.Subscription, error) {
					return &operatorsv1alpha1.InstallPlanList{}, &operatorsv1alpha1.Subscription{}, nil
				},
			},
			Product:  &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder: setupRecorder(),
		},
		{
			Name:           "test components creating returns phase in progress",
			ExpectedStatus: integreatlyv1alpha1.PhaseInProgress,
			Installation:   basicInstallation(),
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, ns, operatorNS, federationNs, basicInstallation()),
			FakeConfig:     basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient k8sclient.Client, sub string, ns string) (*operatorsv1alpha1.InstallPlanList, *operatorsv1alpha1.Subscription, error) {
					return &operatorsv1alpha1.InstallPlanList{
							TypeMeta: metav1.TypeMeta{
								Kind:       "ApplicationMonitoring",
								APIVersion: monitoringv1.SchemeGroupVersion.String(),
							},
							ListMeta: metav1.ListMeta{},
						}, &operatorsv1alpha1.Subscription{
							Status: operatorsv1alpha1.SubscriptionStatus{
								Install: &operatorsv1alpha1.InstallPlanReference{
									Name: "monitoring-install-plan",
								},
							},
						}, nil
				},
			},
			Product:  &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder: setupRecorder(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM, tc.Recorder, getLogger())
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

func TestReconciler_reconcileAlertManagerConfigSecret(t *testing.T) {
	basicScheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}
	basicLogger := getLogger()
	basicReconciler := func() *Reconciler {
		return &Reconciler{
			installation: basicInstallation(),
			Log:          basicLogger,
			Config: &config.Monitoring{
				Config: map[string]string{
					"OPERATOR_NAMESPACE": defaultInstallationNamespace,
				},
			},
		}
	}

	installation := basicInstallation()

	smtpSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockSMTPSecretName,
			Namespace: installation.Namespace,
		},
		Data: map[string][]byte{
			"host":     []byte("smtp.sendgrid.com"),
			"port":     []byte("587"),
			"username": []byte("test"),
			"password": []byte("test"),
		},
		Type: corev1.SecretTypeOpaque,
	}

	pagerdutySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockPagerdutySecretName,
			Namespace: installation.Namespace,
		},
		Data: map[string][]byte{
			"serviceKey": []byte("test"),
		},
		Type: corev1.SecretTypeOpaque,
	}

	dmsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockDMSSecretName,
			Namespace: installation.Namespace,
		},
		Data: map[string][]byte{
			"url": []byte("https://example.com"),
		},
		Type: corev1.SecretTypeOpaque,
	}
	alertmanagerConfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      alertManagerConfigSecretName,
			Namespace: defaultInstallationNamespace,
		},
		Data: map[string][]byte{},
		Type: corev1.SecretTypeOpaque,
	}
	alertmanagerRoute := &v1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      alertManagerRouteName,
			Namespace: defaultInstallationNamespace,
		},
		Spec: v1.RouteSpec{
			Host: "example.com",
		},
	}

	clusterInfra := &configv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterInfraName,
		},
		Status: configv1.InfrastructureStatus{
			InfrastructureName: "cluster-infra",
		},
	}

	templateUtil := NewTemplateHelper(map[string]string{
		"SMTPHost":              string(smtpSecret.Data["host"]),
		"SMTPPort":              string(smtpSecret.Data["port"]),
		"SMTPFrom":              mockAlertFromAddress,
		"SMTPUsername":          string(smtpSecret.Data["username"]),
		"SMTPPassword":          string(smtpSecret.Data["password"]),
		"PagerDutyServiceKey":   string(pagerdutySecret.Data["serviceKey"]),
		"DeadMansSnitchURL":     string(dmsSecret.Data["url"]),
		"SMTPToCustomerAddress": mockCustomerAlertingEmailAddress,
		"SMTPToSREAddress":      mockAlertingEmailAddress,
		"SMTPToBUAddress":       mockBUAlertingEmailAddress,
		"Subject":               fmt.Sprintf(`[%s] {{template "email.default.subject" . }}`, clusterInfra.Status.InfrastructureName),
	})

	testSecretData, err := templateUtil.loadTemplate(alertManagerConfigTemplatePath)

	tests := []struct {
		name         string
		serverClient func() k8sclient.Client
		reconciler   func() *Reconciler
		setup        func() error
		want         integreatlyv1alpha1.StatusPhase
		wantFn       func(c k8sclient.Client) error
		wantErr      string
	}{
		{
			name: "fails when pager duty secret cannot be found",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme, smtpSecret, alertmanagerRoute)
			},
			reconciler: func() *Reconciler {
				return basicReconciler()
			},
			wantErr: "could not obtain pagerduty credentials secret: secrets \"test-pd\" not found",
			want:    integreatlyv1alpha1.PhaseFailed,
		},
		{
			name: "fails when pager duty service key is not defined",
			serverClient: func() k8sclient.Client {
				emptyPagerdutySecret := pagerdutySecret.DeepCopy()
				emptyPagerdutySecret.Data = map[string][]byte{}
				return fakeclient.NewFakeClientWithScheme(basicScheme, smtpSecret, emptyPagerdutySecret, alertmanagerRoute)
			},
			reconciler: func() *Reconciler {
				return basicReconciler()
			},
			wantErr: "secret key is undefined in pager duty secret",
			want:    integreatlyv1alpha1.PhaseFailed,
		},
		{
			name: "fails when dead mans snitch secret cannot be found",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme, smtpSecret, pagerdutySecret, alertmanagerRoute)
			},
			reconciler: func() *Reconciler {
				return basicReconciler()
			},
			wantErr: "could not obtain dead mans snitch credentials secret: secrets \"test-dms\" not found",
			want:    integreatlyv1alpha1.PhaseFailed,
		},
		{
			name: "fails when dead mans snitch url is not defined",
			serverClient: func() k8sclient.Client {
				emptyDMSSecret := dmsSecret.DeepCopy()
				emptyDMSSecret.Data = map[string][]byte{}
				return fakeclient.NewFakeClientWithScheme(basicScheme, smtpSecret, pagerdutySecret, emptyDMSSecret, alertmanagerRoute)
			},
			reconciler: func() *Reconciler {
				return basicReconciler()
			},
			wantErr: "url is undefined in dead mans snitch secret",
			want:    integreatlyv1alpha1.PhaseFailed,
		},
		{
			name: "awaiting components when alert manager route cannot be found",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme, smtpSecret, pagerdutySecret, dmsSecret)
			},
			reconciler: func() *Reconciler {
				return basicReconciler()
			},
			want: integreatlyv1alpha1.PhaseAwaitingComponents,
		},
		{
			name: "fails when alert manager route fails to be retrieved",
			serverClient: func() k8sclient.Client {
				return &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
						return fmt.Errorf("test")
					},
				}
			},
			reconciler: func() *Reconciler {
				return basicReconciler()
			},
			wantErr: "could not obtain alert manager route: test",
			want:    integreatlyv1alpha1.PhaseFailed,
		},
		{
			name: "fails cluster infra cannot  be retrieved",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme, smtpSecret, pagerdutySecret, dmsSecret, alertmanagerRoute)
			},
			reconciler: func() *Reconciler {
				return basicReconciler()
			},
			want: integreatlyv1alpha1.PhaseFailed,
		},
		{
			name: "secret created successfully",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme, smtpSecret, pagerdutySecret, dmsSecret, alertmanagerRoute, clusterInfra)
			},
			reconciler: func() *Reconciler {
				rec := basicReconciler()
				rec.installation = basicInstallationWithAlertEmailAddress()
				return rec
			},
			want: integreatlyv1alpha1.PhaseCompleted,
			wantFn: func(c k8sclient.Client) error {
				configSecret := &corev1.Secret{}
				if err := c.Get(context.TODO(), types.NamespacedName{Name: alertManagerConfigSecretName, Namespace: defaultInstallationNamespace}, configSecret); err != nil {
					return err
				}
				if !bytes.Equal(configSecret.Data[alertManagerConfigSecretFileName], testSecretData) {
					return fmt.Errorf("secret data is not equal, got = %v,\n want = %v", string(configSecret.Data[alertManagerConfigSecretFileName]), string(testSecretData))
				}
				return nil
			},
		},
		{
			name: "secret data is overridden if already exists",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme, smtpSecret, pagerdutySecret, dmsSecret, alertmanagerRoute, alertmanagerConfigSecret, clusterInfra)
			},
			reconciler: func() *Reconciler {
				rec := basicReconciler()
				rec.installation = basicInstallationWithAlertEmailAddress()
				return rec
			},
			want: integreatlyv1alpha1.PhaseCompleted,
			wantFn: func(c k8sclient.Client) error {
				configSecret := &corev1.Secret{}
				if err := c.Get(context.TODO(), types.NamespacedName{Name: alertManagerConfigSecretName, Namespace: defaultInstallationNamespace}, configSecret); err != nil {
					return err
				}
				if !bytes.Equal(configSecret.Data[alertManagerConfigSecretFileName], testSecretData) {
					return fmt.Errorf("secret data is not equal, got = %v,\n want = %v", string(configSecret.Data[alertManagerConfigSecretFileName]), string(testSecretData))
				}
				return nil
			},
		},
		{
			name: "alert address env override is successful",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme, smtpSecret, pagerdutySecret, dmsSecret, alertmanagerRoute, clusterInfra)
			},
			reconciler: func() *Reconciler {
				reconciler := basicReconciler()
				reconciler.installation = basicInstallationWithAlertEmailAddress()
				return reconciler
			},
			want: integreatlyv1alpha1.PhaseCompleted,
			wantFn: func(c k8sclient.Client) error {
				configSecret := &corev1.Secret{}
				if err := c.Get(context.TODO(), types.NamespacedName{Name: alertManagerConfigSecretName, Namespace: defaultInstallationNamespace}, configSecret); err != nil {
					return err
				}
				templateUtil := NewTemplateHelper(map[string]string{
					"SMTPHost":              string(smtpSecret.Data["host"]),
					"SMTPPort":              string(smtpSecret.Data["port"]),
					"SMTPFrom":              mockAlertFromAddress,
					"SMTPUsername":          string(smtpSecret.Data["username"]),
					"SMTPPassword":          string(smtpSecret.Data["password"]),
					"PagerDutyServiceKey":   string(pagerdutySecret.Data["serviceKey"]),
					"DeadMansSnitchURL":     string(dmsSecret.Data["url"]),
					"SMTPToCustomerAddress": mockCustomerAlertingEmailAddress,
					"SMTPToSREAddress":      mockAlertingEmailAddress,
					"SMTPToBUAddress":       mockBUAlertingEmailAddress,
					"Subject":               fmt.Sprintf(`[%s] {{template "email.default.subject" . }}`, clusterInfra.Status.InfrastructureName),
				})

				testSecretData, err := templateUtil.loadTemplate(alertManagerConfigTemplatePath)
				if err != nil {
					return err
				}
				if !bytes.Equal(configSecret.Data[alertManagerConfigSecretFileName], testSecretData) {
					return fmt.Errorf("secret data is not equal, got = %v,\n want = %v", string(configSecret.Data[alertManagerConfigSecretFileName]), string(testSecretData))
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			reconciler := tt.reconciler()
			serverClient := tt.serverClient()

			got, err := reconciler.reconcileAlertManagerConfigSecret(context.TODO(), serverClient)
			if tt.wantErr != "" && err.Error() != tt.wantErr {
				t.Errorf("reconcileAlertManagerConfigSecret() error = %v, wantErr %v", err.Error(), tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("reconcileAlertManagerConfigSecret() got = %v, want %v", got, tt.want)
			}
			if tt.wantFn != nil {
				if err := tt.wantFn(serverClient); err != nil {
					t.Errorf("reconcileAlertManagerConfigSecret() error = %v", err)
				}
			}
		})
	}
}

func TestReconciler_getPagerDutySecret(t *testing.T) {
	basicScheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}
	basicReconciler := &Reconciler{
		installation: basicInstallation(),
		Log:          getLogger(),
		Config: &config.Monitoring{
			Config: map[string]string{
				"OPERATOR_NAMESPACE": defaultInstallationNamespace,
			},
		},
	}

	installation := basicInstallation()

	pagerdutySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockPagerdutySecretName,
			Namespace: installation.Namespace,
		},
		Data: map[string][]byte{
			"PAGERDUTY_KEY": []byte("test"),
		},
		Type: corev1.SecretTypeOpaque,
	}

	tests := []struct {
		name         string
		serverClient func() k8sclient.Client
		reconciler   func() *Reconciler
		setup        func() error
		want         string
		wantErr      string
	}{
		{
			name: "fails when pager duty secret cannot be found",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme)
			},
			reconciler: func() *Reconciler {
				return basicReconciler
			},
			wantErr: "could not obtain pagerduty credentials secret: secrets \"test-pd\" not found",
		},
		{
			name: "fails when pager duty service key is not defined",
			serverClient: func() k8sclient.Client {
				emptyPagerdutySecret := pagerdutySecret.DeepCopy()
				emptyPagerdutySecret.Data = map[string][]byte{}
				return fakeclient.NewFakeClientWithScheme(basicScheme, emptyPagerdutySecret)
			},
			reconciler: func() *Reconciler {
				return basicReconciler
			},
			wantErr: "secret key is undefined in pager duty secret",
		},

		{
			name: "fails when pager duty service key - value is not defined",
			serverClient: func() k8sclient.Client {
				emptyPagerdutySecret := pagerdutySecret.DeepCopy()
				emptyPagerdutySecret.Data = map[string][]byte{}
				emptyPagerdutySecret.Data["serviceKey"] = []byte("")
				return fakeclient.NewFakeClientWithScheme(basicScheme, emptyPagerdutySecret)
			},
			reconciler: func() *Reconciler {
				return basicReconciler
			},
			wantErr: "secret key is undefined in pager duty secret",
		},
		{
			name: "secret read successfully - from pager duty operator secret",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme, pagerdutySecret)
			},
			reconciler: func() *Reconciler {
				return basicReconciler
			},
			want: "test",
		},
		{
			name: "secret read successfully - from cssre pager duty operator secret",
			serverClient: func() k8sclient.Client {
				cssrePagerDutySecret := pagerdutySecret.DeepCopy()
				cssrePagerDutySecret.Data = make(map[string][]byte, 0)
				cssrePagerDutySecret.Data["serviceKey"] = []byte("cssre-pg-secret")
				return fakeclient.NewFakeClientWithScheme(basicScheme, cssrePagerDutySecret)
			},
			reconciler: func() *Reconciler {
				return basicReconciler
			},
			want: "cssre-pg-secret",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			reconciler := tt.reconciler()
			serverClient := tt.serverClient()

			got, err := reconciler.getPagerDutySecret(context.TODO(), serverClient)
			if tt.wantErr != "" && err.Error() != tt.wantErr {
				t.Errorf("getPagerDutySecret() error = %v, wantErr %v", err.Error(), tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getPagerDutySecret() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_getDMSSecret(t *testing.T) {
	basicScheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}
	basicReconciler := &Reconciler{
		installation: basicInstallation(),
		Log:          getLogger(),
		Config: &config.Monitoring{
			Config: map[string]string{
				"OPERATOR_NAMESPACE": defaultInstallationNamespace,
			},
		},
	}

	installation := basicInstallation()

	dmsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockDMSSecretName,
			Namespace: installation.Namespace,
		},
		Data: map[string][]byte{
			"SNITCH_URL": []byte("https://example.com"),
		},
		Type: corev1.SecretTypeOpaque,
	}

	tests := []struct {
		name         string
		serverClient func() k8sclient.Client
		reconciler   func() *Reconciler
		setup        func() error
		want         string
		wantErr      string
	}{
		{
			name: "fails when dead man switch secret cannot be found",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme)
			},
			reconciler: func() *Reconciler {
				return basicReconciler
			},
			wantErr: "could not obtain dead mans snitch credentials secret: secrets \"test-dms\" not found",
		},
		{
			name: "fails when dead man switch secret url is not defined",
			serverClient: func() k8sclient.Client {
				emptyDMSSecret := dmsSecret.DeepCopy()
				emptyDMSSecret.Data = map[string][]byte{}
				return fakeclient.NewFakeClientWithScheme(basicScheme, emptyDMSSecret)
			},
			reconciler: func() *Reconciler {
				return basicReconciler
			},
			wantErr: "url is undefined in dead mans snitch secret",
		},

		{
			name: "fails when dead man switch secret SNITCHH_URL - value is not defined",
			serverClient: func() k8sclient.Client {
				emptyDMSSecret := dmsSecret.DeepCopy()
				emptyDMSSecret.Data = map[string][]byte{}
				emptyDMSSecret.Data["SNITCH_URL"] = []byte("")
				return fakeclient.NewFakeClientWithScheme(basicScheme, emptyDMSSecret)
			},
			reconciler: func() *Reconciler {
				return basicReconciler
			},
			wantErr: "url is undefined in dead mans snitch secret",
		},
		{
			name: "secret read successfully - from dead man switch operator secret",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(basicScheme, dmsSecret)
			},
			reconciler: func() *Reconciler {
				return basicReconciler
			},
			want: "https://example.com",
		},
		{
			name: "secret read successfully - from cssre dead man switch operator secret",
			serverClient: func() k8sclient.Client {
				cssreDMSSecret := dmsSecret.DeepCopy()
				cssreDMSSecret.Data = make(map[string][]byte, 0)
				cssreDMSSecret.Data["url"] = []byte("https://example-cssredms-secret.com")
				return fakeclient.NewFakeClientWithScheme(basicScheme, cssreDMSSecret)
			},
			reconciler: func() *Reconciler {
				return basicReconciler
			},
			want: "https://example-cssredms-secret.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			reconciler := tt.reconciler()
			serverClient := tt.serverClient()

			got, err := reconciler.getDMSSecret(context.TODO(), serverClient)
			if tt.wantErr != "" && err.Error() != tt.wantErr {
				t.Errorf("getDMSSecret() error = %v, wantErr %v", err.Error(), tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getDMSSecret() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.ProductApicurioRegistry})
}
