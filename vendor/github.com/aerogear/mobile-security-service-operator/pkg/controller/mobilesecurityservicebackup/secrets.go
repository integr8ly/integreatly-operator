package mobilesecurityservicebackup

import (
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	awsSecretPrefix     = "aws-"
	dbSecretPrefix      = "db-"
	encryptionKeySecret = "encryption-"
)

//Returns the buildDatabaseSecret object for the Mobile Security Service Backup
func (r *ReconcileMobileSecurityServiceBackup) buildSecret(bkp *mobilesecurityservicev1alpha1.MobileSecurityServiceBackup, prefix string, secretData map[string][]byte, secretStringData map[string]string) *corev1.Secret {
	ls := getBkpLabels(bkp.Name)

	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      prefix + bkp.Name,
			Namespace: bkp.Namespace,
			Labels:    ls,
		},
		Data: secretData,
		Type: "Opaque",
	}

	// Add string data
	if secretStringData != nil && len(secretStringData) > 0 {
		secret.StringData = secretStringData
	}

	// Set MobileSecurityServiceBackup as the owner and controller
	controllerutil.SetControllerReference(bkp, secret, r.scheme)
	return secret
}
