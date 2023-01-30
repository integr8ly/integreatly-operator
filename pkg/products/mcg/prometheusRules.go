package mcg

import (
	"fmt"

	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *Reconciler) newAlertReconciler(logger l.Logger, installType string) (resources.AlertReconciler, error) {
	installationName := resources.InstallationNames[installType]

	observabilityConfig, err := r.ConfigManager.ReadObservability()
	if err != nil {
		logger.Warning("failed to get observability config")
		return nil, err
	}

	namespace := observabilityConfig.GetNamespace()
	return &resources.AlertReconcilerImpl{
		Installation: r.installation,
		Log:          logger,
		ProductName:  "mcg",
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: "mcg-operator-ksm-endpoint-alerts",
				GroupName: "mcg-operator-endpoint.rules",
				Namespace: namespace,
				Rules: []monitoringv1.Rule{
					{
						Alert: "RHOAMMCGOperatorMetricsServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("absent(kube_endpoint_address{ready='true',namespace='%s', endpoint='noobaa-operator-service'})", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "RHOAMMCGOperatorRhmiRegistryCsServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("absent(kube_endpoint_address{ready='true',endpoint='rhmi-registry-cs', namespace=`%s`})", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
				},
			},
			{
				AlertName: "mcg-ksm-endpoint-alerts",
				GroupName: "general.rules",
				Namespace: namespace,
				Rules: []monitoringv1.Rule{
					{
						Alert: "NooBaaCorePod",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "MCG noobaa-core has no pods in a ready state.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("sum(kube_pod_status_ready{condition='true',namespace='%[1]s', pod=~'noobaa-core.*'} * on(pod, namespace) group_left() kube_pod_status_phase{phase='Running',namespace='%[1]s'}) < 1 OR absent(kube_pod_status_ready{condition='true',namespace='%[1]s',pod=~'noobaa-core.*'})", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "NooBaaDBPod",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "MCG noobaa-db has no pods in a ready state.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("sum(kube_pod_status_ready{condition='true',namespace='%[1]s', pod=~'noobaa-db.*'} * on(pod, namespace) group_left() kube_pod_status_phase{phase='Running',namespace='%[1]s'}) < 1 OR absent(kube_pod_status_ready{condition='true',namespace='%[1]s',pod=~'noobaa-db.*'})", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "NooBaaDefaultBackingStorePod",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "MCG noobaa-default-backing-store has no pods in a ready state.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("sum(kube_pod_status_ready{condition='true',namespace='%[1]s', pod=~'noobaa-default-backing-store.*'} * on(pod, namespace) group_left() kube_pod_status_phase{phase='Running',namespace='%[1]s'}) < 1 OR absent(kube_pod_status_ready{condition='true',namespace='%[1]s',pod=~'noobaa-default-backing-store.*'})", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "NooBaaEndpointPod",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "MCG noobaa-endpoint has no pods in a ready state.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("sum(kube_pod_status_ready{condition='true',namespace='%[1]s', pod=~'noobaa-endpoint.*'} * on(pod, namespace) group_left() kube_pod_status_phase{phase='Running',namespace='%[1]s'}) < 1 OR absent(kube_pod_status_ready{condition='true',namespace='%[1]s',pod=~'noobaa-endpoint.*'})", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "NooBaaS3Endpoint",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": "MCG s3 endpoint is not available.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("absent(kube_endpoint_address{ready='true',namespace='%[1]s', endpoint='s3'})", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "NooBaaBucketCapacityOver85Percent",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "MCG s3 bucket is over 85% capacity.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("NooBaa_bucket_capacity{namespace='%[1]s', bucket_name=~'%[2]s.*'} > 85", r.Config.GetOperatorNamespace(), threescaleBucket)),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "NooBaaBucketCapacityOver95Percent",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "MCG s3 bucket is over 95% capacity.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("NooBaa_bucket_capacity{namespace='%[1]s', bucket_name=~'%[2]s.*'} > 95", r.Config.GetOperatorNamespace(), threescaleBucket)),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
				},
			},
		},
	}, nil
}
