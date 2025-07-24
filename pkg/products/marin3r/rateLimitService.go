package marin3r

import (
	"context"
	"crypto/sha256"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	marin3rconfig "github.com/integr8ly/integreatly-operator/pkg/products/marin3r/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"github.com/integr8ly/integreatly-operator/pkg/resources/ratelimit"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"math"
	"reflect"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sort"
	"strconv"
)

const (
	genericKey                    = "generic_key"
	headerMatch                   = "header_match"
	headerKey                     = "tenant"
	mtUnit                        = "minute"
	possibleTenants               = 200
	multitenantLimitConfigMap     = "multitenant-config"
	multitenantRateLimit          = "mulitenantLimit"
	multitenantDescriptorValue    = "per-mt-limit"
	RateLimitingConfigMapName     = "ratelimit-config"
	RateLimitingConfigMapDataName = "apicast-ratelimiting.yaml"
	rateLimitImage                = "quay.io/kuadrant/limitador:v2.0.0"
)

type RateLimitServiceReconciler struct {
	Namespace       string
	RedisSecretName string
	Installation    *integreatlyv1alpha1.RHMI
	RateLimitConfig marin3rconfig.RateLimitConfig
	PodExecutor     resources.PodExecutorInterface
	ConfigManager   config.ConfigReadWriter
}

func NewRateLimitServiceReconciler(config marin3rconfig.RateLimitConfig, installation *integreatlyv1alpha1.RHMI, namespace, redisSecretName string, podExecutor resources.PodExecutorInterface, configManager config.ConfigReadWriter) *RateLimitServiceReconciler {
	return &RateLimitServiceReconciler{
		RateLimitConfig: config,
		Installation:    installation,
		Namespace:       namespace,
		RedisSecretName: redisSecretName,
		PodExecutor:     podExecutor,
		ConfigManager:   configManager,
	}
}

type limitadorLimit struct {
	Namespace  string   `yaml:"namespace" json:"namespace"`
	MaxValue   uint32   `yaml:"max_value" json:"max_value"`
	Seconds    uint64   `yaml:"seconds" json:"seconds"`
	Conditions []string `yaml:"conditions" json:"conditions"`
	Variables  []string `yaml:"variables" json:"variables"`
}

// ReconcileRateLimitService creates the resources to deploy the rate limit service
// It reconciles a ConfigMap to configure the service, a Deployment to run it, and
// exposes it as a Service
func (r *RateLimitServiceReconciler) ReconcileRateLimitService(ctx context.Context, client k8sclient.Client, productConfig quota.ProductConfig) (integreatlyv1alpha1.StatusPhase, error) {
	phase, err := r.reconcileConfigMap(ctx, client)
	if err != nil {
		return phase, err
	}

	phase, err = r.reconcileDeployment(ctx, client, productConfig)
	if phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileService(ctx, client)
	if phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, err
	}

	// Following notes just for Info, about deletion of ensureLimits() related code:
	// Limitador quay.io/kuadrant/limitador:v1.3.0 does not support DELETE method
	// It will pick up the changes from Config file and do the appropriate actions
	// It does not require restart the pod. In the case of a restart, repicked up the initial state from the Config file
	// So we don't need ensureLimits(), that compared RHOAM and Limitador configuration and used old limitador DELETE method
	// Slack thread with discussion - https://redhat-internal.slack.com/archives/C04J77H00TD/p1707736217609739
	// return r.ensureLimits(ctx, client)

	return integreatlyv1alpha1.PhaseCompleted, nil

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
		limitadorLimit, err := r.getLimitadorSetting(ctx, client)
		if err != nil {
			return err
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

	key := k8sclient.ObjectKeyFromObject(deployment)
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

		deployment.Spec.Template.ObjectMeta = v1.ObjectMeta{
			Labels: map[string]string{
				"app": quota.RateLimitName,
			},
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
			ProbeHandler: corev1.ProbeHandler{
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
			ProbeHandler: corev1.ProbeHandler{
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
				TargetPort: intstr.IntOrString{IntVal: 8080},
			},
			{
				Name:       "grpc",
				Protocol:   corev1.ProtocolTCP,
				Port:       8081,
				TargetPort: intstr.IntOrString{IntVal: 8081},
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

	return fmt.Sprintf("%x", sha256.Sum256([]byte(str)))
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
		return 0, fmt.Errorf("error when getting the config map %w", err)
	} else if k8sError.IsNotFound(err) {
		configMap.Data = map[string]string{}
		_, err = controllerutil.CreateOrUpdate(ctx, client, configMap, func() error {
			configMap.Data[multitenantRateLimit] = fmt.Sprint(r.getLimitPerTenant())
			return nil
		})
		if err != nil {
			return 0, fmt.Errorf("error when creating config map %w", err)
		}
	}

	limitPerTenant, err := strconv.ParseInt(configMap.Data[multitenantRateLimit], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error converting limit per tenant value %w", err)
	}
	if limitPerTenant < 0 || limitPerTenant > math.MaxUint32 {
		return 0, fmt.Errorf("rate limit value %d is out of the valid range for uint32", limitPerTenant)
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
		return "", fmt.Errorf("error when getting the config map %w", err)
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
				fmt.Sprintf(`descriptors[0]['%s'] == "%s"`, genericKey, ratelimit.RateLimitDescriptorValue),
			},
			Variables: []string{},
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
				fmt.Sprintf(`descriptors[0]['%s'] == "%s"`, genericKey, ratelimit.RateLimitDescriptorValue),
			},
			Variables: []string{},
		},
		{
			Namespace: ratelimit.RateLimitDomain,
			MaxValue:  limitPerTenant,
			Seconds:   unitInSeconds,
			Conditions: []string{
				fmt.Sprintf("%s == %s", headerMatch, multitenantDescriptorValue),
			},
			Variables: []string{
				headerKey,
			},
		},
	}, nil
}

func (r *RateLimitServiceReconciler) getLimitadorSetting(ctx context.Context, client k8sclient.Client) ([]limitadorLimit, error) {
	if !integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(r.Installation.Spec.Type)) {
		limitadorLimit, err := r.getRHOAMLimitadorSetting()
		if err != nil {
			return nil, fmt.Errorf("failed to marshall rate limit config: %v", err)
		}
		return limitadorLimit, nil
	}

	limitadorLimit, err := r.getMultitenantRHOAMLimitadorSetting(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to marshall rate limit config: %v", err)
	}

	return limitadorLimit, nil
}

func (r *RateLimitServiceReconciler) differentLimitSettings(redisLimits []limitadorLimit, currentLimits []limitadorLimit) bool {
	if len(redisLimits) != len(currentLimits) {
		return true
	}

	sortByNamespaceAndMaxValue(redisLimits)
	sortByNamespaceAndMaxValue(currentLimits)

	return !reflect.DeepEqual(redisLimits, currentLimits)
}

func sortByNamespaceAndMaxValue(elems []limitadorLimit) {
	sort.Slice(elems, func(i, j int) bool {
		if elems[i].Namespace != elems[j].Namespace {
			return elems[i].Namespace < elems[j].Namespace
		}
		return elems[i].MaxValue < elems[j].MaxValue
	})
}
