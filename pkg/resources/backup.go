package resources

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"

	productsConfig "github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"

	v1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
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
	BackupServiceAccountName = "integreatly-backupjob"
	BackupClusterRoleSuffix  = "-backupjob"
)

func ReconcileBackup(ctx context.Context, serverClient pkgclient.Client, config BackupConfig, owner ownerutil.Owner) error {
	logrus.Infof("reconciling backups: %s", config.Name)
	err := reconcileClusterRole(ctx, serverClient, config, owner)
	if err != nil {
		return err
	}

	err = reconcileServiceAccount(ctx, serverClient, config)
	if err != nil {
		return err
	}

	err = reconcileClusterRoleBinding(ctx, serverClient, config, owner)
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

func reconcileClusterRole(ctx context.Context, serverClient pkgclient.Client, config BackupConfig, owner ownerutil.Owner) error {
	backupJobsClusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.Name + BackupClusterRoleSuffix,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods/exec"},
				Verbs:     []string{"create"},
			},
		},
	}
	ref := metav1.NewControllerRef(owner, owner.GetObjectKind().GroupVersionKind())
	backupJobsClusterRole.OwnerReferences = append(backupJobsClusterRole.OwnerReferences, *ref)
	return CreateOrUpdate(ctx, serverClient, backupJobsClusterRole)
}

func reconcileServiceAccount(ctx context.Context, serverClient pkgclient.Client, config BackupConfig) error {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: config.Namespace,
			Name:      BackupServiceAccountName,
		},
	}

	return CreateOrUpdate(ctx, serverClient, serviceAccount)
}

func reconcileClusterRoleBinding(ctx context.Context, serverClient pkgclient.Client, config BackupConfig, owner ownerutil.Owner) error {
	backupJobsRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.Name + BackupClusterRoleSuffix,
		},
		RoleRef: rbacv1.RoleRef{
			Name: config.Name + BackupClusterRoleSuffix,
			Kind: "ClusterRole",
		},
		Subjects: []rbacv1.Subject{
			{
				Name:      BackupServiceAccountName,
				Kind:      "ServiceAccount",
				Namespace: config.Namespace,
			},
		},
	}

	ref := metav1.NewControllerRef(owner, owner.GetObjectKind().GroupVersionKind())
	backupJobsRoleBinding.OwnerReferences = append(backupJobsRoleBinding.OwnerReferences, *ref)
	return CreateOrUpdate(ctx, serverClient, backupJobsRoleBinding)
}

func reconcileCronjobs(ctx context.Context, serverClient pkgclient.Client, config BackupConfig) error {
	for _, component := range config.Components {
		err := reconcileCronjob(ctx, serverClient, config, component)
		if err != nil {
			return fmt.Errorf("error reconciling backup job %s, for component %s: %w", config.Name, component, err)
		}
	}
	return nil
}

func reconcileCronjob(ctx context.Context, serverClient pkgclient.Client, config BackupConfig, component BackupComponent) error {
	monitoringConfig := productsConfig.NewMonitoring(productsConfig.ProductConfig{})

	cronjob := &v1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      component.Name,
			Namespace: config.Namespace,
			Labels:    map[string]string{"integreatly": "yes", monitoringConfig.GetLabelSelectorKey(): monitoringConfig.GetLabelSelector()},
		},
		Spec: v1beta1.CronJobSpec{
			Schedule:          component.Schedule,
			ConcurrencyPolicy: "Forbid",
			JobTemplate: v1beta1.JobTemplateSpec{
				Spec: v1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:   config.Name,
							Labels: map[string]string{"integreatly": "yes", "cronjob-name": component.Name},
						},
						Spec: corev1.PodSpec{
							ServiceAccountName: BackupServiceAccountName,
							RestartPolicy:      corev1.RestartPolicyOnFailure,
							Containers: []corev1.Container{
								{
									Name:            "backup-cronjob",
									Image:           "quay.io/integreatly/backup-container:master",
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
											Name:  "PRODUCT_NAMESPACE_PREFIX",
											Value: config.Namespace,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return CreateOrUpdate(ctx, serverClient, cronjob)
}

func reconcileCronjobAlerts(ctx context.Context, serverClient pkgclient.Client, config BackupConfig) error {
	monitoringConfig := productsConfig.NewMonitoring(productsConfig.ProductConfig{})

	rules := []monitoringv1.Rule{}
	for _, component := range config.Components {
		rules = append(rules, monitoringv1.Rule{
			Alert: "CronJobExists_" + config.Namespace + "_" + component.Name,
			Annotations: map[string]string{
				"sop_url": "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/alerts_and_troubleshooting.md",
				"message": "CronJob {{ $labels.namespace }}/{{ $labels.cronjob }} does not exist",
			},
			Expr:   intstr.FromString("absent(kube_cronjob_info{cronjob=\"" + component.Name + "\", namespace=\"" + config.Namespace + "\"})"),
			For:    "60s",
			Labels: map[string]string{"severity": "critical"},
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
