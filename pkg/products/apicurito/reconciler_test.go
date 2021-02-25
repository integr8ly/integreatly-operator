package apicurito

import (
	"context"
	"testing"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	apicuritov1alpha1 "github.com/apicurio/apicurio-operators/apicurito/pkg/apis/apicur/v1alpha1"
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	appsv1 "github.com/openshift/api/apps/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	defaultOperandNamespace = "apicurito"
)

func TestReconciler_fullReconcile(t *testing.T) {

	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	installation := getInstallation()

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultOperandNamespace,
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
			Name: defaultOperandNamespace + "-operator",
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
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, installation, getApicuritoCr(), ns, operatorNS, getSecret(), getRoute(), getDeploymentConfig(), getDeployment(), getPodsComplete()),
			FakeConfig:     basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlan{
							ObjectMeta: metav1.ObjectMeta{
								Name: "apicurito-install-plan",
							},
							Status: operatorsv1alpha1.InstallPlanStatus{
								Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
							},
						}, &operatorsv1alpha1.Subscription{
							Status: operatorsv1alpha1.SubscriptionStatus{
								Install: &operatorsv1alpha1.InstallPlanReference{
									Name: "apicurito-install-plan",
								},
							},
						}, nil
				},
			},
			Installation: installation,
			Product:      &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:     setupRecorder(),
		},
		{
			Name:           "test failed reconcile, no namespace created",
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			ExpectError:    true,
			ExpectedError:  "failed to create apicurito namespace: could not retrieve apicurito namespace: namespaces \"apicurito\" not found",
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, installation, getApicuritoCr(), getSecret(), getRoute(), getDeploymentConfig(), getDeployment(), getPodsComplete()),
			FakeConfig:     basicConfigMock(),
			Installation:   installation,
			Product:        &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:       setupRecorder(),
		},
		{
			Name:           "test failed reconcile, no pull secret",
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			ExpectError:    true,
			ExpectedError:  "failed to create apicurito namespace: could not retrieve apicurito namespace: namespaces \"apicurito\" not found",
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, installation, getApicuritoCr(), ns),
			FakeConfig:     basicConfigMock(),
			Installation:   installation,
			Product:        &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:       setupRecorder(),
		},
		{
			Name:           "test failed reconcile, no install plans",
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			ExpectError:    true,
			ExpectedError:  "failed to create install plans",
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, installation, getApicuritoCr(), ns),
			FakeConfig:     basicConfigMock(),
			Installation:   installation,
			Product:        &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:       setupRecorder(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
					return nil, nil, k8serr.NewNotFound(schema.GroupResource{}, "subs")
				},
			},
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
				t.Fatalf("expected no errors, but got one: %v", err)
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

func TestReconciler_handleProgress(t *testing.T) {

	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	installation := getInstallation()

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
			Name:           "test reconcile still in progress",
			ExpectedStatus: integreatlyv1alpha1.PhaseInProgress,
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, installation, getPodsNotComplete()),
			FakeConfig:     basicConfigMock(),
			Installation:   installation,
			Product:        &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:       setupRecorder(),
		},
		{
			Name:           "test no pods in namespace",
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			ExpectError:    true,
			ExpectedError:  "failed to check apicurito installation:",
			FakeClient: &moqclient.SigsClientInterfaceMock{
				ListFunc: func(ctx context.Context, list runtime.Object, opts ...k8sclient.ListOption) error {
					return k8serr.NewNotFound(schema.GroupResource{}, "pods")
				},
			},
			FakeConfig: basicConfigMock(),
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

			status, err := testReconciler.handleProgressPhase(context.TODO(), tc.FakeClient)

			if err != nil && !tc.ExpectError {
				t.Fatalf("expected no errors, but got one: %v", err)
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

func getSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pull-secret",
			Namespace: "openshift-config",
		},
		Data: map[string][]byte{
			"credentialKeyID":     []byte("test"),
			"credentialSecretKey": []byte("test"),
		},
	}
}

func getDeploymentConfig() *appsv1.DeploymentConfig {
	return &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fuse-apicurito-generator",
			Namespace: defaultOperandNamespace,
		},
	}
}

func getDeployment() *v1.Deployment {
	return &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "apicurito",
			Namespace: defaultOperandNamespace,
		},
		Spec: v1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes:    []corev1.Volume{},
					Containers: []corev1.Container{{}},
				},
			},
		},
	}
}

func getRoute() *routev1.Route {
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "apicurito",
			Namespace: defaultOperandNamespace,
		},
		Spec: routev1.RouteSpec{
			Host: "sampleHost",
		},
	}
}

func getApicuritoCr() *apicuritov1alpha1.Apicurito {

	apicuritoCR := &apicuritov1alpha1.Apicurito{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apicuritoName,
			Namespace: defaultOperandNamespace,
		},
	}
	return apicuritoCR
}

func basicConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		GetOperatorNamespaceFunc: func() string {
			return defaultOperandNamespace + "-operator"
		},
		ReadApicuritoFunc: func() (apicurito *config.Apicurito, err error) {
			return config.NewApicurito(config.ProductConfig{
				"NAMESPACE": defaultOperandNamespace,
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

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
}

func getPodsNotComplete() *corev1.PodList {
	return &corev1.PodList{
		Items: []corev1.Pod{
			getPodNotReady("pod1"),
		},
	}
}

func getPodsComplete() *corev1.PodList {
	return &corev1.PodList{
		Items: []corev1.Pod{
			getPodReady("pod1"),
			getPodReady("pod2"),
		},
	}
}

func getPodReady(name string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: defaultOperandNamespace,
		},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.ContainersReady,
					Status: corev1.ConditionStatus("True"),
				},
			},
		},
	}
}

func getPodNotReady(name string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: defaultOperandNamespace,
		},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.ContainersReady,
					Status: corev1.ConditionStatus("False"),
				},
			},
		},
	}
}

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = operatorsv1alpha1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = apicuritov1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = corev1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = coreosv1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = v1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = v1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = appsv1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = routev1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = projectv1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = monitoringv1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	return scheme, err
}

func getInstallation() *integreatlyv1alpha1.RHMI {
	return &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "installation",
			Namespace:  defaultOperandNamespace,
			Finalizers: []string{"finalizer.apicurito.integreatly.org"},
			UID:        types.UID("xyz"),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "RHMI",
			APIVersion: integreatlyv1alpha1.GroupVersion.String(),
		},
		Status: integreatlyv1alpha1.RHMIStatus{
			Stages: map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus{
				"apicurito-stage": {
					Name: "apicurito-stage",
					Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
						integreatlyv1alpha1.ProductApicurito: {
							Name:   integreatlyv1alpha1.ProductApicurito,
							Status: integreatlyv1alpha1.PhaseCreatingComponents,
						},
					},
				},
			},
		},
	}
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.ProductApicurito})
}
