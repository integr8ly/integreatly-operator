package monitoringcommon

import (
	"context"
	grafanav1alpha1 "github.com/grafana-operator/grafana-operator/v4/api/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/metrics"
	monitoringcommon "github.com/integr8ly/integreatly-operator/pkg/products/monitoringcommon/dashboards"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/test/utils"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func Test_getSpecDetailsForDashboard(t *testing.T) {
	// Get containerCpuMetric which will be the same for all specs on the same cluster
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}
	version := &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Status: configv1.ClusterVersionStatus{
			History: []configv1.UpdateHistory{
				{
					State:          "",
					StartedTime:    metav1.Time{},
					CompletionTime: nil,
					Version:        "4.9.0-rc123",
					Image:          "",
					Verified:       false,
				},
			},
		},
	}
	client := fakeclient.NewFakeClientWithScheme(scheme, version)
	containerCpuMetric, err := metrics.GetContainerCPUMetric(context.TODO(), client, l.NewLoggerWithContext(l.Fields{}))
	if err != nil {
		t.Fatal(err)
	}

	// Create basic RHMI and get the installation type from it
	rhmi := basicInstallation()
	installationType := resources.InstallationNames[rhmi.Spec.Type]

	tests := []struct {
		testName  string
		dashboard string
		wantSpec  string
		wantName  string
		wantErr   string
	}{
		{
			testName:  "successfully get spec for endpointsdetailed dashboard",
			dashboard: "endpointsdetailed",
			wantSpec:  monitoringcommon.GetMonitoringGrafanaDBEndpointsDetailedJSON(rhmi.ObjectMeta.Name),
			wantName:  "endpointsdetailed.json",
			wantErr:   "",
		},
		{
			testName:  "successfully get spec for endpointsreport dashboard",
			dashboard: "endpointsreport",
			wantSpec:  monitoringcommon.GetMonitoringGrafanaDBEndpointsReportJSON(rhmi.ObjectMeta.Name),
			wantName:  "endpointsreport.json",
			wantErr:   "",
		},
		{
			testName:  "successfully get spec for endpointssummary dashboard",
			dashboard: "endpointssummary",
			wantSpec:  monitoringcommon.GetMonitoringGrafanaDBEndpointsSummaryJSON(rhmi.ObjectMeta.Name),
			wantName:  "endpointssummary.json",
			wantErr:   "",
		},
		{
			testName:  "successfully get spec for resources-by-namespace dashboard",
			dashboard: "resources-by-namespace",
			wantSpec:  monitoringcommon.GetMonitoringGrafanaDBResourceByNSJSON(rhmi.Spec.NamespacePrefix, rhmi.ObjectMeta.Name, containerCpuMetric),
			wantName:  "resources-by-namespace.json",
			wantErr:   "",
		},
		{
			testName:  "successfully get spec for resources-by-pod dashboard",
			dashboard: "resources-by-pod",
			wantSpec:  monitoringcommon.GetMonitoringGrafanaDBResourceByPodJSON(rhmi.Spec.NamespacePrefix, rhmi.ObjectMeta.Name, containerCpuMetric),
			wantName:  "resources-by-pod.json",
			wantErr:   "",
		},
		{
			testName:  "successfully get spec for cluster-resources dashboard",
			dashboard: "cluster-resources",
			wantSpec:  monitoringcommon.GetMonitoringGrafanaDBClusterResourcesJSON(rhmi.Spec.NamespacePrefix, rhmi.ObjectMeta.Name, containerCpuMetric),
			wantName:  "cluster-resources-new.json",
			wantErr:   "",
		},
		{
			testName:  "successfully get spec for critical-slo-rhmi-alerts dashboard",
			dashboard: "critical-slo-rhmi-alerts",
			wantSpec:  monitoringcommon.GetMonitoringGrafanaDBCriticalSLORHMIAlertsJSON(rhmi.Spec.NamespacePrefix, installationType),
			wantName:  "critical-slo-alerts.json",
			wantErr:   "",
		},
		{
			testName:  "successfully get spec for critical-slo-managed-api-alerts dashboard",
			dashboard: "critical-slo-managed-api-alerts",
			wantSpec:  monitoringcommon.GetMonitoringGrafanaDBCriticalSLOManagedAPIAlertsJSON(rhmi.Spec.NamespacePrefix, installationType),
			wantName:  "critical-slo-alerts.json",
			wantErr:   "",
		},
		{
			testName:  "successfully get spec for cro-resources dashboard",
			dashboard: "cro-resources",
			wantSpec:  monitoringcommon.MonitoringGrafanaDBCROResourcesJSON,
			wantName:  "cro-resources.json",
			wantErr:   "",
		},
		{
			testName:  "successfully get spec for rhoam-rhsso-availability-slo dashboard",
			dashboard: "rhoam-rhsso-availability-slo",
			wantSpec:  monitoringcommon.GetMonitoringGrafanaDBRhssoAvailabilityErrorBudgetBurnJSON(rhmi.ObjectMeta.Name),
			wantName:  "rhoam-rhsso-availability-slo.json",
			wantErr:   "",
		},
		{
			testName:  "successfully get spec for multitenancy-detailed dashboard",
			dashboard: "multitenancy-detailed",
			wantSpec:  monitoringcommon.MonitoringGrafanaDBMultitenancyDetailedJSON,
			wantName:  "multitenancy-detailed.json",
			wantErr:   "",
		},
		{
			testName:  "fail on empty dashboard name",
			dashboard: "",
			wantSpec:  "",
			wantName:  "",
			wantErr:   "Invalid/Unsupported Grafana Dashboard",
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			spec, name, err := GetSpecDetailsForDashboard(tt.dashboard, rhmi, containerCpuMetric)
			if tt.wantErr != "" && err.Error() != tt.wantErr {
				t.Errorf("GetSpecDetailsForDashboard() error = %v, wantErr %v", err.Error(), tt.wantErr)
				return
			}
			if spec != tt.wantSpec {
				t.Errorf("GetSpecDetailsForDashboard() got %v for the spec but wanted %v", spec, tt.wantSpec)
				return
			}
			if name != tt.wantName {
				t.Errorf("GetSpecDetailsForDashboard() got %v for the name but wanted %v", name, tt.wantName)
				return
			}
		})
	}
}

func Test_getPluginsForGrafanaDashboard(t *testing.T) {
	tests := []struct {
		testName  string
		dashboard string
		want      grafanav1alpha1.PluginList
	}{
		{
			testName:  "get empty PluginList when dashboard name isn't set",
			dashboard: "",
			want:      grafanav1alpha1.PluginList{},
		},
		{
			testName:  "successfully get correct PluginList when dashboard name is endpointsdetailed",
			dashboard: "endpointsdetailed",
			want: grafanav1alpha1.PluginList{
				grafanav1alpha1.GrafanaPlugin{
					Name:    "natel-discrete-panel",
					Version: "0.0.9",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			pluginList := GetPluginsForGrafanaDashboard(tt.dashboard)

			if len(pluginList) != len(tt.want) {
				t.Errorf("GetPluginsForGrafanaDashboard() got %v but wanted %v", pluginList, tt.want)
				return
			}
			if len(pluginList) > 0 && len(tt.want) > 0 {
				if pluginList[0] != tt.want[0] {
					t.Errorf("GetPluginsForGrafanaDashboard() got %v but wanted %v", pluginList, tt.want)
					return
				}
			}
		})
	}
}
