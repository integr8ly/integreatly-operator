package apicurito

import (
	"context"
	"fmt"
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) reconcilePodCountAlert(ctx context.Context, client k8sclient.Client) error {
	const apicuritoPodCountExpected int = 3
	ns := r.Config.GetNamespace()
	prometheusRule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ksm-apicurito-alerts",
			Namespace: ns,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, prometheusRule, func() error {
		owner.AddIntegreatlyOwnerAnnotations(prometheusRule, r.installation)

		prometheusRule.Spec = monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name: "apicurito.rules",
					Rules: []monitoringv1.Rule{
						{
							Alert: "ApicuritoPodCount",
							Annotations: map[string]string{
								"sop_url": resources.SopUrlAlertsAndTroubleshooting,
								"message": fmt.Sprintf("Pod count for namespace %s is %s. Expected exactly %d pods.", "{{ $labels.namespace }}", "{{  printf \"%.0f\" $value }}", apicuritoPodCountExpected),
							},
							Expr: intstr.FromString(
								fmt.Sprintf("(1-absent(kube_pod_status_ready{condition='true', namespace='%s'})) or sum(kube_pod_status_ready{condition='true', namespace='%s'}) != %d", ns, ns, apicuritoPodCountExpected)),
							For:    "5m",
							Labels: map[string]string{"severity": "critical"},
						},
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error creating apicurito PrometheusRule: %w", err)
	}

	return nil

}

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
		}}

	or, err := controllerutil.CreateOrUpdate(ctx, client, rule, func() error {
		rule.ObjectMeta.Labels = map[string]string{"integreatly": "yes", monitoringConfig.GetLabelSelectorKey(): monitoringConfig.GetLabelSelector()}
		rule.Spec = monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name:  "apicurito-endpoint.rules",
					Rules: rules,
				},
			},
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating apicurito PrometheusRule: %w", err)
	}

	if or != controllerutil.OperationResultNone {
		r.logger.Infof("The operation result for apicurito %s was %s", rule.Name, or)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileKubeStateMetricsOperatorEndpointAvailableAlerts(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	monitoringConfig := config.NewMonitoring(config.ProductConfig{})
	rule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ksm-endpoint-alerts",
			Namespace: r.Config.GetOperatorNamespace(),
		},
	}

	rules := []monitoringv1.Rule{
		{
			Alert: "RHMIApicuritoOperatorRhmiRegistryCsServiceEndpointDown",
			Annotations: map[string]string{
				"sop_url": resources.SopUrlEndpointAvailableAlert,
				"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
			},
			Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='rhmi-registry-cs', namespace=`%s`} * on (namespace) group_left kube_namespace_labels{label_monitoring_key='middleware'} < 1", r.Config.GetOperatorNamespace())),
			For:    "5m",
			Labels: map[string]string{"severity": "critical"},
		}}

	or, err := controllerutil.CreateOrUpdate(ctx, client, rule, func() error {
		rule.ObjectMeta.Labels = map[string]string{"integreatly": "yes", monitoringConfig.GetLabelSelectorKey(): monitoringConfig.GetLabelSelector()}
		rule.Spec = monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name:  "apicurito-operator-endpoint.rules",
					Rules: rules,
				},
			},
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating apicurito operator PrometheusRule: %w", err)
	}

	if or != controllerutil.OperationResultNone {
		r.logger.Infof("The operation result for apicurito operator %s was %s", rule.Name, or)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
