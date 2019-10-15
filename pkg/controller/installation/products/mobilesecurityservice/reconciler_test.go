package mobilesecurityservice

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mss "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	v1Route "github.com/openshift/api/route/v1"
	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	err = operatorsv1alpha1.AddToScheme(scheme)
	err = marketplacev1.SchemeBuilder.AddToScheme(scheme)
	err = corev1.SchemeBuilder.AddToScheme(scheme)
	err = coreosv1.SchemeBuilder.AddToScheme(scheme)
	err = mss.SchemeBuilder.AddToScheme(scheme)
	err = v1Route.AddToScheme(scheme)
	return scheme, err
}

func basicInstallation() *v1alpha1.Installation {
	return &v1alpha1.Installation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "installation",
			Namespace: defaultInstallationNamespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "installation",
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
	}
}

func basicConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		ReadMobileSecurityServiceFunc: func() (ready *config.MobileSecurityService, e error) {
			return config.NewMobileSecurityService(config.ProductConfig{}), nil
		},
		WriteConfigFunc: func(config config.ConfigReadable) error {
			return nil
		},
	}
}

func TestReconciler_config(t *testing.T) {
	cases := []struct {
		Name           string
		ExpectError    bool
		ExpectedError  string
		ExpectedStatus v1alpha1.StatusPhase
		FakeConfig     *config.ConfigReadWriterMock
		FakeClient     pkgclient.Client
		FakeMPM        *marketplace.MarketplaceInterfaceMock
		Installation   *v1alpha1.Installation
	}{
		{
			Name:           "test error on failed config",
			ExpectError:    true,
			ExpectedError:  "could not read mobile security service config: could not read mss config",
			ExpectedStatus: v1alpha1.PhaseFailed,
			Installation:   &v1alpha1.Installation{},
			FakeClient:     fakeclient.NewFakeClient(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadMobileSecurityServiceFunc: func() (ready *config.MobileSecurityService, e error) {
					return nil, errors.New("could not read mss config")
				},
			},
		},
		{
			Name:         "test namespace is set without fail",
			Installation: &v1alpha1.Installation{},
			FakeClient:   fakeclient.NewFakeClient(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadMobileSecurityServiceFunc: func() (ready *config.MobileSecurityService, e error) {
					return config.NewMobileSecurityService(config.ProductConfig{
						"NAMESPACE": "",
					}), nil
				},
			},
		},
		{
			Name:           "test subscription phase with error from mpm",
			ExpectedStatus: v1alpha1.PhaseFailed,
			ExpectError:    true,
			Installation:   &v1alpha1.Installation{},
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, os marketplacev1.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {
					return errors.New("dummy error")
				},
			},
			FakeClient: fakeclient.NewFakeClient(),
			FakeConfig: basicConfigMock(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM)
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
		FakeClient     pkgclient.Client
		FakeConfig     *config.ConfigReadWriterMock
		Installation   *v1alpha1.Installation
		ExpectError    bool
		ExpectedStatus v1alpha1.StatusPhase
		FakeMPM        *marketplace.MarketplaceInterfaceMock
	}{
		{
			Name:           "Test reconcile custom resource returns in progress when successful created",
			FakeClient:     fakeclient.NewFakeClientWithScheme(scheme),
			FakeConfig:     basicConfigMock(),
			Installation:   &v1alpha1.Installation{},
			ExpectedStatus: v1alpha1.PhaseCompleted,
		},
		{
			Name: "Test reconcile custom resource returns failed when cr status is failed",
			FakeClient: &moqclient.SigsClientInterfaceMock{
				CreateFunc: func(ctx context.Context, obj runtime.Object) error {
					return errors.New("dummy create error")
				},
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return k8serr.NewNotFound(schema.GroupResource{
						Group:    mss.SchemeBuilder.GroupVersion.Group,
						Resource: "MobileSecurityServiceDB",
					}, key.Name)
				},
			},
			FakeConfig:     basicConfigMock(),
			Installation:   &v1alpha1.Installation{},
			ExpectedStatus: v1alpha1.PhaseFailed,
			ExpectError:    true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM)
			if err != nil {
				t.Fatal("unexpected err ", err)
			}

			phase, err := reconciler.reconcileComponents(context.TODO(), tc.FakeClient, tc.Installation)
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

	mssDbNotUp := &mss.MobileSecurityServiceDB{
		TypeMeta: metav1.TypeMeta{
			Kind: "MobileSecurityServiceDB",
			APIVersion: fmt.Sprintf(
				"%s/%s",
				mss.SchemeGroupVersion.Group,
				mss.SchemeGroupVersion.Version),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mobile-security-service-db",
			Namespace: "mobile-security-service",
		},
		Status: mss.MobileSecurityServiceDBStatus{
			DatabaseStatus: "",
		},
	}

	mssDbUp := &mss.MobileSecurityServiceDB{
		TypeMeta: metav1.TypeMeta{
			Kind: "MobileSecurityServiceDB",
			APIVersion: fmt.Sprintf(
				"%s/%s",
				mss.SchemeGroupVersion.Group,
				mss.SchemeGroupVersion.Version),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mobile-security-service-db",
			Namespace: "mobile-security-service",
		},
		Status: mss.MobileSecurityServiceDBStatus{
			DatabaseStatus: "OK",
		},
	}

	mssNotUp := &mss.MobileSecurityService{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MobileSecurityService",
			APIVersion: mss.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mobile-security-service",
			Namespace: "mobile-security-service",
		},
		Status: mss.MobileSecurityServiceStatus{
			AppStatus: "",
		},
	}

	mssUp := &mss.MobileSecurityService{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MobileSecurityService",
			APIVersion: mss.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mobile-security-service",
			Namespace: "mobile-security-service",
		},
		Status: mss.MobileSecurityServiceStatus{
			AppStatus: "OK",
		},
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
			Name:           "test failure to get db cr",
			ExpectedStatus: v1alpha1.PhaseFailed,
			ExpectedError:  "failed to get mss db cr when reconciling custom resource",
			ExpectError:    true,
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme),
			FakeConfig:     basicConfigMock(),
			Installation:   &v1alpha1.Installation{},
		},
		{
			Name:           "test failure to get mss cr",
			ExpectedStatus: v1alpha1.PhaseFailed,
			ExpectedError:  "failed to get mss cr when reconciling custom resource",
			ExpectError:    true,
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, mssDbUp),
			FakeConfig:     basicConfigMock(),
			Installation:   &v1alpha1.Installation{},
		},
		{
			Name:           "test unready db cr returns phase in progress",
			ExpectedStatus: v1alpha1.PhaseInProgress,
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, mssDbNotUp),
			FakeConfig:     basicConfigMock(),
			Installation:   &v1alpha1.Installation{},
		},
		{
			Name:           "test unready mss cr returns phase in progress",
			ExpectedStatus: v1alpha1.PhaseInProgress,
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, mssDbUp, mssNotUp),
			FakeConfig:     basicConfigMock(),
			Installation:   &v1alpha1.Installation{},
		},
		{
			Name:           "test ready db and mss crs returns phase complete",
			ExpectedStatus: v1alpha1.PhaseCompleted,
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, mssDbUp, mssUp),
			FakeConfig:     basicConfigMock(),
			Installation:   &v1alpha1.Installation{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM)
			if err != nil && err.Error() != tc.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
			}

			status, err := reconciler.handleProgressPhase(context.TODO(), tc.FakeClient)
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

	// initialise runtime objects

	namespace := &corev1.Namespace{
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

	route := &v1Route.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Route",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defaultInstallationNamespace,
			Name:      "mobile-security-service",
		},
	}

	mssDb := &mss.MobileSecurityServiceDB{
		TypeMeta: metav1.TypeMeta{
			Kind: "MobileSecurityServiceDB",
			APIVersion: fmt.Sprintf(
				"%s/%s",
				mss.SchemeGroupVersion.Group,
				mss.SchemeGroupVersion.Version),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbClusterName,
			Namespace: defaultInstallationNamespace,
		},
		Status: mss.MobileSecurityServiceDBStatus{
			DatabaseStatus: "OK",
		},
	}

	mssServer := &mss.MobileSecurityService{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MobileSecurityService",
			APIVersion: mss.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serverClusterName,
			Namespace: defaultInstallationNamespace,
		},
		Status: mss.MobileSecurityServiceStatus{
			AppStatus: "OK",
		},
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
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, namespace, route, mssDb, mssServer),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadMobileSecurityServiceFunc: func() (ready *config.MobileSecurityService, e error) {
					return config.NewMobileSecurityService(config.ProductConfig{
						"NAMESPACE": "",
					}), nil
				},
			},
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, os marketplacev1.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {
					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient client.Client, subName string, ns string) (plan *operatorsv1alpha1.InstallPlanList, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlanList{
							TypeMeta: metav1.TypeMeta{
								Kind:       "MobileSecurityService",
								APIVersion: mss.SchemeGroupVersion.String(),
							},
							ListMeta: metav1.ListMeta{},
						}, &operatorsv1alpha1.Subscription{
							Status: operatorsv1alpha1.SubscriptionStatus{
								Install: &operatorsv1alpha1.InstallPlanReference{
									Name: "mss-install-plan",
								},
							},
						}, nil
				},
			},
			Installation: basicInstallation(),
			Product:      &v1alpha1.InstallationProductStatus{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM)
			if err != nil && err.Error() != tc.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
			}

			status, err := reconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.FakeClient)
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

func TestReconciler_testPhases(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		Name           string
		ExpectedStatus v1alpha1.StatusPhase
		FakeConfig     *config.ConfigReadWriterMock
		FakeClient     pkgclient.Client
		FakeMPM        *marketplace.MarketplaceInterfaceMock
		Installation   *v1alpha1.Installation
		Product        *v1alpha1.InstallationProductStatus
	}{
		{
			Name:           "test namespace terminating returns phase in progress",
			ExpectedStatus: v1alpha1.PhaseInProgress,
			Installation:   basicInstallation(),
			FakeClient: moqclient.NewSigsClientMoqWithScheme(scheme, &corev1.Namespace{
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
					Phase: corev1.NamespaceTerminating,
				},
			}),
			FakeConfig: basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, os marketplacev1.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {
					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient client.Client, subName string, ns string) (plan *operatorsv1alpha1.InstallPlanList, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlanList{}, &operatorsv1alpha1.Subscription{}, nil
				},
			},
			Product: &v1alpha1.InstallationProductStatus{},
		},
		{
			Name:           "test subscription creating returns phase in progress",
			ExpectedStatus: v1alpha1.PhaseInProgress,
			Installation:   basicInstallation(),
			FakeClient:     fakeclient.NewFakeClientWithScheme(scheme),
			FakeConfig:     basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, os marketplacev1.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {
					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient client.Client, subName string, ns string) (*operatorsv1alpha1.InstallPlanList, *operatorsv1alpha1.Subscription, error) {
					return &operatorsv1alpha1.InstallPlanList{}, &operatorsv1alpha1.Subscription{}, nil
				},
			},
			Product: &v1alpha1.InstallationProductStatus{},
		},
		{
			Name:           "test components creating returns phase in progress",
			ExpectedStatus: v1alpha1.PhaseInProgress,
			Installation:   basicInstallation(),
			FakeClient:     fakeclient.NewFakeClientWithScheme(scheme),
			FakeConfig:     basicConfigMock(),
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner, os marketplacev1.OperatorSource, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval) error {
					return nil
				},
				GetSubscriptionInstallPlansFunc: func(ctx context.Context, serverClient client.Client, sub string, ns string) (*operatorsv1alpha1.InstallPlanList, *operatorsv1alpha1.Subscription, error) {
					return &operatorsv1alpha1.InstallPlanList{
							TypeMeta: metav1.TypeMeta{
								Kind:       "MobileSecurityService",
								APIVersion: mss.SchemeGroupVersion.String(),
							},
							ListMeta: metav1.ListMeta{},
						}, &operatorsv1alpha1.Subscription{
							Status: operatorsv1alpha1.SubscriptionStatus{
								Install: &operatorsv1alpha1.InstallPlanReference{
									Name: "mss-install-plan",
								},
							},
						}, nil
				},
			},
			Product: &v1alpha1.InstallationProductStatus{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM)
			if err != nil {
				t.Fatalf("unexpected error : '%v'", err)
			}

			status, err := reconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.FakeClient)
			if err != nil {
				t.Fatalf("expected no error but got one: %v", err)
			}
			if status != tc.ExpectedStatus {
				t.Fatalf("Expected status: '%v', got: '%v'", tc.ExpectedStatus, status)
			}
		})
	}
}
