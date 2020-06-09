package rhssouser

import (
	"context"
	"fmt"
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) reconcileKubeStateMetricsEndpointAvailableAlerts(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	monitoringConfig := config.NewMonitoring(config.ProductConfig{})
	rule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ksm-endpoint-alerts",
			Namespace: r.Config.GetNamespace(),
		},
	}

	rules := []monitoringv1.Rule{
		{
			Alert: "RHMIUserRhssoKeycloakServiceEndpointDown",
			Annotations: map[string]string{
				"sop_url": "https://github.com/RHCloudServices/integreatly-help/tree/master/sops/2.x/alerts",
				"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
			},
			Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='keycloak'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
			For:    "1m",
			Labels: map[string]string{"severity": "critical"},
		},
		{
			Alert: "RHMIUserRhssoKeycloakDiscoveryServiceEndpointDown",
			Annotations: map[string]string{
				"sop_url": "https://github.com/RHCloudServices/integreatly-help/tree/master/sops/2.x/alerts",
				"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetNamespace()),
			},
			Expr:   intstr.FromString("kube_endpoint_address_available{endpoint='keycloak-discovery'} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1"),
			For:    "1m",
			Labels: map[string]string{"severity": "critical"},
		}}

	_, err := controllerutil.CreateOrUpdate(ctx, client, rule, func() error {
		rule.ObjectMeta.Labels = map[string]string{"integreatly": "yes", monitoringConfig.GetLabelSelectorKey(): monitoringConfig.GetLabelSelector()}
		rule.Spec = monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name:  "user-rhsso.rules",
					Rules: rules,
				},
			},
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating user-rhsso PrometheusRule: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
