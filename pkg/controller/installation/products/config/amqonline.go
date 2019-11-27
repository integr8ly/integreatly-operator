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

func (r *AMQOnline) GetLabelSelector() string {
	return "middleware"
}

func (r *AMQOnline) GetTemplateList() []string {
	template_list := []string{
		"kube_state_metrics_amqonline_alerts.yaml",
	}
	return template_list
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

func (a *AMQOnline) GetProductVersion() v1alpha1.ProductVersion {
	return v1alpha1.VersionAMQOnline
}

func (a *AMQOnline) GetOperatorVersion() v1alpha1.OperatorVersion {
	return v1alpha1.OperatorVersionAMQOnline
}

func (c *AMQOnline) GetBackendSecretName() string {
	return "s3-credentials"
}

func (c *AMQOnline) GetBackupSchedule() string {
	return "30 2 * * *"
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
