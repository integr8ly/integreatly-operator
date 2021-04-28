package marin3r

import (
	"context"
	"crypto/md5"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	marin3rconfig "github.com/integr8ly/integreatly-operator/pkg/products/marin3r/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type RateLimitServiceReconciler struct {
	Namespace       string
	RedisSecretName string
	Installation    *integreatlyv1alpha1.RHMI
	StatsdConfig    *StatsdConfig
	RateLimitConfig marin3rconfig.RateLimitConfig
}

type StatsdConfig struct {
	Host string
	Port string
}

const (
	RateLimitingConfigMapName     = "ratelimit-config"
	RateLimitingConfigMapDataName = "apicast-ratelimiting.yaml"
)

func NewRateLimitServiceReconciler(config marin3rconfig.RateLimitConfig, installation *integreatlyv1alpha1.RHMI, namespace, redisSecretName string) *RateLimitServiceReconciler {
	return &RateLimitServiceReconciler{
		RateLimitConfig: config,
		Installation:    installation,
		Namespace:       namespace,
		RedisSecretName: redisSecretName,
	}
}

// Config types, taken from unexported types in ratelimit service:
// https://github.com/envoyproxy/ratelimit/blob/master/src/config/config_impl.go#L15-L44

type yamlRateLimit struct {
	RequestsPerUnit uint32 `yaml:"requests_per_unit"`
	Unit            string `yaml:"unit"`
}

type yamlDescriptor struct {
	Key         string           `yaml:"key"`
	Value       string           `yaml:"value"`
	RateLimit   *yamlRateLimit   `yaml:"rate_limit"`
	Descriptors []yamlDescriptor `yaml:"descriptors"`
}

type yamlRoot struct {
	Domain      string           `yaml:"domain"`
	Descriptors []yamlDescriptor `yaml:"descriptors"`
}

// ReconcileRateLimitService creates the resources to deploy the rate limit service
// It reconciles a ConfigMap to configure the service, a Deployment to run it, and
// exposes it as a Service
func (r *RateLimitServiceReconciler) ReconcileRateLimitService(ctx context.Context, client k8sclient.Client, productConfig quota.ProductConfig) (integreatlyv1alpha1.StatusPhase, error) {
	phase, err := r.reconcileConfigMap(ctx, client)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	phase, err = r.reconcileDeployment(ctx, client, productConfig)
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
			Name:      RateLimitingConfigMapName,
			Namespace: r.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, cm, func() error {
		stagingconfig := yamlRoot{
			Domain: "apicast-ratelimit",
			Descriptors: []yamlDescriptor{
				{
					Key:   "generic_key",
					Value: "slowpath",
					RateLimit: &yamlRateLimit{
						Unit:            r.RateLimitConfig.Unit,
						RequestsPerUnit: r.RateLimitConfig.RequestsPerUnit,
					},
				},
			},
		}

		stagingConfigYamlMarshalled, err := yaml.Marshal(stagingconfig)
		if err != nil {
			return fmt.Errorf("failed to marshall rate limit config: %v", err)
		}

		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		if cm.Labels == nil {
			cm.Labels = map[string]string{}
		}

		cm.Data[RateLimitingConfigMapDataName] = string(stagingConfigYamlMarshalled)
		cm.Labels["app"] = quota.RateLimitName
		cm.Labels["part-of"] = "3scale-saas"
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *RateLimitServiceReconciler) reconcileDeployment(ctx context.Context, client k8sclient.Client, productConfig quota.ProductConfig) (integreatlyv1alpha1.StatusPhase, error) {
	redisSecret, err := r.getRedisSecret(ctx, client)
	if err != nil {
		if k8sError.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseAwaitingComponents, nil
		} else {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      quota.RateLimitName,
			Namespace: r.Namespace,
		},
	}

	key, err := k8sclient.ObjectKeyFromObject(deployment)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	err = client.Get(ctx, key, deployment)
	if err != nil {
		if !k8sError.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	_, err = controllerutil.CreateOrUpdate(ctx, client, deployment, func() error {
		if deployment.Labels == nil {
			deployment.Labels = map[string]string{}
		}

		deployment.Labels["app"] = quota.RateLimitName
		deployment.Spec.Selector = &v1.LabelSelector{
			MatchLabels: map[string]string{
				"app": quota.RateLimitName,
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
				Name:  "RUNTIME_ROOT",
				Value: "/srv/runtime_data",
			},
			{
				Name:  "RUNTIME_SUBDIRECTORY",
				Value: "current",
			},
			{
				Name:  "RUNTIME_WATCH_ROOT",
				Value: "false",
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
		if &deployment.Spec.Template == nil {
			deployment.Spec.Template = corev1.PodTemplateSpec{}
		}
		deployment.Spec.Template.ObjectMeta = v1.ObjectMeta{
			Labels: map[string]string{
				"app": quota.RateLimitName,
			},
		}
		if &deployment.Spec.Template.Spec == nil {
			deployment.Spec.Template.Spec = corev1.PodSpec{}
		}
		deployment.Spec.Template.Spec.PriorityClassName = r.Installation.Spec.PriorityClassName
		deployment.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: "runtime-config",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: RateLimitingConfigMapName,
						},
						Items: []corev1.KeyToPath{
							{
								Key:  "apicast-ratelimiting.yaml",
								Path: fmt.Sprintf("apicast-ratelimiting-%s.yaml", uniqueKey(r.RateLimitConfig)),
							},
						},
					},
				},
			},
		}
		if deployment.Spec.Template.Spec.Containers == nil {
			deployment.Spec.Template.Spec.Containers = []corev1.Container{{}}
		}
		deployment.Spec.Template.Spec.Containers[0].Name = quota.RateLimitName
		deployment.Spec.Template.Spec.Containers[0].Image = "quay.io/integreatly/ratelimit:v1.4.0"
		deployment.Spec.Template.Spec.Containers[0].Command = []string{quota.RateLimitName}
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
			{
				MountPath: "/srv/runtime_data/current/config",
				Name:      "runtime-config",
			},
		}
		deployment.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{
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
		}
		deployment.Spec.Template.Spec.Containers[0].Env = envs

		if err := resources.SetPodTemplate(
			resources.SelectFromDeployment,
			resources.AllMutationsOf(
				resources.MutateZoneTopologySpreadConstraints("app"),
				resources.MutateMultiAZAntiAffinity(ctx, client, "app"),
			),
			deployment,
		); err != nil {
			return fmt.Errorf("failed to set zone topology spread constraints: %w", err)
		}

		err = productConfig.Configure(deployment)
		if err != nil {
			return err
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
			Name:      quota.RateLimitName,
			Namespace: r.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, service, func() error {
		if service.Labels == nil {
			service.Labels = map[string]string{}
		}

		service.Labels["app"] = quota.RateLimitName
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
			"app": quota.RateLimitName,
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

// uniqueKey generates a unique string for each possible rate limit configuration
// combination
func uniqueKey(r marin3rconfig.RateLimitConfig) string {
	str := fmt.Sprintf("%s/%d", r.Unit, r.RequestsPerUnit)
	return fmt.Sprintf("%x", md5.Sum([]byte(str)))
}

func GetRateLimitFromConfig(c *corev1.ConfigMap) (*yamlRateLimit, error) {
	ratelimitconfig := yamlRoot{}
	err := yaml.Unmarshal([]byte(c.Data[RateLimitingConfigMapDataName]), &ratelimitconfig)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("error unmarshalling ratelimiting config from configmap '%s'", c.Name), err)
	}
	return ratelimitconfig.Descriptors[0].RateLimit, nil
}
