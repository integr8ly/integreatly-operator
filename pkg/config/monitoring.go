package config

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

type Monitoring struct {
	Config ProductConfig
}

func NewMonitoring(config ProductConfig) *Monitoring {
	return &Monitoring{Config: config}
}
func (m *Monitoring) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{
		// FIXME INTLY-5018 - uncomment ApplicationMonitoring
		// &monitoring_v1alpha1.ApplicationMonitoring{
		// 	TypeMeta: v1.TypeMeta{
		// 		Kind:       monitoring_v1alpha1.ApplicationMonitoringKind,
		// 		APIVersion: monitoring_v1alpha1.SchemeGroupVersion.String(),
		// 	},
		// },
	}
}

func (m *Monitoring) GetExtraParam(key string) string {
	return m.Config[key]
}

func (m *Monitoring) SetExtraParam(key string, val string) {
	m.Config[key] = val
}

func (m *Monitoring) GetNamespace() string {
	return m.Config["NAMESPACE"]
}

func (m *Monitoring) SetNamespace(newNamespace string) {
	m.Config["NAMESPACE"] = newNamespace
}

func (m *Monitoring) GetFederationNamespace() string {
	return m.Config["FEDERATION_NAMESPACE"]
}

func (m *Monitoring) SetFederationNamespace(newNamespace string) {
	m.Config["FEDERATION_NAMESPACE"] = newNamespace
}

func (m *Monitoring) GetOperatorNamespace() string {
	return m.Config["OPERATOR_NAMESPACE"]
}

func (m *Monitoring) SetOperatorNamespace(newNamespace string) {
	m.Config["OPERATOR_NAMESPACE"] = newNamespace
}

func (m *Monitoring) GetNamespacePrefix() string {
	return m.Config["NAMESPACE_PREFIX"]
}

func (m *Monitoring) SetNamespacePrefix(newNamespacePrefix string) {
	m.Config["NAMESPACE_PREFIX"] = newNamespacePrefix
}

func (m *Monitoring) GetMonitoringConfigurationNamespace() string {
	return m.Config["NAMESPACE_PREFIX"] + m.Config["NAMESPACE"] + "-config"
}

func (m *Monitoring) GetHost() string {
	return m.Config["HOST"]
}

func (m *Monitoring) SetHost(newHost string) {
	m.Config["HOST"] = newHost
}

func (m *Monitoring) Read() ProductConfig {
	return m.Config
}

func (m *Monitoring) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductMonitoring
}

func (m *Monitoring) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.VersionMonitoring
}

func (m *Monitoring) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionMonitoring
}

func (m *Monitoring) SetProductVersion(version string) {
	m.Config["VERSION"] = version
}

func (m *Monitoring) GetLabelSelector() string {
	return "middleware"
}

func (m *Monitoring) GetLabelSelectorKey() string {
	return "monitoring-key"
}

func (m *Monitoring) GetAdditionalScrapeConfigSecretName() string {
	return "rhmi-additional-scrape-configs"
}

func (m *Monitoring) GetAdditionalScrapeConfigSecretKey() string {
	return "integreatly-additional.yaml"
}

func (m *Monitoring) GetPrometheusRetention() string {
	return "45d"
}

func (m *Monitoring) GetPrometheusStorageRequest() string {
	return "50Gi"
}

func (m *Monitoring) GetTemplateList() []string {
	templateList := []string{
		"kube_state_metrics_monitoring_alerts.yaml",
		"endpointsdetailed.yaml",
		"endpointsreport.yaml",
		"endpointssummary.yaml",
		"resources-by-namespace.yaml",
		"resources-by-pod.yaml",
		"cluster-resources.yaml",
		"critical_slo_alerts.yaml",
	}
	return templateList
}

func (m *Monitoring) GetJobTemplates() []string {
	return []string{
		"jobs/3scale.yaml",
		"jobs/openshift_monitoring_federation.yaml",
	}
}

func (m *Monitoring) Validate() error {
	if m.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}

	if m.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}

	if m.GetProductVersion() == "" {
		return errors.New("config product version is not defined")
	}

	return nil
}
