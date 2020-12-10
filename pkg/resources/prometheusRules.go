package resources

import (
	"context"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type AlertReconciler interface {
	ReconcileAlerts(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error)
}

type AlertReconcilerImpl struct {
	ProductName  string
	Log          l.Logger
	Installation *integreatlyv1alpha1.RHMI
	Alerts       []AlertConfiguration
}

var _ AlertReconciler = &AlertReconcilerImpl{}

type AlertConfiguration struct {
	AlertName string
	GroupName string
	Namespace string
	Interval  string
	Rules     []monitoringv1.Rule
}

func (r *AlertReconcilerImpl) ReconcileAlerts(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// If the installation was marked for deletion, delete the alerts
	if r.Installation.DeletionTimestamp != nil {
		if err := r.deleteAlerts(ctx, client); err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	monitoringConfig := config.NewMonitoring(config.ProductConfig{})

	for _, alert := range r.Alerts {
		if or, err := r.reconcileRule(ctx, client, monitoringConfig, alert); err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		} else if or != controllerutil.OperationResultNone {
			r.Log.Infof("Operation result", l.Fields{"productName": r.ProductName, "alertName": alert.AlertName, "result": string(or)})
		}
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *AlertReconcilerImpl) reconcileRule(ctx context.Context, client k8sclient.Client, monitoringConfig *config.Monitoring, alert AlertConfiguration) (controllerutil.OperationResult, error) {
	rule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      alert.AlertName,
			Namespace: alert.Namespace,
		},
	}

	return controllerutil.CreateOrUpdate(ctx, client, rule, func() error {
		rule.ObjectMeta.Labels = map[string]string{
			"integreatly":                          "yes",
			monitoringConfig.GetLabelSelectorKey(): monitoringConfig.GetLabelSelector(),
		}
		rule.Spec = monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name:     alert.GroupName,
					Rules:    alert.Rules,
					Interval: alert.Interval,
				},
			},
		}

		return nil
	})
}

func (r *AlertReconcilerImpl) deleteAlerts(ctx context.Context, client k8sclient.Client) error {
	rule := &monitoringv1.PrometheusRule{}

	for _, alert := range r.Alerts {
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
