package config

import (
	"errors"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

type Ups struct {
	config ProductConfig
}

func NewUps(config ProductConfig) *Ups {
	return &Ups{config: config}
}

func (u *Ups) GetHost() string {
	return u.config["HOST"]
}

func (u *Ups) SetHost(newHost string) {
	u.config["HOST"] = newHost
}

func (u *Ups) GetNamespace() string {
	return u.config["NAMESPACE"]
}

func (u *Ups) SetNamespace(newNamespace string) {
	u.config["NAMESPACE"] = newNamespace
}

func (u *Ups) Read() ProductConfig {
	return u.config
}

func (u *Ups) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductUps
}

func (u *Ups) GetProductVersion() v1alpha1.ProductVersion {
	return v1alpha1.VersionUps
}

func (u *Ups) Validate() error {
	if u.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}
	if u.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}
	if u.GetHost() == "" {
		return errors.New("config host is not defined")
	}
	return nil
}
