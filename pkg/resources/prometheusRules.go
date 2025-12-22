package resources

import (
	"context"
	"fmt"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	InstallationNames = map[string]string{
		string(integreatlyv1alpha1.InstallationTypeManagedApi):            "rhoam",
		string(integreatlyv1alpha1.InstallationTypeMultitenantManagedApi): "rhoam",
	}
)

type AlertReconciler interface {
	ReconcileAlerts(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error)
}

type AlertReconcilerImpl struct {
	ProductName  string
	Log          l.Logger
	Installation *integreatlyv1alpha1.RHMI
	Alerts       []AlertConfiguration
	// RemovedAlerts Should contain Alerts that have been removed or renamed to ensure there is no orphaned
	// Alerts on clusters after upgrades. These alerts will be deleted as part of ReconcileAlerts
	RemovedAlerts []AlertConfiguration
}

var _ AlertReconciler = &AlertReconcilerImpl{}

type AlertConfiguration struct {
	AlertName string
	GroupName string
	Namespace string
	Interval  string
	Rules     interface{}
}

func (r *AlertReconcilerImpl) ReconcileAlerts(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// If the installation was marked for deletion, delete the alerts
	if r.Installation.DeletionTimestamp != nil {
		allAlerts := append(r.Alerts, r.RemovedAlerts...)
		if err := r.deleteAlerts(ctx, client, allAlerts); err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	for _, alert := range r.Alerts {
		if or, err := r.reconcileRule(ctx, client, alert); err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		} else if or != controllerutil.OperationResultNone {
			r.Log.Infof("Operation result", l.Fields{"productName": r.ProductName, "alertName": alert.AlertName, "result": string(or)})
		}
	}

	if err := r.deleteAlerts(ctx, client, r.RemovedAlerts); err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *AlertReconcilerImpl) reconcileRule(ctx context.Context, client k8sclient.Client, alert AlertConfiguration) (controllerutil.OperationResult, error) {

	var alertRulesType interface{} = alert.Rules

	switch alertRules := alertRulesType.(type) {
	case []monv1.Rule:
		rule := &monv1.PrometheusRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      alert.AlertName,
				Namespace: alert.Namespace,
			},
		}

		return controllerutil.CreateOrUpdate(ctx, client, rule, func() error {
			rule.ObjectMeta.Labels = map[string]string{
				"integreatly":                   "yes",
				config.GetOboLabelSelectorKey(): config.GetOboLabelSelector(),
			}
			var intervalPtr *monv1.Duration
			if alert.Interval != "" {
				intervalPtr = DurationPtr(alert.Interval)
			}
			rule.Spec = monv1.PrometheusRuleSpec{
				Groups: []monv1.RuleGroup{
					{
						Name:     alert.GroupName,
						Rules:    alertRules,
						Interval: intervalPtr,
					},
				},
			}

			return nil
		})
	case []monitoringv1.Rule:
		rule := &monitoringv1.PrometheusRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      alert.AlertName,
				Namespace: alert.Namespace,
			},
		}

		return controllerutil.CreateOrUpdate(ctx, client, rule, func() error {
			rule.ObjectMeta.Labels = map[string]string{
				"integreatly":                   "yes",
				config.GetOboLabelSelectorKey(): config.GetOboLabelSelector(),
			}
			var upIntervalPtr *monitoringv1.Duration
			if alert.Interval != "" {
				upIntervalPtr = UpDurationPtr(alert.Interval)
			}
			rule.Spec = monitoringv1.PrometheusRuleSpec{
				Groups: []monitoringv1.RuleGroup{
					{
						Name:     alert.GroupName,
						Rules:    alertRules,
						Interval: upIntervalPtr,
					},
				},
			}

			return nil
		})
	default:
		return controllerutil.OperationResultNone, fmt.Errorf("failed to find alert type")
	}

}

func (r *AlertReconcilerImpl) deleteAlerts(ctx context.Context, client k8sclient.Client, alerts []AlertConfiguration) error {
	rule := &monitoringv1.PrometheusRule{}

	for _, alert := range alerts {
		if err := client.Get(ctx, k8sclient.ObjectKey{
			Name:      alert.AlertName,
			Namespace: alert.Namespace,
		}, rule); err != nil {
			if errors.IsNotFound(err) {
				continue
			}

			return fmt.Errorf("failed to retrieve alert %s: %v", alert.AlertName, err)
		}

		if err := client.Delete(ctx, rule); err != nil {
			return fmt.Errorf("failed to delete alert %s: %v", alert.AlertName, err)
		}
	}

	return nil
}

type NoopAlertReconciler struct{}

var _ AlertReconciler = &NoopAlertReconciler{}

func (n *NoopAlertReconciler) ReconcileAlerts(_ context.Context, _ k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	return integreatlyv1alpha1.PhaseCompleted, nil
}
