package sku

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	appsv1 "github.com/openshift/api/apps/v1"
	appsv12 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v13 "k8s.io/api/core/v1"
	"reflect"
)

type SKU struct {
	name           string
	productConfigs map[v1alpha1.ProductName]ProductConfig
	isUpdated      bool
}

type ProductConfig struct {
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
	err := json.Unmarshal([]byte(SKUConfig.Data["sku-configs"]), allSKUs)
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
	if skuReceiver.Name == "" {
		return errors.New(fmt.Sprintf("could not find sku config with name '%s'", SKUId))
	}

	// map of products iterate over that to build the return map
	products := map[v1alpha1.ProductName][]string{
		v1alpha1.Product3Scale: {
			"backend_listener",
			"backend_worker",
			"apicast_production",
		},
		v1alpha1.ProductRHSSOUser: {
			"keycloak",
		},
		v1alpha1.ProductMarin3r: {
			"marin3r",
		},
	}
	retSku.name = SKUId
	retSku.productConfigs = map[v1alpha1.ProductName]ProductConfig{}
	retSku.isUpdated = isUpdated

	// loop through array of ddcss (deployment deploymentConfig StatefulSets)
	for product, ddcssNames := range products {
		pc := ProductConfig{
			sku:             retSku,
			productName:     product,
			resourceConfigs: map[string]ResourceConfig{},
		}
		// if any are missing set sane defaults
		for _, ddcssName := range ddcssNames {
			pc.resourceConfigs[ddcssName] = skuReceiver.Resources[ddcssName]
		}
		retSku.productConfigs[product] = pc
	}
	return nil
}

func (s *SKU) GetProduct(productName v1alpha1.ProductName) ProductConfig {
	return s.productConfigs[productName]
}

func (s *SKU) IsUpdated() bool {
	return s.isUpdated
}

func (p *ProductConfig) GetResourceConfig(ddcssName string) corev1.ResourceRequirements {
	return p.resourceConfigs[ddcssName].Resources
}

func (p ProductConfig) GetReplicas(ddcssName string) int32 {
	return p.resourceConfigs[ddcssName].Replicas
}

func (p *ProductConfig) Configure(obj interface{}) error {
	switch t := obj.(type) {
	case *appsv1.DeploymentConfig:
		if p.sku.isUpdated && t.Spec.Replicas < p.resourceConfigs[t.ObjectMeta.Name].Replicas {
			t.Spec.Replicas = p.resourceConfigs[t.ObjectMeta.Name].Replicas
		}
		p.mutate(t.Spec.Template, t.ObjectMeta.Name)
		break
	case *appsv12.Deployment:
		configReplicas := p.resourceConfigs[t.ObjectMeta.Name].Replicas
		if p.sku.isUpdated && *t.Spec.Replicas < p.resourceConfigs[t.ObjectMeta.Name].Replicas {
			t.Spec.Replicas = &configReplicas
		}
		p.mutate(&t.Spec.Template, t.ObjectMeta.Name)
		break
	case *appsv12.StatefulSet:
		configReplicas := p.resourceConfigs[t.ObjectMeta.Name].Replicas
		if p.sku.isUpdated && *t.Spec.Replicas < p.resourceConfigs[t.ObjectMeta.Name].Replicas {
			t.Spec.Replicas = &configReplicas
		}
		p.mutate(&t.Spec.Template, t.ObjectMeta.Name)
		break
	default:
		return errors.New(fmt.Sprintf("sku configuration can only be applied to Deployments, StatefulSets or Deployment Configs, found %s", reflect.TypeOf(obj)))
	}
	return nil
}

func (p *ProductConfig) mutate(temp *v13.PodTemplateSpec, name string) {
	resources := p.resourceConfigs[name].Resources
	templateResources := &temp.Spec.Containers[0].Resources

	p.mutateResources(templateResources.Limits, resources.Limits)
	p.mutateResources(templateResources.Requests, resources.Requests)
}

func (p *ProductConfig) mutateResources(pod, cfg v13.ResourceList) {
	if p.sku.isUpdated && pod.Cpu().MilliValue() < cfg.Cpu().MilliValue() {
		pod[v13.ResourceCPU] = cfg[v13.ResourceCPU]
	}
	if p.sku.isUpdated && pod.Memory().MilliValue() < cfg.Memory().MilliValue() {
		pod[v13.ResourceMemory] = cfg[v13.ResourceMemory]
	}
}
