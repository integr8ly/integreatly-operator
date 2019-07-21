package resources

import (
	"context"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
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
	return scheme
}

func TestNewReconciler_ReconcileSubscription(t *testing.T) {
	cases := []struct {
		Name             string
		FakeMPM          *marketplace.MarketplaceInterfaceMock
		client           client.Client
		SubscriptionName string
		ExpectErr        bool
		ExpectedStatus   v1alpha1.StatusPhase
		Installation     *v1alpha1.Installation
		Validate         func(t *testing.T, mock *marketplace.MarketplaceInterfaceMock)
	}{
		{
			Name: "test reconcile subscription creates a new subscription  completes successfully ",
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				CreateSubscriptionFunc: func(ctx context.Context, serverClient client.Client, owner ownerutil.Owner, os v13.OperatorSource, ns string, pkg string, channel string, operatorGroupNamespaces []string, approvalStrategy alpha1.Approval) error {
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
				if len(mock.CreateSubscriptionCalls()) != 1 {
					t.Fatalf("expected create subscription to be called once but was called %v", len(mock.CreateSubscriptionCalls()))
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
				CreateSubscriptionFunc: func(ctx context.Context, serverClient client.Client, owner ownerutil.Owner, os v13.OperatorSource, ns string, pkg string, channel string, operatorGroupNamespaces []string, approvalStrategy alpha1.Approval) error {
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
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler := NewReconciler(
				tc.FakeMPM,
			)
			status, err := reconciler.ReconcileSubscription(context.TODO(), tc.Installation, tc.SubscriptionName, "test-ns", tc.client)
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
				tc.Validate(t, tc.FakeMPM)
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
