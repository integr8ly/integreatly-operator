package config

import "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

type AMQStreams struct {
	config v1alpha1.ProductConfig
}

func newAMQStreams(config v1alpha1.ProductConfig) *AMQStreams {
	if config == nil {
		config = v1alpha1.ProductConfig{}
	}
	return &AMQStreams{config: config}
}

func (a *AMQStreams) GetHost() string {
	return a.config["HOST"]
}

func (a *AMQStreams) SetHost(newHost string) {
	a.config["HOST"] = newHost
}

func (a *AMQStreams) GetNamespace() string {
	return a.config["NAMESPACE"]
}

func (a *AMQStreams) SetNamespace(newNamespace string) {
	a.config["NAMESPACE"] = newNamespace
}

func (a *AMQStreams) Read() v1alpha1.ProductConfig {
	return a.config
}

func (a *AMQStreams) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductAMQStreams
}
