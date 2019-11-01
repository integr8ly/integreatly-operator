package mobilesecurityservicebackup

import (
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//Returns the buildCronJob object for the Mobile Security Service Backup
func (r *ReconcileMobileSecurityServiceBackup) buildCronJob(bkp *mobilesecurityservicev1alpha1.MobileSecurityServiceBackup) *v1beta1.CronJob {
	cron := &v1beta1.CronJob{
		ObjectMeta: v1.ObjectMeta{
			Name:      bkp.Name,
			Namespace: bkp.Namespace,
			Labels:    getBkpLabels(bkp.Name),
		},
		Spec: v1beta1.CronJobSpec{
			Schedule: bkp.Spec.Schedule,
			JobTemplate: v1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							ServiceAccountName: "mobile-security-service-operator",
							Containers: []corev1.Container{
								{
									Name:    bkp.Name,
									Image:   bkp.Spec.Image,
									Command: []string{"/opt/intly/tools/entrypoint.sh", "-c", "postgres", "-n", bkp.Namespace, "-b", "s3", "-e", ""},
									Env: []corev1.EnvVar{
										{
											Name:  "BACKEND_SECRET_NAME",
											Value: getAWSSecretName(bkp),
										},
										{
											Name:  "BACKEND_SECRET_NAMESPACE",
											Value: getAwsSecretNamespace(bkp),
										},
										{
											Name:  "ENCRYPTION_SECRET_NAME",
											Value: getEncSecretName(bkp),
										},
										{
											Name:  "ENCRYPTION_SECRET_NAMESPACE",
											Value: getEncSecretNamespace(bkp),
										},
										{
											Name:  "COMPONENT_SECRET_NAME",
											Value: dbSecretPrefix + bkp.Name,
										},
										{
											Name:  "COMPONENT_SECRET_NAMESPACE",
											Value: bkp.Namespace,
										},
										{
											Name:  "PRODUCT_NAME",
											Value: bkp.Spec.ProductName,
										},
									},
								},
							},
							RestartPolicy: corev1.RestartPolicyOnFailure,
						},
					},
				},
			},
		},
	}
	// Set MobileSecurityServiceDB db as the owner and controller
	controllerutil.SetControllerReference(bkp, cron, r.scheme)
	return cron
}
