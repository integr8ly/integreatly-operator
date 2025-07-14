package obo

import (
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strings"
)

func OboAlertsReconciler(logger l.Logger, installation *integreatlyv1alpha1.RHMI) resources.AlertReconciler {
	installationName := resources.InstallationNames[installation.Spec.Type]
	nsPrefix := installation.Spec.NamespacePrefix
	namespace := config.GetOboNamespace(installation.Namespace)

	alerts := []resources.AlertConfiguration{
		{
			AlertName: "backup-monitoring-alerts",
			GroupName: "general.rules",
			Namespace: namespace,
			Rules: []monv1.Rule{
				{
					Alert: "JobRunningTimeExceeded",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlAlertsAndTroubleshooting,
						"message": " Job {{ $labels.namespace }} / {{ $labels.job  }} has been running for longer than 300 seconds",
					},
					Expr:   intstr.FromString("time() - (max(kube_job_status_active * ON(job_name) GROUP_RIGHT() kube_job_labels{label_monitoring_key='middleware'}) BY (job_name) * ON(job_name) GROUP_RIGHT() max(kube_job_status_start_time * ON(job_name) GROUP_RIGHT() kube_job_labels{label_monitoring_key='middleware'}) BY (job_name, namespace, label_cronjob_name) > 0) > 300 "),
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
				{
					Alert: "JobRunningTimeExceeded",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlAlertsAndTroubleshooting,
						"message": " Job {{ $labels.namespace }} / {{ $labels.job  }} has been running for longer than 600 seconds",
					},
					Expr:   intstr.FromString("time() - (max(kube_job_status_active * ON(job_name) GROUP_RIGHT() kube_job_labels{label_monitoring_key='middleware'}) BY (job_name) * ON(job_name) GROUP_RIGHT() max(kube_job_status_start_time * ON(job_name) GROUP_RIGHT() kube_job_labels{label_monitoring_key='middleware'}) BY (job_name, namespace, label_cronjob_name) > 0) > 600 "),
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
				{
					Alert: "CronJobNotRunInThreshold",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlAlertsAndTroubleshooting,
						"message": " CronJob {{ $labels.namespace }} / {{ $labels.label_cronjob_name }} has not started a Job in 25 hours",
					},
					Expr: intstr.FromString("(time() - (max( kube_job_status_start_time * ON(job_name) GROUP_RIGHT() kube_job_labels{label_monitoring_key='middleware'} ) BY (job_name, label_cronjob_name) == ON(label_cronjob_name) GROUP_LEFT() max( kube_job_status_start_time * ON(job_name) GROUP_RIGHT() kube_job_labels{label_monitoring_key='middleware'} ) BY (label_cronjob_name))) > 60*60*25"),
				},
			},
		},
		{
			AlertName: "ksm-alerts",
			Namespace: namespace,
			GroupName: "general.rules",
			Rules: []monv1.Rule{
				{
					Alert: "KubePodCrashLooping",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlAlertsAndTroubleshooting,
						"message": "Pod {{  $labels.namespace  }} / {{  $labels.pod  }} ({{  $labels.container  }}) is restarting {{  $value  }} times every 10 minutes; for the last 15 minutes",
					},
					Expr:   intstr.FromString(fmt.Sprintf("rate(kube_pod_container_status_restarts_total{job='kube-state-metrics',namespace=~'%s.*'}[10m]) * 60 * 5 > 0", nsPrefix)),
					For:    "15m",
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
				{
					Alert: "KubePodNotReady",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlAlertsAndTroubleshooting,
						"message": "Pod {{ $labels.namespace }} / {{ $labels.pod }}  has been in a non-ready state for longer than 15 minutes.",
					},
					Expr:   intstr.FromString(fmt.Sprintf("sum by(pod, namespace) (kube_pod_status_phase{phase=~'Pending|Unknown', namespace=~'%s.*'}) > 0", nsPrefix)),
					For:    "15m",
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
				{
					Alert: "KubePodImagePullBackOff",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlAlertsAndTroubleshooting,
						"message": "Pod {{ $labels.namespace }} / {{  $labels.pod  }} has been unable to pull its image for longer than 5 minutes.",
					},
					Expr:   intstr.FromString(fmt.Sprintf("kube_pod_container_status_waiting_reason{reason='ImagePullBackOff',namespace=~'%s.*'} > 0", nsPrefix)),
					For:    "5m",
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
				{
					Alert: "KubePodBadConfig",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlAlertsAndTroubleshooting,
						"message": " Pod {{ $labels.namespace  }} / {{  $labels.pod  }} has been unable to start due to a bad configuration for longer than 5 minutes",
					},
					Expr:   intstr.FromString(fmt.Sprintf("kube_pod_container_status_waiting_reason{reason='CreateContainerConfigError',namespace=~'%s.*'} > 0", nsPrefix)),
					For:    "5m",
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
				{
					Alert: "KubePodStuckCreating",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlAlertsAndTroubleshooting,
						"message": "Pod {{  $labels.namespace }} / {{  $labels.pod  }} has been trying to start for longer than 15 minutes - this could indicate a configuration error.",
					},
					Expr:   intstr.FromString(fmt.Sprintf("kube_pod_container_status_waiting_reason{reason='ContainerCreating',namespace=~'%s.*'} > 0", nsPrefix)),
					For:    "15m",
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
				{
					Alert: "ClusterSchedulableMemoryLow",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlClusterSchedulableResourcesLow,
						"message": "The cluster has {{  $value }} percent of memory requested and unavailable for scheduling for longer than 15 minutes.",
					},
					Expr:   intstr.FromString("((sum(sum by(node) (sum by(pod, node) (kube_pod_container_resource_requests{resource='memory'} * on(node) group_left() (sum by(node) (kube_node_role{role='worker'}  == 1))) * on(pod) group_left() (sum by(pod) (kube_pod_status_phase{phase='Running'}) == 1)))) / ((sum((kube_node_role{role='worker'}  == 1) * on(node) group_left() (sum by(node) (kube_node_status_allocatable{resource='memory'})))))) * 100 > 85"),
					For:    "15m",
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
				{
					Alert: "ClusterSchedulableCPULow",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlClusterSchedulableResourcesLow,
						"message": "The cluster has {{ $value }} percent of CPU cores requested and unavailable for scheduling for longer than 15 minutes.",
					},
					Expr:   intstr.FromString("((sum(sum by(node) (sum by(pod, node) (kube_pod_container_resource_requests{resource='cpu'} * on(node) group_left() (sum by(node) (kube_node_role{role='worker'} == 1))) * on(pod) group_left() (sum by(pod) (kube_pod_status_phase{phase='Running'}) == 1)))) / ((sum((kube_node_role{role='worker'} == 1) * on(node) group_left() (sum by(node) (kube_node_status_allocatable{resource='cpu'})))))) * 100 > 85"),
					For:    "15m",
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
				{
					Alert: "PVCStorageAvailable",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlClusterSchedulableResourcesLow,
						"message": "The {{  $labels.persistentvolumeclaim  }} PVC has has been {{ $value }} percent full for longer than 15 minutes.",
					},
					Expr:   intstr.FromString(fmt.Sprintf("((sum by(persistentvolumeclaim, namespace) (kubelet_volume_stats_used_bytes{namespace=~'%[1]s.*'})) / (sum by(persistentvolumeclaim, namespace) (kube_persistentvolumeclaim_resource_requests_storage_bytes{namespace=~'%[1]s.*'}))) * 100 > 85", nsPrefix)),
					For:    "15m",
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
				{
					Alert: "PVCStorageMetricsAvailable",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlClusterSchedulableResourcesLow,
						"message": "PVC storage metrics are not available",
					},
					Expr:   intstr.FromString("absent(kubelet_volume_stats_available_bytes) == 1 or absent(kubelet_volume_stats_capacity_bytes) == 1 or absent(kubelet_volume_stats_used_bytes) == 1 or absent(kube_persistentvolumeclaim_resource_requests_storage_bytes) == 1"),
					For:    "15m",
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
				{
					Alert: "KubePersistentVolumeFillingUp4h",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlKubePersistentVolumeFillingUp4h,
						"message": "Based on recent sampling, the PersistentVolume claimed by {{ $labels.persistentvolumeclaim }} in Namespace {{ $labels.namespace }} is expected to fill up within four days. Currently {{ $value | humanizePercentage }} is available.",
					},
					Expr:   intstr.FromString(fmt.Sprintf("(predict_linear(kubelet_volume_stats_available_bytes{job='kubelet', namespace=~'%[1]s.*'} [6h], 4 * 24 * 3600) <= 0 and kubelet_volume_stats_available_bytes{job='kubelet', namespace=~'%[1]s.*'}  / kubelet_volume_stats_capacity_bytes{job='kubelet'} < 0.15)", nsPrefix)),
					For:    "15m",
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
				{
					Alert: "KubePersistentVolumeFillingUp",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlKubePersistentVolumeFillingUp,
						"message": "The PersistentVolume claimed by {{ $labels.persistentvolumeclaim }} in Namespace {{ $labels.namespace }} is only {{ $value | humanizePercentage }} free.",
					},
					Expr:   intstr.FromString(fmt.Sprintf("(kubelet_volume_stats_available_bytes{job='kubelet', namespace=~'%[1]s.*'} / kubelet_volume_stats_capacity_bytes{job='kubelet', namespace=~'%[1]s.*'} < 0.03)", nsPrefix)),
					For:    "15m",
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},

				{
					Alert: "PersistentVolumeErrors",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlPersistentVolumeErrors,
						"message": "The PVC {{  $labels.persistentvolumeclaim  }} is in status {{  $labels.phase  }} in namespace {{  $labels.namespace }} ",
					},
					Expr:   intstr.FromString(fmt.Sprintf("(sum by(persistentvolumeclaim, namespace, phase) (kube_persistentvolumeclaim_status_phase{phase=~'Failed|Pending|Lost', namespace=~'%s.*'})) > 0", nsPrefix)),
					For:    "15m",
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
			},
		},
		{
			AlertName: "obo-ksm-endpoint-alerts",
			Namespace: namespace,
			GroupName: "obo-endpoint.rules",
			Rules: []monv1.Rule{
				{
					Alert: "RHOAMOboAlertmanagerOperatedServiceEndpointDown",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlEndpointAvailableAlert,
						"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", namespace),
					},
					Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='alertmanager-operated', namespace='%s'} < 1", namespace)),
					For:    "5m",
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
				{
					Alert: "RHOAMOboAlertmanagerServiceEndpointDown",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlEndpointAvailableAlert,
						"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", namespace),
					},
					Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='rhoam-alertmanager', namespace='%s'} < 1", namespace)),
					For:    "5m",
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
				{
					Alert: "RHOAMOboPrometheusOperatedServiceEndpointDown",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlEndpointAvailableAlert,
						"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", namespace),
					},
					Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='prometheus-operated', namespace='%s'} < 1", namespace)),
					For:    "5m",
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
				{
					Alert: "RHOAMOboPrometheusServiceEndpointDown",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlEndpointAvailableAlert,
						"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", namespace),
					},
					Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='rhoam-prometheus', namespace='%s'} < 1", namespace)),
					For:    "5m",
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
				{
					Alert: "RHOAMOboBlackboxExporterEndpointDown",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlEndpointAvailableAlert,
						"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", namespace),
					},
					Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='blackbox-exporter', namespace='%s'} < 1", namespace)),
					For:    "5m",
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
			},
		},
		{
			AlertName: "install-upgrade-alerts",
			Namespace: namespace,
			GroupName: "general.rules",
			Rules: []monv1.Rule{
				{
					Alert: "RHOAMCSVRequirementsNotMet",
					Annotations: map[string]string{
						"message": "RequirementsNotMet for CSV '{{$labels.name}}' in namespace '{{$labels.exported_namespace}}'. Phase is not succeeded",
					},
					Expr:   intstr.FromString(fmt.Sprintf(`csv_succeeded{exported_namespace=~"%s.*"} != 1`, nsPrefix)),
					For:    "15m",
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
			},
		},
		{
			AlertName: "multi-az-pod-distribution",
			Namespace: namespace,
			GroupName: "general.rules",
			Rules: []monv1.Rule{
				{
					Alert: "MultiAZPodDistribution",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlPodDistributionIncorrect,
						"message": "Pod {{  $labels.namespace  }} / {{  $labels.pod  }} ({{  $labels.container  }}) is incorrectly distributed to the zone {{  $value  }} ; for the last 5 minutes",
					},
					Expr:   intstr.FromString(installationName + "_version{to_version=\"\"} and (count by(namespace, created_by_name, label_topology_kubernetes_io_zone) (kube_pod_info{namespace=~'" + nsPrefix + ".*', created_by_kind!=\"<none>\"} == on(node) group_left(label_topology_kubernetes_io_zone) kube_node_labels) == on (namespace, created_by_name)(count by(namespace, created_by_name) (kube_pod_info{namespace=~'" + nsPrefix + ".*', created_by_kind!=\"<none>\"}) > 1) > scalar((count(count by (label_topology_kubernetes_io_zone) (kube_node_labels)) >= bool 2) == 1))"),
					For:    "5m",
					Labels: map[string]string{"severity": "warning", "product": installationName},
				},
			},
		},
		{
			AlertName: "test-alerts",
			Namespace: namespace,
			GroupName: "test.rules",
			Rules: []monv1.Rule{
				{
					Alert: "TestFireCriticalAlert",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlTestFireAlerts,
						"message": "This is occasional Test Fire alert from Team SRE",
					},
					Expr:   intstr.FromString("count(kube_secret_info{namespace='" + namespace + "', secret='cj3cssrec'}) > 0"),
					For:    "10s",
					Labels: map[string]string{"severity": "critical", "product": "secret"},
				},
				{
					Alert: "TestFireWarningAlert",
					Annotations: map[string]string{
						"sop_url": resources.SopUrlTestFireAlerts,
						"message": "This is occasional Test Fire alert from Team SRE",
					},
					Expr:   intstr.FromString("count(kube_secret_info{namespace='" + namespace + "', secret='wj3cssrew'}) > 0"),
					For:    "10s",
					Labels: map[string]string{"severity": "warning", "product": "secret"},
				},
			},
		},
		{
			AlertName: fmt.Sprintf("%s-rhmi-controller-alerts", installationName),
			Namespace: namespace,
			GroupName: fmt.Sprintf("%s-installation.rules", installationName),
			Rules: []monv1.Rule{
				{
					Alert: fmt.Sprintf("%sIsInReconcilingErrorState", strings.ToUpper(installationName)),
					Annotations: map[string]string{
						"sop_url": resources.SopUrlRHOAMIsInReconcilingErrorState,
						"message": fmt.Sprintf("%s operator has finished installing, but has been in a error state while reconciling for last 1 hour", strings.ToUpper(installationName)),
					},
					Expr:   intstr.FromString(fmt.Sprintf(`(%s_status{stage!="complete"} > 0) * on(pod) group_left(to_version, version) (%[1]s_version{to_version="",version=~".+"} > 0)`, installationName)),
					For:    "90m",
					Labels: map[string]string{"severity": "critical", "product": installationName},
				},
			},
		},

		{
			AlertName: fmt.Sprintf("%s-aws-postgres-version-updates-available", installationName),
			Namespace: namespace,
			GroupName: fmt.Sprintf("%s-installation.rules", installationName),
			Rules: []monv1.Rule{
				{
					Alert: fmt.Sprintf("%sRHOAMAwsPostgresVersionUpdatesAvailable", strings.ToUpper(installationName)),
					Annotations: map[string]string{
						"sop_url": resources.SopUrlRHOAMAwsPostgresVersionUpdatesAvailable,
						"message": "PostgreSQL upgrade(s) available for improved security and performance. Current version: {{ $labels.current_version }}, Available version: {{ $labels.available_version }}",
						"summary": "{{ $labels.instance_id }} has PostgreSQL upgrade(s) available",
						"description": "One or more PostgreSQL engine version upgrades are available for RDS instance {{ $labels.instance_id }}. Upgrading can provide security patches, performance improvements, and new features.",
					},
					Expr:   intstr.FromString("postgres_upgrade_available > 0"),
					For:    "10m",
					Labels: map[string]string{"severity": "info", "product": installationName, "alert_type": "upgrade_notification"},
				},
			},
		},
	}

	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installation.Spec.Type)) {
		multitenantAlert := resources.AlertConfiguration{
			AlertName: "multitenancy-api-management-tenant-alerts",
			Namespace: namespace,
			GroupName: "multitenancy-api-management-tenant.rules",
			Rules: []monv1.Rule{
				{
					Alert: "ApiManagementTenantCRFailed",
					Annotations: map[string]string{
						"sop_url": resources.SopApiManagementTenantCRFailed,
						"message": "An APIManagementTenant CR has failed to reconcile. See the labels for details.",
					},
					Expr:   intstr.FromString(`tenants_summary{provisioningStatus!="3scale account ready"} and tenants_summary{provisioningStatus != "won't provision"}`),
					For:    "10m",
					Labels: map[string]string{"severity": "critical", "product": installationName},
				},
			},
		}
		alerts = append(alerts, multitenantAlert)
	}

	alertsToBeRemoved := []resources.AlertConfiguration{
		{
			AlertName: fmt.Sprintf("%s-rhmi-controller-alerts", installationName),
			Namespace: namespace,
			GroupName: fmt.Sprintf("%s-installation.rules", installationName),
			Rules: []monv1.Rule{
				{
					Alert: fmt.Sprintf("%sInstallationControllerReconcileDelayed", strings.ToUpper(installationName)),
					Annotations: map[string]string{
						"sop_url": resources.SopUrlAlertsAndTroubleshooting,
						"message": fmt.Sprintf("The reconcile function of the installation controller in a completed state of %s operator is taking more than 2 hours to complete", strings.ToUpper(installationName)),
					},
					Expr:   intstr.FromString(installationName + `_version{version=~".+", to_version=""} * on(pod) (installation_controller_reconcile_delayed == 1)`),
					For:    "90m",
					Labels: map[string]string{"severity": "critical", "product": installationName},
				},
			},
		},
	}

	return &resources.AlertReconcilerImpl{
		ProductName:   "OBO",
		Installation:  installation,
		Log:           logger,
		Alerts:        alerts,
		RemovedAlerts: alertsToBeRemoved,
	}
}
