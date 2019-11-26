package ups

import (
	"context"
	"testing"

	upsv1alpha1 "github.com/aerogear/unifiedpush-operator/pkg/apis/push/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"

	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/pkg/errors"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := upsv1alpha1.SchemeBuilder.AddToScheme(scheme)
	err = routev1.AddToScheme(scheme)
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
		TypeMeta: v1.TypeMeta{
			Kind:       "Route",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Namespace: "ups",
			Name:      defaultRoutename,
		},
	}
}

func mockUpsCRWithStatus(phase upsv1alpha1.StatusPhase) *upsv1alpha1.UnifiedPushServer {
	return &upsv1alpha1.UnifiedPushServer{
		ObjectMeta: v1.ObjectMeta{
			Namespace: "ups",
			Name:      "ups",
		},
		Status: upsv1alpha1.UnifiedPushServerStatus{
			Phase: phase,
		},
	}
}

func TestReconciler_ReconcileCustomResource(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("Error creating build scheme")
	}

	cases := []struct {
		Name           string
		FakeClient     client.Client
		FakeConfig     *config.ConfigReadWriterMock
		Installation   *v1alpha1.Installation
		ExpectErr      bool
		ExpectedStatus v1alpha1.StatusPhase
		FakeMPM        *marketplace.MarketplaceInterfaceMock
	}{
		{
			Name:           "UPS Test: test custom resource is reconciled and phase complete returned",
			ExpectedStatus: v1alpha1.PhaseCompleted,
			FakeMPM:        &marketplace.MarketplaceInterfaceMock{},
			Installation:   &v1alpha1.Installation{},
			FakeConfig:     basicConfigMock(),
			FakeClient:     fake.NewFakeClientWithScheme(scheme, mockUpsCRWithStatus(upsv1alpha1.PhaseComplete)),
		},
		{
			Name:           "UPS Test: Phase failed when error in creating custom resource",
			ExpectedStatus: v1alpha1.PhaseFailed,
			FakeMPM:        &marketplace.MarketplaceInterfaceMock{},
			Installation:   &v1alpha1.Installation{},
			FakeConfig:     basicConfigMock(),
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return k8serr.NewNotFound(schema.GroupResource{}, "unifiedpushserver")
				},
				CreateFunc: func(ctx context.Context, obj runtime.Object, opts ...pkgclient.CreateOption) error {
					return errors.New("dummy create error")
				},
			},
			ExpectErr: true,
		},
		{
			Name:           "UPS Test: Phase failed when general error in finding custom resource",
			ExpectedStatus: v1alpha1.PhaseFailed,
			FakeMPM:        &marketplace.MarketplaceInterfaceMock{},
			Installation:   &v1alpha1.Installation{},
			FakeConfig:     basicConfigMock(),
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return errors.New("General error")
				},
			},
			ExpectErr: true,
		},
		{
			Name:           "UPS Test: Phase in progress when custom resource is not in phase complete",
			ExpectedStatus: v1alpha1.PhaseInProgress,
			FakeMPM:        &marketplace.MarketplaceInterfaceMock{},
			Installation:   &v1alpha1.Installation{},
			FakeConfig:     basicConfigMock(),
			FakeClient:     fake.NewFakeClientWithScheme(scheme, mockUpsCRWithStatus(upsv1alpha1.PhaseProvision)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM)
			if err != nil {
				t.Fatal("unexpected err settin up reconciler ", err)
			}
			status, err := reconciler.reconcileCustomResource(context.TODO(), tc.FakeClient)
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
		FakeClient     client.Client
		FakeConfig     *config.ConfigReadWriterMock
		Installation   *v1alpha1.Installation
		ExpectErr      bool
		ExpectedStatus v1alpha1.StatusPhase
		FakeMPM        *marketplace.MarketplaceInterfaceMock
	}{
		{
			Name:           "UPS Test: Config is updated with route url correctly - phase complete",
			ExpectedStatus: v1alpha1.PhaseCompleted,
			FakeMPM:        &marketplace.MarketplaceInterfaceMock{},
			Installation:   &v1alpha1.Installation{},
			FakeConfig:     basicConfigMock(),
			FakeClient:     fake.NewFakeClientWithScheme(scheme, basicRouteMock()),
		},
		{
			Name:           "UPS Test: Cannot retrieve route - phase failed",
			ExpectedStatus: v1alpha1.PhaseFailed,
			FakeMPM:        &marketplace.MarketplaceInterfaceMock{},
			ExpectErr:      true,
			Installation:   &v1alpha1.Installation{},
			FakeConfig:     errorConfigMock(),
			FakeClient:     fake.NewFakeClientWithScheme(scheme),
		},
		{
			Name:           "UPS Test: Cannot update config with route url - phase failed",
			ExpectedStatus: v1alpha1.PhaseFailed,
			FakeMPM:        &marketplace.MarketplaceInterfaceMock{},
			ExpectErr:      true,
			Installation:   &v1alpha1.Installation{},
			FakeConfig:     errorConfigMock(),
			FakeClient:     fake.NewFakeClientWithScheme(scheme, basicRouteMock()),
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM)
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
