package config

const (
	OpenshiftMonitoringNamespace = "openshift-monitoring"

	OboLabelSelector    = "middleware"
	OboLabelSelectorKey = "monitoring-key"
	OboNamespaceSuffix  = "-observability"

	// Alertmanager configuration
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
