package functional

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	croAWS "github.com/integr8ly/cloud-resource-operator/pkg/providers/aws"

	"github.com/integr8ly/cloud-resource-operator/pkg/resources"

	"github.com/aws/aws-sdk-go/service/elasticache"

	v1 "k8s.io/api/core/v1"

	"github.com/aws/aws-sdk-go/service/rds"
	"k8s.io/apimachinery/pkg/types"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/integr8ly/integreatly-operator/test/common"
)

var (
	tier                 = "production"
	awsAllowedCidrRanges = []string{
		"10.255.255.255/8",
		"172.31.255.255/12",
	}
)

const (
	vpcClusterTagKey         = "tag:integreatly.org/clusterID"
	clusterOwnedTagKeyPrefix = "tag:kubernetes.io/cluster/"
	clusterOwnedTagValue     = "owned"
)

// a custom error for reporting errors for each
// network component
type networkConfigTestError struct {
	vpcError                  []error
	subnetsError              []error
	securityGroupError        []error
	peeringConnError          []error
	standaloneRouteTableError []error
	clusterRouteTablesError   []error
	rdsSubnetGroupsError      []error
	cacheSubnetGroupsError    []error
}

// pretty print the error message if there are any errors
// this looks ugly but the output looks good!
func (e *networkConfigTestError) Error() string {
	var str strings.Builder
	if len(e.vpcError) != 0 {
		str.WriteString("\nVPC errors:")
		for _, item := range e.vpcError {
			str.WriteString(fmt.Sprintf("\n\t%s", item.Error()))
		}
	}
	if len(e.subnetsError) != 0 {
		str.WriteString("\nSubnet errors:")
		for _, item := range e.subnetsError {
			str.WriteString(fmt.Sprintf("\n\t%s", item.Error()))
		}
	}
	if len(e.securityGroupError) != 0 {
		str.WriteString("\nSecurity Group errors:")
		for _, item := range e.securityGroupError {
			str.WriteString(fmt.Sprintf("\n\t%s", item.Error()))
		}
	}
	if len(e.peeringConnError) != 0 {
		str.WriteString("\nPeering Connection errors:")
		for _, item := range e.peeringConnError {
			str.WriteString(fmt.Sprintf("\n\t%s", item.Error()))
		}
	}
	if len(e.standaloneRouteTableError) != 0 {
		str.WriteString("\nStandalone Route Table errors:")
		for _, item := range e.standaloneRouteTableError {
			str.WriteString(fmt.Sprintf("\n\t%s", item.Error()))
		}
	}
	if len(e.clusterRouteTablesError) != 0 {
		str.WriteString("\nCluster Route Table errors:")
		for _, item := range e.clusterRouteTablesError {
			str.WriteString(fmt.Sprintf("\n\t%s", item.Error()))
		}
	}
	if len(e.rdsSubnetGroupsError) != 0 {
		str.WriteString("\nRDS Subnet Groups errors:")
		for _, item := range e.rdsSubnetGroupsError {
			str.WriteString(fmt.Sprintf("\n\t%s", item.Error()))
		}
	}
	if len(e.cacheSubnetGroupsError) != 0 {
		str.WriteString("\nElasticache Subnet Groups errors:")
		for _, item := range e.cacheSubnetGroupsError {
			str.WriteString(fmt.Sprintf("\n\t%s", item.Error()))
		}
	}
	return str.String()
}

// the error is valid if any of the error slices are not empty
func (e *networkConfigTestError) hasError() bool {
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
func TestStandaloneVPCExists(t common.TestingTB, testingCtx *common.TestingContext) {
	ctx := context.TODO()
	testErrors := &networkConfigTestError{}

	// create a new session
	session, isSTS, err := CreateAWSSession(ctx, testingCtx.Client)
	if err != nil {
		t.Fatal("could not create aws session", err)
	}
	ec2Sess := ec2.New(session)

	// get the aws strategy map to get the vpc cidr block
	// from the _network key
	strategyMap := &v1.ConfigMap{}
	err = testingCtx.Client.Get(ctx, types.NamespacedName{
		Namespace: common.RHOAMOperatorNamespace,
		Name:      croAWS.DefaultConfigMapName,
	}, strategyMap)
	if err != nil {
		t.Fatal("could not get aws strategy map", err)
	}

	// get the create strategy for _network in the aws strategy configmap
	// if this doesn't exist, skip the test completely since we're dealing
	// with legacy cro networking
	strat, err := getStrategyForResource(strategyMap, networkResourceType, tier)
	if err != nil {
		t.Skip("_network key does not exist in aws strategy configmap, skipping standalone vpc network test")
	}

	// get the cluster id used for tagging aws resources
	clusterTag, err := getClusterID(ctx, testingCtx.Client)
	if err != nil {
		t.Fatal("could not get cluster id", err)
	}

	// get the vpc cidr block
	expectedCidr, err := getCidrBlockFromStrategyMap(strat)
	if err != nil {
		t.Fatal("could not get cidr block from strategy map", err)
	}

	// if the cidr strategy map is empty then attempt to retrieve the standaloneCidr cidr block from the vpc
	if expectedCidr == "" {
		standaloneCidr, err := getVpcCidrBlock(ec2Sess, vpcClusterTagKey, clusterTag)
		if err != nil {
			t.Fatal("could not get cidr block from vpc", err)
		}

		// check if the cidr block is in the allowed range
		err = verifyCidrBlockIsInAllowedRange(standaloneCidr, awsAllowedCidrRanges)
		if err != nil {
			t.Fatalf("cidr block %s is not within the allowed range %s", standaloneCidr, err)
		}

		// build tag key to retrieve cluster vpc
		// denoted -> kubernetes.io/cluster/<cluster-id>=owned
		clusterOwnedVpcKey := fmt.Sprintf("%s%s", clusterOwnedTagKeyPrefix, clusterTag)

		// retrieve the cluster cidr
		clusterCidr, err := getVpcCidrBlock(ec2Sess, clusterOwnedVpcKey, clusterOwnedTagValue)
		if err != nil {
			t.Fatal("could not get cidr block from vpc", err)
		}

		// check if the cidr blocks overlap
		err = checkForOverlappingCidrBlocks(standaloneCidr, clusterCidr)
		if err != nil {
			t.Fatal(err)
		}

		expectedCidr = standaloneCidr
	}

	clusterNodes := &v1.NodeList{}
	err = testingCtx.Client.List(ctx, clusterNodes)
	if err != nil {
		t.Errorf("Error when getting the list of OpenShift cluster nodes: %s", err)
	}
	availableZones := GetClustersAvailableZones(clusterNodes)

	// verify vpc
	vpc, err := verifyVpc(ec2Sess, clusterTag, expectedCidr, isSTS)
	testErrors.vpcError = err.(*networkConfigTestError).vpcError

	// fail immediately if the vpc is nil, since all of the
	// other network components are associated with it
	if vpc == nil {
		t.Fatal(testErrors.Error())
	}

	// verify subnets
	subnets, err := verifySubnets(ec2Sess, clusterTag, expectedCidr)
	testErrors.subnetsError = err.(*networkConfigTestError).subnetsError

	// verify security groups
	err = verifySecurityGroup(ec2Sess, clusterTag)
	testErrors.securityGroupError = err.(*networkConfigTestError).securityGroupError

	// we have to manually construct the subnet group names for rds and elasticache,
	// since tag filtering isn't currently available
	subnetGroupName := resources.ShortenString(fmt.Sprintf("%s-%s", clusterTag, "subnet-group"), 40)

	// build array list of all vpc private subnets
	var subnetIDs []*string
	for _, subnet := range subnets {
		subnetIDs = append(subnetIDs, subnet.SubnetId)
	}

	// verify rds subnet groups
	rdsSvc := rds.New(session)
	err = verifyRdsSubnetGroups(rdsSvc, subnetGroupName, subnetIDs)
	testErrors.rdsSubnetGroupsError = err.(*networkConfigTestError).rdsSubnetGroupsError

	// verify elasticache subnet groups
	cacheSvc := elasticache.New(session)
	err = verifyCacheSubnetGroups(cacheSvc, subnetGroupName, subnetIDs)
	testErrors.cacheSubnetGroupsError = err.(*networkConfigTestError).cacheSubnetGroupsError

	// verify peering connection
	conn, err := verifyPeeringConnection(ec2Sess, clusterTag, expectedCidr, aws.StringValue(vpc.VpcId))
	testErrors.peeringConnError = err.(*networkConfigTestError).peeringConnError

	// verify standalone vpc route table
	err = verifyStandaloneRouteTable(ec2Sess, clusterTag, conn)
	testErrors.standaloneRouteTableError = err.(*networkConfigTestError).standaloneRouteTableError

	// verify cluster route table
	err = verifyClusterRouteTables(ec2Sess, clusterTag, expectedCidr, conn, availableZones)
	testErrors.clusterRouteTablesError = err.(*networkConfigTestError).clusterRouteTablesError

	// if any error was found, fail the test
	if testErrors.hasError() {
		t.Fatal(testErrors.Error())
	}
}

// verify that the standalone vpc is created
func verifyVpc(session *ec2.EC2, clusterTag, expectedCidr string, isSTS bool) (*ec2.Vpc, error) {
	newErr := &networkConfigTestError{
		vpcError: []error{},
	}

	// filter vpcs by integreatly cluster id tag
	describeVpcs, err := session.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String(vpcClusterTagKey),
				Values: []*string{aws.String(clusterTag)},
			},
		},
	})
	if err != nil {
		newErr.vpcError = append(newErr.vpcError, fmt.Errorf("could not find vpc: %v", err))
		return nil, newErr
	}

	// only one vpc is expected
	vpcs := describeVpcs.Vpcs
	if len(vpcs) != 1 {
		newErr.vpcError = append(newErr.vpcError, fmt.Errorf("expected 1 vpc but found %d", len(vpcs)))
		return nil, newErr
	}

	// cidr blocks should match
	vpc := vpcs[0]
	foundCidr := aws.StringValue(vpc.CidrBlock)
	if foundCidr != expectedCidr {
		errMsg := fmt.Errorf("expected vpc cidr block to match _network cidr block in aws strategy configmap. Expected %s, but got %s", expectedCidr, foundCidr)
		newErr.vpcError = append(newErr.vpcError, errMsg)
	}

	if !ec2TagsContains(vpc.Tags, awsManagedTagKey, awsManagedTagValue) {
		newErr.vpcError = append(newErr.vpcError, fmt.Errorf("vpc does not have expected %s tag", awsManagedTagKey))
	}
	if isSTS && !ec2TagsContains(vpc.Tags, awsClusterTypeKey, awsClusterTypeRosaValue) {
		newErr.vpcError = append(newErr.vpcError, fmt.Errorf("vpc does not have expected %s tag", awsClusterTypeKey))
	}

	return vpc, newErr
}

// verify that the vpc subnets are created
func verifySubnets(session *ec2.EC2, clusterTag, expectedCidr string) ([]*ec2.Subnet, error) {
	newErr := &networkConfigTestError{
		subnetsError: []error{},
	}

	// filter subnets by integreatly cluster id tag
	describeSubnets, err := session.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String(vpcClusterTagKey),
				Values: []*string{aws.String(clusterTag)},
			},
		},
	})
	if err != nil {
		errMsg := fmt.Errorf("could not describe subnets: %v", err)
		newErr.subnetsError = append(newErr.subnetsError, errMsg)
		return nil, newErr
	}

	// parse the vpc cidr block from the createStrategy for _network
	subnets := describeSubnets.Subnets
	_, cidr, err := net.ParseCIDR(expectedCidr)
	if err != nil {
		errMsg := fmt.Errorf("could not parse vpc cidr block: %v", err)
		newErr.subnetsError = append(newErr.subnetsError, errMsg)
		return subnets, newErr
	}
	cidrMask, _ := cidr.Mask.Size()

	// verify the subnet masks for the subnets are one bit bigger
	// than the vpc subnet mask
	for _, subnet := range subnets {
		_, subnetCidr, err := net.ParseCIDR(aws.StringValue(subnet.CidrBlock))
		if err != nil {
			errMsg := fmt.Errorf("could not parse subnet mask for vpc subnets: %v", err)
			newErr.subnetsError = append(newErr.subnetsError, errMsg)
			return subnets, newErr
		}
		subnetCidrMask, _ := subnetCidr.Mask.Size()
		if subnetCidrMask != cidrMask+1 {
			errMsg := fmt.Errorf("subnet mask expect to be 1 bit greater than vpc subnet mask, found: %d, expected %d", subnetCidrMask, cidrMask+1)
			newErr.subnetsError = append(newErr.subnetsError, errMsg)
		}
	}

	return subnets, newErr
}

// verify vpc security group
func verifySecurityGroup(session *ec2.EC2, clusterTag string) error {
	newErr := &networkConfigTestError{
		securityGroupError: []error{},
	}

	// filter security groups by integreatly cluster id tag
	describeGroups, err := session.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String(vpcClusterTagKey),
				Values: []*string{aws.String(clusterTag)},
			},
		},
	})
	if err != nil {
		errMsg := fmt.Errorf("could not find vpc security group: %v", err)
		newErr.securityGroupError = append(newErr.securityGroupError, errMsg)
	}

	// expect 1 security group
	secGroups := describeGroups.SecurityGroups
	if len(secGroups) != 1 {
		errMsg := fmt.Errorf("unexpected number of security groups: %d", len(secGroups))
		newErr.securityGroupError = append(newErr.securityGroupError, errMsg)
		return newErr
	}

	return newErr
}

// verify that the subnet groups for rds are created
func verifyRdsSubnetGroups(rdsSess *rds.RDS, name string, subnets []*string) error {
	newErr := &networkConfigTestError{
		rdsSubnetGroupsError: []error{},
	}

	// get rds subnet groups by subnet group name
	describeGroups, err := rdsSess.DescribeDBSubnetGroups(&rds.DescribeDBSubnetGroupsInput{
		DBSubnetGroupName: aws.String(name),
	})
	if err != nil {
		errMsg := fmt.Errorf("error describing rds subnet groups: %v", err)
		newErr.rdsSubnetGroupsError = append(newErr.rdsSubnetGroupsError, errMsg)
		return newErr
	}

	// expect 1 subnet group
	subnetGroups := describeGroups.DBSubnetGroups
	if len(subnetGroups) != 1 {
		errMsg := fmt.Errorf("unexpected number of rds subnet groups: %d", len(subnetGroups))
		newErr.rdsSubnetGroupsError = append(newErr.rdsSubnetGroupsError, errMsg)
		return newErr
	}

	// ensure all subnets exist in subnet group
	subnetsExist := true
	for _, subnet := range subnetGroups[0].Subnets {
		if !contains(subnets, subnet.SubnetIdentifier) {
			subnetsExist = false
			break
		}
	}
	if !subnetsExist {
		errMsg := fmt.Errorf("rds subnet group does not contain expected subnets: %s, %s", aws.StringValue(subnets[0]), aws.StringValue(subnets[1]))
		newErr.rdsSubnetGroupsError = append(newErr.rdsSubnetGroupsError, errMsg)
	}

	return newErr
}

// verify that the subnet groups for elasticache are created
func verifyCacheSubnetGroups(cacheSvc *elasticache.ElastiCache, name string, subnets []*string) error {
	newErr := &networkConfigTestError{
		cacheSubnetGroupsError: []error{},
	}

	// get elasticache subnet groups by subnet group name
	describeCacheGroups, err := cacheSvc.DescribeCacheSubnetGroups(&elasticache.DescribeCacheSubnetGroupsInput{
		CacheSubnetGroupName: aws.String(name),
	})
	if err != nil {
		errMsg := fmt.Errorf("error describing elasticache subnet groups: %v", err)
		newErr.cacheSubnetGroupsError = append(newErr.cacheSubnetGroupsError, errMsg)
		return newErr
	}

	// expect 1 subnet group
	cacheSubnetGroups := describeCacheGroups.CacheSubnetGroups
	if len(cacheSubnetGroups) != 1 {
		errMsg := fmt.Errorf("unexpected number of elasticache subnet groups: %d", len(cacheSubnetGroups))
		newErr.cacheSubnetGroupsError = append(newErr.cacheSubnetGroupsError, errMsg)
	}

	// ensure all subnets exist in subnet group
	subnetsExist := true
	for _, subnet := range cacheSubnetGroups[0].Subnets {
		if !contains(subnets, subnet.SubnetIdentifier) {
			subnetsExist = false
			break
		}
	}
	if !subnetsExist {
		errMsg := fmt.Errorf("elasticache subnet group does not contain expected subnets: %s, %s", aws.StringValue(subnets[0]), aws.StringValue(subnets[1]))
		newErr.cacheSubnetGroupsError = append(newErr.cacheSubnetGroupsError, errMsg)
	}

	return newErr
}

// verify that the peering connection we create has the correct requester info
func verifyPeeringConnection(session *ec2.EC2, clusterTag, expectedCidr, vpcID string) (*ec2.VpcPeeringConnection, error) {
	newErr := &networkConfigTestError{
		peeringConnError: []error{},
	}

	// filter the peering connections by integreatly cluster id tag
	peeringConn, err := session.DescribeVpcPeeringConnections(&ec2.DescribeVpcPeeringConnectionsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String(vpcClusterTagKey),
				Values: []*string{aws.String(clusterTag)},
			},
		},
	})
	if err != nil {
		errMsg := fmt.Errorf("could not describe peering connections: %v", err)
		newErr.peeringConnError = append(newErr.peeringConnError, errMsg)
		return nil, newErr
	}

	// expect 1 peering connection to be found
	conns := peeringConn.VpcPeeringConnections
	if len(conns) != 1 {
		errMsg := fmt.Errorf("unexpected number of vpc peering connections: %d", len(conns))
		newErr.peeringConnError = append(newErr.peeringConnError, errMsg)
		return nil, newErr
	}

	// verify that the requester info is correct
	conn := conns[0]
	if aws.StringValue(conn.RequesterVpcInfo.CidrBlock) != expectedCidr && aws.StringValue(conn.RequesterVpcInfo.VpcId) != vpcID {
		errMsg := fmt.Errorf("unexpected accepter vpc cidr block: %d", len(conns))
		newErr.peeringConnError = append(newErr.peeringConnError, errMsg)
	}

	// verify the peering connection state is active
	if aws.StringValue(conn.Status.Code) != ec2.VpcPeeringConnectionStateReasonCodeActive {
		errMsg := fmt.Errorf("unexpected peering connection status: %s", aws.StringValue(conn.Status.Code))
		newErr.peeringConnError = append(newErr.peeringConnError, errMsg)
	}

	return conn, newErr
}

// verify that the standalone route table contains a route to the peering connection
func verifyStandaloneRouteTable(session *ec2.EC2, clusterTag string, conn *ec2.VpcPeeringConnection) error {
	newErr := &networkConfigTestError{
		standaloneRouteTableError: []error{},
	}

	// filter the route tables by integreatly cluster id tag
	describeRouteTables, err := session.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String(vpcClusterTagKey),
				Values: []*string{aws.String(clusterTag)},
			},
		},
	})
	if err != nil {
		errMsg := fmt.Errorf("could not describe route tables: %v", err)
		newErr.standaloneRouteTableError = append(newErr.standaloneRouteTableError, errMsg)
		return newErr
	}

	// expect 1 route table
	routeTables := describeRouteTables.RouteTables
	if len(routeTables) != 1 {
		errMsg := fmt.Errorf("unexpected number of route tables: %d", len(routeTables))
		newErr.standaloneRouteTableError = append(newErr.standaloneRouteTableError, errMsg)
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
		errMsg := fmt.Errorf("did not find expected route with peering connection: %s", aws.StringValue(conn.VpcPeeringConnectionId))
		newErr.standaloneRouteTableError = append(newErr.standaloneRouteTableError, errMsg)
	}

	return newErr
}

// verify that the cluster route tables contain a route to the peering connection and the standalone vpc
func verifyClusterRouteTables(session *ec2.EC2, clusterTag, vpcCidr string, peeringConn *ec2.VpcPeeringConnection, availableZones map[string]bool) error {
	newErr := &networkConfigTestError{
		clusterRouteTablesError: []error{},
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
		errMsg := fmt.Errorf("could not describe route tables: %v", err)
		newErr.clusterRouteTablesError = append(newErr.clusterRouteTablesError, errMsg)
		return newErr
	}

	// 1 private route per AZ + 1 public route
	expectedRouteTableCount := len(availableZones) + 1
	routeTables := describeRouteTables.RouteTables
	if len(routeTables) != expectedRouteTableCount {
		errMsg := fmt.Errorf("unexpected number of route tables: %d", len(routeTables))
		newErr.clusterRouteTablesError = append(newErr.clusterRouteTablesError, errMsg)
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
			errMsg := fmt.Errorf("expected route for cluster route table %s not found", tableID)
			newErr.clusterRouteTablesError = append(newErr.clusterRouteTablesError, errMsg)
		}
	}

	return newErr
}

func getCidrBlockFromStrategyMap(strat *strategyMap) (string, error) {
	vpcCreateConfig := &ec2.CreateVpcInput{}
	if err := json.Unmarshal(strat.CreateStrategy, vpcCreateConfig); err != nil {
		return "", err
	}
	if vpcCreateConfig.CidrBlock == nil {
		return "", fmt.Errorf("cidr block cannot be nil")
	}
	return aws.StringValue(vpcCreateConfig.CidrBlock), nil
}

func contains(strs []*string, str *string) bool {
	for _, s := range strs {
		if aws.StringValue(str) == aws.StringValue(s) {
			return true
		}
	}
	return false
}
