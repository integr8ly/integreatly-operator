package common

import (
	goctx "context"
	"encoding/json"
	"fmt"
	"strings"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
)

type dashboardsTestRule struct {
	Type  string `json:"type"`
	Title string `json:"title"`
}

// Common to all install types including managed api

var customerRHOAMDashboards = []dashboardsTestRule{
	{
		Title: "Rate Limiting",
	},
}

func TestIntegreatlyCustomerDashboardsExist(t TestingTB, ctx *TestingContext) {
	// get console master url
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	// Pod and container to perform curls from
	prometheusPodName, err := getMonitoringAppPodName("prometheus", ctx)
	if err != nil {
		t.Fatal("failed to get prometheus pod name", err)
	}
	curlContainerName := "prometheus"

	customerMonitoringGrafanaPods := getGrafanaPods(t, ctx, CustomerGrafanaNamespace)

	output, err := execToPod(fmt.Sprintf("wget -qO - %v:3000/api/search", customerMonitoringGrafanaPods.Items[0].Status.PodIP),
		prometheusPodName,
		ObservabilityProductNamespace,
		curlContainerName, ctx)
	if err != nil {
		t.Fatal("failed to exec to pod:", err, "pod name:", prometheusPodName, "container name:", curlContainerName, "namespace:", ObservabilityProductNamespace)
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
	err = verifyExpectedDashboards(expectedDashboards, removeNamespaceDashboardFolder(grafanaApiCallOutput))
	if err != nil {
		t.Fatalf("Verify Expected Dashboards failed: ", err)
	}
}

func verifyExpectedDashboards(expectedDashboards []dashboardsTestRule, grafanaApiCallOutput []dashboardsTestRule) error {
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
		// The workload dashboard is added for upgrade pipelines and needs to be allowed for.
		if allowOnlyWorkLoadDashboard(dashboardDiffUnexpected) {
			return nil
		}
		return fmt.Errorf("unexpected dashboards found: %s", strings.Join(dashboardDiffUnexpected, ", "))
	}

	if len(dashboardDiffMissing) > 0 {
		return fmt.Errorf("missing dashboards found: %s", strings.Join(dashboardDiffMissing, ", "))
	}

	return nil
}

func allowOnlyWorkLoadDashboard(unexpected []string) bool {
	workLoadDashboard := "Workload App"
	if len(unexpected) != 1 {
		return false
	}
	for _, dashboard := range unexpected {
		if dashboard == workLoadDashboard {
			return true
		}
	}
	return false
}

func getExpectedCustomerDashboard(installType string) []dashboardsTestRule {
	return customerRHOAMDashboards
}

func getGrafanaPods(t TestingTB, ctx *TestingContext, ns string) corev1.PodList {
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
		if dashboard.Type != "dash-folder" {
			actualGrafanaDashboards = append(actualGrafanaDashboards, dashboard)
		}
	}

	return actualGrafanaDashboards
}
