package threescale

import (
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *Reconciler) newEnvoyAlertReconciler(logger l.Logger, installType string, namespace string) resources.AlertReconciler {
	installationName := resources.InstallationNames[installType]
	alertName := "3scale-ksm-marin3r-alerts"

	return &resources.AlertReconcilerImpl{
		Installation: r.installation,
		Log:          logger,
		ProductName:  "3scale",
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: alertName,
				GroupName: "general.rules",
				Namespace: namespace,
				Rules: []monv1.Rule{
					{
						Alert: "Marin3rEnvoyApicastStagingContainerDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlMarin3rEnvoyApicastStagingContainerDown,
							"message": "3Scale apicast-staging pods have no ratelimiting sidecar container attached.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("(sum(kube_pod_container_status_ready{container='envoy-sidecar'} * on (pod,namespace) kube_pod_labels{label_deployment='apicast-staging',namespace='%v'})) < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
					{
						Alert: "Marin3rEnvoyApicastProductionContainerDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlMarin3rEnvoyApicastProductionContainerDown,
							"message": "3Scale apicast-production pods have no ratelimiting sidecar container attached.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("(sum(kube_pod_container_status_ready{container='envoy-sidecar'} * on (pod,namespace) kube_pod_labels{label_deployment='apicast-production',namespace='%v'})) < 1", r.Config.GetNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
				},
			},
		},
	}
}
