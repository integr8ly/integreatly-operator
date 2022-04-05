package monitoringcommon

import (
	"fmt"
	v1alpha12 "github.com/integr8ly/grafana-operator/v3/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	monitoringcommon "github.com/integr8ly/integreatly-operator/pkg/products/monitoringcommon/dashboards"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
)

func GetSpecDetailsForDashboard(dashboard string, rhmi *v1alpha1.RHMI, containerCpuMetric string) (string, string, error) {
	installationName := resources.InstallationNames[rhmi.Spec.Type]

	switch dashboard {

	case "endpointsdetailed":
		return monitoringcommon.GetMonitoringGrafanaDBEndpointsDetailedJSON(rhmi.ObjectMeta.Name), "endpointsdetailed.json", nil

	case "endpointsreport":
		return monitoringcommon.GetMonitoringGrafanaDBEndpointsReportJSON(rhmi.ObjectMeta.Name), "endpointsreport.json", nil

	case "endpointssummary":
		return monitoringcommon.GetMonitoringGrafanaDBEndpointsSummaryJSON(rhmi.ObjectMeta.Name), "endpointssummary.json", nil

	case "resources-by-namespace":
		return monitoringcommon.GetMonitoringGrafanaDBResourceByNSJSON(rhmi.Spec.NamespacePrefix, rhmi.ObjectMeta.Name, containerCpuMetric), "resources-by-namespace.json", nil

	case "resources-by-pod":
		return monitoringcommon.GetMonitoringGrafanaDBResourceByPodJSON(rhmi.Spec.NamespacePrefix, rhmi.ObjectMeta.Name, containerCpuMetric), "resources-by-pod.json", nil

	case "cluster-resources":
		return monitoringcommon.GetMonitoringGrafanaDBClusterResourcesJSON(rhmi.Spec.NamespacePrefix, rhmi.ObjectMeta.Name, containerCpuMetric), "cluster-resources-new.json", nil
	case "critical-slo-rhmi-alerts":
		return monitoringcommon.GetMonitoringGrafanaDBCriticalSLORHMIAlertsJSON(rhmi.Spec.NamespacePrefix, installationName), "critical-slo-alerts.json", nil

	case "critical-slo-managed-api-alerts":
		return monitoringcommon.GetMonitoringGrafanaDBCriticalSLOManagedAPIAlertsJSON(rhmi.Spec.NamespacePrefix, installationName), "critical-slo-alerts.json", nil

	case "cro-resources":
		return monitoringcommon.MonitoringGrafanaDBCROResourcesJSON, "cro-resources.json", nil

	case "rhoam-rhsso-availability-slo":
		return monitoringcommon.GetMonitoringGrafanaDBRhssoAvailabilityErrorBudgetBurnJSON(rhmi.ObjectMeta.Name), "rhoam-rhsso-availability-slo.json", nil

	case "rhoam-fleet-wide-view":
		return monitoringcommon.ObservatoriumFleetWideJSON, "rhoam-fleet-wide-view.json", nil

	default:
		return "", "", fmt.Errorf("Invalid/Unsupported Grafana Dashboard")

	}
}

func GetPluginsForGrafanaDashboard(name string) v1alpha12.PluginList {
	var pluginsList v1alpha12.PluginList
	if name == "endpointsdetailed" {
		plugin := v1alpha12.GrafanaPlugin{
			Name:    "natel-discrete-panel",
			Version: "0.0.9",
		}
		pluginsList = append(pluginsList, plugin)
	}
	return pluginsList
}
