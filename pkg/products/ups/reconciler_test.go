package ups

import (
	"context"
	"errors"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"testing"

	upsv1alpha1 "github.com/aerogear/unifiedpush-operator/pkg/apis/push/v1alpha1"
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	croTypes "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"

	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := upsv1alpha1.SchemeBuilder.AddToScheme(scheme)
	err = routev1.AddToScheme(scheme)
	err = crov1.SchemeBuilder.AddToScheme(scheme)
	err = monitoringv1.AddToScheme(scheme)
	projectv1.AddToScheme(scheme)
	return scheme, err
}

func basicConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		ReadUpsFunc: func() (ups *config.Ups, e error) {
			return config.NewUps(config.ProductConfig{}), nil
		},
		WriteConfigFunc: func(config config.ConfigReadable) error {
			return nil
		},
	}
}

func errorConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		ReadUpsFunc: func() (ups *config.Ups, e error) {
			return config.NewUps(config.ProductConfig{}), nil
		},
		WriteConfigFunc: func(config config.ConfigReadable) error {
			return errors.New("some error")
		},
	}
}

func basicRouteMock() *routev1.Route {
	return &routev1.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Route",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ups",
			Name:      defaultRoutename,
		},
	}
}

func mockUpsCRWithStatus(phase upsv1alpha1.StatusPhase) *upsv1alpha1.UnifiedPushServer {
	return &upsv1alpha1.UnifiedPushServer{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ups",
			Name:      "ups",
		},
		Status: upsv1alpha1.UnifiedPushServerStatus{
			Phase: phase,
		},
	}
}

func getTestPostgres() *crov1.Postgres {
	return &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ups-postgres-test",
			Namespace: "ups",
		},
		Status: crov1.PostgresStatus{
			Phase: croTypes.PhaseComplete,
			SecretRef: &croTypes.SecretRef{
				Name:      "test-postgres",
				Namespace: "ups",
			},
			Strategy: "openshift",
		},
	}
}

func getTestPostgresSec() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-postgres",
			Namespace: "ups",
		},
		Data: map[string][]byte{
			"host":     []byte("test"),
			"password": []byte("test"),
			"port":     []byte("test"),
			"tls":      []byte("test"),
			"username": []byte("test"),
		},
	}
}

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
}

func TestReconciler_ReconcileCustomResource(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("Error creating build scheme")
	}

	cases := []struct {
		Name           string
		FakeClient     k8sclient.Client
		FakeConfig     *config.ConfigReadWriterMock
		Installation   *integreatlyv1alpha1.RHMI
		ExpectErr      bool
		ExpectedStatus integreatlyv1alpha1.StatusPhase
		FakeMPM        *marketplace.MarketplaceInterfaceMock
		Recorder       record.EventRecorder
	}{
		{
			Name:           "UPS Test: test custom resource is reconciled and phase complete returned",
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			FakeMPM:        &marketplace.MarketplaceInterfaceMock{},
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "ups",
				},
			},
			FakeConfig: basicConfigMock(),
			FakeClient: fake.NewFakeClientWithScheme(scheme, getTestPostgres(), getTestPostgresSec(), mockUpsCRWithStatus(upsv1alpha1.PhaseReconciling)),
			Recorder:   setupRecorder(),
		},
		{
			Name:           "UPS Test: Phase failed when error in creating custom resource",
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			FakeMPM:        &marketplace.MarketplaceInterfaceMock{},
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeConfig:     basicConfigMock(),
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return k8serr.NewNotFound(schema.GroupResource{}, "unifiedpushserver")
				},
				CreateFunc: func(ctx context.Context, obj runtime.Object, opts ...k8sclient.CreateOption) error {
					return errors.New("dummy create error")
				},
			},
			ExpectErr: true,
			Recorder:  setupRecorder(),
		},
		{
			Name:           "UPS Test: Phase failed when general error in finding custom resource",
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			FakeMPM:        &marketplace.MarketplaceInterfaceMock{},
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeConfig:     basicConfigMock(),
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return errors.New("General error")
				},
			},
			ExpectErr: true,
			Recorder:  setupRecorder(),
		},
		{
			Name:           "UPS Test: Phase in progress when custom resource is not in phase complete",
			ExpectedStatus: integreatlyv1alpha1.PhaseInProgress,
			FakeMPM:        &marketplace.MarketplaceInterfaceMock{},
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "ups",
				},
			},
			FakeConfig: basicConfigMock(),
			FakeClient: fake.NewFakeClientWithScheme(scheme, getTestPostgres(), getTestPostgresSec(), mockUpsCRWithStatus(upsv1alpha1.PhaseInitializing)),
			Recorder:   setupRecorder(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM, tc.Recorder, getLogger())
			if err != nil {
				t.Fatal("unexpected err settin up reconciler ", err)
			}
			status, err := reconciler.reconcileComponents(context.TODO(), tc.Installation, tc.FakeClient)
			if tc.ExpectErr && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectErr && err != nil {
				t.Fatal("expected no error but got one ", err)
			}
			if tc.ExpectedStatus != status {
				t.Fatalf("expected phase %s but got %s", tc.ExpectedStatus, status)
			}
		})
	}
}

func TestReconciler_ReconcileHost(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("Error creating build scheme")
	}

	cases := []struct {
		Name           string
		FakeClient     k8sclient.Client
		FakeConfig     *config.ConfigReadWriterMock
		Installation   *integreatlyv1alpha1.RHMI
		ExpectErr      bool
		ExpectedStatus integreatlyv1alpha1.StatusPhase
		FakeMPM        *marketplace.MarketplaceInterfaceMock
		Recorder       record.EventRecorder
	}{
		{
			Name:           "UPS Test: Config is updated with route url correctly - phase complete",
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			FakeMPM:        &marketplace.MarketplaceInterfaceMock{},
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeConfig:     basicConfigMock(),
			FakeClient:     fake.NewFakeClientWithScheme(scheme, basicRouteMock()),
			Recorder:       setupRecorder(),
		},
		{
			Name:           "UPS Test: Cannot retrieve route - phase failed",
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			FakeMPM:        &marketplace.MarketplaceInterfaceMock{},
			ExpectErr:      true,
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeConfig:     errorConfigMock(),
			FakeClient:     fake.NewFakeClientWithScheme(scheme),
			Recorder:       setupRecorder(),
		},
		{
			Name:           "UPS Test: Cannot update config with route url - phase failed",
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			FakeMPM:        &marketplace.MarketplaceInterfaceMock{},
			ExpectErr:      true,
			Installation:   &integreatlyv1alpha1.RHMI{},
			FakeConfig:     errorConfigMock(),
			FakeClient:     fake.NewFakeClientWithScheme(scheme, basicRouteMock()),
			Recorder:       setupRecorder(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM, tc.Recorder, getLogger())
			if err != nil {
				t.Fatal("unexpected err settin up reconciler ", err)
			}
			status, err := reconciler.reconcileHost(context.TODO(), tc.FakeClient)
			if tc.ExpectErr && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectErr && err != nil {
				t.Fatal("expected no error but got one ", err)
			}
			if tc.ExpectedStatus != status {
				t.Fatalf("expected phase %s but got %s", tc.ExpectedStatus, status)
			}
		})
	}
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.ProductUps})
}
