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
	KeycloakName          = "keycloak"
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

	//if the skuid is empty then build up a defaulting skureceiver
	// for defaulting we can use the twenty million sku
	if skuReceiver.Name == "" {
		skuReceiver.Name = "default"
		skuReceiver.Resources = map[string]ResourceConfig{
			BackendListenerName: {
				Replicas: 3,
				Resources: v13.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("0.35"),
						corev1.ResourceMemory: resource.MustParse("450"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("0.45"),
						corev1.ResourceMemory: resource.MustParse("500"),
					},
				},
			},
			BackendWorkerName: {
				Replicas: 3,
				Resources: v13.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("0.15"),
						corev1.ResourceMemory: resource.MustParse("100"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("0.2"),
						corev1.ResourceMemory: resource.MustParse("100"),
					},
				},
			},
			ApicastProductionName: {
				Replicas: 3,
				Resources: v13.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("0.3"),
						corev1.ResourceMemory: resource.MustParse("250"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("0.3"),
						corev1.ResourceMemory: resource.MustParse("300"),
					},
				},
			},
			ApicastStagingName: {
				Replicas: 3,
				Resources: v13.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("0.3"),
						corev1.ResourceMemory: resource.MustParse("250"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("0.3"),
						corev1.ResourceMemory: resource.MustParse("300"),
					},
				},
			},
			KeycloakName: {
				Replicas: 3,
				Resources: v13.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("0.75"),
						corev1.ResourceMemory: resource.MustParse("1500"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("0.75"),
						corev1.ResourceMemory: resource.MustParse("1500"),
					},
				},
			},
			RateLimitName: {
				Replicas:  3,
				Resources: v13.ResourceRequirements{},
			},
		}
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

func (s *SKU) GetName() string {
	return s.name
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

func (p *ProductConfig) Configure(obj metav1.Object) error {

	var replicas *int32
	var podTemplate *v13.PodTemplateSpec

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

	default:
		return errors.New(fmt.Sprintf("sku configuration can only be applied to Deployments, StatefulSets or Deployment Configs, found %s", reflect.TypeOf(obj)))
	}

	configReplicas := p.resourceConfigs[obj.GetName()].Replicas
	if p.sku.isUpdated && *replicas < configReplicas {
		*replicas = configReplicas
	}
	p.mutate(podTemplate, obj.GetName())

	return nil
}

func (p *ProductConfig) mutate(temp *v13.PodTemplateSpec, name string) {
	resources := p.resourceConfigs[name].Resources
	templateResources := &temp.Spec.Containers[0].Resources

	p.mutateResources(templateResources.Limits, resources.Limits)
	p.mutateResources(templateResources.Requests, resources.Requests)
}

func (p *ProductConfig) mutateResources(pod, cfg v13.ResourceList) {
	if p.sku.isUpdated {
		pod[v13.ResourceCPU] = cfg[v13.ResourceCPU]
	}
	if p.sku.isUpdated {
		pod[v13.ResourceMemory] = cfg[v13.ResourceMemory]
	}
}
