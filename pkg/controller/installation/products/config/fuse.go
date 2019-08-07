package config

import (
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/pkg/errors"
)

func NewFuse(config ProductConfig) *Fuse {
	return &Fuse{config: config}
}

type Fuse struct {
	config ProductConfig
}

func (f *Fuse) GetNamespace() string {
	return f.config["NAMESPACE"]
}

func (f *Fuse) SetNamespace(newNamespace string) {
	f.config["NAMESPACE"] = newNamespace
}

func (f *Fuse) GetHost() string {
	return f.config["HOST"]
}

func (f *Fuse) SetHost(newHost string) {
	f.config["HOST"] = newHost
}

func (f *Fuse) Read() ProductConfig {
	return f.config
}

func (f *Fuse) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductFuse
}

func (f *Fuse) GetProductVersion() v1alpha1.ProductVersion {
	return v1alpha1.VersionFuse
}

func (f *Fuse) Validate() error {
	if f.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}

	if f.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}
	if f.GetProductVersion() == "" {
		return errors.New("version is not defined")
	}
	return nil
}
