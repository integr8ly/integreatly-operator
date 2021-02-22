package monitoringspec

import (
	"context"
	"errors"
	"testing"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"

	prometheusmonitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	projectv1 "github.com/openshift/api/project/v1"

	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"

	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	mockSMTPSecretName      = "test-smtp"
	mockPagerdutySecretName = "test-pd"
	mockDMSSecretName       = "test-dms"
)

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
		},
	}
}

func createServicemonitor(name, namespace string) *prometheusmonitoringv1.ServiceMonitor {
	return &prometheusmonitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: prometheusmonitoringv1.ServiceMonitorSpec{
			Endpoints: []prometheusmonitoringv1.Endpoint{
				{
					Port:   "upstream",
					Path:   "/name",
					Scheme: "http",
					Params: map[string][]string{
						"match[]": []string{"{__name__=\"ALERTS\",alertstate=\"firing\"}"},
					},
					Interval:      "30s",
					ScrapeTimeout: "30s",
					HonorLabels:   true,
				},
			},
		},
	}
}

func createRoleBinding(name, namespace string) *rbac.RoleBinding {
	roleBinding := &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Subjects: []rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      clusterMonitoringPrometheusServiceAccount,
				Namespace: clusterMonitoringNamespace,
			},
		},
		RoleRef: rbac.RoleRef{
			APIGroup: roleRefAPIGroup,
			Kind:     bundle.RoleKind,
			Name:     roleRefName,
		},
	}
	return roleBinding
}

func createRole(name, namespace string) *rbac.Role {

	resources := []string{
		"services",
		"endpoints",
		"pods",
	}

	verbs := []string{
		"get",
		"list",
		"watch",
	}
	apiGroups := []string{""}

	role := &rbac.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleBindingName,
			Namespace: namespace,
		},
	}
	role.Rules = []rbac.PolicyRule{
		{
			APIGroups: apiGroups,
			Resources: resources,
			Verbs:     verbs,
		},
	}
	return role
}
func basicConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		ReadMonitoringSpecFunc: func() (ready *config.MonitoringSpec, e error) {
			return config.NewMonitoringSpec(config.ProductConfig{}), nil
		},
		WriteConfigFunc: func(config config.ConfigReadable) error {
			return nil
		},
	}
}

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := projectv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := prometheusmonitoringv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := rbac.AddToScheme(scheme); err != nil {
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
				ReadMonitoringSpecFunc: func() (ready *config.MonitoringSpec, e error) {
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
				ReadMonitoringSpecFunc: func() (ready *config.MonitoringSpec, e error) {
					return config.NewMonitoringSpec(config.ProductConfig{
						"NAMESPACE": "",
					}), nil
				},
			},
			Recorder: setupRecorder(),
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

// Test case - creates a monitoring and fuse namespaces
// Creates a servicemonitor in  fuse namespace
// Verifies that the service monitor is cloned in the monitoring namespace
// Verifies that a rolebinding is created in the fuse namespace
func TestReconciler_fullReconcile(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}
	// initialise runtime objects

	//Monitoring namespace
	monitoringns := &corev1.Namespace{
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
	//Fuse namespace
	fusens := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fuse",
			Labels: map[string]string{
				resources.OwnerLabelKey: string(basicInstallation().GetUID()),
				"monitoring-key":        "middleware",
			},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}
	//Service monitor inside fuse namespace
	fusesm := createServicemonitor("fuse-servicemon", "fuse")

	installation := basicInstallation()

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
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, installation, monitoringns, fusens, fusesm),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadMonitoringSpecFunc: func() (ready *config.MonitoringSpec, e error) {
					return config.NewMonitoringSpec(config.ProductConfig{
						"NAMESPACE":          "",
						"OPERATOR_NAMESPACE": defaultInstallationNamespace,
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
							ObjectMeta: metav1.ObjectMeta{
								Name: "monitoring-install-plan",
							},
							Status: operatorsv1alpha1.InstallPlanStatus{
								Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
							},
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

			ctx := context.TODO()
			//Verify that reconcilation was completed successfuly
			status, err := reconciler.Reconcile(ctx, tc.Installation, tc.Product, tc.FakeClient)
			if err != nil && !tc.ExpectError {
				t.Fatalf("expected no error but got one: %v", err)
			}
			if err == nil && tc.ExpectError {
				t.Fatal("expected error but got none")
			}
			if status != tc.ExpectedStatus {
				t.Fatalf("Expected status: '%v', got: '%v'", tc.ExpectedStatus, status)
			}
			//Verify that a new servicemonitor is created in the namespace
			sermon := &prometheusmonitoringv1.ServiceMonitor{}
			err = tc.FakeClient.Get(ctx, k8sclient.ObjectKey{Name: "fuse-fuse-servicemon", Namespace: defaultInstallationNamespace}, sermon)
			if err != nil {
				t.Fatalf("expected no error but got one: %v", err)
			}
			//Verify that a role binding was created in the fuse namespace
			rb := &rbac.RoleBinding{}
			err = tc.FakeClient.Get(ctx, k8sclient.ObjectKey{Name: roleBindingName, Namespace: "fuse"}, rb)
			if err != nil {
				t.Fatalf("expected no error but got one: %v", err)
			}
			//Verify that a role was created in the fuse namespace
			role := &rbac.Role{}
			err = tc.FakeClient.Get(ctx, k8sclient.ObjectKey{Name: roleRefName, Namespace: "fuse"}, role)
			if err != nil {
				t.Fatalf("expected no error but got one: %v", err)
			}

		})
	}
}

// Test case - creates a monitoring and fuse namespaces
// Creates a rolebinding in  fuse namespace - stale
// Creates a servicemonitor in the monitoring namespace - stale
// Verifies that the service monitor is removed in the monitoring namespace
// Verifies that a rolebinding is removed in the fuse namespace
func TestReconciler_fullReconcileWithCleanUp(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}
	// initialise runtime objects

	//Monitoring namespace
	monitoringns := &corev1.Namespace{
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

	//Fuse namespace
	fusens := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fuse",
			Labels: map[string]string{
				resources.OwnerLabelKey: string(basicInstallation().GetUID()),
				"monitoring-key":        "middleware",
			},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}
	//Create a UPS servicemonitor in just monitoring namespace - stale one
	upssm := createServicemonitor("ups-servicemon", defaultInstallationNamespace)

	//Create a rolebinding in fuse namespace
	rb := createRoleBinding(roleBindingName, "fuse")
	role := createRole(roleRefName, "fuse")
	if len(upssm.Labels) == 0 {
		upssm.Labels = make(map[string]string)
	}
	upssm.Labels[clonedServiceMonitorLabelKey] = clonedServiceMonitorLabelValue

	installation := basicInstallation()

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
			Name:           "test successful reconcile with cleanup",
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, installation, monitoringns, upssm, fusens, rb, role),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadMonitoringSpecFunc: func() (ready *config.MonitoringSpec, e error) {
					return config.NewMonitoringSpec(config.ProductConfig{
						"NAMESPACE":          "",
						"OPERATOR_NAMESPACE": defaultInstallationNamespace,
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
							ObjectMeta: metav1.ObjectMeta{
								Name: "monitoring-install-plan",
							},
							Status: operatorsv1alpha1.InstallPlanStatus{
								Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
							},
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

			ctx := context.TODO()

			//Verify that the sm exisits in monitoring namespace
			sermon := &prometheusmonitoringv1.ServiceMonitor{}
			err = tc.FakeClient.Get(ctx, k8sclient.ObjectKey{Name: "ups-servicemon", Namespace: defaultInstallationNamespace}, sermon)
			if err != nil {
				t.Fatalf("expected no error but got one: %v", err)
			}

			//Verify fuse namespace has a stale rolebinding
			rb := &rbac.RoleBinding{}
			err = tc.FakeClient.Get(ctx, k8sclient.ObjectKey{Name: roleBindingName, Namespace: "fuse"}, rb)
			if err != nil {
				t.Fatalf("expected no error but got one: %v", err)
			}

			//Verify that the fuse namespace has a stale role
			role := &rbac.Role{}
			err = tc.FakeClient.Get(ctx, k8sclient.ObjectKey{Name: roleRefName, Namespace: "fuse"}, role)
			if err != nil {
				t.Fatalf("expected no error but got one: %v", err)
			}

			//Verify that reconcilation was completed successfuly
			status, err := reconciler.Reconcile(ctx, tc.Installation, tc.Product, tc.FakeClient)
			if err != nil && !tc.ExpectError {
				t.Fatalf("expected no error but got one: %v", err)
			}
			if err == nil && tc.ExpectError {
				t.Fatal("expected error but got none")
			}
			if status != tc.ExpectedStatus {
				t.Fatalf("Expected status: '%v', got: '%v'", tc.ExpectedStatus, status)
			}
			//Verify that the stale servicemonitor is removed
			sermon = &prometheusmonitoringv1.ServiceMonitor{}
			err = tc.FakeClient.Get(ctx, k8sclient.ObjectKey{Name: "ups-servicemon", Namespace: defaultInstallationNamespace}, sermon)
			if err != nil && !k8serr.IsNotFound(err) {
				t.Fatalf("expected no error but got one: %v", err)
			}
			//Verify that the stale rolebinding is removed
			rb = &rbac.RoleBinding{}
			err = tc.FakeClient.Get(ctx, k8sclient.ObjectKey{Name: roleBindingName, Namespace: "fuse"}, rb)
			if err != nil && !k8serr.IsNotFound(err) {
				t.Fatalf("expected no error but got one: %v", err)
			}

			//Verify that the stale role is removed
			role = &rbac.Role{}
			err = tc.FakeClient.Get(ctx, k8sclient.ObjectKey{Name: roleBindingName, Namespace: "fuse"}, role)
			if err != nil && !k8serr.IsNotFound(err) {
				t.Fatalf("expected no error but got one: %v", err)
			}
		})
	}
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.ProductMonitoringSpec})
}
