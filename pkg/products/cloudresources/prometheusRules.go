package cloudresources

import (
	"context"
	"fmt"
	crov1alpha1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	"strings"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Reconciler) newAlertsReconciler(ctx context.Context, client k8sclient.Client, logger l.Logger, installType string, ns string) (resources.AlertReconciler, error) {
	installationName := resources.InstallationNames[installType]

	namespace := r.Config.GetOperatorNamespace()
	alertName := "ksm-endpoint-alerts"

	if integreatlyv1alpha1.IsRHOAM(integreatlyv1alpha1.InstallationType(installType)) {
		observabilityConfig, err := r.ConfigManager.ReadObservability()
		if err != nil {
			logger.Warning("failed to get observability config")
			return nil, nil
		}

		namespace = observabilityConfig.GetNamespace()
		alertName = "cro-ksm-endpoint-alerts"
	}

	alertsReconciler := &resources.AlertReconcilerImpl{
		ProductName:  "Cloud Resources Operator",
		Installation: r.installation,
		Log:          logger,
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: alertName,
				Namespace: namespace,
				GroupName: "cloud-resources-operator-endpoint.rules",
				Rules: []monitoringv1.Rule{
					{
						Alert: "RHMICloudResourceOperatorMetricsServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlRHMICloudResourceOperatorMetricsServiceEndpointDown,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{namespace='%s', endpoint='operator-metrics-service'} < 1", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					}, {
						Alert: "RHMICloudResourceOperatorRhmiRegistryCsServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='rhmi-registry-cs', namespace='%s'} < 1", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					},
				},
			},
		},
	}

	return addElasticCacheSnapshotNotFoundAlert(ctx, client, logger, installationName, *alertsReconciler, ns)
}

func addElasticCacheSnapshotNotFoundAlert(ctx context.Context, client k8sclient.Client, logger l.Logger, installationName string, alertsReconciler resources.AlertReconcilerImpl, ns string) (resources.AlertReconciler, error) {

	names, err := getRedisCRsNames(ctx, client, logger, ns)
	if err != nil {
		logger.Error("Error getting redis names", err)
		return &alertsReconciler, err
	}
	if len(names) == 0 {
		return &alertsReconciler, nil
	}

	metricsCheck := ""
	for _, name := range names {
		metricsCheck = metricsCheck + "cro_redis_snapshot_not_found_" + name + " > 1 or "
	}
	metricsCheck = strings.TrimSuffix(metricsCheck, " or ")
	// sanitise
	metricsCheck = sanitize(metricsCheck)

	alertsReconciler.Alerts[0].Rules = append(alertsReconciler.Alerts[0].Rules, monitoringv1.Rule{
		Alert: "RHMICloudResourceOperatorElasticCacheSnapshotsNotFound",
		Annotations: map[string]string{
			"sop_url": resources.SopUrlAlertsAndTroubleshooting,
			"message": fmt.Sprintf("Elastic Cache snapshot not found or not available for tagging."),
		},
		Expr:   intstr.FromString(metricsCheck),
		Labels: map[string]string{"severity": "warning", "product": installationName},
	})

	return &alertsReconciler, nil
}

func getRedisCRsNames(ctx context.Context, client k8sclient.Client, logger l.Logger, ns string) ([]string, error) {

	names := []string{}

	// ensure redis instances are cleaned up
	redisInstances := &crov1alpha1.RedisList{}
	redisInstanceOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(ns),
	}
	err := client.List(ctx, redisInstances, redisInstanceOpts...)
	if err != nil {
		return []string{}, fmt.Errorf("failed to list redis instances: %w", err)
	}

	for _, redisInst := range redisInstances.Items {
		names = append(names, redisInst.Name)
	}

	return names, nil
}

func sanitize(metricsCheck string) string {
	// Convention for CRs is - but _ for prom metrics
	return strings.ToLower(strings.ReplaceAll(metricsCheck, "-", "_"))
}
