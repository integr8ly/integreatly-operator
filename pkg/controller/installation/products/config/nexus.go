package config

import (
	"errors"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

type Nexus struct {
	Config ProductConfig
}

func NewNexus(config ProductConfig) *Nexus {
	return &Nexus{Config: config}
}

func (r *Nexus) GetNamespace() string {
	return r.Config["NAMESPACE"]
}

func (r *Nexus) SetNamespace(newNamespace string) {
	r.Config["NAMESPACE"] = newNamespace
}

func (r *Nexus) Read() ProductConfig {
	return r.Config
}

func (r *Nexus) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductNexus
}

func (r *Nexus) Validate() error {
	if r.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}
	if r.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}
	return nil
}
