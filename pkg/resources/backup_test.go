package resources

import (
	"context"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func basicClient(objects ...runtime.Object) client.Client {
	scheme := runtime.NewScheme()
	v1alpha1.SchemeBuilder.AddToScheme(scheme)
	rbacv1.SchemeBuilder.AddToScheme(scheme)
	corev1.SchemeBuilder.AddToScheme(scheme)
	batchv1.SchemeBuilder.AddToScheme(scheme)
	batchv1beta1.SchemeBuilder.AddToScheme(scheme)
	return fakeclient.NewFakeClientWithScheme(scheme, objects...)
}

func basicInstallationObject() *v1alpha1.Installation {
	return &v1alpha1.Installation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "installation",
			Namespace: "installation-namespace",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "installation",
			APIVersion: "integreatly.org",
		},
	}
}

func TestBackups(t *testing.T) {

	scenarios := []struct {
		Name         string
		BackupConfig BackupConfig
		Context      context.Context
		Instance     *v1alpha1.Installation
		Client       client.Client
		Validation   func(e error, t *testing.T)
	}{
		{
			Name:     "test backups reconcile without errors",
			Context:  context.TODO(),
			Instance: basicInstallationObject(),
			Client:   basicClient(),
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
			Name:     "test backups reconcile without errors when objects already exist",
			Context:  context.TODO(),
			Instance: basicInstallationObject(),
			Client: basicClient(
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "backupjob",
					},
				},
			),
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
			err := ReconcileBackup(scenario.Context, scenario.Client, scenario.Instance, scenario.BackupConfig)

			if scenario.Validation != nil {
				scenario.Validation(err, t)
			}
		})
	}
}
