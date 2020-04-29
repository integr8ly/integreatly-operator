package installation

import (
	"context"
	"errors"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
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
	}{
		{
			Name: "Test Role and Role Binding is created",
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			reconciler, err := NewBootstrapReconciler(tt.FakeConfig, tt.Installation, tt.FakeMPM, tt.Recorder)
			if err != nil {
				t.Fatalf("Error creating bootstrap reconciler: %s", err)
			}

			phase, err := reconciler.reconcileRHMIConfigPermissions(context.TODO(), tt.FakeClient)

			if phase != tt.ExpectedStatus {
				t.Fatalf("Expected %s phase but got %s", tt.ExpectedStatus, phase)
			}

		})
	}
}
