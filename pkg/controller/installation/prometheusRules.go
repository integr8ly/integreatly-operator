package installation

import (
	"context"
	"fmt"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileInstallation) reconcileRHMIInstallationAlerts(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {
	monitoringConfig := config.NewMonitoring(config.ProductConfig{})

	rule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhmi-installation-controller-alerts",
			Namespace: installation.Namespace,
		},
	}

	rules := []monitoringv1.Rule{
		{
			Alert: "RHMIInstallationControllerIsNotReconciling",
			Annotations: map[string]string{
				"sop_url": resources.SopUrlAlertsAndTroubleshooting,
				"message": "RHMI operator has not reconciled successfully in the interval of 15m over the past 1 hour",
			},
			Expr:   intstr.FromString(fmt.Sprint("rhmi_status{stage='complete'} AND on(namespace) rate(controller_runtime_reconcile_total{controller='installation-controller', result='success'}[15m]) == 0")),
			For:    "1h",
			Labels: map[string]string{"severity": "warning"},
		},
		{
			Alert: "RHMIInstallationControllerStoppedReconciling",
			Annotations: map[string]string{
				"sop_url": resources.SopUrlAlertsAndTroubleshooting,
				"message": "RHMI operator has not reconciled successfully in the interval of 30m over the past 2 hours",
			},
			Expr:   intstr.FromString(fmt.Sprint("rhmi_status{stage='complete'} AND on(namespace) rate(controller_runtime_reconcile_total{controller='installation-controller', result='success'}[30m]) == 0")),
			For:    "2h",
			Labels: map[string]string{"severity": "critical"},
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, rule, func() error {
		rule.ObjectMeta.Labels = map[string]string{"integreatly": "yes", monitoringConfig.GetLabelSelectorKey(): monitoringConfig.GetLabelSelector()}
		rule.Spec = monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name:  "rhmi-installation.rules",
					Rules: rules,
				},
			},
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating rhmi installation PrometheusRule: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *ReconcileInstallation) reconcileRHMIInstallationAlertsOpenshiftMonitoring(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {
	monitoringConfig := config.NewMonitoring(config.ProductConfig{})

	rule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhmi-installation-alerts",
			Namespace: "openshift-monitoring",
		},
	}

	rules := []monitoringv1.Rule{
		{
			Alert: "RHMIOperatorInstallDelayed",
			Annotations: map[string]string{
				"sop_url": resources.SopUrlAlertsAndTroubleshooting,
				"message": "RHMI operator is taking more than 2 hours to go to a complete stage",
			},
			Expr:   intstr.FromString(fmt.Sprint("absent(rhmi_status{stage='complete'} == 1)")),
			For:    "120m",
			Labels: map[string]string{"severity": "critical"},
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, rule, func() error {
		rule.ObjectMeta.Labels = map[string]string{"integreatly": "yes", monitoringConfig.GetLabelSelectorKey(): monitoringConfig.GetLabelSelector()}
		rule.Spec = monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name:  "rhmi-installation.rules",
					Rules: rules,
				},
			},
		}
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating rhmi installation PrometheusRule: %w", err)
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}
