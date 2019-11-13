package resources

import (
	"context"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	pkgerr "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
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
			return pkgerr.Wrapf(err, "error reconciling backup job %s, for component %s", config.Name, component)
		}
	}
	return nil
}
func reconcileCronjob(ctx context.Context, serverClient pkgclient.Client, config BackupConfig, component BackupComponent) error {
	cronjob := &v1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      component.Name,
			Namespace: config.Namespace,
			Labels:    map[string]string{"integreatly": "yes"},
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
