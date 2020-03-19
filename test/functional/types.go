package functional

import (
	"github.com/aws/aws-sdk-go/service/elasticache/elasticacheiface"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3control/s3controliface"
)

//TODO update and add Fuse when Fuse Online is ready as per https://gitlab.cee.redhat.com/integreatly-qe/integreatly-test-cases/blob/master/tests/high-availability/f01-verify-all-aws-resources-are-created-in-aws.md
const (
	RDSSuffix                                = "postgres-integreatly"
	UPSResourceName                          = "ups-" + RDSSuffix
	CRResourceName                           = "codeready-" + RDSSuffix
	ClusterSSOResourceName                   = "rhsso-" + RDSSuffix
	ApplicationSSOResourceName               = "rhssouser-" + RDSSuffix
	ThreeScaleResourceName                   = "threescale" + RDSSuffix
	ThreeScaleNameElement                    = "rhmioperatorthre"
	S3Backup                                 = "rhmioperatorback"
	awsCredsNamespace                        = "redhat-rhmi-operator"
	awsCredsSecretName                       = "cloud-resources-aws-credentials"
	tagKeyClusterId                          = "integreatly.org/clusterID"
	clusterID                                = ""
	expectedElasticacheReplicationGroupCount = 2
)

var (
	expectedResources = []string{
		ThreeScaleResourceName,
		UPSResourceName,
		CRResourceName,
		ClusterSSOResourceName,
		ApplicationSSOResourceName,
	}
)

type awsClients struct {
	rdsClient         rdsiface.RDSAPI
	elasticacheClient elasticacheiface.ElastiCacheAPI
	s3Client          s3iface.S3API
	s3ControlClient   s3controliface.S3ControlAPI
	taggingClient     resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
}
