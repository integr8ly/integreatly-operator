package resources

import (
	"context"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	productsConfig "github.com/integr8ly/integreatly-operator/pkg/config"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type BackupConfig struct {
	Name             string
	Namespace        string
	Components       []BackupComponent
	BackendSecret    BackupSecretLocation
	EncryptionSecret BackupSecretLocation
}

type BackupComponent struct {
	Name     string
	Type     string
	Secret   BackupSecretLocation
	Schedule string
}

type BackupSecretLocation struct {
	Name      string
	Namespace string
}

var (
	BackupServiceAccountName = "rhmi-backupjob"
	BackupRoleName           = "rhmi-backupjob"
	BackupRoleBindingName    = "rhmi-backupjob"
)

func ReconcileBackup(ctx context.Context, serverClient k8sclient.Client, config BackupConfig, configManager productsConfig.ConfigReadWriter, log l.Logger) error {
	log.Infof("reconciling backups", l.Fields{"configMap": config.Name})

	err := reconcileBackendSecret(ctx, serverClient, config, configManager.GetBackupsSecretName(), configManager.GetOperatorNamespace())
	if err != nil {
		return err
	}

	err = reconcileRole(ctx, serverClient, config)
	if err != nil {
		return err
	}

	err = reconcileServiceAccount(ctx, serverClient, config)
	if err != nil {
		return err
	}

	err = reconcileRoleBinding(ctx, serverClient, config)
	if err != nil {
		return err
	}

	err = reconcileCronjobs(ctx, serverClient, config)
	if err != nil {
		return err
	}

	err = reconcileCronjobAlerts(ctx, serverClient, config)
	if err != nil {
		return err
	}
	return nil
}

func reconcileBackendSecret(ctx context.Context, serverClient k8sclient.Client, config BackupConfig, secretName string, secretNamespace string) error {
	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespace,
		},
	}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Namespace: sourceSecret.Namespace, Name: sourceSecret.Name}, sourceSecret)
	if err != nil {
		return fmt.Errorf("Could not get secret that contains S3 credentials for backup CronJobs - %s Secret from %s namespace: %w", sourceSecret.Name, sourceSecret.Namespace, err)
	}

	destinationSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.BackendSecret.Name,
			Namespace: config.BackendSecret.Namespace,
		},
	}
	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, destinationSecret, func() error {
		// Transforming from Secret field names of CRO to the names consumed by our scripts:
		// https://github.com/integr8ly/backup-container-image/blob/master/image/tools/lib/backend/s3.sh#L10-L20
		destinationSecret.Data = map[string][]byte{
			"AWS_ACCESS_KEY_ID":     sourceSecret.Data["credentialKeyID"],
			"AWS_SECRET_ACCESS_KEY": sourceSecret.Data["credentialSecretKey"],
			"AWS_S3_BUCKET_NAME":    sourceSecret.Data["bucketName"],
			"AWS_S3_REGION":         sourceSecret.Data["bucketRegion"],
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Could not %s backup Secret %s in %s namespace: %w", or, destinationSecret.Name, destinationSecret.Namespace, err)
	}

	return nil
}

func reconcileRole(ctx context.Context, serverClient k8sclient.Client, config BackupConfig) error {
	backupJobsRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BackupRoleName,
			Namespace: config.Namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, backupJobsRole, func() error {
		backupJobsRole.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods", "secrets"},
				Verbs:     []string{"get", "list"},
			}, {
				APIGroups: []string{"admin.enmasse.io"},
				Resources: []string{
					"addressplans",
					"addressspaceplans",
					"authenticationservices",
					"brokeredinfraconfigs",
					"standardinfraconfigs",
				},
				Verbs: []string{"get", "list"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods/exec"},
				Verbs:     []string{"create"},
			},
		}
		return nil
	})
	return err
}

func reconcileServiceAccount(ctx context.Context, serverClient k8sclient.Client, config BackupConfig) error {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: config.Namespace,
			Name:      BackupServiceAccountName,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, serviceAccount, func() error {
		return nil
	})
	return err
}

func reconcileRoleBinding(ctx context.Context, serverClient k8sclient.Client, config BackupConfig) error {
	backupJobsRoleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BackupRoleBindingName,
			Namespace: config.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, backupJobsRoleBinding, func() error {
		backupJobsRoleBinding.RoleRef = rbacv1.RoleRef{
			Name: BackupRoleName,
			Kind: "Role",
		}
		backupJobsRoleBinding.Subjects = []rbacv1.Subject{
			{
				Name:      BackupServiceAccountName,
				Kind:      "ServiceAccount",
				Namespace: config.Namespace,
			},
		}
		return nil
	})
	return err
}

func reconcileCronjobs(ctx context.Context, serverClient k8sclient.Client, config BackupConfig) error {
	for _, component := range config.Components {
		err := reconcileCronjob(ctx, serverClient, config, component)
		if err != nil {
			return fmt.Errorf("error reconciling backup job %s, for component %s: %w", config.Name, component, err)
		}
	}
	return nil
}

func reconcileCronjob(ctx context.Context, serverClient k8sclient.Client, config BackupConfig, component BackupComponent) error {
	monitoringConfig := productsConfig.NewMonitoring(productsConfig.ProductConfig{})

	cronjob := &batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      component.Name,
			Namespace: config.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, cronjob, func() error {
		cronjob.Labels = map[string]string{"integreatly": "yes", monitoringConfig.GetLabelSelectorKey(): monitoringConfig.GetLabelSelector()}
		cronjob.Spec = batchv1beta1.CronJobSpec{
			Schedule:          component.Schedule,
			ConcurrencyPolicy: "Forbid",
			JobTemplate: batchv1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:   config.Name,
							Labels: map[string]string{"integreatly": "yes", "cronjob-name": component.Name, "monitoring_key": "middleware"},
						},
						Spec: corev1.PodSpec{
							ServiceAccountName: BackupServiceAccountName,
							RestartPolicy:      corev1.RestartPolicyOnFailure,
							Containers: []corev1.Container{
								{
									Name:            "backup-cronjob",
									Image:           "quay.io/integreatly/backup-container:1.0.16",
									ImagePullPolicy: "Always",
									Command: []string{
										"/opt/intly/tools/entrypoint.sh",
										"-c",
										component.Type,
										"-b",
										"s3",
										"-e",
										"",
										"-d",
										"",
									},
									Env: []corev1.EnvVar{
										{
											Name:  "BACKEND_SECRET_NAME",
											Value: config.BackendSecret.Name,
										},
										{
											Name:  "BACKEND_SECRET_NAMESPACE",
											Value: config.BackendSecret.Namespace,
										},
										{
											Name:  "ENCRYPTION_SECRET_NAME",
											Value: config.EncryptionSecret.Name,
										},
										{
											Name:  "ENCRYPTION_SECRET_NAMESPACE",
											Value: config.EncryptionSecret.Namespace,
										},
										{
											Name:  "COMPONENT_SECRET_NAME",
											Value: component.Secret.Name,
										},
										{
											Name:  "COMPONENT_SECRET_NAMESPACE",
											Value: component.Secret.Namespace,
										},
										{
											Name:  "PRODUCT_NAME",
											Value: config.Name,
										},
										{
											Name:  "PRODUCT_NAMESPACE",
											Value: config.Namespace,
										},
									},
								},
							},
						},
					},
				},
			},
		}
		return nil
	})
	return err
}

func reconcileCronjobAlerts(ctx context.Context, serverClient k8sclient.Client, config BackupConfig) error {
	monitoringConfig := productsConfig.NewMonitoring(productsConfig.ProductConfig{})

	rules := []monitoringv1.Rule{}
	for _, component := range config.Components {
		rules = append(rules, monitoringv1.Rule{
			Alert: "CronJobExists_" + config.Namespace + "_" + component.Name,
			Annotations: map[string]string{
				"sop_url": SopUrlAlertsAndTroubleshooting,
				"message": "CronJob {{ $labels.namespace }}/{{ $labels.cronjob }} does not exist",
			},
			Expr:   intstr.FromString("absent(kube_cronjob_info{cronjob=\"" + component.Name + "\", namespace=\"" + config.Namespace + "\"})"),
			For:    "5m",
			Labels: map[string]string{"severity": "warning"},
		})
	}

	rule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "backupjobs-exist-alerts",
			Namespace: config.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, rule, func() error {
		rule.ObjectMeta.Labels = map[string]string{"integreatly": "yes", monitoringConfig.GetLabelSelectorKey(): monitoringConfig.GetLabelSelector()}
		rule.Spec = monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				monitoringv1.RuleGroup{
					Name:  "general.rules",
					Rules: rules,
				},
			},
		}
		return nil
	})
	return err
}
