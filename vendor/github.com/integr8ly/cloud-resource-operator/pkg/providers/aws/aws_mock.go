package aws

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/stretchr/testify/mock"
	"sync"
	"time"
)

var (
	lockMockEc2ClientDescribeRouteTables       sync.RWMutex
	lockMockEc2ClientDescribeSecurityGroups    sync.RWMutex
	lockMockEc2ClientDescribeSubnets           sync.RWMutex
	lockMockEc2ClientDescribeAvailabilityZones sync.RWMutex
	lockMockEc2ClientDescribeVpcs              sync.RWMutex
	lockMockEc2ClientCreateRoute               sync.RWMutex
)

// Define a mock type that implements the EC2API interface.
type mock_Ec2Client struct {
	mock.Mock
	firstSubnet     *ec2types.Subnet
	secondSubnet    *ec2types.Subnet
	subnets         []ec2types.Subnet
	vpcs            []ec2types.Vpc
	vpc             *ec2types.Vpc
	secGroups       []ec2types.SecurityGroup
	azs             []ec2types.AvailabilityZone
	wantErrList     bool
	returnSecondSub bool

	// Optional call tracking if needed
	calls struct {
		DescribeRouteTables       []ec2.DescribeRouteTablesInput
		DescribeSecurityGroups    []ec2.DescribeSecurityGroupsInput
		DescribeAvailabilityZones []ec2.DescribeAvailabilityZonesInput
		DescribeSubnets           []struct {
			Input *ec2.DescribeSubnetsInput
		}
		DescribeVpcs []struct {
			Input *ec2.DescribeVpcsInput
		}
		CreateRoute []ec2.CreateRouteInput
	}
}

func (m *mock_Ec2Client) CreateTags(ctx context.Context, input *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*ec2.CreateTagsOutput), args.Error(1)
}

func (m *mock_Ec2Client) CreateVpcPeeringConnection(ctx context.Context, input *ec2.CreateVpcPeeringConnectionInput, optFns ...func(*ec2.Options)) (*ec2.CreateVpcPeeringConnectionOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*ec2.CreateVpcPeeringConnectionOutput), args.Error(1)
}

func (m *mock_Ec2Client) DeleteSubnet(ctx context.Context, input *ec2.DeleteSubnetInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSubnetOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*ec2.DeleteSubnetOutput), args.Error(1)
}

func (m *mock_Ec2Client) AuthorizeSecurityGroupIngress(ctx context.Context, input *ec2.AuthorizeSecurityGroupIngressInput, optFns ...func(*ec2.Options)) (*ec2.AuthorizeSecurityGroupIngressOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*ec2.AuthorizeSecurityGroupIngressOutput), args.Error(1)
}

func (m *mock_Ec2Client) DescribeInstanceTypes(ctx context.Context, input *ec2.DescribeInstanceTypesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*ec2.DescribeInstanceTypesOutput), args.Error(1)
}

// Implement the DescribeVpcs method.
func (m *mock_Ec2Client) DescribeVpcs(ctx context.Context, input *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*ec2.DescribeVpcsOutput), args.Error(1)
}

// Implement the DescribeSubnets method.
func (m *mock_Ec2Client) DescribeSubnets(ctx context.Context, input *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	args := m.Called(ctx, input, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ec2.DescribeSubnetsOutput), args.Error(1)
}

// Implement the CreateSecurityGroup method.
func (m *mock_Ec2Client) CreateSecurityGroup(ctx context.Context, input *ec2.CreateSecurityGroupInput, optFns ...func(*ec2.Options)) (*ec2.CreateSecurityGroupOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*ec2.CreateSecurityGroupOutput), args.Error(1)
}

// Implement the DeleteSecurityGroup method.
func (m *mock_Ec2Client) DeleteSecurityGroup(ctx context.Context, input *ec2.DeleteSecurityGroupInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSecurityGroupOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*ec2.DeleteSecurityGroupOutput), args.Error(1)
}

// Implement the DescribeSecurityGroups method.
func (m *mock_Ec2Client) DescribeSecurityGroups(ctx context.Context, input *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*ec2.DescribeSecurityGroupsOutput), args.Error(1)
}

// Implement the CreateVpc method.
func (m *mock_Ec2Client) CreateVpc(ctx context.Context, input *ec2.CreateVpcInput, optFns ...func(*ec2.Options)) (*ec2.CreateVpcOutput, error) {
	args := m.Called(ctx, input, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ec2.CreateVpcOutput), args.Error(1)
}

// Implement the DeleteVpc method.
func (m *mock_Ec2Client) DeleteVpc(ctx context.Context, input *ec2.DeleteVpcInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*ec2.DeleteVpcOutput), args.Error(1)
}

// Implement the WaitUntilVpcExists method.
func (m *mock_Ec2Client) WaitUntilVpcExists(ctx context.Context, input *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) error {
	args := m.Called(ctx, input, optFns)
	return args.Error(0)
}

type MockVpcWaiter struct {
	mock.Mock
}

func (m *MockVpcWaiter) Wait(ctx context.Context, input *ec2.DescribeVpcsInput, maxWaitTime time.Duration, optFns ...func(*ec2.VpcExistsWaiterOptions)) error {
	args := m.Called(ctx, input, maxWaitTime)
	return args.Error(0)
}

func (m *mock_Ec2Client) Wait(ctx context.Context, input *ec2.DescribeVpcsInput, maxWait time.Duration, optFns ...func(*ec2.VpcExistsWaiterOptions)) error {
	args := m.Called(ctx, input, maxWait)
	return args.Error(0)
}

// Implement the CreateSubnet method.
func (m *mock_Ec2Client) CreateSubnet(ctx context.Context, input *ec2.CreateSubnetInput, optFns ...func(*ec2.Options)) (*ec2.CreateSubnetOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*ec2.CreateSubnetOutput), args.Error(1)
}

// Implement the DescribeInstanceTypeOfferings method.
func (m *mock_Ec2Client) DescribeInstanceTypeOfferings(ctx context.Context, input *ec2.DescribeInstanceTypeOfferingsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypeOfferingsOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*ec2.DescribeInstanceTypeOfferingsOutput), args.Error(1)
}

// Implement the DescribeAvailabilityZones method.
func (m *mock_Ec2Client) DescribeAvailabilityZones(ctx context.Context, input *ec2.DescribeAvailabilityZonesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeAvailabilityZonesOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*ec2.DescribeAvailabilityZonesOutput), args.Error(1)
}

func (m *mock_Ec2Client) DescribeVpcPeeringConnections(ctx context.Context, input *ec2.DescribeVpcPeeringConnectionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcPeeringConnectionsOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*ec2.DescribeVpcPeeringConnectionsOutput), args.Error(1)
}

func (m *mock_Ec2Client) AcceptVpcPeeringConnection(ctx context.Context, input *ec2.AcceptVpcPeeringConnectionInput, optFns ...func(*ec2.Options)) (*ec2.AcceptVpcPeeringConnectionOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*ec2.AcceptVpcPeeringConnectionOutput), args.Error(1)
}

// Implement the DeleteVpcPeeringConnection method.
func (m *mock_Ec2Client) DeleteVpcPeeringConnection(ctx context.Context, input *ec2.DeleteVpcPeeringConnectionInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcPeeringConnectionOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*ec2.DeleteVpcPeeringConnectionOutput), args.Error(1)
}

// Implement the DescribeRouteTables method.
func (m *mock_Ec2Client) DescribeRouteTables(ctx context.Context, input *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*ec2.DescribeRouteTablesOutput), args.Error(1)
}

// Implement the CreateRoute method.
func (m *mock_Ec2Client) CreateRoute(ctx context.Context, input *ec2.CreateRouteInput, optFns ...func(*ec2.Options)) (*ec2.CreateRouteOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*ec2.CreateRouteOutput), args.Error(1)
}

// Implement the DeleteRoute method.
func (m *mock_Ec2Client) DeleteRoute(ctx context.Context, input *ec2.DeleteRouteInput, optFns ...func(*ec2.Options)) (*ec2.DeleteRouteOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*ec2.DeleteRouteOutput), args.Error(1)
}

func (m *mock_Ec2Client) DescribeSecurityGroupsCalls() []struct {
	Groups *ec2.DescribeSecurityGroupsInput
} {
	var calls []struct {
		Groups *ec2.DescribeSecurityGroupsInput
	}

	lockMockEc2ClientDescribeSecurityGroups.RLock()
	for _, groupInput := range m.calls.DescribeSecurityGroups {
		currentGroup := groupInput //fix gosec error G601 (CWE-118): Implicit memory aliasing in for loop.
		calls = append(calls, struct {
			Groups *ec2.DescribeSecurityGroupsInput
		}{
			Groups: &currentGroup,
		})
	}
	lockMockEc2ClientDescribeSecurityGroups.RUnlock()

	return calls
}

func (m *mock_Ec2Client) DescribeRouteTablesCalls() []struct {
	Tables *ec2.DescribeRouteTablesInput
} {
	var calls []struct {
		Tables *ec2.DescribeRouteTablesInput
	}
	lockMockEc2ClientDescribeRouteTables.RLock()
	for _, tableInput := range m.calls.DescribeRouteTables {
		currentTable := tableInput //fix gosec error G601 (CWE-118): Implicit memory aliasing in for loop.
		calls = append(calls, struct {
			Tables *ec2.DescribeRouteTablesInput
		}{
			Tables: &currentTable,
		})
	}
	lockMockEc2ClientDescribeRouteTables.RUnlock()

	return calls
}

func (m *mock_Ec2Client) CreateRouteCalls() []struct {
	Route *ec2.CreateRouteInput
} {
	var calls []struct {
		Route *ec2.CreateRouteInput
	}
	lockMockEc2ClientCreateRoute.RLock()
	for _, routeInput := range m.calls.CreateRoute {
		currentRoute := routeInput //fix gosec error G601 (CWE-118): Implicit memory aliasing in for loop.
		calls = append(calls, struct {
			Route *ec2.CreateRouteInput
		}{
			Route: &currentRoute,
		})
	}
	lockMockEc2ClientCreateRoute.RUnlock()

	return calls
}

// RDS mocks
type mock_RdsClient struct {
	mock.Mock
	DescribeDBSnapshotsFunc func(ctx context.Context, input *rds.DescribeDBSnapshotsInput, opts ...func(*rds.Options)) (*rds.DescribeDBSnapshotsOutput, error)
	CreateDBSnapshotFunc    func(ctx context.Context, input *rds.CreateDBSnapshotInput, opts ...func(*rds.Options)) (*rds.CreateDBSnapshotOutput, error)
	DeleteDBSnapshotFunc    func(ctx context.Context, in1 *rds.DeleteDBSnapshotInput, opts ...func(*rds.Options)) (*rds.DeleteDBSnapshotOutput, error)
	calls                   struct {
		DescribeDBSnapshots []struct {
			Ctx   context.Context
			Input *rds.DescribeDBSnapshotsInput
		}
		CreateDBSnapshot []struct {
			Ctx   context.Context
			Input *rds.CreateDBSnapshotInput
		}
		DeleteDBSnapshot []struct {
			Ctx   context.Context
			Input *rds.DeleteDBSnapshotInput
		}
	}
}

func (m *mock_RdsClient) DescribeDBInstances(ctx context.Context, input *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*rds.DescribeDBInstancesOutput), args.Error(1)
}

func (m *mock_RdsClient) ModifyDBSubnetGroup(ctx context.Context, input *rds.ModifyDBSubnetGroupInput, optFns ...func(*rds.Options)) (*rds.ModifyDBSubnetGroupOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*rds.ModifyDBSubnetGroupOutput), args.Error(1)
}

func (m *mock_RdsClient) ListTagsForResource(ctx context.Context, input *rds.ListTagsForResourceInput, optFns ...func(*rds.Options)) (*rds.ListTagsForResourceOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*rds.ListTagsForResourceOutput), args.Error(1)
}

func (m *mock_RdsClient) RemoveTagsFromResource(ctx context.Context, input *rds.RemoveTagsFromResourceInput, optFns ...func(*rds.Options)) (*rds.RemoveTagsFromResourceOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*rds.RemoveTagsFromResourceOutput), args.Error(1)
}

func (m *mock_RdsClient) DeleteDBSubnetGroup(ctx context.Context, input *rds.DeleteDBSubnetGroupInput, optFns ...func(*rds.Options)) (*rds.DeleteDBSubnetGroupOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*rds.DeleteDBSubnetGroupOutput), args.Error(1)
}

func (m *mock_RdsClient) AddTagsToResource(ctx context.Context, input *rds.AddTagsToResourceInput, optFns ...func(*rds.Options)) (*rds.AddTagsToResourceOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*rds.AddTagsToResourceOutput), args.Error(1)
}

func (m *mock_RdsClient) DescribeDBSnapshots(ctx context.Context, input *rds.DescribeDBSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSnapshotsOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*rds.DescribeDBSnapshotsOutput), args.Error(1)
}

func (m *mock_RdsClient) DescribeDBSubnetGroups(ctx context.Context, input *rds.DescribeDBSubnetGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSubnetGroupsOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*rds.DescribeDBSubnetGroupsOutput), args.Error(1)
}

func (m *mock_RdsClient) DescribePendingMaintenanceActions(ctx context.Context, input *rds.DescribePendingMaintenanceActionsInput, optFns ...func(*rds.Options)) (*rds.DescribePendingMaintenanceActionsOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*rds.DescribePendingMaintenanceActionsOutput), args.Error(1)
}

func (m *mock_RdsClient) ApplyPendingMaintenanceAction(ctx context.Context, input *rds.ApplyPendingMaintenanceActionInput, optFns ...func(*rds.Options)) (*rds.ApplyPendingMaintenanceActionOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*rds.ApplyPendingMaintenanceActionOutput), args.Error(1)
}

func (m *mock_RdsClient) ModifyDBInstance(ctx context.Context, input *rds.ModifyDBInstanceInput, optFns ...func(*rds.Options)) (*rds.ModifyDBInstanceOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*rds.ModifyDBInstanceOutput), args.Error(1)
}

func (m *mock_RdsClient) CreateDBSubnetGroup(ctx context.Context, input *rds.CreateDBSubnetGroupInput, optFns ...func(*rds.Options)) (*rds.CreateDBSubnetGroupOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*rds.CreateDBSubnetGroupOutput), args.Error(1)
}

func (m *mock_RdsClient) CreateDBInstance(ctx context.Context, input *rds.CreateDBInstanceInput, optFns ...func(*rds.Options)) (*rds.CreateDBInstanceOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*rds.CreateDBInstanceOutput), args.Error(1)
}

func (m *mock_RdsClient) DeleteDBInstance(ctx context.Context, input *rds.DeleteDBInstanceInput, optFns ...func(*rds.Options)) (*rds.DeleteDBInstanceOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*rds.DeleteDBInstanceOutput), args.Error(1)
}

func (m *mock_RdsClient) CreateDBSnapshot(ctx context.Context, input *rds.CreateDBSnapshotInput, optFns ...func(*rds.Options)) (*rds.CreateDBSnapshotOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*rds.CreateDBSnapshotOutput), args.Error(1)
}

func (m *mock_RdsClient) DeleteDBSnapshot(ctx context.Context, input *rds.DeleteDBSnapshotInput, optFns ...func(*rds.Options)) (*rds.DeleteDBSnapshotOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*rds.DeleteDBSnapshotOutput), args.Error(1)
}

func (m *mock_RdsClient) DescribeDBEngineVersions(ctx context.Context, input *rds.DescribeDBEngineVersionsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBEngineVersionsOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*rds.DescribeDBEngineVersionsOutput), args.Error(1)
}

// Elasticache Mock
type mock_ElasticacheClient struct {
	mock.Mock
	calls struct {
		DescribeSnapshots []struct {
			In1 *elasticache.DescribeSnapshotsInput
		}
		DescribeReplicationGroups []struct {
			In1 *elasticache.DescribeReplicationGroupsInput
		}
		CreateSnapshot []struct {
			In1 *elasticache.CreateSnapshotInput
		}
		DeleteSnapshot []struct {
			In1 *elasticache.DeleteSnapshotInput
		}
		DescribeUpdateActions []struct {
			In1 *elasticache.DescribeUpdateActionsInput
		}
		ModifyReplicationGroup []struct {
			In1 *elasticache.ModifyReplicationGroupInput
		}
		BatchApplyUpdateAction []struct {
			In1 *elasticache.BatchApplyUpdateActionInput
		}
		CreateReplicationGroup []struct {
			In1 *elasticache.CreateReplicationGroupInput
		}
	}
}

func (m *mock_ElasticacheClient) DeleteReplicationGroup(ctx context.Context, input *elasticache.DeleteReplicationGroupInput, optFns ...func(*elasticache.Options)) (*elasticache.DeleteReplicationGroupOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*elasticache.DeleteReplicationGroupOutput), args.Error(1)
}

func (m *mock_ElasticacheClient) CreateCacheSubnetGroup(ctx context.Context, input *elasticache.CreateCacheSubnetGroupInput, optFns ...func(*elasticache.Options)) (*elasticache.CreateCacheSubnetGroupOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*elasticache.CreateCacheSubnetGroupOutput), args.Error(1)
}

func (m *mock_ElasticacheClient) ModifyCacheSubnetGroup(ctx context.Context, input *elasticache.ModifyCacheSubnetGroupInput, optFns ...func(*elasticache.Options)) (*elasticache.ModifyCacheSubnetGroupOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*elasticache.ModifyCacheSubnetGroupOutput), args.Error(1)
}

func (m *mock_ElasticacheClient) DeleteCacheSubnetGroup(ctx context.Context, input *elasticache.DeleteCacheSubnetGroupInput, optFns ...func(*elasticache.Options)) (*elasticache.DeleteCacheSubnetGroupOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*elasticache.DeleteCacheSubnetGroupOutput), args.Error(1)
}

func (m *mock_ElasticacheClient) DescribeCacheSubnetGroups(ctx context.Context, input *elasticache.DescribeCacheSubnetGroupsInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeCacheSubnetGroupsOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*elasticache.DescribeCacheSubnetGroupsOutput), args.Error(1)
}

func (m *mock_ElasticacheClient) DescribeCacheClusters(ctx context.Context, input *elasticache.DescribeCacheClustersInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeCacheClustersOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*elasticache.DescribeCacheClustersOutput), args.Error(1)
}

func (m *mock_ElasticacheClient) DescribeReplicationGroups(ctx context.Context, input *elasticache.DescribeReplicationGroupsInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeReplicationGroupsOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*elasticache.DescribeReplicationGroupsOutput), args.Error(1)
}

func (m *mock_ElasticacheClient) DescribeSnapshots(ctx context.Context, input *elasticache.DescribeSnapshotsInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeSnapshotsOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*elasticache.DescribeSnapshotsOutput), args.Error(1)
}

func (m *mock_ElasticacheClient) CreateSnapshot(ctx context.Context, input *elasticache.CreateSnapshotInput, optFns ...func(*elasticache.Options)) (*elasticache.CreateSnapshotOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*elasticache.CreateSnapshotOutput), args.Error(1)
}

func (m *mock_ElasticacheClient) DeleteSnapshot(ctx context.Context, input *elasticache.DeleteSnapshotInput, optFns ...func(*elasticache.Options)) (*elasticache.DeleteSnapshotOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*elasticache.DeleteSnapshotOutput), args.Error(1)
}

func (m *mock_ElasticacheClient) DescribeUpdateActions(ctx context.Context, input *elasticache.DescribeUpdateActionsInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeUpdateActionsOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*elasticache.DescribeUpdateActionsOutput), args.Error(1)
}

func (m *mock_ElasticacheClient) ModifyReplicationGroup(ctx context.Context, input *elasticache.ModifyReplicationGroupInput, optFns ...func(*elasticache.Options)) (*elasticache.ModifyReplicationGroupOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*elasticache.ModifyReplicationGroupOutput), args.Error(1)
}

func (m *mock_ElasticacheClient) BatchApplyUpdateAction(ctx context.Context, input *elasticache.BatchApplyUpdateActionInput, optFns ...func(*elasticache.Options)) (*elasticache.BatchApplyUpdateActionOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*elasticache.BatchApplyUpdateActionOutput), args.Error(1)
}

func (m *mock_ElasticacheClient) AddTagsToResource(ctx context.Context, input *elasticache.AddTagsToResourceInput, optFns ...func(*elasticache.Options)) (*elasticache.AddTagsToResourceOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*elasticache.AddTagsToResourceOutput), args.Error(1)
}

func (m *mock_ElasticacheClient) CreateReplicationGroup(ctx context.Context, input *elasticache.CreateReplicationGroupInput, optFns ...func(*elasticache.Options)) (*elasticache.CreateReplicationGroupOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*elasticache.CreateReplicationGroupOutput), args.Error(1)
}

// S3 Mock
type mock_S3Client struct {
	mock.Mock
}

func (m *mock_S3Client) PutBucketTagging(ctx context.Context, input *s3.PutBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.PutBucketTaggingOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*s3.PutBucketTaggingOutput), args.Error(1)
}

func (m *mock_S3Client) PutPublicAccessBlock(ctx context.Context, input *s3.PutPublicAccessBlockInput, optFns ...func(*s3.Options)) (*s3.PutPublicAccessBlockOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*s3.PutPublicAccessBlockOutput), args.Error(1)
}

func (m *mock_S3Client) PutBucketEncryption(ctx context.Context, input *s3.PutBucketEncryptionInput, optFns ...func(*s3.Options)) (*s3.PutBucketEncryptionOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*s3.PutBucketEncryptionOutput), args.Error(1)
}

func (m *mock_S3Client) DeleteObjects(ctx context.Context, input *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*s3.DeleteObjectsOutput), args.Error(1)
}

func (m *mock_S3Client) CreateBucket(ctx context.Context, input *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*s3.CreateBucketOutput), args.Error(1)
}

func (m *mock_S3Client) DeleteBucket(ctx context.Context, input *s3.DeleteBucketInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*s3.DeleteBucketOutput), args.Error(1)
}

func (m *mock_S3Client) ListBuckets(ctx context.Context, input *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*s3.ListBucketsOutput), args.Error(1)
}

func (m *mock_S3Client) PutObject(ctx context.Context, input *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

func (m *mock_S3Client) GetObject(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*s3.GetObjectOutput), args.Error(1)
}

func (m *mock_S3Client) DeleteObject(ctx context.Context, input *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*s3.DeleteObjectOutput), args.Error(1)
}

func (m *mock_S3Client) ListObjectsV2(ctx context.Context, input *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*s3.ListObjectsV2Output), args.Error(1)
}

// CloudWatch Mock
type mock_CloudWatchClient struct {
	mock.Mock
}

func (m *mock_CloudWatchClient) GetMetricData(ctx context.Context, input *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*cloudwatch.GetMetricDataOutput), args.Error(1)
}

func (m *mock_CloudWatchClient) PutMetricData(ctx context.Context, input *cloudwatch.PutMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.PutMetricDataOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*cloudwatch.PutMetricDataOutput), args.Error(1)
}

func (m *mock_CloudWatchClient) GetMetricStatistics(ctx context.Context, input *cloudwatch.GetMetricStatisticsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricStatisticsOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*cloudwatch.GetMetricStatisticsOutput), args.Error(1)
}

func (m *mock_CloudWatchClient) ListMetrics(ctx context.Context, input *cloudwatch.ListMetricsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.ListMetricsOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*cloudwatch.ListMetricsOutput), args.Error(1)
}

func (m *mock_CloudWatchClient) DescribeAlarms(ctx context.Context, input *cloudwatch.DescribeAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmsOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*cloudwatch.DescribeAlarmsOutput), args.Error(1)
}

func (m *mock_CloudWatchClient) PutMetricAlarm(ctx context.Context, input *cloudwatch.PutMetricAlarmInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.PutMetricAlarmOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*cloudwatch.PutMetricAlarmOutput), args.Error(1)
}

func (m *mock_CloudWatchClient) DeleteAlarms(ctx context.Context, input *cloudwatch.DeleteAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DeleteAlarmsOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*cloudwatch.DeleteAlarmsOutput), args.Error(1)
}

// STS mock
type mock_STSClient struct {
	mock.Mock
}

func (m *mock_STSClient) GetCallerIdentity(ctx context.Context, input *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*sts.GetCallerIdentityOutput), args.Error(1)
}

func (m *mock_STSClient) AssumeRole(ctx context.Context, input *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*sts.AssumeRoleOutput), args.Error(1)

}

func (m *mock_STSClient) GetFederationToken(ctx context.Context, input *sts.GetFederationTokenInput, optFns ...func(*sts.Options)) (*sts.GetFederationTokenOutput, error) {
	args := m.Called(ctx, input, optFns)
	return args.Get(0).(*sts.GetFederationTokenOutput), args.Error(1)

}
