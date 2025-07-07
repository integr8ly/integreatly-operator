package controllers

import (
	"context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	installationNames = map[string]string{
		string(integreatlyv1alpha1.InstallationTypeManagedApi):            "rhoam",
		string(integreatlyv1alpha1.InstallationTypeMultitenantManagedApi): "rhoam",
	}
)

func (r *RHMIReconciler) removeInstallationRules(installation *integreatlyv1alpha1.RHMI, ctx context.Context, apiClient k8sclient.Client) (err error) {
	installationName := installationNames[installation.Spec.Type]
	installationAlert := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-installation-alerts", installationName),
			Namespace: config.OpenshiftMonitoringNamespace,
		},
	}
	err = apiClient.Delete(ctx, installationAlert)
	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}
	upgradeAlert := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-upgrade-alerts", installationName),
			Namespace: config.OpenshiftMonitoringNamespace,
		},
	}
	err = apiClient.Delete(ctx, upgradeAlert)
	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}
	metricMissingAlert := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-missing-metrics", installationName),
			Namespace: config.OpenshiftMonitoringNamespace,
		},
	}
	err = apiClient.Delete(ctx, metricMissingAlert)
	if err != nil && !k8serr.IsNotFound(err) {
		return err
	}
	return nil
}
