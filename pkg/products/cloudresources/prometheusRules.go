package cloudresources

import (
	"fmt"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *Reconciler) newAlertsReconciler(logger l.Logger, installType string) resources.AlertReconciler {
	installationName := resources.InstallationNames[installType]

	return &resources.AlertReconcilerImpl{
		ProductName:  "Cloud Resources Operator",
		Installation: r.installation,
		Log:          logger,
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: "ksm-endpoint-alerts",
				Namespace: r.Config.GetOperatorNamespace(),
				GroupName: "cloud-resources-operator-endpoint.rules",
				Rules: []monitoringv1.Rule{
					{
						Alert: "RHMICloudResourceOperatorMetricsServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlRHMICloudResourceOperatorMetricsServiceEndpointDown,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='cloud-resource-operator-metrics', namespace='%s'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					}, {
						Alert: "RHMICloudResourceOperatorRhmiRegistryCsServiceEndpointDown",
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
		},
	}
}
