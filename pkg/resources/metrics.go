package resources

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/sirupsen/logrus"

	prometheusv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	croResources "github.com/integr8ly/cloud-resource-operator/pkg/resources"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const alertFor = "5m"

// CreatePostgresAvailabilityAlert creates a PrometheusRule alert to watch for the availability
// of a Postgres instance
func CreatePostgresAvailabilityAlert(ctx context.Context, client k8sclient.Client, inst *v1alpha1.RHMI, cr *crov1.Postgres) (*prometheusv1.PrometheusRule, error) {
	if strings.ToLower(inst.Spec.UseClusterStorage) == "true" {
		logrus.Info("skipping postgres alert creation, useClusterStorage is true")
		return nil, nil
	}

	productName := cr.Labels["productName"]
	postgresCRName := strings.Title(strings.Replace(cr.Name, "postgres-example-rhmi", "", -1))
	alertName := postgresCRName + "PostgresInstanceUnavailable"
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
	pr, err := reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, alertExp, labels)
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
	postgresCRName := strings.Title(strings.Replace(cr.Name, "postgres-example-rhmi", "", -1))
	alertName := postgresCRName + "PostgresConnectionFailed"
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
	pr, err := reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, alertExp, labels)
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
	redisCRName := strings.Title(strings.Replace(cr.Name, "redis-example-rhmi", "", -1))
	alertName := redisCRName + "RedisCacheUnavailable"
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
	pr, err := reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, alertExp, labels)
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
	redisCRName := strings.Title(strings.Replace(cr.Name, "redis-example-rhmi", "", -1))
	alertName := redisCRName + "RedisCacheConnectionFailed"
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
	pr, err := reconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertName, alertDescription, alertExp, labels)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// reconcilePrometheusRule will create a PrometheusRule object
func reconcilePrometheusRule(ctx context.Context, client k8sclient.Client, ruleName, ns, alertName, desc string, alertExp intstr.IntOrString, labels map[string]string) (*prometheusv1.PrometheusRule, error) {
	alertGroupName := alertName + "Group"
	groups := []prometheusv1.RuleGroup{
		{
			Name: alertGroupName,
			Rules: []prometheusv1.Rule{
				{
					Alert:  alertName,
					Expr:   alertExp,
					For:    alertFor,
					Labels: labels,
					Annotations: map[string]string{
						"description": desc,
					},
				},
			},
		},
	}

	rule := &prometheusv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleName,
			Namespace: ns,
			Labels: map[string]string{
				"monitoring-key": "middleware",
			},
		},
		Spec: prometheusv1.PrometheusRuleSpec{
			Groups: groups,
		},
	}

	// create or update the resource
	_, err := controllerutil.CreateOrUpdate(ctx, client, rule, func() error {
		rule.Name = ruleName
		rule.Namespace = ns
		rule.Spec.Groups = []prometheusv1.RuleGroup{
			{
				Name: alertGroupName,
				Rules: []prometheusv1.Rule{
					{
						Alert:  alertName,
						Expr:   alertExp,
						For:    alertFor,
						Labels: labels,
						Annotations: map[string]string{
							"description": desc,
						},
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to reconcile prometheus rule request for %s", ruleName)
	}

	return rule, nil
}
