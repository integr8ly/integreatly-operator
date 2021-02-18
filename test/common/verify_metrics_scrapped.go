package common

import (
	"encoding/json"
	"fmt"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

func mangedApiTargets() map[string][]string {
	return map[string][]string{
		// TODO: Should include other expected targets
		Marin3rProductNamespace: {
			"/prom-statsd-exporter/0",
		},
	}
}

func TestMetricsScrappedByPrometheus(t TestingTB, ctx *TestingContext) {
	// get all active targets in prometheus
	output, err := execToPod("curl localhost:9090/api/v1/targets?state=active",
		"prometheus-application-monitoring-0",
		GetPrefixedNamespace("middleware-monitoring-operator"),
		"prometheus",
		ctx)
	if err != nil {
		t.Fatalf("failed to exec to prometheus pod: %s", err)
	}

	// get all found active targets from the prometheus api
	var promApiCallOutput prometheusAPIResponse
	err = json.Unmarshal([]byte(output), &promApiCallOutput)
	if err != nil {
		t.Fatalf("failed to unmarshal json: %s", err)
	}

	var targetResult prometheusv1.TargetsResult
	err = json.Unmarshal(promApiCallOutput.Data, &targetResult)
	if err != nil {
		t.Fatalf("failed to unmarshal json: %s", err)
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
			for _, target := range targetResult.Active {
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

func getTargets(installType string) map[string][]string {
	if installType == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		return mangedApiTargets()
	} else {
		// TODO - return list for managed install type
		return map[string][]string{}
	}
}
