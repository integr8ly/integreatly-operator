package config

import (
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

type CodeReady struct {
	Config ProductConfig
}

func NewCodeReady(config ProductConfig) *CodeReady {
	return &CodeReady{Config: config}
}

func (c *CodeReady) GetHost() string {
	return c.Config["HOST"]
}

func (c *CodeReady) SetHost(newHost string) {
	c.Config["HOST"] = newHost
}

func (c *CodeReady) GetNamespace() string {
	return c.Config["NAMESPACE"]
}

func (c *CodeReady) SetNamespace(newNamespace string) {
	c.Config["NAMESPACE"] = newNamespace
}

func (c *CodeReady) Read() ProductConfig {
	return c.Config
}

func (c *CodeReady) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductCodeReadyWorkspaces
}

func (c *CodeReady) GetProductVersion() v1alpha1.ProductVersion {
	return v1alpha1.VersionCodeReadyWorkspaces
}

func (c *CodeReady) GetOperatorVersion() v1alpha1.OperatorVersion {
	return v1alpha1.OperatorVersionCodeReadyWorkspaces
}

func (c *CodeReady) GetBackendSecretName() string {
	return "s3-credentials"
}

func (c *CodeReady) GetPostgresBackupSecretName() string {
	return "codeready-postgres-secret"
}

func (c *CodeReady) GetBackupSchedule() string {
	return "30 2 * * *"
}
