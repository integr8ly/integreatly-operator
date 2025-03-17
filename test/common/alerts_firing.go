package common

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/types"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"k8s.io/apimachinery/pkg/util/wait"

	goctx "context"

	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const deadMansSwitch = "DeadMansSwitch"
const missingSmtpSecret = "SendgridSmtpSecretExists"    // #nosec G101 -- This is a false positive
const dmsSecretAlertName = "DeadMansSnitchSecretExists" // #nosec G101 -- This is a false positive

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
	// Applicable to RHOAM
	rhoamExpectedPodNamespaces = []string{
		Marin3rOperatorNamespace,
		Marin3rProductNamespace,
		CustomerGrafanaNamespace,
		RHSSOUserProductNamespace,
		RHSSOUserOperatorNamespace,
	}

	// Applicable to all install types
	commonPodNamespaces = []string{
		RHOAMOperatorNamespace,
		ObservabilityProductNamespace,
		CloudResourceOperatorNamespace,
		RHSSOProductNamespace,
		RHSSOOperatorNamespace,
		ThreeScaleProductNamespace,
		ThreeScaleOperatorNamespace,
	}
)

// Error implements the error interface and returns a readable output message
func (e *alertsFiringError) Error() string {
	var str strings.Builder

	if !e.deadMansSwitchFiring {
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

func smtpMissing(ctx context.Context, serverClient k8sclient.Client, t TestingTB) bool {
	smtpSecret := &corev1.Secret{}
	if err := serverClient.Get(ctx, types.NamespacedName{Name: SMTPSecretName, Namespace: RHOAMOperatorNamespace}, smtpSecret); err != nil {
		t.Logf("SMTP secret is missing from %s namespace, expecting %s to fire\n", RHOAMOperatorNamespace, missingSmtpSecret)
		return true
	}
	return false
}

func dmsSecretMissing(ctx context.Context, serverClient k8sclient.Client, t TestingTB) bool {
	dmsSecret := &corev1.Secret{}
	if err := serverClient.Get(ctx, types.NamespacedName{Name: DMSSecretName, Namespace: RHOAMOperatorNamespace}, dmsSecret); err != nil {
		t.Logf("DeadMansSnitch secret is missing from %s namespace, expecting %s to fire\n", RHOAMOperatorNamespace, dmsSecretAlertName)
		return true
	}
	return false
}

// This test ensures that no alerts are firing during or after installation
func TestIntegreatlyAlertsFiring(t TestingTB, ctx *TestingContext) {
	//fail immediately if one or more alerts have fired
	if err := getFiringAlerts(t, ctx); err != nil {
		podLogs(t, ctx)
		isInProw, e := isInProw(ctx)
		if e != nil {
			t.Logf("error getting in_prow annotation: %s", e)
		}
		if isInProw {
			t.Skipf("Skipping due to known issue in Prow, UIBBT alests fire there: %v, reported in Jira: https://issues.redhat.com/browse/MGDAPI-5193", err.Error())
		} else {
			t.Fatal(err.Error())
		}
	}

}
func getFiringAlerts(t TestingTB, ctx *TestingContext) error {
	output, err := execToPod("curl -s localhost:9090/api/v1/alerts",
		ObservabilityPrometheusPodName,
		ObservabilityProductNamespace,
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
		//  ignore firing dms and missingSmtp alerts
		if alertName == deadMansSwitch ||
			(alertName == missingSmtpSecret &&
				alert.State == prometheusv1.AlertStateFiring &&
				smtpMissing(context.TODO(), ctx.Client, t)) ||
			(alertName == dmsSecretAlertName &&
				alert.State == prometheusv1.AlertStateFiring &&
				dmsSecretMissing(context.TODO(), ctx.Client, t)) {
			continue
		}

		// tmp workaround for **ErrorBudgetBurn alerts firing during installation (https://issues.redhat.com/browse/MGDAPI-5992)
		ignoredAlertsPatterns := []string{
			"soAvailability[a-z0-9]+ErrorBudgetBurn",
		}
		isIgnored := false
		for _, ignoredAlertPattern := range ignoredAlertsPatterns {
			t.Logf("\tPattern ignored: %s", ignoredAlertPattern)
			matchFound, err := regexp.MatchString(ignoredAlertPattern, alertName)
			if err == nil && matchFound {
				t.Logf("\tFiring alert to be ignored: %s", alertName)
				isIgnored = true
				continue
			}
		}
		// end of tmp workaround - don't forget to remove the isIgnored reference below

		// add firing alerts
		if !isIgnored && alert.State == prometheusv1.AlertStateFiring {
			alertsError.alertsFiring = append(alertsError.alertsFiring, alertMetadata)
		}
	}

	if alertsError.isValid() {
		return alertsError

	}
	return nil
}

// Makes a api call a to get all pods in the rhmi namespaces
func podLogs(t TestingTB, ctx *TestingContext) {
	pods := &corev1.PodList{}

	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("failed to get the RHMI: %s", err)
	}
	podNamespaces := getPodNamespaces(rhmi.Spec.Type, ctx)

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

func getPodNamespaces(installType string, ctx *TestingContext) []string {
	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installType)) {
		return commonPodNamespaces
	} else {
		return append(commonPodNamespaces, rhoamExpectedPodNamespaces...)
	}
}

// TestIntegreatlyAlertsFiring reports any firing or pending alerts
func TestIntegreatlyAlertsPendingOrFiring(t TestingTB, ctx *TestingContext) {

	isInProw, err := isInProw(ctx)
	if err != nil {
		t.Logf("error getting in_prow annotation: %s", err)
	}
	if isInProw {
		t.Skip("Skipping due to UIBBT alerts firing in Prow, see https://issues.redhat.com/browse/MGDAPI-5193")
	}

	var lastError error

	// retry the tests every minute for up to 15 minutes
	monitoringTimeout := 15 * time.Minute
	monitoringRetryInterval := 1 * time.Minute
	err = wait.PollUntilContextTimeout(context.TODO(), monitoringRetryInterval, monitoringTimeout, false, func(ctx2 context.Context) (done bool, err error) {
		if newErr := getFiringOrPendingAlerts(t, ctx); newErr != nil {
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

func getFiringOrPendingAlerts(t TestingTB, ctx *TestingContext) error {
	output, err := execToPod("curl -s localhost:9090/api/v1/alerts",
		ObservabilityPrometheusPodName,
		ObservabilityProductNamespace,
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
		// ignore firing and pending  dms and missingSmtp alerts
		if alertName == deadMansSwitch ||
			(alertName == missingSmtpSecret &&
				alert.State == prometheusv1.AlertStateFiring &&
				smtpMissing(context.TODO(), ctx.Client, t)) ||
			(alertName == dmsSecretAlertName &&
				alert.State == prometheusv1.AlertStateFiring &&
				dmsSecretMissing(context.TODO(), ctx.Client, t)) {
			continue
		}

		// tmp workaround for **ErrorBudgetBurn alerts firing during installation (https://issues.redhat.com/browse/MGDAPI-5992)
		ignoredAlertsPatterns := []string{
			"soAvailability[a-z0-9]+ErrorBudgetBurn",
		}
		isIgnored := false
		for _, ignoredAlertPattern := range ignoredAlertsPatterns {
			t.Logf("\tPattern ignored: %s", ignoredAlertPattern)
			matchFound, err := regexp.MatchString(ignoredAlertPattern, alertName)
			if err == nil && matchFound {
				t.Logf("\tFiring/Pending alert to be ignored: %s", alertName)
				isIgnored = true
				continue
			}
		}
		// end of tmp workaround - don't forget to remove the two isIgnored references below

		// add firing alerts
		if !isIgnored && alert.State == prometheusv1.AlertStateFiring {
			alertsError.alertsFiring = append(alertsError.alertsFiring, alertMetadata)
		}
		//add pending alerts
		if !isIgnored && alert.State == prometheusv1.AlertStatePending {
			alertsError.alertsPending = append(alertsError.alertsPending, alertMetadata)
		}
	}
	if alertsError.isValid() {
		return alertsError

	}
	return nil
}
