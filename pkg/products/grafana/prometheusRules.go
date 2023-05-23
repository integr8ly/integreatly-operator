package grafana

import (
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *Reconciler) newAlertReconciler(logger l.Logger, installType string, namespace string) resources.AlertReconciler {
	installationName := resources.InstallationNames[installType]

	alertNamePrefix := "customer-monitoring-"

	return &resources.AlertReconcilerImpl{
		Installation: r.installation,
		Log:          logger,
		ProductName:  "Grafana",
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: alertNamePrefix + "ksm-endpoint-alerts",
				GroupName: "grafana-operator-endpoint.rules",
				Namespace: namespace,
				Rules: []monv1.Rule{
					{
						Alert: "GrafanaOperatorRhmiRegistryCsServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='rhmi-registry-cs', namespace='%s'} < 1", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "GrafanaServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='grafana-service', namespace='%s'} < 1", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
				},
			},
			{
				AlertName: alertNamePrefix + "ksm-grafana-alerts",
				GroupName: "general.rules",
				Namespace: namespace,
				Rules: []monv1.Rule{
					{
						Alert: "GrafanaOperatorPod",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "Grafana Operator has no pods in ready state.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("sum(kube_pod_status_ready{condition='true',namespace='%[1]v', pod=~'grafana-operator.*'} * on(pod, namespace) group_left() kube_pod_status_phase{phase='Running'}) < 1", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
					{
						Alert: "GrafanaServicePod",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "Grafana Service has no pods in ready state.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("sum(kube_pod_status_ready{condition='true',namespace='%[1]v', pod=~'grafana-deployment.*'} * on(pod, namespace) group_left() kube_pod_status_phase{phase='Running'}) < 1", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
				},
			},
		},
	}
}
