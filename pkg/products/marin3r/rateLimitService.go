package marin3r

import (
	"context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type RateLimitServiceReconciler struct {
	Namespace       string
	RedisSecretName string
	StatsdConfig    *StatsdConfig
}

type StatsdConfig struct {
	Host string
	Port string
}

func NewRateLimitServiceReconciler(namespace, redisSecretName string) *RateLimitServiceReconciler {
	return &RateLimitServiceReconciler{
		Namespace:       namespace,
		RedisSecretName: redisSecretName,
	}
}

// Config types, taken from unexported types in ratelimit service:
// https://github.com/envoyproxy/ratelimit/blob/master/src/config/config_impl.go#L15-L44

type yamlRateLimit struct {
	RequestsPerUnit uint32 `yaml:"requests_per_unit"`
	Unit            string
}

type yamlDescriptor struct {
	Key         string
	Value       string
	RateLimit   *yamlRateLimit `yaml:"rate_limit"`
	Descriptors []yamlDescriptor
}

type yamlRoot struct {
	Domain      string
	Descriptors []yamlDescriptor
}

// ReconcileRateLimitService creates the resources to deploy the rate limit service
// It reconciles a ConfigMap to configure the service, a Deployment to run it, and
// exposes it as a Service
func (r *RateLimitServiceReconciler) ReconcileRateLimitService(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	phase, err := r.reconcileConfigMap(ctx, client)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	phase, err = r.reconcileDeployment(ctx, client)
	if phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, err
	}

	return r.reconcileService(ctx, client)
}

// WithStatsdConfig mutates r setting r.StatsdConfig to the value of config
func (r *RateLimitServiceReconciler) WithStatsdConfig(config StatsdConfig) *RateLimitServiceReconciler {
	r.StatsdConfig = &config
	return r
}

func (r *RateLimitServiceReconciler) reconcileConfigMap(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	cm := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      "ratelimit-config",
			Namespace: r.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, cm, func() error {
		configYaml := yamlRoot{
			Domain: "kuard",
			Descriptors: []yamlDescriptor{
				{
					Key:   "generic_key",
					Value: "slowpath",
					RateLimit: &yamlRateLimit{
						Unit:            "minute",
						RequestsPerUnit: 1,
					},
				},
			},
		}
		configYamlMarshalled, err := yaml.Marshal(configYaml)
		if err != nil {
			return fmt.Errorf("failed to marshall rate limit config: %v", err)
		}

		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		if cm.Labels == nil {
			cm.Labels = map[string]string{}
		}

		cm.Data["kuard.yaml"] = string(configYamlMarshalled)
		cm.Labels["app"] = "ratelimit"
		cm.Labels["part-of"] = "3scale-saas"
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *RateLimitServiceReconciler) reconcileDeployment(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	redisSecret, err := r.getRedisSecret(ctx, client)
	if err != nil {
		if errors.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseAwaitingComponents, nil
		} else {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      "ratelimit",
			Namespace: r.Namespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, client, deployment, func() error {
		if deployment.Labels == nil {
			deployment.Labels = map[string]string{}
		}

		deployment.Labels["app"] = "ratelimit"
		deployment.Spec.Selector = &v1.LabelSelector{
			MatchLabels: map[string]string{
				"app": "ratelimit",
			},
		}
		deployment.Spec.Strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		}

		useStatsd := "false"
		if r.StatsdConfig != nil {
			useStatsd = "true"
		}

		envs := []corev1.EnvVar{
			{
				Name:  "REDIS_SOCKET_TYPE",
				Value: "tcp",
			},
			{
				Name:  "REDIS_URL",
				Value: string(redisSecret.Data["URL"]),
			},
			{
				Name:  "USE_STATSD",
				Value: useStatsd,
			},
			{
				Name:  "RUNIME_ROOT",
				Value: "/srv/runtime_data/current",
			},
			{
				Name:  "RUNTIME_SUBDIRECTORY",
				Value: "/",
			},
			{
				Name:  "RUNTIME_IGNOREDOTFILES",
				Value: "true",
			},
		}

		if r.StatsdConfig != nil {
			envs = append(envs, corev1.EnvVar{
				Name:  "STATSD_PORT",
				Value: r.StatsdConfig.Port,
			}, corev1.EnvVar{
				Name:  "STATSD_HOST",
				Value: r.StatsdConfig.Host,
			})
		}

		deployment.Spec.Template = corev1.PodTemplateSpec{
			ObjectMeta: v1.ObjectMeta{
				Labels: map[string]string{
					"app": "ratelimit",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:    "ratelimit",
						Image:   "envoyproxy/ratelimit:v1.4.0",
						Command: []string{"ratelimit"},
						VolumeMounts: []corev1.VolumeMount{
							{
								MountPath: "/srv/runtime_data/current/config",
								Name:      "runtime-config",
							},
						},
						Ports: []corev1.ContainerPort{
							{
								Name:          "http",
								ContainerPort: 8080,
							},
							{
								Name:          "grpc",
								ContainerPort: 8081,
							},
							{
								Name:          "debug",
								ContainerPort: 6070,
							},
						},
						Env: envs,
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "runtime-config",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "ratelimit-config",
								},
							},
						},
					},
				},
			},
		}

		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *RateLimitServiceReconciler) reconcileService(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	service := &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      "ratelimit",
			Namespace: r.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, service, func() error {
		if service.Labels == nil {
			service.Labels = map[string]string{}
		}

		service.Labels["app"] = "ratelimit"
		service.Spec.Ports = []corev1.ServicePort{
			{
				Name:       "http",
				Protocol:   corev1.ProtocolTCP,
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			},
			{
				Name:       "debug",
				Protocol:   corev1.ProtocolTCP,
				Port:       6070,
				TargetPort: intstr.FromInt(6070),
			},
			{
				Name:       "grpc",
				Protocol:   corev1.ProtocolTCP,
				Port:       8081,
				TargetPort: intstr.FromInt(8081),
			},
		}
		service.Spec.Selector = map[string]string{
			"app": "ratelimit",
		}

		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *RateLimitServiceReconciler) getRedisSecret(ctx context.Context, client k8sclient.Client) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := client.Get(ctx, k8sclient.ObjectKey{
		Name:      r.RedisSecretName,
		Namespace: r.Namespace,
	}, secret)

	return secret, err
}
