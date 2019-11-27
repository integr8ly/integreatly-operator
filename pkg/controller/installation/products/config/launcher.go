package config

import "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

type Launcher struct {
	config ProductConfig
}

func NewLauncher(config ProductConfig) *Launcher {
	return &Launcher{config: config}
}

func (a *Launcher) GetHost() string {
	return a.config["HOST"]
}

func (r *Launcher) GetLabelSelector() string {
	return "middleware"
}

func (r *Launcher) GetTemplateList() []string {
	template_list := []string{
		"kube_state_metrics_launcher_alerts.yaml",
	}
	return template_list
}

func (a *Launcher) SetHost(newHost string) {
	a.config["HOST"] = newHost
}

func (a *Launcher) GetNamespace() string {
	return a.config["NAMESPACE"]
}

func (a *Launcher) SetNamespace(newNamespace string) {
	a.config["NAMESPACE"] = newNamespace
}

func (a *Launcher) Read() ProductConfig {
	return a.config
}

func (a *Launcher) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductLauncher
}

func (c *Launcher) GetProductVersion() v1alpha1.ProductVersion {
	return v1alpha1.VersionLauncher
}

func (c *Launcher) GetOperatorVersion() v1alpha1.OperatorVersion {
	return v1alpha1.OperatorVersionLauncher
}
