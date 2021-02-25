package config

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
)

const (
	// Configuration key to set the scrape interval for the federate metrics
	MonitoringParamFederateScrapeInterval = "FEDERATE_SCRAPE_INTERVAL"
	// Configuration key to set the scrape timeout for the federate metrics
	MonitoringParamFederateScrapeTimeout = "FEDERATE_SCRAPE_TIMEOUT"
	// The default value for the scrape interval for the federate metrics. Default is 60s.
	MonitoringDefaultFederateScrapeInterval = "60s"
	// The default value for the scrape interval for the federate timeout. Default is 30s.
	MonitoringDefaultFederateScrapeTimeout = "30s"
)

type Monitoring struct {
	Config ProductConfig
}

var rhmiTemplateList = []string{
	"endpointsdetailed",
	"endpointsreport",
	"endpointssummary",
	"resources-by-namespace",
	"resources-by-pod",
	"cluster-resources",
	"critical-slo-rhmi-alerts",
	"cro-resources",
}

var managedAPITemplateList = []string{
	"endpointsdetailed",
	"endpointsreport",
	"endpointssummary",
	"resources-by-namespace",
	"resources-by-pod",
	"cluster-resources",
	"critical-slo-managed-api-alerts",
	"cro-resources",
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

func (m *Monitoring) GetDashboards(installType integreatlyv1alpha1.InstallationType) []string {
	switch installType {
	case integreatlyv1alpha1.InstallationTypeManaged, integreatlyv1alpha1.InstallationTypeSelfManaged, integreatlyv1alpha1.InstallationTypeWorkshop:
		return rhmiTemplateList
	case integreatlyv1alpha1.InstallationTypeManagedApi:
		return managedAPITemplateList
	default:
		return rhmiTemplateList
	}
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

// Try to get the value for the given key from the configuration if it exists. Otherwise return the default value instead.
func (m *Monitoring) GetExtraParamWithDefault(key string, v string) string {
	if val, ok := m.Config[key]; ok {
		return val
	}
	return v
}
