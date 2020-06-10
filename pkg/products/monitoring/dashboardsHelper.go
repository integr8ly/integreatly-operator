package monitoring

import (
	"fmt"

	monitoring "github.com/integr8ly/integreatly-operator/pkg/products/monitoring/dashboards"
)

func getSpecDetailsForDashBoard(dbName string) (string, string, error) {

	switch dbName {

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

	case "critical-slo-alerts":
		return monitoring.MonitoringGrafanaDBCriticalSLOAlertsJSON, "critical-slo-alerts.json", nil

	default:
		return "", "", fmt.Errorf("Invalid/Unsupported Grafana Dashboard")

	}
}
