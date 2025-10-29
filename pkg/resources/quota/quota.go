package quota

import (
	"encoding/json"
	"fmt"
	"reflect"

	"context"

	threescalev1 "github.com/3scale/3scale-operator/apis/apps/v1alpha1"
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	marin3rconfig "github.com/integr8ly/integreatly-operator/pkg/products/marin3r/config"
	keycloak "github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"
	appsv1 "github.com/openshift/api/apps/v1"
	appsv12 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ConfigMapData               = "quota-configs"
	ConfigMapName               = "quota-config-managed-api-service"
	RateLimitName               = "ratelimit"
	BackendListenerName         = "backend_listener"
	BackendWorkerName           = "backend_worker"
	ApicastProductionName       = "apicast_production"
	ApicastStagingName          = "apicast_staging"
	KeycloakName                = "rhssouser"
	GrafanaName                 = "grafana"
	OneHundredThousandQuotaName = "100K"
	OneMillionQuotaName         = "1 Million"
	FiveMillionQuotaName        = "5 Million"
	TenMillionQuotaName         = "10 Million"
	TwentyMillionQuotaName      = "20 Million"
	FiftyMillionQuotaName       = "50 Million"
	OneHundredMillionQuotaName  = "100 Million"
)

var (
	// map of products iterate over that to build the return map
	products = map[v1alpha1.ProductName][]string{
		v1alpha1.Product3Scale: {
			BackendListenerName,
			BackendWorkerName,
			ApicastProductionName,
			ApicastStagingName,
		},
		v1alpha1.ProductRHSSOUser: {
			KeycloakName,
		},
		v1alpha1.ProductMarin3r: {
			RateLimitName,
		},
		v1alpha1.ProductGrafana: {
			GrafanaName,
		},
	}
)

type Quota struct {
	name            string
	productConfigs  map[v1alpha1.ProductName]QuotaProductConfig
	isUpdated       bool
	rateLimitConfig marin3rconfig.RateLimitConfig
}

//go:generate moq -out product_config_moq.go . ProductConfig
type ProductConfig interface {
	Configure(obj metav1.Object) error
	GetResourceConfig(ddcssName string) (corev1.ResourceRequirements, bool)
	GetReplicas(ddcssName string) int32
	GetRateLimitConfig() marin3rconfig.RateLimitConfig
	GetActiveQuota() string
}

var _ ProductConfig = QuotaProductConfig{}

type QuotaProductConfig struct {
	productName     v1alpha1.ProductName
	resourceConfigs map[string]ResourceConfig
	quota           *Quota
}

type ResourceConfig struct {
	Replicas  int32                       `json:"replicas,omitempty"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

type quotaConfigReceiver struct {
	Name      string                        `json:"name,omitempty"`
	Param     string                        `json:"param"`
	RateLimit marin3rconfig.RateLimitConfig `json:"rate-limiting,omitempty"`
	Resources map[string]ResourceConfig     `json:"resources,omitempty"`
}

func GetQuota(ctx context.Context, c client.Client, quotaParam string, QuotaConfig *corev1.ConfigMap, retQuota *Quota) error {
	allQuotas := &[]quotaConfigReceiver{}
	err := json.Unmarshal([]byte(QuotaConfig.Data[ConfigMapData]), allQuotas)
	if err != nil {
		return err
	}
	quotaReceiver := quotaConfigReceiver{}

	for _, quota := range *allQuotas {
		if quota.Param == quotaParam {
			quotaReceiver = quota
			break
		}
	}
	// if the quota receiver is empty at this point we haven't found a quota which matches the config
	// return in progress
	if quotaReceiver.Name == "" {
		return fmt.Errorf("wasn't able to find a quota in the quota config which matches the '%s' quota parameter", quotaParam)
	}

	retQuota.name = quotaReceiver.Name
	retQuota.productConfigs = map[v1alpha1.ProductName]QuotaProductConfig{}

	// loop through array of ddcss (deployment deploymentConfig StatefulSets)
	for product, ddcssNames := range products {
		pc := QuotaProductConfig{
			quota:           retQuota,
			productName:     product,
			resourceConfigs: map[string]ResourceConfig{},
		}
		for _, ddcssName := range ddcssNames {
			pc.resourceConfigs[ddcssName] = quotaReceiver.Resources[ddcssName]
		}
		retQuota.productConfigs[product] = pc
	}

	//populate rate limit configuration
	retQuota.rateLimitConfig = quotaReceiver.RateLimit
	return nil
}

func (s *Quota) GetProduct(productName v1alpha1.ProductName) QuotaProductConfig {
	// handle product not found e.g. return nil?
	return s.productConfigs[productName]
}

func (s *Quota) GetName() string {
	return s.name
}

func (s *Quota) IsUpdated() bool {
	return s.isUpdated
}

func (s *Quota) SetIsUpdated(isUpdated bool) {
	s.isUpdated = isUpdated
}

func (p QuotaProductConfig) GetResourceConfig(ddcssName string) (corev1.ResourceRequirements, bool) {
	if _, ok := p.resourceConfigs[ddcssName]; !ok {
		return corev1.ResourceRequirements{}, false
	}
	return p.resourceConfigs[ddcssName].Resources, true
}

func (p QuotaProductConfig) GetRateLimitConfig() marin3rconfig.RateLimitConfig {
	return p.quota.rateLimitConfig
}

func (s *Quota) GetRateLimitConfig() marin3rconfig.RateLimitConfig {
	return s.rateLimitConfig
}

func (p QuotaProductConfig) GetActiveQuota() string {
	return p.quota.name
}

func (p QuotaProductConfig) GetReplicas(ddcssName string) int32 {
	return p.resourceConfigs[ddcssName].Replicas
}

func (p QuotaProductConfig) Configure(obj metav1.Object) error {
	name := obj.GetName()

	switch t := obj.(type) {
	case *appsv1.DeploymentConfig:
		p.mutateReplicas(&t.Spec.Replicas, name)
		p.mutatePodTemplate(t.Spec.Template, name)
	case *appsv12.Deployment:
		checkDeploymentReplicas(t)
		p.mutateReplicas(t.Spec.Replicas, name)
		p.mutatePodTemplate(&t.Spec.Template, name)
	case *appsv12.StatefulSet:
		checkStatefulSetReplicas(t)
		p.mutateReplicas(t.Spec.Replicas, name)
		p.mutatePodTemplate(&t.Spec.Template, name)
	case *keycloak.Keycloak:
		configReplicas := p.resourceConfigs[name].Replicas
		if p.quota.isUpdated || t.Spec.Instances < int(configReplicas) {
			t.Spec.Instances = int(configReplicas)
		}
		resources := p.resourceConfigs[KeycloakName].Resources
		checkResourceBlock(&t.Spec.KeycloakDeploymentSpec.Resources)
		p.mutateResources(t.Spec.KeycloakDeploymentSpec.Resources.Requests, resources.Requests)
		p.mutateResources(t.Spec.KeycloakDeploymentSpec.Resources.Limits, resources.Limits)
	case *threescalev1.APIManager:
		checkApiManager(t)

		p.mutateAPIManagerReplicas(t.Spec.Apicast.ProductionSpec.Replicas, ApicastProductionName)
		p.mutateResourcesRequirement(t.Spec.Apicast.ProductionSpec.Resources, ApicastProductionName)

		p.mutateAPIManagerReplicas(t.Spec.Backend.ListenerSpec.Replicas, BackendListenerName)
		p.mutateResourcesRequirement(t.Spec.Backend.ListenerSpec.Resources, BackendListenerName)

		p.mutateAPIManagerReplicas(t.Spec.Backend.WorkerSpec.Replicas, BackendWorkerName)
		p.mutateResourcesRequirement(t.Spec.Backend.WorkerSpec.Resources, BackendWorkerName)

	default:
		return fmt.Errorf("quota configuration can only be applied to Deployments, StatefulSets, Deployment Configs, ApiManager, Keycloak found %s", reflect.TypeOf(obj))
	}

	return nil
}

func checkDeploymentReplicas(deployment *appsv12.Deployment) {
	if deployment.Spec.Replicas == nil {
		temp := int32(0)
		deployment.Spec.Replicas = &temp
	}
}

func checkStatefulSetReplicas(deployment *appsv12.StatefulSet) {
	if deployment.Spec.Replicas == nil {
		temp := int32(0)
		deployment.Spec.Replicas = &temp
	}
}

func (p QuotaProductConfig) mutateAPIManagerReplicas(replicas *int64, name string) {
	configReplicas := p.resourceConfigs[name].Replicas
	value := int64(configReplicas)
	if p.quota.isUpdated || *replicas < value || *replicas == 0 {
		*replicas = value
	}
}

func (p QuotaProductConfig) mutatePodTemplate(template *corev1.PodTemplateSpec, name string) {
	for i := range template.Spec.Containers {
		p.mutateResourcesRequirement(&template.Spec.Containers[i].Resources, name)
	}
}

func (p QuotaProductConfig) mutateReplicas(replicas *int32, name string) {
	configReplicas := p.resourceConfigs[name].Replicas
	if p.quota.isUpdated || *replicas < configReplicas || *replicas == 0 {
		*replicas = configReplicas
	}
}

func (p QuotaProductConfig) mutateResourcesRequirement(resourceRequirements *corev1.ResourceRequirements, name string) {
	resources := p.resourceConfigs[name].Resources

	if resourceRequirements == nil {
		resourceRequirements = &corev1.ResourceRequirements{}
	}
	checkResourceBlock(resourceRequirements)

	p.mutateResources(resourceRequirements.Limits, resources.Limits)
	p.mutateResources(resourceRequirements.Requests, resources.Requests)
}

func (p QuotaProductConfig) mutateResources(pod, cfg corev1.ResourceList) {
	podcpu := pod[corev1.ResourceCPU]
	//Cmp returns -1 if the quantity is less than y (passed value) so if podcpu is less than cfg cpu
	if p.quota.isUpdated || podcpu.Cmp(cfg[corev1.ResourceCPU]) == -1 || podcpu.IsZero() {
		quantity := cfg[corev1.ResourceCPU]
		pod[corev1.ResourceCPU] = resource.MustParse(quantity.String())
	}
	podmem := pod[corev1.ResourceMemory]
	//Cmp returns -1 if the quantity is less than y (passed value) so if podmem is less than cfg memory
	if p.quota.isUpdated || podmem.Cmp(cfg[corev1.ResourceMemory]) == -1 || podmem.IsZero() {
		quantity := cfg[corev1.ResourceMemory]
		pod[corev1.ResourceMemory] = resource.MustParse(quantity.String())
	}
}

func checkResourceBlock(resourceRequirement *corev1.ResourceRequirements) {
	if resourceRequirement.Requests == nil {
		resourceRequirement.Requests = make(map[corev1.ResourceName]resource.Quantity)
	}
	if resourceRequirement.Limits == nil {
		resourceRequirement.Limits = make(map[corev1.ResourceName]resource.Quantity)
	}
}

func checkApiManager(t *threescalev1.APIManager) {
	if t.Spec.Apicast == nil {
		t.Spec.Apicast = &threescalev1.ApicastSpec{}
	}
	if t.Spec.Apicast.ProductionSpec == nil {
		t.Spec.Apicast.ProductionSpec = &threescalev1.ApicastProductionSpec{}
	}
	if t.Spec.Apicast.StagingSpec == nil {
		t.Spec.Apicast.StagingSpec = &threescalev1.ApicastStagingSpec{}
	}
	if t.Spec.Backend == nil {
		t.Spec.Backend = &threescalev1.BackendSpec{}
	}
	if t.Spec.Backend.ListenerSpec == nil {
		t.Spec.Backend.ListenerSpec = &threescalev1.BackendListenerSpec{}
	}
	if t.Spec.Backend.WorkerSpec == nil {
		t.Spec.Backend.WorkerSpec = &threescalev1.BackendWorkerSpec{}
	}

	if t.Spec.Apicast.ProductionSpec.Replicas == nil {
		temp := int64(0)
		t.Spec.Apicast.ProductionSpec.Replicas = &temp
	}
	if t.Spec.Apicast.StagingSpec.Replicas == nil {
		temp := int64(0)
		t.Spec.Apicast.StagingSpec.Replicas = &temp
	}
	if t.Spec.Backend.ListenerSpec.Replicas == nil {
		temp := int64(0)
		t.Spec.Backend.ListenerSpec.Replicas = &temp
	}
	if t.Spec.Backend.WorkerSpec.Replicas == nil {
		temp := int64(0)
		t.Spec.Backend.WorkerSpec.Replicas = &temp
	}

	if t.Spec.Apicast.ProductionSpec.Resources == nil {
		t.Spec.Apicast.ProductionSpec.Resources = &corev1.ResourceRequirements{}
	}
	if t.Spec.Apicast.StagingSpec.Resources == nil {
		t.Spec.Apicast.StagingSpec.Resources = &corev1.ResourceRequirements{}
	}
	if t.Spec.Backend.ListenerSpec.Resources == nil {
		t.Spec.Backend.ListenerSpec.Resources = &corev1.ResourceRequirements{}
	}
	if t.Spec.Backend.WorkerSpec.Resources == nil {
		t.Spec.Backend.WorkerSpec.Resources = &corev1.ResourceRequirements{}
	}
	checkResourceBlock(t.Spec.Apicast.ProductionSpec.Resources)
	checkResourceBlock(t.Spec.Apicast.StagingSpec.Resources)
	checkResourceBlock(t.Spec.Backend.ListenerSpec.Resources)
	checkResourceBlock(t.Spec.Backend.WorkerSpec.Resources)

}
