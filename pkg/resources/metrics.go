package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/sirupsen/logrus"

	prometheusv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	croResources "github.com/integr8ly/cloud-resource-operator/pkg/resources"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CreatePostgresAvailabilityAlert creates a PrometheusRule alert to watch for the availability
// of a Postgres instance
func CreatePostgresAvailabilityAlert(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Postgres) (*prometheusv1.PrometheusRule, error) {
	if strings.ToLower(inst.Spec.UseClusterStorage) == "true" {
		logrus.Info("skipping postgres alert creation, useClusterStorage is true")
		return nil, nil
	}

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

// CreatePostgresConnectivityAlert creates a PrometheusRule alert to watch for the connectivity
// of a Postgres instance
func CreatePostgresConnectivityAlert(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Postgres) (*prometheusv1.PrometheusRule, error) {
	if strings.ToLower(inst.Spec.UseClusterStorage) == "true" {
		logrus.Info("skipping postgres connectivity alert creation, useClusterStorage is true")
		return nil, nil
	}
	productName := cr.Labels["productName"]
	alertName := productName + "PostgresConnectionFailed"
	ruleName := fmt.Sprintf("connectivity-rule-%s", cr.Name)
	alertExp := intstr.FromString(
		fmt.Sprintf("absent(%s{exported_namespace='%s',resourceID='%s',productName='%s'} == 1)",
			croResources.DefaultPostgresConnectionMetricName, cr.Namespace, cr.Name, productName),
	)
	alertDescription := fmt.Sprintf("Unable to connect to Postgres instance. Postgres Custom Resource: %s in namespace %s (strategy: %s) for product: %s", cr.Name, cr.Namespace, cr.Status.Strategy, productName)
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
func CreateRedisAvailabilityAlert(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Redis) (*prometheusv1.PrometheusRule, error) {
	if strings.ToLower(inst.Spec.UseClusterStorage) == "true" {
		logrus.Info("skipping redis alert creation, useClusterStorage is true")
		return nil, nil
	}

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

// CreateRedisConnectivityAlert creates a PrometheusRule alert to watch for the connectivity
// of a Redis cache
func CreateRedisConnectivityAlert(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Redis) (*prometheusv1.PrometheusRule, error) {
	if strings.ToLower(inst.Spec.UseClusterStorage) == "true" {
		logrus.Info("skipping redis connectivity alert creation, useClusterStorage is true")
		return nil, nil
	}
	productName := cr.Labels["productName"]
	alertName := productName + "RedisCacheConnectionFailed"
	ruleName := fmt.Sprintf("connectivity-rule-%s", cr.Name)
	alertExp := intstr.FromString(
		fmt.Sprintf("absent(%s{exported_namespace='%s',resourceID='%s',productName='%s'} == 1)",
			croResources.DefaultRedisConnectionMetricName, cr.Namespace, cr.Name, productName),
	)
	alertDescription := fmt.Sprintf("Unable to connect to Redis instance. Redis Custom Resource: %s in namespace %s (strategy: %s) for the product: %s", cr.Name, cr.Namespace, cr.Status.Strategy, productName)
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
