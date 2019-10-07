package config

import "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

type Monitoring struct {
	Config ProductConfig
}

func NewMonitoring(config ProductConfig) *Monitoring {
	return &Monitoring{Config: config}
}

func (r *Monitoring) GetNamespace() string {
	return r.Config["NAMESPACE"]
}

func (r *Monitoring) SetNamespace(newNamespace string) {
	r.Config["NAMESPACE"] = newNamespace
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
	return v1alpha1.ProductVersion(r.Config["VERSION"])
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
