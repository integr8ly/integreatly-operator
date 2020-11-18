package common

import (
	goctx "context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
)

type dashboardsTestRule struct {
	Title string `json:"title"`
}

// Common to all install types including managed api
var commonExpectedDashboards = []dashboardsTestRule{
	{
		Title: "Endpoints Detailed",
	},
	{
		Title: "Endpoints Report",
	},
	{
		Title: "Endpoints Summary",
	},
	{
		Title: "Keycloak",
	},
	{
		Title: "Resource Usage By Namespace",
	},
	{
		Title: "Resource Usage By Pod",
	},
	{
		Title: "Resource Usage for Cluster",
	},
	{
		Title: "Critical SLO summary",
	},
}

// Applicable to install types used in 2.X
var rhmi2ExpectedDashboards = []dashboardsTestRule{
	{
		Title: "Syndesis - Infra - API",
	},
	{
		Title: "Syndesis - Infra - Home",
	},
	{
		Title: "Syndesis - Infra - JVM",
	},
	{
		Title: "Syndesis - Integrations - Camel",
	},
	{
		Title: "Syndesis - Integrations - Home",
	},
	{
		Title: "Syndesis - Integrations - JVM",
	},
	{
		Title: "UnifiedPush Operator",
	},
	{
		Title: "UnifiedPush Server",
	},
	{
		Title: "AMQ Online",
	},
	{
		Title: "EnMasse Brokers",
	},
	{
		Title: "EnMasse Console",
	},
	{
		Title: "EnMasse Routers",
	},
}

var customerRHOAMDashboards = []dashboardsTestRule{
	{
		Title: "Rate Limiting",
	},
}

func TestIntegreatlyCustomerDashboardsExist(t *testing.T, ctx *TestingContext) {
	t.Log("skipping customer dashboards exist test due to flakiness")
	t.SkipNow()
	// get console master url
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	monitoringGrafanaPods := getGrafanaPods(t, ctx, MonitoringOperatorNamespace)
	customerMonitoringGrafanaPods := getGrafanaPods(t, ctx, CustomerGrafanaNamespace)

	output, err := execToPod(fmt.Sprintf("curl %v:3000/api/search", customerMonitoringGrafanaPods.Items[0].Status.PodIP),
		monitoringGrafanaPods.Items[0].ObjectMeta.Name,
		MonitoringOperatorNamespace,
		"grafana", ctx)
	if err != nil {
		t.Fatal("failed to exec to pod:", err, "pod name:", customerMonitoringGrafanaPods.Items[0].Name)
	}

	var grafanaApiCallOutput []dashboardsTestRule
	err = json.Unmarshal([]byte(output), &grafanaApiCallOutput)
	if err != nil {
		t.Logf("failed to unmarshall json: %s", err)
	}

	if len(grafanaApiCallOutput) == 0 {
		t.Fatal("no grafana dashboards were found : %w", grafanaApiCallOutput)
	}

	expectedDashboards := getExpectedCustomerDashboard(rhmi.Spec.Type)
	verifyExpectedDashboards(t, expectedDashboards, removeNamespaceDashboardFolder(grafanaApiCallOutput))
}

func TestIntegreatlyMiddelewareDashboardsExist(t *testing.T, ctx *TestingContext) {
	// get console master url
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	monitoringGrafanaPods := getGrafanaPods(t, ctx, MonitoringOperatorNamespace)

	output, err := execToPod("curl localhost:3000/api/search",
		monitoringGrafanaPods.Items[0].ObjectMeta.Name,
		MonitoringOperatorNamespace,
		"grafana", ctx)
	if err != nil {
		t.Fatal("failed to exec to pod:", err)
	}

	var grafanaApiCallOutput []dashboardsTestRule
	err = json.Unmarshal([]byte(output), &grafanaApiCallOutput)
	if err != nil {
		t.Logf("failed to unmarshall json: %s", err)
	}

	if len(grafanaApiCallOutput) == 0 {
		t.Fatal("no grafana dashboards were found : %w", grafanaApiCallOutput)
	}

	expectedDashboards := getExpectedMiddlewareDashboard(rhmi.Spec.Type)
	verifyExpectedDashboards(t, expectedDashboards, grafanaApiCallOutput)

}

func verifyExpectedDashboards(t *testing.T, expectedDashboards []dashboardsTestRule, grafanaApiCallOutput []dashboardsTestRule) {
	var expectedDashboardTitles []string
	for _, dashboard := range expectedDashboards {
		expectedDashboardTitles = append(expectedDashboardTitles, dashboard.Title)
	}
	var actualDashboardTitles []string
	for _, dashboard := range grafanaApiCallOutput {
		actualDashboardTitles = append(actualDashboardTitles, dashboard.Title)
	}

	dashboardDiffUnexpected := difference(actualDashboardTitles, expectedDashboardTitles)
	dashboardDiffMissing := difference(expectedDashboardTitles, actualDashboardTitles)

	if len(dashboardDiffUnexpected) > 0 {
		t.Logf("unexpected dashboards found: %s", strings.Join(dashboardDiffUnexpected, ", "))
	}

	if len(dashboardDiffMissing) > 0 {
		t.Logf("missing dashboards found: %s", strings.Join(dashboardDiffMissing, ", "))
	}

	if len(dashboardDiffUnexpected) > 0 || len(dashboardDiffMissing) > 0 {
		t.Fatal("missing or too many dashboards found")
	}
}

func getExpectedCustomerDashboard(installType string) []dashboardsTestRule {
	if installType == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		return customerRHOAMDashboards
	}
	return nil
}

func getExpectedMiddlewareDashboard(installType string) []dashboardsTestRule {
	if installType == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		return commonExpectedDashboards
	} else {
		return append(commonExpectedDashboards, rhmi2ExpectedDashboards...)
	}
}

func getGrafanaPods(t *testing.T, ctx *TestingContext, ns string) corev1.PodList {
	pods := &corev1.PodList{}
	opts := []k8sclient.ListOption{
		k8sclient.InNamespace(ns),
		k8sclient.MatchingLabels{"app": "grafana"},
	}

	err := ctx.Client.List(goctx.TODO(), pods, opts...)
	if err != nil {
		t.Fatalf("failed to list pods in ns %v", ns)
	}

	if len(pods.Items) != 1 {
		t.Fatalf("grafana pod not found in ns %v", ns)
	}

	return *pods
}

func removeNamespaceDashboardFolder(grafanaApiOutput []dashboardsTestRule) []dashboardsTestRule {
	var actualGrafanaDashboards []dashboardsTestRule

	for _, dashboard := range grafanaApiOutput {
		if dashboard.Title != CustomerGrafanaNamespace {
			actualGrafanaDashboards = append(actualGrafanaDashboards, dashboard)
		}
	}

	return actualGrafanaDashboards
}
