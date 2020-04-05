package common

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
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
			"CronJobExists_redhat-rhmi-amq-online_enmasse-postgres-backup",
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
			"AMQOnlineContainerHighMemory",
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
			"ThreeScaleContainerHighMemory",
			"ThreeScaleContainerHighCPU",
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
	{
		File:  "redhat-rhmi-amq-online-enmasse-console-rules.yaml",
		Rules: []string{},
	},
}

var expectedAWSRules = []alertsTestRule{
	{
		File: RHMIOperatorNamespace + "-connectivity-rule-threescale-redis-" + InstallationName + ".yaml",
		Rules: []string{
			"3scaleRedisCacheConnectionFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-connectivity-rule-threescale-backend-redis-" + InstallationName + ".yaml",
		Rules: []string{
			"3scaleRedisCacheConnectionFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-connectivity-rule-threescale-postgres-" + InstallationName + ".yaml",
		Rules: []string{
			"3scalePostgresConnectionFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-availability-rule-threescale-redis-" + InstallationName + ".yaml",
		Rules: []string{
			"3scaleRedisCacheUnavailable",
		},
	},
	{
		File: RHMIOperatorNamespace + "-availability-rule-threescale-backend-redis-" + InstallationName + ".yaml",
		Rules: []string{
			"3scaleRedisCacheUnavailable",
		},
	},
	{
		File: RHMIOperatorNamespace + "-availability-rule-threescale-postgres-" + InstallationName + ".yaml",
		Rules: []string{
			"3scalePostgresInstanceUnavailable",
		},
	},
	{
		File: RHMIOperatorNamespace + "-connectivity-rule-ups-postgres-" + InstallationName + ".yaml",
		Rules: []string{
			"upsPostgresConnectionFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-availability-rule-ups-postgres-" + InstallationName + ".yaml",
		Rules: []string{
			"upsPostgresInstanceUnavailable",
		},
	},
	{
		File: RHMIOperatorNamespace + "-availability-rule-codeready-postgres-" + InstallationName + ".yaml",
		Rules: []string{
			"codeready-workspacesPostgresInstanceUnavailable",
		},
	},
	{
		File: RHMIOperatorNamespace + "-connectivity-rule-codeready-postgres-" + InstallationName + ".yaml",
		Rules: []string{
			"codeready-workspacesPostgresConnectionFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-connectivity-rule-rhssouser-postgres-" + InstallationName + ".yaml",
		Rules: []string{
			"user-ssoPostgresConnectionFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-availability-rule-rhssouser-postgres-" + InstallationName + ".yaml",
		Rules: []string{
			"user-ssoPostgresInstanceUnavailable",
		},
	},
	{
		File: RHMIOperatorNamespace + "-connectivity-rule-rhsso-postgres-" + InstallationName + ".yaml",
		Rules: []string{
			"rhssoPostgresConnectionFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-availability-rule-rhsso-postgres-" + InstallationName + ".yaml",
		Rules: []string{
			"rhssoPostgresInstanceUnavailable",
		},
	},
}

func TestIntegreatlyAlertsExist(t *testing.T, ctx *TestingContext) {
	isClusterStorage, err := isClusterStorage(ctx)
	if err != nil {
		t.Fatal("error getting isClusterStorage:", err)
	}

	// add external database alerts to list of expected rules if
	// cluster storage is not being used
	if !isClusterStorage {
		for _, rule := range expectedAWSRules {
			expectedRules = append(expectedRules, rule)
		}
	}

	// exec into the prometheus pod
	output, err := execToPod("curl localhost:9090/api/v1/rules",
		"prometheus-application-monitoring-0",
		MonitoringOperatorNamespace,
		"prometheus", ctx)
	if err != nil {
		t.Fatal("failed to exec to pod:", err)
	}

	// get all found rules from the prometheus api
	var promApiCallOutput prometheusAPIResponse
	err = json.Unmarshal([]byte(output), &promApiCallOutput)
	if err != nil {
		t.Fatal("failed to unmarshal json:", err)
	}
	var rulesResult prometheusv1.RulesResult
	err = json.Unmarshal([]byte(promApiCallOutput.Data), &rulesResult)
	if err != nil {
		t.Fatal("failed to unmarshal json:", err)
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
				fmt.Print("got a recording rule")
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
