package config

import (
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/pkg/errors"
)

type FuseOnOpenshift struct {
	config ProductConfig
}

func NewFuseOnOpenshift(config ProductConfig) *FuseOnOpenshift {
	return &FuseOnOpenshift{config: config}
}

func (f *FuseOnOpenshift) GetNamespace() string {
	return f.config["NAMESPACE"]
}

func (f *FuseOnOpenshift) SetNamespace(newNamespace string) {
	f.config["NAMESPACE"] = newNamespace
}

func (f *FuseOnOpenshift) Read() ProductConfig {
	return f.config
}

func (f *FuseOnOpenshift) GetHost() string {
	return f.config["HOST"]
}

func (f *FuseOnOpenshift) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductFuseOnOpenshift
}

func (f *FuseOnOpenshift) GetProductVersion() v1alpha1.ProductVersion {
	return v1alpha1.VersionFuseOnOpenshift
}

func (f *FuseOnOpenshift) Validate() error {
	if f.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}

	if f.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}

	if f.GetProductVersion() == "" {
		return errors.New("config product version is not defined")
	}

	return nil
}
