package common

import (
	goctx "context"
	"encoding/json"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"net/url"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type prometheusQueryResult struct {
	Metric map[string]interface{} `json:"metric"`
	Value  []interface{}          `json:"value"`
}

type prometheusQueryData struct {
	Result []prometheusQueryResult `json:"result"`
}

type prometheusQueryResponse struct {
	Status string              `json:"status"`
	Data   prometheusQueryData `json:"data"`
}

func getMonitoringAppPodName(app string, ctx *TestingContext) (string, error) {
	pods := &corev1.PodList{}
	opts := []k8sclient.ListOption{
		k8sclient.InNamespace(ObservabilityProductNamespace),
		k8sclient.MatchingLabels{"app.kubernetes.io/name": app},
	}

	err := ctx.Client.List(goctx.TODO(), pods, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to list pods: %w", err)
	}

	if len(pods.Items) < 1 {
		return "", fmt.Errorf("grafana pod not found for app %s", app)
	}

	return pods.Items[0].ObjectMeta.Name, nil
}

func queryPrometheus(query string, podName string, ctx *TestingContext) ([]prometheusQueryResult, error) {
	queryOutput, err := execToPod("wget -qO - localhost:9090/api/v1/query?query="+url.QueryEscape(query),
		podName,
		ObservabilityProductNamespace,
		"prometheus", ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to exec to prometheus pod: %w", err)
	}

	return getPrometheusQueryResult(queryOutput)
}

func getPrometheusQueryResult(output string) ([]prometheusQueryResult, error) {
	var queryResponse prometheusQueryResponse
	err := json.Unmarshal([]byte(output), &queryResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall json: %w", err)
	}

	if queryResponse.Status != "success" {
		return nil, fmt.Errorf("response status: %s", queryResponse.Status)
	}

	if len(queryResponse.Data.Result) == 0 {
		return nil, nil
	}

	return queryResponse.Data.Result, nil
}
