package aws

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"time"
)

type EC2API interface {
	DescribeSubnets(ctx context.Context, input *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
	CreateTags(ctx context.Context, input *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
	DescribeVpcs(ctx context.Context, input *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	DescribeSecurityGroups(ctx context.Context, input *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error)
	DeleteSecurityGroup(ctx context.Context, input *ec2.DeleteSecurityGroupInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSecurityGroupOutput, error)
	CreateVpcPeeringConnection(ctx context.Context, input *ec2.CreateVpcPeeringConnectionInput, optFns ...func(*ec2.Options)) (*ec2.CreateVpcPeeringConnectionOutput, error)
	AcceptVpcPeeringConnection(ctx context.Context, input *ec2.AcceptVpcPeeringConnectionInput, optFns ...func(*ec2.Options)) (*ec2.AcceptVpcPeeringConnectionOutput, error)
	DeleteVpcPeeringConnection(ctx context.Context, input *ec2.DeleteVpcPeeringConnectionInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcPeeringConnectionOutput, error)
	DescribeRouteTables(ctx context.Context, input *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error)
	CreateRoute(ctx context.Context, input *ec2.CreateRouteInput, optFns ...func(*ec2.Options)) (*ec2.CreateRouteOutput, error)
	DeleteRoute(ctx context.Context, input *ec2.DeleteRouteInput, optFns ...func(*ec2.Options)) (*ec2.DeleteRouteOutput, error)
	CreateVpc(ctx context.Context, input *ec2.CreateVpcInput, optFns ...func(*ec2.Options)) (*ec2.CreateVpcOutput, error)
	DeleteVpc(ctx context.Context, input *ec2.DeleteVpcInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcOutput, error)
	WaitUntilVpcExists(ctx context.Context, input *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) error
	CreateSubnet(ctx context.Context, input *ec2.CreateSubnetInput, optFns ...func(*ec2.Options)) (*ec2.CreateSubnetOutput, error)
	DescribeInstanceTypeOfferings(ctx context.Context, input *ec2.DescribeInstanceTypeOfferingsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypeOfferingsOutput, error)
	DescribeAvailabilityZones(ctx context.Context, input *ec2.DescribeAvailabilityZonesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeAvailabilityZonesOutput, error)
	CreateSecurityGroup(ctx context.Context, input *ec2.CreateSecurityGroupInput, optFns ...func(*ec2.Options)) (*ec2.CreateSecurityGroupOutput, error)
	DeleteSubnet(ctx context.Context, input *ec2.DeleteSubnetInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSubnetOutput, error)
	AuthorizeSecurityGroupIngress(ctx context.Context, input *ec2.AuthorizeSecurityGroupIngressInput, optFns ...func(*ec2.Options)) (*ec2.AuthorizeSecurityGroupIngressOutput, error)
	DescribeVpcPeeringConnections(ctx context.Context, input *ec2.DescribeVpcPeeringConnectionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcPeeringConnectionsOutput, error)
	DescribeInstanceTypes(ctx context.Context, input *ec2.DescribeInstanceTypesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error)
}
type VpcWaiter interface {
	Wait(ctx context.Context, params *ec2.DescribeVpcsInput, maxWaitTime time.Duration, optFns ...func(*ec2.VpcExistsWaiterOptions)) error
}

type RDSAPI interface {
	DescribeDBInstances(ctx context.Context, input *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error)
	ModifyDBSubnetGroup(ctx context.Context, input *rds.ModifyDBSubnetGroupInput, optFns ...func(*rds.Options)) (*rds.ModifyDBSubnetGroupOutput, error)
	ListTagsForResource(ctx context.Context, input *rds.ListTagsForResourceInput, optFns ...func(*rds.Options)) (*rds.ListTagsForResourceOutput, error)
	RemoveTagsFromResource(ctx context.Context, input *rds.RemoveTagsFromResourceInput, optFns ...func(*rds.Options)) (*rds.RemoveTagsFromResourceOutput, error)
	DeleteDBSubnetGroup(ctx context.Context, input *rds.DeleteDBSubnetGroupInput, optFns ...func(*rds.Options)) (*rds.DeleteDBSubnetGroupOutput, error)
	AddTagsToResource(ctx context.Context, input *rds.AddTagsToResourceInput, optFns ...func(*rds.Options)) (*rds.AddTagsToResourceOutput, error)
	DescribeDBSnapshots(ctx context.Context, input *rds.DescribeDBSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSnapshotsOutput, error)
	DescribeDBSubnetGroups(ctx context.Context, input *rds.DescribeDBSubnetGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSubnetGroupsOutput, error)
	DescribePendingMaintenanceActions(ctx context.Context, input *rds.DescribePendingMaintenanceActionsInput, optFns ...func(*rds.Options)) (*rds.DescribePendingMaintenanceActionsOutput, error)
	ApplyPendingMaintenanceAction(ctx context.Context, input *rds.ApplyPendingMaintenanceActionInput, optFns ...func(*rds.Options)) (*rds.ApplyPendingMaintenanceActionOutput, error)
	ModifyDBInstance(ctx context.Context, input *rds.ModifyDBInstanceInput, optFns ...func(*rds.Options)) (*rds.ModifyDBInstanceOutput, error)
	CreateDBSubnetGroup(ctx context.Context, input *rds.CreateDBSubnetGroupInput, optFns ...func(*rds.Options)) (*rds.CreateDBSubnetGroupOutput, error)
	CreateDBInstance(ctx context.Context, cfg *rds.CreateDBInstanceInput, optFns ...func(*rds.Options)) (*rds.CreateDBInstanceOutput, error)
	DeleteDBInstance(ctx context.Context, config *rds.DeleteDBInstanceInput, optFns ...func(*rds.Options)) (*rds.DeleteDBInstanceOutput, error)
	CreateDBSnapshot(ctx context.Context, r *rds.CreateDBSnapshotInput, optFns ...func(*rds.Options)) (*rds.CreateDBSnapshotOutput, error)
	DeleteDBSnapshot(ctx context.Context, input *rds.DeleteDBSnapshotInput, optFns ...func(*rds.Options)) (*rds.DeleteDBSnapshotOutput, error)
	DescribeDBEngineVersions(ctx context.Context, input *rds.DescribeDBEngineVersionsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBEngineVersionsOutput, error)
}

type ElastiCacheAPI interface {
	DescribeCacheClusters(ctx context.Context, input *elasticache.DescribeCacheClustersInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeCacheClustersOutput, error)
	ModifyCacheSubnetGroup(ctx context.Context, input *elasticache.ModifyCacheSubnetGroupInput, optFns ...func(*elasticache.Options)) (*elasticache.ModifyCacheSubnetGroupOutput, error)
	DeleteCacheSubnetGroup(ctx context.Context, input *elasticache.DeleteCacheSubnetGroupInput, optFns ...func(*elasticache.Options)) (*elasticache.DeleteCacheSubnetGroupOutput, error)
	DescribeCacheSubnetGroups(ctx context.Context, input *elasticache.DescribeCacheSubnetGroupsInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeCacheSubnetGroupsOutput, error)
	DescribeReplicationGroups(ctx context.Context, input *elasticache.DescribeReplicationGroupsInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeReplicationGroupsOutput, error)
	DescribeSnapshots(ctx context.Context, input *elasticache.DescribeSnapshotsInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeSnapshotsOutput, error)
	CreateSnapshot(ctx context.Context, input *elasticache.CreateSnapshotInput, optFns ...func(*elasticache.Options)) (*elasticache.CreateSnapshotOutput, error)
	DeleteSnapshot(ctx context.Context, input *elasticache.DeleteSnapshotInput, optFns ...func(*elasticache.Options)) (*elasticache.DeleteSnapshotOutput, error)
	DescribeUpdateActions(ctx context.Context, input *elasticache.DescribeUpdateActionsInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeUpdateActionsOutput, error)
	ModifyReplicationGroup(ctx context.Context, input *elasticache.ModifyReplicationGroupInput, optFns ...func(*elasticache.Options)) (*elasticache.ModifyReplicationGroupOutput, error)
	BatchApplyUpdateAction(ctx context.Context, input *elasticache.BatchApplyUpdateActionInput, optFns ...func(*elasticache.Options)) (*elasticache.BatchApplyUpdateActionOutput, error)
	AddTagsToResource(ctx context.Context, input *elasticache.AddTagsToResourceInput, optFns ...func(*elasticache.Options)) (*elasticache.AddTagsToResourceOutput, error)
	CreateReplicationGroup(ctx context.Context, input *elasticache.CreateReplicationGroupInput, optFns ...func(*elasticache.Options)) (*elasticache.CreateReplicationGroupOutput, error)
	CreateCacheSubnetGroup(ctx context.Context, input *elasticache.CreateCacheSubnetGroupInput, optFns ...func(*elasticache.Options)) (*elasticache.CreateCacheSubnetGroupOutput, error)
	DeleteReplicationGroup(ctx context.Context, input *elasticache.DeleteReplicationGroupInput, optFns ...func(*elasticache.Options)) (*elasticache.DeleteReplicationGroupOutput, error)
}

type S3API interface {
	CreateBucket(ctx context.Context, input *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error)
	DeleteBucket(ctx context.Context, input *s3.DeleteBucketInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, error)
	ListBuckets(ctx context.Context, input *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error)
	PutObject(ctx context.Context, input *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(ctx context.Context, input *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	ListObjectsV2(ctx context.Context, input *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	PutBucketTagging(ctx context.Context, input *s3.PutBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.PutBucketTaggingOutput, error)
	DeleteObjects(ctx context.Context, input *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error)
	PutPublicAccessBlock(ctx context.Context, input *s3.PutPublicAccessBlockInput, optFns ...func(*s3.Options)) (*s3.PutPublicAccessBlockOutput, error)
	PutBucketEncryption(ctx context.Context, input *s3.PutBucketEncryptionInput, optFns ...func(*s3.Options)) (*s3.PutBucketEncryptionOutput, error)
}

type CloudWatchAPI interface {
	PutMetricData(ctx context.Context, input *cloudwatch.PutMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.PutMetricDataOutput, error)
	GetMetricStatistics(ctx context.Context, input *cloudwatch.GetMetricStatisticsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricStatisticsOutput, error)
	ListMetrics(ctx context.Context, input *cloudwatch.ListMetricsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.ListMetricsOutput, error)
	DescribeAlarms(ctx context.Context, input *cloudwatch.DescribeAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmsOutput, error)
	PutMetricAlarm(ctx context.Context, input *cloudwatch.PutMetricAlarmInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.PutMetricAlarmOutput, error)
	DeleteAlarms(ctx context.Context, input *cloudwatch.DeleteAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DeleteAlarmsOutput, error)
	GetMetricData(ctx context.Context, input *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error)
}
type STSAPI interface {
	GetCallerIdentity(ctx context.Context, input *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
	AssumeRole(ctx context.Context, input *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
	GetFederationToken(ctx context.Context, input *sts.GetFederationTokenInput, optFns ...func(*sts.Options)) (*sts.GetFederationTokenOutput, error)
}
