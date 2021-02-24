package monitoring

import (
	"fmt"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *Reconciler) newAlertsReconciler(isClusterMultiAZ bool, logger l.Logger, installType string) resources.AlertReconciler {
	installationName := resources.InstallationNames[installType]
	nsPrefix := r.installation.Spec.NamespacePrefix

	monitoringExpectedPodCount := 7
	if isClusterMultiAZ {
		monitoringExpectedPodCount = 11
	}

	return &resources.AlertReconcilerImpl{
		ProductName:  "monitoring",
		Installation: r.installation,
		Log:          logger,
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: "backup-monitoring-alerts",
				GroupName: "general.rules",
				Namespace: r.Config.GetOperatorNamespace(),
				Rules: []monitoringv1.Rule{
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
					{
						Alert: "CronJobsFailed",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "Job {{ $labels.namespace  }} / {{  $labels.job  }} has failed",
						},
						Expr:   intstr.FromString("clamp_max(max(kube_job_status_start_time * ON(job_name) GROUP_RIGHT() kube_job_labels{label_monitoring_key='middleware'} ) BY (job_name, label_cronjob_name, namespace) == ON(label_cronjob_name) GROUP_LEFT() max(kube_job_status_start_time * ON(job_name) GROUP_RIGHT() kube_job_labels{label_monitoring_key='middleware'}) BY (label_cronjob_name), 1) * ON(job_name) GROUP_LEFT() kube_job_status_failed > 0"),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
				},
			},

			{
				AlertName: "ksm-alerts",
				Namespace: r.Config.GetOperatorNamespace(),
				GroupName: "general.rules",
				Rules: []monitoringv1.Rule{
					{
						Alert: "KubePodCrashLooping",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "Pod {{  $labels.namespace  }} / {{  $labels.pod  }} ({{  $labels.container  }}) is restarting {{  $value  }} times every 10 minutes; for the last 15 minutes",
						},
						Expr:   intstr.FromString("rate(kube_pod_container_status_restarts_total{job='kube-state-metrics'}[10m]) * on (namespace) group_left(label_monitoring_key) kube_namespace_labels{label_monitoring_key='middleware'} * 60 * 5 > 0"),
						For:    "15m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "KubePodNotReady",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "Pod {{ $labels.namespace }} / {{ $labels.pod }}  has been in a non-ready state for longer than 15 minutes.",
						},
						Expr:   intstr.FromString("sum by(pod, namespace) (kube_pod_status_phase{phase=~'Pending|Unknown'} * on (namespace, namespace) group_left(label_monitoring_key) kube_namespace_labels{label_monitoring_key='middleware'}) > 0"),
						For:    "15m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "KubePodImagePullBackOff",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "Pod {{ $labels.namespace }} / {{  $labels.pod  }} has been unable to pull its image for longer than 5 minutes.",
						},
						Expr:   intstr.FromString("(kube_pod_container_status_waiting_reason{reason='ImagePullBackOff'} * on (namespace, namespace) group_left(label_monitoring_key) kube_namespace_labels{label_monitoring_key='middleware'}) > 0"),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "KubePodBadConfig",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": " Pod {{ $labels.namespace  }} / {{  $labels.pod  }} has been unable to start due to a bad configuration for longer than 5 minutes",
						},
						Expr:   intstr.FromString("(kube_pod_container_status_waiting_reason{reason='CreateContainerConfigError'} * on (namespace, namespace) group_left(label_monitoring_key) kube_namespace_labels{label_monitoring_key='middleware'}) > 0"),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "KubePodStuckCreating",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "Pod {{  $labels.namespace }} / {{  $labels.pod  }} has been trying to start for longer than 15 minutes - this could indicate a configuration error.",
						},
						Expr:   intstr.FromString("(kube_pod_container_status_waiting_reason{reason='ContainerCreating'} * on (namespace, namespace) group_left(label_monitoring_key) kube_namespace_labels{label_monitoring_key='middleware'}) > 0"),
						For:    "15m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "ClusterSchedulableMemoryLow",
						Annotations: map[string]string{
							"sop_url": "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/alerts/Cluster_Schedulable_Resources_Low.asciidoc",
							"message": "The cluster has {{  $value }} percent of memory requested and unavailable for scheduling for longer than 15 minutes.",
						},
						Expr:   intstr.FromString("((sum(sum by(node) (sum by(pod, node) (kube_pod_container_resource_requests_memory_bytes * on(node) group_left() (sum by(node) (kube_node_labels{label_node_role_kubernetes_io_compute='true'} == 1))) * on(pod) group_left() (sum by(pod) (kube_pod_status_phase{phase='Running'}) == 1)))) / ((sum((kube_node_labels{label_node_role_kubernetes_io_compute='true'} == 1) * on(node) group_left() (sum by(node) (kube_node_status_allocatable_memory_bytes)))))) * 100 > 85"),
						For:    "15m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "ClusterSchedulableCPULow",
						Annotations: map[string]string{
							"sop_url": "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/alerts/Cluster_Schedulable_Resources_Low.asciidoc",
							"message": "The cluster has {{ $value }} percent of CPU cores requested and unavailable for scheduling for longer than 15 minutes.",
						},
						Expr:   intstr.FromString("((sum(sum by(node) (sum by(pod, node) (kube_pod_container_resource_requests_cpu_cores * on(node) group_left() (sum by(node) (kube_node_labels{label_node_role_kubernetes_io_compute='true'} == 1))) * on(pod) group_left() (sum by(pod) (kube_pod_status_phase{phase='Running'}) == 1)))) / ((sum((kube_node_labels{label_node_role_kubernetes_io_compute='true'} == 1) * on(node) group_left() (sum by(node) (kube_node_status_allocatable_cpu_cores)))))) * 100 > 85"),
						For:    "15m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "PVCStorageAvailable",
						Annotations: map[string]string{
							"sop_url": "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/alerts/Cluster_Schedulable_Resources_Low.asciidoc",
							"message": "The {{  $labels.persistentvolumeclaim  }} PVC has has been {{ $value }} percent full for longer than 15 minutes.",
						},
						Expr:   intstr.FromString("((sum by(persistentvolumeclaim, namespace) (kubelet_volume_stats_used_bytes) * on ( namespace) group_left(label_monitoring_key) kube_namespace_labels{label_monitoring_key='middleware'}) / (sum by(persistentvolumeclaim, namespace) (kube_persistentvolumeclaim_resource_requests_storage_bytes) * on ( namespace) group_left(label_monitoring_key) kube_namespace_labels{label_monitoring_key='middleware'})) * 100 > 85"),
						For:    "15m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "PVCStorageMetricsAvailable",
						Annotations: map[string]string{
							"sop_url": "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/alerts/Cluster_Schedulable_Resources_Low.asciidoc",
							"message": "PVC storage metrics are not available",
						},
						Expr:   intstr.FromString("absent(kubelet_volume_stats_available_bytes) == 1 or absent(kubelet_volume_stats_capacity_bytes) == 1 or absent(kubelet_volume_stats_used_bytes) == 1 or absent(kube_persistentvolumeclaim_resource_requests_storage_bytes) == 1"),
						For:    "15m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "KubePersistentVolumeFillingUp",
						Annotations: map[string]string{
							"sop_url": "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/alerts/pvc_storage.asciidoc#pvcstoragewillfillin4hours",
							"message": "Based on recent sampling, the PersistentVolume claimed by {{ $labels.persistentvolumeclaim }} in Namespace {{ $labels.namespace }} is expected to fill up within four days. Currently {{ $value | humanizePercentage }} is available.",
						},
						Expr:   intstr.FromString("(predict_linear(kubelet_volume_stats_available_bytes{job='kubelet'}[6h], 4 * 24 * 3600) <= 0 and kubelet_volume_stats_available_bytes{job='kubelet'} / kubelet_volume_stats_capacity_bytes{job='kubelet'} < 0.15) * on(namespace) group_left(label_monitoring_key) kube_namespace_labels{label_monitoring_key='middleware'}"),
						For:    "15m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "KubePersistentVolumeFillingUp",
						Annotations: map[string]string{
							"sop_url": "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/alerts/pvc_storage.asciidoc#kubepersistentvolumefillingup",
							"message": "The PersistentVolume claimed by {{ $labels.persistentvolumeclaim }} in Namespace {{ $labels.namespace }} is only {{ $value | humanizePercentage }} free.",
						},
						Expr:   intstr.FromString("(kubelet_volume_stats_available_bytes{job='kubelet'} / kubelet_volume_stats_capacity_bytes{job='kubelet'} < 0.03) * on(namespace) group_left(label_monitoring_key) kube_namespace_labels{label_monitoring_key='middleware'}"),
						For:    "15m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},

					{
						Alert: "PersistentVolumeErrors",
						Annotations: map[string]string{
							"sop_url": "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/2.x/alerts/pvc_storage.asciidoc#persistentvolumeerrors",
							"message": "The PVC {{  $labels.persistentvolumeclaim  }} is in status {{  $labels.phase  }} in namespace {{  $labels.namespace }} ",
						},
						Expr:   intstr.FromString("(sum by(persistentvolumeclaim, namespace, phase) (kube_persistentvolumeclaim_status_phase{phase=~'Failed|Pending|Lost'}) * on ( namespace) group_left(label_monitoring_key) kube_namespace_labels{label_monitoring_key='middleware'}) > 0"),
						For:    "15m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
				},
			},

			{
				AlertName: "ksm-monitoring-alerts",
				Namespace: r.Config.GetOperatorNamespace(),
				GroupName: "general.rules",
				Rules: []monitoringv1.Rule{
					{
						Alert: "MiddlewareMonitoringPodCount",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": fmt.Sprintf("Pod count for namespace {{ $labels.namespace }} is {{ $value }}. Expected exactly %d pods.", monitoringExpectedPodCount),
						},
						Expr: intstr.FromString(fmt.Sprintf("(1 - absent(kube_pod_status_ready{condition='true',namespace='"+nsPrefix+"middleware-monitoring-operator'})) or sum(kube_pod_status_ready{condition='true',namespace='"+nsPrefix+"middleware-monitoring-operator'}) != %d", monitoringExpectedPodCount)), For: "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
				},
			},

			{
				AlertName: "ksm-endpoint-alerts",
				Namespace: r.Config.GetOperatorNamespace(),
				GroupName: "middleware-monitoring-operator-endpoint.rules",
				Rules: []monitoringv1.Rule{
					{
						Alert: "RHMIMiddlewareMonitoringOperatorAlertmanagerOperatedServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='alertmanager-operated'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHMIMiddlewareMonitoringOperatorAlertmanagerServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='alertmanager-service'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHMIMiddlewareMonitoringOperatorApplicationMonitoringMetricsServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='application-monitoring-operator-metrics'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHMIMiddlewareMonitoringOperatorGrafanaServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='grafana-service'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHMIMiddlewareMonitoringOperatorPrometheusOperatedServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='prometheus-operated'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHMIMiddlewareMonitoringOperatorPrometheusServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='prometheus-service'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHMIMiddlewareMonitoringOperatorRhmiRegistryCsServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='rhmi-registry-cs', namespace='%s'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
				},
			},
			{
				AlertName: "install-upgrade-alerts",
				Namespace: r.Config.GetOperatorNamespace(),
				GroupName: "general.rules",
				Rules: []monitoringv1.Rule{
					{
						Alert: "RHMICSVRequirementsNotMet",
						Annotations: map[string]string{
							"message": "RequirementsNotMet for CSV '{{$labels.name}}' in namespace '{{$labels.namespace}}'. Phase is {{$labels.phase}}",
						},
						Expr:   intstr.FromString(fmt.Sprintf("csv_abnormal{phase=~'Pending|Failed',exported_namespace=~'%s.*'}", r.Config.GetNamespacePrefix())),
						For:    "15m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
				},
			},
			{
				AlertName: "multi-az-pod-distribution",
				Namespace: r.Config.GetOperatorNamespace(),
				GroupName: "general.rules",
				Rules: []monitoringv1.Rule{
					{
						Alert: "MultiAZPodDistribution",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlPodDistributionIncorrect,
							"message": "Pod {{  $labels.namespace  }} / {{  $labels.pod  }} ({{  $labels.container  }}) is incorretly distributed to the zone {{  $value  }} ; for the last 5 minutes",
						},
						Expr:   intstr.FromString("count by(namespace, created_by_name, label_topology_kubernetes_io_zone) (kube_pod_info{namespace=~'" + nsPrefix + ".*', created_by_kind!=\"<none>\"} == on(node) group_left(label_topology_kubernetes_io_zone) kube_node_labels) == on (namespace, created_by_name)(count by(namespace, created_by_name) (kube_pod_info{namespace=~'" + nsPrefix + ".*', created_by_kind!=\"<none>\"}) > 1) > scalar((count(count by (label_topology_kubernetes_io_zone) (kube_node_labels)) >= bool 2) == 1)"),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
				},
			},
		},
	}
}
