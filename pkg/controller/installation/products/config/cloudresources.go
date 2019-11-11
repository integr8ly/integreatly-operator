package config

import (
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

type CloudResources struct {
	Config ProductConfig
}

func NewCloudResources(config ProductConfig) *CloudResources {
	return &CloudResources{Config: config}
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

func (c *CloudResources) Read() ProductConfig {
	return c.Config
}

func (c *CloudResources) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductCloudResources
}

func (c *CloudResources) GetProductVersion() v1alpha1.ProductVersion {
	return v1alpha1.VersionCloudResources
}
