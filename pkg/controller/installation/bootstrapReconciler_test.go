package installation

import (
	"context"
	"errors"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconciler_reconcileRHMIConfigPermissions(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = rbacv1.SchemeBuilder.AddToScheme(scheme)

	tests := []struct {
		Name           string
		ExpectedStatus integreatlyv1alpha1.StatusPhase
		FakeConfig     *config.ConfigReadWriterMock
		FakeMPM        *marketplace.MarketplaceInterfaceMock
		Installation   *integreatlyv1alpha1.RHMI
		Recorder       record.EventRecorder
		FakeClient     k8sclient.Client
		Assertion      func(k8sclient.Client) error
	}{
		{
			Name: "Test Role and Role Binding is not created",
			FakeConfig: &config.ConfigReadWriterMock{
				GetOperatorNamespaceFunc: func() string {
					return "test-namespace"
				},
			},
			FakeMPM:        &marketplace.MarketplaceInterfaceMock{},
			Installation:   &integreatlyv1alpha1.RHMI{},
			Recorder:       record.NewFakeRecorder(50),
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			FakeClient:     fakeclient.NewFakeClientWithScheme(scheme),
			Assertion:      assertRoleBindingNotFound,
		},
		{
			Name: "Test - error in creating role and role binding",
			FakeConfig: &config.ConfigReadWriterMock{
				GetOperatorNamespaceFunc: func() string {
					return "test-namespace"
				},
			},
			FakeMPM:        &marketplace.MarketplaceInterfaceMock{},
			Installation:   &integreatlyv1alpha1.RHMI{},
			Recorder:       record.NewFakeRecorder(50),
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return errors.New("dummy get error")
				},
				CreateFunc: func(ctx context.Context, obj runtime.Object, opts ...k8sclient.CreateOption) error {
					return errors.New("dummy create error")
				},
			},
			Assertion: func(client k8sclient.Client) error {
				return nil
			},
		},
		{
			Name: "Test that existing role binding is deleted",
			FakeConfig: &config.ConfigReadWriterMock{
				GetOperatorNamespaceFunc: func() string {
					return "test-namespace"
				},
			},
			FakeMPM:        &marketplace.MarketplaceInterfaceMock{},
			Installation:   &integreatlyv1alpha1.RHMI{},
			Recorder:       record.NewFakeRecorder(50),
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			FakeClient: fake.NewFakeClientWithScheme(scheme, &rbacv1.RoleBinding{
				ObjectMeta: v1.ObjectMeta{
					Name:      "rhmiconfig-dedicated-admins-role-binding",
					Namespace: "test-namespace",
				},
			}),
			Assertion: assertRoleBindingNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			reconciler, err := NewBootstrapReconciler(tt.FakeConfig, tt.Installation, tt.FakeMPM, tt.Recorder, l.NewLogger())
			if err != nil {
				t.Fatalf("Error creating bootstrap reconciler: %s", err)
			}

			phase, err := reconciler.reconcileRHMIConfigPermissions(context.TODO(), tt.FakeClient)

			if phase != tt.ExpectedStatus {
				t.Fatalf("Expected %s phase but got %s", tt.ExpectedStatus, phase)
			}

			if err := tt.Assertion(tt.FakeClient); err != nil {
				t.Fatalf("Failed assertion: %v", err)
			}
		})
	}
}

func assertRoleBindingNotFound(client k8sclient.Client) error {
	configRole := &rbacv1.Role{}
	err := client.Get(context.TODO(), k8sclient.ObjectKey{
		Name:      "rhmiconfig-dedicated-admins-role",
		Namespace: "test-namespace",
	}, configRole)
	if err == nil {
		return errors.New("Role rhmiconfig-dedicated-admins-role should not exist")
	}

	if !k8serr.IsNotFound(err) {
		return fmt.Errorf("Unexpected error occurred: %v", err)
	}

	return nil
}
