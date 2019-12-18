package config

import (
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
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

func (r *CodeReady) GetLabelSelector() string {
	return "middleware"
}

func (r *CodeReady) GetTemplateList() []string {
	template_list := []string{
		"kube_state_metrics_codeready_alerts.yaml",
	}
	return template_list
}

func (c *CodeReady) SetNamespace(newNamespace string) {
	c.Config["NAMESPACE"] = newNamespace
}

func (c *CodeReady) Read() ProductConfig {
	return c.Config
}

func (c *CodeReady) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductCodeReadyWorkspaces
}

func (c *CodeReady) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.VersionCodeReadyWorkspaces
}

func (c *CodeReady) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionCodeReadyWorkspaces
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
