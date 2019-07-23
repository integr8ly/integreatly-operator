package fuse

import (
	"context"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	syn "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1alpha1"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func basicConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		ReadFuseFunc: func() (ready *config.Fuse, e error) {
			return config.NewFuse(config.ProductConfig{}), nil
		},
		ReadRHSSOFunc: func() (*config.RHSSO, error) {
			return config.NewRHSSO(config.ProductConfig{
				"NAMESPACE": "fuse",
				"URL":       "fuse.openshift-cluster.com",
			}), nil
		},
	}
}

func TestReconciler_reconcileCustomResource(t *testing.T) {
	scheme := runtime.NewScheme()
	syn.SchemeBuilder.AddToScheme(scheme)
	cases := []struct {
		Name           string
		client         client.Client
		FakeConfig     *config.ConfigReadWriterMock
		Installation   *v1alpha1.Installation
		ExpectErr      bool
		ExpectedStatus v1alpha1.StatusPhase
		FakeMPM        *marketplace.MarketplaceInterfaceMock
	}{
		{
			Name: "Test reconcile custom resource returns in progress when successful created",
			client: pkgclient.NewFakeClientWithScheme(scheme, &syn.Syndesis{
				ObjectMeta: v12.ObjectMeta{
					Name: "integreatly",
				},
				Status: syn.SyndesisStatus{
					Phase: syn.SyndesisPhaseInstalling,
				},
			}),
			FakeConfig: basicConfigMock(),
			Installation: &v1alpha1.Installation{
				TypeMeta: v12.TypeMeta{
					Kind:       "installation",
					APIVersion: "integreatly.org/v1alpha1",
				},
			},
			ExpectedStatus: v1alpha1.PhaseInProgress,
		},
		{
			Name: "Test reconcile custom resource returns failed when cr status is failed",
			client: pkgclient.NewFakeClientWithScheme(scheme, &syn.Syndesis{
				ObjectMeta: v12.ObjectMeta{
					Name:      "integreatly",
					Namespace: "fuse",
				},
				Status: syn.SyndesisStatus{
					Phase: syn.SyndesisPhaseStartupFailed,
				},
			}),
			FakeConfig: basicConfigMock(),
			Installation: &v1alpha1.Installation{
				TypeMeta: v12.TypeMeta{
					Kind:       "installation",
					APIVersion: "integreatly.org/v1alpha1",
				},
			},
			ExpectedStatus: v1alpha1.PhaseFailed,
			ExpectErr:      true,
		},
		{
			Name: "Test reconcile custom resource returns no phase when cr status is installed",
			client: pkgclient.NewFakeClientWithScheme(scheme, &syn.Syndesis{
				ObjectMeta: v12.ObjectMeta{
					Name:      "integreatly",
					Namespace: "fuse",
				},
				Status: syn.SyndesisStatus{
					Phase: syn.SyndesisPhaseInstalled,
				},
			}),
			FakeConfig: basicConfigMock(),
			Installation: &v1alpha1.Installation{
				TypeMeta: v12.TypeMeta{
					Kind:       "installation",
					APIVersion: "integreatly.org/v1alpha1",
				},
			},
			ExpectedStatus: v1alpha1.PhaseNone,
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
			phase, err := reconciler.reconcileCustomResource(context.TODO(), tc.Installation, tc.client)
			if tc.ExpectErr && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectErr && err != nil {
				t.Fatal("expected no error but got one ", err)
			}
			if tc.ExpectedStatus != phase {
				t.Fatal("expected phase ", tc.ExpectedStatus, " but got ", phase)
			}
		})
	}
}

func TestReconciler_ReconcileNamespace(t *testing.T) {
	defaultInstallation := &v1alpha1.Installation{ObjectMeta: v12.ObjectMeta{Name: "install"}, TypeMeta: v12.TypeMeta{APIVersion: v1alpha1.SchemeGroupVersion.String()}}
	cases := []struct {
		Name           string
		client         client.Client
		FakeK8sClient  *k8sclient.Clientset
		FakeConfig     *config.ConfigReadWriterMock
		Installation   *v1alpha1.Installation
		ExpectErr      bool
		ExpectedStatus v1alpha1.StatusPhase
		FakeMPM        *marketplace.MarketplaceInterfaceMock
	}{
		{
			Name: "Test namespace reconcile completes without error",
			client: pkgclient.NewFakeClient(&v1.Namespace{
				ObjectMeta: v12.ObjectMeta{
					Name: "fuse",
					OwnerReferences: []v12.OwnerReference{
						{
							Name:       "install",
							APIVersion: v1alpha1.SchemeGroupVersion.String(),
						},
					},
				},
				Status: v1.NamespaceStatus{
					Phase: v1.NamespaceActive,
				},
			}),
			FakeConfig:     basicConfigMock(),
			Installation:   defaultInstallation,
			ExpectedStatus: v1alpha1.PhaseNone,
		},
		{
			Name: "Test namespace reconcile returns waiting when ns not ready",
			client: pkgclient.NewFakeClient(&v1.Namespace{
				ObjectMeta: v12.ObjectMeta{
					Name: "fuse",
					OwnerReferences: []v12.OwnerReference{
						{
							Name:       "install",
							APIVersion: v1alpha1.SchemeGroupVersion.String(),
						},
					},
				},
			}),
			FakeConfig:     basicConfigMock(),
			Installation:   defaultInstallation,
			ExpectedStatus: v1alpha1.PhaseAwaitingNS,
		},
		{
			Name: "Test namespace reconcile returns waiting when ns is terminating",
			client: pkgclient.NewFakeClient(&v1.Namespace{
				ObjectMeta: v12.ObjectMeta{
					Name: "fuse",
				},
				Status: v1.NamespaceStatus{
					Phase: v1.NamespaceTerminating,
				},
			}),
			FakeConfig: basicConfigMock(),
			Installation: &v1alpha1.Installation{
				TypeMeta: v12.TypeMeta{
					Kind:       "installation",
					APIVersion: "integreatly.org/v1alpha1",
				},
			},
			ExpectedStatus: v1alpha1.PhaseAwaitingNS,
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
			phase, err := reconciler.reconcileNamespace(context.TODO(), tc.Installation, tc.client)
			if tc.ExpectErr && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectErr && err != nil {
				t.Fatal("expected no error but got one ", err)
			}
			if tc.ExpectedStatus != phase {
				t.Fatal("expected phase ", tc.ExpectedStatus, " but got ", phase)
			}
		})
	}
}
