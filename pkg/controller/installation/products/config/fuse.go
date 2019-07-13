package config

import (
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/pkg/errors"
)

func newFuse(config ProductConfig) *Fuse {
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

func (f *Fuse) GetURL() string {
	return f.config["URL"]
}

func (f *Fuse) SetURL(newURL string) {
	f.config["URL"] = newURL
}

func (f *Fuse) Read() ProductConfig {
	return f.config
}

func (f *Fuse) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductFuse
}

func (f *Fuse) Validate() error {
	if f.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}

	if f.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}
	return nil
}
