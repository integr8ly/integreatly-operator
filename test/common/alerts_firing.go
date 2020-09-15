package common

import (
	"encoding/json"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"k8s.io/apimachinery/pkg/util/wait"
	"strings"
	"testing"
	"time"

	goctx "context"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
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

var (
	// Applicable to all install types
	commonPodNamespaces = []string{
		RHMIOperatorNamespace,
		MonitoringOperatorNamespace,
		MonitoringFederateNamespace,
		CloudResourceOperatorNamespace,
		RHSSOUserProductOperatorNamespace,
		RHSSOUserOperatorNamespace,
		RHSSOProductNamespace,
		RHSSOOperatorNamespace,
		ThreeScaleProductNamespace,
		ThreeScaleOperatorNamespace,
	}
	// Applicable to rhmi 2 install types only
	rhmi2PodNamespaces = []string{
		AMQOnlineOperatorNamespace,
		ApicuritoProductNamespace,
		ApicuritoOperatorNamespace,
		CodeReadyProductNamespace,
		CodeReadyOperatorNamespace,
		FuseProductNamespace,
		FuseOperatorNamespace,
		SolutionExplorerProductNamespace,
		SolutionExplorerOperatorNamespace,
		UPSProductNamespace,
		UPSOperatorNamespace,
	}
)

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
			str.WriteString(fmt.Sprintf("\n\talert: %s", alert.alertName))
			if alert.podName != "" {
				str.WriteString(fmt.Sprintf(", for pod: %s, in namespace: %s", alert.podName, alert.namespace))
			}
		}
	}
	if len(e.alertsFiring) != 0 {
		str.WriteString("\nThe following alerts were fired:")
		for _, alert := range e.alertsFiring {
			str.WriteString(fmt.Sprintf("\n\talert: %s", alert.alertName))
			if alert.podName != "" {
				str.WriteString(fmt.Sprintf(", for pod: %s, in namespace: %s", alert.podName, alert.namespace))
			}
		}
	}
	return str.String()
}

// isNotEmpty checks whether or not the error contains firing or pending alerts
func (e *alertsFiringError) isValid() bool {
	return !e.deadMansSwitchFiring || len(e.alertsFiring) != 0 || len(e.alertsPending) != 0
}

// This test ensures that no alerts are firing during or after installation
func TestIntegreatlyAlertsFiring(t *testing.T, ctx *TestingContext) {
	//fail immediately if one or more alerts have fired
	if err := getFiringAlerts(t, ctx); err != nil {
		podLogs(t, ctx)
		t.Fatal(err.Error())
	}

}
func getFiringAlerts(t *testing.T, ctx *TestingContext) error {
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

	// create a custom alerts error to keep track of all firing alerts
	alertsError := &alertsFiringError{
		alertsFiring:         []alertTestMetadata{},
		deadMansSwitchFiring: true,
	}

	// // check if any alerts other than DeadMansSwitch are firing
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
		// check for firing alerts
		if alertName != deadMansSwitch {
			if alert.State == prometheusv1.AlertStateFiring {
				alertsError.alertsFiring = append(alertsError.alertsFiring, alertMetadata)

			}

		}
	}

	if alertsError.isValid() {
		return alertsError

	}
	return nil
}

// Makes a api call a to get all pods in the rhmi namespaces
func podLogs(t *testing.T, ctx *TestingContext) {
	pods := &corev1.PodList{}

	rhmi, err := getRHMI(ctx.Client)
	if err != nil {
		t.Fatalf("failed to get the RHMI: %s", err)
	}
	podNamespaces := getPodNamespaces(rhmi.Spec.Type)

	for _, namespaces := range podNamespaces {
		err := ctx.Client.List(goctx.TODO(), pods, &k8sclient.ListOptions{Namespace: namespaces})
		if err != nil {
			t.Error("Error getting namespaces:", err)
		}
		t.Log("Namespace:", namespaces)
		for _, pod := range pods.Items {
			t.Logf("\tPod name: %s, Status: %s", pod.Name, pod.Status.Phase)
		}
	}
}

func getPodNamespaces(installType string) []string {
	if installType == string(integreatlyv1alpha1.InstallationTypeManaged3scale) {
		return commonPodNamespaces
	} else {
		return append(commonPodNamespaces, rhmi2PodNamespaces...)
	}
}

// TestIntegreatlyAlertsFiring reports any firing or pending alerts
func TestIntegreatlyAlertsPendingOrFiring(t *testing.T, ctx *TestingContext) {
	var lastError error

	// retry the tests every minute for up to 15 minutes
	monitoringTimeout := 15 * time.Minute
	monitoringRetryInterval := 1 * time.Minute
	err := wait.Poll(monitoringRetryInterval, monitoringTimeout, func() (done bool, err error) {
		if newErr := getFiringOrPendingAlerts(ctx); newErr != nil {
			lastError = newErr
			if _, ok := newErr.(*alertsFiringError); ok {
				t.Log("Waiting 1 minute for alerts to normalise before retrying")
				return false, nil
			}
			return false, newErr
		}
		return true, nil
	},
	)
	if err != nil {
		podLogs(t, ctx)
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
	if alertsError.isValid() {
		return alertsError

	}
	return nil
}
