package unifiedpushserver

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"

	pushv1alpha1 "github.com/aerogear/unifiedpush-operator/pkg/apis/push/v1alpha1"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/pkg/errors"
)

func newUnifiedPushServiceAccount(cr *pushv1alpha1.UnifiedPushServer) (*corev1.ServiceAccount, error) {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Annotations: map[string]string{
				"serviceaccounts.openshift.io/oauth-redirectreference.ups": fmt.Sprintf("{\"kind\":\"OAuthRedirectReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"Route\",\"name\":\"%s-unifiedpush-proxy\"}}", cr.Name),
			},
		},
	}, nil
}

func newOauthProxyService(cr *pushv1alpha1.UnifiedPushServer) (*corev1.Service, error) {
	return &corev1.Service{
		ObjectMeta: objectMeta(cr, "unifiedpush-proxy"),
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":     cr.Name,
				"service": "ups",
			},
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Name:     "web",
					Protocol: corev1.ProtocolTCP,
					Port:     80,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 4180,
					},
				},
			},
		},
	}, nil
}

func newOauthProxyRoute(cr *pushv1alpha1.UnifiedPushServer) (*routev1.Route, error) {
	return &routev1.Route{
		ObjectMeta: objectMeta(cr, "unifiedpush-proxy"),
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: fmt.Sprintf("%s-%s", cr.Name, "unifiedpush-proxy"),
			},
			TLS: &routev1.TLSConfig{
				Termination:                   routev1.TLSTerminationEdge,
				InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyNone,
			},
		},
	}, nil
}
func newOauthProxyImageStream(cr *pushv1alpha1.UnifiedPushServer) (*imagev1.ImageStream, error) {
	return &imagev1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      cfg.OauthProxyImageStreamName,
			Labels:    labels(cr, cfg.OauthProxyImageStreamName),
		},
		Spec: imagev1.ImageStreamSpec{
			Tags: []imagev1.TagReference{
				{
					Name: cfg.OauthProxyImageStreamTag,
					From: &corev1.ObjectReference{
						Kind: "DockerImage",
						Name: cfg.OauthProxyImageStreamInitialImage,
					},
					ImportPolicy: imagev1.TagImportPolicy{
						Scheduled: false,
					},
				},
			},
		},
	}, nil
}

func buildEnv(cr *pushv1alpha1.UnifiedPushServer) []corev1.EnvVar {
	var env = []corev1.EnvVar{
		{
			Name: "POSTGRES_SERVICE_HOST",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "POSTGRES_HOST",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-postgresql", cr.Name),
					},
				},
			},
		},
		{
			Name:  "POSTGRES_SERVICE_PORT",
			Value: "5432",
		},
		{
			Name: "POSTGRES_USER",
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
			Name: "POSTGRES_PASSWORD",
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
			Name: "POSTGRES_DATABASE",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "POSTGRES_DATABASE",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-postgresql", cr.Name),
					},
				},
			},
		},
	}

	if cr.Spec.UseMessageBroker {
		env = append(env,
			corev1.EnvVar{
				Name:  "ARTEMIS_USER",
				Value: "upsuser",
			},

			corev1.EnvVar{
				Name: "ARTEMIS_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						Key: "artemis-password",
						LocalObjectReference: corev1.LocalObjectReference{
							Name: fmt.Sprintf("%s-amq", cr.Name),
						},
					},
				},
			},
			corev1.EnvVar{
				Name: "ARTEMIS_SERVICE_HOST",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						Key: "artemis-url",
						LocalObjectReference: corev1.LocalObjectReference{
							Name: fmt.Sprintf("%s-amq", cr.Name),
						},
					},
				},
			},
			corev1.EnvVar{
				Name:  "ARTEMIS_SERVICE_PORT",
				Value: "5672",
			})
	}

	return env

}

func newUnifiedPushServerDeploymentConfig(cr *pushv1alpha1.UnifiedPushServer) (*openshiftappsv1.DeploymentConfig, error) {

	labels := map[string]string{
		"app":     cr.Name,
		"service": "ups",
	}

	cookieSecret, err := generatePassword()
	if err != nil {
		return nil, errors.Wrap(err, "error generating cookie secret")
	}

	return &openshiftappsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: openshiftappsv1.DeploymentConfigSpec{
			Replicas: 1,
			Selector: labels,
			Triggers: openshiftappsv1.DeploymentTriggerPolicies{
				openshiftappsv1.DeploymentTriggerPolicy{
					Type: openshiftappsv1.DeploymentTriggerOnConfigChange,
				},
				openshiftappsv1.DeploymentTriggerPolicy{
					Type: openshiftappsv1.DeploymentTriggerOnImageChange,
					ImageChangeParams: &openshiftappsv1.DeploymentTriggerImageChangeParams{
						Automatic:      true,
						ContainerNames: []string{cfg.UPSContainerName},
						From: corev1.ObjectReference{
							Kind: "ImageStreamTag",
							Name: cfg.UPSImageStreamName + ":" + cfg.UPSImageStreamTag,
						},
					},
				},
				openshiftappsv1.DeploymentTriggerPolicy{
					Type: openshiftappsv1.DeploymentTriggerOnImageChange,
					ImageChangeParams: &openshiftappsv1.DeploymentTriggerImageChangeParams{
						Automatic:      true,
						ContainerNames: []string{cfg.OauthProxyContainerName},
						From: corev1.ObjectReference{
							Kind: "ImageStreamTag",
							Name: cfg.OauthProxyImageStreamName + ":" + cfg.OauthProxyImageStreamTag,
						},
					},
				},
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
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: cr.Name,
					InitContainers: []corev1.Container{
						{
							Name:            cfg.PostgresContainerName,
							Image:           cfg.PostgresImageStreamName + ":" + cfg.PostgresImageStreamTag,
							ImagePullPolicy: corev1.PullAlways,
							Env: []corev1.EnvVar{
								{
									Name:  "POSTGRES_SERVICE_HOST",
									Value: fmt.Sprintf("%s-postgresql", cr.Name),
								},
							},
							Command: []string{
								"/bin/sh",
								"-c",
								"source /opt/rh/rh-postgresql96/enable && until pg_isready -h $POSTGRES_SERVICE_HOST; do echo waiting for database; sleep 2; done;",
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            cfg.UPSContainerName,
							Image:           cfg.UPSImageStreamName + ":" + cfg.UPSImageStreamTag,
							ImagePullPolicy: corev1.PullAlways,
							Env:             buildEnv(cr),
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"memory": resource.MustParse("2Gi"),
									"cpu":    resource.MustParse("1"),
								},
								Requests: corev1.ResourceList{
									"memory": resource.MustParse("512Mi"),
									"cpu":    resource.MustParse("500m"),
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          cfg.UPSContainerName,
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 8080,
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/rest/applications",
										Port: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: 8080,
										},
									},
								},
								InitialDelaySeconds: 15,
								TimeoutSeconds:      2,
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/rest/applications",
										Port: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: 8080,
										},
									},
								},
								InitialDelaySeconds: 120,
								TimeoutSeconds:      10,
							},
						},
						{
							Name:            cfg.OauthProxyContainerName,
							Image:           cfg.OauthProxyImageStreamName + ":" + cfg.OauthProxyImageStreamTag,
							ImagePullPolicy: corev1.PullAlways,
							Ports: []corev1.ContainerPort{
								{
									Name:          "public",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 4180,
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"memory": resource.MustParse("64Mi"),
									"cpu":    resource.MustParse("20m"),
								},
								Requests: corev1.ResourceList{
									"memory": resource.MustParse("32Mi"),
									"cpu":    resource.MustParse("10m"),
								},
							},
							Args: []string{
								"--provider=openshift",
								fmt.Sprintf("--openshift-service-account=%s", cr.Name),
								"--upstream=http://localhost:8080",
								"--http-address=0.0.0.0:4180",
								"--skip-auth-regex=/rest/sender,/rest/registry/device,/rest/prometheus/metrics,/rest/auth/config",
								"--https-address=",
								fmt.Sprintf("--cookie-secret=%s", cookieSecret),
							},
						},
					},
				},
			},
		},
	}, nil
}

func newUnifiedPushServerService(cr *pushv1alpha1.UnifiedPushServer) (*corev1.Service, error) {
	serviceObjectMeta := objectMeta(cr, "unifiedpush")
	serviceObjectMeta.Annotations = map[string]string{
		"org.aerogear.metrics/plain_endpoint": "/rest/prometheus/metrics",
	}
	serviceObjectMeta.Labels["mobile"] = "enabled"
	serviceObjectMeta.Labels["internal"] = "unifiedpush"

	return &corev1.Service{
		ObjectMeta: serviceObjectMeta,
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":     cr.Name,
				"service": "ups",
			},
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Name:     "web",
					Protocol: corev1.ProtocolTCP,
					Port:     80,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8080,
					},
				},
			},
		},
	}, nil
}

func newUnifiedPushImageStream(cr *pushv1alpha1.UnifiedPushServer) (*imagev1.ImageStream, error) {
	return &imagev1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			Name:      cfg.UPSImageStreamName,
			Labels:    labels(cr, cfg.UPSImageStreamName),
		},
		Spec: imagev1.ImageStreamSpec{
			Tags: []imagev1.TagReference{
				{
					Name: cfg.UPSImageStreamTag,
					From: &corev1.ObjectReference{
						Kind: "DockerImage",
						Name: cfg.UPSImageStreamInitialImage,
					},
					ImportPolicy: imagev1.TagImportPolicy{
						Scheduled: false,
					},
				},
			},
		},
	}, nil
}
