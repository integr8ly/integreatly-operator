package fuse

import (
	"fmt"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *Reconciler) newAlertsReconcilerInverted(installType string) resources.AlertReconciler {
	installationName := resources.InstallationNames[installType]

	return &resources.AlertReconcilerImpl{
		ProductName:  "Fuse",
		Installation: r.installation,
		Log:          r.log,
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: "ksm-endpoint-alerts",
				Namespace: r.Config.GetNamespace(),
				GroupName: "fuse-online-endpoint.rules",
				Rules: []monitoringv1.Rule{
					{
						Alert: "RHMIFuseOnlineBrokerAmqTcpServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlScaleDownFuse,
							"message": fmt.Sprintf("{{  $labels.endpoint  }} endpoints in namespace %s found. Expected none.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='broker-amq-tcp', namespace='%s'} > 0", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "RHMIFuseOnlineSyndesisMetaServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlScaleDownFuse,
							"message": fmt.Sprintf("{{  $labels.endpoint  }} endpoints in namespace %s found. Expected none.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='syndesis-meta', namespace='%s'} > 0", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "RHMIFuseOnlineSyndesisOauthproxyServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlScaleDownFuse,
							"message": fmt.Sprintf("{{  $labels.endpoint  }} endpoints in namespace %s found. Expected none.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='syndesis-oauthproxy', namespace='%s'} > 0", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "RHMIFuseOnlineSyndesisPrometheusServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlScaleDownFuse,
							"message": fmt.Sprintf("Endpoints in namespace %s. Expected none.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='syndesis-prometheus', namespace='%s'} > 0", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "RHMIFuseOnlineSyndesisServerServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlScaleDownFuse,
							"message": fmt.Sprintf("{{  $labels.endpoint  }} endpoints in namespace %s. Expected none.", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='syndesis-server', namespace='%s'} > 0", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "RHMIFuseOnlineSyndesisUiServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlScaleDownFuse,
							"message": fmt.Sprintf("Endpoints available for the {{  $labels.endpoint  }} service in the %s namespace", r.Config.GetNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='syndesis-ui', namespace='%s'} > 0", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
				},
			},
		},
	}
}
