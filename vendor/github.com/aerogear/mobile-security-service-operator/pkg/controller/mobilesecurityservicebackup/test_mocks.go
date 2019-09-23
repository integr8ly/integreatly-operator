package mobilesecurityservicebackup

import (
	"github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/aerogear/mobile-security-service-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Centralized mock objects for use in tests
var (
	bkpInstance = v1alpha1.MobileSecurityServiceBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mobile-security-service-backup",
			Namespace: utils.OperatorNamespaceForLocalEnv,
		},
		Spec: v1alpha1.MobileSecurityServiceBackupSpec{
			Image:              "quay.io/integreatly/backup-container:latest",
			Schedule:           "0 0 * * *",
			AwsS3BucketName:    "example-awsS3BucketName",
			AwsAccessKeyId:     "example-awsAccessKeyId",
			AwsSecretAccessKey: "example-awsSecretAccessKey",
		},
	}

	bkpInstanceWithSecretNames = v1alpha1.MobileSecurityServiceBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mobile-security-service-backup",
			Namespace: utils.OperatorNamespaceForLocalEnv,
		},
		Spec: v1alpha1.MobileSecurityServiceBackupSpec{
			Image:                    "quay.io/integreatly/backup-container:latest",
			Schedule:                 "0 0 * * *",
			EncryptionKeySecretName:  "enc-secret-test",
			AwsCredentialsSecretName: "aws-secret-test",
		},
	}

	bkpInstanceNonDefaultNamespace = v1alpha1.MobileSecurityServiceBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mobile-security-service-db-backup",
			Namespace: "mobile-security-service-namespace",
		},
		Spec: v1alpha1.MobileSecurityServiceBackupSpec{
			Image:              "quay.io/integreatly/backup-container:latest",
			Schedule:           "0 0 * * *",
			AwsS3BucketName:    "example-awsS3BucketName",
			AwsAccessKeyId:     "example-awsAccessKeyId",
			AwsSecretAccessKey: "example-awsSecretAccessKey",
		},
	}

	bkpInstanceWithoutSpec = v1alpha1.MobileSecurityServiceBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mobile-security-service-backup",
			Namespace: utils.OperatorNamespaceForLocalEnv,
		},
	}

	lsDB = map[string]string{"app": "mobilesecurityservice", "mobilesecurityservice_cr": utils.MobileSecurityServiceDBCRName}

	podDatabase = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mobile-security-service-db",
			Namespace: utils.OperatorNamespaceForLocalEnv,
			Labels:    lsDB,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Image: "mobile-security-service-db",
				Name:  "mobile-security-service-db",
				Ports: []corev1.ContainerPort{{
					ContainerPort: 5000,
					Protocol:      "TCP",
				}},
				Env: []corev1.EnvVar{
					corev1.EnvVar{
						Name:  "PGDATABASE",
						Value: "test",
					},
					corev1.EnvVar{
						Name:  "PGUSER",
						Value: "test",
					},
					corev1.EnvVar{
						Name:  "PGPASSWORD",
						Value: "test",
					},
					{
						Name:  "PGDATA",
						Value: "/var/lib/pgsql/data/pgdata",
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "test",
						MountPath: "/var/lib/pgsql/data",
					},
				},
			}},
		},
	}

	serviceDatabase = corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mobile-security-service-db",
			Namespace: utils.OperatorNamespaceForLocalEnv,
			Labels:    lsDB,
		},
	}
)
