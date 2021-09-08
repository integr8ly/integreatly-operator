package monitoring

import (
	"context"
	"errors"
	"testing"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"

	grafanav1alpha1 "github.com/integr8ly/grafana-operator/v3/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	v1 "github.com/openshift/api/route/v1"

	prometheusmonitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"

	monitoringv1 "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	projectv1 "github.com/openshift/api/project/v1"

	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

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

var localProductDeclaration = marketplace.LocalProductDeclaration("integreatly-monitoring")

func basicInstallation() *integreatlyv1alpha1.RHMI {
	return &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "installation",
			Namespace: defaultInstallationNamespace,
			UID:       types.UID("xyz"),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "RHMI",
			APIVersion: integreatlyv1alpha1.GroupVersion.String(),
		},
		Spec: integreatlyv1alpha1.RHMISpec{
			SMTPSecret:           mockSMTPSecretName,
			PagerDutySecret:      mockPagerdutySecretName,
			DeadMansSnitchSecret: mockDMSSecretName,
			Type:                 string(integreatlyv1alpha1.InstallationTypeManaged),
		},
	}
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
	if err := routev1.AddToScheme(scheme); err != nil {
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
			_, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM, tc.Recorder, getLogger(), localProductDeclaration)
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
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM, tc.Recorder, getLogger(), localProductDeclaration)
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

	clusterVersion := &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterIDValue,
		},
		Spec: configv1.ClusterVersionSpec{
			ClusterID: "cluster-id",
		},
	}

	clusterRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      openShiftConsoleRoute,
			Namespace: openShiftConsoleNamespace,
		},
		Spec: routev1.RouteSpec{
			Host: "example-console.com",
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
				dmsSecret, alertmanagerRoute, grafana, clusterInfra, clusterVersion, clusterRoute),
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
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plan *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlan{
							TypeMeta: metav1.TypeMeta{
								Kind:       "ApplicationMonitoring",
								APIVersion: monitoringv1.SchemeGroupVersion.String(),
							},
							// Items: []operatorsv1alpha1.InstallPlan{
							// 	{
							ObjectMeta: metav1.ObjectMeta{
								Name: "monitoring-install-plan",
							},
							Status: operatorsv1alpha1.InstallPlanStatus{
								Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
							},
							// },
							// },
							// ListMeta: metav1.ListMeta{},
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
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM, tc.Recorder, getLogger(), localProductDeclaration)
			if err != nil && err.Error() != tc.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
			}

			status, err := reconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.FakeClient, &quota.ProductConfigMock{})
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
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plan *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
					return nil, &operatorsv1alpha1.Subscription{}, nil
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
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (*operatorsv1alpha1.InstallPlan, *operatorsv1alpha1.Subscription, error) {
					return nil, &operatorsv1alpha1.Subscription{}, nil
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
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, sub string, ns string) (*operatorsv1alpha1.InstallPlan, *operatorsv1alpha1.Subscription, error) {
					return nil, &operatorsv1alpha1.Subscription{
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
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM, tc.Recorder, getLogger(), localProductDeclaration)
			if err != nil {
				t.Fatalf("unexpected error : '%v'", err)
			}

			status, err := reconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.FakeClient, &quota.ProductConfigMock{})
			if err != nil {
				t.Fatalf("expected no error but got one: %v", err)
			}
			if status != tc.ExpectedStatus {
				t.Fatalf("Expected status: '%v', got: '%v'", tc.ExpectedStatus, status)
			}
		})
	}
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.ProductApicurioRegistry})
}
