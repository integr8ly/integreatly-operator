package installation

import (
	"context"
	"errors"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestReconciler_reconcileRHSSOAdminCredentials(t *testing.T) {

	scheme := runtime.NewScheme()
	corev1.SchemeBuilder.AddToScheme(scheme)

	basicConfig := &config.ConfigReadWriterMock{
		GetRHSSOAdminCredentialSeedSecretNameFunc: func() string {
			return "credentials-sso-seed"
		},
		GetOperatorNamespaceFunc: func() string {
			return "test-ns"
		},
	}

	tests := []struct {
		Name           string
		FakeConfig     *config.ConfigReadWriterMock
		FakeClient     k8sclient.Client
		ExpectedStatus integreatlyv1alpha1.StatusPhase
		ExpectedError  bool
	}{
		{
			Name:           "Successfully created RHSSO Admin Credential Seed",
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			ExpectedError:  false,
			FakeClient:     fakeclient.NewFakeClientWithScheme(scheme),
			FakeConfig:     basicConfig,
		},
		{
			Name:           "Failed created RHSSO Admin Credential Seed",
			ExpectedStatus: integreatlyv1alpha1.PhaseFailed,
			ExpectedError:  true,
			FakeClient: &moqclient.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return nil
				},
				CreateFunc: func(ctx context.Context, obj runtime.Object, opts ...k8sclient.CreateOption) error {
					return errors.New("dummy create error")
				},
				UpdateFunc: func(ctx context.Context, obj runtime.Object, opts ...k8sclient.UpdateOption) error {
					return errors.New("dummy update error")
				},
			},
			FakeConfig: basicConfig,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.FakeConfig,
			}
			got, err := r.reconcileRHSSOAdminCredentials(context.TODO(), tt.FakeClient)
			if (err != nil) != tt.ExpectedError {
				t.Errorf("reconcileRHSSOAdminCredentials() error = %v, wantErr %v", err, tt.ExpectedError)
			}
			if got != tt.ExpectedStatus {
				t.Errorf("reconcileRHSSOAdminCredentials() got = %v, want %v", got, tt.ExpectedStatus)
			}
		})
	}
}
