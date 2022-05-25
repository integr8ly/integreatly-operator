package marin3r

import (
	"context"
	"crypto/md5"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/resources/ratelimit"
	"strconv"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	marin3rconfig "github.com/integr8ly/integreatly-operator/pkg/products/marin3r/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/autoscaling"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
 
)

const (
	genericKey                 = "generic_key"
	headerMatch                = "header_match"
	headerKey                  = "tenant"
	mtUnit                     = "minute"
	possibleTenants            = 200
	multitenantLimitConfigMap  = "multitenant-config"
	multitenantRateLimit       = "mulitenantLimit"
	multitenantDescriptorValue = "per-mt-limit"
)

type RateLimitServiceReconciler struct {
	Namespace       string
	RedisSecretName string
	Installation    *integreatlyv1alpha1.RHMI
	RateLimitConfig marin3rconfig.RateLimitConfig
}

const (
	RateLimitingConfigMapName     = "ratelimit-config"
	RateLimitingConfigMapDataName = "apicast-ratelimiting.yaml"
	rateLimitImage                = "quay.io/3scale/limitador:v0.5.1"
)

func NewRateLimitServiceReconciler(config marin3rconfig.RateLimitConfig, installation *integreatlyv1alpha1.RHMI, namespace, redisSecretName string) *RateLimitServiceReconciler {
	return &RateLimitServiceReconciler{
		RateLimitConfig: config,
		Installation:    installation,
		Namespace:       namespace,
		RedisSecretName: redisSecretName,
	}
}

type limitadorLimit struct {
	Namespace  string   `yaml:"namespace"`
	MaxValue   uint32   `yaml:"max_value"`
	Seconds    uint64   `yaml:"seconds"`
	Conditions []string `yaml:"conditions"`
	Variables  []string `yaml:"variables"`
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

	if r.Installation.Spec.AutoscalingEnabled {
		phase, err = autoscaling.ReconcileHPA(ctx, client, quota.RateLimitName, r.Namespace, 1, *int32(1))
		if phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}
	}

	return r.reconcileService(ctx, client)
}

func (r *RateLimitServiceReconciler) reconcileConfigMap(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	var err error

	cm := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      RateLimitingConfigMapName,
			Namespace: r.Namespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, client, cm, func() error {
		var limitadorLimit []limitadorLimit

		if !integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(r.Installation.Spec.Type)) {
			limitadorLimit, err = r.getRHOAMLimitadorSetting()
			if err != nil {
				return fmt.Errorf("failed to marshall rate limit config: %v", err)
			}
		} else {
			limitadorLimit, err = r.getMultitenantRHOAMLimitadorSetting(ctx, client)
			if err != nil {
				return fmt.Errorf("failed to marshall rate limit config: %v", err)
			}
		}

		limitadorConfigYamlMarshalled, err := yaml.Marshal(limitadorLimit)
		if err != nil {
			return fmt.Errorf("failed to marshall rate limit config: %v", err)
		}

		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		if cm.Labels == nil {
			cm.Labels = map[string]string{}
		}

		cm.Data[RateLimitingConfigMapDataName] = string(limitadorConfigYamlMarshalled)
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
	currentRateLimit := ""

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

	if r.Installation.Spec.AutoscalingEnabled == true {
		
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

	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(r.Installation.Spec.Type)) {
		currentRateLimit, err = r.getCurrentLimitPerTenant(client)
		if err != nil {
			if !k8sError.IsNotFound(err) {
				return integreatlyv1alpha1.PhaseFailed, err
			}
		}
	}

	_, err = controllerutil.CreateOrUpdate(ctx, client, deployment, func() error {
		limitsFile := fmt.Sprintf("apicast-ratelimiting-%s.yaml", r.uniqueKey(r.RateLimitConfig, currentRateLimit))

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

		envs := []corev1.EnvVar{
			{
				Name:  "RUST_LOG",
				Value: "info",
			},
			{
				Name:  "REDIS_URL",
				Value: fmt.Sprintf("redis://%s", string(redisSecret.Data["URL"])),
			},
			{
				Name:  "LIMITS_FILE",
				Value: fmt.Sprintf("/srv/runtime_data/current/config/%s", limitsFile),
			},
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
								Key:  RateLimitingConfigMapDataName,
								Path: limitsFile,
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
		deployment.Spec.Template.Spec.Containers[0].Image = rateLimitImage
		// TODO - Remove after next release
		// Remove command in upgrade scenario
		deployment.Spec.Template.Spec.Containers[0].Command = nil
		// END of removal
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
		}
		deployment.Spec.Template.Spec.Containers[0].Env = envs
		deployment.Spec.Template.Spec.Containers[0].LivenessProbe = &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/status",
					Port: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "http",
					},
					Scheme: "HTTP",
				},
			},
			InitialDelaySeconds: 5,
			TimeoutSeconds:      2,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			FailureThreshold:    3,
		}

		deployment.Spec.Template.Spec.Containers[0].ReadinessProbe = &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/status",
					Port: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "http",
					},
					Scheme: "HTTP",
				},
			},
			InitialDelaySeconds: 5,
			TimeoutSeconds:      5,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			FailureThreshold:    3,
		}

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
				TargetPort: intstr.IntOrString{StrVal: "http"},
			},
			{
				Name:       "grpc",
				Protocol:   corev1.ProtocolTCP,
				Port:       8081,
				TargetPort: intstr.IntOrString{StrVal: "grpc"},
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
func (r *RateLimitServiceReconciler) uniqueKey(ratelimitConfig marin3rconfig.RateLimitConfig, currentRateLimit string) string {
	var str string

	if !integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(r.Installation.Spec.Type)) {
		str = fmt.Sprintf("%s/%d", ratelimitConfig.Unit, ratelimitConfig.RequestsPerUnit)
	} else {
		str = fmt.Sprintf("%s/%d/%s", ratelimitConfig.Unit, ratelimitConfig.RequestsPerUnit, currentRateLimit)
	}

	return fmt.Sprintf("%x", md5.Sum([]byte(str)))
}

func GetRateLimitFromConfig(c *corev1.ConfigMap) (*limitadorLimit, error) {
	var ratelimitconfig []limitadorLimit
	err := yaml.Unmarshal([]byte(c.Data[RateLimitingConfigMapDataName]), &ratelimitconfig)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("error unmarshalling ratelimiting config from configmap '%s'", c.Name), err)
	}
	return &ratelimitconfig[0], nil
}

func (r *RateLimitServiceReconciler) getUnitInSeconds(rateLimitUnit string) (uint64, error) {
	if rateLimitUnit == "second" {
		return 1, nil
	} else if rateLimitUnit == "minute" {
		return 60, nil
	} else if rateLimitUnit == "hour" {
		return 60 * 60, nil
	} else if rateLimitUnit == "day" {
		return 60 * 60 * 24, nil
	} else {
		return 0, fmt.Errorf("unexpected Rate Limit Unit %v, while getting unit in seconds", rateLimitUnit)
	}
}

func GetSecondsInUnit(seconds uint64) (string, error) {
	if seconds == 1 {
		return "second", nil
	} else if seconds == 60 {
		return "minute", nil
	} else if seconds == (60 * 60) {
		return "hour", nil
	} else if seconds == (60 * 60 * 24) {
		return "day", nil
	} else {
		return "", fmt.Errorf("unexpected seconds value: %v, while getting seconds in Rate Limit Unit", seconds)
	}
}

func (r *RateLimitServiceReconciler) getLimitPerTenantFromConfigMap(client k8sclient.Client, ctx context.Context) (uint32, error) {

	configMap := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      multitenantLimitConfigMap,
			Namespace: r.Namespace,
		},
	}

	err := client.Get(context.TODO(), types.NamespacedName{Namespace: r.Namespace, Name: multitenantLimitConfigMap}, configMap)
	if err != nil && !k8sError.IsNotFound(err) {
		return 0, fmt.Errorf("Error when getting the config map %w", err)
	} else if k8sError.IsNotFound(err) {
		configMap.Data = map[string]string{}
		_, err = controllerutil.CreateOrUpdate(ctx, client, configMap, func() error {
			configMap.Data[multitenantRateLimit] = fmt.Sprint(r.getLimitPerTenant())
			return nil
		})
		if err != nil {
			return 0, fmt.Errorf("Error when creating config map %w", err)
		}
	}

	limitPerTenant, err := strconv.ParseInt(configMap.Data[multitenantRateLimit], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("Error converting limit per tenant value %w", err)
	}

	return uint32(limitPerTenant), nil
}

func (r *RateLimitServiceReconciler) getLimitPerTenant() uint32 {
	limitPerTenant := r.RateLimitConfig.RequestsPerUnit / possibleTenants
	// Ensure there is a 200 per tenant limit at least
	if limitPerTenant < 200 {
		limitPerTenant = 200
	}

	return uint32(limitPerTenant)
}

func (r *RateLimitServiceReconciler) getCurrentLimitPerTenant(client k8sclient.Client) (string, error) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      multitenantLimitConfigMap,
			Namespace: r.Namespace,
		},
	}

	err := client.Get(context.TODO(), types.NamespacedName{Namespace: r.Namespace, Name: multitenantLimitConfigMap}, configMap)
	if err != nil {
		return "", fmt.Errorf("Error when getting the config map %w", err)
	}

	currentLimit := configMap.Data[multitenantRateLimit]

	return currentLimit, nil
}

func (r *RateLimitServiceReconciler) getRHOAMLimitadorSetting() ([]limitadorLimit, error) {
	unitInSeconds, err := r.getUnitInSeconds(r.RateLimitConfig.Unit)
	if err != nil {
		return nil, err
	}

	return []limitadorLimit{
		{
			Namespace: ratelimit.RateLimitDomain,
			MaxValue:  r.RateLimitConfig.RequestsPerUnit,
			Seconds:   unitInSeconds,
			Conditions: []string{
				fmt.Sprintf("generic_key == %s", ratelimit.RateLimitDescriptorValue),
			},
			Variables: []string{
				"generic_key",
			},
		},
	}, nil
}

func (r *RateLimitServiceReconciler) getMultitenantRHOAMLimitadorSetting(ctx context.Context, client k8sclient.Client) ([]limitadorLimit, error) {

	limitPerTenant, err := r.getLimitPerTenantFromConfigMap(client, ctx)
	if err != nil {
		return nil, err
	}

	unitInSeconds, err := r.getUnitInSeconds(r.RateLimitConfig.Unit)
	if err != nil {
		return nil, err
	}

	return []limitadorLimit{
		{
			Namespace: ratelimit.RateLimitDomain,
			MaxValue:  r.RateLimitConfig.RequestsPerUnit,
			Seconds:   unitInSeconds,
			Conditions: []string{
				fmt.Sprintf("generic_key == %s", ratelimit.RateLimitDescriptorValue),
			},
			Variables: []string{
				"generic_key",
			},
		},
		{
			Namespace: ratelimit.RateLimitDomain,
			MaxValue:  limitPerTenant,
			Seconds:   unitInSeconds,
			Conditions: []string{
				fmt.Sprintf("header_match == %s", multitenantDescriptorValue),
			},
			Variables: []string{
				"tenant",
			},
		},
	}, nil
}
