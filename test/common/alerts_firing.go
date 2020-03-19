package common

import (
	"encoding/json"
	"strings"
	"testing"

	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

const deadMansSwitch = "DeadMansSwitch"

func TestIntegreatlyAlertsFiring(t *testing.T, ctx *TestingContext) {
	output, err := execToPod("curl localhost:9090/api/v1/alerts",
		"prometheus-application-monitoring-0",
		MonitoringOperatorNamespace,
		"prometheus",
		ctx)
	if err != nil {
		t.Fatal("Failed to exec to prometheus pod:", err)
	}

	// get all found rules from the prometheus api
	var promApiCallOutput prometheusAPIResponse
	err = json.Unmarshal([]byte(output), &promApiCallOutput)
	if err != nil {
		t.Fatalf("Failed to unmarshal json: %s", err)
	}
	var alertsResult prometheusv1.AlertsResult
	err = json.Unmarshal(promApiCallOutput.Data, &alertsResult)
	if err != nil {
		t.Fatalf("Failed to unmarshal json: %s", err)
	}

	// check if any alerts other than DeadMansSwitch are firing or pending
	var firingAlerts []string
	var pendingAlerts []string
	for _, alert := range alertsResult.Alerts {
		alertName := alert.Labels["alertname"]

		// dead mans switch is not firing, so fail the test
		if alertName == deadMansSwitch && alert.State != prometheusv1.AlertStateFiring {
			t.Fatalf("Alert: %s is not firing", deadMansSwitch)
		}
		// check for pending or firing alerts
		if alertName != deadMansSwitch {
			if alert.State == prometheusv1.AlertStateFiring {
				firingAlerts = append(firingAlerts, string(alertName))
			}
			if alert.State == prometheusv1.AlertStatePending {
				pendingAlerts = append(pendingAlerts, string(alertName))
			}
		}
	}

	// report the firing or pending alerts and fail the test
	if len(firingAlerts) > 0 {
		t.Logf("The following alerts were fired: %s", strings.Join(firingAlerts, ", "))
	}
	if len(pendingAlerts) > 0 {
		t.Logf("The following alerts were pending: %s", strings.Join(pendingAlerts, ", "))
	}
	if len(firingAlerts) > 0 || len(pendingAlerts) > 0 {
		t.Fatal("Found pending or firing alerts")
	}
}
