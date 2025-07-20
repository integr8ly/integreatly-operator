package resources

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	customMetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

type MonitoringResourceType string

const (
	BytesInGibiBytes                        = 1073741824
	DefaultBlobStorageStatusMetricName      = "cro_blobstorage_status_phase"
	DefaultPostgresAvailMetricName          = "cro_postgres_available"
	DefaultPostgresConnectionMetricName     = "cro_postgres_connection"
	DefaultPostgresDeletionMetricName       = "cro_postgres_deletion_timestamp"
	DefaultPostgresInfoMetricName           = "cro_postgres_info"
	DefaultPostgresMaintenanceMetricName    = "cro_postgres_service_maintenance"
	DefaultPostgresSnapshotStatusMetricName = "cro_postgres_snapshot_status_phase"
	DefaultPostgresStatusMetricName         = "cro_postgres_status_phase"
	DefaultRedisAvailMetricName             = "cro_redis_available"
	DefaultRedisConnectionMetricName        = "cro_redis_connection"
	DefaultRedisDeletionMetricName          = "cro_redis_deletion_timestamp"
	DefaultRedisInfoMetricName              = "cro_redis_info"
	DefaultRedisMaintenanceMetricName       = "cro_redis_service_maintenance"
	DefaultRedisSnapshotNotAvailable        = "cro_redis_snapshot_not_found"
	DefaultRedisSnapshotStatusMetricName    = "cro_redis_snapshot_status_phase"
	DefaultRedisStatusMetricName            = "cro_redis_status_phase"
	DefaultSTSCredentialsSecretMetricName   = "cro_sts_credentials_secret" // #nosec G101 -- false positive (ref: https://securego.io/docs/rules/g101.html)
	DefaultVpcActionMetricName              = "cro_vpc_action"

	MonitoringResourceTypeRedisInstance    MonitoringResourceType = "redis_instance"
	MonitoringResourceTypeCloudsqlDatabase MonitoringResourceType = "cloudsql_database"

	PostgresFreeStorageAverageMetricName    = "cro_postgres_free_storage_average"
	PostgresCPUUtilizationAverageMetricName = "cro_postgres_cpu_utilization_average"
	PostgresFreeableMemoryAverageMetricName = "cro_postgres_freeable_memory_average"
	PostgresMaxMemoryMetricName             = "cro_postgres_max_memory"
	PostgresAllocatedStorageMetricName      = "cro_postgres_current_allocated_storage"
	PostgresUpgradeAvailableMetricName      = "cro_postgres_upgrade_available"

	RedisMemoryUsagePercentageAverageMetricName = "cro_redis_memory_usage_percentage_average"
	RedisFreeableMemoryAverageMetricName        = "cro_redis_freeable_memory_average"
	RedisCPUUtilizationAverageMetricName        = "cro_redis_cpu_utilization_average"
	RedisEngineCPUUtilizationAverageMetricName  = "cro_redis_engine_cpu_utilization_average"
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
			time.Sleep(time.Duration(3600) * time.Second)
		}
	}()
}

func ResetMetric(name string) {
	logrus.Info(fmt.Sprintf("Resetting metric %s", name))
	// set vector value
	gv, ok := MetricVecs[name]
	if ok {
		gv.Reset()
		logrus.Info(fmt.Sprintf("successfully reset metric value for %s", name))
		return
	}
}

// SetMetric Set exports a Prometheus Gauge
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

// SetMetricCurrentTime Set current time wraps set metric
func SetMetricCurrentTime(name string, labels map[string]string) {
	SetMetric(name, labels, float64(time.Now().UnixNano())/1e9)
}

// SetVpcAction sets cro_vpc_action metric
func SetVpcAction(action string, status string, err string, code float64) {
	SetMetric(DefaultVpcActionMetricName,
		map[string]string{
			"action": action,
			"status": status,
			"error":  err,
		}, code)
}

// ResetVpcAction resets cro_vpc_action metric
func ResetVpcAction() {
	if val, ok := MetricVecs[DefaultVpcActionMetricName]; ok {
		val.Reset()
	}
}

// SetSTSCredentialsSecretMetric sets cro_sts_credentials_secret metric
func SetSTSCredentialsSecretMetric(ns string, err error) {
	labels := map[string]string{
		"namespace": ns,
		"error":     err.Error(),
	}
	SetMetric(DefaultSTSCredentialsSecretMetricName, labels, 1)
}

// ResetSTSCredentialsSecretMetric resets cro_sts_credentials_secret metric
func ResetSTSCredentialsSecretMetric() {
	if val, ok := MetricVecs[DefaultSTSCredentialsSecretMetricName]; ok {
		val.Reset()
	}
}

func IsCompoundMetric(metric string) bool {
	for _, compoundMetric := range getCompoundMetrics() {
		if metric == compoundMetric {
			return true
		}
	}
	return false
}

func IsComputedCpuMetric(metric string) bool {
	for _, computedCpuMetric := range getComputedCpuMetrics() {
		if metric == computedCpuMetric {
			return true
		}
	}
	return false
}

func getCompoundMetrics() []string {
	return []string{
		RedisFreeableMemoryAverageMetricName,
		PostgresFreeStorageAverageMetricName,
		PostgresFreeableMemoryAverageMetricName,
	}
}

func getComputedCpuMetrics() []string {
	return []string{
		RedisCPUUtilizationAverageMetricName,
		RedisEngineCPUUtilizationAverageMetricName,
	}
}
