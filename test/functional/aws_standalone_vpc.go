package functional

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/integr8ly/cloud-resource-operator/pkg/resources"

	"github.com/aws/aws-sdk-go/service/elasticache"

	v1 "k8s.io/api/core/v1"

	configv1 "github.com/openshift/api/config/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aws/aws-sdk-go/service/rds"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/integr8ly/integreatly-operator/test/common"
)

var (
	resourceType    = "_network"
	tier            = "development"
	strategyMapName = "cloud-resources-aws-strategies"

	vpcTagName           = "RHMI Cloud Resource VPC"
	subnetTagName        = "RHMI Cloud Resource Subnet"
	securityGroupTagName = "RHMI Cloud Resource Security Group"
	peeringTagName       = "RHMI Cloud Resource Peering Connection"
	routeTableTagName    = "RHMI Cloud Resource Route Table"
)

type strategyMap struct {
	CreateStrategy json.RawMessage `json:"createStrategy"`
}

// a custom error for reporting errors for each
// network component
type networkConfigTestError struct {
	vpcError                  []string
	subnetsError              []string
	securityGroupError        []string
	peeringConnError          []string
	standaloneRouteTableError []string
	clusterRouteTablesError   []string
	rdsSubnetGroupsError      []string
	cacheSubnetGroupsError    []string
}

// pretty print the error message if there are any errors
// this looks ugly but the output looks good!
func (e *networkConfigTestError) Error() string {
	var str strings.Builder
	if len(e.vpcError) != 0 {
		str.WriteString("\nVPC errors:")
		for _, item := range e.vpcError {
			str.WriteString(fmt.Sprintf("\n\t%s", item))
		}
	}
	if len(e.subnetsError) != 0 {
		str.WriteString("\nSubnet errors:")
		for _, item := range e.subnetsError {
			str.WriteString(fmt.Sprintf("\n\t%s", item))
		}
	}
	if len(e.securityGroupError) != 0 {
		str.WriteString("\nSecurity Group errors:")
		for _, item := range e.securityGroupError {
			str.WriteString(fmt.Sprintf("\n\t%s", item))
		}
	}
	if len(e.peeringConnError) != 0 {
		str.WriteString("\nPeering Connection errors:")
		for _, item := range e.peeringConnError {
			str.WriteString(fmt.Sprintf("\n\t%s", item))
		}
	}
	if len(e.standaloneRouteTableError) != 0 {
		str.WriteString("\nStandalone Route Table errors:")
		for _, item := range e.standaloneRouteTableError {
			str.WriteString(fmt.Sprintf("\n\t%s", item))
		}
	}
	if len(e.clusterRouteTablesError) != 0 {
		str.WriteString("\nCluster Route Table errors:")
		for _, item := range e.clusterRouteTablesError {
			str.WriteString(fmt.Sprintf("\n\t%s", item))
		}
	}
	if len(e.rdsSubnetGroupsError) != 0 {
		str.WriteString("\nRDS Subnet Groups errors:")
		for _, item := range e.rdsSubnetGroupsError {
			str.WriteString(fmt.Sprintf("\n\t%s", item))
		}
	}
	if len(e.cacheSubnetGroupsError) != 0 {
		str.WriteString("\nElasticache Subnet Groups errors:")
		for _, item := range e.cacheSubnetGroupsError {
			str.WriteString(fmt.Sprintf("\n\t%s", item))
		}
	}
	return str.String()
}

// the error is valid if any of the error slices are not empty
func (e *networkConfigTestError) isValid() bool {
	return len(e.vpcError) != 0 ||
		len(e.subnetsError) != 0 ||
		len(e.securityGroupError) != 0 ||
		len(e.peeringConnError) != 0 ||
		len(e.standaloneRouteTableError) != 0 ||
		len(e.clusterRouteTablesError) != 0 ||
		len(e.rdsSubnetGroupsError) != 0 ||
		len(e.cacheSubnetGroupsError) != 0
}

// TestStandaloneVPCExists tests that the cloud resource operator network components
// have been correctly set up and configured
func TestStandaloneVPCExists(t *testing.T, testingCtx *common.TestingContext) {
	ctx := context.TODO()
	testErrors := &networkConfigTestError{}

	// create a new session
	session, err := CreateAWSSession(ctx, testingCtx.Client)
	if err != nil {
		t.Fatal("could not create aws session", err)
	}
	ec2Sess := ec2.New(session)

	// get the aws strategy map to get the vpc cidr block
	// from the _network key
	strategyMap := &v1.ConfigMap{}
	err = testingCtx.Client.Get(ctx, types.NamespacedName{
		Namespace: awsCredsNamespace,
		Name:      strategyMapName,
	}, strategyMap)
	if err != nil {
		t.Fatal("could not get aws strategy map", err)
	}

	// get the vpc cidr block
	expectedCidr, err := getCidrBlock(ctx, strategyMap)
	if err != nil {
		t.Fatal("could not get cidr block", err)
	}

	// get the cluster id used for tagging aws resources
	clusterTag, err := getClusterID(ctx, testingCtx.Client)
	if err != nil {
		t.Fatal("could not get cluster id", err)
	}

	// verify vpc
	vpc, err := verifyVpc(ec2Sess, clusterTag, expectedCidr)
	testErrors.vpcError = err.(*networkConfigTestError).vpcError

	// verify subnets
	err = verifySubnets(ec2Sess, clusterTag, expectedCidr)
	testErrors.subnetsError = err.(*networkConfigTestError).subnetsError

	// verify security groups
	err = verifySecurityGroup(ec2Sess, clusterTag)
	testErrors.securityGroupError = err.(*networkConfigTestError).securityGroupError

	// we have to manually construct the subnet group names for rds and elasticache,
	// since tag filtering isnt currently available
	name := resources.ShortenString(fmt.Sprintf("%s-%s", clusterTag, "subnet-group"), 40)

	// verify rds subnet groups
	rdsSvc := rds.New(session)
	err = verifyRdsSubnetGroups(rdsSvc, name)
	testErrors.rdsSubnetGroupsError = err.(*networkConfigTestError).rdsSubnetGroupsError

	// verify elasticache subnet groups
	cacheSvc := elasticache.New(session)
	err = verifyCacheSubnetGroups(cacheSvc, name)
	testErrors.rdsSubnetGroupsError = err.(*networkConfigTestError).rdsSubnetGroupsError

	// peering connection and route tables
	if vpc == nil {
		testErrors.peeringConnError = []string{"skipping peering connection test, vpc cannot be nil"}
		testErrors.standaloneRouteTableError = []string{"skipping standalone route table test, vpc cannot be nil"}
		testErrors.clusterRouteTablesError = []string{"skipping cluster route table test, vpc cannot be nil"}
	} else {
		conn, err := verifyPeeringConnection(ec2Sess, clusterTag, expectedCidr, aws.StringValue(vpc.VpcId))
		testErrors.peeringConnError = err.(*networkConfigTestError).peeringConnError

		err = verifyStandaloneRouteTable(ec2Sess, clusterTag, conn)
		testErrors.standaloneRouteTableError = err.(*networkConfigTestError).peeringConnError

		err = verifyClusterRouteTables(ec2Sess, clusterTag, expectedCidr, conn)
		testErrors.clusterRouteTablesError = err.(*networkConfigTestError).clusterRouteTablesError
	}

	if testErrors.isValid() {
		t.Fatal(testErrors.Error())
	}
}

// verify that the standalone vpc is created
func verifyVpc(session *ec2.EC2, clusterTag, expectedCidr string) (*ec2.Vpc, error) {
	newErr := &networkConfigTestError{
		vpcError: []string{},
	}

	// filter vpcs by name and integreatly tag
	describeVpcs, err := session.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []*string{aws.String(vpcTagName)},
			},
			{
				Name:   aws.String("tag:integreatly.org/clusterID"),
				Values: []*string{aws.String(clusterTag)},
			},
		},
	})
	if err != nil {
		newErr.vpcError = append(newErr.vpcError, fmt.Sprintf("could not find vpc: %v", err))
		return nil, newErr
	}

	// only one vpc is expected
	vpcs := describeVpcs.Vpcs
	if len(vpcs) != 1 {
		newErr.vpcError = append(newErr.vpcError, fmt.Sprintf("expected 1 vpc but found %d", len(vpcs)))
		return nil, newErr
	}

	// cidr blocks should match
	vpc := vpcs[0]
	foundCidr := aws.StringValue(vpc.CidrBlock)
	if foundCidr != expectedCidr {
		newErr.vpcError = append(newErr.vpcError, fmt.Sprintf("cidr blocks not equal, expected %s but got %s", expectedCidr, foundCidr))
	}

	return vpc, newErr
}

// verify that the vpc subnets are created
func verifySubnets(session *ec2.EC2, clusterTag, expectedCidr string) error {
	newErr := &networkConfigTestError{
		subnetsError: []string{},
	}

	// filter subnets by name and integreatly tag
	describeSubnets, err := session.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []*string{aws.String(subnetTagName)},
			},
			{
				Name:   aws.String("tag:integreatly.org/clusterID"),
				Values: []*string{aws.String(clusterTag)},
			},
		},
	})
	if err != nil {
		newErr.subnetsError = append(newErr.subnetsError, fmt.Sprintf("could not describe subnets: %v", err))
		return newErr
	}

	// parse the vpc cidr block from the createStrategy for _network
	_, cidr, err := net.ParseCIDR(expectedCidr)
	if err != nil {
		newErr.subnetsError = append(newErr.subnetsError, fmt.Sprintf("could not parse vpc cidr block: %v", err))
		return newErr
	}
	cidrMask, _ := cidr.Mask.Size()

	// verify the subnet masks for the subnets are one bit bigger
	// than the vpc subnet mask
	subnets := describeSubnets.Subnets
	for _, subnet := range subnets {
		_, subnetCidr, err := net.ParseCIDR(aws.StringValue(subnet.CidrBlock))
		if err != nil {
			newErr.subnetsError = append(newErr.subnetsError, fmt.Sprintf("could not parse subnet mask for vpc subnets: %v", err))
			return newErr
		}
		subnetCidrMask, _ := subnetCidr.Mask.Size()
		if subnetCidrMask != cidrMask+1 {
			newErr.subnetsError = append(newErr.subnetsError, fmt.Sprintf("unexpected subnet mask size for vpc subnets: %v", err))
		}
	}

	return newErr
}

// verify vpc security group
func verifySecurityGroup(session *ec2.EC2, clusterTag string) error {
	newErr := &networkConfigTestError{
		securityGroupError: []string{},
	}

	// filter security groups by name and integreatly tag
	describeGroups, err := session.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []*string{aws.String(securityGroupTagName)},
			},
			{
				Name:   aws.String("tag:integreatly.org/clusterID"),
				Values: []*string{aws.String(clusterTag)},
			},
		},
	})
	if err != nil {
		newErr.securityGroupError = append(newErr.securityGroupError, fmt.Sprintf("could not find vpc security group: %v", err))
	}

	// expect 1 security group
	secGroups := describeGroups.SecurityGroups
	if len(secGroups) != 1 {
		newErr.securityGroupError = append(newErr.securityGroupError, fmt.Sprintf("unexpected number of security groups: %d", len(secGroups)))
		return newErr
	}

	return newErr
}

// verify that the subnet groups for rds are created
func verifyRdsSubnetGroups(rdsSess *rds.RDS, name string) error {
	newErr := &networkConfigTestError{
		rdsSubnetGroupsError: []string{},
	}

	// get rds subnet groups by subnet group name
	describeGroups, err := rdsSess.DescribeDBSubnetGroups(&rds.DescribeDBSubnetGroupsInput{
		DBSubnetGroupName: aws.String(name),
	})
	if err != nil {
		newErr.rdsSubnetGroupsError = append(newErr.rdsSubnetGroupsError, fmt.Sprintf("error describing rds subnet groups: %v", err))
		return newErr
	}

	// expect 1 subnet group
	subnetGroups := describeGroups.DBSubnetGroups
	if len(subnetGroups) != 1 {
		newErr.rdsSubnetGroupsError = append(newErr.rdsSubnetGroupsError, fmt.Sprintf("unexpected number of rds subnet groups: %d", len(subnetGroups)))
		return newErr
	}

	return newErr
}

// verify that the subnet groups for elasticache are created
func verifyCacheSubnetGroups(cacheSvc *elasticache.ElastiCache, name string) error {
	newErr := &networkConfigTestError{
		cacheSubnetGroupsError: []string{},
	}

	// get elasticache subnet groups by subnet group name
	describeCacheGroups, err := cacheSvc.DescribeCacheSubnetGroups(&elasticache.DescribeCacheSubnetGroupsInput{
		CacheSubnetGroupName: aws.String(name),
	})
	if err != nil {
		newErr.cacheSubnetGroupsError = append(newErr.cacheSubnetGroupsError, fmt.Sprintf("error describing elasticache subnet groups: %v", err))
		return newErr
	}

	// expect 1 subnet group
	cacheSubnetGroups := describeCacheGroups.CacheSubnetGroups
	if len(cacheSubnetGroups) != 1 {
		newErr.cacheSubnetGroupsError = append(newErr.cacheSubnetGroupsError, fmt.Sprintf("unexpected number of elasticache subnet groups: %d", len(cacheSubnetGroups)))
	}

	return newErr
}

// verify that the peering connection we create has the correct requester info
func verifyPeeringConnection(session *ec2.EC2, clusterTag, expectedCidr, vpcID string) (*ec2.VpcPeeringConnection, error) {
	newErr := &networkConfigTestError{
		peeringConnError: []string{},
	}

	// filter the peering connections by name and integreatly tag
	peeringConn, err := session.DescribeVpcPeeringConnections(&ec2.DescribeVpcPeeringConnectionsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []*string{aws.String(peeringTagName)},
			},
			{
				Name:   aws.String("tag:integreatly.org/clusterID"),
				Values: []*string{aws.String(clusterTag)},
			},
		},
	})
	if err != nil {
		newErr.peeringConnError = append(newErr.peeringConnError, fmt.Sprintf("could not describe peering connections: %v", err))
		return nil, newErr
	}

	// expect 1 peering connection to be found
	conns := peeringConn.VpcPeeringConnections
	if len(conns) != 1 {
		newErr.peeringConnError = append(newErr.peeringConnError, fmt.Sprintf("unexpected number of vpc peering connections: %d", len(conns)))
		return nil, newErr
	}

	// verify that the requester info is correct
	conn := conns[0]
	if aws.StringValue(conn.RequesterVpcInfo.CidrBlock) != expectedCidr && aws.StringValue(conn.RequesterVpcInfo.VpcId) != vpcID {
		newErr.peeringConnError = append(newErr.peeringConnError, fmt.Sprintf("unexpected accepter vpc cidr block: %d", len(conns)))
	}

	return conn, newErr
}

// verify that the standalone route table contains a route to the peering connection
func verifyStandaloneRouteTable(session *ec2.EC2, clusterTag string, conn *ec2.VpcPeeringConnection) error {
	newErr := &networkConfigTestError{
		standaloneRouteTableError: []string{},
	}

	// filter the route tables by name and integreatly tag
	describeRouteTables, err := session.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []*string{aws.String(routeTableTagName)},
			},
			{
				Name:   aws.String("tag:integreatly.org/clusterID"),
				Values: []*string{aws.String(clusterTag)},
			},
		},
	})
	if err != nil {
		newErr.standaloneRouteTableError = append(newErr.standaloneRouteTableError, fmt.Sprintf("could not describe route tab;es: %v", err))
		return newErr
	}

	// expect 1 route table
	routeTables := describeRouteTables.RouteTables
	if len(routeTables) != 1 {
		newErr.standaloneRouteTableError = append(newErr.standaloneRouteTableError, fmt.Sprintf("unexpected number of route tables: %d", len(routeTables)))
		return newErr
	}

	// verify that the route table has a route to the peering connection
	foundRoute := false
	for _, route := range routeTables[0].Routes {
		if aws.StringValue(route.VpcPeeringConnectionId) == aws.StringValue(conn.VpcPeeringConnectionId) {
			foundRoute = true
		}
	}
	if !foundRoute {
		newErr.standaloneRouteTableError = append(newErr.standaloneRouteTableError, "did not find expected route table entries")
	}

	return newErr
}

// verify that the cluster route tables contain a route to the peering connection and the standalone vpc
func verifyClusterRouteTables(session *ec2.EC2, clusterTag, vpcCidr string, peeringConn *ec2.VpcPeeringConnection) error {
	newErr := &networkConfigTestError{
		clusterRouteTablesError: []string{},
	}

	// filter the route tables by kubernetes owner id
	describeRouteTables, err := session.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String(fmt.Sprintf("tag:kubernetes.io/cluster/%s", clusterTag)),
				Values: []*string{aws.String("owned")},
			},
		},
	})
	if err != nil {
		newErr.clusterRouteTablesError = append(newErr.clusterRouteTablesError, fmt.Sprintf("could not describe route tables: %v", err))
		return newErr
	}

	// expect 2 route tables (main and non-main)
	routeTables := describeRouteTables.RouteTables
	if len(routeTables) != 2 {
		newErr.clusterRouteTablesError = append(newErr.clusterRouteTablesError, fmt.Sprintf("unexpected number of route tables: %d", len(routeTables)))
		return newErr
	}

	// verify that each route table has a route to
	// the peering connection
	for _, routeTable := range routeTables {
		foundRoute := false
		for _, route := range routeTable.Routes {
			if aws.StringValue(route.DestinationCidrBlock) == vpcCidr &&
				aws.StringValue(route.VpcPeeringConnectionId) == aws.StringValue(peeringConn.VpcPeeringConnectionId) {
				foundRoute = true
			}
		}
		if !foundRoute {
			tableID := aws.StringValue(routeTable.RouteTableId)
			newErr.clusterRouteTablesError = append(newErr.clusterRouteTablesError, fmt.Sprintf("expected route for cluster route table %s not found", tableID))
		}
	}

	return newErr
}

func getClusterID(ctx context.Context, client client.Client) (string, error) {
	infra := &configv1.Infrastructure{}
	err := client.Get(ctx, types.NamespacedName{Name: "cluster"}, infra)
	if err != nil {
		return "", fmt.Errorf("failed to get aws region : %w", err)
	}
	return infra.Status.InfrastructureName, nil
}

func getCidrBlock(ctx context.Context, strategyMap *v1.ConfigMap) (string, error) {
	strat, err := getStrategyForResource(ctx, strategyMap, resourceType, tier)
	if err != nil {
		return "", err
	}
	vpcCreateConfig := &ec2.CreateVpcInput{}
	if err := json.Unmarshal(strat.CreateStrategy, vpcCreateConfig); err != nil {
		return "", err
	}
	if vpcCreateConfig.CidrBlock == nil {
		return "", fmt.Errorf("cidr block cannot be empty")
	}
	return aws.StringValue(vpcCreateConfig.CidrBlock), nil
}
