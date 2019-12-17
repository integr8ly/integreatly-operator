package config

import (
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
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

func (c *CloudResources) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductCloudResources
}

func (c *CloudResources) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.VersionCloudResources
}

func (c *CloudResources) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionCloudResources
}
