package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	prometheusv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/pkg/errors"
	errorUtil "github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	customMetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	sleepytime                     = 3600
	DefaultPostgresAvailMetricName = "cro_postgres_available"
	DefaultRedisAvailMetricName    = "cro_redis_available"
)

var (
	// MetricVecs create the map of vectors
	MetricVecs map[string]prometheus.GaugeVec
	logger     *logrus.Entry
)

func init() {
	StartGaugeVector()
}

// StartGaugeVector periodic loop that is wiping all known vectors.
func StartGaugeVector() {
	MetricVecs = map[string]prometheus.GaugeVec{}
	logger = logrus.WithFields(logrus.Fields{"custom_metrics": "StartGaugeVector"})

	go func() {
		for {
			logger.Info("calling reset on all prometheus gauge vectors")
			for _, val := range MetricVecs {
				val.Reset()
			}
			time.Sleep(time.Duration(sleepytime) * time.Second)
		}
	}()
}

// SetMetric Set exports a Prometheus Gauge
func SetMetric(name string, labels map[string]string, value float64) error {
	// set vector value
	gv, ok := MetricVecs[name]
	if ok {
		gv.With(labels).Set(value)
		return nil
	}

	// create label array for vector creation
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}

	// the vector does not exist, create it, register and then add this gauge metric to the gauge vector
	gv = *prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name}, keys)
	customMetrics.Registry.MustRegister(gv)
	MetricVecs[name] = gv

	return nil
}

// SetMetricCurrentTime Set current time wraps set metric
func SetMetricCurrentTime(name string, labels map[string]string) error {
	if err := SetMetric(name, labels, float64(time.Now().UnixNano())/1e9); err != nil {
		return errorUtil.Wrap(err, "unable to set current time gauge vector")
	}
	return nil
}

// CreatePrometheusRule will create a PrometheusRule object
func ReconcilePrometheusRule(ctx context.Context, client client.Client, ruleName, ns, alertName, desc string, alertExp intstr.IntOrString, labels map[string]string) (*prometheusv1.PrometheusRule, error) {
	alertGroupName := alertName + "Group"
	groups := []prometheusv1.RuleGroup{
		{
			Name: alertGroupName,
			Rules: []prometheusv1.Rule{
				{
					Alert:  alertName,
					Expr:   alertExp,
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
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to reconcile prometheus rule request for %s", ruleName)
	}

	return rule, nil
}

// DeletePrometheusRule will delete a prometheus rule object
func DeletePrometheusRule(ctx context.Context, client client.Client, ruleName, ns string) error {
	rule := &prometheusv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      ruleName,
		},
	}

	// call delete on that object
	if err := client.Delete(ctx, rule); err != nil {
		return err
	}

	return nil
}

// CreatePostgresAvailabilityAlert creates an alert for the availability of a postgres instance
func CreatePostgresAvailabilityAlert(ctx context.Context, client client.Client, cr *v1alpha1.Postgres) (*prometheusv1.PrometheusRule, error) {
	clusterID, err := GetClusterID(ctx, client)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to retrieve cluster identifier")
	}
	ruleName := fmt.Sprintf("availability-rule-%s", cr.Name)
	alertRuleName := "PostgresInstanceUnavailable"
	alertExp := intstr.FromString(
		fmt.Sprintf("absent(%s{exported_namespace='%s',resourceID='%s'} == 1)",
			DefaultPostgresAvailMetricName, cr.Namespace, cr.Name),
	)
	alertDescription := fmt.Sprintf("Postgres instance: %s on cluster: %s for product: %s (strategy: %s) is unavailable", cr.Name, clusterID, cr.Labels["productName"], cr.Status.Strategy)
	labels := map[string]string{
		"severity":    "critical",
		"productName": cr.Labels["productName"],
	}

	// create the rule
	pr, err := ReconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertRuleName, alertDescription, alertExp, labels)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// DeletePostgresAvailabilityAlert deletes the postgres availability alert
func DeletePostgresAvailabilityAlert(ctx context.Context, client client.Client, cr *v1alpha1.Postgres) error {
	ruleName := fmt.Sprintf("availability-rule-%s", cr.Name)
	return DeletePrometheusRule(ctx, client, ruleName, cr.Namespace)
}

// CreateRedisAvailabilityAlert creates an alert for the availability of a redis cache
func CreateRedisAvailabilityAlert(ctx context.Context, client client.Client, cr *v1alpha1.Redis) (*prometheusv1.PrometheusRule, error) {
	clusterID, err := GetClusterID(ctx, client)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to retrieve cluster identifier")
	}
	ruleName := fmt.Sprintf("availability-rule-%s", cr.Name)
	alertRuleName := "RedisInstanceUnavailable"
	alertExp := intstr.FromString(
		fmt.Sprintf("absent(%s{exported_namespace='%s',resourceID='%s'} == 1)",
			DefaultRedisAvailMetricName, cr.Namespace, cr.Name),
	)
	alertDescription := fmt.Sprintf("Redis cache: %s on cluster: %s for product: %s (strategy: %s) is unavailable", cr.Name, clusterID, cr.Labels["productName"], cr.Status.Strategy)
	labels := map[string]string{
		"severity":    "critical",
		"productName": cr.Labels["productName"],
	}

	// create the rule
	pr, err := ReconcilePrometheusRule(ctx, client, ruleName, cr.Namespace, alertRuleName, alertDescription, alertExp, labels)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// DeleteRedisAvailabilityAlert deletes the redis availability alert
func DeleteRedisAvailabilityAlert(ctx context.Context, client client.Client, cr *v1alpha1.Redis) error {
	ruleName := fmt.Sprintf("availability-rule-%s", cr.Name)
	return DeletePrometheusRule(ctx, client, ruleName, cr.Namespace)
}
