package apicurito

import (
	"fmt"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *Reconciler) newAlertsReconciler(logger l.Logger) resources.AlertReconciler {
	const apicuritoPodCountExpected int = 3

	return &resources.AlertReconcilerImpl{
		ProductName:  "Apicurito",
		Installation: r.installation,
		Log:          logger,
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: "ksm-apicurito-alerts",
				Namespace: r.Config.GetNamespace(),
				GroupName: "apicurito.rules",
				Rules: []monitoringv1.Rule{
					{
						Alert: "ApicuritoPodCount",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": fmt.Sprintf("Pod count for namespace %s is %s. Expected exactly %d pods.", "{{ $labels.namespace }}", "{{  printf \"%.0f\" $value }}", apicuritoPodCountExpected),
						},
						Expr: intstr.FromString(
							fmt.Sprintf("(1-absent(kube_pod_status_ready{condition='true', namespace='%s'})) or sum(kube_pod_status_ready{condition='true', namespace='%s'}) != %d", r.Config.GetNamespace(), r.Config.GetNamespace(), apicuritoPodCountExpected)),
						For:    "5m",
						Labels: map[string]string{"severity": "critical"},
					},
				},
			},

			{
				AlertName: "ksm-endpoint-alerts",
				Namespace: r.Config.GetNamespace(),
				GroupName: "apicurito-endpoint.rules",
				Rules: []monitoringv1.Rule{
					{
						Alert: "RHMIApicuritoServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='apicurito'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "critical"},
					},
					{
						Alert: "RHMIApicuritoFuseApicuritoGeneratorServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='fuse-apicurito-generator'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
						For:    "5m",
						Labels: map[string]string{"severity": "critical"},
					},
				},
			},

			{
				AlertName: "ksm-endpoint-alerts",
				GroupName: "apicurito-operator-endpoint.rules",
				Namespace: r.Config.GetOperatorNamespace(),
				Rules: []monitoringv1.Rule{
					{
						Alert: "RHMIApicuritoOperatorRhmiRegistryCsServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='rhmi-registry-cs', namespace=`%s`} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning"},
					},
				},
			},
		},
	}
}
