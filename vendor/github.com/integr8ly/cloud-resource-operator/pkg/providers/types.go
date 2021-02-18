package providers

import (
	"context"
	"strconv"
	"time"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	croType "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
)

//go:generate moq -out types_moq.go . DeploymentDetails BlobStorageProvider
type ResourceType string

const (
	ManagedDeploymentType = "managed"

	AWSDeploymentStrategy       = "aws"
	OpenShiftDeploymentStrategy = "openshift"

	BlobStorageResourceType ResourceType = "blobstorage"
	PostgresResourceType    ResourceType = "postgres"
	RedisResourceType       ResourceType = "redis"
	NetworkResourceType     ResourceType = "_network"
)

type DeploymentDetails interface {
	Data() map[string][]byte
}

type BlobStorageInstance struct {
	DeploymentDetails DeploymentDetails
}

type RedisCluster struct {
	DeploymentDetails DeploymentDetails
}

type PostgresInstance struct {
	DeploymentDetails DeploymentDetails
}

type PostgresSnapshotInstance struct {
	Name string
}

type RedisSnapshotInstance struct {
	Name string
}

type BlobStorageProvider interface {
	GetName() string
	SupportsStrategy(s string) bool
	GetReconcileTime(bs *v1alpha1.BlobStorage) time.Duration
	CreateStorage(ctx context.Context, bs *v1alpha1.BlobStorage) (*BlobStorageInstance, croType.StatusMessage, error)
	DeleteStorage(ctx context.Context, bs *v1alpha1.BlobStorage) (croType.StatusMessage, error)
}

type RedisProvider interface {
	GetName() string
	SupportsStrategy(s string) bool
	GetReconcileTime(r *v1alpha1.Redis) time.Duration
	CreateRedis(ctx context.Context, r *v1alpha1.Redis) (*RedisCluster, croType.StatusMessage, error)
	DeleteRedis(ctx context.Context, r *v1alpha1.Redis) (croType.StatusMessage, error)
}

type PostgresProvider interface {
	GetName() string
	SupportsStrategy(s string) bool
	GetReconcileTime(ps *v1alpha1.Postgres) time.Duration
	CreatePostgres(ctx context.Context, ps *v1alpha1.Postgres) (*PostgresInstance, croType.StatusMessage, error)
	DeletePostgres(ctx context.Context, ps *v1alpha1.Postgres) (croType.StatusMessage, error)
}

type PostgresSnapshotProvider interface {
	GetName() string
	SupportsStrategy(s string) bool
	GetReconcileTime(snapshot *v1alpha1.PostgresSnapshot) time.Duration
	CreatePostgresSnapshot(ctx context.Context, snapshot *v1alpha1.PostgresSnapshot, postgres *v1alpha1.Postgres) (*PostgresSnapshotInstance, croType.StatusMessage, error)
	DeletePostgresSnapshot(ctx context.Context, snapshot *v1alpha1.PostgresSnapshot, posgres *v1alpha1.Postgres) (croType.StatusMessage, error)
}

type RedisSnapshotProvider interface {
	GetName() string
	SupportsStrategy(s string) bool
	GetReconcileTime(snapshot *v1alpha1.RedisSnapshot) time.Duration
	CreateRedisSnapshot(ctx context.Context, snapshot *v1alpha1.RedisSnapshot, redis *v1alpha1.Redis) (*RedisSnapshotInstance, croType.StatusMessage, error)
	DeleteRedisSnapshot(ctx context.Context, snapshot *v1alpha1.RedisSnapshot, redis *v1alpha1.Redis) (croType.StatusMessage, error)
}

// RedisDeploymentDetails provider specific details about the AWS Redis Cluster created
type RedisDeploymentDetails struct {
	URI  string
	Port int64
}

//Data Redis provider Data function
func (r *RedisDeploymentDetails) Data() map[string][]byte {
	return map[string][]byte{
		"uri":  []byte(r.URI),
		"port": []byte(strconv.FormatInt(r.Port, 10)),
	}
}

type PostgresDeploymentDetails struct {
	Username string
	Password string
	Host     string
	Database string
	Port     int
}

func (d *PostgresDeploymentDetails) Data() map[string][]byte {
	return map[string][]byte{
		"username": []byte(d.Username),
		"password": []byte(d.Password),
		"host":     []byte(d.Host),
		"database": []byte(d.Database),
		"port":     []byte(strconv.Itoa(d.Port)),
	}
}

// GenericCloudMetric is a wrapper to represent provider specific metrics generically
type GenericCloudMetric struct {
	Name   string
	Labels map[string]string
	Value  float64
}

// CloudProviderMetricType is used to declare a generic type of metric
// it maps provider specific metrics to metrics we expose in prometheus
type CloudProviderMetricType struct {
	//PromethuesMetricName the name of the metric exposed via cro
	PromethuesMetricName string
	//ProviderMetricName the metric we scrape from the cloud provider
	ProviderMetricName string
	//Statistic the type of metric value we return e.g. Average, Sum, Max, Min etc.
	Statistic string
}

// ScrapeMetricsData is a wrapper for output of scrape metrics
type ScrapeMetricsData struct {
	// Metrics is an array of built cloud metrics from scraping a provider
	Metrics []*GenericCloudMetric
}

type RedisMetricsProvider interface {
	SupportsStrategy(s string) bool
	ScrapeRedisMetrics(ctx context.Context, redis *v1alpha1.Redis, metricsTypes []CloudProviderMetricType) (*ScrapeMetricsData, error)
}

type PostgresMetricsProvider interface {
	SupportsStrategy(s string) bool
	ScrapePostgresMetrics(ctx context.Context, postgres *v1alpha1.Postgres, metricTypes []CloudProviderMetricType) (*ScrapeMetricsData, error)
}
