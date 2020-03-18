package functional

import (
	"github.com/aws/aws-sdk-go/service/elasticache/elasticacheiface"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"testing"
)

type ResourceManagerType string

const (
	tagKeyClusterID     = "integreatly.org/clusterID"
	loggingKeyClusterID = "cluster-id"
	loggingKeyManager   = "manager"
	//RDS Resource Names
	rdsSuffix         = "postgres-integreatly"
	threeScaleRDS     = "threescale" + rdsSuffix
	upsRDS            = "ups" + rdsSuffix
	codeReadyRDS      = "codeready" + rdsSuffix
	fuseOnlineRDS     //TODO update this line when Fuse Online is ready according to https://gitlab.cee.redhat.com/integreatly-qe/integreatly-test-cases/blob/master/tests/high-availability/f01-verify-all-aws-resources-are-created-in-aws.md
	clusterSSORDS     = "rhsso" + rdsSuffix
	applicationSSORDS = "rhssouser" + rdsSuffix

	//Elasticache Resource Names
	elasticacheNameContainsThreeScale = "rhmioperatorthre"

	//S3 Resource names
	s3BucketNameContainsThreeScale   = "rhmioperatorthre"
	s3BackupBucketNameContainsBackup = "rhmioperatorback"

	managerRDS         ResourceManagerType = "aws_rds"
	managerElasticache ResourceManagerType = "aws_elasticache"
	managerS3          ResourceManagerType = "aws_s3"
)

// Interfaces

type rdsClient interface {
	rdsiface.RDSAPI
}
type elasticacheClient interface {
	elasticacheiface.ElastiCacheAPI
}
type s3Client interface {
	s3iface.S3API
}
type taggingClient interface {
	resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
}

type ClusterResourceManager interface {
	GetName() string
	GetResourcesForCluster(clusterID string, tags map[string]string) ([]*ResourceOutput, error)
}
type ResourceCollection struct {
	Resources []*ResourceOutput
}

type ResourceOutput struct {
	Name        string
	ResourceARN string
}
type Resource struct {
	Name                  string
	ExpectedResourceCount int32
}

type TestCase struct {
	Description string
	test        func(t *testing.T)
}
