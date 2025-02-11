package cloudresources

import (
	"context"
	"fmt"
	croResources "github.com/integr8ly/cloud-resource-operator/pkg/resources"
	"strings"

	crov1alpha1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"

	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Reconciler) newAlertsReconciler(ctx context.Context, client k8sclient.Client, logger l.Logger, installType string, namespace string) (resources.AlertReconciler, error) {
	installationName := resources.InstallationNames[installType]

	alertsReconciler := &resources.AlertReconcilerImpl{
		ProductName:  "Cloud Resources Operator",
		Installation: r.installation,
		Log:          logger,
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: "cro-ksm-endpoint-alerts",
				Namespace: namespace,
				GroupName: "cloud-resources-operator-endpoint.rules",
				Rules: []monv1.Rule{
					{
						Alert: "RHOAMCloudResourceOperatorMetricsServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlRHOAMCloudResourceOperatorMetricsServiceEndpointDown,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{namespace='%s', endpoint='operator-metrics-service'} < 1", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					}, {
						Alert: "RHOAMCloudResourceOperatorRhmiRegistryCsServiceEndpointDown",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlEndpointAvailableAlert,
							"message": fmt.Sprintf("No {{  $labels.endpoint  }} endpoints in namespace %s. Expected at least 1.", r.Config.GetOperatorNamespace()),
						},
						Expr:   intstr.FromString(fmt.Sprintf("kube_endpoint_address_available{endpoint='rhmi-registry-cs', namespace='%s'} < 1", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "warning", "product": installationName},
					}, {
						Alert: "RHOAMCloudResourceOperatorVPCActionFailed",
						Annotations: map[string]string{
							"sop_url": resources.SopUrlRHOAMCloudResourceOperatorVPCActionFailed,
							"message": "CRO failed to perform an action on a VPC.",
						},
						Expr:   intstr.FromString(fmt.Sprintf("cro_vpc_action{namespace='%s', status='failed', error!=''} > 0", r.Config.GetOperatorNamespace())),
						For:    "5m",
						Labels: map[string]string{"severity": "critical", "product": installationName},
					},
				},
			},
			{
				AlertName: "cloud-resource-operator-metrics-missing",
				Namespace: namespace,
				GroupName: "cloud-resource-operator.rules",
				Rules: []monv1.Rule{
					{
						Alert: fmt.Sprintf("%sCloudResourceOperatorMetricsMissing", strings.ToUpper(installationName)),
						Expr: intstr.FromString(
							fmt.Sprintf(`(absent(%s) or absent(%s) or absent(%s) or absent(%s) or absent(%s) or absent(%s) or absent(%s)) == 1 and rhoam_spec{use_cluster_storage="false"}`,
								croResources.DefaultPostgresAvailMetricName,
								croResources.DefaultPostgresConnectionMetricName,
								croResources.DefaultPostgresStatusMetricName,
								croResources.DefaultRedisAvailMetricName,
								croResources.DefaultRedisConnectionMetricName,
								croResources.DefaultRedisStatusMetricName,
								croResources.PostgresAllocatedStorageMetricName,
							),
						),
						For:    "5m",
						Labels: map[string]string{"severity": "critical"},
						Annotations: map[string]string{
							"sop_url": resources.SopUrlRHOAMCloudResourceOperatorMetricsMissing,
							"message": "one or more cloud-resource-operator metrics have been missing for 5+ minutes",
						},
					},
				},
			},
		},
	}

	return addElasticCacheSnapshotNotFoundAlert(ctx, client, logger, installationName, *alertsReconciler, r.installation.Namespace)
}

func addElasticCacheSnapshotNotFoundAlert(ctx context.Context, client k8sclient.Client, logger l.Logger, installationName string, alertsReconciler resources.AlertReconcilerImpl, ns string) (resources.AlertReconciler, error) {

	names, err := getRedisCRsNames(ctx, client, ns)
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

	alertListType := alertsReconciler.Alerts[0].Rules
	if alertListType, ok := alertListType.([]monv1.Rule); ok {
		alertListType = append(alertListType, monv1.Rule{
			Alert: "RHOAMCloudResourceOperatorElasticCacheSnapshotsNotFound",
			Annotations: map[string]string{
				"sop_url": resources.SopUrlAlertsAndTroubleshooting,
				"message": "Elastic Cache snapshot not found or not available for tagging.",
			},
			Expr:   intstr.FromString(metricsCheck),
			Labels: map[string]string{"severity": "critical", "product": installationName},
		})
		alertsReconciler.Alerts[0].Rules = alertListType
	}

	return &alertsReconciler, nil
}

func getRedisCRsNames(ctx context.Context, client k8sclient.Client, ns string) ([]string, error) {

	var names []string

	// ensure redis instances are cleaned up
	redisInstances := &crov1alpha1.RedisList{}
	redisInstanceOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(ns),
	}
	err := client.List(ctx, redisInstances, redisInstanceOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to list redis instances: %w", err)
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
