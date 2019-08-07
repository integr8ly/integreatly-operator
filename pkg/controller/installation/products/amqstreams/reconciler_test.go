package amqstreams

import (
	"context"
	"fmt"
	"testing"

	threescalev1 "github.com/integr8ly/integreatly-operator/pkg/apis/3scale/v1alpha1"
	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	kafkav1 "github.com/integr8ly/integreatly-operator/pkg/apis/kafka.strimzi.io/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	operatorsv1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func basicConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		ReadAMQStreamsFunc: func() (ready *config.AMQStreams, e error) {
			return config.NewAMQStreams(config.ProductConfig{}), nil
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
	err = integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	err = operatorsv1alpha1.AddToScheme(scheme)
	err = marketplacev1.SchemeBuilder.AddToScheme(scheme)
	err = corev1.SchemeBuilder.AddToScheme(scheme)
	err = coreosv1.SchemeBuilder.AddToScheme(scheme)
	err = kafkav1.SchemeBuilder.AddToScheme(scheme)
	return scheme, err
}

func TestReconciler_config(t *testing.T) {
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
			ExpectedError:  "could not read amq streams config: could not read amq streams config",
			Installation:   &v1alpha1.Installation{},
			FakeClient:     fakeclient.NewFakeClient(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadAMQStreamsFunc: func() (ready *config.AMQStreams, e error) {
					return nil, errors.New("could not read amq streams config")
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
				InstallOperatorFunc: func(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, os operatorsv1.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {
					return errors.New("dummy")
				},
			},
			FakeClient: fakeclient.NewFakeClient(),
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
	scheme := runtime.NewScheme()
	kafkav1.SchemeBuilder.AddToScheme(scheme)

	cases := []struct {
		Name           string
		FakeClient     client.Client
		FakeConfig     *config.ConfigReadWriterMock
		Installation   *v1alpha1.Installation
		ExpectError    bool
		ExpectedError  string
		ExpectedStatus v1alpha1.StatusPhase
		FakeMPM        *marketplace.MarketplaceInterfaceMock
	}{
		{
			Name:       "Test reconcile custom resource returns completed when successful created",
			FakeClient: fakeclient.NewFakeClientWithScheme(scheme),
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
			Name: "Test reconcile custom resource returns failed on unsuccessful create",
			FakeClient: &moqclient.SigsClientInterfaceMock{
				CreateFunc: func(ctx context.Context, obj runtime.Object) error {
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
			ExpectError:    true,
			ExpectedError:  "failed to get or create a kafka custom resource",
			ExpectedStatus: v1alpha1.PhaseFailed,
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
			phase, err := reconciler.handleCreatingComponents(context.TODO(), tc.FakeClient, tc.Installation)
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

	kafka := &kafkav1.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "integreatly-cluster",
			Namespace: defaultInstallationNamespace,
		},
	}

	unreadyPods := []runtime.Object{}
	for i := 0; i < 8; i++ {
		unreadyPods = append(unreadyPods, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%d", defaultSubscriptionName, i),
				Namespace: defaultInstallationNamespace,
			},
			Status: corev1.PodStatus{
				Conditions: []corev1.PodCondition{
					corev1.PodCondition{
						Type:   corev1.ContainersReady,
						Status: corev1.ConditionUnknown,
					},
				},
			},
		})
	}

	readyPods := []runtime.Object{}
	for i := 0; i < 8; i++ {
		readyPods = append(readyPods, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%d", defaultSubscriptionName, i),
				Namespace: defaultInstallationNamespace,
			},
			Status: corev1.PodStatus{
				Conditions: []corev1.PodCondition{
					corev1.PodCondition{
						Type:   corev1.ContainersReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		})
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
			Name:           "test failure to list pods",
			ExpectedStatus: v1alpha1.PhaseFailed,
			ExpectedError:  "failed to check amq streams installation",
			ExpectError:    true,
			FakeClient: &moqclient.SigsClientInterfaceMock{
				ListFunc: func(ctx context.Context, opts *client.ListOptions, list runtime.Object) error {
					return errors.New("dummy create error")
				},
			},
			FakeConfig:   basicConfigMock(),
			Installation: &v1alpha1.Installation{},
		},
		{
			Name:           "test incomplete amount of pods returns phase in progress",
			ExpectedStatus: v1alpha1.PhaseInProgress,
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme),
			FakeConfig:     basicConfigMock(),
			Installation:   &v1alpha1.Installation{},
		},
		{
			Name:           "test unready pods returns phase in progress",
			ExpectedStatus: v1alpha1.PhaseInProgress,
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, unreadyPods...),
			FakeConfig:     basicConfigMock(),
			Installation:   &v1alpha1.Installation{},
		},
		{
			Name:           "test ready pods returns phase complete",
			ExpectedStatus: v1alpha1.PhaseCompleted,
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, append(readyPods, kafka)...),
			FakeConfig:     basicConfigMock(),
			Installation:   &v1alpha1.Installation{},
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

			status, err := testReconciler.handleProgressPhase(context.TODO(), tc.FakeClient)

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

	objs := []runtime.Object{}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultInstallationNamespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       "installation",
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
				},
			},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}
	objs = append(objs, ns)

	for i := 0; i < 8; i++ {
		objs = append(objs, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%d", defaultSubscriptionName, i),
				Namespace: defaultInstallationNamespace,
			},
			Status: corev1.PodStatus{
				Conditions: []corev1.PodCondition{
					corev1.PodCondition{
						Type:   corev1.ContainersReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		})
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
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, objs...),
			FakeConfig:     basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, os operatorsv1.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient client.Client, subName string, ns string) (plan *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlan{
							ObjectMeta: metav1.ObjectMeta{
								Name: "amq-install-plan",
							},
							Status: operatorsv1alpha1.InstallPlanStatus{
								Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
							},
						}, &operatorsv1alpha1.Subscription{
							Status: operatorsv1alpha1.SubscriptionStatus{
								Install: &operatorsv1alpha1.InstallPlanReference{
									Name: "amq-install-plan",
								},
							},
						}, nil
				},
			},
			Installation: &v1alpha1.Installation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "installation",
					Namespace: defaultInstallationNamespace,
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "installation",
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
				},
			},
			Product: &v1alpha1.InstallationProductStatus{},
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
