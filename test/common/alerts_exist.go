package common

import (
	"encoding/json"
	"fmt"
	"strings"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
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
func rhmi2ExpectedRules(installationName string) []alertsTestRule {

	titledName := strings.Title(installationName)

	return []alertsTestRule{
		{
			File: ObservabilityNamespacePrefix + "rhssouser.yaml",
			Rules: []string{
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
			File: ObservabilityNamespacePrefix + "user-sso-slo-availability-alerts.yaml",
			Rules: []string{
				fmt.Sprintf("%sUserSsoAvailability5mto1hErrorBudgetBurn", strings.ToUpper(titledName)),
				fmt.Sprintf("%sUserSsoAvailability30mto6hErrorBudgetBurn", strings.ToUpper(titledName)),
				fmt.Sprintf("%sUserSsoAvailability2hto1dErrorBudgetBurn", strings.ToUpper(titledName)),
				fmt.Sprintf("%sUserSsoAvailability6hto3dErrorBudgetBurn", strings.ToUpper(titledName)),
			},
		},
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
			File: ObservabilityNamespacePrefix + "user-sso-operator-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"RHOAMUserRhssoOperatorRhmiRegistryCsMetricsServiceEndpointDown",
				"RHOAMUserRhssoKeycloakOperatorMetricsServiceEndpointDown",
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
				"RHOAMApicuritoServiceEndpointDown",
				"RHOAMApicuritoFuseApicuritoGeneratorServiceEndpointDown",
			},
		},
		{
			File: NamespacePrefix + "amq-online-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"RHOAMAMQOnlineNoneAuthServiceEndpointDown",
				"RHOAMAMQOnlineAddressSpaceControllerServiceEndpointDown",
				"RHOAMAMQOnlineConsoleServiceEndpointDown",
				"RHOAMAMQOnlineRegistryCsServiceEndpointDown",
				"RHOAMAMQOnlineStandardAuthServiceEndpointDown",
				"RHOAMAMQOnlineEnmasseOperatorMetricsServiceEndpointDown",
			},
		},
		{
			File: NamespacePrefix + "fuse-operator-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"RHOAMFuseOnlineOperatorRhmiRegistryCsServiceEndpointDown",
				"RHOAMFuseOnlineOperatorSyndesisOperatorMetricsServiceEndpointDown",
			},
		},
		{
			File: NamespacePrefix + "ups-operator-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"RHOAMUPSOperatorRhmiRegistryCsServiceEndpointDown",
				"RHOAMUPSOperatorUnifiedPushOperatorMetricsServiceEndpointDown",
			},
		},
		{
			File: NamespacePrefix + "codeready-workspaces-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"RHOAMCodeReadyCheHostServiceEndpointDown",
				"RHOAMCodeReadyDevfileRegistryServiceEndpointDown",
				"RHOAMCodeReadyPluginRegistryServiceEndpointDown",
			},
		},
		{
			File: NamespacePrefix + "apicurito-operator-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"RHOAMApicuritoOperatorRhmiRegistryCsServiceEndpointDown",
			},
		},
		{
			File: NamespacePrefix + "solution-explorer-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"RHOAMSolutionExplorerTutorialWebAppServiceEndpointDown",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "user-sso-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"RHOAMUserRhssoKeycloakServiceEndpointDown",
				"RHOAMUserRhssoKeycloakDiscoveryServiceEndpointDown",
			},
		},
		{
			File: NamespacePrefix + "solution-explorer-operator-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"RHOAMSolutionExplorerOperatorRhmiRegistryCsServiceEndpointDown",
			},
		},
		{
			File: NamespacePrefix + "codeready-workspaces-operator-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"RHOAMCodeReadyOperatorRhmiRegistryCsServiceEndpointDown",
			},
		},
		{
			File: NamespacePrefix + "fuse-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"RHOAMFuseOnlineBrokerAmqTcpServiceEndpointDown",
				"RHOAMFuseOnlineSyndesisMetaServiceEndpointDown",
				"RHOAMFuseOnlineSyndesisOauthproxyServiceEndpointDown",
				"RHOAMFuseOnlineSyndesisPrometheusServiceEndpointDown",
				"RHOAMFuseOnlineSyndesisServerServiceEndpointDown",
				"RHOAMFuseOnlineSyndesisUiServiceEndpointDown",
			},
		},
		{
			File: NamespacePrefix + "ups-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"RHOAMUPSUnifiedPushServiceEndpointDown",
				"RHOAMUPSUnifiedpushProxyServiceEndpointDown",
			},
		},
	}
}

// Managed-Api-Service rules
func managedApiSpecificRules(installationName string) []alertsTestRule {

	titledName := strings.Title(installationName)

	return []alertsTestRule{
		{
			File: ObservabilityNamespacePrefix + "rhssouser.yaml",
			Rules: []string{
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
			File: ObservabilityNamespacePrefix + "user-sso-slo-availability-alerts.yaml",
			Rules: []string{
				fmt.Sprintf("%sUserSsoAvailability5mto1hErrorBudgetBurn", strings.ToUpper(titledName)),
				fmt.Sprintf("%sUserSsoAvailability30mto6hErrorBudgetBurn", strings.ToUpper(titledName)),
				fmt.Sprintf("%sUserSsoAvailability2hto1dErrorBudgetBurn", strings.ToUpper(titledName)),
				fmt.Sprintf("%sUserSsoAvailability6hto3dErrorBudgetBurn", strings.ToUpper(titledName)),
			},
		},
		{
			File: ObservabilityNamespacePrefix + "marin3r-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"Marin3rDiscoveryServiceEndpointDown",
				"Marin3rRateLimitServiceEndpointDown",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "user-sso-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"RHOAMUserRhssoKeycloakServiceEndpointDown",
				"RHOAMUserRhssoKeycloakDiscoveryServiceEndpointDown",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "marin3r-operator-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"Marin3rOperatorRhmiRegistryCsServiceEndpointDown",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "marin3r-operator-ksm-marin3r-alerts.yaml",
			Rules: []string{
				"Marin3rOperatorPod",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "marin3r-ksm-marin3r-alerts.yaml",
			Rules: []string{
				"Marin3rWebhookPod",
				"Marin3rRateLimitPod",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "3scale-ksm-marin3r-alerts.yaml",
			Rules: []string{
				"Marin3rEnvoyApicastStagingContainerDown",
				"Marin3rEnvoyApicastProductionContainerDown",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "user-sso-operator-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"RHOAMUserRhssoOperatorRhmiRegistryCsMetricsServiceEndpointDown",
				"RHOAMUserRhssoKeycloakOperatorMetricsServiceEndpointDown",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "customer-monitoring-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"GrafanaOperatorRhmiRegistryCsServiceEndpointDown",
				"GrafanaServiceEndpointDown",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "customer-monitoring-ksm-grafana-alerts.yaml",
			Rules: []string{
				"GrafanaOperatorPod",
				"GrafanaServicePod",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "marin3r-api-usage-alert-level1.yaml",
			Rules: []string{
				"RHOAMApiUsageLevel1ThresholdExceeded",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "marin3r-api-usage-alert-level2.yaml",
			Rules: []string{
				"RHOAMApiUsageLevel2ThresholdExceeded",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "marin3r-api-usage-alert-level3.yaml",
			Rules: []string{
				"RHOAMApiUsageLevel3ThresholdExceeded",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "marin3r-rate-limit-spike.yaml",
			Rules: []string{
				"RHOAMApiUsageOverLimit",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "marin3r-rejected-requests.yaml",
			Rules: []string{
				"RHOAMApiUsageRejectedRequestsMismatch",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "rhoam-rhmi-controller-alerts.yaml",
			Rules: []string{
				"RHOAMIsInReconcilingErrorState",
				"RHOAMInstallationControllerReconcileDelayed",
			},
		},
	}
}

// Managed-Api-Service rules
func mtManagedApiSpecificRules() []alertsTestRule {

	return []alertsTestRule{
		{
			File: ObservabilityNamespacePrefix + "multitenancy-api-management-tenant-alerts.yaml",
			Rules: []string{
				"ApiManagementTenantCRFailed",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "marin3r-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"Marin3rDiscoveryServiceEndpointDown",
				"Marin3rRateLimitServiceEndpointDown",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "marin3r-operator-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"Marin3rOperatorRhmiRegistryCsServiceEndpointDown",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "marin3r-operator-ksm-marin3r-alerts.yaml",
			Rules: []string{
				"Marin3rOperatorPod",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "marin3r-ksm-marin3r-alerts.yaml",
			Rules: []string{
				"Marin3rWebhookPod",
				"Marin3rRateLimitPod",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "3scale-ksm-marin3r-alerts.yaml",
			Rules: []string{
				"Marin3rEnvoyApicastStagingContainerDown",
				"Marin3rEnvoyApicastProductionContainerDown",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "customer-monitoring-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"GrafanaOperatorRhmiRegistryCsServiceEndpointDown",
				"GrafanaServiceEndpointDown",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "customer-monitoring-ksm-grafana-alerts.yaml",
			Rules: []string{
				"GrafanaOperatorPod",
				"GrafanaServicePod",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "marin3r-rejected-requests.yaml",
			Rules: []string{
				"RHOAMApiUsageRejectedRequestsMismatch",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "rhoam-rhmi-controller-alerts.yaml",
			Rules: []string{
				"RHOAMIsInReconcilingErrorState",
				"RHOAMInstallationControllerReconcileDelayed",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "api-usage-alert-level1.yaml",
			Rules: []string{
				"RHOAMApiUsageLevel1ThresholdExceeded",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "api-usage-alert-level2.yaml",
			Rules: []string{
				"RHOAMApiUsageLevel2ThresholdExceeded",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "api-usage-alert-level3.yaml",
			Rules: []string{
				"RHOAMApiUsageLevel3ThresholdExceeded",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "rate-limit-spike.yaml",
			Rules: []string{
				"RHOAMApiUsageOverLimit",
			},
		},
	}
}

// Common to all install types
func commonExpectedRules(installationName string) []alertsTestRule {
	titledName := strings.Title(installationName)
	return []alertsTestRule{
		{
			File: ObservabilityNamespacePrefix + "backup-monitoring-alerts.yaml",
			Rules: []string{
				"JobRunningTimeExceeded",
				"JobRunningTimeExceeded",
				"CronJobNotRunInThreshold",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "rhsso.yaml",
			Rules: []string{
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
			File: ObservabilityNamespacePrefix + "ksm-alerts.yaml",
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
				"KubePersistentVolumeFillingUp4h",
				"KubePersistentVolumeFillingUp",
				"PersistentVolumeErrors",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "3scale-ksm-3scale-alerts.yaml",
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
				"ThreescaleApicastWorkerRestart",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "generated-deadmansswitch.yaml",
			Rules: []string{
				"DeadMansSwitch",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "3scale-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"RHOAMThreeScaleApicastProductionServiceEndpointDown",
				"RHOAMThreeScaleApicastStagingServiceEndpointDown",
				"RHOAMThreeScaleBackendListenerServiceEndpointDown",
				"RHOAMThreeScaleSystemDeveloperServiceEndpointDown",
				"RHOAMThreeScaleSystemMasterServiceEndpointDown",
				"RHOAMThreeScaleSystemMemcacheServiceEndpointDown",
				"RHOAMThreeScaleSystemProviderServiceEndpointDown",
				"RHOAMThreeScaleSystemSphinxServiceEndpointDown",
				"RHOAMThreeScaleZyncDatabaseServiceEndpointDown",
				"RHOAMThreeScaleZyncServiceEndpointDown",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "3scale-ksm-3scale-user-alerts.yaml",
			Rules: []string{
				"ThreeScaleUserCreationFailed",
				"ThreeScaleUserDeletionFailed",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "cro-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"RHOAMCloudResourceOperatorMetricsServiceEndpointDown",
				"RHOAMCloudResourceOperatorRhmiRegistryCsServiceEndpointDown",
				"RHOAMCloudResourceOperatorElasticCacheSnapshotsNotFound",
				"RHOAMCloudResourceOperatorVPCActionFailed",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "ksm-endpoint-alerts.yaml",
			Rules: []string{
				"RHOAMMiddlewareMonitoringOperatorAlertmanagerOperatedServiceEndpointDown",
				"RHOAMMiddlewareMonitoringOperatorAlertmanagerServiceEndpointDown",
				"RHOAMMiddlewareMonitoringOperatorApplicationMonitoringMetricsServiceEndpointDown",
				"RHOAMMiddlewareMonitoringOperatorGrafanaServiceEndpointDown",
				"RHOAMMiddlewareMonitoringOperatorPrometheusOperatedServiceEndpointDown",
				"RHOAMMiddlewareMonitoringOperatorPrometheusServiceEndpointDown",
				"RHOAMMiddlewareMonitoringOperatorRhmiRegistryCsServiceEndpointDown",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "rhsso-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"RHOAMRhssoKeycloakServiceEndpointDown",
				"RHOAMRhssoKeycloakDiscoveryServiceEndpointDown",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "rhsso-operator-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"RHOAMRhssoKeycloakOperatorRhmiRegistryCsServiceEndpointDown",
				"RHOAMRhssoKeycloakOperatorMetricsServiceEndpointDown",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "3scale-operator-ksm-endpoint-alerts.yaml",
			Rules: []string{
				"RHOAMThreeScaleOperatorRhmiRegistryCsServiceEndpointDown",
				"RHOAMThreeScaleOperatorServiceEndpointDown",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "install-upgrade-alerts.yaml",
			Rules: []string{
				"RHOAMCSVRequirementsNotMet",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "sendgrid-smtp-secret-exists-rule.yaml",
			Rules: []string{
				"SendgridSmtpSecretExists",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "addon-managed-api-service-parameters-secret-exists-rule.yaml",
			Rules: []string{
				"AddonManagedApiServiceParametersExists",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "deadmanssnitch-secret-exists-rule.yaml",
			Rules: []string{
				"DeadMansSnitchSecretExists",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "multi-az-pod-distribution.yaml",
			Rules: []string{
				"MultiAZPodDistribution",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "rhsso-slo-availability-alerts.yaml",
			Rules: []string{
				fmt.Sprintf("%sRhssoAvailability5mto1hErrorBudgetBurn", strings.ToUpper(titledName)),
				fmt.Sprintf("%sRhssoAvailability30mto6hErrorBudgetBurn", strings.ToUpper(titledName)),
				fmt.Sprintf("%sRhssoAvailability2hto1dErrorBudgetBurn", strings.ToUpper(titledName)),
				fmt.Sprintf("%sRhssoAvailability6hto3dErrorBudgetBurn", strings.ToUpper(titledName)),
			},
		},
		{
			File: ObservabilityNamespacePrefix + "test-alerts.yaml",
			Rules: []string{
				"TestFireCriticalAlert",
				"TestFireWarningAlert",
			},
		},
		{
			File: ObservabilityNamespacePrefix + "rhoam-custom-domain-alert.yaml",
			Rules: []string{
				"CustomDomainCRErrorState",
				"DnsBypassThreeScaleAdminUI",
				"DnsBypassThreeScaleDeveloperUI",
				"DnsBypassThreeScaleSystemAdminUI",
			},
		},
	}
}

// common aws rules applicable to all install types
func commonExpectedAWSRules(installationName string) []alertsTestRule {
	titledName := strings.Title(installationName)

	return []alertsTestRule{
		{
			File: fmt.Sprintf("%s-connectivity-rule-threescale-redis-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Threescale-Redis-%sRedisCacheConnectionFailed", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-connectivity-rule-threescale-backend-redis-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Threescale-Backend-Redis-%sRedisCacheConnectionFailed", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-connectivity-rule-threescale-postgres-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Threescale-Postgres-%sPostgresConnectionFailed", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-availability-rule-threescale-redis-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Threescale-Redis-%sRedisCacheUnavailable", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-availability-rule-threescale-backend-redis-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Threescale-Backend-Redis-%sRedisCacheUnavailable", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-resource-deletion-status-phase-failed-rule-threescale-redis-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Threescale-Redis-%sRedisResourceDeletionStatusPhaseFailed", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-resource-deletion-status-phase-failed-rule-threescale-backend-redis-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Threescale-Backend-Redis-%sRedisResourceDeletionStatusPhaseFailed", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-resource-deletion-status-phase-failed-rule-threescale-postgres-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Threescale-Postgres-%sPostgresResourceDeletionStatusPhaseFailed", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-resource-deletion-status-phase-failed-rule-rhsso-postgres-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Rhsso-Postgres-%sPostgresResourceDeletionStatusPhaseFailed", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-availability-rule-threescale-postgres-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Threescale-Postgres-%sPostgresInstanceUnavailable", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-connectivity-rule-rhsso-postgres-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Rhsso-Postgres-%sPostgresConnectionFailed", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-availability-rule-rhsso-postgres-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Rhsso-Postgres-%sPostgresInstanceUnavailable", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-resource-status-phase-failed-rule-threescale-redis-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Threescale-Redis-%sRedisResourceStatusPhaseFailed", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-resource-status-phase-failed-rule-rhsso-postgres-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Rhsso-Postgres-%sPostgresResourceStatusPhaseFailed", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-resource-status-phase-pending-rule-threescale-postgres-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Threescale-Postgres-%sPostgresResourceStatusPhasePending", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-resource-status-phase-pending-rule-threescale-redis-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Threescale-Redis-%sRedisResourceStatusPhasePending", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-resource-status-phase-pending-rule-rhsso-postgres-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Rhsso-Postgres-%sPostgresResourceStatusPhasePending", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-resource-status-phase-pending-rule-threescale-backend-redis-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Threescale-Backend-Redis-%sRedisResourceStatusPhasePending", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-resource-status-phase-failed-rule-threescale-backend-redis-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Threescale-Backend-Redis-%sRedisResourceStatusPhaseFailed", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-resource-status-phase-failed-rule-threescale-postgres-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Threescale-Postgres-%sPostgresResourceStatusPhaseFailed", titledName),
			},
		},
		{
			File: ObservabilityProductNamespace + "-postgres-storage-will-fill-in-4-hours.yaml",
			Rules: []string{
				"PostgresStorageWillFillIn4Hours",
			},
		},
		{
			File: ObservabilityProductNamespace + "-postgres-storage-will-fill-in-4-days.yaml",
			Rules: []string{
				"PostgresStorageWillFillIn4Days",
			},
		},
		{
			File: ObservabilityProductNamespace + "-postgres-storage-low.yaml",
			Rules: []string{
				"PostgresStorageLow",
			},
		},
		{
			File: ObservabilityProductNamespace + "-postgres-cpu-high.yaml",
			Rules: []string{
				"PostgresCPUHigh",
			},
		},
		{
			File: ObservabilityProductNamespace + "-postgres-freeable-memory-low.yaml",
			Rules: []string{
				"PostgresFreeableMemoryLow",
			},
		},
		{
			File: ObservabilityProductNamespace + "-redis-memory-usage-high.yaml",
			Rules: []string{
				"RedisMemoryUsageHigh",
			},
		},
		{
			File: ObservabilityProductNamespace + "-redis-memory-usage-will-max-in-4-hours.yaml",
			Rules: []string{
				"RedisMemoryUsageMaxIn4Hours",
			},
		},
		{
			File: ObservabilityProductNamespace + "-redis-memory-usage-max-fill-in-4-days.yaml",
			Rules: []string{
				"RedisMemoryUsageMaxIn4Days",
			},
		},
		{
			File: ObservabilityProductNamespace + "-redis-cpu-usage-high.yaml",
			Rules: []string{
				"RedisCpuUsageHigh",
			},
		},
		{
			File: ObservabilityProductNamespace + "-redis-service-maintenance-critical.yaml",
			Rules: []string{
				"RedisServiceMaintenanceCritical",
			},
		},
	}
}

// rhmi2 aws rules
func rhmi2ExpectedAWSRules(installationName string) []alertsTestRule {
	titledName := strings.Title(installationName)

	return []alertsTestRule{
		{
			File: fmt.Sprintf("%s-resource-status-phase-pending-rule-rhssouser-postgres-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Rhssouser-Postgres-%sPostgresResourceStatusPhasePending", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-connectivity-rule-rhssouser-postgres-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Rhssouser-Postgres-%sPostgresConnectionFailed", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-availability-rule-rhssouser-postgres-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Rhssouser-Postgres-%sPostgresInstanceUnavailable", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-resource-deletion-status-phase-failed-rule-rhssouser-postgres-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Rhssouser-Postgres-%sPostgresResourceDeletionStatusPhaseFailed", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-resource-status-phase-failed-rule-rhssouser-postgres-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Rhssouser-Postgres-%sPostgresResourceStatusPhaseFailed", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-resource-deletion-status-phase-failed-rule-codeready-postgres-%s.yaml", RHOAMOperatorNamespace, installationName),
			Rules: []string{
				"Codeready-Postgres-RHOAMPostgresResourceDeletionStatusPhaseFailed",
			},
		},
		{
			File: fmt.Sprintf("%s-resource-deletion-status-phase-failed-rule-ups-postgres-%s.yaml", RHOAMOperatorNamespace, installationName),
			Rules: []string{
				"Ups-Postgres-RHOAMPostgresResourceDeletionStatusPhaseFailed",
			},
		},
		{
			File: fmt.Sprintf("%s-resource-deletion-status-phase-failed-rule-fuse-postgres-%s.yaml", RHOAMOperatorNamespace, installationName),
			Rules: []string{
				"Fuse-Postgres-RHOAMPostgresResourceDeletionStatusPhaseFailed",
			},
		},
		{
			File: fmt.Sprintf("%s-connectivity-rule-ups-postgres-%s.yaml", RHOAMOperatorNamespace, installationName),
			Rules: []string{
				"Ups-Postgres-RHOAMPostgresConnectionFailed",
			},
		},
		{
			File: fmt.Sprintf("%s-availability-rule-ups-postgres-%s.yaml", RHOAMOperatorNamespace, installationName),
			Rules: []string{
				"Ups-Postgres-RHOAMPostgresInstanceUnavailable",
			},
		},
		{
			File: fmt.Sprintf("%s-availability-rule-codeready-postgres-%s.yaml", RHOAMOperatorNamespace, installationName),
			Rules: []string{
				"Codeready-Postgres-RHOAMPostgresInstanceUnavailable",
			},
		},
		{
			File: fmt.Sprintf("%s-connectivity-rule-codeready-postgres-%s.yaml", RHOAMOperatorNamespace, installationName),
			Rules: []string{
				"Codeready-Postgres-RHOAMPostgresConnectionFailed",
			},
		},
		{
			File: fmt.Sprintf("%s-connectivity-rule-fuse-postgres-%s.yaml", RHOAMOperatorNamespace, installationName),
			Rules: []string{
				"Fuse-Postgres-RHOAMPostgresConnectionFailed",
			},
		},
		{
			File: fmt.Sprintf("%s-availability-rule-fuse-postgres-%s.yaml", RHOAMOperatorNamespace, installationName),
			Rules: []string{
				"Fuse-Postgres-RHOAMPostgresInstanceUnavailable",
			},
		},
		{
			File: fmt.Sprintf("%soperator-resource-status-phase-failed-rule-fuse-postgres-%s.yaml", NamespacePrefix, installationName),
			Rules: []string{
				"Fuse-Postgres-RHOAMPostgresResourceStatusPhaseFailed",
			},
		},
		{
			File: fmt.Sprintf("%soperator-resource-status-phase-failed-rule-ups-postgres-%s.yaml", NamespacePrefix, installationName),
			Rules: []string{
				"Ups-Postgres-RHOAMPostgresResourceStatusPhaseFailed",
			},
		},
		{
			File: fmt.Sprintf("%soperator-resource-status-phase-pending-rule-ups-postgres-%s.yaml", NamespacePrefix, installationName),
			Rules: []string{
				"Ups-Postgres-RHOAMPostgresResourceStatusPhasePending",
			},
		},
		{
			File: fmt.Sprintf("%soperator-resource-status-phase-pending-rule-fuse-postgres-%s.yaml", NamespacePrefix, installationName),
			Rules: []string{
				"Fuse-Postgres-RHOAMPostgresResourceStatusPhasePending",
			},
		},
		{
			File: fmt.Sprintf("%soperator-resource-status-phase-failed-rule-codeready-postgres-%s.yaml", NamespacePrefix, installationName),
			Rules: []string{
				"Codeready-Postgres-RHOAMPostgresResourceStatusPhaseFailed",
			},
		},
		{
			File: fmt.Sprintf("%soperator-resource-status-phase-pending-rule-codeready-postgres-%s.yaml", NamespacePrefix, installationName),
			Rules: []string{
				"Codeready-Postgres-RHOAMPostgresResourceStatusPhasePending",
			},
		},
	}
}

func managedApiAwsExpectedRules(installationName string) []alertsTestRule {
	titledName := strings.Title(installationName)

	return []alertsTestRule{
		{
			File: fmt.Sprintf("%s-resource-status-phase-pending-rule-rhssouser-postgres-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Rhssouser-Postgres-%sPostgresResourceStatusPhasePending", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-connectivity-rule-rhssouser-postgres-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Rhssouser-Postgres-%sPostgresConnectionFailed", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-availability-rule-rhssouser-postgres-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Rhssouser-Postgres-%sPostgresInstanceUnavailable", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-resource-deletion-status-phase-failed-rule-rhssouser-postgres-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Rhssouser-Postgres-%sPostgresResourceDeletionStatusPhaseFailed", titledName),
			},
		},
		{
			File: fmt.Sprintf("%s-resource-status-phase-failed-rule-rhssouser-postgres-%s.yaml", ObservabilityProductNamespace, installationName),
			Rules: []string{
				fmt.Sprintf("Rhssouser-Postgres-%sPostgresResourceStatusPhaseFailed", titledName),
			},
		},
		{
			File: fmt.Sprintf("%sresource-status-phase-failed-rule-ratelimit-service-redis-%s.yaml", ObservabilityNamespacePrefix, installationName),
			Rules: []string{
				fmt.Sprintf("Ratelimit-Service-Redis-%sRedisResourceStatusPhaseFailed", titledName),
			},
		},
		{
			File: fmt.Sprintf("%sresource-status-phase-pending-rule-ratelimit-service-redis-%s.yaml", ObservabilityNamespacePrefix, installationName),
			Rules: []string{
				fmt.Sprintf("Ratelimit-Service-Redis-%sRedisResourceStatusPhasePending", titledName),
			},
		},
		{
			File: fmt.Sprintf("%savailability-rule-ratelimit-service-redis-%s.yaml", ObservabilityNamespacePrefix, installationName),
			Rules: []string{
				fmt.Sprintf("Ratelimit-Service-Redis-%sRedisCacheUnavailable", titledName),
			},
		},
		{
			File: fmt.Sprintf("%sconnectivity-rule-ratelimit-service-redis-%s.yaml", ObservabilityNamespacePrefix, installationName),
			Rules: []string{
				fmt.Sprintf("Ratelimit-Service-Redis-%sRedisCacheConnectionFailed", titledName),
			},
		},
		{
			File: fmt.Sprintf("%sresource-deletion-status-phase-failed-rule-ratelimit-service-redis-%s.yaml", ObservabilityNamespacePrefix, installationName),
			Rules: []string{
				fmt.Sprintf("Ratelimit-Service-Redis-%sRedisResourceDeletionStatusPhaseFailed", titledName),
			},
		},
	}

}

func TestIntegreatlyAlertsExist(t TestingTB, ctx *TestingContext) {
	isClusterStorage, err := isClusterStorage(ctx)
	if err != nil {
		t.Fatal("error getting isClusterStorage:", err)
	}

	rhmi, err := GetRHMI(ctx.Client, true)

	if err != nil {
		t.Fatalf("failed to get the RHMI: %s", err)
	}
	expectedAWSRules := getExpectedAWSRules(rhmi.Spec.Type, rhmi.Name)
	expectedRules := getExpectedRules(rhmi.Spec.Type, rhmi.Name)

	// add external database alerts to list of expected rules if
	// cluster storage is not being used
	if !isClusterStorage {
		for _, rule := range expectedAWSRules {
			expectedRules = append(expectedRules, rule)
		}
	}

	// exec into the prometheus pod
	output, err := execToPod("wget -qO - localhost:9090/api/v1/rules",
		"prometheus-prometheus-0",
		ObservabilityProductNamespace,
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
		t.Log("\nMissing alerts were found from Prometheus. If the removal of these Alert rules was intentional, please update this test to remove them from the check. If the removal of these Alert rules was not intended or you are not sure, please create a Jira & discuss with the monitoring team on how best to proceed")
	}
	if extraCount > 0 {
		t.Log("\nUnexpected alerts were found in Prometheus. If these Alert rules were intentionally added, please update this test to add them to the check. If these Alert rules were not added intentionally or you are not sure, please create a Jira & discuss with the monitoring team on how best to proceed.")
	}
	if extraCount > 0 || missingCount > 0 {
		t.Fatal("Found missing or too many alerts")
	}
}

func getExpectedAWSRules(installType string, installationName string) []alertsTestRule {
	if rhmiv1alpha1.IsRHOAMSingletenant(rhmiv1alpha1.InstallationType(installType)) {
		return append(commonExpectedAWSRules(installationName), managedApiAwsExpectedRules(installationName)...)
	} else if rhmiv1alpha1.IsRHOAMMultitenant(rhmiv1alpha1.InstallationType(installType)) {
		return commonExpectedAWSRules(installationName)
	} else {
		return append(commonExpectedAWSRules(installationName), rhmi2ExpectedAWSRules(installationName)...)
	}
}

func getExpectedRules(installType string, installationName string) []alertsTestRule {
	if rhmiv1alpha1.IsRHOAMSingletenant(rhmiv1alpha1.InstallationType(installType)) {
		return append(commonExpectedRules(installationName), managedApiSpecificRules(installationName)...)
	} else if rhmiv1alpha1.IsRHOAMMultitenant(rhmiv1alpha1.InstallationType(installType)) {
		return append(commonExpectedRules(installationName), mtManagedApiSpecificRules()...)
	} else {
		return append(commonExpectedRules(installationName), rhmi2ExpectedRules(installationName)...)
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
