package common

import (
	"bytes"
	goctx "context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"

	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/remotecommand"
)

type alertsTestRule struct {
	File  string   `json:"file"`
	Rules []string `json:"rules"`
}

type alertsTestReport struct {
	MissingRules    []string             `json:"missing"`
	AdditionalRules []string             `json:"additional"`
	Status          alertsTestFileStatus `json:"status"`
}

func newDefaultReport(status alertsTestFileStatus) *alertsTestReport {
	return &alertsTestReport{
		MissingRules:    []string{},
		AdditionalRules: []string{},
		Status:          status,
	}
}

type alertsTestFileStatus string

var (
	fileMissing    alertsTestFileStatus = "File expected but not found"
	fileAdditional alertsTestFileStatus = "File found but not expected"
	fileExists     alertsTestFileStatus = "File found with missing or unexpected rules"
	fileCorrect    alertsTestFileStatus = "File found with all alerts present"
)

var expectedRules = []alertsTestRule{
	{
		File: "redhat-rhmi-middleware-monitoring-operator-backup-monitoring-alerts.yaml",
		Rules: []string{
			"JobRunningTimeExceeded",
			"JobRunningTimeExceeded",
			"CronJobSuspended",
			"CronJobsFailed",
			"CronJobNotRunInThreshold",
		},
	},
	{
		File: "redhat-rhmi-amq-online-backupjobs-exist-alerts.yaml",
		Rules: []string{
			"CronJobExists_redhat-rhmi-amq-online_enmasse-pv-backup",
		},
	},
	{
		File: "redhat-rhmi-codeready-workspaces-backupjobs-exist-alerts.yaml",
		Rules: []string{
			"CronJobExists_redhat-rhmi-codeready-workspaces_codeready-pv-backup",
		},
	},
	{
		File: "redhat-rhmi-rhsso-keycloak.yaml",
		Rules: []string{
			"KeycloakJavaHeapThresholdExceeded",
			"KeycloakJavaNonHeapThresholdExceeded",
			"KeycloakJavaGCTimePerMinuteScavenge",
			"KeycloakJavaGCTimePerMinuteMarkSweep",
			"KeycloakJavaDeadlockedThreads",
			"KeycloakLoginFailedThresholdExceeded",
			"KeycloakInstanceNotAvailable",
			"KeycloakAPIRequestDuration90PercThresholdExceeded",
			"KeycloakAPIRequestDuration99.5PercThresholdExceeded",
		},
	},
	{
		File: "redhat-rhmi-user-sso-keycloak.yaml",
		Rules: []string{
			"KeycloakJavaHeapThresholdExceeded",
			"KeycloakJavaNonHeapThresholdExceeded",
			"KeycloakJavaGCTimePerMinuteScavenge",
			"KeycloakJavaGCTimePerMinuteMarkSweep",
			"KeycloakJavaDeadlockedThreads",
			"KeycloakLoginFailedThresholdExceeded",
			"KeycloakInstanceNotAvailable",
			"KeycloakAPIRequestDuration90PercThresholdExceeded",
			"KeycloakAPIRequestDuration99.5PercThresholdExceeded",
		},
	},
	{
		File: "redhat-rhmi-middleware-monitoring-operator-ksm-alerts.yaml",
		Rules: []string{
			"KubePodCrashLooping",
			"KubePodNotReady",
			"KubePodImagePullBackOff",
			"KubePodBadConfig",
			"KubePodStuckCreating",
			"ClusterSchedulableMemoryLow",
			"ClusterSchedulableCPULow",
			"PVCStorageAvailable",
			"PVCStorageMetricsAvailable",
			"PVCStorageWillFillIn4Days",
			"PVCStorageWillFillIn4Hours",
			"PersistentVolumeErrors",
		},
	},
	{
		File: "redhat-rhmi-amq-online-ksm-amqonline-alerts.yaml",
		Rules: []string{
			"AMQOnlinePodCount",
			"AMQOnlinePodHighMemory",
		},
	},
	{
		File: "redhat-rhmi-apicurito-ksm-apicurito-alerts.yaml",
		Rules: []string{
			"ApicuritoPodCount",
		},
	},
	{
		File: "redhat-rhmi-fuse-ksm-fuse-online-alerts.yaml",
		Rules: []string{
			"FuseOnlineSyndesisServerInstanceDown",
			"FuseOnlineSyndesisUIInstanceDown",
			"FuseOnlinePodCount",
		},
	},
	{
		File: "redhat-rhmi-middleware-monitoring-operator-ksm-monitoring-alerts.yaml",
		Rules: []string{
			"MiddlewareMonitoringPodCount",
		},
	},
	{
		File: "redhat-rhmi-rhsso-ksm-rhsso-alerts.yaml",
		Rules: []string{
			"RHSSOPodHighMemory",
		},
	},
	{
		File: "redhat-rhmi-codeready-workspaces-ksm-codeready-alerts.yaml",
		Rules: []string{
			"CodeReadyPodCount",
		},
	},
	{
		File: "redhat-rhmi-3scale-ksm-3scale-alerts.yaml",
		Rules: []string{
			"ThreeScaleApicastStagingPod",
			"ThreeScaleApicastProductionPod",
			"ThreeScaleBackendWorkerPod",
			"ThreeScaleBackendListenerPod",
			"ThreeScaleSystemAppPod",
			"ThreeScaleAdminUIBBT",
			"ThreeScaleDeveloperUIBBT",
			"ThreeScaleSystemAdminUIBBT",
			"ThreeScalePodHighMemory",
			"ThreeScalePodHighCPU",
			"ThreeScaleZyncPodAvailability",
			"ThreeScaleZyncDatabasePodAvailability",
		},
	},
	{
		File: "redhat-rhmi-middleware-monitoring-operator-prometheus-application-monitoring-rules.yaml",
		Rules: []string{
			"DeadMansSwitch",
		},
	},
	{
		File: "redhat-rhmi-amq-online-rhmi-amq-online-slo.yaml",
		Rules: []string{
			"AMQOnlineConsoleAvailable",
			"AMQOnlineKeycloakAvailable",
			"AMQOnlineOperatorAvailable",
		},
	},
	{
		File: "redhat-rhmi-solution-explorer-ksm-solution-explorer-alerts.yaml",
		Rules: []string{
			"SolutionExplorerPodCount",
		},
	},
	{
		File: "redhat-rhmi-fuse-syndesis-infra-db-alerting-rules.yaml",
		Rules: []string{
			"FuseOnlineDatabaseInstanceDown",
			"FuseOnlinePostgresExporterDown",
		},
	},
	{
		File: "redhat-rhmi-fuse-syndesis-infra-meta-alerting-rules.yaml",
		Rules: []string{
			"FuseOnlineRestApiHighEndpointErrorRate",
			"FuseOnlineRestApiHighEndpointLatency",
		},
	},
	{
		File: "redhat-rhmi-fuse-syndesis-infra-server-alerting-rules.yaml",
		Rules: []string{
			"FuseOnlineRestApiHighEndpointErrorRate",
			"FuseOnlineRestApiHighEndpointLatency",
		},
	},
	{
		File: "redhat-rhmi-fuse-syndesis-integrations-alerting-rules.yaml",
		Rules: []string{
			"IntegrationExchangesHighFailureRate",
		},
	},
	{
		File: "redhat-rhmi-ups-unifiedpush.yaml",
		Rules: []string{
			"UnifiedPushDown",
			"UnifiedPushConsoleDown",
			"UnifiedPushJavaHeapThresholdExceeded",
			"UnifiedPushJavaNonHeapThresholdExceeded",
			"UnifiedPushJavaGCTimePerMinuteScavenge",
			"UnifiedPushJavaDeadlockedThreads",
			"UnifiedPushMessagesFailures",
		},
	},
	{
		File: "redhat-rhmi-ups-operator-unifiedpush-operator.yaml",
		Rules: []string{
			"UnifiedPushOperatorDown",
		},
	},
}

var expectedAWSRules = []alertsTestRule{
	{
		File: "redhat-rhmi-operator-connectivity-rule-threescale-redis-example-rhmi.yaml",
		Rules: []string{
			"3scaleRedisCacheConnectionFailed",
		},
	},
	{
		File: "redhat-rhmi-operator-connectivity-rule-threescale-backend-redis-example-rhmi.yaml",
		Rules: []string{
			"3scaleRedisCacheConnectionFailed",
		},
	},
	{
		File: "redhat-rhmi-operator-connectivity-rule-threescale-postgres-example-rhmi.yaml",
		Rules: []string{
			"3scalePostgresConnectionFailed",
		},
	},
	{
		File: "redhat-rhmi-operator-availability-rule-threescale-redis-example-rhmi.yaml",
		Rules: []string{
			"3scaleRedisCacheUnavailable",
		},
	},
	{
		File: "redhat-rhmi-operator-availability-rule-threescale-backend-redis-example-rhmi.yaml",
		Rules: []string{
			"3scaleRedisCacheUnavailable",
		},
	},
	{
		File: "redhat-rhmi-operator-availability-rule-threescale-postgres-example-rhmi.yaml",
		Rules: []string{
			"3scalePostgresInstanceUnavailable",
		},
	},
	{
		File: "redhat-rhmi-operator-connectivity-rule-ups-postgres-example-rhmi.yaml",
		Rules: []string{
			"upsPostgresConnectionFailed",
		},
	},
	{
		File: "redhat-rhmi-operator-availability-rule-ups-postgres-example-rhmi.yaml",
		Rules: []string{
			"upsPostgresInstanceUnavailable",
		},
	},
	{
		File: "redhat-rhmi-operator-availability-rule-codeready-postgres-example-rhmi.yaml",
		Rules: []string{
			"codeready-workspacesPostgresInstanceUnavailable",
		},
	},
	{
		File: "redhat-rhmi-operator-connectivity-rule-codeready-postgres-example-rhmi.yaml",
		Rules: []string{
			"codeready-workspacesPostgresConnectionFailed",
		},
	},
	{
		File: "redhat-rhmi-operator-connectivity-rule-rhssouser-postgres-example-rhmi.yaml",
		Rules: []string{
			"user-ssoPostgresConnectionFailed",
		},
	},
	{
		File: "redhat-rhmi-operator-availability-rule-rhssouser-postgres-example-rhmi.yaml",
		Rules: []string{
			"user-ssoPostgresInstanceUnavailable",
		},
	},
	{
		File: "redhat-rhmi-operator-connectivity-rule-rhsso-postgres-example-rhmi.yaml",
		Rules: []string{
			"rhssoPostgresConnectionFailed",
		},
	},
	{
		File: "redhat-rhmi-operator-availability-rule-rhsso-postgres-example-rhmi.yaml",
		Rules: []string{
			"rhssoPostgresInstanceUnavailable",
		},
	},
}

func TestIntegreatlyAlertsExist(t *testing.T, ctx *TestingContext) {
	// get the RHMI custom resource to check what storage type is being used
	rhmi := &v1alpha1.RHMI{}
	ns := fmt.Sprintf("%soperator", namespacePrefix)
	err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: installationName, Namespace: ns}, rhmi)
	if err != nil {
		t.Fatal("error getting RHMI CR:", err)
	}

	// add external database alerts to list of expected rules if
	// cluster storage is not being used
	if rhmi.Spec.UseClusterStorage != "true" {
		for _, rule := range expectedAWSRules {
			expectedRules = append(expectedRules, rule)
		}
	}

	// exec into the prometheus pod
	output, err := execToPod("curl localhost:9090/api/v1/rules",
		"prometheus-application-monitoring-0",
		namespacePrefix+"middleware-monitoring-operator",
		"prometheus", ctx)
	if err != nil {
		t.Fatal("failed to exec to pod:", err)
	}

	// get all found rules from the prometheus api
	var promApiCallOutput prometheusAPIResponse
	err = json.Unmarshal([]byte(output), &promApiCallOutput)
	if err != nil {
		t.Fatal("Failed to unmarshal json:", err)
	}
	var rulesResult prometheusv1.RulesResult
	err = json.Unmarshal([]byte(promApiCallOutput.Data), &rulesResult)
	if err != nil {
		t.Fatal("Failed to unmarshal json:", err)
	}

	// convert prometheus rule to PrometheusRule type
	var actualRules []alertsTestRule
	for _, group := range rulesResult.Groups {
		ruleName := strings.Split(group.File, "/")
		rule := alertsTestRule{
			File: ruleName[len(ruleName)-1],
		}
		for _, promRule := range group.Rules {
			switch v := promRule.(type) {
			case prometheusv1.RecordingRule:
				recRule := promRule.(prometheusv1.RecordingRule)
				rule.Rules = append(rule.Rules, recRule.Name)
			case prometheusv1.AlertingRule:
				alertRule := promRule.(prometheusv1.AlertingRule)
				rule.Rules = append(rule.Rules, alertRule.Name)
			default:
				fmt.Printf("unknown rule type %s", v)
			}
		}
		actualRules = append(actualRules, rule)
	}

	// build up a reportMapping of missing or unexpected files
	reportMapping := make(map[string]*alertsTestReport, 0)

	// unexpected/additional
	// if an unexpected file is found, add it to the reportMapping
	ruleDiff := ruleDifference(actualRules, expectedRules)
	for _, rule := range ruleDiff {
		reportMapping[rule.File] = &alertsTestReport{
			AdditionalRules: rule.Rules,
			Status:          fileAdditional,
		}
	}

	// missing file
	// if an expected file is not found, add it to the reportMapping
	ruleDiff = ruleDifference(expectedRules, actualRules)
	for _, rule := range ruleDiff {
		reportMapping[rule.File] = &alertsTestReport{
			MissingRules: rule.Rules,
			Status:       fileMissing,
		}
	}

	// the file exists, do left and right diffs to ensure
	// all rules exist and no unexpected rules are found
	for _, actualRule := range actualRules {
		for _, expectedRule := range expectedRules {
			if actualRule.File == expectedRule.File {
				reportMapping[actualRule.File] = buildReport(actualRule, expectedRule, reportMapping[actualRule.File])
			}
		}
	}

	// report the status
	missingCount := 0
	extraCount := 0
	for k, v := range reportMapping {
		if v.Status != fileCorrect {
			fmt.Println("\nFile Name:", k)
			fmt.Println("Missing Rules:", v.MissingRules)
			fmt.Println("Unexpected Rules:", v.AdditionalRules)
			fmt.Println("Status:", v.Status)
		}
		if v.Status == fileMissing || len(v.MissingRules) > 0 {
			missingCount++
		}
		if v.Status == fileAdditional || len(v.AdditionalRules) > 0 {
			extraCount++
		}
	}

	if missingCount > 0 {
		fmt.Println("\nMissing alerts were found from Prometheus. If the removal of these Alert rules was intentional, please update this test to remove them from the check. If the removal of these Alert rules was not intendended or you are not sure, please create a Jira & discuss with the monitoring team on how best to proceed")
	}
	if extraCount > 0 {
		fmt.Println("\nUnexpected alerts were found in Prometheus. If these Alert rules were intentionally added, please update this test to add them to the check. If these Alert rules were not added intentionally or you are not sure, please create a Jira & discuss with the monitoring team on how best to proceed.")
	}
	if extraCount > 0 || missingCount > 0 {
		t.Fatal("Found missing or too many alerts")
	}
}

func execToPod(command string, podname string, namespace string, container string, ctx *TestingContext) (string, error) {
	req := ctx.KubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podname).
		Namespace(namespace).
		SubResource("exec").
		Param("container", container)
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return "", fmt.Errorf("error adding to scheme: %v", err)
	}
	parameterCodec := runtime.NewParameterCodec(scheme)
	req.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command:   strings.Fields(command),
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, parameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(ctx.KubeConfig, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("error while creating Executor: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return "", fmt.Errorf("error in Stream: %v", err)
	}

	return stdout.String(), nil
}

// difference one-way diff that return strings in sliceSource that are not in sliceTarget
func difference(sliceSource, sliceTarget []string) []string {
	// create an empty lookup map with keys from sliceTarget
	diffSourceLookupMap := make(map[string]struct{}, len(sliceTarget))
	for _, item := range sliceTarget {
		diffSourceLookupMap[item] = struct{}{}
	}
	// use the lookup map to find items in sliceSource that are not in sliceTarget
	// and store them in a diff slice
	var diff []string
	for _, item := range sliceSource {
		if _, found := diffSourceLookupMap[item]; !found {
			diff = append(diff, item)
		}
	}
	return diff
}

// ruleDifference one-way diff that return rules in diffSource that are not in diffTarget
func ruleDifference(diffSource, diffTarget []alertsTestRule) []alertsTestRule {
	// create an empty lookup map with keys from diffTarget
	diffSourceLookupMap := make(map[string]struct{}, len(diffTarget))
	for _, rule := range diffTarget {
		diffSourceLookupMap[rule.File] = struct{}{}
	}
	// use the lookup map to find items in diffSource that are not in diffTarget
	// and store them in a diff slice
	var diff []alertsTestRule
	for _, rule := range diffSource {
		if _, found := diffSourceLookupMap[rule.File]; !found {
			diff = append(diff, rule)
		}
	}
	return diff
}

// build report builds up a report of missing or unexpected rules for a given file name
func buildReport(actualRule, expectedRule alertsTestRule, report *alertsTestReport) *alertsTestReport {
	// pre-req
	if report == nil {
		report = newDefaultReport(fileCorrect)
	}
	// build report
	report.MissingRules = append(report.MissingRules, difference(expectedRule.Rules, actualRule.Rules)...)
	report.AdditionalRules = append(report.AdditionalRules, difference(actualRule.Rules, expectedRule.Rules)...)
	if len(report.MissingRules) != 0 || len(report.AdditionalRules) != 0 {
		report.Status = fileExists
	}
	return report
}
