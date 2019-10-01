package unifiedpushserver

import (
	"fmt"

	pushv1alpha1 "github.com/aerogear/unifiedpush-operator/pkg/apis/push/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func backups(ups *pushv1alpha1.UnifiedPushServer) ([]batchv1beta1.CronJob, error) {
	cronjobs := []batchv1beta1.CronJob{}
	for _, upsBackup := range ups.Spec.Backups {
		cronJobLabels := labels(ups, "backup")
		jobLabels := cronJobLabels
		jobLabels["cronjob-name"] = upsBackup.Name

		cronjobs = append(cronjobs, batchv1beta1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      upsBackup.Name,
				Namespace: ups.Namespace,
				Labels:    cronJobLabels,
			},
			Spec: batchv1beta1.CronJobSpec{
				Schedule: upsBackup.Schedule,
				JobTemplate: batchv1beta1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: jobLabels,
							},
							Spec: corev1.PodSpec{
								// This SA needs to be created beforehand
								// https://github.com/integr8ly/backup-container-image/tree/master/templates/openshift/rbac
								ServiceAccountName: "backupjob",
								Containers: []corev1.Container{
									{
										Name:            upsBackup.Name + "-ups-backup",
										Image:           cfg.BackupImage,
										ImagePullPolicy: "Always",
										Command:         buildBackupContainerCommand(upsBackup, ups.Namespace),
										Env:             buildBackupCronJobEnvVars(upsBackup, ups.Name, ups.Namespace),
									},
								},
								RestartPolicy: corev1.RestartPolicyOnFailure,
							},
						},
					},
				},
			},
		})
	}
	return cronjobs, nil
}

func buildBackupContainerCommand(upsBackup pushv1alpha1.UnifiedPushServerBackup, upsNamespace string) []string {
	command := []string{"/opt/intly/tools/entrypoint.sh", "-c", "postgres", "-n", upsNamespace}

	// If there is no encryption secret, we need to inhibit the
	// encryption behaviour
	if upsBackup.EncryptionKeySecretName == "" {
		command = append(command, "-e", "")
	}

	return command
}

func buildBackupCronJobEnvVars(upsBackup pushv1alpha1.UnifiedPushServerBackup, upsName string, upsNamespace string) []corev1.EnvVar {

	envVars := []corev1.EnvVar{
		{
			Name:  "PRODUCT_NAME",
			Value: "unifiedpush",
		},
		{
			Name:  "COMPONENT_SECRET_NAME",
			Value: fmt.Sprintf("%s-%s", upsName, "postgresql"),
		},
		{
			Name:  "COMPONENT_SECRET_NAMESPACE",
			Value: upsNamespace,
		},
	}

	backendSecretNamespace := upsBackup.BackendSecretNamespace
	if backendSecretNamespace == "" {
		backendSecretNamespace = upsNamespace
	}

	encryptionKeySecretNamespace := upsBackup.EncryptionKeySecretNamespace
	if encryptionKeySecretNamespace == "" {
		encryptionKeySecretNamespace = upsNamespace
	}

	if upsBackup.BackendSecretName != "" {
		envVars = append(envVars,
			corev1.EnvVar{
				Name:  "BACKEND_SECRET_NAME",
				Value: upsBackup.BackendSecretName,
			},
			corev1.EnvVar{
				Name:  "BACKEND_SECRET_NAMESPACE",
				Value: backendSecretNamespace,
			},
		)
	}

	if upsBackup.EncryptionKeySecretName != "" {
		envVars = append(envVars,
			corev1.EnvVar{
				Name:  "ENCRYPTION_SECRET_NAME",
				Value: upsBackup.EncryptionKeySecretName,
			},
			corev1.EnvVar{
				Name:  "ENCRYPTION_SECRET_NAMESPACE",
				Value: encryptionKeySecretNamespace,
			},
		)
	}

	return envVars
}
