package common

import (
	"bytes"
	goctx "context"
	"encoding/json"
	"fmt"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/remotecommand"
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

	type Output []dashboardsTestRule

	output, err := execToGrafanaPod("curl localhost:3000/api/search",
		pods.Items[0].ObjectMeta.Name,
		namespacePrefix+"middleware-monitoring-operator",
		"grafana", ctx)
	if err != nil {
		t.Fatal("failed to exec to pod:", err)
	}

	var grafanaApiCallOutput Output
	err = json.Unmarshal([]byte(output), &grafanaApiCallOutput)
	if err != nil {
		t.Logf("Failed to unmarshall json: %s", err)
	}

	if len(grafanaApiCallOutput) == 0 {
		t.Fatal("grafana dashboard not found : %w", grafanaApiCallOutput)
	}

	ruleDiff := dashboardDifference(grafanaApiCallOutput, expectedDashboards)
	if len(ruleDiff) > 0 {
		t.Fatalf("Unexpected dashboards were found. If these dashboards were intentionally added, please update this test to include them. If these dashboards were not added intentionally or you are not sure, please create a JIRA and discuss with the monitoring team on how best to proceed. Additional dashboards: %s", strings.Join(ruleDiff, ", "))
	}

	ruleDiff = dashboardDifference(expectedDashboards, grafanaApiCallOutput)
	if len(ruleDiff) > 0 {
		t.Fatalf("Missing dashboards were found. If the removal of these dashboards was intentional, please update this test to remove them from the check. If the removal of these dashboards was not intended or you are not sure, please create a JIRA and discuss with the monitoring team on how best to proceed. Missing dashboards: %s", strings.Join(ruleDiff, ", "))
	}
}

func execToGrafanaPod(command string, podName string, namespace string, container string, ctx *TestingContext) (string, error) {
	req := ctx.KubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		Param("container", container)
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return "", fmt.Errorf("error adding to scheme: %v", err)
	}
	parameterCodec := runtime.NewParameterCodec(scheme)
	req.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command:   strings.Fields(command),
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, parameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(ctx.KubeConfig, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("error while creating Executor: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return "", fmt.Errorf("error in Stream: %v", err)
	}
	return stdout.String(), nil
}

// dashboardDifference returns the elements in list of dashboardsTestRule `a` that aren't in list `b`.
func dashboardDifference(a, b []dashboardsTestRule) []string {
	diffLookupMap := make(map[string]struct{}, len(b))
	for _, x := range b {
		diffLookupMap[x.Title] = struct{}{}
	}

	var diff []string
	for _, x := range a {
		if _, found := diffLookupMap[x.Title]; !found {
			diff = append(diff, x.Title)
		}
	}
	return diff
}
