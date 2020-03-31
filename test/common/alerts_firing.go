package common

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

const deadMansSwitch = "DeadMansSwitch"

// alertsTestMetadata contains metadata about the alert
type alertTestMetadata struct {
	alertName string
	podName   string
	namespace string
}

// alertsFiringError is a custom alerts error
type alertsFiringError struct {
	alertsPending        []alertTestMetadata
	alertsFiring         []alertTestMetadata
	deadMansSwitchFiring bool
}

// Error implements the error interface and returns a readable output message
func (e *alertsFiringError) Error() string {
	var str strings.Builder
	if e.deadMansSwitchFiring {
		str.WriteString("\nThe following alerts were not fired, but were expected to be firing:")
		str.WriteString(fmt.Sprintf("\n\talert: %s", deadMansSwitch))
	}
	if len(e.alertsPending) != 0 {
		str.WriteString("\nThe following alerts were pending:")
		for _, alert := range e.alertsPending {
			str.WriteString(fmt.Sprintf("\n\talert: %s, for pod: %s, in namespace: %s", alert.alertName, alert.podName, alert.namespace))
		}
	}
	if len(e.alertsFiring) != 0 {
		str.WriteString("\nThe following alerts were fired:")
		for _, alert := range e.alertsFiring {
			str.WriteString(fmt.Sprintf("\n\talert: %s, for pod: %s, in namespace: %s", alert.alertName, alert.podName, alert.namespace))
		}
	}
	return str.String()
}

// isNotEmpty checks whether or not the error contains firing or pending alerts
func (e *alertsFiringError) isNotEmpty() bool {
	return !e.deadMansSwitchFiring || len(e.alertsPending) != 0 || len(e.alertsPending) != 0
}

func TestIntegreatlyAlertsFiring(t *testing.T, ctx *TestingContext) {
	var lastError error

	// retry the tests every minute for up to 15 minutes
	monitoringTimeout := 15 * time.Minute
	monitoringRetryInterval := 1 * time.Minute
	err := wait.Poll(monitoringRetryInterval, monitoringTimeout, func() (done bool, err error) {
		if newErr := getFiringOrPendingAlerts(ctx); newErr != nil {
			lastError = newErr
			if _, ok := newErr.(*alertsFiringError); ok {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	if err != nil {
		t.Fatal(lastError.Error())
	}
}

func getFiringOrPendingAlerts(ctx *TestingContext) error {
	output, err := execToPod("curl localhost:9090/api/v1/alerts",
		"prometheus-application-monitoring-0",
		MonitoringOperatorNamespace,
		"prometheus",
		ctx)
	if err != nil {
		return fmt.Errorf("failed to exec to prometheus pod: %w", err)
	}

	// get all found rules from the prometheus api
	var promApiCallOutput prometheusAPIResponse
	err = json.Unmarshal([]byte(output), &promApiCallOutput)
	if err != nil {
		return fmt.Errorf("failed to unmarshal json: %w", err)
	}
	var alertsResult prometheusv1.AlertsResult
	err = json.Unmarshal(promApiCallOutput.Data, &alertsResult)
	if err != nil {
		return fmt.Errorf("failed to unmarshal json: %w", err)
	}

	// create a custom alerts error to keep track of all pending and firing alerts
	alertsError := &alertsFiringError{
		alertsPending:        []alertTestMetadata{},
		alertsFiring:         []alertTestMetadata{},
		deadMansSwitchFiring: true,
	}

	// check if any alerts other than DeadMansSwitch are firing or pending
	for _, alert := range alertsResult.Alerts {
		alertName := string(alert.Labels["alertname"])
		alertMetadata := alertTestMetadata{
			alertName: alertName,
			podName:   string(alert.Labels["pod"]),
			namespace: string(alert.Labels["namespace"]),
		}

		// check if dead mans switch is firing
		if alertName == deadMansSwitch && alert.State != prometheusv1.AlertStateFiring {
			alertsError.deadMansSwitchFiring = false
		}
		// check for pending or firing alerts
		if alertName != deadMansSwitch {
			if alert.State == prometheusv1.AlertStateFiring {
				alertsError.alertsFiring = append(alertsError.alertsFiring, alertMetadata)
			}
			if alert.State == prometheusv1.AlertStatePending {
				alertsError.alertsPending = append(alertsError.alertsPending, alertMetadata)
			}
		}
	}
	if alertsError.isNotEmpty() {
		return alertsError
	}
	return nil
}
