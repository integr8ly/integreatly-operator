package config

import (
	"errors"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

type AMQOnline struct {
	config ProductConfig
}

func NewAMQOnline(config ProductConfig) *AMQOnline {
	return &AMQOnline{config: config}
}

func (a *AMQOnline) GetHost() string {
	return a.config["HOST"]
}

func (a *AMQOnline) SetHost(newHost string) {
	a.config["HOST"] = newHost
}

func (a *AMQOnline) GetNamespace() string {
	return a.config["NAMESPACE"]
}

func (a *AMQOnline) SetNamespace(newNamespace string) {
	a.config["NAMESPACE"] = newNamespace
}

func (a *AMQOnline) Read() ProductConfig {
	return a.config
}

func (a *AMQOnline) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductAMQOnline
}

func (a *AMQOnline) Validate() error {
	if a.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}
	if a.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}
	if a.GetHost() == "" {
		return errors.New("config host is not defined")
	}
	return nil
}
