package functional

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/test/common"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var internalSubnetTag = "kubernetes.io/role/internal-elb"

func TestLegacyClusterVPC(t common.TestingTB, testingCtx *common.TestingContext) {
	ctx := context.TODO()
	testErrors := &networkConfigTestError{}

	// get the aws strategy map to get the vpc cidr block
	// from the _network key
	strategyMap := &v1.ConfigMap{}
	err := testingCtx.Client.Get(ctx, types.NamespacedName{
		Namespace: common.RHMIOperatorNamespace,
		Name:      strategyMapName,
	}, strategyMap)

	if err != nil {
		t.Fatal("could not get aws strategy map", err)
	}

	// get the create strategy for _network in the aws strategy configmap
	// if this doesn't exist, skip the test completely since we're dealing
	// with legacy cro networking
	networkStrategy := strategyMap.Data[resourceType]
	if networkStrategy != "" {
		t.Skip("_network key exists in aws strategy configmap, skipping legacy cluster vpc network test")
	}
	t.Log("Doing Legacy VPC Test")

	// get the cluster id used for tagging aws resources
	clusterTag, err := getClusterID(ctx, testingCtx.Client)
	if err != nil {
		t.Fatal("could not get cluster id", err)
	}

	// create a new session
	session, err := CreateAWSSession(ctx, testingCtx.Client)
	if err != nil {
		t.Fatal("could not create aws session", err)
	}

	ec2Svc := ec2.New(session)

	// Get the availability zones for the region the cluster is in
	azs, err := getAvailabilityZones(ec2Svc)

	if err != nil {
		t.Fatal("could not get aws availability zones", err)
	}

	// find and verify the cluster VPC
	vpc, err := verifyLegacyVPC(ec2Svc, clusterTag)
	testErrors.vpcError = err.(*networkConfigTestError).vpcError

	if vpc == nil {
		t.Fatal(testErrors.Error())
	}

	clusterVpcId := *vpc.VpcId

	vpcSubnets, err := getLegacySubnets(ec2Svc, clusterTag, clusterVpcId)
	testErrors.subnetsError = err.(*networkConfigTestError).subnetsError

	// we have to manually construct the subnet group names for rds and elasticache,
	// since tag filtering isnt currently available
	subnetGroupName := resources.ShortenString(fmt.Sprintf("%s-%s", clusterTag, "subnet-group"), 40)

	rdsSvc := rds.New(session)
	err = verifyLegacyRDSSubnetGroups(rdsSvc, subnetGroupName, clusterVpcId, vpcSubnets, azs)
	testErrors.rdsSubnetGroupsError = err.(*networkConfigTestError).rdsSubnetGroupsError

	cacheSvc := elasticache.New(session)
	err = verifyLegacyCacheSubnetGroups(cacheSvc, subnetGroupName, clusterVpcId, vpcSubnets, azs)
	testErrors.cacheSubnetGroupsError = err.(*networkConfigTestError).cacheSubnetGroupsError

	// if any error was found, fail the test
	if testErrors.hasError() {
		t.Fatal(testErrors.Error())
	}
}

func getAvailabilityZones(ec2Svc *ec2.EC2) ([]*string, error) {
	azs := make([]*string, 0)

	describeOutput, err := ec2Svc.DescribeAvailabilityZones(&ec2.DescribeAvailabilityZonesInput{})

	if err != nil {
		return azs, err
	}

	for _, zone := range describeOutput.AvailabilityZones {
		azs = append(azs, zone.ZoneName)
	}

	return azs, nil
}

func verifyLegacyVPC(session *ec2.EC2, clusterTag string) (*ec2.Vpc, error) {
	// filter vpcs by integreatly cluster id tag

	newErr := &networkConfigTestError{
		vpcError: []error{},
	}

	describeVpcs, err := session.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String(fmt.Sprintf("tag:kubernetes.io/cluster/%s", clusterTag)),
				Values: []*string{aws.String("owned")},
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

	return vpcs[0], newErr
}

// verify that the vpc subnets are created
func getLegacySubnets(session *ec2.EC2, clusterTag, clusterVPCId string) ([]*ec2.Subnet, error) {
	newErr := &networkConfigTestError{
		subnetsError: []error{},
	}

	// filter subnets by integreatly cluster id tag
	describeSubnets, err := session.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(clusterVPCId)},
			},
		},
	})
	if err != nil {
		errMsg := fmt.Errorf("could not describe subnets: %v", err)
		newErr.subnetsError = append(newErr.subnetsError, errMsg)
		return nil, newErr
	}

	return describeSubnets.Subnets, newErr
}

func verifyLegacyRDSSubnetGroups(rdsSvc *rds.RDS, subnetGroupName, clusterVPCId string, vpcSubnets []*ec2.Subnet, azs []*string) error {
	newErr := &networkConfigTestError{
		rdsSubnetGroupsError: []error{},
	}

	// get rds subnet groups by subnet group name
	describeGroups, err := rdsSvc.DescribeDBSubnetGroups(&rds.DescribeDBSubnetGroupsInput{
		DBSubnetGroupName: aws.String(subnetGroupName),
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

	subnetGroup := subnetGroups[0]

	// ensure the subnet group belongs to the cluster VPC
	if *subnetGroup.VpcId != clusterVPCId {
		errMsg := fmt.Errorf("rds subnet group %s does not belong to cluster VPC. got = %s, wanted = %s", *subnetGroup.DBSubnetGroupName, *subnetGroup.VpcId, clusterVPCId)
		newErr.rdsSubnetGroupsError = append(newErr.rdsSubnetGroupsError, errMsg)
	}

	// ensure each subnet is found in the list of subnets found in the cluster vpc
	subnetGroupSubnets := make([]*ec2.Subnet, 0)
	for _, cacheSubnet := range subnetGroup.Subnets {
		ec2Subnet := findSubnetInList(*cacheSubnet.SubnetIdentifier, vpcSubnets)
		if ec2Subnet == nil {
			errMsg := fmt.Errorf("rds subnet group %+v has a subnet that doesn't belong to the cluster VPC %s", subnetGroup, clusterVPCId)
			newErr.rdsSubnetGroupsError = append(newErr.rdsSubnetGroupsError, errMsg)
		}
		subnetGroupSubnets = append(subnetGroupSubnets, ec2Subnet)
	}

	// verify that the subnets have the right tag that indicates it's private
	for _, subnet := range subnetGroupSubnets {
		if !subnetContainsTagKey(subnet, internalSubnetTag) {
			errMsg := fmt.Errorf("rds subnet %s doesn't have the tag %s", *subnet.SubnetId, internalSubnetTag)
			newErr.rdsSubnetGroupsError = append(newErr.rdsSubnetGroupsError, errMsg)
		}
	}

	// the list of availability zones that have been covered by the subnets
	// within the DBSubnetGroup
	coveredAZs := make([]*string, 0)
	for _, subnet := range subnetGroup.Subnets {
		if !contains(coveredAZs, subnet.SubnetAvailabilityZone.Name) {
			coveredAZs = append(coveredAZs, subnet.SubnetAvailabilityZone.Name)
		}
	}

	// check the lengths match, i.e. the subnets cover every AZ in the region
	if len(coveredAZs) != len(azs) {
		errMsg := fmt.Errorf("rds subnet group does not have a subnet group for each availability zone. Availability Zones: %+v Subnet Groups: %+v", subnetGroup.Subnets, azs)
		newErr.rdsSubnetGroupsError = append(newErr.rdsSubnetGroupsError, errMsg)
	}

	return newErr
}

func verifyLegacyCacheSubnetGroups(cacheSvc *elasticache.ElastiCache, subnetGroupName, clusterVPCId string, vpcSubnets []*ec2.Subnet, azs []*string) error {
	newErr := &networkConfigTestError{
		cacheSubnetGroupsError: []error{},
	}

	// get elasticache subnet groups by subnet group name
	describeCacheGroups, err := cacheSvc.DescribeCacheSubnetGroups(&elasticache.DescribeCacheSubnetGroupsInput{
		CacheSubnetGroupName: aws.String(subnetGroupName),
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

	subnetGroup := cacheSubnetGroups[0]

	// ensure the subnet group belongs to the cluster vpc
	if *subnetGroup.VpcId != clusterVPCId {
		errMsg := fmt.Errorf("elasticache subnet group %s does not belong to cluster VPC. got = %s, wanted = %s", *subnetGroup.CacheSubnetGroupName, *subnetGroup.VpcId, clusterVPCId)
		newErr.cacheSubnetGroupsError = append(newErr.cacheSubnetGroupsError, errMsg)
	}

	// ensure each subnet is found in the list of subnets found in the cluster vpc
	subnetGroupSubnets := make([]*ec2.Subnet, 0)
	for _, cacheSubnet := range subnetGroup.Subnets {
		ec2Subnet := findSubnetInList(*cacheSubnet.SubnetIdentifier, vpcSubnets)
		if ec2Subnet == nil {
			errMsg := fmt.Errorf("elasticache subnet group %+v has a subnet that doesn't belong to the cluster VPC %s", subnetGroup, clusterVPCId)
			newErr.cacheSubnetGroupsError = append(newErr.cacheSubnetGroupsError, errMsg)
		}
		subnetGroupSubnets = append(subnetGroupSubnets, ec2Subnet)
	}

	// verify that the subnets have the right tag that indicates it's private
	for _, subnet := range subnetGroupSubnets {
		if !subnetContainsTagKey(subnet, internalSubnetTag) {
			errMsg := fmt.Errorf("elasticache subnet %s doesn't have the tag %s", *subnet.SubnetId, internalSubnetTag)
			newErr.cacheSubnetGroupsError = append(newErr.cacheSubnetGroupsError, errMsg)
		}
	}

	// the list of availability zones that have been covered by the subnets
	// within the DBSubnetGroup
	coveredAZs := make([]*string, 0)
	for _, subnet := range subnetGroup.Subnets {
		if !contains(coveredAZs, subnet.SubnetAvailabilityZone.Name) {
			coveredAZs = append(coveredAZs, subnet.SubnetAvailabilityZone.Name)
		}
	}

	// check the lengths match, i.e. the subnets cover every AZ in the region
	if len(coveredAZs) != len(azs) {
		errMsg := fmt.Errorf("elasticache subnet group does not have a subnet group for each availability zone. Availability Zones: %+v Subnet Groups: %+v", subnetGroup.Subnets, azs)
		newErr.cacheSubnetGroupsError = append(newErr.cacheSubnetGroupsError, errMsg)
	}

	return newErr
}

func findSubnetInList(subnetIdentifier string, vpcSubnets []*ec2.Subnet) *ec2.Subnet {
	for _, vpcSubnet := range vpcSubnets {
		if subnetIdentifier == *vpcSubnet.SubnetId {
			return vpcSubnet
		}
	}
	return nil
}

func subnetContainsTagKey(subnet *ec2.Subnet, tagKey string) bool {
	for _, tag := range subnet.Tags {
		if *tag.Key == tagKey {
			return true
		}
	}
	return false
}
