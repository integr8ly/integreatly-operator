package config

import (
	"errors"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

type ThreeScale struct {
	config ProductConfig
}

func NewThreeScale(config ProductConfig) *ThreeScale {
	return &ThreeScale{config: config}
}

func (t *ThreeScale) GetHost() string {
	return t.config["HOST"]
}

func (t *ThreeScale) SetHost(newHost string) {
	t.config["HOST"] = newHost
}

func (t *ThreeScale) GetNamespace() string {
	return t.config["NAMESPACE"]
}

func (t *ThreeScale) SetNamespace(newNamespace string) {
	t.config["NAMESPACE"] = newNamespace
}

func (t *ThreeScale) Read() ProductConfig {
	return t.config
}

func (t *ThreeScale) GetProductName() v1alpha1.ProductName {
	return v1alpha1.Product3Scale
}

func (t *ThreeScale) GetProductVersion() v1alpha1.ProductVersion {
	return v1alpha1.ProductVersion(t.config["VERSION"])
}

func (t *ThreeScale) SetProductVersion(newVersion string) {
	t.config["VERSION"] = newVersion
}

func (t *ThreeScale) Validate() error {
	if t.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}
	if t.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}
	if t.GetHost() == "" {
		return errors.New("config host is not defined")
	}
	return nil
}
