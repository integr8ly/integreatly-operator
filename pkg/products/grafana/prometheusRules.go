package grafana

import (
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *Reconciler) newAlertReconciler() resources.AlertReconciler {
	return &resources.AlertReconcilerImpl{
		Installation: r.installation,
		Logger:       logger.NewLogger(),
		ProductName:  "Grafana",
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: "ksm-endpoint-alerts",
				GroupName: "grafana-operator-endpoint.rules",
				Namespace: r.Config.GetOperatorNamespace(),
				Rules: []monitoringv1.Rule{
					{
						Alert: "GrafanaOperatorRhmiRegistryCsServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='rhmi-registry-cs', namespace='%[1]v'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical"},
					},
					{
						Alert: "GrafanaServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='grafana-service'} * on(namespace) group_left() kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "critical"},
					},
				},
			},
			{
				AlertName: "ksm-grafana-alerts",
				GroupName: "general.rules",
				Namespace: r.Config.GetOperatorNamespace(),
				Rules: []monitoringv1.Rule{
					{
						Alert: "GrafanaOperatorPod",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "Grafana Operator has no pods in ready state.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("(1 - absent(kube_pod_status_ready{condition='true',namespace='%[1]v'} * on(pod, namespace) kube_pod_labels{label_name='grafana-operator',namespace='%[1]v'})) or count(kube_pod_status_ready{condition='true',namespace='%[1]v'} * on(pod, namespace) kube_pod_labels{label_name='grafana-operator',namespace='%[1]v'}) < 1", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical"},
					},
					{
						Alert: "GrafanaServicePod",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "Grafana Service has no pods in ready state.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("(1 - absent(kube_pod_status_ready{condition='true',namespace='%[1]v'} * on(pod, namespace) kube_pod_labels{label_app='grafana',namespace='%[1]v'})) or count(kube_pod_status_ready{condition='true',namespace=`%[1]v`} * on(pod, namespace) kube_pod_labels{label_app='grafana',namespace='%[1]v'}) < 1", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical"},
					},
				},
			},
		},
	}
}
