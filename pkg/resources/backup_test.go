package resources

import (
	"context"
	"testing"

	"github.com/integr8ly/integreatly-operator/utils"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func basicClient(scheme *runtime.Scheme, objects ...runtime.Object) k8sclient.Client {
	return utils.NewTestClient(scheme, objects...)
}

func TestBackups(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []struct {
		Name          string
		BackupConfig  BackupConfig
		Context       context.Context
		ConfigManager *config.ConfigReadWriterMock
		Instance      *integreatlyv1alpha1.RHMI
		Client        k8sclient.Client
		Validation    func(e error, t *testing.T)
	}{
		{
			Name:          "test backups reconcile without errors",
			Context:       context.TODO(),
			Client:        basicClient(scheme, backupsSecretMock()),
			ConfigManager: getMockConfigManager(),
			BackupConfig: BackupConfig{
				Name:      "test-backups",
				Namespace: "backups",
				Components: []BackupComponent{
					{
						Name:     "component",
						Schedule: "3 20 * * *",
						Secret:   BackupSecretLocation{Name: "Component-Secret", Namespace: "secret-namespace"},
						Type:     "test",
					},
				},
				BackendSecret:    BackupSecretLocation{Name: "backend-secret", Namespace: "backend-secret-namespace"},
				EncryptionSecret: BackupSecretLocation{Name: "encryption-secret", Namespace: "encryption-secret-namespace"},
			},
			Validation: func(e error, t *testing.T) {
				if e != nil {
					t.Fatalf("expected no error, but got: %s", e.Error())
				}
			},
		},
		{
			Name:    "test backups reconcile without errors when objects already exist",
			Context: context.TODO(),
			Client: basicClient(
				scheme,
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "backupjob",
					},
				},
				backupsSecretMock(),
			),
			ConfigManager: getMockConfigManager(),
			BackupConfig: BackupConfig{
				Name:      "test-backups",
				Namespace: "backups",
				Components: []BackupComponent{
					{
						Name:     "component",
						Schedule: "3 20 * * *",
						Secret:   BackupSecretLocation{Name: "Component-Secret", Namespace: "secret-namespace"},
						Type:     "test",
					},
				},
				BackendSecret:    BackupSecretLocation{Name: "backend-secret", Namespace: "backend-secret-namespace"},
				EncryptionSecret: BackupSecretLocation{Name: "encryption-secret", Namespace: "encryption-secret-namespace"},
			},
			Validation: func(e error, t *testing.T) {
				if e != nil {
					t.Fatalf("expected no error, but got: %s", e.Error())
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			err := ReconcileBackup(scenario.Context, scenario.Client, scenario.BackupConfig, scenario.ConfigManager, getLogger(), "rhoam")

			if scenario.Validation != nil {
				scenario.Validation(err, t)
			}
		})
	}
}

func getMockConfigManager() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		GetOperatorNamespaceFunc: func() string {
			return "integreatly-operator"
		},
		GetBackupsSecretNameFunc: func() string {
			return "backups-s3-credentials"
		},
	}
}

func backupsSecretMock() *corev1.Secret {
	config := getMockConfigManager()
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.GetBackupsSecretNameFunc(),
			Namespace: config.GetOperatorNamespace(),
		},
		Data: map[string][]byte{},
	}
}
