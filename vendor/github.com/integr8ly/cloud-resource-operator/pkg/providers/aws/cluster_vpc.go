package aws

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"net"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	errorUtil "github.com/pkg/errors"
)

const (
	defaultSubnetPostfix          = "subnet-group"
	defaultSecurityGroupPostfix   = "security-group"
	defaultAWSPrivateSubnetTagKey = "kubernetes.io/role/internal-elb"
	defaultSubnetGroupDesc        = "Subnet group created and managed by the Cloud Resource Operator"
	// Default subnet mask is AWS's minimum possible value
	defaultSubnetMask = 27
)

// ensures a subnet group is in place for the creation of a resource
func configureSecurityGroup(ctx context.Context, c client.Client, ec2Svc ec2iface.EC2API) error {
	logrus.Info("ensuring security group is correct for resource")
	// get cluster id
	clusterID, err := resources.GetClusterID(ctx, c)
	if err != nil {
		return errorUtil.Wrap(err, "error getting cluster id")
	}

	// build security group name
	secName, err := BuildInfraName(ctx, c, defaultSecurityGroupPostfix, DefaultAwsIdentifierLength)
	logrus.Info(fmt.Sprintf("setting resource security group %s", secName))
	if err != nil {
		return errorUtil.Wrap(err, "error building subnet group name")
	}

	// get cluster cidr group
	vpcID, cidr, err := GetCidr(ctx, c, ec2Svc)
	if err != nil {
		return errorUtil.Wrap(err, "error finding cidr block")
	}

	foundSecGroup, err := getSecurityGroup(ec2Svc, secName)
	if err != nil {
		return errorUtil.Wrap(err, "error get security group")
	}

	if foundSecGroup == nil {
		// create security group
		logrus.Info(fmt.Sprintf("creating security group from cluster %s", clusterID))
		if _, err := ec2Svc.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
			Description: aws.String(fmt.Sprintf("security group for cluster %s", clusterID)),
			GroupName:   aws.String(secName),
			VpcId:       aws.String(vpcID),
		}); err != nil {
			return errorUtil.Wrap(err, "error creating security group")
		}
		return nil
	}

	// build ip permission
	ipPermission := &ec2.IpPermission{
		IpProtocol: aws.String("-1"),
		IpRanges: []*ec2.IpRange{
			{
				CidrIp: aws.String(cidr),
			},
		},
	}

	// check if correct permissions are in place
	for _, perm := range foundSecGroup.IpPermissions {
		if reflect.DeepEqual(perm, ipPermission) {
			logrus.Info("ip permissions are correct for postgres resource")
			return nil
		}
	}

	// authorize ingress
	logrus.Info(fmt.Sprintf("setting ingress ip permissions for %s ", *foundSecGroup.GroupName))
	if _, err := ec2Svc.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(*foundSecGroup.GroupId),
		IpPermissions: []*ec2.IpPermission{
			ipPermission,
		},
	}); err != nil {
		return errorUtil.Wrap(err, "error authorizing security group ingress")
	}

	return nil
}

// GetVPCSubnets returns a list of subnets associated with cluster VPC
func GetVPCSubnets(ctx context.Context, c client.Client, ec2Svc ec2iface.EC2API) ([]*ec2.Subnet, error) {
	logrus.Info("gathering cluster vpc and subnet information")
	// poll subnets to ensure credentials have reconciled
	subs, err := getSubnets(ec2Svc)
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting subnets")
	}

	// get cluster vpc
	foundVPC, err := getVpc(ctx, c, ec2Svc)
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting vpcs")
	}

	// check if found cluster vpc
	if foundVPC == nil {
		return nil, errorUtil.New("error, unable to find a vpc")
	}

	// find associated subnets
	var associatedSubs []*ec2.Subnet
	for _, sub := range subs {
		if *sub.VpcId == *foundVPC.VpcId {
			associatedSubs = append(associatedSubs, sub)
		}
	}

	// check if found subnets associated with cluster vpc
	if associatedSubs == nil {
		return nil, errorUtil.New("error, unable to find subnets associated with cluster vpc")
	}

	return associatedSubs, nil
}

// GetSubnetIDS returns a list of subnet ids associated with cluster vpc
func GetPrivateSubnetIDS(ctx context.Context, c client.Client, ec2Svc ec2iface.EC2API) ([]*string, error) {
	logrus.Info("gathering all private subnets in cluster vpc")
	// get cluster vpc
	foundVPC, err := getVpc(ctx, c, ec2Svc)
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting vpcs")
	}

	// get subnets in vpc
	subs, err := GetVPCSubnets(ctx, c, ec2Svc)
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting vpc subnets")
	}

	// get a list of availability zones
	azs, err := getAZs(ec2Svc)
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting availability zones")
	}

	// filter based on a tag key attached to private subnets
	var privSubs []*ec2.Subnet
	for _, sub := range subs {
		for _, tags := range sub.Tags {
			if *tags.Key == defaultAWSPrivateSubnetTagKey {
				privSubs = append(privSubs, sub)
			}
		}
	}

	// for every az check there is a private subnet, if none create one
	for _, az := range azs {
		logrus.Infof("checking if private subnet exists in zone %s", *az.ZoneName)
		if !privateSubnetExists(privSubs, az) {
			logrus.Info(fmt.Sprintf("no private subnet found in %s", *az.ZoneName))
			subnet, err := createPrivateSubnet(ctx, c, ec2Svc, foundVPC, *az.ZoneName)
			if err != nil {
				return nil, errorUtil.Wrap(err, "failed to created private subnet")
			}
			privSubs = append(privSubs, subnet)
		}
	}

	// build list of subnet ids
	var subIDs []*string
	for _, sub := range privSubs {
		subIDs = append(subIDs, sub.SubnetId)
	}

	if subIDs == nil {
		return nil, errorUtil.New("failed to get list of private subnet ids")
	}

	return subIDs, nil
}

// checks is a private subnet exists and is available in an availability zone
func privateSubnetExists(privSubs []*ec2.Subnet, zone *ec2.AvailabilityZone) bool {
	for _, subnet := range privSubs {
		if *subnet.AvailabilityZone == *zone.ZoneName && *zone.State == "available" {
			return true
		}
	}
	return false
}

// creates and tags a private subnet
func createPrivateSubnet(ctx context.Context, c client.Client, ec2Svc ec2iface.EC2API, vpc *ec2.Vpc, zone string) (*ec2.Subnet, error) {
	// get list of potential subnet addresses
	logrus.Info(fmt.Sprintf("creating private subnet in %s", *vpc.VpcId))
	subs, err := buildSubnetAddress(vpc)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to build subnets")
	}

	// create subnet looping through potential subnet list
	var subnet *ec2.Subnet
	for _, ip := range subs {
		logrus.Infof("attempting to create subnet with cidr block %s for vpc %s in zone %s", ip.String(), *vpc.VpcId, zone)
		createOutput, err := ec2Svc.CreateSubnet(&ec2.CreateSubnetInput{
			AvailabilityZone: aws.String(zone),
			CidrBlock:        aws.String(ip.String()),
			VpcId:            aws.String(*vpc.VpcId),
		})
		ec2err, isAwsErr := err.(awserr.Error)
		if err != nil && isAwsErr && ec2err.Code() == "InvalidSubnet.Conflict" {
			logrus.Info(fmt.Sprintf("%s conflicts with a current subnet, trying again", ip))
			continue
		}
		if err != nil {
			return nil, errorUtil.Wrap(err, "error creating new subnet")
		}
		if newErr := tagPrivateSubnet(ctx, c, ec2Svc, createOutput.Subnet); newErr != nil {
			return nil, newErr
		}
		logrus.Info(fmt.Sprintf("created new subnet %s in %s", ip, *vpc.VpcId))
		subnet = createOutput.Subnet
		break
	}

	return subnet, nil
}

// tags a private subnet with the default aws private subnet tag
func tagPrivateSubnet(ctx context.Context, c client.Client, ec2Svc ec2iface.EC2API, sub *ec2.Subnet) error {
	logrus.Info(fmt.Sprintf("adding tags to subnet %s", *sub.SubnetId))
	// get cluster id
	clusterID, err := resources.GetClusterID(ctx, c)
	if err != nil {
		return errorUtil.Wrap(err, "error getting clusterID")
	}
	organizationTag := resources.GetOrganizationTag()

	_, err = ec2Svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{
			aws.String(*sub.SubnetId),
		},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String(defaultAWSPrivateSubnetTagKey),
				Value: aws.String("1"),
			}, {
				Key:   aws.String(fmt.Sprintf("%sclusterID", organizationTag)),
				Value: aws.String(clusterID),
			},
		},
	})
	if err != nil {
		return errorUtil.Wrap(err, "failed to tag subnet")
	}
	return nil
}

// builds an array list of potential subnet addresses
func buildSubnetAddress(vpc *ec2.Vpc) ([]net.IPNet, error) {
	logrus.Info(fmt.Sprintf("calculating subnet mask and address for vpc cidr %s", *vpc.CidrBlock))
	if *vpc.CidrBlock == "" {
		return nil, errorUtil.New("vpc cidr block can't be empty")
	}

	// Get details about the VPC CIDR block
	_, awsCIDR, err := net.ParseCIDR(*vpc.CidrBlock)
	if err != nil {
		return nil, errorUtil.Wrapf(err, "failed to parse vpc cidr block %s", *vpc.CidrBlock)
	}
	maskSize, _ := awsCIDR.Mask.Size()

	// Return descriptive error if VPC CIDR block cannot contain subnet we want to generate
	if maskSize >= defaultSubnetMask {
		return nil, errorUtil.New(fmt.Sprintf("vpc cidr block %s cannot contain generated subnet mask /%d", *vpc.CidrBlock, defaultSubnetMask))
	}

	croCIDRStr := fmt.Sprintf("%s/%d", awsCIDR.IP.String(), defaultSubnetMask)
	_, croCIDR, err := net.ParseCIDR(croCIDRStr)
	if err != nil {
		return nil, errorUtil.Wrapf(err, "failed to parse cro cidr block %s", croCIDRStr)
	}
	networks := generateAvailableSubnets(awsCIDR, croCIDR)
	// Reverse the network list as the end networks are more likely to be unused
	for i, j := 0, len(networks)-1; i < j; i, j = i+1, j-1 {
		networks[i], networks[j] = networks[j], networks[i]
	}
	return networks, nil
}

func generateAvailableSubnets(fromCIDR, toCIDR *net.IPNet) []net.IPNet {
	toIPv4 := toCIDR.IP.To4()

	networks := []net.IPNet{
		{
			IP:   toIPv4,
			Mask: toCIDR.Mask,
		},
	}
	for i := 0; fromCIDR.Contains(incrementIP(toIPv4, i)); i++ {
		nextFoundNetwork := incrementIP(toIPv4, i)
		nextFoundNetworkMasked := nextFoundNetwork.Mask(toCIDR.Mask)
		// Don't need duplicates
		if containsNetwork(networks, nextFoundNetworkMasked) {
			continue
		}
		networks = append(networks, net.IPNet{
			IP:   nextFoundNetworkMasked,
			Mask: toCIDR.Mask,
		})
	}
	return networks
}

func containsNetwork(networks []net.IPNet, toFind net.IP) bool {
	for _, n := range networks {
		if n.IP.Equal(toFind) {
			return true
		}
	}
	return false
}

func incrementIP(ip net.IP, inc int) net.IP {
	ipv4 := ip.To4()
	v := uint(ipv4[0])<<24 + uint(ipv4[1])<<16 + uint(ipv4[2])<<8 + uint(ipv4[3])
	v += uint(inc)
	v3 := byte(v & 0xFF)
	v2 := byte((v >> 8) & 0xFF)
	v1 := byte((v >> 16) & 0xFF)
	v0 := byte((v >> 24) & 0xFF)
	return net.IPv4(v0, v1, v2, v3)
}

// returns vpc id and cidr block for found vpc
func GetCidr(ctx context.Context, c client.Client, ec2Svc ec2iface.EC2API) (string, string, error) {
	logrus.Info("gathering cidr block for cluster")
	foundVPC, err := getVpc(ctx, c, ec2Svc)
	if err != nil {
		return "", "", errorUtil.Wrap(err, "error getting vpcs")
	}

	// check if found cluster vpc
	if foundVPC == nil {
		return "", "", errorUtil.New("error, unable to find a vpc")
	}

	return *foundVPC.VpcId, *foundVPC.CidrBlock, nil
}

// function to get AZ
func getAZs(ec2Svc ec2iface.EC2API) ([]*ec2.AvailabilityZone, error) {
	logrus.Info("gathering cluster availability zones")
	azs, err := ec2Svc.DescribeAvailabilityZones(&ec2.DescribeAvailabilityZonesInput{})
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting availability zones")
	}
	return azs.AvailabilityZones, nil
}

// function to get subnets, used to check/wait on AWS credentials
func getSubnets(ec2Svc ec2iface.EC2API) ([]*ec2.Subnet, error) {
	var subs []*ec2.Subnet
	err := wait.PollImmediate(time.Second*5, time.Minute*5, func() (done bool, err error) {
		listOutput, err := ec2Svc.DescribeSubnets(&ec2.DescribeSubnetsInput{})
		if err != nil {
			return false, nil
		}
		subs = listOutput.Subnets
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return subs, nil
}

// function to get vpc of a cluster
func getVpc(ctx context.Context, c client.Client, ec2Svc ec2iface.EC2API) (*ec2.Vpc, error) {
	logrus.Info("finding cluster vpc")
	// get vpcs
	vpcs, err := ec2Svc.DescribeVpcs(&ec2.DescribeVpcsInput{})
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting subnets")
	}

	// get cluster id
	clusterID, err := resources.GetClusterID(ctx, c)
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting clusterID")
	}

	// find associated vpc to cluster
	var foundVPC *ec2.Vpc
	for _, vpc := range vpcs.Vpcs {
		for _, tag := range vpc.Tags {
			if *tag.Value == fmt.Sprintf("%s-vpc", clusterID) {
				foundVPC = vpc
			}
		}
	}

	if foundVPC == nil {
		return nil, errorUtil.New("error, no vpc found")
	}

	return foundVPC, nil
}

func getSecurityGroup(ec2Svc ec2iface.EC2API, secName string) (*ec2.SecurityGroup, error) {
	// get security groups
	secGroups, err := ec2Svc.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to return information about security groups")
	}

	// check if security group exists
	var foundSecGroup *ec2.SecurityGroup
	for _, sec := range secGroups.SecurityGroups {
		if *sec.GroupName == secName {
			foundSecGroup = sec
			break
		}
	}

	return foundSecGroup, nil
}
