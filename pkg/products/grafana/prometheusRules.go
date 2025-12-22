package grafana

import (
	"context"
	"fmt"

	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Reconciler) newAlertReconciler(logger l.Logger, installType string, namespace string) resources.AlertReconciler {
	installationName := resources.InstallationNames[installType]
	alertNamePrefix := "customer-monitoring-po-"

	return &resources.AlertReconcilerImpl{
		Installation: r.installation,
		Log:          logger,
		ProductName:  "Grafana",
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: alertNamePrefix + "ksm-endpoint-alerts",
				GroupName: "grafana-endpoint.rules",
				Namespace: namespace,
				Rules: []monv1.Rule{
					{
						Alert: "GrafanaServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", namespace),
						},
						Expr:   intstr.FromString(fmt.Sprintf("absent(kube_endpoint_address{endpoint='grafana-service', namespace='%s'})", namespace)),
						For:    resources.DurationPtr("5m"),
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
				},
			},
			{
				AlertName: alertNamePrefix + "ksm-grafana-alerts",
				GroupName: "grafana-general.rules",
				Namespace: namespace,
				Rules: []monv1.Rule{
					{
						Alert: "GrafanaServicePod",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlAlertsAndTroubleshooting,
							"message": "Grafana Service has no pods in ready state.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("sum(kube_pod_status_ready{condition='true',namespace='%[1]v', pod=~'grafana-deployment.*'} * on(pod, namespace) group_left() kube_pod_status_phase{phase='Running'}) < 1", namespace)),
						For:    resources.DurationPtr("5m"),
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
				},
			},
		},
	}
}

func (r *Reconciler) removeGrafanaOperatorAlerts(nsPrefix string, ctx context.Context, apiClient k8sclient.Client) error {
	prometheusRule := &monv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "customer-monitoring-ksm-grafana-alerts",
			Namespace: nsPrefix + "operator-observability",
		},
	}
	err := apiClient.Delete(ctx, prometheusRule)
	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}
	prometheusRule = &monv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "customer-monitoring-ksm-endpoint-alerts",
			Namespace: nsPrefix + "operator-observability",
		},
	}
	err = apiClient.Delete(ctx, prometheusRule)
	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}
	return nil
}
