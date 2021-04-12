package sku

import (
	"encoding/json"
	"errors"
	"fmt"
	threescalev1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	marin3rconfig "github.com/integr8ly/integreatly-operator/pkg/products/marin3r/config"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	appsv1 "github.com/openshift/api/apps/v1"
	appsv12 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
)

const (
	ConfigMapData         = "sku-configs"
	ConfigMapName         = "sku-config"
	RateLimitName         = "ratelimit"
	BackendListenerName   = "backend_listener"
	BackendWorkerName     = "backend_worker"
	ApicastProductionName = "apicast_production"
	ApicastStagingName    = "apicast_staging"
	KeycloakName          = "rhssouser"
	GrafanaName           = "grafana"
)

var (
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
	}
)

type SKU struct {
	name            string
	productConfigs  map[v1alpha1.ProductName]AProductConfig
	isUpdated       bool
	rateLimitConfig marin3rconfig.RateLimitConfig
}

//go:generate moq -out product_config_moq.go . ProductConfig
type ProductConfig interface {
	Configure(obj metav1.Object) error
	GetResourceConfig(ddcssName string) (corev1.ResourceRequirements, bool)
	GetReplicas(ddcssName string) int32
	GetRateLimitConfig() marin3rconfig.RateLimitConfig
}

var _ ProductConfig = AProductConfig{}

type AProductConfig struct {
	productName     v1alpha1.ProductName
	resourceConfigs map[string]ResourceConfig
	sku             *SKU
}

type ResourceConfig struct {
	Replicas  int32                       `json:"replicas,omitempty"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

type skuConfigReceiver struct {
	Name      string                        `json:"name,omitempty"`
	RateLimit marin3rconfig.RateLimitConfig `json:"rate-limiting,omitempty"`
	Resources map[string]ResourceConfig     `json:"resources,omitempty"`
}

func GetSKU(SKUId string, SKUConfig *corev1.ConfigMap, retSku *SKU, isUpdated bool) error {
	allSKUs := &[]skuConfigReceiver{}
	err := json.Unmarshal([]byte(SKUConfig.Data[ConfigMapData]), allSKUs)
	if err != nil {
		return err
	}
	skuReceiver := skuConfigReceiver{}

	for _, sku := range *allSKUs {
		if sku.Name == SKUId {
			skuReceiver = sku
			break
		}
	}

	// if the sku receiver is empty at this point we haven't found a sku which matches the config
	// return in progress
	if skuReceiver.Name == "" {
		return errors.New("wasn't able to find a sku in the sku config which matches the SKUid")
	}

	// map of products iterate over that to build the return map
	products := map[v1alpha1.ProductName][]string{
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
	retSku.name = skuReceiver.Name
	retSku.productConfigs = map[v1alpha1.ProductName]AProductConfig{}
	retSku.isUpdated = isUpdated

	// loop through array of ddcss (deployment deploymentConfig StatefulSets)
	for product, ddcssNames := range products {
		pc := AProductConfig{
			sku:             retSku,
			productName:     product,
			resourceConfigs: map[string]ResourceConfig{},
		}
		for _, ddcssName := range ddcssNames {
			pc.resourceConfigs[ddcssName] = skuReceiver.Resources[ddcssName]
		}
		retSku.productConfigs[product] = pc
	}

	//populate rate limit configuration
	retSku.rateLimitConfig = skuReceiver.RateLimit
	return nil
}

func (s *SKU) GetProduct(productName v1alpha1.ProductName) AProductConfig {
	// handle product not found e.g. return nil?
	return s.productConfigs[productName]
}

func (s *SKU) GetName() string {
	return s.name
}

func (s *SKU) IsUpdated() bool {
	return s.isUpdated
}

func (p AProductConfig) GetResourceConfig(ddcssName string) (corev1.ResourceRequirements, bool) {
	if _, ok := p.resourceConfigs[ddcssName]; !ok {
		return corev1.ResourceRequirements{}, false
	}
	return p.resourceConfigs[ddcssName].Resources, true
}

func (p AProductConfig) GetRateLimitConfig() marin3rconfig.RateLimitConfig {
	return p.sku.rateLimitConfig
}

func (p AProductConfig) GetReplicas(ddcssName string) int32 {
	return p.resourceConfigs[ddcssName].Replicas
}

func (p AProductConfig) Configure(obj metav1.Object) error {
	name := obj.GetName()

	switch t := obj.(type) {
	case *appsv1.DeploymentConfig:
		p.mutateReplicas(&t.Spec.Replicas, name)
		p.mutatePodTemplate(t.Spec.Template, name)
		break
	case *appsv12.Deployment:
		p.mutateReplicas(t.Spec.Replicas, name)
		p.mutatePodTemplate(&t.Spec.Template, name)
		break
	case *appsv12.StatefulSet:
		p.mutateReplicas(t.Spec.Replicas, name)
		p.mutatePodTemplate(&t.Spec.Template, name)
		break
	case *keycloak.Keycloak:
		configReplicas := p.resourceConfigs[name].Replicas
		if p.sku.isUpdated || t.Spec.Instances < int(configReplicas) {
			t.Spec.Instances = int(configReplicas)
		}
		resources := p.resourceConfigs[KeycloakName].Resources
		if &t.Spec.KeycloakDeploymentSpec.Resources == nil {
			t.Spec.KeycloakDeploymentSpec.Resources = corev1.ResourceRequirements{}
		}
		checkResourceBlock(&t.Spec.KeycloakDeploymentSpec.Resources)
		p.mutateResources(t.Spec.KeycloakDeploymentSpec.Resources.Requests, resources.Requests)
		p.mutateResources(t.Spec.KeycloakDeploymentSpec.Resources.Limits, resources.Limits)
		break
	case *threescalev1.APIManager:
		checkApiManager(t)

		p.mutateAPIManagerReplicas(t.Spec.Apicast.ProductionSpec.Replicas, ApicastProductionName)
		p.mutateResourcesRequirement(t.Spec.Apicast.ProductionSpec.Resources, ApicastProductionName)

		p.mutateAPIManagerReplicas(t.Spec.Apicast.StagingSpec.Replicas, ApicastProductionName)
		p.mutateResourcesRequirement(t.Spec.Apicast.StagingSpec.Resources, ApicastProductionName)

		p.mutateAPIManagerReplicas(t.Spec.Backend.ListenerSpec.Replicas, BackendListenerName)
		p.mutateResourcesRequirement(t.Spec.Backend.ListenerSpec.Resources, BackendListenerName)

		p.mutateAPIManagerReplicas(t.Spec.Backend.WorkerSpec.Replicas, BackendWorkerName)
		p.mutateResourcesRequirement(t.Spec.Backend.WorkerSpec.Resources, BackendWorkerName)

	default:
		return errors.New(fmt.Sprintf("sku configuration can only be applied to Deployments, StatefulSets or Deployment Configs, found %s", reflect.TypeOf(obj)))
	}

	return nil
}

func (p AProductConfig) mutateAPIManagerReplicas(replicas *int64, name string) {
	configReplicas := p.resourceConfigs[name].Replicas
	value := int64(configReplicas)
	if p.sku.isUpdated || *replicas < value || *replicas == 0 {
		*replicas = value
	}
}

func (p AProductConfig) mutatePodTemplate(template *corev1.PodTemplateSpec, name string) {
	for i, _ := range template.Spec.Containers {
		p.mutateResourcesRequirement(&template.Spec.Containers[i].Resources, name)
	}
}

func (p AProductConfig) mutateReplicas(replicas *int32, name string) {
	if replicas == nil {
		replicas = &[]int32{0}[0]
	}
	configReplicas := p.resourceConfigs[name].Replicas
	if p.sku.isUpdated || *replicas < configReplicas || *replicas == 0 {
		*replicas = configReplicas
	}
}

func (p AProductConfig) mutateResourcesRequirement(resourceRequirements *corev1.ResourceRequirements, name string) {
	resources := p.resourceConfigs[name].Resources

	if resourceRequirements == nil {
		resourceRequirements = &corev1.ResourceRequirements{}
	}
	checkResourceBlock(resourceRequirements)

	p.mutateResources(resourceRequirements.Limits, resources.Limits)
	p.mutateResources(resourceRequirements.Requests, resources.Requests)
}

func (p AProductConfig) mutateResources(pod, cfg corev1.ResourceList) {
	podcpu := pod[corev1.ResourceCPU]
	//Cmp returns -1 if the quantity is less than y (passed value) so if podcpu is less than cfg cpu
	if p.sku.isUpdated || podcpu.Cmp(cfg[corev1.ResourceCPU]) == -1 || podcpu.IsZero() {
		quantity := cfg[corev1.ResourceCPU]
		pod[corev1.ResourceCPU] = resource.MustParse(quantity.String())
	}
	podmem := pod[corev1.ResourceMemory]
	//Cmp returns -1 if the quantity is less than y (passed value) so if podmem is less than cfg memory
	if p.sku.isUpdated || podmem.Cmp(cfg[corev1.ResourceMemory]) == -1 || podmem.IsZero() {
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
	if &t.Spec == nil {
		t.Spec = threescalev1.APIManagerSpec{}
	}
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
		t.Spec.Apicast.ProductionSpec.Replicas = &[]int64{0}[0]
	}
	if t.Spec.Apicast.StagingSpec.Replicas == nil {
		t.Spec.Apicast.StagingSpec.Replicas = &[]int64{0}[0]
	}
	if t.Spec.Backend.ListenerSpec.Replicas == nil {
		t.Spec.Backend.ListenerSpec.Replicas = &[]int64{0}[0]
	}
	if t.Spec.Backend.WorkerSpec.Replicas == nil {
		t.Spec.Backend.WorkerSpec.Replicas = &[]int64{0}[0]
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
