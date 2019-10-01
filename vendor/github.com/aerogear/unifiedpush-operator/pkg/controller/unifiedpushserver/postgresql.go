package unifiedpushserver

import (
	"fmt"

	pushv1alpha1 "github.com/aerogear/unifiedpush-operator/pkg/apis/push/v1alpha1"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func newPostgresqlPersistentVolumeClaim(cr *pushv1alpha1.UnifiedPushServer) (*corev1.PersistentVolumeClaim, error) {
	pvcSize, err := resource.ParseQuantity("5Gi")
	if err != nil {
		return nil, errors.Wrap(err, "error parsing PostgreSQL PVC storage size")
	}

	return &corev1.PersistentVolumeClaim{
		ObjectMeta: objectMeta(cr, "postgresql"),
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: pvcSize,
				},
			},
		},
	}, nil
}

func newPostgresqlSecret(cr *pushv1alpha1.UnifiedPushServer) (*corev1.Secret, error) {
	databasePassword, err := generatePassword()
	if err != nil {
		return nil, err
	}

	return &corev1.Secret{
		ObjectMeta: objectMeta(cr, "postgresql"),
		StringData: map[string]string{
			"POSTGRES_DATABASE":  "unifiedpush",
			"POSTGRES_USERNAME":  "unifiedpush",
			"POSTGRES_PASSWORD":  databasePassword,
			"POSTGRES_HOST":      fmt.Sprintf("%s-postgresql.%s.svc", cr.Name, cr.Namespace),
			"POSTGRES_SUPERUSER": "false",
			"POSTGRES_VERSION":   cfg.PostgresImageStreamTag,
		},
	}, nil
}

func newPostgresqlDeploymentConfig(cr *pushv1alpha1.UnifiedPushServer) (*openshiftappsv1.DeploymentConfig, error) {

	return &openshiftappsv1.DeploymentConfig{
		ObjectMeta: objectMeta(cr, "postgresql"),
		Spec: openshiftappsv1.DeploymentConfigSpec{
			Replicas: 1,
			Selector: labels(cr, "postgresql"),
			Strategy: openshiftappsv1.DeploymentStrategy{
				Type: openshiftappsv1.DeploymentStrategyTypeRecreate,
			},
			Triggers: openshiftappsv1.DeploymentTriggerPolicies{
				openshiftappsv1.DeploymentTriggerPolicy{
					Type: openshiftappsv1.DeploymentTriggerOnImageChange,
					ImageChangeParams: &openshiftappsv1.DeploymentTriggerImageChangeParams{
						Automatic:      true,
						ContainerNames: []string{cfg.PostgresContainerName},
						From: corev1.ObjectReference{
							Kind:      "ImageStreamTag",
							Namespace: cfg.PostgresImageStreamNamespace,
							Name:      cfg.PostgresImageStreamName + ":" + cfg.PostgresImageStreamTag,
						},
					},
				},
			},
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels(cr, "postgresql"),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            cfg.PostgresContainerName,
							Image:           cfg.PostgresImageStreamName + ":" + cfg.PostgresImageStreamTag,
							ImagePullPolicy: corev1.PullAlways,
							Env: []corev1.EnvVar{
								{
									Name: "POSTGRESQL_USER",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											Key: "POSTGRES_USERNAME",
											LocalObjectReference: corev1.LocalObjectReference{
												Name: fmt.Sprintf("%s-postgresql", cr.Name),
											},
										},
									},
								},
								{
									Name: "POSTGRESQL_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											Key: "POSTGRES_PASSWORD",
											LocalObjectReference: corev1.LocalObjectReference{
												Name: fmt.Sprintf("%s-postgresql", cr.Name),
											},
										},
									},
								},
								{
									Name: "POSTGRESQL_DATABASE",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											Key: "POSTGRES_DATABASE",
											LocalObjectReference: corev1.LocalObjectReference{
												Name: fmt.Sprintf("%s-postgresql", cr.Name),
											},
										},
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          cfg.PostgresContainerName,
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 5432,
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"/bin/sh",
											"-i",
											"-c",
											"psql -h 127.0.0.1 -U $POSTGRESQL_USER -q -d $POSTGRESQL_DATABASE -c 'SELECT 1'",
										},
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: 5432,
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("512Mi"),
									corev1.ResourceCPU:    resource.MustParse("1"),
								}, Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("256Mi"),
									corev1.ResourceCPU:    resource.MustParse("250m"),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								corev1.VolumeMount{
									Name:      fmt.Sprintf("%s-postgresql-data", cr.Name),
									MountPath: "/var/lib/pgsql/data",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						corev1.Volume{
							Name: fmt.Sprintf("%s-postgresql-data", cr.Name),
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: fmt.Sprintf("%s-postgresql", cr.Name),
								},
							},
						},
					},
				},
			},
		},
	}, nil
}

func newPostgresqlService(cr *pushv1alpha1.UnifiedPushServer) (*corev1.Service, error) {
	return &corev1.Service{
		ObjectMeta: objectMeta(cr, "postgresql"),
		Spec: corev1.ServiceSpec{
			Selector: labels(cr, "postgresql"),
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Name:     "postgresql",
					Protocol: corev1.ProtocolTCP,
					Port:     5432,
				},
			},
		},
	}, nil
}
