package config

import (
	"errors"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

type Monitoring struct {
	Config ProductConfig
}

func NewMonitoring(config ProductConfig) *Monitoring {
	return &Monitoring{Config: config}
}

func (r *Monitoring) GetExtraParam(key string) string {
	return r.Config[key]
}

func (r *Monitoring) SetExtraParam(key string, val string) {
	r.Config[key] = val
}

func (r *Monitoring) GetNamespace() string {
	return r.Config["NAMESPACE"]
}

func (r *Monitoring) SetNamespace(newNamespace string) {
	r.Config["NAMESPACE"] = newNamespace
}

func (r *Monitoring) GetNamespacePrefix() string {
	return r.Config["NAMESPACE_PREFIX"]
}

func (r *Monitoring) SetNamespacePrefix(newNamespacePrefix string) {
	r.Config["NAMESPACE_PREFIX"] = newNamespacePrefix
}

func (r *Monitoring) GetMonitoringConfigurationNamespace() string {
	return r.Config["NAMESPACE_PREFIX"] + r.Config["NAMESPACE"] + "-config"
}

func (r *Monitoring) GetHost() string {
	return r.Config["HOST"]
}

func (r *Monitoring) SetHost(newHost string) {
	r.Config["HOST"] = newHost
}

func (r *Monitoring) Read() ProductConfig {
	return r.Config
}

func (r *Monitoring) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductMonitoring
}

func (r *Monitoring) GetProductVersion() v1alpha1.ProductVersion {
	return v1alpha1.VersionMonitoring
}

func (r *Monitoring) GetOperatorVersion() v1alpha1.OperatorVersion {
	return v1alpha1.OperatorVersionMonitoring
}

func (r *Monitoring) SetProductVersion(version string) {
	r.Config["VERSION"] = version
}

func (r *Monitoring) GetLabelSelector() string {
	return "middleware"
}

func (r *Monitoring) GetAdditionalScrapeConfigSecretName() string {
	return "integreatly-additional-scrape-configs"
}

func (r *Monitoring) GetAdditionalScrapeConfigSecretKey() string {
	return "integreatly-additional.yaml"
}

func (r *Monitoring) GetPrometheusRetention() string {
	return "15d"
}

func (r *Monitoring) GetPrometheusStorageRequest() string {
	return "10Gi"
}

func (r *Monitoring) GetTemplateList() []string {
	template_list := []string{
		"kube_state_metrics_alerts.yaml",
		"kube_state_metrics_monitoring_alerts.yaml",
		"endpointsdetailed.yaml",
		"endpointsreport.yaml",
		"endpointssummary.yaml",
		"resources-by-namespace.yaml",
		"resources-by-pod.yaml",
		"cluster-resources.yaml",
	}
	return template_list
}

func (r *Monitoring) GetJobTemplates() []string {
	return []string{
		"jobs/3scale.yaml",
		"jobs/openshift_monitoring_federation.yaml",
	}
}

func (f *Monitoring) Validate() error {
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
