package resources

import (
	"context"
	"fmt"

	prometheusv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	croResources "github.com/integr8ly/cloud-resource-operator/pkg/resources"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CreatePostgresAvailabilityAlert creates a PrometheusRule alert to watch for the availability
// of a Postgres instance
func CreatePostgresAvailabilityAlert(ctx context.Context, client k8sclient.Client, cr *v1alpha1.Postgres) (*prometheusv1.PrometheusRule, error) {
	productName := cr.Labels["productName"]
	alertName := productName + "PostgresInstanceUnavailable"
	ruleName := fmt.Sprintf("availability-rule-%s", cr.Name)
	alertExp := intstr.FromString(
		fmt.Sprintf("absent(%s{exported_namespace='%s',resourceID='%s',productName='%s'} == 1)",
			croResources.DefaultPostgresAvailMetricName, cr.Namespace, cr.Name, productName),
	)
	alertDescription := fmt.Sprintf("Postgres instance: '%s' (strategy: %s) for product: %s is unavailable", cr.Name, cr.Status.Strategy, productName)
	labels := map[string]string{
		"severity":    "critical",
		"productName": cr.Labels["productName"],
	}

	// create the rule
	pr, err := croResources.ReconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, alertExp, labels)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// CreateRedisAvailabilityAlert creates a PrometheusRule alert to watch for the availability
// of a Redis cache
func CreateRedisAvailabilityAlert(ctx context.Context, client k8sclient.Client, cr *v1alpha1.Redis) (*prometheusv1.PrometheusRule, error) {
	productName := cr.Labels["productName"]
	alertName := productName + "RedisCacheUnavailable"
	ruleName := fmt.Sprintf("availability-rule-%s", cr.Name)
	alertExp := intstr.FromString(
		fmt.Sprintf("absent(%s{exported_namespace='%s',resourceID='%s',productName='%s'} == 1)",
			croResources.DefaultRedisAvailMetricName, cr.Namespace, cr.Name, productName),
	)
	alertDescription := fmt.Sprintf("Redis instance: '%s' (strategy: %s) for the product: %s is unavailable", cr.Name, cr.Status.Strategy, productName)
	labels := map[string]string{
		"severity":    "critical",
		"productName": cr.Labels["productName"],
	}

	// create the rule
	pr, err := croResources.ReconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, alertExp, labels)
	if err != nil {
		return nil, err
	}
	return pr, nil
}
