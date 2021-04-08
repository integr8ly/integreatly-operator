package sku

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
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
)

type SKU struct {
	name           string
	productConfigs map[v1alpha1.ProductName]AProductConfig
	isUpdated      bool
}

//go:generate moq -out product_config_moq.go . ProductConfig
type ProductConfig interface {
	Configure(obj metav1.Object) error
	GetResourceConfig(ddcssName string) (corev1.ResourceRequirements, bool)
	GetReplicas(ddcssName string) int32
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

type RateLimit struct {
	Unit            string  `json:"unit,omitempty"`
	RequestsPerUnit int64   `json:"requests_per_unit,omitempty"`
	AlertLimits     []int64 `json:"alert_limits,omitempty"`
}

type skuConfigReceiver struct {
	Name      string                    `json:"name,omitempty"`
	RateLimit RateLimit                 `json:"rate-limiting,omitempty"`
	Resources map[string]ResourceConfig `json:"resources,omitempty"`
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
	return nil
}

func (s *SKU) GetProduct(productName v1alpha1.ProductName) AProductConfig {
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

func (p AProductConfig) GetReplicas(ddcssName string) int32 {
	return p.resourceConfigs[ddcssName].Replicas
}

func (p AProductConfig) Configure(obj metav1.Object) error {

	var replicas *int32
	var podTemplate *corev1.PodTemplateSpec
	configReplicas := p.resourceConfigs[obj.GetName()].Replicas

	switch t := obj.(type) {
	case *appsv1.DeploymentConfig:
		replicas = &t.Spec.Replicas
		podTemplate = t.Spec.Template
		break
	case *appsv12.Deployment:
		replicas = t.Spec.Replicas
		podTemplate = &t.Spec.Template
		break
	case *appsv12.StatefulSet:
		replicas = t.Spec.Replicas
		podTemplate = &t.Spec.Template
		break
	case *keycloak.Keycloak:
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
		return nil
	default:
		return errors.New(fmt.Sprintf("sku configuration can only be applied to Deployments, StatefulSets or Deployment Configs, found %s", reflect.TypeOf(obj)))
	}

	if p.sku.isUpdated || *replicas < configReplicas {
		replicas = &configReplicas
	}
	p.mutate(podTemplate, obj.GetName())
	return nil
}

func (p AProductConfig) mutate(podTemplateSpec *corev1.PodTemplateSpec, name string) {
	resources := p.resourceConfigs[name].Resources
	for i, container := range podTemplateSpec.Spec.Containers {
		if &container.Resources == nil {
			podTemplateSpec.Spec.Containers[i].Resources = corev1.ResourceRequirements{}
		}
		checkResourceBlock(&podTemplateSpec.Spec.Containers[i].Resources)
		p.mutateResources(podTemplateSpec.Spec.Containers[i].Resources.Limits, resources.Limits)
		p.mutateResources(podTemplateSpec.Spec.Containers[i].Resources.Requests, resources.Requests)
	}
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