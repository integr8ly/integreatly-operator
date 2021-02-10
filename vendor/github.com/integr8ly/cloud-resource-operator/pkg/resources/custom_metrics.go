package resources

import (
	"context"
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	prometheusv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	customMetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	sleepytime                                = 3600
	DefaultPostgresMaintenanceMetricName      = "cro_postgres_service_maintenance"
	DefaultPostgresInfoMetricName             = "cro_postgres_info"
	DefaultPostgresAvailMetricName            = "cro_postgres_available"
	DefaultPostgresConnectionMetricName       = "cro_postgres_connection"
	DefaultPostgresStatusMetricName           = "cro_postgres_status_phase"
	DefaultPostgresDeletionMetricName         = "cro_postgres_deletion_timestamp"
	DefaultPostgresSnapshotStatusMetricName   = "cro_postgres_snapshot_status_phase"
	DefaultPostgresAllocatedStorageMetricName = "cro_postgres_current_allocated_storage"
	DefaultPostgresMaxMemoryMetricName        = "cro_postgres_max_memory"
	DefaultRedisMaintenanceMetricName         = "cro_redis_service_maintenance"
	DefaultRedisInfoMetricName                = "cro_redis_info"
	DefaultRedisAvailMetricName               = "cro_redis_available"
	DefaultRedisConnectionMetricName          = "cro_redis_connection"
	DefaultRedisStatusMetricName              = "cro_redis_status_phase"
	DefaultRedisDeletionMetricName            = "cro_redis_deletion_timestamp"
	DefaultRedisSnapshotStatusMetricName      = "cro_redis_snapshot_status_phase"
	DefaultBlobStorageStatusMetricName        = "cro_blobstorage_status_phase"

	BytesInGibiBytes = 1073741824
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

//SetMetric Set exports a Prometheus Gauge
func SetMetric(name string, labels map[string]string, value float64) {
	// set vector value
	gv, ok := MetricVecs[name]
	if ok {
		gv.With(labels).Set(value)
		logrus.Info(fmt.Sprintf("successfully set metric value for %s", name))
		return
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

	logrus.Info(fmt.Sprintf("successfully created new gauge vector metric %s", name))
}

//SetMetricCurrentTime Set current time wraps set metric
func SetMetricCurrentTime(name string, labels map[string]string) {
	SetMetric(name, labels, float64(time.Now().UnixNano())/1e9)
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
