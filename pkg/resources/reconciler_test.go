package resources

import (
	"context"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	v14 "github.com/openshift/api/oauth/v1"
	alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	v13 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
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

func buildScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	alpha1.AddToScheme(scheme)
	v14.AddToScheme(scheme)
	v13.SchemeBuilder.AddToScheme(scheme)

	return scheme
}

func TestNewReconciler_ReconcileSubscription(t *testing.T) {
	ownerInstall := &v1alpha1.Installation{
		TypeMeta: v12.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "Installation",
		},
	}
	catalogSourceConfig := &v13.CatalogSourceConfig{
		ObjectMeta: v12.ObjectMeta{
			Name:      "installed-integreatly-test-ns",
			Namespace: "openshift-marketplace",
		},
	}
	ownerutil.AddOwner(catalogSourceConfig, ownerInstall, true, true)
	cases := []struct {
		Name             string
		FakeMPM          marketplace.MarketplaceInterface
		client           client.Client
		SubscriptionName string
		ExpectErr        bool
		ExpectedStatus   v1alpha1.StatusPhase
		Installation     *v1alpha1.Installation
		Target           marketplace.Target
		Validate         func(t *testing.T, mock *marketplace.MarketplaceInterfaceMock)
	}{
		{
			Name: "test reconcile subscription creates a new subscription  completes successfully ",
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient client.Client, owner ownerutil.Owner, os v13.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy alpha1.Approval) error {

					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient client.Client, subName string, ns string) (plan *alpha1.InstallPlan, subscription *alpha1.Subscription, e error) {
					return &alpha1.InstallPlan{Status: alpha1.InstallPlanStatus{Phase: alpha1.InstallPlanPhaseComplete}}, &alpha1.Subscription{}, nil
				},
			},
			SubscriptionName: "something",
			ExpectedStatus:   v1alpha1.PhaseCompleted,
			Installation:     &v1alpha1.Installation{},
			Validate: func(t *testing.T, mock *marketplace.MarketplaceInterfaceMock) {
				if len(mock.InstallOperatorCalls()) != 1 {
					t.Fatalf("expected create subscription to be called once but was called %v", len(mock.InstallOperatorCalls()))
				}
				if len(mock.GetSubscriptionInstallPlanCalls()) != 1 {
					t.Fatalf("expected GetSubscriptionInstallPlanCalls to be called once but was called %v", len(mock.GetSubscriptionInstallPlanCalls()))
				}
			},
		},
		{
			Name:   "test reconcile subscription recreates subscription when installation plan not found completes successfully ",
			client: pkgclient.NewFakeClientWithScheme(buildScheme()),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient client.Client, owner ownerutil.Owner, os v13.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy alpha1.Approval) error {

					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient client.Client, subName string, ns string) (plan *alpha1.InstallPlan, subscription *alpha1.Subscription, e error) {
					return nil, &alpha1.Subscription{ObjectMeta: v12.ObjectMeta{
						// simulate the time has passed
						CreationTimestamp: v12.Time{Time: time.Now().AddDate(0, 0, -1)},
					}}, errors.NewNotFound(alpha1.Resource("installplan"), "my-install-plan")
				},
			},
			SubscriptionName: "something",
			ExpectedStatus:   v1alpha1.PhaseAwaitingOperator,
		},
		{
			Name: "test reconcile subscription returns waiting for operator when catalog source config not ready",
			client: pkgclient.NewFakeClientWithScheme(buildScheme(), catalogSourceConfig, &alpha1.CatalogSourceList{
				Items: []alpha1.CatalogSource{
					alpha1.CatalogSource{
						ObjectMeta: v12.ObjectMeta{
							Name:      "test",
							Namespace: "test-ns",
						},
					},
				},
			}),
			SubscriptionName: "something",
			ExpectedStatus:   v1alpha1.PhaseFailed,
			FakeMPM:          marketplace.NewManager(),
			Installation:     ownerInstall,
			ExpectErr:        true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler := NewReconciler(
				tc.FakeMPM,
			)
			status, err := reconciler.ReconcileSubscription(context.TODO(), tc.Installation, marketplace.Target{Namespace: "test-ns", Channel: "integreatly", Pkg: tc.SubscriptionName}, tc.client)
			if tc.ExpectErr && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectErr && err != nil {
				t.Fatal("expected no error but got one ", err)
			}
			if tc.ExpectedStatus != status {
				t.Fatal("expected phase ", tc.ExpectedStatus, " but got ", status)
			}
			if tc.Validate != nil {
				tc.Validate(t, tc.FakeMPM.(*marketplace.MarketplaceInterfaceMock))
			}

		})
	}
}

func TestReconciler_ReconcileOauthClient(t *testing.T) {
	existingClient := &v14.OAuthClient{
		GrantMethod:  v14.GrantHandlerAuto,
		Secret:       "test",
		RedirectURIs: []string{"http://test.com"},
	}
	cases := []struct {
		Name           string
		OauthClient    *v14.OAuthClient
		ExpectErr      bool
		ExpectedStatus v1alpha1.StatusPhase
		client         client.Client
		Installation   *v1alpha1.Installation
	}{
		{
			Name: "test oauth client is reconciled correctly when it does not exist",
			OauthClient: &v14.OAuthClient{
				GrantMethod:  v14.GrantHandlerAuto,
				Secret:       "test",
				RedirectURIs: []string{"http://test.com"},
			},
			Installation: &v1alpha1.Installation{
				TypeMeta: v12.TypeMeta{
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
					Kind:       "Installation",
				},
				ObjectMeta: v12.ObjectMeta{
					Name: "test-install",
				},
			},
			client:         pkgclient.NewFakeClientWithScheme(buildScheme()),
			ExpectedStatus: v1alpha1.PhaseCompleted,
		},
		{
			Name:        "test oauth client is reconciled correctly when it does exist",
			OauthClient: existingClient,
			Installation: &v1alpha1.Installation{
				TypeMeta: v12.TypeMeta{
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
					Kind:       "Installation",
				},
				ObjectMeta: v12.ObjectMeta{
					Name: "test-install",
				},
			},
			client:         pkgclient.NewFakeClientWithScheme(buildScheme(), existingClient),
			ExpectedStatus: v1alpha1.PhaseCompleted,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler := NewReconciler(nil)
			phase, err := reconciler.ReconcileOauthClient(context.TODO(), tc.Installation, tc.OauthClient, tc.client)
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
	nsName := "test-ns"
	defaultInstallation := &v1alpha1.Installation{ObjectMeta: v12.ObjectMeta{Name: "install"}, TypeMeta: v12.TypeMeta{APIVersion: v1alpha1.SchemeGroupVersion.String()}}
	cases := []struct {
		Name           string
		client         client.Client
		Installation   *v1alpha1.Installation
		ExpectErr      bool
		ExpectedStatus v1alpha1.StatusPhase
		FakeMPM        *marketplace.MarketplaceInterfaceMock
	}{
		{
			Name: "Test namespace reconcile completes without error",
			client: pkgclient.NewFakeClient(&v1.Namespace{
				ObjectMeta: v12.ObjectMeta{
					Name: nsName,
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
			Installation:   defaultInstallation,
			ExpectedStatus: v1alpha1.PhaseCompleted,
		},
		{
			Name: "Test namespace reconcile returns waiting when ns not ready",
			client: pkgclient.NewFakeClient(&v1.Namespace{
				ObjectMeta: v12.ObjectMeta{
					Name: nsName,
					OwnerReferences: []v12.OwnerReference{
						{
							Name:       "install",
							APIVersion: v1alpha1.SchemeGroupVersion.String(),
						},
					},
				},
				Status: v1.NamespaceStatus{},
			}),
			Installation:   defaultInstallation,
			ExpectedStatus: v1alpha1.PhaseInProgress,
		},
		{
			Name: "Test namespace reconcile returns waiting when ns is terminating",
			client: pkgclient.NewFakeClient(&v1.Namespace{
				ObjectMeta: v12.ObjectMeta{
					Name: nsName,
				},
				Status: v1.NamespaceStatus{
					Phase: v1.NamespaceTerminating,
				},
			}),
			Installation:   &v1alpha1.Installation{},
			ExpectedStatus: v1alpha1.PhaseInProgress,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler := NewReconciler(
				tc.FakeMPM,
			)
			phase, err := reconciler.ReconcileNamespace(context.TODO(), "test-ns", tc.Installation, tc.client)
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
