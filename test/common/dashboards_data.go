package common

import (
	goctx "context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
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
		"rhssouser-ui",
		"3scale-system-admin-ui",
	}
	rhmi2ExpectedServices = []string{
		"apicurito-ui",
		"codeready-ui",
		"amq-service-broker",
		"webapp-ui",
		"syndesis-ui",
		"ups-ui",
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

// TestDashboardsData verifies that all dashboards are installed and all the graphs are filled with data
func TestDashboardsData(t TestingTB, ctx *TestingContext) {
	// Get grafana pod ip to curl
	monitoringGrafanaPods := getGrafanaPods(t, ctx, MonitoringOperatorNamespace)
	grafanaPodIP := monitoringGrafanaPods.Items[0].Status.PodIP

	// Pod and container name to perform curls from
	curlContainerName := "prometheus"
	prometheusPodName, err := getMonitoringAppPodName("prometheus", ctx)
	if err != nil {
		t.Fatal("failed to get prometheus pod name", err)
	}

	// retry the tests every minute for up to 10 minutes
	monitoringTimeout := 10 * time.Minute
	monitoringRetryInterval := 1 * time.Minute
	err = wait.PollImmediate(monitoringRetryInterval, monitoringTimeout, func() (done bool, err error) {
		expressions, err := getDashboardExpressions(grafanaPodIP, prometheusPodName, curlContainerName, prometheusPodName, ctx, t)
		if err != nil {
			return false, fmt.Errorf("failed to get dashboard expressions: %w", err)
		}

		queryOutputs, err := queryPrometheusMany(expressions, prometheusPodName, ctx)
		if err != nil {
			return false, fmt.Errorf("failed to query prometheus many: %w", err)
		}

		var failedQueries []string

		for i, queryOutput := range queryOutputs {
			_, err := getPrometheusQueryResult(queryOutput)
			if err != nil {
				failedQueries = append(failedQueries, expressions[i])
			}
		}

		failed := false

		for _, failedQuery := range failedQueries {
			// not all containers define resource requests and limits -> expected failure
			if strings.Contains(failedQuery, "kube_pod_container_resource_requests") ||
				strings.Contains(failedQuery, "kube_pod_container_resource_limits") {
				continue
			}

			// following query might fail, but dashboards are still visible -> ignoring
			if strings.Contains(failedQuery, "probe_ssl_earliest_cert_expiry") {
				continue
			}

			// completed pods don't use resources -> expected failure
			matched, _ := regexp.Match(`pod=~'.*(deploy|hook-pre|hook-post|build|pv-backup)*'`, []byte(failedQuery))
			if matched {
				continue
			}

			t.Log("failed query:", failedQuery)
			failed = true
		}

		if failed {
			t.Log("waiting 1 minute before retrying")
		}

		return !failed, nil
	})
	if err != nil {
		t.Fatalf("failed queries: %s", err)
	}
}

func getDashboardExpressions(grafanaPodIp string, curlPodName string, curlContainerName string, prometheusPodName string, ctx *TestingContext, t TestingTB) ([]string, error) {

	// get console master url
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	expectedServices := getExpectedServices(rhmi.Spec.Type)

	rhmiNamespaces, err := getRHMINamespaces(rhmi.Spec.NamespacePrefix, prometheusPodName, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get RHMI namespaces: %w", err)
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
	if integreatlyv1alpha1.IsRHOAM(integreatlyv1alpha1.InstallationType(installType)) {
		return commonExpectedServices
	} else {
		return append(commonExpectedServices, rhmi2ExpectedServices...)
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

func getRHMINamespaces(namespacePrefix, prometheusPodName string, ctx *TestingContext) ([]string, error) {
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
		k8sclient.InNamespace(MonitoringOperatorNamespace),
		k8sclient.MatchingLabels{"app": app},
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
	return execToPod(fmt.Sprintf("curl %s:3000", grafanaPodIp)+path,
		curlPodName,
		MonitoringOperatorNamespace,
		curlContainerName, ctx)
}

func queryPrometheus(query string, podName string, ctx *TestingContext) ([]prometheusQueryResult, error) {
	queryOutput, err := execToPod("curl localhost:9090/api/v1/query?query="+url.QueryEscape(query),
		podName,
		MonitoringOperatorNamespace,
		"prometheus", ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to exec to prometheus pod: %w", err)
	}

	return getPrometheusQueryResult(queryOutput)
}

func queryPrometheusMany(queries []string, podName string, ctx *TestingContext) ([]string, error) {
	command := ""

	for _, query := range queries {
		command += "curl localhost:9090/api/v1/query?query=" + url.QueryEscape(query) + ";"
	}

	output, err := execToPod(command,
		podName,
		MonitoringOperatorNamespace,
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
