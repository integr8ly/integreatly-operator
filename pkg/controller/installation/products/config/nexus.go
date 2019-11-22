package config

import (
	"errors"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

type Nexus struct {
	config ProductConfig
}

func NewNexus(config ProductConfig) *Nexus {
	return &Nexus{config: config}
}

func (n *Nexus) GetNamespace() string {
	return n.config["NAMESPACE"]
}

func (n *Nexus) SetNamespace(newNamespace string) {
	n.config["NAMESPACE"] = newNamespace
}

func (n *Nexus) Read() ProductConfig {
	return n.config
}

func (n *Nexus) GetHost() string {
	return n.config["HOST"]
}

func (n *Nexus) SetHost(newHost string) {
	n.config["HOST"] = newHost
}

func (n *Nexus) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductNexus
}

func (n *Nexus) GetProductVersion() v1alpha1.ProductVersion {
	return v1alpha1.VersionNexus
}

func (n *Nexus) GetOperatorVersion() v1alpha1.OperatorVersion {
	return v1alpha1.OperatorVersionNexus
}

func (n *Nexus) Validate() error {
	if n.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}
	if n.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}
	return nil
}
