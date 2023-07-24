package config

import (
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	oo "github.com/redhat-developer/observability-operator/v4/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	OboLabelSelector    = "middleware"
	OboLabelSelectorKey = "monitoring-key"
	OboNamespaceSuffix  = "-observability"

	// Alert manager configuration
	AlertManagerConfigSecretName            = "alertmanager-rhoam"
	AlertManagerConfigSecretFileName        = "alertmanager.yaml"
	AlertManagerEmailTemplateSecretFileName = "alertmanager-email-config.tmpl"
	AlertManagerConfigTemplatePath          = "alertmanager/alertmanager-application-monitoring.yaml"
	AlertManagerCustomTemplatePath          = "alertmanager/alertmanager-email-config.tmpl"

	// CR Overrides
	AlertManagerOverride = "alertmanager"
	GrafanaOverride      = "grafana"
	PrometheusOverride   = "prometheus"
)

func GetOboNamespace(installationNamespace string) string {
	return installationNamespace + OboNamespaceSuffix
}

func GetOboLabelSelector() string {
	return OboLabelSelector
}

func GetOboLabelSelectorKey() string {
	return OboLabelSelectorKey
}

// TODO MGDAPI-5833: everything *Observability related to be removed

var rhmiTemplateList = []string{
	"endpointsdetailed",
	"endpointsreport",
	"endpointssummary",
	"resources-by-namespace",
	"resources-by-pod",
	"cluster-resources",
	"critical-slo-rhmi-alerts",
	"cro-resources",
	"rhoam-rhsso-availability-slo",
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
	"rhoam-rhsso-availability-slo",
}

var multitenantManagedAPITemplateList = []string{
	"endpointsdetailed",
	"endpointsreport",
	"endpointssummary",
	"resources-by-namespace",
	"resources-by-pod",
	"cluster-resources",
	"critical-slo-managed-api-alerts",
	"cro-resources",
	"rhoam-rhsso-availability-slo",
	"multitenancy-detailed",
}

type Observability struct {
	Config ProductConfig
}

func NewObservability(config ProductConfig) *Observability {
	return &Observability{Config: config}
}

func (m *Observability) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductObservability
}

func (m *Observability) GetOperatorNamespace() string {
	return m.Config["OPERATOR_NAMESPACE"]
}

func (m *Observability) SetOperatorNamespace(newNamespace string) {
	m.Config["OPERATOR_NAMESPACE"] = newNamespace
}

func (m *Observability) GetNamespace() string {
	return m.Config["NAMESPACE"]
}

func (m *Observability) SetNamespace(newNamespace string) {
	m.Config["NAMESPACE"] = newNamespace
}

func (m *Observability) GetNamespacePrefix() string {
	return m.Config["NAMESPACE_PREFIX"]
}

func (m *Observability) SetNamespacePrefix(newNamespacePrefix string) {
	m.Config["NAMESPACE_PREFIX"] = newNamespacePrefix
}

func (m *Observability) Read() ProductConfig {
	return m.Config
}

func (m *Observability) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.VersionObservability
}

func (m *Observability) GetLabelSelector() string {
	return "middleware"
}

func (m *Observability) GetLabelSelectorKey() string {
	return "monitoring-key"
}

func (m *Observability) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionObservability
}

func (m *Observability) SetProductVersion(newVersion string) {
	m.Config["VERSION"] = newVersion
}

func (m *Observability) GetHost() string {
	return m.Config["HOST"]
}

func (m *Observability) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{
		&oo.Observability{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Observability",
				APIVersion: oo.GroupVersion.String(),
			},
		},
	}
}

func (m *Observability) GetDashboards(installType integreatlyv1alpha1.InstallationType) []string {
	switch installType {
	case integreatlyv1alpha1.InstallationTypeManagedApi:
		return managedAPITemplateList
	case integreatlyv1alpha1.InstallationTypeMultitenantManagedApi:
		return multitenantManagedAPITemplateList
	default:
		return rhmiTemplateList
	}
}

func (m *Observability) GetAlertManagerVersion() string {
	return "v0.22.2"
}

func (m *Observability) GetAlertManagerRouteName() string {
	return AlertManagerOverride
}

func (m *Observability) GetAlertManagerOverride() string {
	return AlertManagerOverride
}

func (m *Observability) GetAlertManagerServiceName() string {
	return AlertManagerOverride
}

func (m *Observability) GetAlertManagerStorageRequest() string {
	return "1Gi"
}

func (m *Observability) GetPrometheusVersion() string {
	return "v2.29.2"
}

func (m *Observability) GetPrometheusRouteName() string {
	return PrometheusOverride
}

func (m *Observability) GetPrometheusOverride() string {
	return PrometheusOverride
}

func (m *Observability) GetPrometheusServiceName() string {
	return PrometheusOverride
}

func (m *Observability) GetPrometheusRetention() string {
	return "45d"
}

func (m *Observability) GetPrometheusStorageRequest() string {
	return "50Gi"
}

func (m *Observability) GetGrafanaRouteName() string {
	return "grafana-route"
}

func (m *Observability) GetGrafanaOverride() string {
	return GrafanaOverride
}

func (m *Observability) GetGrafanaServiceName() string {
	return "grafana-service"
}

func (m *Observability) GetAlertManagerResourceRequirements() *corev1.ResourceRequirements {
	return &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("200Mi")},
	}
}

func (m *Observability) GetGrafanaResourceRequirements() *corev1.ResourceRequirements {
	return &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("256Mi")},
		Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m"), corev1.ResourceMemory: resource.MustParse("1Gi")},
	}
}

func (m *Observability) GetPrometheusResourceRequirements() *corev1.ResourceRequirements {
	return &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("400Mi")},
	}
}

func (m *Observability) GetPrometheusOperatorResourceRequirements() *corev1.ResourceRequirements {
	return &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("200Mi")},
		Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("200m"), corev1.ResourceMemory: resource.MustParse("400Mi")},
	}
}
