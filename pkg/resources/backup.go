package resources

import (
	"context"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	pkgerr "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type BackupConfig struct {
	Name             string
	Namespace        string
	Components       []string
	BackendSecret    BackupSecretLocation
	EncryptionSecret BackupSecretLocation
	ComponentSecret  BackupSecretLocation
}

type BackupSecretLocation struct {
	Name      string
	Namespace string
}

func ReconcileBackup(ctx context.Context, serverClient pkgclient.Client, inst *v1alpha1.Installation, config BackupConfig) error {
	logrus.Infof("reconciling backups: %s", config.Name)
	err := reconcileClusterRole(ctx, serverClient, inst)
	if err != nil {
		return err
	}

	err = reconcileServiceAccount(ctx, serverClient, inst, config)
	if err != nil {
		return err
	}

	err = reconcileClusterRoleBinding(ctx, serverClient, inst, config)
	if err != nil {
		return err
	}

	err = reconcileCronjobs(ctx, serverClient, inst, config)
	if err != nil {
		return err
	}
	return nil
}

func reconcileClusterRole(ctx context.Context, serverClient pkgclient.Client, inst *v1alpha1.Installation) error {
	backupJobsClusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "backupjob",
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
	ref := metav1.NewControllerRef(inst, v1alpha1.SchemaGroupVersionKind)
	backupJobsClusterRole.OwnerReferences = append(backupJobsClusterRole.OwnerReferences, *ref)
	return clusterExistsOrCreate(ctx, serverClient, "backupjob", backupJobsClusterRole)
}

func reconcileServiceAccount(ctx context.Context, serverClient pkgclient.Client, inst *v1alpha1.Installation, config BackupConfig) error {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      "backupjob",
		},
	}

	ref := metav1.NewControllerRef(inst, v1alpha1.SchemaGroupVersionKind)
	serviceAccount.OwnerReferences = append(serviceAccount.OwnerReferences, *ref)
	return existsOrCreate(ctx, serverClient, "backupjob", config.Namespace, serviceAccount)
}

func reconcileClusterRoleBinding(ctx context.Context, serverClient pkgclient.Client, config BackupConfig) error {
	backupJobsRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobname,
			Namespace: r.Config.GetNamespace(),
		},
		RoleRef: rbacv1.RoleRef{
			Name: "backupjob",
			Kind: "ClusterRole",
		},
		Subjects: []rbacv1.Subject{
			{
				Name:      "backupjob",
				Kind:      "ServiceAccount",
				Namespace: ns,
			},
		},
	}

	ref := metav1.NewControllerRef(inst, v1alpha1.SchemaGroupVersionKind)
	backupJobsRoleBinding.OwnerReferences = append(backupJobsRoleBinding.OwnerReferences, *ref)
	return existsOrCreate(ctx, serverClient, jobname, ns, backupJobsRoleBinding)
}

func reconcileCronjobs(ctx context.Context, serverClient pkgclient.Client, inst *v1alpha1.Installation, config BackupConfig) error {
	for _, component := range config.Components {
		err := reconcileCronjob(ctx, serverClient, inst, config, component)
		if err != nil {
			return pkgerr.Wrapf(err, "error reconciling backup job %s, for component %s", config.Name, component)
		}
	}
	return nil
}
func reconcileCronjob(ctx context.Context, serverClient pkgclient.Client, inst *v1alpha1.Installation, config BackupConfig, component string) error {
	cronjob := &v1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:   config.Name,
			Labels: map[string]string{"integreatly": "yes"},
		},
		Spec: v1beta1.CronJobSpec{
			Schedule:          "30 2 * * *",
			ConcurrencyPolicy: "Forbid",
			JobTemplate: v1beta1.JobTemplateSpec{
				Spec: v1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:   config.Name,
							Labels: map[string]string{"integreatly": "yes", "cronjob-name": config.Name},
						},
						Spec: corev1.PodSpec{
							ServiceAccountName: "backupjob",
							Containers: []corev1.Container{
								{
									Name:            "backup-cronjob",
									Image:           "quay.io/integreatly/backup-container:master",
									ImagePullPolicy: "Always",
									Command: []string{
										"/opt/intly/tools/entrypoint.sh",
										"-c",
										component,
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
											Value: config.ComponentSecret.Name,
										},
										{
											Name:  "COMPONENT_SECRET_NAMESPACE",
											Value: config.ComponentSecret.Namespace,
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

	ref := metav1.NewControllerRef(inst, v1alpha1.SchemaGroupVersionKind)
	cronjob.OwnerReferences = append(cronjob.OwnerReferences, *ref)
	return existsOrCreate(ctx, serverClient, jobName, ns, cronjob)
}

func existsOrCreate(ctx context.Context, serverClient pkgclient.Client, name, namespace string, obj runtime.Object) error {
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: name, Namespace: namespace}, obj)
	if err != nil && !k8serr.IsNotFound(err) {
		return pkgerr.Wrap(err, "could not get '"+name+"'")
	} else if k8serr.IsNotFound(err) {
		err = serverClient.Create(ctx, obj)
		if err != nil {
			return pkgerr.Wrap(err, "could not create '"+name+"'")
		}
	}
	return nil
}

func clusterExistsOrCreate(ctx context.Context, serverClient pkgclient.Client, name string, obj runtime.Object) error {
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: name}, obj)
	if err != nil && !k8serr.IsNotFound(err) {
		return pkgerr.Wrap(err, "could not get '"+name+"'")
	} else if k8serr.IsNotFound(err) {
		err = serverClient.Create(ctx, obj)
		if err != nil {
			return pkgerr.Wrap(err, "could not create '"+name+"'")
		}
	}
	return nil
}
