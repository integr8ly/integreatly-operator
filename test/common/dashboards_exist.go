package common

import (
	goctx "context"
	"encoding/json"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

type dashboardsTestRule struct {
	Title string `json:"title"`
}

var expectedDashboards = []dashboardsTestRule{
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
		Title: "Syndesis - Infra - API",
	},
	{
		Title: "Syndesis - Infra - DB",
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
}

func TestIntegreatlyDashboardsExist(t *testing.T, ctx *TestingContext) {
	pods := &corev1.PodList{}
	opts := []k8sclient.ListOption{
		k8sclient.InNamespace(namespacePrefix + "middleware-monitoring-operator"),
		k8sclient.MatchingLabels{"app": "grafana"},
	}

	err := ctx.Client.List(goctx.TODO(), pods, opts...)
	if err != nil {
		t.Fatal("failed to list pods", err)
	}

	if len(pods.Items) != 1 {
		t.Fatal("grafana pod not found")
	}

	output, err := execToPod("curl localhost:3000/api/search",
		pods.Items[0].ObjectMeta.Name,
		namespacePrefix+"middleware-monitoring-operator",
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

	dashboardDiffUnexpected := dashboardDifference(grafanaApiCallOutput, expectedDashboards)
	dashboardDiffMissing := dashboardDifference(expectedDashboards, grafanaApiCallOutput)

	if len(dashboardDiffUnexpected) > 0 || len(dashboardDiffMissing) > 0 {
		t.Logf("unexpected dashboards found: %s", strings.Join(dashboardDiffUnexpected, ", "))
		t.Logf("missing dashboards found: %s", strings.Join(dashboardDiffMissing, ", "))
		t.Fail()
	}
}

// dashboardDifference returns the elements in list of dashboardsTestRule `diffSource` that aren't in list `diffTarget`.
func dashboardDifference(diffSource, diffTarget []dashboardsTestRule) []string {
	diffLookupMap := make(map[string]struct{}, len(diffTarget))
	for _, dashboard := range diffTarget {
		diffLookupMap[dashboard.Title] = struct{}{}
	}

	var diff []string
	for _, x := range diffSource {
		if _, found := diffLookupMap[x.Title]; !found {
			diff = append(diff, x.Title)
		}
	}
	return diff
}
