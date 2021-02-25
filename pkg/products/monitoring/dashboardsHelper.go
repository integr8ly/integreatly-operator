package monitoring

import (
	"fmt"

	grafanav1alpha1 "github.com/integr8ly/grafana-operator/v3/pkg/apis/integreatly/v1alpha1"
	monitoring "github.com/integr8ly/integreatly-operator/pkg/products/monitoring/dashboards"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
)

func getSpecDetailsForDashboard(dashboard, nsPrefix string, installType string) (string, string, error) {
	product := resources.InstallationNames[installType]

	switch dashboard {

	case "endpointsdetailed":
		return monitoring.MonitoringGrafanaDBEndpointsDetailedJSON, "endpointsdetailed.json", nil

	case "endpointsreport":
		return monitoring.MonitoringGrafanaDBEndpointsReportJSON, "endpointsreport.json", nil

	case "endpointssummary":
		return monitoring.MonitoringGrafanaDBEndpointsSummaryJSON, "endpointssummary.json", nil

	case "resources-by-namespace":
		return monitoring.MonitoringGrafanaDBResourceByNSJSON, "resources-by-namespace.json", nil

	case "resources-by-pod":
		return monitoring.MonitoringGrafanaDBResourceByPodJSON, "resources-by-pod.json", nil

	case "cluster-resources":
		return monitoring.MonitoringGrafanaDBClusterResourcesJSON, "cluster-resources-new.json", nil

	case "critical-slo-rhmi-alerts":
		return monitoring.GetMonitoringGrafanaDBCriticalSLORHMIAlertsJSON(nsPrefix, product), "critical-slo-alerts.json", nil

	case "critical-slo-managed-api-alerts":
		return monitoring.GetMonitoringGrafanaDBCriticalSLOManagedAPIAlertsJSON(nsPrefix, product), "critical-slo-alerts.json", nil

	case "cro-resources":
		return monitoring.MonitoringGrafanaDBCROResourcesJSON, "cro-resources.json", nil

	default:
		return "", "", fmt.Errorf("Invalid/Unsupported Grafana Dashboard")

	}
}

func getPluginsForGrafanaDashboard(name string) grafanav1alpha1.PluginList {
	var pluginsList grafanav1alpha1.PluginList
	if name == "endpointsdetailed" {
		plugin := grafanav1alpha1.GrafanaPlugin{
			Name:    "natel-discrete-panel",
			Version: "0.0.9",
		}
		pluginsList = append(pluginsList, plugin)
	}
	return pluginsList
}
