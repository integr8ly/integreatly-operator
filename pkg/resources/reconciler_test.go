package resources

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/integr8ly/integreatly-operator/utils"
	k8sappsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"testing"
	"time"

	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"

	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	oauthv1 "github.com/openshift/api/oauth/v1"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func basicConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		ReadRHSSOFunc: func() (*config.RHSSO, error) {
			return config.NewRHSSO(config.ProductConfig{
				"NAMESPACE": "3scale",
				"URL":       "3scale.openshift-cluster.com",
			}), nil
		},
		WriteConfigFunc: func(config config.ConfigReadable) error {
			return nil
		},
	}
}

func TestNewReconciler_ReconcileSubscription(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		Name             string
		FakeMPM          marketplace.MarketplaceInterface
		client           k8sclient.Client
		SubscriptionName string
		ExpectErr        bool
		ExpectedStatus   integreatlyv1alpha1.StatusPhase
		Installation     *integreatlyv1alpha1.RHMI
		Target           marketplace.Target
		Validate         func(t *testing.T, mock *marketplace.MarketplaceInterfaceMock)
		Assertion        func(k8sclient.Client) error
	}{
		{
			Name: "test reconcile subscription creates a new subscription  completes successfully ",
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalgSourceReconciler marketplace.CatalogSourceReconciler) error {

					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plan *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlan{Status: operatorsv1alpha1.InstallPlanStatus{Phase: operatorsv1alpha1.InstallPlanPhaseComplete}}, &operatorsv1alpha1.Subscription{}, nil
				},
			},
			SubscriptionName: "something",
			ExpectedStatus:   integreatlyv1alpha1.PhaseCompleted,
			Installation:     &integreatlyv1alpha1.RHMI{},
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
			client: utils.NewTestClient(scheme),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalgSourceReconciler marketplace.CatalogSourceReconciler) error {

					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
					return nil, &operatorsv1alpha1.Subscription{ObjectMeta: metav1.ObjectMeta{
						// simulate the time has passed
						CreationTimestamp: metav1.Time{Time: time.Now().AddDate(0, 0, -1)},
					}}, k8serr.NewNotFound(operatorsv1alpha1.Resource("installplan"), "my-install-plan")
				},
			},
			SubscriptionName: "something",
			ExpectedStatus:   integreatlyv1alpha1.PhaseAwaitingOperator,
		},
		{
			Name:             "test reconcile subscription returns phase failed when unable to create catalog source",
			client:           utils.NewTestClient(runtime.NewScheme()),
			SubscriptionName: "something",
			ExpectedStatus:   integreatlyv1alpha1.PhaseFailed,
			FakeMPM:          marketplace.NewManager(),
			ExpectErr:        true,
		},
		{
			Name: "test reconcile subscription returns phase in progress if there is an install plan approved but not completed or failed",
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalgSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlan{
						Spec: operatorsv1alpha1.InstallPlanSpec{Approved: true},
						Status: operatorsv1alpha1.InstallPlanStatus{
							Phase: operatorsv1alpha1.InstallPlanPhaseInstalling,
						},
					}, &operatorsv1alpha1.Subscription{}, nil
				},
			},
			SubscriptionName: "something",
			ExpectedStatus:   integreatlyv1alpha1.PhaseInProgress,
			Installation:     &integreatlyv1alpha1.RHMI{},
		},
		{
			Name: "test reconcile subscription returns phase failed if unable to retrieve install plans",
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalgSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
					return nil, nil, fmt.Errorf("simulate error gettiing install plans")
				},
			},
			SubscriptionName: "something",
			ExpectedStatus:   integreatlyv1alpha1.PhaseFailed,
			Installation:     &integreatlyv1alpha1.RHMI{},
			ExpectErr:        true,
		},
		{
			Name: "test reconcile subscription returns phase in progress if there are no install plans for subscription",
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalgSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
					return nil, &operatorsv1alpha1.Subscription{}, nil
				},
			},
			SubscriptionName: "something",
			ExpectedStatus:   integreatlyv1alpha1.PhaseInProgress,
			Installation:     &integreatlyv1alpha1.RHMI{},
		},
		{
			Name: "test reconcile subscription returns phase failed if unable to delete subscription due for re-install ",
			client: &moqclient.SigsClientInterfaceMock{DeleteFunc: func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.DeleteOption) error {
				return fmt.Errorf("some error")
			}},
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalgSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlan{
						Status: operatorsv1alpha1.InstallPlanStatus{Phase: operatorsv1alpha1.InstallPlanPhaseFailed},
					}, &operatorsv1alpha1.Subscription{}, nil
				},
			},
			SubscriptionName: "something",
			ExpectedStatus:   integreatlyv1alpha1.PhaseFailed,
			Installation:     &integreatlyv1alpha1.RHMI{},
			ExpectErr:        true,
		},
		{
			Name: "test reconcile subscription returns phase failed if unable to delete csv for re-install ",
			client: &moqclient.SigsClientInterfaceMock{DeleteFunc: func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.DeleteOption) error {
				return fmt.Errorf("some error")
			}},
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalgSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlan{
						Status: operatorsv1alpha1.InstallPlanStatus{Phase: operatorsv1alpha1.InstallPlanPhaseFailed},
					}, &operatorsv1alpha1.Subscription{Status: operatorsv1alpha1.SubscriptionStatus{InstalledCSV: "test-csv"}}, nil
				},
			},
			SubscriptionName: "something",
			ExpectedStatus:   integreatlyv1alpha1.PhaseFailed,
			Installation:     &integreatlyv1alpha1.RHMI{},
			ExpectErr:        true,
		},
		{
			Name:   "test reconcile subscription returns phase awaiting operator after successful delete of failed install plan and csv",
			client: utils.NewTestClient(scheme, &operatorsv1alpha1.ClusterServiceVersion{ObjectMeta: metav1.ObjectMeta{Name: "test-csv", Namespace: "test-ns"}}, &operatorsv1alpha1.Subscription{Status: operatorsv1alpha1.SubscriptionStatus{InstalledCSV: "test-csv"}}),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalgSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlan{
						Status: operatorsv1alpha1.InstallPlanStatus{Phase: operatorsv1alpha1.InstallPlanPhaseFailed},
					}, &operatorsv1alpha1.Subscription{Status: operatorsv1alpha1.SubscriptionStatus{InstalledCSV: "test-csv"}}, nil
				},
			},
			SubscriptionName: "something",
			ExpectedStatus:   integreatlyv1alpha1.PhaseAwaitingOperator,
			Installation:     &integreatlyv1alpha1.RHMI{},
		},
		{
			Name: "test reconcile subscription deletes CSV and subscription if the CSV doesn't have a deployment",
			client: utils.NewTestClient(scheme,
				&operatorsv1alpha1.ClusterServiceVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-csv",
						Namespace: "test-ns",
					},
					Spec: operatorsv1alpha1.ClusterServiceVersionSpec{
						InstallStrategy: operatorsv1alpha1.NamedInstallStrategy{
							StrategyName: operatorsv1alpha1.InstallStrategyNameDeployment,
							StrategySpec: operatorsv1alpha1.StrategyDetailsDeployment{
								DeploymentSpecs: []operatorsv1alpha1.StrategyDeploymentSpec{},
							},
						},
					},
				},
				&operatorsv1alpha1.Subscription{
					Status: operatorsv1alpha1.SubscriptionStatus{
						InstalledCSV: "test-csv",
					},
				},
			),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName, ns string) (*operatorsv1alpha1.InstallPlan, *operatorsv1alpha1.Subscription, error) {
					return &operatorsv1alpha1.InstallPlan{
						Spec: operatorsv1alpha1.InstallPlanSpec{
							Approved: true,
							ClusterServiceVersionNames: []string{
								"test-csv",
							},
						},
						Status: operatorsv1alpha1.InstallPlanStatus{Phase: operatorsv1alpha1.InstallPlanPhaseComplete},
					}, &operatorsv1alpha1.Subscription{Status: operatorsv1alpha1.SubscriptionStatus{InstalledCSV: "test-csv"}}, nil
				},
			},
			SubscriptionName: "something",
			ExpectedStatus:   integreatlyv1alpha1.PhaseAwaitingOperator,
			Installation:     &integreatlyv1alpha1.RHMI{},
			Assertion: func(client k8sclient.Client) error {
				csv := &operatorsv1alpha1.ClusterServiceVersion{}
				err := client.Get(context.TODO(), k8sclient.ObjectKey{
					Name:      "test-csv",
					Namespace: "test-ns",
				}, csv)
				if err == nil || !k8serr.IsNotFound(err) {
					return errors.New("CSV was not deleted")
				}

				sub := &operatorsv1alpha1.Subscription{}
				err = client.Get(context.TODO(), k8sclient.ObjectKey{
					Name:      "something",
					Namespace: "test-ns",
				}, sub)
				if err == nil || !k8serr.IsNotFound(err) {
					return errors.New("Susbcription was not deleted")
				}

				return nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler := NewReconciler(
				tc.FakeMPM,
			)

			testNamespace := "test-ns"
			manifestsDirectory := "fakemanifestsdirectory"
			cfgMapCsReconciler := marketplace.NewConfigMapCatalogSourceReconciler(manifestsDirectory, tc.client, testNamespace, marketplace.CatalogSourceName)
			status, err := reconciler.ReconcileSubscription(context.TODO(), marketplace.Target{Namespace: testNamespace, Channel: "integreatly", SubscriptionName: tc.SubscriptionName, Package: tc.SubscriptionName}, []string{testNamespace}, backup.NewNoopBackupExecutor(), tc.client, cfgMapCsReconciler, getLogger())
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
			if tc.Assertion != nil {
				if err := tc.Assertion(tc.client); err != nil {
					t.Errorf("failed assertion: %v", err)
				}
			}
		})
	}
}

func TestReconciler_reconcilePullSecret(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	defPullSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      integreatlyv1alpha1.DefaultOriginPullSecretName,
			Namespace: integreatlyv1alpha1.DefaultOriginPullSecretNamespace,
		},
		Data: map[string][]byte{
			"test": {'t', 'e', 's', 't'},
		},
	}

	customPullSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Data: map[string][]byte{
			"test": {'t', 'e', 's', 't'},
		},
	}

	cases := []struct {
		Name         string
		Client       k8sclient.Client
		Installation *integreatlyv1alpha1.RHMI
		Config       *config.ConfigReadWriterMock
		Validate     func(c k8sclient.Client) error
	}{
		{
			Name:   "test pull secret is reconciled successfully",
			Client: utils.NewTestClient(scheme, defPullSecret, customPullSecret),
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testinstall",
					Namespace: "testinstall",
				},
				Spec: integreatlyv1alpha1.RHMISpec{
					PullSecret: integreatlyv1alpha1.PullSecretSpec{
						Name:      "test",
						Namespace: "test",
					},
				},
			},
			Config: basicConfigMock(),
			Validate: func(c k8sclient.Client) error {
				s := &corev1.Secret{}
				err := c.Get(context.TODO(), k8sclient.ObjectKey{Name: "test", Namespace: "test"}, s)
				if err != nil {
					return err
				}
				if !bytes.Equal(s.Data["test"], customPullSecret.Data["test"]) {
					return fmt.Errorf("expected data %v, but got %v", customPullSecret.Data["test"], s.Data["test"])
				}
				return nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			testReconciler := NewReconciler(nil)
			_, err := testReconciler.ReconcilePullSecret(context.TODO(), "test", "new-pull-secret-name", tc.Installation, tc.Client)
			if err != nil {
				t.Fatal("failed to run pull secret reconcile: ", err)
			}
			if err = tc.Validate(tc.Client); err != nil {
				t.Fatal("test validation failed: ", err)
			}
		})
	}
}

func TestReconciler_ReconcileOauthClient(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}
	existingClient := &oauthv1.OAuthClient{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		GrantMethod:  oauthv1.GrantHandlerAuto,
		Secret:       "test",
		RedirectURIs: []string{"http://test.com"},
	}
	cases := []struct {
		Name           string
		OauthClient    *oauthv1.OAuthClient
		ExpectErr      bool
		ExpectedStatus integreatlyv1alpha1.StatusPhase
		client         k8sclient.Client
		Installation   *integreatlyv1alpha1.RHMI
	}{
		{
			Name: "test oauth client is reconciled correctly when it does not exist",
			OauthClient: &oauthv1.OAuthClient{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				GrantMethod:  oauthv1.GrantHandlerAuto,
				Secret:       "test",
				RedirectURIs: []string{"http://test.com"},
			},
			Installation: &integreatlyv1alpha1.RHMI{
				TypeMeta: metav1.TypeMeta{
					Kind:       "RHMI",
					APIVersion: integreatlyv1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-install",
				},
			},
			client:         utils.NewTestClient(scheme),
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
		},
		{
			Name:        "test oauth client is reconciled correctly when it does exist",
			OauthClient: existingClient,
			Installation: &integreatlyv1alpha1.RHMI{
				TypeMeta: metav1.TypeMeta{
					Kind:       "RHMI",
					APIVersion: integreatlyv1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-install",
				},
			},
			client:         utils.NewTestClient(scheme, existingClient),
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
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
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}
	nsName := "test-ns"
	installation := &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name: "install",
			UID:  types.UID("xyz"),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "RHMI",
			APIVersion: integreatlyv1alpha1.GroupVersion.String(),
		},
	}
	cases := []struct {
		Name           string
		client         k8sclient.Client
		Installation   *integreatlyv1alpha1.RHMI
		ExpectErr      bool
		ExpectedStatus integreatlyv1alpha1.StatusPhase
		ExpectLabel    bool
		FakeMPM        *marketplace.MarketplaceInterfaceMock
	}{
		{
			Name: "Test namespace reconcile completes without error",
			client: utils.NewTestClient(scheme, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						OwnerLabelKey: string(installation.GetUID()),
					},
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceActive,
				},
			}),
			Installation:   installation,
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			ExpectLabel:    true,
		},
		{
			Name: "Test namespace reconcile returns waiting when ns not ready",
			client: utils.NewTestClient(scheme, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						OwnerLabelKey: string(installation.GetUID()),
					},
				},
				Status: corev1.NamespaceStatus{},
			}),
			Installation:   installation,
			ExpectedStatus: integreatlyv1alpha1.PhaseInProgress,
			ExpectLabel:    true,
		},
		{
			Name: "Test namespace reconcile returns waiting when ns is terminating",
			client: utils.NewTestClient(scheme, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						OwnerLabelKey: string(installation.GetUID()),
					},
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceTerminating,
				},
			}),
			Installation:   &integreatlyv1alpha1.RHMI{},
			ExpectedStatus: integreatlyv1alpha1.PhaseInProgress,
			ExpectLabel:    true,
		},
		{
			Name: "Test namespace reconcile return error when pulling secret fails",
			client: utils.NewTestClient(scheme, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						OwnerLabelKey: string(installation.GetUID()),
					},
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceActive,
				},
			}),
			Installation: &integreatlyv1alpha1.RHMI{
				Spec: integreatlyv1alpha1.RHMISpec{
					PullSecret: integreatlyv1alpha1.PullSecretSpec{
						Name:      "test",
						Namespace: "test",
					},
				},
			},
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			ExpectErr:      true,
			ExpectLabel:    false,
		},
		{
			Name: "Test if label is added to an existing namespace",
			client: utils.NewTestClient(scheme, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
				},
			}),
			Installation:   installation,
			ExpectedStatus: integreatlyv1alpha1.PhaseInProgress,
			ExpectErr:      false,
			ExpectLabel:    true,
		},
		{
			Name: "Test if label is changed to false when namespace is reconciled",
			client: utils.NewTestClient(scheme, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						"openshift.io/user-monitoring": "true",
					},
				},
			}),
			Installation:   installation,
			ExpectedStatus: integreatlyv1alpha1.PhaseInProgress,
			ExpectErr:      false,
			ExpectLabel:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler := NewReconciler(
				tc.FakeMPM,
			)
			phase, err := reconciler.ReconcileNamespace(context.TODO(), "test-ns", tc.Installation, tc.client, getLogger())
			if tc.ExpectErr && err == nil {
				t.Fatal("expected an error but got none")
			}
			if !tc.ExpectErr && err != nil {
				t.Fatal("expected no error but got one ", err)
			}
			if tc.ExpectedStatus != phase {
				t.Fatal("expected phase ", tc.ExpectedStatus, " but got ", phase)
			}
			labelExists, labelIsTrue, err := verifyNSLabelExistsAndIsTrue(tc.client)
			if err != nil {
				t.Fatal("error when verifying namespace label exists")
			}
			if !labelExists && tc.ExpectLabel {
				t.Fatal("error because label was not applied to namespace")
			}
			if labelIsTrue && tc.ExpectLabel {
				t.Fatal("error when verifying namespace label was changed to false during reconcile")
			}
		})
	}
}

func verifyNSLabelExistsAndIsTrue(client k8sclient.Client) (bool, bool, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ns",
		},
	}
	err := client.Get(context.TODO(), k8sclient.ObjectKey{Name: ns.Name}, ns)
	if err != nil {
		return false, false, err
	}
	labelExists, labelIsTrue := false, false
	if ns.Labels["openshift.io/user-monitoring"] != "" {
		labelExists = true
	}
	if ns.Labels["openshift.io/user-monitoring"] != "false" && labelExists {
		labelIsTrue = true
	}
	return labelExists, labelIsTrue, nil
}

func TestReconciler_ReconcileCsvDeploymentsPriority(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}
	type fields struct {
		mpm                marketplace.MarketplaceInterface
		productDeclaration *marketplace.ProductDeclaration
	}
	type args struct {
		ctx               context.Context
		client            k8sclient.Client
		csvName           string
		csvNamespace      string
		priorityClassName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name:   "success reconciling csv deployments priority",
			fields: fields{},
			args: args{
				ctx: context.TODO(),
				client: utils.NewTestClient(scheme, &operatorsv1alpha1.ClusterServiceVersion{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "csvName",
						Namespace: "csvNamespace",
					},
					Spec: operatorsv1alpha1.ClusterServiceVersionSpec{
						InstallStrategy: operatorsv1alpha1.NamedInstallStrategy{
							StrategyName: "",
							StrategySpec: operatorsv1alpha1.StrategyDetailsDeployment{
								DeploymentSpecs: []operatorsv1alpha1.StrategyDeploymentSpec{
									{
										Spec: k8sappsv1.DeploymentSpec{
											Template: corev1.PodTemplateSpec{
												Spec: corev1.PodSpec{
													PriorityClassName: "priorityClassName",
												},
											},
										},
									},
								},
							},
						},
					},
				}),
				csvName:           "csvName",
				csvNamespace:      "csvNamespace",
				priorityClassName: "priorityClassName",
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name:   "failure reconciling csv deployments priority",
			fields: fields{},
			args: args{
				ctx: context.TODO(),
				client: &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
						return nil
					},
					PatchFunc: func(ctx context.Context, obj k8sclient.Object, patch k8sclient.Patch, opts ...k8sclient.PatchOption) error {
						return fmt.Errorf("generic error")
					},
				},
				csvName:           "csvName",
				csvNamespace:      "csvNamespace",
				priorityClassName: "priorityClassName",
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				mpm:                tt.fields.mpm,
				productDeclaration: tt.fields.productDeclaration,
			}
			got, err := r.ReconcileCsvDeploymentsPriority(tt.args.ctx, tt.args.client, tt.args.csvName, tt.args.csvNamespace, tt.args.priorityClassName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileCsvDeploymentsPriority() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReconcileCsvDeploymentsPriority() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_CreateNSWithProjectRequest(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}
	nsName := "test-ns"
	testNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
		},
	}
	installation := &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name: "install",
			UID:  types.UID("xyz"),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "RHMI",
			APIVersion: integreatlyv1alpha1.GroupVersion.String(),
		},
	}

	type args struct {
		ctx                       context.Context
		namespace                 string
		client                    k8sclient.Client
		inst                      *integreatlyv1alpha1.RHMI
		addRHMIMonitoringLabels   bool
		addClusterMonitoringLabel bool
		disableUserAlerting       bool
	}
	tests := []struct {
		name    string
		args    args
		want    *corev1.Namespace
		wantErr bool
	}{
		{
			name: "test create namespace from project request with basic labels",
			args: args{
				ctx:                       context.TODO(),
				namespace:                 nsName,
				client:                    utils.NewTestClient(scheme, installation, testNamespace),
				inst:                      installation,
				addRHMIMonitoringLabels:   false,
				addClusterMonitoringLabel: false,
				disableUserAlerting:       false,
			},
			want: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						"integreatly": "true",
						OwnerLabelKey: string(installation.GetUID()),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "test create namespace from project request with rhmi monitoring labels",
			args: args{
				ctx:                       context.TODO(),
				namespace:                 nsName,
				client:                    utils.NewTestClient(scheme, installation, testNamespace),
				inst:                      installation,
				addRHMIMonitoringLabels:   true,
				addClusterMonitoringLabel: false,
				disableUserAlerting:       false,
			},
			want: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						"monitoring-key": "middleware",
						"integreatly":    "true",
						OwnerLabelKey:    string(installation.GetUID()),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "test create namespace from project request with cluster monitoring labels",
			args: args{
				ctx:                       context.TODO(),
				namespace:                 nsName,
				client:                    utils.NewTestClient(scheme, installation, testNamespace),
				inst:                      installation,
				addRHMIMonitoringLabels:   false,
				addClusterMonitoringLabel: true,
				disableUserAlerting:       false,
			},
			want: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						"openshift.io/cluster-monitoring": "true",
						"integreatly":                     "true",
						OwnerLabelKey:                     string(installation.GetUID()),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "test create namespace from project request with user monitoring disabled",
			args: args{
				ctx:                       context.TODO(),
				namespace:                 nsName,
				client:                    utils.NewTestClient(scheme, installation, testNamespace),
				inst:                      installation,
				addRHMIMonitoringLabels:   false,
				addClusterMonitoringLabel: false,
				disableUserAlerting:       true,
			},
			want: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						"openshift.io/user-monitoring": "false",
						"integreatly":                  "true",
						OwnerLabelKey:                  string(installation.GetUID()),
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateNSWithProjectRequest(tt.args.ctx, tt.args.namespace, tt.args.client, tt.args.inst, tt.args.addRHMIMonitoringLabels, tt.args.addClusterMonitoringLabel, tt.args.disableUserAlerting)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateNSWithProjectRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.GetLabels(), tt.want.GetLabels()) {
				t.Errorf("CreateNSWithProjectRequest() got Labels = %v, want Labels %v", got, tt.want)
			}
		})
	}
}
