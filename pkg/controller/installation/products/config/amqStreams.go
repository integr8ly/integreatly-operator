package config

import integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

type AMQStreams struct {
	config ProductConfig
}

func NewAMQStreams(config ProductConfig) *AMQStreams {
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

func (a *AMQStreams) Read() ProductConfig {
	return a.config
}

func (a *AMQStreams) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductAMQStreams
}

func (a *AMQStreams) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.VersionAMQStreams
}

func (a *AMQStreams) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionAMQStreams
}
