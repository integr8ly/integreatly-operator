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
