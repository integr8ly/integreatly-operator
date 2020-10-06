package common

import (
	"encoding/json"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
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

// Specific to RHMI2
var rhmi2ExpectedRules = []alertsTestRule{
	{
		File: NamespacePrefix + "amq-online-backupjobs-exist-alerts.yaml",
		Rules: []string{
			"CronJobExists_" + NamespacePrefix + "amq-online_enmasse-pv-backup",
			"CronJobExists_" + NamespacePrefix + "amq-online_enmasse-postgres-backup",
			"CronJobExists_" + NamespacePrefix + "amq-online_resources-backup",
		},
	},
	{
		File: NamespacePrefix + "codeready-workspaces-backupjobs-exist-alerts.yaml",
		Rules: []string{
			"CronJobExists_" + NamespacePrefix + "codeready-workspaces_codeready-pv-backup",
		},
	},
	{
		File: NamespacePrefix + "amq-online-ksm-amqonline-alerts.yaml",
		Rules: []string{
			"AMQOnlinePodCount",
			"AMQOnlineContainerHighMemory",
		},
	},
	{
		File: NamespacePrefix + "apicurito-ksm-apicurito-alerts.yaml",
		Rules: []string{
			"ApicuritoPodCount",
		},
	},
	{
		File: NamespacePrefix + "fuse-ksm-fuse-online-alerts.yaml",
		Rules: []string{
			"FuseOnlineSyndesisServerInstanceDown",
			"FuseOnlineSyndesisUIInstanceDown",
		},
	},
	{
		File: NamespacePrefix + "codeready-workspaces-ksm-codeready-alerts.yaml",
		Rules: []string{
			"CodeReadyPodCount",
		},
	},
	{
		File: NamespacePrefix + "amq-online-rhmi-amq-online-slo.yaml",
		Rules: []string{
			"AMQOnlineConsoleAvailable",
			"AMQOnlineKeycloakAvailable",
			"AMQOnlineOperatorAvailable",
		},
	},
	{
		File: NamespacePrefix + "solution-explorer-ksm-solution-explorer-alerts.yaml",
		Rules: []string{
			"SolutionExplorerPodCount",
		},
	},
	{
		File: NamespacePrefix + "fuse-syndesis-infra-meta-alerting-rules.yaml",
		Rules: []string{
			"FuseOnlineRestApiHighEndpointErrorRate",
			"FuseOnlineRestApiHighEndpointLatency",
		},
	},
	{
		File: NamespacePrefix + "fuse-syndesis-infra-server-alerting-rules.yaml",
		Rules: []string{
			"FuseOnlineRestApiHighEndpointErrorRate",
			"FuseOnlineRestApiHighEndpointLatency",
		},
	},
	{
		File: NamespacePrefix + "fuse-syndesis-integrations-alerting-rules.yaml",
		Rules: []string{
			"IntegrationExchangesHighFailureRate",
		},
	},
	{
		File: NamespacePrefix + "ups-unifiedpush.yaml",
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
		File: NamespacePrefix + "ups-operator-unifiedpush-operator.yaml",
		Rules: []string{
			"UnifiedPushOperatorDown",
		},
	},
	{
		File: NamespacePrefix + "amq-online-kube-metrics.yaml",
		Rules: []string{
			"TerminatingPods",
			"RestartingPods",
			"RestartingPods",
			"PendingPods",
		},
	},
	{
		File: NamespacePrefix + "amq-online-enmasse.yaml",
		Rules: []string{
			"ComponentHealth",
			"AuthenticationService",
			"AddressSpaceHealth",
			"AddressHealth",
			"RouterMeshConnectivityHealth",
			"RouterMeshUndeliveredHealth",
			"BrokerMemory",
		},
	},
	{
		File:  NamespacePrefix + "amq-online-enmasse-console-rules.yaml",
		Rules: []string{},
	},
	{
		File: NamespacePrefix + "apicurito-ksm-endpoint-alerts.yaml",
		Rules: []string{
			"RHMIApicuritoServiceEndpointDown",
			"RHMIApicuritoFuseApicuritoGeneratorServiceEndpointDown",
		},
	},
	{
		File: NamespacePrefix + "amq-online-ksm-endpoint-alerts.yaml",
		Rules: []string{
			"RHMIAMQOnlineNoneAuthServiceEndpointDown",
			"RHMIAMQOnlineAddressSpaceControllerServiceEndpointDown",
			"RHMIAMQOnlineConsoleServiceEndpointDown",
			"RHMIAMQOnlineRegistryCsServiceEndpointDown",
			"RHMIAMQOnlineStandardAuthServiceEndpointDown",
			"RHMIAMQOnlineEnmasseOperatorMetricsServiceEndpointDown",
		},
	},
	{
		File: NamespacePrefix + "fuse-operator-ksm-endpoint-alerts.yaml",
		Rules: []string{
			"RHMIFuseOnlineOperatorRhmiRegistryCsServiceEndpointDown",
			"RHMIFuseOnlineOperatorSyndesisOperatorMetricsServiceEndpointDown",
		},
	},
	{
		File: NamespacePrefix + "ups-operator-ksm-endpoint-alerts.yaml",
		Rules: []string{
			"RHMIUPSOperatorRhmiRegistryCsServiceEndpointDown",
			"RHMIUPSOperatorUnifiedPushOperatorMetricsServiceEndpointDown",
		},
	},
	{
		File: NamespacePrefix + "codeready-workspaces-ksm-endpoint-alerts.yaml",
		Rules: []string{
			"RHMICodeReadyCheHostServiceEndpointDown",
			"RHMICodeReadyDevfileRegistryServiceEndpointDown",
			"RHMICodeReadyPluginRegistryServiceEndpointDown",
		},
	},
	{
		File: NamespacePrefix + "apicurito-operator-ksm-endpoint-alerts.yaml",
		Rules: []string{
			"RHMIApicuritoOperatorRhmiRegistryCsServiceEndpointDown",
		},
	},
	{
		File: NamespacePrefix + "solution-explorer-ksm-endpoint-alerts.yaml",
		Rules: []string{
			"RHMISolutionExplorerTutorialWebAppServiceEndpointDown",
		},
	},
	{
		File: NamespacePrefix + "solution-explorer-operator-ksm-endpoint-alerts.yaml",
		Rules: []string{
			"RHMISolutionExplorerOperatorRhmiRegistryCsServiceEndpointDown",
		},
	},
	{
		File: NamespacePrefix + "codeready-workspaces-operator-ksm-endpoint-alerts.yaml",
		Rules: []string{
			"RHMICodeReadyOperatorRhmiRegistryCsServiceEndpointDown",
		},
	},
	{
		File: NamespacePrefix + "fuse-ksm-endpoint-alerts.yaml",
		Rules: []string{
			"RHMIFuseOnlineBrokerAmqTcpServiceEndpointDown",
			"RHMIFuseOnlineSyndesisMetaServiceEndpointDown",
			"RHMIFuseOnlineSyndesisOauthproxyServiceEndpointDown",
			"RHMIFuseOnlineSyndesisPrometheusServiceEndpointDown",
			"RHMIFuseOnlineSyndesisServerServiceEndpointDown",
			"RHMIFuseOnlineSyndesisUiServiceEndpointDown",
		},
	},
	{
		File: NamespacePrefix + "ups-ksm-endpoint-alerts.yaml",
		Rules: []string{
			"RHMIUPSUnifiedPushServiceEndpointDown",
			"RHMIUPSUnifiedpushProxyServiceEndpointDown",
		},
	},
}

// Common to all install types
var commonExpectedRules = []alertsTestRule{
	{
		File: NamespacePrefix + "middleware-monitoring-operator-backup-monitoring-alerts.yaml",
		Rules: []string{
			"JobRunningTimeExceeded",
			"JobRunningTimeExceeded",
			"CronJobsFailed",
			"CronJobNotRunInThreshold",
		},
	},
	{
		File: NamespacePrefix + "rhsso-keycloak.yaml",
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
		File: NamespacePrefix + "user-sso-keycloak.yaml",
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
		File: NamespacePrefix + "middleware-monitoring-operator-ksm-alerts.yaml",
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
			"KubePersistentVolumeFillingUp",
			"KubePersistentVolumeFillingUp",
			"PersistentVolumeErrors",
		},
	},
	{
		File: NamespacePrefix + "middleware-monitoring-operator-ksm-monitoring-alerts.yaml",
		Rules: []string{
			"MiddlewareMonitoringPodCount",
		},
	},
	{
		File: NamespacePrefix + "3scale-ksm-3scale-alerts.yaml",
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
		File: NamespacePrefix + "middleware-monitoring-operator-prometheus-application-monitoring-rules.yaml",
		Rules: []string{
			"DeadMansSwitch",
		},
	},
	{
		File: NamespacePrefix + "3scale-ksm-endpoint-alerts.yaml",
		Rules: []string{
			"RHMIThreeScaleApicastProductionServiceEndpointDown",
			"RHMIThreeScaleApicastStagingServiceEndpointDown",
			"RHMIThreeScaleBackendListenerServiceEndpointDown",
			"RHMIThreeScaleSystemDeveloperServiceEndpointDown",
			"RHMIThreeScaleSystemMasterServiceEndpointDown",
			"RHMIThreeScaleSystemMemcacheServiceEndpointDown",
			"RHMIThreeScaleSystemProviderServiceEndpointDown",
			"RHMIThreeScaleSystemSphinxServiceEndpointDown",
			"RHMIThreeScaleZyncDatabaseServiceEndpointDown",
			"RHMIThreeScaleZyncServiceEndpointDown",
		},
	},
	{
		File: NamespacePrefix + "user-sso-ksm-endpoint-alerts.yaml",
		Rules: []string{
			"RHMIUserRhssoKeycloakServiceEndpointDown",
			"RHMIUserRhssoKeycloakDiscoveryServiceEndpointDown",
		},
	},
	{
		File: NamespacePrefix + "cloud-resources-operator-ksm-endpoint-alerts.yaml",
		Rules: []string{
			"RHMICloudResourceOperatorMetricsServiceEndpointDown",
			"RHMICloudResourceOperatorRhmiRegistryCsServiceEndpointDown",
		},
	},
	{
		File: NamespacePrefix + "middleware-monitoring-operator-ksm-endpoint-alerts.yaml",
		Rules: []string{
			"RHMIMiddlewareMonitoringOperatorAlertmanagerOperatedServiceEndpointDown",
			"RHMIMiddlewareMonitoringOperatorAlertmanagerServiceEndpointDown",
			"RHMIMiddlewareMonitoringOperatorApplicationMonitoringMetricsServiceEndpointDown",
			"RHMIMiddlewareMonitoringOperatorGrafanaServiceEndpointDown",
			"RHMIMiddlewareMonitoringOperatorPrometheusOperatedServiceEndpointDown",
			"RHMIMiddlewareMonitoringOperatorPrometheusServiceEndpointDown",
			"RHMIMiddlewareMonitoringOperatorRhmiRegistryCsServiceEndpointDown",
		},
	},
	{
		File: NamespacePrefix + "rhsso-ksm-endpoint-alerts.yaml",
		Rules: []string{
			"RHMIRhssoKeycloakServiceEndpointDown",
			"RHMIRhssoKeycloakDiscoveryServiceEndpointDown",
		},
	},
	{
		File: NamespacePrefix + "rhsso-operator-ksm-endpoint-alerts.yaml",
		Rules: []string{
			"RHMIRhssoKeycloakOperatorRhmiRegistryCsServiceEndpointDown",
			"RHMIRhssoKeycloakOperatorMetricsServiceEndpointDown",
		},
	},
	{
		File: NamespacePrefix + "3scale-operator-ksm-endpoint-alerts.yaml",
		Rules: []string{
			"RHMIThreeScaleOperatorRhmiRegistryCsServiceEndpointDown",
			"RHMIThreeScaleOperatorServiceEndpointDown",
		},
	},
	{
		File: NamespacePrefix + "user-sso-operator-ksm-endpoint-alerts.yaml",
		Rules: []string{
			"RHMIUserRhssoOperatorRhmiRegistryCsMetricsServiceEndpointDown",
			"RHMIUserRhssoKeycloakOperatorMetricsServiceEndpointDown",
		},
	},
	{
		File: NamespacePrefix + "operator-rhmi-installation-controller-alerts.yaml",
		Rules: []string{
			"RHMIInstallationControllerIsNotReconciling",
			"RHMIInstallationControllerStoppedReconciling",
		},
	},
	{
		File: MonitoringOperatorNamespace + "-install-upgrade-alerts.yaml",
		Rules: []string{
			"RHMICSVRequirementsNotMet",
		},
	},
	{
		File: NamespacePrefix + "operator-sendgrid-smtp-secret-exists-rule.yaml",
		Rules: []string{
			"SendgridSmtpSecretExists",
		},
	},
}

// common aws rules applicable to all install types
var commonExpectedAWSRules = []alertsTestRule{
	{
		File: RHMIOperatorNamespace + "-connectivity-rule-threescale-redis-rhmi.yaml",
		Rules: []string{
			"Threescale-Redis-RhmiRedisCacheConnectionFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-connectivity-rule-threescale-backend-redis-rhmi.yaml",
		Rules: []string{
			"Threescale-Backend-Redis-RhmiRedisCacheConnectionFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-connectivity-rule-threescale-postgres-rhmi.yaml",
		Rules: []string{
			"Threescale-Postgres-RhmiPostgresConnectionFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-availability-rule-threescale-redis-rhmi.yaml",
		Rules: []string{
			"Threescale-Redis-RhmiRedisCacheUnavailable",
		},
	},
	{
		File: RHMIOperatorNamespace + "-availability-rule-threescale-backend-redis-rhmi.yaml",
		Rules: []string{
			"Threescale-Backend-Redis-RhmiRedisCacheUnavailable",
		},
	},
	{
		File: RHMIOperatorNamespace + "-resource-deletion-status-phase-failed-rule-threescale-redis-rhmi.yaml",
		Rules: []string{
			"Threescale-Redis-RhmiRedisResourceDeletionStatusPhaseFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-resource-deletion-status-phase-failed-rule-threescale-backend-redis-rhmi.yaml",
		Rules: []string{
			"Threescale-Backend-Redis-RhmiRedisResourceDeletionStatusPhaseFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-resource-deletion-status-phase-failed-rule-threescale-postgres-rhmi.yaml",
		Rules: []string{
			"Threescale-Postgres-RhmiPostgresResourceDeletionStatusPhaseFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-resource-deletion-status-phase-failed-rule-rhsso-postgres-rhmi.yaml",
		Rules: []string{
			"Rhsso-Postgres-RhmiPostgresResourceDeletionStatusPhaseFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-resource-deletion-status-phase-failed-rule-rhssouser-postgres-rhmi.yaml",
		Rules: []string{
			"Rhssouser-Postgres-RhmiPostgresResourceDeletionStatusPhaseFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-availability-rule-threescale-postgres-rhmi.yaml",
		Rules: []string{
			"Threescale-Postgres-RhmiPostgresInstanceUnavailable",
		},
	},
	{
		File: RHMIOperatorNamespace + "-connectivity-rule-rhssouser-postgres-rhmi.yaml",
		Rules: []string{
			"Rhssouser-Postgres-RhmiPostgresConnectionFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-availability-rule-rhssouser-postgres-rhmi.yaml",
		Rules: []string{
			"Rhssouser-Postgres-RhmiPostgresInstanceUnavailable",
		},
	},
	{
		File: RHMIOperatorNamespace + "-connectivity-rule-rhsso-postgres-rhmi.yaml",
		Rules: []string{
			"Rhsso-Postgres-RhmiPostgresConnectionFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-availability-rule-rhsso-postgres-rhmi.yaml",
		Rules: []string{
			"Rhsso-Postgres-RhmiPostgresInstanceUnavailable",
		},
	},
	{
		File: NamespacePrefix + "operator-resource-status-phase-failed-rule-threescale-redis-rhmi.yaml",
		Rules: []string{
			"Threescale-Redis-RhmiRedisResourceStatusPhaseFailed",
		},
	},
	{
		File: NamespacePrefix + "operator-resource-status-phase-pending-rule-rhssouser-postgres-rhmi.yaml",
		Rules: []string{
			"Rhssouser-Postgres-RhmiPostgresResourceStatusPhasePending",
		},
	},
	{
		File: NamespacePrefix + "operator-resource-status-phase-failed-rule-rhsso-postgres-rhmi.yaml",
		Rules: []string{
			"Rhsso-Postgres-RhmiPostgresResourceStatusPhaseFailed",
		},
	},
	{
		File: NamespacePrefix + "operator-resource-status-phase-pending-rule-threescale-postgres-rhmi.yaml",
		Rules: []string{
			"Threescale-Postgres-RhmiPostgresResourceStatusPhasePending",
		},
	},

	{
		File: NamespacePrefix + "operator-resource-status-phase-pending-rule-threescale-redis-rhmi.yaml",
		Rules: []string{
			"Threescale-Redis-RhmiRedisResourceStatusPhasePending",
		},
	},
	{
		File: NamespacePrefix + "operator-resource-status-phase-pending-rule-rhsso-postgres-rhmi.yaml",
		Rules: []string{
			"Rhsso-Postgres-RhmiPostgresResourceStatusPhasePending",
		},
	},
	{
		File: NamespacePrefix + "operator-resource-status-phase-pending-rule-threescale-backend-redis-rhmi.yaml",
		Rules: []string{
			"Threescale-Backend-Redis-RhmiRedisResourceStatusPhasePending",
		},
	},
	{
		File: NamespacePrefix + "operator-resource-status-phase-failed-rule-rhssouser-postgres-rhmi.yaml",
		Rules: []string{
			"Rhssouser-Postgres-RhmiPostgresResourceStatusPhaseFailed",
		},
	},
	{
		File: NamespacePrefix + "operator-resource-status-phase-failed-rule-threescale-backend-redis-rhmi.yaml",
		Rules: []string{
			"Threescale-Backend-Redis-RhmiRedisResourceStatusPhaseFailed",
		},
	},
	{
		File: NamespacePrefix + "operator-resource-status-phase-failed-rule-threescale-postgres-rhmi.yaml",
		Rules: []string{
			"Threescale-Postgres-RhmiPostgresResourceStatusPhaseFailed",
		},
	},
	{
		File: NamespacePrefix + "operator-postgres-storage-will-fill-in-4-hours.yaml",
		Rules: []string{
			"PostgresStorageWillFillIn4Hours",
		},
	},
	{
		File: NamespacePrefix + "operator-postgres-storage-will-fill-in-4-days.yaml",
		Rules: []string{
			"PostgresStorageWillFillIn4Days",
		},
	},
	{
		File: NamespacePrefix + "operator-postgres-storage-low.yaml",
		Rules: []string{
			"PostgresStorageLow",
		},
	},
	{
		File: NamespacePrefix + "operator-postgres-cpu-high.yaml",
		Rules: []string{
			"PostgresCPUHigh",
		},
	},
	{
		File: NamespacePrefix + "operator-postgres-freeable-memory-low.yaml",
		Rules: []string{
			"PostgresFreeableMemoryLow",
		},
	},
	{
		File: NamespacePrefix + "operator-redis-memory-usage-high.yaml",
		Rules: []string{
			"RedisMemoryUsageHigh",
		},
	},
	{
		File: NamespacePrefix + "operator-redis-memory-usage-will-max-in-4-hours.yaml",
		Rules: []string{
			"RedisMemoryUsageMaxIn4Hours",
		},
	},
	{
		File: NamespacePrefix + "operator-redis-memory-usage-max-fill-in-4-days.yaml",
		Rules: []string{
			"RedisMemoryUsageMaxIn4Days",
		},
	},
	{
		File: NamespacePrefix + "operator-redis-cpu-usage-high.yaml",
		Rules: []string{
			"RedisCpuUsageHigh",
		},
	},
}

// rhmi2 aws rules
var rhmi2ExpectedAWSRules = []alertsTestRule{
	{
		File: RHMIOperatorNamespace + "-resource-deletion-status-phase-failed-rule-codeready-postgres-rhmi.yaml",
		Rules: []string{
			"Codeready-Postgres-RhmiPostgresResourceDeletionStatusPhaseFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-resource-deletion-status-phase-failed-rule-ups-postgres-rhmi.yaml",
		Rules: []string{
			"Ups-Postgres-RhmiPostgresResourceDeletionStatusPhaseFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-resource-deletion-status-phase-failed-rule-fuse-postgres-rhmi.yaml",
		Rules: []string{
			"Fuse-Postgres-RhmiPostgresResourceDeletionStatusPhaseFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-connectivity-rule-ups-postgres-rhmi.yaml",
		Rules: []string{
			"Ups-Postgres-RhmiPostgresConnectionFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-availability-rule-ups-postgres-rhmi.yaml",
		Rules: []string{
			"Ups-Postgres-RhmiPostgresInstanceUnavailable",
		},
	},
	{
		File: RHMIOperatorNamespace + "-availability-rule-codeready-postgres-rhmi.yaml",
		Rules: []string{
			"Codeready-Postgres-RhmiPostgresInstanceUnavailable",
		},
	},
	{
		File: RHMIOperatorNamespace + "-connectivity-rule-codeready-postgres-rhmi.yaml",
		Rules: []string{
			"Codeready-Postgres-RhmiPostgresConnectionFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-connectivity-rule-fuse-postgres-rhmi.yaml",
		Rules: []string{
			"Fuse-Postgres-RhmiPostgresConnectionFailed",
		},
	},
	{
		File: RHMIOperatorNamespace + "-availability-rule-fuse-postgres-rhmi.yaml",
		Rules: []string{
			"Fuse-Postgres-RhmiPostgresInstanceUnavailable",
		},
	},
	{
		File: NamespacePrefix + "operator-resource-status-phase-failed-rule-fuse-postgres-rhmi.yaml",
		Rules: []string{
			"Fuse-Postgres-RhmiPostgresResourceStatusPhaseFailed",
		},
	},
	{
		File: NamespacePrefix + "operator-resource-status-phase-failed-rule-ups-postgres-rhmi.yaml",
		Rules: []string{
			"Ups-Postgres-RhmiPostgresResourceStatusPhaseFailed",
		},
	},
	{
		File: NamespacePrefix + "operator-resource-status-phase-pending-rule-ups-postgres-rhmi.yaml",
		Rules: []string{
			"Ups-Postgres-RhmiPostgresResourceStatusPhasePending",
		},
	},
	{
		File: NamespacePrefix + "operator-resource-status-phase-pending-rule-fuse-postgres-rhmi.yaml",
		Rules: []string{
			"Fuse-Postgres-RhmiPostgresResourceStatusPhasePending",
		},
	},
	{
		File: NamespacePrefix + "operator-resource-status-phase-failed-rule-codeready-postgres-rhmi.yaml",
		Rules: []string{
			"Codeready-Postgres-RhmiPostgresResourceStatusPhaseFailed",
		},
	},
	{
		File: NamespacePrefix + "operator-resource-status-phase-pending-rule-codeready-postgres-rhmi.yaml",
		Rules: []string{
			"Codeready-Postgres-RhmiPostgresResourceStatusPhasePending",
		},
	},
}

var managedApiAwsExpectedRules = []alertsTestRule{
	{
		File: NamespacePrefix + "operator-resource-status-phase-failed-rule-ratelimit-service-redis-managed-api.yaml",
		Rules: []string{
			"Ratelimit-Service-Redis-Managed-ApiRedisResourceStatusPhaseFailed",
		},
	},
}

func TestIntegreatlyAlertsExist(t *testing.T, ctx *TestingContext) {
	isClusterStorage, err := isClusterStorage(ctx)
	if err != nil {
		t.Fatal("error getting isClusterStorage:", err)
	}

	rhmi, err := GetRHMI(ctx.Client, true)

	if err != nil {
		t.Fatalf("failed to get the RHMI: %s", err)
	}
	expectedAWSRules := getExpectedAWSRules(rhmi.Spec.Type)
	expectedRules := getExpectedRules(rhmi.Spec.Type)

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
			case prometheusv1.AlertingRule:
				alertRule := promRule.(prometheusv1.AlertingRule)
				rule.Rules = append(rule.Rules, alertRule.Name)
			default:
				t.Logf("unknown rule type %s", v)
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
			t.Log("\nFile Name:", k)
			t.Log("Missing Rules:", v.MissingRules)
			t.Log("Unexpected Rules:", v.AdditionalRules)
			t.Log("Status:", v.Status)
		}
		if v.Status == fileMissing || len(v.MissingRules) > 0 {
			missingCount++
		}
		if v.Status == fileAdditional || len(v.AdditionalRules) > 0 {
			extraCount++
		}
	}

	if missingCount > 0 {
		t.Log("\nMissing alerts were found from Prometheus. If the removal of these Alert rules was intentional, please update this test to remove them from the check. If the removal of these Alert rules was not intendended or you are not sure, please create a Jira & discuss with the monitoring team on how best to proceed")
	}
	if extraCount > 0 {
		t.Log("\nUnexpected alerts were found in Prometheus. If these Alert rules were intentionally added, please update this test to add them to the check. If these Alert rules were not added intentionally or you are not sure, please create a Jira & discuss with the monitoring team on how best to proceed.")
	}
	if extraCount > 0 || missingCount > 0 {
		t.Fatal("Found missing or too many alerts")
	}
}

func getExpectedAWSRules(installType string) []alertsTestRule {
	if installType == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		return commonExpectedAWSRules
	} else {
		return append(append(commonExpectedAWSRules, rhmi2ExpectedAWSRules...), managedApiAwsExpectedRules...)
	}
}

func getExpectedRules(installType string) []alertsTestRule {
	if installType == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		return commonExpectedRules
	} else {
		return append(commonExpectedRules, rhmi2ExpectedRules...)
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
