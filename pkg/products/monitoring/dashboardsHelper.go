package monitoring

import (
	"fmt"

	grafanav1alpha1 "github.com/integr8ly/grafana-operator/v3/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	monitoring "github.com/integr8ly/integreatly-operator/pkg/products/monitoring/dashboards"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
)

func getSpecDetailsForDashboard(dashboard string, rhmi *v1alpha1.RHMI) (string, string, error) {
	installationName := resources.InstallationNames[rhmi.Spec.Type]

	switch dashboard {

	case "endpointsdetailed":
		return monitoring.GetMonitoringGrafanaDBEndpointsDetailedJSON(rhmi.ObjectMeta.Name), "endpointsdetailed.json", nil

	case "endpointsreport":
		return monitoring.GetMonitoringGrafanaDBEndpointsReportJSON(rhmi.ObjectMeta.Name), "endpointsreport.json", nil

	case "endpointssummary":
		return monitoring.GetMonitoringGrafanaDBEndpointsSummaryJSON(rhmi.ObjectMeta.Name), "endpointssummary.json", nil

	case "resources-by-namespace":
		return monitoring.GetMonitoringGrafanaDBResourceByNSJSON(rhmi.ObjectMeta.Name), "resources-by-namespace.json", nil

	case "resources-by-pod":
		return monitoring.GetMonitoringGrafanaDBResourceByPodJSON(rhmi.ObjectMeta.Name), "resources-by-pod.json", nil

	case "cluster-resources":
		return monitoring.GetMonitoringGrafanaDBClusterResourcesJSON(rhmi.ObjectMeta.Name), "cluster-resources-new.json", nil
	case "critical-slo-rhmi-alerts":
		return monitoring.GetMonitoringGrafanaDBCriticalSLORHMIAlertsJSON(rhmi.Spec.NamespacePrefix, installationName), "critical-slo-alerts.json", nil

	case "critical-slo-managed-api-alerts":
		return monitoring.GetMonitoringGrafanaDBCriticalSLOManagedAPIAlertsJSON(rhmi.Spec.NamespacePrefix, installationName), "critical-slo-alerts.json", nil

	case "cro-resources":
		return monitoring.MonitoringGrafanaDBCROResourcesJSON, "cro-resources.json", nil

	case "rhoam-rhsso-availability-slo":
		return monitoring.GetMonitoringGrafanaDBRhssoAvailabilityErrorBudgetBurnJSON(rhmi.ObjectMeta.Name), "rhoam-rhsso-availability-slo.json", nil

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
