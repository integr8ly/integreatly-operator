package config

import (
	"errors"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

type RHSSO struct {
	Config ProductConfig
}

func NewRHSSO(config ProductConfig) *RHSSO {
	return &RHSSO{Config: config}
}

func (r *RHSSO) GetNamespace() string {
	return r.Config["NAMESPACE"]
}

func (r *RHSSO) SetNamespace(newNamespace string) {
	r.Config["NAMESPACE"] = newNamespace
}

func (r *RHSSO) GetRealm() string {
	return r.Config["REALM"]
}

func (r *RHSSO) SetRealm(newRealm string) {
	r.Config["REALM"] = newRealm
}

func (r *RHSSO) GetURL() string {
	return r.Config["URL"]
}

func (r *RHSSO) SetURL(newURL string) {
	r.Config["URL"] = newURL
}

func (r *RHSSO) Read() ProductConfig {
	return r.Config
}

func (r *RHSSO) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductRHSSO
}

func (r *RHSSO) Validate() error {
	if r.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}
	if r.GetRealm() == "" {
		return errors.New("config realm is not defined")
	}
	if r.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}
	if r.GetURL() == "" {
		return errors.New("config url is not defined")
	}
	return nil
}
