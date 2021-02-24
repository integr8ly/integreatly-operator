package threescale

import (
	"fmt"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"net/http"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *Reconciler) newAlertReconciler(logger l.Logger, installType string) resources.AlertReconciler {
	installationName := resources.InstallationNames[installType]

	return &resources.AlertReconcilerImpl{
		Installation: r.installation,
		Log:          logger,
		ProductName:  "3scale",
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: "ksm-endpoint-alerts",
				GroupName: " 3scale-endpoint.rules",
				Namespace: r.Config.GetNamespace(),
				Rules: []monitoringv1.Rule{
					{
						Alert: "RHMIThreeScaleApicastProductionServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlRHMIThreeScaleApicastProductionServiceEndpointDown,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='apicast-production'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "RHMIThreeScaleApicastStagingServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlRHMIThreeScaleApicastStagingServiceEndpointDown,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='apicast-staging'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "RHMIThreeScaleBackendListenerServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlRHMIThreeScaleBackendListenerServiceEndpointDown,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='backend-listener'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "RHMIThreeScaleSystemDeveloperServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='system-developer'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHMIThreeScaleSystemMasterServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='system-master'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHMIThreeScaleSystemMemcacheServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='system-memcache'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHMIThreeScaleSystemProviderServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No endpoints available for the {{  $labels.endpoint  }} service in the %s namespace", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='system-provider'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHMIThreeScaleSystemSphinxServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No endpoints available for the {{  $labels.endpoint  }} service in the %s namespace", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='system-sphinx'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHMIThreeScaleZyncServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlRHMIThreeScaleZyncServiceEndpointDown,
							"message": fmt.Sprintf("No endpoints available for the {{  $labels.endpoint  }} service in the %s namespace", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='zync'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "RHMIThreeScaleZyncDatabaseServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlRHMIThreeScaleZyncDatabaseServiceEndpointDown,
							"message": fmt.Sprintf("No endpoints available for the {{  $labels.endpoint  }} service in the %s namespace", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='zync-database'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
				},
			},

			{
				AlertName: "ksm-endpoint-alerts",
				GroupName: " 3scale-operator-endpoint.rules",
				Namespace: r.Config.GetOperatorNamespace(),
				Rules: []monitoringv1.Rule{
					{
						Alert: "RHMIThreeScaleOperatorRhmiRegistryCsServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='rhmi-registry-cs', namespace=`%s`} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHMIThreeScaleOperatorServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='threescale-operator'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
				},
			},
			{
				AlertName: "ksm-3scale-user-alerts",
				GroupName: "general.rules",
				Namespace: r.Config.GetNamespace(),
				Rules: []monitoringv1.Rule{
					{
						Alert: "ThreeScaleUserCreationFailed",
						Annotations: map[string]string{
							"message": "3Scale user creation failed for user {{  $labels.username  }}",
						},
						Expr:   intstr.FromString(fmt.Sprintf("threescale_user_action{action='%s'} != %d", http.MethodPost, http.StatusCreated)),
						Labels: map[string]string{"severity": "warning"},
					},
					{
						Alert: "ThreeScaleUserDeletionFailed",
						Annotations: map[string]string{
							"message": "3Scale user deletion failed for user {{  $labels.username  }}",
						},
						Expr:   intstr.FromString(fmt.Sprintf("threescale_user_action{action='%s'} != %d", http.MethodDelete, http.StatusOK)),
						Labels: map[string]string{"severity": "warning"},
					},
				},
			},

			{
				AlertName: "ksm-3scale-alerts",
				GroupName: "general.rules",
				Namespace: r.Config.GetNamespace(),
				Rules: []monitoringv1.Rule{
					{
						Alert: "ThreeScaleApicastStagingPod",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "3Scale apicast-staging has no pods in a ready state.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("(1 - absent(kube_pod_status_ready{condition='true',namespace='%[1]v'} * on(pod, namespace) kube_pod_labels{label_deploymentconfig='apicast-staging',namespace='%[1]v' })) or count(kube_pod_status_ready{condition='true',namespace='%[1]v'} * on(pod, namespace) kube_pod_labels{label_deploymentconfig='apicast-staging',namespace='%[1]v'}) < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "ThreeScaleApicastProductionPod",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "3Scale apicast-production has no pods in a ready state.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("(1 - absent(kube_pod_status_ready{condition='true',namespace='%[1]v'} * on(pod, namespace) kube_pod_labels{label_deploymentconfig='apicast-production',namespace='%[1]v'})) or count(kube_pod_status_ready{condition='true',namespace='%[1]v'} * on(pod, namespace) kube_pod_labels{label_deploymentconfig='apicast-production',namespace='%[1]v'}) < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "ThreeScaleBackendWorkerPod",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlThreeScaleBackendWorkerPod,
							"message": "3Scale backend-worker has no pods in a ready state.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("(1 - absent(kube_pod_status_ready{condition='true',namespace='%[1]v'} * on(pod, namespace) kube_pod_labels{label_deploymentconfig='backend-worker',namespace='%[1]v'})) or count(kube_pod_status_ready{condition='true',namespace='%[1]v'} * on(pod, namespace) kube_pod_labels{label_deploymentconfig='backend-worker',namespace='%[1]v'}) < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "ThreeScaleBackendListenerPod",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "3Scale backend-listener has no pods in a ready state",
						},
						Expr:   intstr.FromString(fmt.Sprintf("(1 - absent(kube_pod_status_ready{condition='true',namespace='%[1]v'} * on(pod, namespace) kube_pod_labels{label_deploymentconfig='backend-listener',namespace='%[1]v'})) or count(kube_pod_status_ready{condition='true',namespace='%[1]v'} * on(pod, namespace) kube_pod_labels{label_deploymentconfig='backend-listener',namespace='%[1]v'}) < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "ThreeScaleSystemAppPod",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "3Scale system-app has no pods in a ready state.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("(1 - absent(kube_pod_status_ready{condition='true',namespace='%[1]v'} * on(pod, namespace) kube_pod_labels{label_deploymentconfig='system-app',namespace='%[1]v'})) or count(kube_pod_status_ready{condition='true',namespace='%[1]v'} * on(pod, namespace) kube_pod_labels{label_deploymentconfig='system-app',namespace='%[1]v'}) < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "ThreeScaleAdminUIBBT",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlThreeScaleAdminUIBBT,
							"message": "3Scale Admin UI Blackbox Target: If this console is unavailable,the client is unable to configure or administer their API setup.",
						},
						Expr:   intstr.FromString("absent(probe_success{job='blackbox', service='3scale-admin-ui'} == 1)"),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "ThreeScaleDeveloperUIBBT",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlThreeScaleDeveloperUIBBT,
							"message": "3Scale Developer UI Blackbox Target: If this console isunavailable, the clients developers are unable signup or perform API management.",
						},
						Expr:   intstr.FromString("absent(probe_success{job='blackbox',service='3scale-developer-console-ui'} == 1)"),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "ThreeScaleSystemAdminUIBBT",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlThreeScaleSystemAdminUIBBT,
							"message": "3Scale System Admin UI Blackbox Target: If this console is unavailable, the client is unable to perform Account Management,Analytics or Billing.",
						},
						Expr:   intstr.FromString("probe_success{job='blackbox', service='3scale-system-admin-ui'} == 0 and up{job='blackbox', service='3scale-system-admin-ui'} ==1"),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "ThreeScaleZyncPodAvailability",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": fmt.Sprintf("3Scale Zync has {{   $value  }} pods in a ready state. Expected %d of pods.", r.Config.GetReplicasConfig(r.installation)["zyncApp"]),
						},
						Expr:   intstr.FromString(fmt.Sprintf("(1-absent(kube_pod_status_ready{condition='true',namespace='%[1]v'} * on (pod, namespace) kube_pod_labels{namespace='%[1]v', label_threescale_component='zync', label_threescale_component_element='zync'})) or count(kube_pod_status_ready{condition='true',namespace='%[1]v'} * on (pod, namespace) kube_pod_labels{namespace='%[1]v', label_threescale_component='zync', label_threescale_component_element='zync'}) != %d", r.Config.GetNamespace(), r.Config.GetReplicasConfig(r.installation)["zyncApp"])),
						For:    "15m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "ThreeScaleZyncDatabasePodAvailability",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "3Scale Zync-database has {{  $value  }} pods in a ready state. Expected a minimum of 1 pod.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("(1-absent(kube_pod_status_ready{condition='true',namespace='%[1]v'} * on (pod, namespace) kube_pod_labels{namespace='%[1]v', label_threescale_component='zync', label_threescale_component_element='database'})) or count(kube_pod_status_ready{condition='true',namespace='%[1]v'} * on (pod, namespace) kube_pod_labels{namespace='%[1]v', label_threescale_component='zync', label_threescale_component_element='database'}) != 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "ThreeScaleContainerHighMemory",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "The {{  $labels.container  }} Container in the {{  $labels.pod  }} Pod has been using {{  $value  }}% of available memory for longer than 15 minutes.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("sum by(container, pod) (container_memory_usage_bytes{container!='', container!='system-provider', namespace='%[1]v'}) / sum by(container, pod) (kube_pod_container_resource_limits_memory_bytes{namespace='%[1]v'}) * 100 > 90", r.Config.GetNamespace())),
						For:    "15m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "ThreeScaleContainerHighCPU",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "The {{  $labels.container  }} Container in the {{  $labels.pod }} Pod has been using {{  $value  }}% of available CPU for longer than 15 minutes.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_rate{namespace='%[1]v'}) by (container, pod) / sum(kube_pod_container_resource_limits_cpu_cores{namespace='%[1]v'}) by (container, pod) * 100 > 90", r.Config.GetNamespace())),
						For:    "15m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "ThreescaleApicastWorkerRestart",
						Annotations: map[string]string{
							"summary":     "A new worker process in Nginx has been started",
							"description": "A new thread has been started. This could indicate that a worker process has died due to the memory limits being exceeded. Please investigate the memory pressure on pod (instance {{ $labels.instance }})",
							"sop_url":     "https://github.com/3scale/3scale-Operations/blob/master/sops/alerts/apicast_worker_restart.adoc",
						},
						Expr: intstr.FromString(fmt.Sprintf(`changes(worker_process{kubernetes_namespace='%s', kubernetes_pod_name=~'apicast-production.*'}[5m]) > 0`, r.Config.GetNamespace())),
						For:  "5m",
						Labels: map[string]string{
							"severity": "critical", "product": installationName,
						},
					},
				},
			},
		},
	}
}
