package config

import (
	crov1alpha1 "github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type CloudResources struct {
	Config ProductConfig
}

func NewCloudResources(config ProductConfig) *CloudResources {
	return &CloudResources{Config: config}
}

func (c *CloudResources) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{
		&crov1alpha1.Postgres{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Postgres",
				APIVersion: crov1alpha1.GroupVersion.String(),
			},
		},
		&crov1alpha1.Redis{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Redis",
				APIVersion: crov1alpha1.GroupVersion.String(),
			},
		},
		&crov1alpha1.BlobStorage{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BlobStorage",
				APIVersion: crov1alpha1.GroupVersion.String(),
			},
		},
	}
}

func (c *CloudResources) GetHost() string {
	return c.Config["HOST"]
}

func (c *CloudResources) SetHost(newHost string) {
	c.Config["HOST"] = newHost
}

func (c *CloudResources) GetNamespace() string {
	return c.Config["NAMESPACE"]
}

func (c *CloudResources) SetNamespace(newNamespace string) {
	c.Config["NAMESPACE"] = newNamespace
}

func (c *CloudResources) GetOperatorNamespace() string {
	return c.Config["OPERATOR_NAMESPACE"]
}

func (c *CloudResources) SetOperatorNamespace(newNamespace string) {
	c.Config["OPERATOR_NAMESPACE"] = newNamespace
}

func (c *CloudResources) SetStrategiesConfigMapName(strategiesConfigMapName string) {
	c.Config["STRATEGIES_CONFIG_MAP_NAME"] = strategiesConfigMapName
}

func (c *CloudResources) GetStrategiesConfigMapName() string {
	return c.Config["STRATEGIES_CONFIG_MAP_NAME"]
}

func (c *CloudResources) Read() ProductConfig {
	return c.Config
}

func (c *CloudResources) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductCloudResources
}

func (c *CloudResources) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.VersionCloudResources
}

func (c *CloudResources) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionCloudResources
}
