package config

import "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

type RHSSO struct {
	config ProductConfig
}

func newRHSSO(config ProductConfig) *RHSSO {
	return &RHSSO{config: config}
}

func (a *RHSSO) GetNamespace() string {
	return a.config["NAMESPACE"]
}

func (a *RHSSO) SetNamespace(newNamespace string) {
	a.config["NAMESPACE"] = newNamespace
}

func (a *RHSSO) Read() ProductConfig {
	return a.config
}

func (a *RHSSO) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductRHSSO
}
