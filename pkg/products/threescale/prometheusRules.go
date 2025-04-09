package threescale

import (
	"context"
	"fmt"
	customDomain "github.com/integr8ly/integreatly-operator/pkg/resources/custom-domain"
	"net/http"
	"strings"

	"github.com/integr8ly/integreatly-operator/pkg/metrics"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	"github.com/integr8ly/integreatly-operator/pkg/resources"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Reconciler) newAlertReconciler(logger l.Logger, installType string, ctx context.Context, serverClient k8sclient.Client, namespace string) (resources.AlertReconciler, error) {
	installationName := resources.InstallationNames[installType]

	//clusterVersion
	containerCpuMetric, err := metrics.GetContainerCPUMetric(ctx, serverClient, logger)
	if err != nil {
		return nil, err
	}

	alertNamePrefix := "3scale-"
	operatorAlertNamePrefix := "3scale-operator-"

	missingMetricsExpr := intstr.FromString(`absent(threescale_portals) == 1`)
	if customDomain.IsCustomDomain(r.installation) {
		missingMetricsExpr = intstr.FromString(fmt.Sprintf(`absent(threescale_portals) OR absent(%s_custom_domain) == 1`, installationName))
	}

	return &resources.AlertReconcilerImpl{
		Installation: r.installation,
		Log:          logger,
		ProductName:  "3scale",
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: alertNamePrefix + "ksm-endpoint-alerts",
				GroupName: " 3scale-endpoint.rules",
				Namespace: namespace,
				Rules: []monv1.Rule{
					{
						Alert: "RHOAMThreeScaleApicastProductionServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlRHOAMThreeScaleApicastProductionServiceEndpointDown,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='apicast-production', namespace='%s'} < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "RHOAMThreeScaleApicastStagingServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlRHOAMThreeScaleApicastStagingServiceEndpointDown,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='apicast-staging', namespace='%s'} < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "RHOAMThreeScaleBackendListenerServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlRHOAMThreeScaleBackendListenerServiceEndpointDown,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='backend-listener', namespace='%s'} < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "RHOAMThreeScaleSystemDeveloperServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='system-developer', namespace='%s'} < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHOAMThreeScaleSystemMasterServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='system-master', namespace='%s'} < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHOAMThreeScaleSystemMemcacheServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='system-memcache', namespace='%s'} < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHOAMThreeScaleSystemProviderServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No endpoints available for the {{  $labels.endpoint  }} service in the %s namespace", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='system-provider', namespace='%s'} < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHOAMThreeScaleSystemSearchdServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No endpoints available for the {{  $labels.endpoint  }} service in the %s namespace", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='system-searchd', namespace='%s'} < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHOAMThreeScaleZyncServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlRHOAMThreeScaleZyncServiceEndpointDown,
							"message": fmt.Sprintf("No endpoints available for the {{  $labels.endpoint  }} service in the %s namespace", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='zync', namespace='%s'} < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "RHOAMThreeScaleZyncDatabaseServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlRHOAMThreeScaleZyncDatabaseServiceEndpointDown,
							"message": fmt.Sprintf("No endpoints available for the {{  $labels.endpoint  }} service in the %s namespace", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='zync-database', namespace='%s'} < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
				},
			},
			{
				AlertName: operatorAlertNamePrefix + "ksm-endpoint-alerts",
				GroupName: " 3scale-operator-endpoint.rules",
				Namespace: namespace,
				Rules: []monv1.Rule{
					{
						Alert: "RHOAMThreeScaleOperatorRhmiRegistryCsServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='rhmi-registry-cs', namespace='%s'} < 1", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "RHOAMThreeScaleOperatorServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='threescale-operator-controller-manager-metrics-service', namespace='%s'} < 1", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
				},
			},
			{
				AlertName: alertNamePrefix + "ksm-3scale-user-alerts",
				GroupName: "general.rules",
				Namespace: namespace,
				Rules: []monv1.Rule{
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
				AlertName: alertNamePrefix + "ksm-3scale-alerts",
				GroupName: "general.rules",
				Namespace: namespace,
				Rules: []monv1.Rule{
					{
						Alert: "ThreeScaleApicastStagingPod",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "3Scale apicast-staging has no pods in a ready state.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("sum(kube_pod_status_ready{condition='true',namespace='%[1]v', pod=~'apicast-staging.*'} * on(pod, namespace) group_left() kube_pod_status_phase{phase='Running'}) < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "ThreeScaleApicastProductionPod",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlApicastProductionPodsDown,
							"message": "3Scale apicast-production has no pods in a ready state.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("sum(kube_pod_status_ready{condition='true',namespace='%[1]v', pod=~'apicast-production.*'} * on(pod, namespace) group_left() kube_pod_status_phase{phase='Running'}) < 1", r.Config.GetNamespace())),
						For:    "15m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "ThreeScaleBackendWorkerPod",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlThreeScaleBackendWorkerPod,
							"message": "3Scale backend-worker has no pods in a ready state.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("sum(kube_pod_status_ready{condition='true',namespace='%[1]v', pod=~'backend-worker.*'} * on(pod, namespace) group_left() kube_pod_status_phase{phase='Running'}) < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "ThreeScaleBackendListenerPod",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "3Scale backend-listener has no pods in a ready state",
						},
						Expr:   intstr.FromString(fmt.Sprintf("sum(kube_pod_status_ready{condition='true',namespace='%[1]v', pod=~'backend-listener.*'} * on(pod, namespace) group_left() kube_pod_status_phase{phase='Running'}) < 1", r.Config.GetNamespace())),
						For:    "15m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "ThreeScaleSystemAppPod",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlSystemAppPodsDown,
							"message": "3Scale system-app has no pods in a ready state.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("sum(kube_pod_status_ready{condition='true', namespace='%[1]v', pod=~'system-app.*'} * on(pod, namespace) group_left() kube_pod_status_phase{phase='Running'}) < 1", r.Config.GetNamespace())),
						For:    "15m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "ThreeScaleAdminUIBBT",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlThreeScaleAdminUIBBT,
							"message": "3Scale Admin UI Blackbox Target: If this console is unavailable, the client is unable to configure or administer their API setup.",
						},
						Expr:   intstr.FromString("probe_success{job='blackbox', service='3scale-admin-ui'} != 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "ThreeScaleDeveloperUIBBT",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlThreeScaleDeveloperUIBBT,
							"message": "3Scale Developer UI Blackbox Target: If this console is unavailable, the clients developers are unable signup or perform API management.",
						},
						Expr:   intstr.FromString("probe_success{job='blackbox',service='3scale-developer-console-ui'} != 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "ThreeScaleSystemAdminUIBBT",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlThreeScaleSystemAdminUIBBT,
							"message": "3Scale System Admin UI Blackbox Target: If this console is unavailable, the client is unable to perform Account Management,Analytics or Billing.",
						},
						Expr:   intstr.FromString("probe_success{job='blackbox', service='3scale-system-admin-ui'} == 0 and up{job='blackbox', service='3scale-system-admin-ui'} ==1"),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "ThreeScaleZyncPodAvailability",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": fmt.Sprintf("3Scale Zync has no pods in a ready state. Expected number of pods for zync: %d, and for zync-que: %d.", r.Config.GetReplicasConfig(r.installation)["zyncApp"], r.Config.GetReplicasConfig(r.installation)["zyncQue"]),
						},
						Expr:   intstr.FromString(fmt.Sprintf("sum(kube_pod_status_ready{condition='true', namespace='%[1]v', pod=~'zync.*', pod!~'zync-database-.*'} * on(pod, namespace) group_left() kube_pod_status_phase{phase='Running'}) < 1", r.Config.GetNamespace(), r.Config.GetReplicasConfig(r.installation)["zyncApp"])),
						For:    "15m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "ThreeScaleZyncDatabasePodAvailability",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "3Scale Zync-database has no pods in a ready state. Expected a minimum of 1 pod.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("sum(kube_pod_status_ready{condition='true', namespace='%[1]v', pod=~'zync-database.*'} * on(pod, namespace) group_left() kube_pod_status_phase{phase='Running'}) < 1", r.Config.GetNamespace())),
						For:    "15m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					alertThreeScaleContainerHighMemory(installationName, r.Config.GetNamespace()),
					{
						Alert: "ThreeScaleContainerHighCPU",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "The {{  $labels.container  }} Container in the {{  $labels.pod }} Pod has been using {{  $value  }}% of available CPU for longer than 15 minutes.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("sum("+containerCpuMetric+"{namespace='%[1]v'}) by (container, pod) / sum(kube_pod_container_resource_requests{namespace='%[1]v', resource='cpu'}) by (container, pod) * 100 > 90", r.Config.GetNamespace())),
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
			{
				AlertName: fmt.Sprintf("%s-missing-metrics", installationName),
				Namespace: namespace,
				GroupName: fmt.Sprintf("%s-general.rules", installationName),
				Rules: []monv1.Rule{
					{
						Alert: fmt.Sprintf("%sThreescaleCriticalMetricsMissing", strings.ToUpper(installationName)),
						Annotations: map[string]string{
							"sop_url": resources.SopUrlCriticalMetricsMissing,
							"message": "one or more critical metrics have been missing for 10+ minutes",
						},
						Expr:   missingMetricsExpr,
						For:    "30m",
						Labels: map[string]string{"severity": "critical"},
					},
				},
			},
			{
				AlertName: fmt.Sprintf("%s-custom-domain-alert", installationName),
				Namespace: namespace,
				GroupName: fmt.Sprintf("%s-custom-domaim.rules", installationName),
				Rules: []monv1.Rule{
					{
						Alert: "CustomDomainCRErrorState",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlRHOAMServiceDefinition,
							"message": "Error configuring custom domain, please refer to the documentation to resolve the error.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("%s_custom_domain{active='true'} > 0", installationName)),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "DnsBypassThreeScaleAdminUI",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlDnsBypassThreeScaleAdminUI,
							"message": "3Scale Admin UI, bypassing DNS: If this console is unavailable, the client is unable to configure or administer their API setup.",
						},
						Expr:   intstr.FromString("threescale_portals{system_master='false'} > 0"),
						For:    "15m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "DnsBypassThreeScaleDeveloperUI",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlDnsBypassThreeScaleDeveloperUI,
							"message": "3Scale Developer UI, bypassing DNS: If this console is unavailable, the client developers are unable signup or perform API management.",
						},
						Expr:   intstr.FromString("threescale_portals{system_developer='false'} > 0"),
						For:    "15m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "DnsBypassThreeScaleSystemAdminUI",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlDnsBypassThreeScaleSystemAdminUI,
							"message": "3Scale System Admin UI, bypassing DNS: If this console is unavailable, the client is unable to perform Account Management, Analytics or Billing.",
						},
						Expr:   intstr.FromString("threescale_portals{system_provider='false'} > 0"),
						For:    "15m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
				},
			},
		},
	}, nil
}

func alertThreeScaleContainerHighMemory(installationName string, namespace string) monv1.Rule {
	return monv1.Rule{
		Alert: "ThreeScaleContainerHighMemory",
		Annotations: map[string]string{
			"sop_url": resources.SopUrlAlertsAndTroubleshooting,
			"message": "The {{  $labels.container  }} Container in the {{  $labels.pod  }} Pod has been using {{  $value  }}% of available memory for longer than 15 minutes.",
		},
		Expr:   intstr.FromString(fmt.Sprintf("sum by(container, pod) (container_memory_usage_bytes{container!='', container!='system-provider', namespace='%[1]v'}) / sum by(container, pod) (kube_pod_container_resource_limits{namespace='%[1]v',resource='memory'}) * 100 > 90", namespace)),
		For:    "15m",
		Labels: map[string]string{"severity": "info", "product": installationName},
	}
}
