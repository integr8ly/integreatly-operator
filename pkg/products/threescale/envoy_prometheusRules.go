package threescale

import (
	"fmt"
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *Reconciler) newEnvoyAlertReconciler(logger l.Logger) resources.AlertReconciler {
	return &resources.AlertReconcilerImpl{
		Installation: r.installation,
		Log:          logger,
		ProductName:  "3scale",
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: "ksm-marin3r-alerts",
				GroupName: "general.rules",
				Namespace: r.Config.GetNamespace(),
				Rules: []monitoringv1.Rule{
					{
						Alert: "Marin3rEnvoyApicastStagingContainerDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlMarin3rEnvoyApicastStagingContainerDown,
							"message": "3Scale apicast-staging pods have no ratelimiting sidecar container attached.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("(1 - absent(kube_pod_container_status_running{container='envoy-sidecar'} * on (pod,namespace) kube_pod_labels{label_deploymentconfig='apicast-staging',namespace='%v'})) < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical"},
					},
					{
						Alert: "Marin3rEnvoyApicastProductionContainerDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlMarin3rEnvoyApicastProductionContainerDown,
							"message": "3Scale apicast-production pods have no ratelimiting sidecar container attached.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("(1 - absent(kube_pod_container_status_running{container='envoy-sidecar'} * on (pod,namespace) kube_pod_labels{label_deploymentconfig='apicast-production',namespace='%v'})) < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical"},
					},
				},
			},
		},
	}
}
