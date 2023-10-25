package common

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

func mangedApiTargets() map[string][]string {
	return map[string][]string{
		"probe/" + ObservabilityProductNamespace: {
			"/integreatly-3scale-admin-ui",
			"/integreatly-3scale-system-developer",
			"/integreatly-3scale-system-master",
			"/integreatly-rhsso",
			"/integreatly-rhssouser",
		},
		"serviceMonitor/" + ObservabilityProductNamespace: {
			"/rhsso-keycloak-service-monitor/0",
			"/rhsso-keycloak-service-monitor/1",
		},
		"serviceMonitor/" + ObservabilityProductNamespace: {
			"/rhsso-operator-metrics/0",
			"/rhsso-operator-metrics/1",
		},
		"serviceMonitor/" + ObservabilityProductNamespace: {
			"/user-sso-keycloak-service-monitor/0",
			"/user-sso-keycloak-service-monitor/1",
		},
		"serviceMonitor/" + ObservabilityProductNamespace: {
			"/user-sso-operator-rhsso-operator-metrics/0",
			"/user-sso-operator-rhsso-operator-metrics/1",
		},
		"serviceMonitor/" + ObservabilityProductNamespace: {"/cloud-resource-operator-metrics/0"},
		"serviceMonitor/" + ObservabilityProductNamespace: {"/ratelimit/0"},
		"serviceMonitor/" + ObservabilityProductNamespace: {"/threescale-operator-controller-manager-metrics-monitor/0"},
		"serviceMonitor/" + ObservabilityProductNamespace: {"/3scale-service-monitor/0"},
		"serviceMonitor/" + ObservabilityProductNamespace: {"/openshift-monitoring-federation/0"},
		"serviceMonitor/" + RHOAMOperatorNamespace:        {"/rhmi-operator-metrics/0"},
	}
}

func mtMangedApiTargets() map[string][]string {
	return map[string][]string{
		"probe/" + ObservabilityProductNamespace: {
			"/integreatly-3scale-admin-ui",
			"/integreatly-3scale-system-developer",
			"/integreatly-3scale-system-master",
			//"/integreatly-grafana",
			"/integreatly-rhsso",
		},
		"serviceMonitor/" + ObservabilityProductNamespace: {
			"/rhsso-keycloak-service-monitor/0",
			"/rhsso-keycloak-service-monitor/1",
		},
		"serviceMonitor/" + ObservabilityProductNamespace: {
			"/rhsso-operator-metrics/0",
			"/rhsso-operator-metrics/1",
		},
		"serviceMonitor/" + ObservabilityProductNamespace: {"/cloud-resource-operator-metrics/0"},
		"serviceMonitor/" + ObservabilityProductNamespace: {"/ratelimit/0"},
		"serviceMonitor/" + ObservabilityProductNamespace: {"/threescale-operator-controller-manager-metrics-monitor/0"},
		"serviceMonitor/" + ObservabilityProductNamespace: {"/3scale-service-monitor/0"},
		"serviceMonitor/" + ObservabilityProductNamespace: {"/openshift-monitoring-federation/0"},
		"serviceMonitor/" + RHOAMOperatorNamespace:        {"/rhmi-operator-metrics/0"},
	}
}

func TestMetricsScrappedByPrometheus(t TestingTB, ctx *TestingContext) {
	// get all active targets in prometheus
	targetsResult, err := getPrometheusTargets(ctx)
	if err != nil {
		t.Fatalf("%v", err)
	}

	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	// for every namespace
	for ns, targets := range getTargets(rhmi.Spec.Type) {
		// for every listed target in namespace
		for _, targetName := range targets {
			// check that metrics is being correctly scrapped by target
			correctlyScrapping := false
			for _, target := range targetsResult.Active {
				if target.DiscoveredLabels["job"] == fmt.Sprintf("%s%s", ns, targetName) && target.Health == prometheusv1.HealthGood && target.ScrapeURL != "" {
					correctlyScrapping = true
					break
				}
			}

			if !correctlyScrapping {
				t.Errorf("Not correctly scrapping Prometheus target: %s%s", ns, targetName)
			}
		}
	}
}

func getPrometheusTargets(ctx *TestingContext) (*prometheusv1.TargetsResult, error) {
	output, err := execToPod("wget -qO - localhost:9090/api/v1/targets?state=active",
		ObservabilityPrometheusPodName,
		ObservabilityProductNamespace,
		"prometheus",
		ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to exec to prometheus pod: %v", err)
	}

	// get all found active targets from the prometheus api
	var promApiCallOutput prometheusAPIResponse
	if err = json.Unmarshal([]byte(output), &promApiCallOutput); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json: %v", err)
	}

	var targetsResult prometheusv1.TargetsResult
	if err = json.Unmarshal(promApiCallOutput.Data, &targetsResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json: %s", err)
	}
	return &targetsResult, nil
}

func getTargets(installType string) map[string][]string {
	if integreatlyv1alpha1.IsRHOAMSingletenant(integreatlyv1alpha1.InstallationType(installType)) {
		return mangedApiTargets()
	} else if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installType)) {
		return mtMangedApiTargets()
	} else {
		// TODO - return list for managed install type
		return map[string][]string{}
	}
}

func TestRhoamVersionMetricExposed(t TestingTB, ctx *TestingContext) {
	const rhoamVersionKey = "rhoam_version"
	// Get the rhoam_version metric from prometheus
	promQueryRes, err := queryPrometheus(rhoamVersionKey, ObservabilityPrometheusPodName, ctx)
	if err != nil {
		t.Fatalf("Failed to query prometheus: %s", err)
	}
	if len(promQueryRes) == 0 {
		t.Fatalf("No results for metric %s ", rhoamVersionKey)
	}
	version, ok := promQueryRes[0].Metric["version"]
	if !ok {
		t.Fatalf("Unable to find version field in metric")
	}

	rhoamVersionValue := version.(string)
	// Semver regex (https://regexr.com/39s32)
	re := regexp.MustCompile(`^((([0-9]+)\.([0-9]+)\.([0-9]+)(?:-([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?)(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?)$`)
	if !re.MatchString(rhoamVersionValue) {
		t.Fatalf("Failed to validate RHOAM version format. Expected semantic version, got %s", rhoamVersionValue)
	}
}

func TestAdditionalBlackboxTargets(t TestingTB, ctx *TestingContext) {
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	// get all active targets in prometheus
	expectedBlackboxTargets := getBlackboxTargets(rhmi.Spec.Type)
	targetsResult, err := getPrometheusTargets(ctx)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if len(targetsResult.Active) == 0 {
		t.Fatalf("no active prometheus targets", err)
	}
	var blackboxTargets []string
	for _, target := range targetsResult.Active {
		jobValue := target.Labels["job"]
		serviceValue := target.Labels["service"]
		if jobValue == "blackbox" {
			blackboxTargets = append(blackboxTargets, string(serviceValue))
		}
	}
	if !reflect.DeepEqual(blackboxTargets, expectedBlackboxTargets) {
		t.Fatalf("expected prometheus blackbox targets %v, got %v", expectedBlackboxTargets, blackboxTargets)
	}
}

func getBlackboxTargets(installType string) []string {
	var blackboxTargets []string
	if integreatlyv1alpha1.IsRHOAMSingletenant(integreatlyv1alpha1.InstallationType(installType)) {
		blackboxTargets = []string{
			"3scale-admin-ui",
			"3scale-developer-console-ui",
			"3scale-system-admin-ui",
			"grafana-ui",
			"rhsso-ui",
			"rhssouser-ui",
		}
	} else if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installType)) {
		blackboxTargets = []string{
			"3scale-admin-ui",
			"3scale-developer-console-ui",
			"3scale-system-admin-ui",
			"grafana-ui",
			"rhsso-ui",
		}
	}

	return blackboxTargets
}
