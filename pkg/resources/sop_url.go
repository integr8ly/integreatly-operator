package resources

// #nosec G101 -- false positive from urls containing `DnsBypass`
const (
	sopUrlAddonManagedApiServiceParametersExists               = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/AddonManagedApiServiceParameters.asciidoc"
	sopUrlRhoamBase                                            = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/"
	sopUrlPostgresInstanceUnavailable                          = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/postgres_instance_unavailable.asciidoc"
	sopUrlPostgresConnectionFailed                             = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/postgres_connection_failed.asciidoc"
	sopUrlRedisCacheUnavailable                                = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/redis_cache_unavailable.asciidoc"
	sopUrlRedisConnectionFailed                                = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/redis_connection_failed.asciidoc"
	sopUrlPostgresResourceStatusPhasePending                   = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/postgres_resource_status_phase_pending.asciidoc"
	sopUrlPostgresResourceStatusPhaseFailed                    = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/postgres_resource_status_phase_failed.asciidoc"
	sopUrlRedisResourceStatusPhasePending                      = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/redis_resource_status_phase_pending.asciidoc"
	sopUrlRedisResourceStatusPhaseFailed                       = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/redis_resource_status_phase_failed.asciidoc"
	sopUrlPostgresWillFill                                     = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/postgres_storage_alerts.asciidoc"
	sopUrlPostgresFreeableMemoryLow                            = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/postgres_freeable_memory_low.asciidoc"
	sopUrlRedisMemoryUsageHigh                                 = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/redis_memory_usage_high.asciidoc"
	sopUrlPostgresCpuUsageHigh                                 = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/postgres_cpu_usage_high.asciidoc"
	sopUrlRedisCpuUsageHigh                                    = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/redis_cpu_usage_high.asciidoc"
	sopUrlRedisServiceMaintenanceCritical                      = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/RedisServiceMaintenanceCritical.asciidoc"
	SopUrlEndpointAvailableAlert                               = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/tree/master/sops/rhoam/alerts/service_endpoint_down.asciidoc"
	SopUrlAlertsAndTroubleshooting                             = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/alerts_and_troubleshooting.md"
	SopUrlApicastProductionPodsDown                            = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/ThreeScaleApicastProductionPod.md"
	SopUrlSystemAppPodsDown                                    = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/ThreeScaleSystemAppPod.md"
	sopUrlCloudResourceDeletionStatusFailed                    = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/tree/master/sops/rhoam/alerts/clean_up_cloud_resources_failed_teardown.asciidoc" //#nosec G101 -- This is a false positive
	sopUrlSendGridSmtpSecretExists                             = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/sendgrid_smtp_secret_not_present.asciidoc"         //#nosec G101 -- This is a false positive
	SopUrlDeadMansSnitchSecretExists                           = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/DeadMansSnitchSecretNotPresent.asciidoc"           //#nosec G101 -- This is a false positive
	SopUrlMarin3rEnvoyApicastProductionContainerDown           = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/Marin3rEnvoyApicastProductionContainerDown.asciidoc"
	SopUrlMarin3rEnvoyApicastStagingContainerDown              = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/Marin3rEnvoyApicastStagingContainerDown.asciidoc"
	SopUrlOperatorInstallDelayed                               = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/OperatorInstallDelayed.asciidoc"
	SopUrlUpgradeExpectedDurationExceeded                      = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/UpgradeExpectedDurationExceeded.asciidoc"
	SopUrlRHOAMIsInReconcilingErrorState                       = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/RHOAMIsInReconcilingErrorState.asciidoc"
	SopUrlRHOAMCloudResourceOperatorMetricsServiceEndpointDown = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/RHOAMCloudResourceOperatorMetricsServiceEndpointDown.asciidoc"
	SopUrlRHOAMCloudResourceOperatorVPCActionFailed            = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/RHOAMCloudResourceOperatorVPCActionFailed.asciidoc"
	SopUrlRHOAMThreeScaleApicastProductionServiceEndpointDown  = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/RHOAMThreeScaleApicastProductionServiceEndpointDown.asciidoc"
	SopUrlRHOAMThreeScaleApicastStagingServiceEndpointDown     = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/RHOAMThreeScaleApicastStagingServiceEndpointDown.asciidoc"
	SopUrlRHOAMThreeScaleBackendListenerServiceEndpointDown    = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/RHOAMThreeScaleBackendListenerServiceEndpointDown.asciidoc"
	SopUrlRHOAMThreeScaleZyncServiceEndpointDown               = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/RHOAMThreeScaleZyncServiceEndpointDown.asciidoc"
	SopUrlRHOAMThreeScaleZyncDatabaseServiceEndpointDown       = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/RHOAMThreeScaleZyncDatabaseServiceEndpointDown.asciidoc"
	SopUrlThreeScaleBackendWorkerPod                           = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/ThreeScaleBackendWorkerPod.asciidoc"
	SopUrlThreeScaleAdminUIBBT                                 = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/ThreeScaleAdminUIBBT.asciidoc"
	SopUrlThreeScaleDeveloperUIBBT                             = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/ThreeScaleDeveloperUIBBT.asciidoc"
	SopUrlThreeScaleSystemAdminUIBBT                           = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/ThreeScaleSystemAdminUIBBT.asciidoc"
	SopUrlPodDistributionIncorrect                             = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/multi-az/pod_distribution.md"
	SopUrlSloRhssoAvailabilityAlert                            = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/SloRhssoAvailabilityAlert.asciidoc"
	SopUrlSloUserSsoAvailabilityAlert                          = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/SloUserSsoAvailabilityAlert.asciidoc"
	SopUrlTestFireAlerts                                       = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/cssre_info/info_test_fire_alerts.md#resolve-test-alerts"
	SopUrlRHOAMServiceDefinition                               = "https://access.redhat.com/articles/5534341"
	SopUrlDnsBypassThreeScaleAdminUI                           = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/DnsBypassThreeScaleAdminUI.asciidoc"
	SopUrlDnsBypassThreeScaleDeveloperUI                       = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/DnsBypassThreeScaleDeveloperUI.asciidoc"
	SopUrlDnsBypassThreeScaleSystemAdminUI                     = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/DnsBypassThreeScaleSystemAdminUI.asciidoc"
	SopUrlKeycloakInstanceNotAvailable                         = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/KeycloakInstanceNotAvailable.asciidoc"
	SopUrlCriticalMetricsMissing                               = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/CriticalMetricsMissing.asciidoc"
	SopUrlClusterSchedulableResourcesLow                       = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/alerts/Cluster_Schedulable_Resources_Low.asciidoc"
	SopUrlKubePersistentVolumeFillingUp4h                      = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/pvc_storage.asciidoc#pvcstoragewillfillin4hours"
	SopUrlKubePersistentVolumeFillingUp                        = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/pvc_storage.asciidoc#kubepersistentvolumefillingup"
	SopUrlPersistentVolumeErrors                               = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/pvc_storage.asciidoc#persistentvolumeerrors"
	SopApiManagementTenantCRFailed                             = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/ApiManagementTenantCRFailed.asciidoc"
	SopUrlRHOAMCloudResourceOperatorMetricsMissing             = "https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/blob/master/sops/rhoam/alerts/RHOAMCloudResourceOperatorMetricsMissing.asciidoc"
)
