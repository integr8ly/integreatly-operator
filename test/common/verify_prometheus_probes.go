package common

import (
	"context"
	"fmt"
	"time"

	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// TestPrometheusProbeTargetsActive replaces TestClusterObjectTemplateState.
// It validates that Prometheus is actively scraping required probes by checking the targets API.
func TestPrometheusProbeTargetsActive(t TestingTB, ctx *TestingContext) {
	isInProw, err := isInProw(ctx)
	if err != nil {
		t.Logf("error getting in_prow annotation: %s", err)
	}
	if isInProw {
		t.Skip("Skipping due to Package Operator is missing in Prow")
		return
	}

	probeTargets := []string{
		"integreatly-3scale-admin-ui",
		"integreatly-3scale-system-developer",
		"integreatly-3scale-system-master",
		"integreatly-grafana",
		"integreatly-rhsso",
		"integreatly-rhssouser",
	}

	t.Logf("Starting validation of Prometheus Probe Targets...")
	for _, jobName := range probeTargets {
		// Build full job label as exposed by Prometheus for blackbox probes: probe/<namespace>/<name>
		fullJob := fmt.Sprintf("probe/%s/%s", ObservabilityProductNamespace, jobName)
		if err := checkPrometheusProbeStatus(t, ctx, fullJob); err != nil {
			t.Errorf("Validation failed for job %s: %v", fullJob, err)
		}
	}
}

func checkPrometheusProbeStatus(t TestingTB, ctx *TestingContext, fullJob string) error {
	t.Logf("Polling Prometheus targets for job: %s", fullJob)

	// Poll until the Prometheus targets endpoint shows the probe target as healthy.
	// Max wait time is 2 minutes, polling every 10 seconds.
	pollErr := wait.PollUntilContextTimeout(context.TODO(), 10*time.Second, 2*time.Minute, true, func(pollCtx context.Context) (done bool, err error) {
		targets, err := getPrometheusTargets(ctx)
		if err != nil {
			t.Logf("getPrometheusTargets error for %s: %v", fullJob, err)
			return false, nil
		}

		for _, target := range targets.Active {
			if target.DiscoveredLabels["job"] == fullJob && target.Health == prometheusv1.HealthGood && target.ScrapeURL != "" {
				t.Logf("SUCCESS: Prometheus target %s is active and healthy.", fullJob)
				return true, nil
			}
		}

		t.Logf("Target %s not active/healthy yet. Retrying...", fullJob)
		return false, nil
	})

	if pollErr != nil {
		return fmt.Errorf("failed to confirm Prometheus target %s is active: %w", fullJob, pollErr)
	}

	return nil
}
