package grafana

import (
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func GrafanaAlertReconciler(logger l.Logger, installation *integreatlyv1alpha1.RHMI) resources.AlertReconciler {
	installationName := resources.InstallationNames[installation.Spec.Type]
	nsPrefix := installation.Spec.NamespacePrefix
	grafanaNamespace := nsPrefix + "customer-monitoring"

	alertNamePrefix := "customer-monitoring-"

	return &resources.AlertReconcilerImpl{
		Installation: installation,
		Log:          logger,
		ProductName:  "Grafana",
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: alertNamePrefix + "ksm-endpoint-alerts",
				GroupName: "grafana-operator-endpoint.rules",
				Namespace: grafanaNamespace,
				Rules: []monv1.Rule{
					{
						Alert: "GrafanaServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", grafanaNamespace),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='grafana-service', namespace='%s'} < 1", grafanaNamespace)),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
				},
			},
			{
				AlertName: alertNamePrefix + "ksm-grafana-alerts",
				GroupName: "general.rules",
				Namespace: grafanaNamespace,
				Rules: []monv1.Rule{
					{
						Alert: "GrafanaServicePod",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "Grafana Service has no pods in ready state.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("sum(kube_pod_status_ready{condition='true',namespace='%[1]v', pod=~'grafana-deployment.*'} * on(pod, namespace) group_left() kube_pod_status_phase{phase='Running'}) < 1", grafanaNamespace)),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
				},
			},
		},
	}
}
