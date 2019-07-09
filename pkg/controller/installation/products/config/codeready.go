package config

import (
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

type CodeReady struct {
	config ProductConfig
}

func newCodeReady(config ProductConfig) *CodeReady {
	return &CodeReady{config: config}
}

func (a *CodeReady) GetHost() string {
	return a.config["HOST"]
}

func (a *CodeReady) SetHost(newHost string) {
	a.config["HOST"] = newHost
}

func (a *CodeReady) GetNamespace() string {
	return a.config["NAMESPACE"]
}

func (a *CodeReady) SetNamespace(newNamespace string) {
	a.config["NAMESPACE"] = newNamespace
}

func (a *CodeReady) Read() ProductConfig {
	return a.config
}

func (a *CodeReady) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductCodeReadyWorkspaces
}
