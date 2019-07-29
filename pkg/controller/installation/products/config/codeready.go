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

func (a *CodeReady) GetHost() string {
	return a.Config["HOST"]
}

func (a *CodeReady) SetHost(newHost string) {
	a.Config["HOST"] = newHost
}

func (a *CodeReady) GetNamespace() string {
	return a.Config["NAMESPACE"]
}

func (a *CodeReady) SetNamespace(newNamespace string) {
	a.Config["NAMESPACE"] = newNamespace
}

func (a *CodeReady) Read() ProductConfig {
	return a.Config
}

func (a *CodeReady) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductCodeReadyWorkspaces
}

func (a *CodeReady) GetBackendSecretName() string {
	return "s3-credentials"
}

func (a *CodeReady) GetPostgresBackupSecretName() string {
	return "codeready-postgres-secret"
}

func (a *CodeReady) GetBackupSchedule() string {
	return "30 2 * * *"
}
