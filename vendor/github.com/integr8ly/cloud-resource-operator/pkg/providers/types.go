package providers

import (
	"context"
	"strconv"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
)

//go:generate moq -out types_moq.go . DeploymentDetails BlobStorageProvider SMTPCredentialsProvider

type ResourceType string

const (
	ManagedDeploymentType = "managed"

	AWSDeploymentStrategy       = "aws"
	OpenShiftDeploymentStrategy = "openshift"

	BlobStorageResourceType    ResourceType = "blobstorage"
	PostgresResourceType       ResourceType = "postgres"
	RedisResourceType          ResourceType = "redis"
	SMTPCredentialResourceType ResourceType = "smtpcredential"
)

type DeploymentDetails interface {
	Data() map[string][]byte
}

type SMTPCredentialSetInstance struct {
	DeploymentDetails DeploymentDetails
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
	CreateStorage(ctx context.Context, bs *v1alpha1.BlobStorage) (*BlobStorageInstance, v1alpha1.StatusMessage, error)
	DeleteStorage(ctx context.Context, bs *v1alpha1.BlobStorage) (v1alpha1.StatusMessage, error)
}

type SMTPCredentialsProvider interface {
	GetName() string
	SupportsStrategy(s string) bool
	CreateSMTPCredentials(ctx context.Context, smtpCreds *v1alpha1.SMTPCredentialSet) (*SMTPCredentialSetInstance, v1alpha1.StatusMessage, error)
	DeleteSMTPCredentials(ctx context.Context, smtpCreds *v1alpha1.SMTPCredentialSet) (v1alpha1.StatusMessage, error)
}

type RedisProvider interface {
	GetName() string
	SupportsStrategy(s string) bool
	CreateRedis(ctx context.Context, r *v1alpha1.Redis) (*RedisCluster, v1alpha1.StatusMessage, error)
	DeleteRedis(ctx context.Context, r *v1alpha1.Redis) (v1alpha1.StatusMessage, error)
}

type PostgresProvider interface {
	GetName() string
	SupportsStrategy(s string) bool
	CreatePostgres(ctx context.Context, ps *v1alpha1.Postgres) (*PostgresInstance, v1alpha1.StatusMessage, error)
	DeletePostgres(ctx context.Context, ps *v1alpha1.Postgres) (v1alpha1.StatusMessage, error)
}

// RedisDeploymentDetails provider specific details about the AWS Redis Cluster created
type RedisDeploymentDetails struct {
	URI  string
	Port int64
}

// Redis provider Data function
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
