package common

import (
	goctx "context"
	"encoding/json"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"net/url"
	"strings"

	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type dashboardSearchResponse struct {
	Title string `json:"title"`
	UID   string `json:"uid"`
}

type targetDefinition struct {
	Expression string `json:"expr"`
}

type panelDefinition struct {
	Datasource string             `json:"datasource"`
	Targets    []targetDefinition `json:"targets"`
}

type dashboardDefinition struct {
	Panels []panelDefinition `json:"panels"`
}

type dashboardDetailResponse struct {
	Dashboard dashboardDefinition `json:"dashboard"`
}

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

var (
	commonExpectedServices = []string{
		"3scale-admin-ui",
		"3scale-developer-console-ui",
		"rhsso-ui",
		"3scale-system-admin-ui",
	}
	singleTenantRHOAMExpectedServices = []string{
		"rhssouser-ui",
	}
	dashboardsNames = []string{
		"Endpoints Summary",
		"Endpoints Detailed",
		"Endpoints Report",
		"Resource Usage By Pod",
		"Resource Usage for Cluster",
		"Resource Usage By Namespace",
	}
)

func getDashboardExpressions(grafanaPodIp string, curlPodName string, curlContainerName string, prometheusPodName string, ctx *TestingContext, t TestingTB) ([]string, error) {

	// get console master url
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	expectedServices := getExpectedServices(rhmi.Spec.Type)

	rhmiNamespaces, err := getRHOAMNamespaces(rhmi.Spec.NamespacePrefix, prometheusPodName, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get RHMI namespaces: %w", err)
	}
	if len(rhmiNamespaces) == 0 {
		return nil, fmt.Errorf("failed to get RHMI namespaces - namespaces not found")
	}

	rhmiPods, err := getRHMIPods(rhmiNamespaces, prometheusPodName, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get RHMI pods: %w", err)
	}

	// use map as a set, so that same expressions are not queried more than once
	expressions := make(map[string]struct{})

	for _, dashboardName := range dashboardsNames {
		panels, err := getDashboardPanels(dashboardName, grafanaPodIp, curlPodName, curlContainerName, ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get dashboard panels: %w", err)
		}

		for _, panel := range panels {
			if panel.Datasource == "Prometheus" {
				for _, target := range panel.Targets {
					// "1" has no special meaning there, it is used just to replace variables with some allowed value
					expression := strings.ReplaceAll(target.Expression, "$__range_s", "1")

					// "1d" has no special meaning there, it is used just to replace variables with some allowed value
					expression = strings.ReplaceAll(expression, "$__range", "1d")

					switch {
					case strings.Contains(expression, "$services"):
						for _, service := range expectedServices {
							finalExpression := strings.ReplaceAll(expression, "$services", service)
							expressions[finalExpression] = struct{}{}
						}
					case strings.Contains(expression, "$namespace"):
						for _, namespace := range rhmiNamespaces {
							namespaceExpression := strings.ReplaceAll(expression, "$namespace", namespace)

							for _, pod := range rhmiPods[namespace] {
								finalExpression := strings.ReplaceAll(namespaceExpression, "$pod", pod)
								expressions[finalExpression] = struct{}{}
							}
						}
					default:
						expressions[expression] = struct{}{}
					}
				}
			}
		}
	}

	var expressionsSlice []string
	for expression := range expressions {
		expressionsSlice = append(expressionsSlice, expression)
	}

	return expressionsSlice, nil
}

func getExpectedServices(installType string) []string {
	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installType)) {
		return commonExpectedServices
	} else {
		return append(commonExpectedServices, singleTenantRHOAMExpectedServices...)
	}
}

func getRHMIPods(namespaces []string, prometheusPodName string, ctx *TestingContext) (map[string][]string, error) {
	pods := make(map[string][]string)

	var queries []string

	for _, namespace := range namespaces {
		queries = append(queries, "kube_pod_info{namespace=~'"+namespace+"'}")
	}

	queryOutputs, err := queryPrometheusMany(queries, prometheusPodName, ctx)
	if err != nil {
		return nil, err
	}

	for i, queryOutput := range queryOutputs {
		queryResult, err := getPrometheusQueryResult(queryOutput)
		if err != nil {
			return nil, fmt.Errorf("failed to query prometheus: %w", err)
		}
		if queryResult == nil {
			continue
		}
		for _, result := range queryResult {
			pods[namespaces[i]] = append(pods[namespaces[i]], result.Metric["pod"].(string))
		}
	}
	return pods, nil
}

func getRHOAMNamespaces(namespacePrefix, prometheusPodName string, ctx *TestingContext) ([]string, error) {
	queryResult, err := queryPrometheus(fmt.Sprintf("kube_namespace_labels{namespace=~'%s.*'}", namespacePrefix), prometheusPodName, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query prometheus: %w", err)
	}

	var namespaces []string

	for _, result := range queryResult {
		namespaces = append(namespaces, result.Metric["namespace"].(string))
	}

	return namespaces, nil
}

func getDashboardPanels(dashboardName string, grafanaPodIp string, curlPodName string, curlContainerName string, ctx *TestingContext) ([]panelDefinition, error) {
	query := url.QueryEscape(dashboardName)
	searchOutput, err := curlGrafana(grafanaPodIp, fmt.Sprintf("/api/search?query=%s", query), curlPodName, curlContainerName, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to curl grafana search: %w, dashboard name: %s, grafanaPodIp: %s, curlPodName: %s, curlContainerName: %s", err, dashboardName, grafanaPodIp, curlPodName, curlContainerName)
	}

	var dashboardSearch []dashboardSearchResponse
	err = json.Unmarshal([]byte(searchOutput), &dashboardSearch)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall json: %w", err)
	}

	if len(dashboardSearch) != 1 {
		return nil, fmt.Errorf(dashboardName + " dashboard not found")
	}

	dashboardOutput, err := curlGrafana(grafanaPodIp, fmt.Sprintf("/api/dashboards/uid/%s", dashboardSearch[0].UID), curlPodName, curlContainerName, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to curl grafana dashboard: %w, grafanaPodIp: %s, curlPodName: %s, curlContainerName: %s, dashboard uuid: %s", err, grafanaPodIp, curlPodName, curlContainerName, dashboardSearch[0].UID)
	}

	var dashboardDetail dashboardDetailResponse
	err = json.Unmarshal([]byte(dashboardOutput), &dashboardDetail)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall json: %w", err)
	}

	return dashboardDetail.Dashboard.Panels, nil
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

func curlGrafana(grafanaPodIp string, path string, curlPodName string, curlContainerName string, ctx *TestingContext) (string, error) {
	return execToPod(fmt.Sprintf("wget -qO - %s:3000", grafanaPodIp)+path,
		curlPodName,
		ObservabilityProductNamespace,
		curlContainerName, ctx)
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

func queryPrometheusMany(queries []string, podName string, ctx *TestingContext) ([]string, error) {
	command := ""

	for _, query := range queries {
		command += "wget -qO - localhost:9090/api/v1/query?query=" + url.QueryEscape(query) + ";"
	}

	output, err := execToPod(command,
		podName,
		ObservabilityProductNamespace,
		"prometheus", ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to exec to prometheus pod: %w", err)
	}

	output = strings.ReplaceAll(output, "}{", "}\n{")
	queryOutputs := strings.Split(output, "\n")

	return queryOutputs, nil
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
