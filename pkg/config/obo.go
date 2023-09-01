package config

import (
	"context"
	obov1 "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OpenshiftMonitoringNamespace = "openshift-monitoring"

	OboLabelSelector       = "middleware"
	OboLabelSelectorKey    = "monitoring-key"
	OboNamespaceSuffix     = "-observability"
	OboMonitoringStackName = "rhoam"

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

func GetOboMonitoringStack(client k8sclient.Client, OboNamespace string) (*obov1.MonitoringStack, error) {
	monitoringStack := &obov1.MonitoringStack{}

	err := client.Get(context.TODO(), k8sclient.ObjectKey{Name: OboMonitoringStackName, Namespace: OboNamespace}, monitoringStack)
	if err != nil {
		return nil, err
	}

	return monitoringStack, nil
}
