// utility to manage a standalone vpc for the resources created by the cloud resource operator.
//
// this has been added to allow the operator to work with clusters provisioned with openshift tooling >= 4.4.6,
// as they will not allow multi-az cloud resources to be created in single-az openshift clusters due to the single-az
// cluster subnets taking up all networking addresses in the cluster vpc.
//
// any openshift clusters that have used the cloud resource operator before this utility was added will be using the
// old approach, which is bundling cloud resources in with the cluster vpc. backwards compatibility for this approach
// must be maintained.
//
// the cloud resource vpc will be peered to the openshift cluster vpc. this peering will allow networking between the
// two vpcs e.g. a product in openshift connecting to a database created by the cloud resource operator. the peering
// should be the last element to be removed as it will block an openshift ocm cluster from being torn down which will
// allow for easier alerting with aws resources not being cleaned up successfully.
//
// see [1] for more details.
//
// [1] https://docs.google.com/document/d/1UWfon-tBNfiDS5pJRAUqPXoJuUUqO1P4B6TTR8SMqSc/edit?usp=sharing
//
// terminology
// bundled: refers to networking resources installed using the old approach
// standalone: refers to networking resources installed using the new approach

package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/elasticache/elasticacheiface"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	v12 "github.com/integr8ly/cloud-resource-operator/apis/config/v1"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	errorUtil "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultRHMIVpcNameTagValue           = "RHMI Cloud Resource VPC"
	DefaultRHMISubnetNameTagValue        = "RHMI Cloud Resource Subnet"
	DefaultRHMISecurityGroupNameTagValue = "RHMI Cloud Resource Security Group"
	DefaultRHMIVpcRouteTableNameTagValue = "RHMI Cloud Resource Route Table"
	defaultNumberOfExpectedSubnets       = 2
	clusterOwnedTagKeyPrefix             = "kubernetes.io/cluster/"
	clusterOwnedTagValue                 = "owned"
	//network peering
	tagVpcPeeringNameValue = "RHMI Cloud Resource Peering Connection"
	// filter names for vpc peering connections
	// see https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeVpcPeeringConnections.html
	filterVpcPeeringRequesterId = "requester-vpc-info.vpc-id"
	filterVpcPeeringAccepterId  = "accepter-vpc-info.vpc-id"
	// default cidr values
	defaultCIDRMask = "26"
)

// wrapper for ec2 vpcs, to allow for extensibility
type Network struct {
	Vpc     *ec2.Vpc
	Subnets []*ec2.Subnet
}

// used to map expected ip addresses to availability zones
type NetworkAZSubnet struct {
	IP net.IPNet
	AZ *ec2.AvailabilityZone
}

// wrapper for ec2 vpc peering connections, to allow for extensibility
type NetworkPeering struct {
	PeeringConnection *ec2.VpcPeeringConnection
}

type NetworkConnection struct {
	StandaloneSecurityGroup *ec2.SecurityGroup
}

func (np *NetworkPeering) IsReady() bool {
	return aws.StringValue(np.PeeringConnection.Status.Code) == ec2.VpcPeeringConnectionStateReasonCodeActive
}

//go:generate moq -out cluster_network_provider_moq.go . NetworkManager
type NetworkManager interface {
	CreateNetwork(context.Context, *net.IPNet) (*Network, error)
	DeleteNetwork(context.Context) error
	CreateNetworkConnection(context.Context, *Network) (*NetworkConnection, error)
	DeleteNetworkConnection(context.Context, *NetworkPeering) error
	CreateNetworkPeering(context.Context, *Network) (*NetworkPeering, error)
	GetClusterNetworkPeering(context.Context) (*NetworkPeering, error)
	DeleteNetworkPeering(*NetworkPeering) error
	IsEnabled(context.Context) (bool, error)
	DeleteBundledCloudResources(context.Context) error
}

var _ NetworkManager = (*NetworkProvider)(nil)

type NetworkProvider struct {
	Client         client.Client
	RdsApi         rdsiface.RDSAPI
	Ec2Api         ec2iface.EC2API
	ElasticacheApi elasticacheiface.ElastiCacheAPI
	Logger         *logrus.Entry
}

func NewNetworkManager(session *session.Session, client client.Client, logger *logrus.Entry) *NetworkProvider {
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger())
	}
	return &NetworkProvider{
		Client:         client,
		RdsApi:         rds.New(session),
		Ec2Api:         ec2.New(session),
		ElasticacheApi: elasticache.New(session),
		Logger:         logger.WithField("provider", "standalone_network_provider"),
	}
}

//CreateNetwork returns a Network type or error
//
//VPC's created by the cloud resource operator are identified by having a tag with the name `<organizationTag>/clusterID`.
//By default, `integreatly.org/clusterID`.
//
//CreateNetwork does:
//    * create a VPC with CIDR block and tag it, if a VPC does not exist,
//	  * reconcile on subnets and subnet groups
//
//CreateNetwork does not:
//	  * reconcile the vpc if the VPC already exist (this is to avoid potential changes to the CIDR range and unwanted/unexpected behaviour)
func (n *NetworkProvider) CreateNetwork(ctx context.Context, vpcCidrBlock *net.IPNet) (*Network, error) {
	logger := n.Logger.WithField("action", "CreateNetwork")

	// check if there is cluster specific vpc already created.
	foundVpc, err := getStandaloneVpc(ctx, n.Client, n.Ec2Api, logger)
	if err != nil {
		return nil, errorUtil.Wrap(err, "unable to get vpc")
	}
	if foundVpc == nil {
		//VPCs must be created with a valid CIDR Block range, between \16 and \26
		//if an invalid range is passed, the function returns an error
		//
		//VPCs are tagged with the name `<organizationTag>/clusterID`.
		//By default, `integreatly.org/clusterID`.
		//
		//NOTE - Once a VPC is created we do not want to update it. To avoid changing cidr block
		clusterVPC, err := getClusterVpc(ctx, n.Client, n.Ec2Api, n.Logger)
		if err != nil {
			return nil, errorUtil.Wrap(err, "failed to get cluster vpc")
		}

		_, clusterVPCCidr, err := net.ParseCIDR(*clusterVPC.CidrBlock)
		if err != nil {
			return nil, errorUtil.Wrap(err, "failed to parse cluster vpc cidr block")
		}

		// standalone vpc cidr block can not overlap with existing cluster vpc cidr block
		// issue arises when trying to peer both vpcs with invalid vpc error - `overlapping CIDR range`
		// we need to ensure both cidr ranges do not intersect before creating the standalone vpc
		// as the creation of a standalone vpc is designed to be a one shot pass and not to be updated after the fact
		if err := validateStandaloneCidrBlock(vpcCidrBlock, clusterVPCCidr); err != nil {
			return nil, errorUtil.Wrap(err, "vpc validation failure")
		}
		logger.Infof("cidr %s is valid 👍", vpcCidrBlock.String())

		// create vpc using cidr string from _network
		createVpcOutput, err := n.Ec2Api.CreateVpc(&ec2.CreateVpcInput{
			CidrBlock: aws.String(vpcCidrBlock.String()),
		})

		ec2Err, isAwsErr := err.(awserr.Error)
		if err != nil && isAwsErr && ec2Err.Code() == "InvalidVpc.Range" {
			return nil, errorUtil.New(fmt.Sprintf("%s is out of range, block sizes must be between `/16` and `/26`, please update `_network` strategy", vpcCidrBlock.String()))
		}
		if err != nil {
			return nil, errorUtil.Wrap(err, "unexpected error creating vpc")
		}
		logger.Infof("creating vpc: %s", *createVpcOutput.Vpc.VpcId)

		waitVpcErr := n.Ec2Api.WaitUntilVpcExists(&ec2.DescribeVpcsInput{
			VpcIds: []*string{
				aws.String(*createVpcOutput.Vpc.VpcId),
			},
		})
		if waitVpcErr != nil {
			logger.Warnf(
				"timed out waiting to check if vpc %s with status %s exists, operator will delete the VPC and recreate another one: %v",
				*createVpcOutput.Vpc.VpcId,
				*createVpcOutput.Vpc.State,
				waitVpcErr,
			)
			_, err = n.Ec2Api.DeleteVpc(&ec2.DeleteVpcInput{
				VpcId: aws.String(*createVpcOutput.Vpc.VpcId),
			})
			if err != nil {
				logger.Errorf("unable to delete vpc %s after failing to check if it exists: %v", *createVpcOutput.Vpc.VpcId, err)
			}
			return &Network{}, nil
		}

		logger.Infof("vpc %s is created with state: %s", *createVpcOutput.Vpc.VpcId, *createVpcOutput.Vpc.State)

		// ensure standalone vpc has correct tags
		clusterIDTag, err := getCloudResourceOperatorOwnerTag(ctx, n.Client)
		if err != nil {
			return nil, errorUtil.Wrap(err, "error generating cloud resource owner tag")
		}
		if err = n.reconcileVPCTags(createVpcOutput.Vpc, clusterIDTag); err != nil {
			return nil, errorUtil.Wrapf(err, "unexpected error while reconciling vpc tags")
		}

		return &Network{
			Vpc:     createVpcOutput.Vpc,
			Subnets: nil,
		}, nil
	}

	// reconciling on vpc networking, ensuring the following are present :
	//     * tagging standalone vpc route table
	//     * subnets (2 private)
	//     * subnet groups -> rds and elasticache
	//     * security group
	//     * route table configuration -> cluster and standalone vpc route tables

	// tag standalone vpc route table
	if err := n.reconcileStandaloneRouteTableTags(ctx, foundVpc, n.Logger); err != nil {
		return nil, errorUtil.Wrap(err, "unexpected error tagging standalone vpc route table")
	}

	// create standalone vpc subnets
	privateSubnets, err := n.reconcileStandaloneVPCSubnets(ctx, n.Logger, foundVpc)
	if err != nil {
		return nil, errorUtil.Wrap(err, "unexpected error creating vpc subnets")
	}

	// create rds subnet group
	if err = n.reconcileRDSVpcConfiguration(ctx, privateSubnets); err != nil {
		return nil, errorUtil.Wrap(err, "unexpected error reconciling standalone rds vpc networking")
	}

	// create elasticache subnet groups
	if err = n.reconcileElasticacheVPCConfiguration(ctx, privateSubnets); err != nil {
		return nil, errorUtil.Wrap(err, "unexpected error reconciling standalone elasticache vpc networking")
	}

	return &Network{
		Vpc:     foundVpc,
		Subnets: privateSubnets,
	}, nil
}

//DeleteNetwork returns an error
//
//VPCs are tagged with with the name `<organizationTag>/clusterID`.
//By default, `integreatly.org/clusterID`.
//
//This tag is used to find a standalone VPC
//If found DeleteNetwork will attempt to remove:
//	* all vpc associated subnets
//	* both subnet groups (rds and elasticache)
//	* the vpc
func (n *NetworkProvider) DeleteNetwork(ctx context.Context) error {
	logger := n.Logger.WithField("action", "DeleteNetwork")

	//check if there is a standalone vpc already created.
	foundVpc, err := getStandaloneVpc(ctx, n.Client, n.Ec2Api, logger)
	if err != nil {
		return errorUtil.Wrap(err, "unable to get vpc")
	}
	if foundVpc == nil {
		logger.Info("no standalone vpc found")
		return nil
	}

	// remove all subnets created by cro
	vpcSubs, err := getVPCAssociatedSubnets(n.Ec2Api, logger, foundVpc)
	if err != nil {
		return errorUtil.Wrap(err, "failed to get standalone vpc subnets")
	}
	for _, subnet := range vpcSubs {
		logger.Infof("attempting to delete subnet with id: %s", *subnet.SubnetId)
		_, err = n.Ec2Api.DeleteSubnet(&ec2.DeleteSubnetInput{
			SubnetId: aws.String(*subnet.SubnetId),
		})
		if err != nil {
			return errorUtil.Wrapf(err, "failed to delete subnet with id: %s", *subnet.SubnetId)
		}
	}

	subnetGroupName, err := BuildInfraName(ctx, n.Client, defaultSubnetPostfix, DefaultAwsIdentifierLength)
	if err != nil {
		return errorUtil.Wrap(err, "error building subnet group name")
	}

	// remove rds subnet group created by cro
	rdsSubnetGroup, err := getRDSSubnetGroup(n.RdsApi, subnetGroupName)
	if err != nil {
		return errorUtil.Wrap(err, "error getting subnet group on delete")
	}
	if rdsSubnetGroup != nil {
		logger.Infof("attempting to delete subnetgroup name: %s for clusterID: %s", *rdsSubnetGroup.DBSubnetGroupName, *rdsSubnetGroup.VpcId)
		_, err := n.RdsApi.DeleteDBSubnetGroup(&rds.DeleteDBSubnetGroupInput{
			DBSubnetGroupName: rdsSubnetGroup.DBSubnetGroupName,
		})
		if err != nil {
			return errorUtil.Wrap(err, "error deleting subnet group")
		}
	}

	// remove elasticache subnet group created by cro
	elasticacheSubnetGroup, err := getElasticacheSubnetByGroup(n.ElasticacheApi, subnetGroupName)
	if err != nil {
		return errorUtil.Wrap(err, "error getting subnet group on delete")
	}
	if elasticacheSubnetGroup != nil {
		_, err := n.ElasticacheApi.DeleteCacheSubnetGroup(&elasticache.DeleteCacheSubnetGroupInput{
			CacheSubnetGroupName: aws.String(*elasticacheSubnetGroup.CacheSubnetGroupName),
		})
		if err != nil {
			return errorUtil.Wrap(err, "error deleting subnet group")
		}
	}

	// remove standalone vpc created by cro
	logger.Infof("attempting to delete vpc id: %s", *foundVpc.VpcId)
	_, err = n.Ec2Api.DeleteVpc(&ec2.DeleteVpcInput{
		VpcId: aws.String(*foundVpc.VpcId),
	})
	if err != nil {
		return errorUtil.Wrap(err, "unable to delete vpc")
	}
	logger.Infof("vpc %s deleted successfully", *foundVpc.VpcId)
	return nil
}

// CreateNetworkConnection handles the creation of a connection from the vpc provisioned by cro to the cluster vpc
// here we handle :
// 		* the standalone security group
//		* cro standalone vpc route table
//		* cluster vpc route table
func (n *NetworkProvider) CreateNetworkConnection(ctx context.Context, network *Network) (*NetworkConnection, error) {
	logger := n.Logger.WithField("action", "CreateNetworkConnection")
	logger.Info("preparing to configure network connection")

	// reconcile standalone vpc security groups
	securityGroup, err := n.reconcileStandaloneSecurityGroup(ctx, logger)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failure while reconciling standalone security group")
	}

	// a tag for identifying cluster owned vpc resources
	clusterVpcRouteTableTag, err := getClusterOwnerTag(ctx, n.Client)
	if err != nil {
		return nil, errorUtil.Wrap(err, "error building cluster owner tag")
	}

	// find cluter vpc route tables using cluster vpc route table tag
	// multiples route tables can exist for a single vpc (main and secondary)
	logger.Info("finding cluster route table(s)")
	clusterVpcRouteTables, err := n.getVPCRouteTable(clusterVpcRouteTableTag)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failure while getting vpc route table")
	}
	logger.Infof("found %d cluster vpc route tables", len(clusterVpcRouteTables))

	// get peering connection in order to provide peering connection id to new routes
	peeringConnection, err := n.getNetworkPeering(ctx, network)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failure while getting peering connection")
	}
	// we expect an active peering connection to be in place
	// if none exists return an error and re-reconcile
	if peeringConnection == nil {
		return nil, errorUtil.New("active peering connection expected and not found, can't create routes")
	}

	// declare cluster vpc route
	// we require the destination cidr block to that of the standalone vpc cidr block
	// we require the vpc connection id to be that of the vpc peering connection id
	clusterVpcRoute := &ec2.Route{
		VpcPeeringConnectionId: peeringConnection.VpcPeeringConnectionId,
		DestinationCidrBlock:   network.Vpc.CidrBlock,
	}

	// as more than one route table may exist we need to ensure that the cluster vpc route exists for each
	for _, routeTable := range clusterVpcRouteTables {
		logger.Infof("checking if route already exists for vpc peering connection id %s in route table %s", aws.StringValue(clusterVpcRoute.VpcPeeringConnectionId), aws.StringValue(routeTable.RouteTableId))
		if !routeExists(routeTable.Routes, clusterVpcRoute) {
			logger.Infof("creating route for vpc peering connection id %s in route table %s", aws.StringValue(clusterVpcRoute.VpcPeeringConnectionId), aws.StringValue(routeTable.RouteTableId))
			if _, err := n.Ec2Api.CreateRoute(&ec2.CreateRouteInput{
				VpcPeeringConnectionId: clusterVpcRoute.VpcPeeringConnectionId,
				DestinationCidrBlock:   clusterVpcRoute.DestinationCidrBlock,
				RouteTableId:           routeTable.RouteTableId,
			}); err != nil {
				return nil, errorUtil.Wrap(err, "failure while adding route to route table")
			}
		}
	}

	// get croOwner tag to use in getting standalone vpc route tables
	croOwnerTag, err := getCloudResourceOperatorOwnerTag(ctx, n.Client)
	if err != nil {
		return nil, errorUtil.Wrap(err, "error generating cloud resource owner tag")
	}

	// get standalone vpc route table using cro owner tag
	logger.Info("finding standalone route table(s)")
	standAloneVpcRouteTables, err := n.getVPCRouteTable(genericToEc2Tag(croOwnerTag))
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting standalone vpc route tables")
	}

	// we require the cluster vpc cidr block for standalone vpc route
	clusterVpc, err := getClusterVpc(ctx, n.Client, n.Ec2Api, logger)
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting standalone vpc route tables")
	}

	// declare standalone vpc route
	// we require the destination cidr block to that of the cluster vpc cidr block
	// we require the vpc connection id to be that of the vpc peering connection id
	standaloneVpcRoute := &ec2.Route{
		VpcPeeringConnectionId: peeringConnection.VpcPeeringConnectionId,
		DestinationCidrBlock:   clusterVpc.CidrBlock,
	}

	// we expect a single route table to exist for the standalone vpc
	// to handle the case where there is more than a single route found, loop through all them all and add the route
	for _, routeTable := range standAloneVpcRouteTables {
		logger.Infof("checking if route already exists for vpc peering connection id %s in route table %s", aws.StringValue(standaloneVpcRoute.VpcPeeringConnectionId), aws.StringValue(routeTable.RouteTableId))
		if !routeExists(routeTable.Routes, standaloneVpcRoute) {
			logger.Infof("creating route for vpc peering connection id %s in route table %s", aws.StringValue(standaloneVpcRoute.VpcPeeringConnectionId), aws.StringValue(routeTable.RouteTableId))
			if _, err := n.Ec2Api.CreateRoute(&ec2.CreateRouteInput{
				VpcPeeringConnectionId: standaloneVpcRoute.VpcPeeringConnectionId,
				DestinationCidrBlock:   standaloneVpcRoute.DestinationCidrBlock,
				RouteTableId:           routeTable.RouteTableId,
			}); err != nil {
				return nil, errorUtil.Wrap(err, "failure while adding route to route table")
			}
		}
	}

	return &NetworkConnection{
		StandaloneSecurityGroup: securityGroup,
	}, nil
}

// DeleteNetworkConnection removes the security group created by cro
func (n *NetworkProvider) DeleteNetworkConnection(ctx context.Context, networkPeering *NetworkPeering) error {
	logger := n.Logger.WithField("action", "DeleteNetworkConnection")
	// build security group name
	standaloneSecurityGroupName, err := BuildInfraName(ctx, n.Client, defaultSecurityGroupPostfix, DefaultAwsIdentifierLength)
	logger.Info(fmt.Sprintf("setting resource security group %s", standaloneSecurityGroupName))
	if err != nil {
		return errorUtil.Wrap(err, "error building subnet group name")
	}

	// get standalone security group
	standaloneSecGroup, err := getSecurityGroup(n.Ec2Api, standaloneSecurityGroupName)
	if err != nil {
		return errorUtil.Wrap(err, "failed to find standalone security group")
	}
	if standaloneSecGroup != nil {
		if _, err := n.Ec2Api.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
			GroupId: standaloneSecGroup.GroupId,
		}); err != nil {
			return errorUtil.Wrap(err, "failed to delete standalone security group")
		}
	}

	// a tag for identifying cluster owned vpc resources
	clusterVpcRouteTableTag, err := getClusterOwnerTag(ctx, n.Client)
	if err != nil {
		return errorUtil.Wrap(err, "error building cluster owner tag")
	}

	// find cluster vpc route tables using cluster vpc route table tag
	// multiple route tables can exist for a single vpc (main and secondary)
	logger.Info("finding cluster route table(s)")
	clusterVpcRouteTables, err := n.getVPCRouteTable(clusterVpcRouteTableTag)
	if err != nil {
		return errorUtil.Wrap(err, "failure while getting vpc route table")
	}
	logger.Infof("found %d cluster vpc route tables", len(clusterVpcRouteTables))

	standaloneVpc, err := getStandaloneVpc(ctx, n.Client, n.Ec2Api, logger)
	if err != nil {
		return errorUtil.Wrap(err, "could not find standalone vpc")
	}

	// we expect a peering connection to be in place to remove routes
	// if none exists return an error and re-reconcile to avoid nil pointer
	if networkPeering.PeeringConnection == nil {
		return errorUtil.New("peering connection expected and not found, can't delete routes")
	}

	// as more than one route table may exist we need to ensure that the cluster vpc route is deleted for each
	for _, routeTable := range clusterVpcRouteTables {
		logger.Infof("checking if route exists for standalone vpc id %s in route table %s", aws.StringValue(standaloneVpc.VpcId), aws.StringValue(routeTable.RouteTableId))
		if routeExists(routeTable.Routes, &ec2.Route{
			DestinationCidrBlock:   standaloneVpc.CidrBlock,
			VpcPeeringConnectionId: networkPeering.PeeringConnection.VpcPeeringConnectionId,
		}) {
			logger.Infof("deleting route for standalone vpc id %s in route table %s", aws.StringValue(standaloneVpc.VpcId), aws.StringValue(routeTable.RouteTableId))
			if _, err := n.Ec2Api.DeleteRoute(&ec2.DeleteRouteInput{
				DestinationCidrBlock: standaloneVpc.CidrBlock,
				RouteTableId:         routeTable.RouteTableId,
			}); err != nil {
				return errorUtil.Wrap(err, "failure while deleting route from route table")
			}
		}
	}
	logger.Info("standalone security group and cluster vpc routes deleted")
	return nil
}

// creates a peering connection between a provided vpc and the openshift cluster vpc
// used to enable network connectivity between the vpcs, so services in the openshift cluster can reach databases in
// the provided vpc
func (n *NetworkProvider) CreateNetworkPeering(ctx context.Context, network *Network) (*NetworkPeering, error) {
	logger := resources.NewActionLogger(n.Logger, "CreateNetworkPeering")

	clusterVpc, err := getClusterVpc(ctx, n.Client, n.Ec2Api, n.Logger)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to get cluster vpc")
	}

	peeringConnection, err := n.getNetworkPeering(ctx, network)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to get peering connection")
	}

	// create the vpc, we make an assumption they're in the same aws account and use the aws region in the aws client
	// provided to the NetworkProvider struct
	if peeringConnection == nil {
		logger.Infof("creating cluster peering connection for vpc %s", aws.StringValue(network.Vpc.VpcId))
		createPeeringConnOutput, err := n.Ec2Api.CreateVpcPeeringConnection(&ec2.CreateVpcPeeringConnectionInput{
			PeerVpcId: clusterVpc.VpcId,
			VpcId:     network.Vpc.VpcId,
		})
		if err != nil {
			return nil, errorUtil.Wrap(err, "failed to create vpc peering connection")
		}
		logger.Info("successfully peered vpc")
		peeringConnection = createPeeringConnOutput.VpcPeeringConnection
	}

	// once we have the peering connection, tag it so it's identifiable as belonging to this operator
	// this helps with cleaning up resources
	logger.Info("getting cluster identifier")
	clusterID, err := resources.GetClusterID(ctx, n.Client)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to get cluster id")
	}
	organizationTag := resources.GetOrganizationTag()
	clusterIDTagName := fmt.Sprintf("%sclusterID", organizationTag)

	logger.Infof("checking tags on peering connection")
	peeringConnectionTags := ec2TagsToGeneric(peeringConnection.Tags)
	if !tagsContains(peeringConnectionTags, tagDisplayName, tagVpcPeeringNameValue) ||
		!tagsContains(peeringConnectionTags, clusterIDTagName, clusterID) {
		logger.Info("creating tags on peering connection")
		_, err = n.Ec2Api.CreateTags(&ec2.CreateTagsInput{
			Resources: []*string{peeringConnection.VpcPeeringConnectionId},
			Tags: []*ec2.Tag{
				{
					Key:   aws.String(tagDisplayName),
					Value: aws.String(tagVpcPeeringNameValue),
				},
				{
					Key:   aws.String(clusterIDTagName),
					Value: aws.String(clusterID),
				},
			},
		})
		if err != nil {
			return nil, errorUtil.Wrap(err, "failed to tag peering connection")
		}
	} else {
		logger.Info("expected tags found on peering connection")
	}

	// peering connection now exists, we need to accept it to complete the setup
	logger.Infof("handling peering connection status %s", aws.StringValue(peeringConnection.Status.Code))
	switch aws.StringValue(peeringConnection.Status.Code) {
	case ec2.VpcPeeringConnectionStateReasonCodePendingAcceptance:
		logger.Info("accepting peering connection")
		_, err = n.Ec2Api.AcceptVpcPeeringConnection(&ec2.AcceptVpcPeeringConnectionInput{
			VpcPeeringConnectionId: peeringConnection.VpcPeeringConnectionId,
		})
		if err != nil {
			return nil, errorUtil.Wrap(err, "failed to accept vpc peering connection")
		}
		logger.Infof("accepted peering connection")
	case ec2.VpcPeeringConnectionStateReasonCodeActive, ec2.VpcPeeringConnectionStateReasonCodeProvisioning, ec2.VpcPeeringConnectionStateReasonCodeInitiatingRequest:
	default:
		return nil, errorUtil.New(fmt.Sprintf("vpc peering connection %s is in an invalid state '%s' with message '%s'", aws.StringValue(peeringConnection.VpcPeeringConnectionId), aws.StringValue(peeringConnection.Status.Code), aws.StringValue(peeringConnection.Status.Message)))
	}

	// return a wrapped vpc peering connection
	return &NetworkPeering{
		PeeringConnection: peeringConnection,
	}, nil
}

//GetClusterNetworking returns an active Net
func (n *NetworkProvider) GetClusterNetworkPeering(ctx context.Context) (*NetworkPeering, error) {
	logger := resources.NewActionLogger(n.Logger, "GetClusterNetworkPeering")

	logger.Info("getting standalone vpc")
	vpc, err := getStandaloneVpc(ctx, n.Client, n.Ec2Api, logger)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to get standalone vpc")
	}

	logger.Info("getting cluster network vpc peering")
	networkPeering, err := n.getNetworkPeering(ctx, &Network{
		Vpc: vpc,
	})
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to get network peering")
	}

	return &NetworkPeering{PeeringConnection: networkPeering}, nil
}

// deletes a provided vpc peering connection
// this will remove network connectivity between the vpcs that are part of the provided peering connection
func (n *NetworkProvider) DeleteNetworkPeering(peering *NetworkPeering) error {
	logger := resources.NewActionLogger(n.Logger, "DeleteNetworkPeering")
	if peering.PeeringConnection == nil {
		logger.Info("networking peering connection nil, skipping delete network peering")
		return nil
	}
	logger = logger.WithField("vpc_peering", aws.StringValue(peering.PeeringConnection.VpcPeeringConnectionId))

	// describe the vpc peering connection first, it could be possible that the peering connection is already in a
	// deleting state or is already deleted due to the way aws performs caching
	// get the vpc peering connection so we can have a look at it first to decide if a deletion request is required
	logger.Info("getting vpc peering")
	describePeeringOutput, err := n.Ec2Api.DescribeVpcPeeringConnections(&ec2.DescribeVpcPeeringConnectionsInput{
		VpcPeeringConnectionIds: []*string{peering.PeeringConnection.VpcPeeringConnectionId},
	})
	if err != nil {
		return errorUtil.Wrap(err, "failed to get vpc")
	}
	// we expect the describe function to return up to one vpc peering connection
	if len(describePeeringOutput.VpcPeeringConnections) == 0 {
		logger.Info("could not find peering connection, assuming already removed")
		return nil
	}
	logger.Infof("found %d vpc peering connections, taking first", len(describePeeringOutput.VpcPeeringConnections))
	toDelete := describePeeringOutput.VpcPeeringConnections[0]
	// if the vpc peering connection is in a deleting/deleted state, ignore it as aws will handle it
	switch aws.StringValue(toDelete.Status.Code) {
	case ec2.VpcPeeringConnectionStateReasonCodeDeleting, ec2.VpcPeeringConnectionStateReasonCodeDeleted:
		logger.Infof("vpc peering is in state %s, assuming deletion in progress, skipping", aws.StringValue(toDelete.Status.Code))
		return nil
	}
	_, err = n.Ec2Api.DeleteVpcPeeringConnection(&ec2.DeleteVpcPeeringConnectionInput{
		VpcPeeringConnectionId: toDelete.VpcPeeringConnectionId,
	})
	if err != nil {
		return errorUtil.Wrap(err, "failed to delete vpc peering connection")
	}
	return nil
}

//IsEnabled returns true when no bundled subnets are found in the openshift cluster vpc.
//
//All subnets created by the cloud resource operator are identified by having a tag with the name `<organizationTag>/clusterID`.
//By default, `integreatly.org/clusterID`.
//
//this check allows us to maintain backwards compatibility with openshift clusters that used the cloud resource operator before this standalone vpc provider was added.
//If this function returns false, we should continue using the backwards compatible approach of bundling resources in with the openshift cluster vpc.
func (n *NetworkProvider) IsEnabled(ctx context.Context) (bool, error) {
	logger := n.Logger.WithField("action", "isEnabled")

	//check if there is a standalone vpc already created.
	foundVpc, err := getClusterVpc(ctx, n.Client, n.Ec2Api, logger)
	if err != nil {
		return false, errorUtil.Wrap(err, "unable to get vpc")
	}

	clusterID, err := resources.GetClusterID(ctx, n.Client)
	if err != nil {
		return false, errorUtil.Wrap(err, "unable to get cluster id")
	}

	// returning subnets from cluster vpc
	logger.Info("getting cluster vpc subnets")
	vpcSubnets, err := GetVPCSubnets(n.Ec2Api, logger, foundVpc)
	if err != nil {
		return false, errorUtil.Wrap(err, "error happened while returning vpc subnets")
	}

	// iterate all cluster vpc's checking for valid bundled vpc subnets
	n.Logger.Infof("checking cluster vpc subnets for cluster id tag, %s", clusterID)
	organizationTag := resources.GetOrganizationTag()
	var validBundledVPCSubnets []*ec2.Subnet
	for _, subnet := range vpcSubnets {
		for _, tag := range subnet.Tags {
			if aws.StringValue(tag.Key) == fmt.Sprintf("%sclusterID", organizationTag) &&
				aws.StringValue(tag.Value) == clusterID {
				validBundledVPCSubnets = append(validBundledVPCSubnets, subnet)
				logger.Infof("found bundled vpc subnet %s in cluster vpc %s", *subnet.SubnetId, *subnet.VpcId)
			}
		}
	}
	logger.Infof("found %d bundled vpc subnets in cluster vpc", len(validBundledVPCSubnets))
	return len(validBundledVPCSubnets) == 0, nil
}

// DeleteBundledCloudResources returns an error on any error deleting of the following resources
// * elasticache subnet group
// * rds subnet group
// * ec2 security group
//
// it has been located under the cluster network provider as it requires 3 different aws sessions
// (elasticache, rds and ec2) to delete the required resources even though it deals with bundled
// resources. The majority of the functionality in this file relates to standalone aws vpc and it's
// resources.
func (n *NetworkProvider) DeleteBundledCloudResources(ctx context.Context) error {
	logger := n.Logger.WithField("action", "deleteBundledCloudResources")

	subnetGroupName, err := BuildInfraName(ctx, n.Client, "subnetgroup", DefaultAwsIdentifierLength)
	if err != nil {
		return errorUtil.Wrap(err, "error building bundle subnet group resource name on deletion")
	}
	logger.Infof("deleting bundled elasticache subnet group %s, if it's not found it's already deleted and will continue", subnetGroupName)
	_, err = n.ElasticacheApi.DeleteCacheSubnetGroup(&elasticache.DeleteCacheSubnetGroupInput{
		CacheSubnetGroupName: aws.String(subnetGroupName),
	})
	cacheSubnetErr, isAwsErr := err.(awserr.Error)
	if err != nil && isAwsErr && cacheSubnetErr.Code() != elasticache.ErrCodeCacheSubnetGroupNotFoundFault {
		return errorUtil.Wrap(err, "error deleting elasticache subnet group")
	}
	logger.Infof("deleting bundled rds subnet group %s, if it's not found it's already deleted and will continue", subnetGroupName)
	_, err = n.RdsApi.DeleteDBSubnetGroup(&rds.DeleteDBSubnetGroupInput{
		DBSubnetGroupName: aws.String(subnetGroupName),
	})
	dbSubnetErr, isAwsErr := err.(awserr.Error)
	if err != nil && isAwsErr && dbSubnetErr.Code() != rds.ErrCodeDBSubnetGroupNotFoundFault {
		return errorUtil.Wrap(err, "error deleting rds subnet group")
	}

	securityGroupName, err := BuildInfraName(ctx, n.Client, "securitygroup", DefaultAwsIdentifierLength)
	if err != nil {
		return errorUtil.Wrap(err, "error building bundle security group resource name on deletion")
	}
	logger.Infof("Deleting bundled ec2 security group %s, if it's not found it's already deleted and will continue", securityGroupName)
	// in the case of the security group the Group Id is required in order to delete security groups
	// not connected with the default vpc. In order to delete it it is required to describe them
	// all in the account and then find the one with the correct group name and then request deletion
	// using the group id of the matched security group
	securityGroup, err := getSecurityGroup(n.Ec2Api, securityGroupName)
	if err != nil {
		return errorUtil.Wrap(err, "error getting ec2 security group")
	}
	if securityGroup == nil {
		return nil
	}
	vpc, err := getClusterVpc(ctx, n.Client, n.Ec2Api, logger)
	if err != nil {
		return errorUtil.Wrap(err, "error getting cluster vpc")
	}
	if securityGroup.VpcId != nil && vpc.VpcId != nil && aws.StringValue(securityGroup.VpcId) == aws.StringValue(vpc.VpcId) {
		if _, err = n.Ec2Api.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
			GroupId: securityGroup.GroupId,
		}); err != nil {
			return errorUtil.Wrap(err, "error deleting bundled security group")
		}
	}
	return nil
}

func (n *NetworkProvider) getNetworkPeering(ctx context.Context, network *Network) (*ec2.VpcPeeringConnection, error) {
	logger := resources.NewActionLogger(n.Logger, "getNetworkPeering")
	// we will always peer with the openshift/kubernetes cluster vpc that this operator is running on
	logger.Info("getting cluster vpc")
	clusterVpc, err := getClusterVpc(ctx, n.Client, n.Ec2Api, logger)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to get cluster vpc")
	}
	logger.Infof("found cluster vpc %s", aws.StringValue(clusterVpc.VpcId))

	// the peering connection will either be found or created below
	var peeringConnection *ec2.VpcPeeringConnection

	// check if a peering connection already exists between the two networks
	logger.Info("checking for an existing peering connection")
	describeVpcPeerOutput, err := n.Ec2Api.DescribeVpcPeeringConnections(&ec2.DescribeVpcPeeringConnectionsInput{
		DryRun: nil,
		Filters: []*ec2.Filter{
			{
				Name:   aws.String(filterVpcPeeringRequesterId),
				Values: []*string{network.Vpc.VpcId},
			},
			{
				Name:   aws.String(filterVpcPeeringAccepterId),
				Values: []*string{clusterVpc.VpcId},
			},
		},
	})
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to describe peering connections")
	}
	if len(describeVpcPeerOutput.VpcPeeringConnections) > 0 {
		// vpc peering connections exist between the two vpcs, now find one in a healthy state
		for peeringConnIdx, peeringConn := range describeVpcPeerOutput.VpcPeeringConnections {
			// deleted peering connections can stay around for quite a while, ignore them
			if aws.StringValue(peeringConn.Status.Code) != ec2.VpcPeeringConnectionStateReasonCodeDeleted &&
				aws.StringValue(peeringConn.Status.Code) != ec2.VpcPeeringConnectionStateReasonCodeDeleting {
				peeringConnection = describeVpcPeerOutput.VpcPeeringConnections[peeringConnIdx]
				logger.Infof("existing vpc peering connection found %s", aws.StringValue(peeringConnection.VpcPeeringConnectionId))
				break
			}
		}
	}
	return peeringConnection, nil
}

// reconcileStandaloneSecurityGroup reconciles the standalone security group, ensuring correct tags and ip permissions
// we require every resource (rds/elasticache) provisioned by cro in the cro standalone vpc to have a security group
// this security group should allow all ingress traffic from the cluster
// as the cluster vpc and the standalone vpc are peered we need to use the cluster cidr block as an ip permission to allow ingress traffic
// see -> https://docs.aws.amazon.com/vpc/latest/peering/vpc-peering-security-groups.html
func (n *NetworkProvider) reconcileStandaloneSecurityGroup(ctx context.Context, logger *logrus.Entry) (*ec2.SecurityGroup, error) {
	// build security group name
	standaloneSecurityGroupName, err := BuildInfraName(ctx, n.Client, defaultSecurityGroupPostfix, DefaultAwsIdentifierLength)
	logger.Info(fmt.Sprintf("setting resource security group %s", standaloneSecurityGroupName))
	if err != nil {
		return nil, errorUtil.Wrap(err, "error building subnet group name")
	}

	// get standalone security group
	standaloneSecGroup, err := getSecurityGroup(n.Ec2Api, standaloneSecurityGroupName)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to find standalone security group")
	}

	// get the cro standalone vpc
	standaloneVpc, err := getStandaloneVpc(ctx, n.Client, n.Ec2Api, logger)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to get standalone vpc")
	}
	if standaloneVpc == nil {
		return nil, errorUtil.New("standalone vpc can not be nil")
	}

	// get the cluster bundled vpc
	clusterVpc, err := getClusterVpc(ctx, n.Client, n.Ec2Api, logger)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to get cluster vpc")
	}

	// if no security group exists in standalone vpc create it
	if standaloneSecGroup == nil {
		// create security group
		logger.Infof("creating security group for standalone vpc")
		createdSecurityGroupOutput, err := n.Ec2Api.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
			Description: aws.String("rhmi cro security group for cro standalone vpc"),
			GroupName:   aws.String(standaloneSecurityGroupName),
			VpcId:       standaloneVpc.VpcId,
		})
		if err != nil {
			return nil, errorUtil.Wrap(err, "error creating security group")
		}
		// get created security group as we expect it to exist before beginning to provision a resource
		secGroup, err := n.Ec2Api.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
			GroupIds: []*string{
				createdSecurityGroupOutput.GroupId,
			},
		})
		if err != nil {
			return nil, errorUtil.Wrap(err, "error getting created security group")
		}
		// if the security group does not exist after creation we should error here before continuing with reconcile of resource
		if len(secGroup.SecurityGroups) == 0 {
			return nil, errorUtil.New(fmt.Sprintf("expected to find created security group %s", standaloneSecurityGroupName))
		}
		logger.Infof("created security group %s", aws.StringValue(secGroup.SecurityGroups[0].GroupName))
		// if the security group has created successfully, set the standaloneSecGroup to newly created group
		// this is because we require the correct tags and permissions to be added
		standaloneSecGroup = secGroup.SecurityGroups[0]
	}
	logger.Infof("found security group %s", *standaloneSecGroup.GroupId)

	// ensure standalone vpc has correct tags
	// we require the subnet group to be tagged with the cro owner tag
	croOwnerTag, err := getCloudResourceOperatorOwnerTag(ctx, n.Client)
	if err != nil {
		return nil, errorUtil.Wrap(err, "error generating cloud resource owner tag")
	}
	securityGroupTags := ec2TagsToGeneric(standaloneSecGroup.Tags)
	if !tagsContains(securityGroupTags, tagDisplayName, DefaultRHMISecurityGroupNameTagValue) ||
		!tagsContains(securityGroupTags, croOwnerTag.key, croOwnerTag.value) {
		logger.Infof("tagging security group %s", aws.StringValue(standaloneSecGroup.GroupId))
		_, err := n.Ec2Api.CreateTags(&ec2.CreateTagsInput{
			Resources: []*string{
				standaloneSecGroup.GroupId,
			},
			Tags: []*ec2.Tag{
				{
					Key:   aws.String(tagDisplayName),
					Value: aws.String(DefaultRHMISecurityGroupNameTagValue),
				}, genericToEc2Tag(croOwnerTag),
			},
		})
		if err != nil {
			return nil, errorUtil.Wrap(err, "unable to tag security group")
		}
		logger.Infof("successfully tagged security group: %s for vpcid: %s", *standaloneSecGroup.GroupId, *standaloneSecGroup.VpcId)
	}

	// we need to ensure permissions for standalone security group will accept traffic from the cluster vpc
	// currently we can not use the cluster vpc security group, this is a limitation from aws
	// see for more -> https://docs.aws.amazon.com/vpc/latest/peering/vpc-peering-security-groups.html
	// it is recommended by aws docs to use the cidr block from the peered vpc

	// build ip permission
	ipPermission := &ec2.IpPermission{
		IpProtocol: aws.String("-1"),
		IpRanges: []*ec2.IpRange{
			{
				CidrIp: clusterVpc.CidrBlock,
			},
		},
	}

	// ensure ip permission correct and valid in the standalone security group
	for _, perm := range standaloneSecGroup.IpPermissions {
		if reflect.DeepEqual(perm, ipPermission) {
			logger.Infof("ip permissions are correct for security group %s", *standaloneSecGroup.GroupName)
			return standaloneSecGroup, nil
		}
	}

	// authorize the seucrity group ingres if it is not as expected
	if _, err := n.Ec2Api.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(*standaloneSecGroup.GroupId),
		IpPermissions: []*ec2.IpPermission{
			ipPermission,
		},
	}); err != nil {
		return nil, errorUtil.Wrap(err, "error authorizing security group ingress")
	}
	logger.Infof("ip permissions have been updated to expected permissions for security group %s", *standaloneSecGroup.GroupName)
	return standaloneSecGroup, nil
}

// reconcileStandaloneRouteTableTags adds cro owner tag on standalone route table
// we require owner tag for easy identification and filtering of route tables
func (n *NetworkProvider) reconcileStandaloneRouteTableTags(ctx context.Context, vpc *ec2.Vpc, logger *logrus.Entry) error {
	logger.Infof("checking vpc %s route table has correct tags", aws.StringValue(vpc.VpcId))

	routeTableOutput, err := n.Ec2Api.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{vpc.VpcId},
			},
		},
	})
	if err != nil {
		return errorUtil.Wrap(err, "unexpected error on getting route tables")
	}

	routeTables := routeTableOutput.RouteTables
	if len(routeTables) == 0 {
		return errorUtil.New(fmt.Sprint("did not find any route associated with vpc %", vpc.VpcId))
	}

	croOwnerTag, err := getCloudResourceOperatorOwnerTag(ctx, n.Client)
	if err != nil {
		return errorUtil.Wrap(err, "error generating cloud resource owner tag")
	}

	for _, routeTable := range routeTables {
		routeTableTags := ec2TagsToGeneric(routeTable.Tags)
		if !tagsContains(routeTableTags, tagDisplayName, DefaultRHMIVpcRouteTableNameTagValue) ||
			!tagsContains(routeTableTags, croOwnerTag.key, croOwnerTag.value) {
			_, err := n.Ec2Api.CreateTags(&ec2.CreateTagsInput{
				Resources: []*string{
					aws.String(*routeTable.RouteTableId),
				},
				Tags: []*ec2.Tag{
					{
						Key:   aws.String(tagDisplayName),
						Value: aws.String(DefaultRHMIVpcRouteTableNameTagValue),
					}, genericToEc2Tag(croOwnerTag),
				},
			})
			if err != nil {
				return errorUtil.Wrap(err, "unable to tag route table")
			}
			logger.Infof("successfully tagged route table: %s for clusterID: %s", aws.StringValue(routeTable.RouteTableId), croOwnerTag.value)
		}
	}
	return nil
}

//reconcileStandaloneVPCSubnets returns an array list of private subnets associated with a vpc or an error
//
//each standalone vpc cidr block is split in half, to create two private subnets.
//these subnets are located in different az's
//the az is determined by the cro strategy, either provided by override config map or provided by the infrastructure CR
func (n *NetworkProvider) reconcileStandaloneVPCSubnets(ctx context.Context, logger *logrus.Entry, vpc *ec2.Vpc) ([]*ec2.Subnet, error) {
	logger.Info("gathering all private subnets in cluster vpc")

	// build our subnets, so we know if the vpc /26 then we /27
	if *vpc.CidrBlock == "" {
		return nil, errorUtil.New("standalone vpc cidr block can't be empty")
	}

	// AWS stores it's CIDR block as a string, convert it
	_, awsCIDR, err := net.ParseCIDR(*vpc.CidrBlock)
	if err != nil {
		return nil, errorUtil.Wrapf(err, "failed to parse vpc cidr block %s", *vpc.CidrBlock)
	}
	// Get the cluster VPC mask size
	// e.g. If the cluster VPC CIDR block is 10.0.0.0/8, the size is 8 (8 bits)
	maskSize, _ := awsCIDR.Mask.Size()

	// If the VPC CIDR mask size is greater or equal to the size that CRO requires
	// - If equal, CRO will not be able to subdivide the VPC CIDR into sub-networks
	// - If greater, there will be fewer host addresses available in the sub-networks than CRO needs
	// Note: The larger the mask size, the less hosts the network can support
	if maskSize >= defaultSubnetMask {
		return nil, errorUtil.New(fmt.Sprintf("vpc cidr block %s cannot contain generated subnet mask /%d", *vpc.CidrBlock, defaultSubnetMask))
	}

	// Split vpc cidr mask by increasing mask by 1
	halfMaskStr := fmt.Sprintf("%s/%d", awsCIDR.IP.String(), maskSize+1)
	_, halfMaskCidr, err := net.ParseCIDR(halfMaskStr)
	if err != nil {
		return nil, errorUtil.Wrapf(err, "failed to parse half mask cidr block %s", halfMaskStr)
	}

	// Generate 2 valid sub-networks that can be used in the cluster VPC CIDR range
	validSubnets := generateAvailableSubnets(awsCIDR, halfMaskCidr)
	if len(validSubnets) != defaultNumberOfExpectedSubnets {
		return nil, errorUtil.New(fmt.Sprintf("expected at least two subnet ranges, found %s", validSubnets))
	}

	// get a list of valid availability zones
	var validAzs []*ec2.AvailabilityZone
	azs, err := n.Ec2Api.DescribeAvailabilityZones(&ec2.DescribeAvailabilityZonesInput{})
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting availability zones")
	}

	// get availability zones that support the current default rds and elasticache instance sizes.
	describeInstanceTypeOfferingsOutput, err := n.Ec2Api.DescribeInstanceTypeOfferings(&ec2.DescribeInstanceTypeOfferingsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance-type"),
				Values: aws.StringSlice([]string{strings.Replace(defaultCacheNodeType, "cache.", "", 1), strings.Replace(defaultAwsDBInstanceClass, "db.", "", 1)}),
			},
		},
		LocationType: aws.String(ec2.LocationTypeAvailabilityZone),
	})
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to list ")
	}

	// filter the availability zones to only include ones that support the default instance types.
	// ensure if any duplicate regions are returned, they are removed.
	var supportedAzs []*ec2.AvailabilityZone
	for _, az := range azs.AvailabilityZones {
		foundAz := false
		for _, instanceTypeOffering := range describeInstanceTypeOfferingsOutput.InstanceTypeOfferings {
			if aws.StringValue(instanceTypeOffering.Location) == aws.StringValue(az.ZoneName) {
				foundAz = true
				break
			}
		}
		if foundAz {
			supportedAzs = append(supportedAzs, az)
		}
	}

	// sort the azs first
	sort.Sort(azByZoneName(supportedAzs))

	for index, az := range supportedAzs {
		validAzs = append(validAzs, az)
		if index == 1 {
			break
		}
	}
	if len(validAzs) != defaultNumberOfExpectedSubnets {
		return nil, errorUtil.New(fmt.Sprintf("expected 2 availability zones, found %s", validAzs))
	}

	// validSubnets and validAzs contain the same index (2 items)
	// to mitigate the chance of a nil pointer during subnet creation,
	// both azs and subnets are mapped to type `NetworkAZSubnet`
	var expectedAZSubnets []*NetworkAZSubnet
	for subnetIndex, subnet := range validSubnets {
		for azIndex, az := range validAzs {
			if azIndex == subnetIndex {
				azSubnet := &NetworkAZSubnet{
					IP: subnet,
					AZ: az,
				}
				expectedAZSubnets = append(expectedAZSubnets, azSubnet)
			}
		}
	}

	// check expected subnets exist in expect az
	// filter based on a tag key attached to private subnets
	// get subnets in vpc
	subs, err := getVPCAssociatedSubnets(n.Ec2Api, logger, vpc)
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting vpc subnets")
	}
	// for create a subnet for every expected subnet to exist
	for _, expectedAZSubnet := range expectedAZSubnets {
		if !subnetExists(subs, expectedAZSubnet.IP.String()) {
			zoneName := expectedAZSubnet.AZ.ZoneName
			logger.Infof("attempting to create subnet with cidr block %s for vpc %s in zone %s", expectedAZSubnet.IP.String(), *vpc.VpcId, *zoneName)
			createOutput, err := n.Ec2Api.CreateSubnet(&ec2.CreateSubnetInput{
				AvailabilityZone: aws.String(*zoneName),
				CidrBlock:        aws.String(expectedAZSubnet.IP.String()),
				VpcId:            aws.String(*vpc.VpcId),
			})
			ec2err, isAwsErr := err.(awserr.Error)
			if err != nil && isAwsErr && ec2err.Code() == "InvalidSubnet.Conflict" {
				// if two or more crs are created at the same time the network provider may run in parallel
				// in this case it's expected that there will be a conflict, as they will each be reconciling the required subnets
				// one will get in first and the following ones will see the expected conflict as the subnet is already created
				logger.Debugf("%s conflicts with a current subnet", expectedAZSubnet.IP.String())
			}
			if err != nil {
				return nil, errorUtil.Wrap(err, "error creating new subnet")
			}
			if newErr := tagPrivateSubnet(ctx, n.Client, n.Ec2Api, createOutput.Subnet, logger); newErr != nil {
				return nil, newErr
			}

			subs = append(subs, createOutput.Subnet)
			logger.Infof("created new subnet %s in %s", expectedAZSubnet.IP.String(), *vpc.VpcId)
		}
	}

	croClusterOwnerTag, err := getCloudResourceOperatorOwnerTag(ctx, n.Client)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to build cro owner tag")
	}

	// ensure subnets have the correct tags
	for _, sub := range subs {
		logger.Infof("validating subnet %s", *sub.SubnetId)
		if !tagsContains(ec2TagsToGeneric(sub.Tags), defaultAWSPrivateSubnetTagKey, "1") ||
			!tagsContains(ec2TagsToGeneric(sub.Tags), croClusterOwnerTag.key, croClusterOwnerTag.value) ||
			!tagsContains(ec2TagsToGeneric(sub.Tags), "Name", DefaultRHMISubnetNameTagValue) {
			if err := tagPrivateSubnet(ctx, n.Client, n.Ec2Api, sub, logger); err != nil {
				return nil, errorUtil.Wrap(err, "failed to tag subnet")
			}
		}
	}

	return subs, nil
}

// getVPCRouteTable will return a list of route tables based on a route table tag
// we expect there to be route tables, if none are found we return an error
func (n *NetworkProvider) getVPCRouteTable(routeTableTag *ec2.Tag) ([]*ec2.RouteTable, error) {
	routeTables, err := n.Ec2Api.DescribeRouteTables(&ec2.DescribeRouteTablesInput{})
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to get route tables")
	}

	var foundRouteTables []*ec2.RouteTable
	for _, routeTable := range routeTables.RouteTables {
		routeTableTags := ec2TagsToGeneric(routeTable.Tags)
		if tagsContains(routeTableTags, aws.StringValue(routeTableTag.Key), aws.StringValue(routeTableTag.Value)) {
			foundRouteTables = append(foundRouteTables, routeTable)
		}
	}

	if len(foundRouteTables) == 0 {
		return nil, errorUtil.New(fmt.Sprintf("could not find any route table with the tag key: %s and value: %s", aws.StringValue(routeTableTag.Key), aws.StringValue(routeTableTag.Value)))
	}
	return foundRouteTables, nil
}

//reconcileVPCTags will tag a VPC or return an error
//
//VPCs are tagged with with the name `<organizationTag>/clusterID`.
//By default, `integreatly.org/clusterID`.
func (n *NetworkProvider) reconcileVPCTags(vpc *ec2.Vpc, clusterIDTag *tag) error {
	logger := n.Logger.WithField("action", "reconcileVPCTags")

	vpcTags := ec2TagsToGeneric(vpc.Tags)
	if !tagsContains(vpcTags, tagDisplayName, DefaultRHMIVpcNameTagValue) ||
		!tagsContains(vpcTags, clusterIDTag.key, clusterIDTag.value) {

		_, err := n.Ec2Api.CreateTags(&ec2.CreateTagsInput{
			Resources: []*string{
				aws.String(*vpc.VpcId),
			},
			Tags: []*ec2.Tag{
				{
					Key:   aws.String(tagDisplayName),
					Value: aws.String(DefaultRHMIVpcNameTagValue),
				}, genericToEc2Tag(clusterIDTag),
			},
		})
		if err != nil {
			return errorUtil.Wrapf(err, "unable to tag vpc %s with state %s", *vpc.VpcId, *vpc.State)
		}
		logger.Infof("successfully tagged vpc: %s for clusterID: %s with state %s", *vpc.VpcId, clusterIDTag.value, *vpc.State)
	}
	return nil
}

//an rds subnet group is required to be in place when provisioning rds resources
//
//reconcileRDSVpcConfiguration ensures that an rds subnet group is created with 2 private subnets
func (n *NetworkProvider) reconcileRDSVpcConfiguration(ctx context.Context, privateVPCSubnets []*ec2.Subnet) error {
	logger := n.Logger.WithField("action", "reconcileRDSVpcConfiguration")
	logger.Info("ensuring rds subnet groups in vpc are as expected")
	// get subnet group id
	subnetGroupName, err := BuildInfraName(ctx, n.Client, defaultSubnetPostfix, DefaultAwsIdentifierLength)
	if err != nil {
		return errorUtil.Wrap(err, "error building subnet group name")
	}

	// build array list of all vpc private subnets
	var subnetIds []*string
	for _, subnet := range privateVPCSubnets {
		subnetIds = append(subnetIds, subnet.SubnetId)
	}

	// in the case of no private subnets being found, we return a less verbose error message compared to obscure aws error message
	if len(privateVPCSubnets) == 0 {
		return errorUtil.New("no private subnets found, can not create subnet group for rds")
	}

	croOwnerTag, err := getCloudResourceOperatorOwnerTag(ctx, n.Client)
	if err != nil {
		return errorUtil.Wrap(err, "failed to build cro owner tag")
	}

	foundSubnetGroup, err := getRDSSubnetGroup(n.RdsApi, subnetGroupName)
	if err != nil {
		return errorUtil.Wrap(err, "failed getting rds subnet group")
	}
	if foundSubnetGroup != nil {
		logger.Infof("subnet group %s found, verifying it is in the expected state", *foundSubnetGroup.DBSubnetGroupName)

		// ensure all subnets exist in subnet group
		subnetExists := true
		for _, subnet := range foundSubnetGroup.Subnets {
			if !contains(subnetIds, subnet.SubnetIdentifier) {
				subnetExists = false
				break
			}
		}

		if !subnetExists || aws.StringValue(foundSubnetGroup.DBSubnetGroupDescription) != defaultSubnetGroupDesc {
			logger.Info("rds subnet group not as expected, updating.")
			if _, err := n.RdsApi.ModifyDBSubnetGroup(&rds.ModifyDBSubnetGroupInput{
				DBSubnetGroupDescription: aws.String(defaultSubnetGroupDesc),
				DBSubnetGroupName:        foundSubnetGroup.DBSubnetGroupName,
				SubnetIds:                subnetIds,
			}); err != nil {
				return errorUtil.Wrap(err, "error updating db subnet group description")
			}
		}

		// get tags for rds subnet group
		tags, err := n.RdsApi.ListTagsForResource(&rds.ListTagsForResourceInput{
			ResourceName: foundSubnetGroup.DBSubnetGroupArn,
		})
		if err != nil {
			return errorUtil.Wrap(err, "error getting subnet group tags")
		}

		// ensure tags exist on rds subnet group
		subnetTags := rdsTagstoGeneric(tags.TagList)
		if !tagsContains(subnetTags, croOwnerTag.key, croOwnerTag.value) {
			err := n.updateRdsSubnetGroupTags(foundSubnetGroup, genericToRdsTag(croOwnerTag))
			if err != nil {
				return errorUtil.Wrap(err, "error updating subnet group tags")
			}
		}
		return nil
	}

	// build subnet group input
	subnetGroupInput := &rds.CreateDBSubnetGroupInput{
		DBSubnetGroupDescription: aws.String(defaultSubnetGroupDesc),
		DBSubnetGroupName:        aws.String(subnetGroupName),
		SubnetIds:                subnetIds,
		Tags: []*rds.Tag{
			genericToRdsTag(croOwnerTag),
		},
	}

	// create db subnet group
	logger.Infof("creating resource subnet group %s", *subnetGroupInput.DBSubnetGroupName)
	if _, err := n.RdsApi.CreateDBSubnetGroup(subnetGroupInput); err != nil {
		return errorUtil.Wrap(err, "unable to create db subnet group")
	}
	return nil
}

//this function removes tags and reads them to an rds subnet group
func (n *NetworkProvider) updateRdsSubnetGroupTags(foundSubnet *rds.DBSubnetGroup, croOwnerTag *rds.Tag) error {
	_, err := n.RdsApi.RemoveTagsFromResource(&rds.RemoveTagsFromResourceInput{
		ResourceName: foundSubnet.DBSubnetGroupArn,
		TagKeys: []*string{
			croOwnerTag.Key,
		},
	})
	if err != nil {
		return errorUtil.Wrap(err, "error updating db subnet group tags")
	}
	_, err = n.RdsApi.AddTagsToResource(&rds.AddTagsToResourceInput{
		ResourceName: foundSubnet.DBSubnetGroupArn,
		Tags: []*rds.Tag{
			croOwnerTag,
		},
	})
	if err != nil {
		return errorUtil.Wrap(err, "error updating db subnet group tags")
	}
	return nil
}

//It is required to have an elasticache subnet group in place when provisioning elasticache resources
//
//reconcileElasticacheVPCConfiguration ensures that an elasticache subnet group is created with 2 private subnets
func (n *NetworkProvider) reconcileElasticacheVPCConfiguration(ctx context.Context, privateVPCSubnets []*ec2.Subnet) error {
	logger := n.Logger.WithField("action", "reconcileElasticacheVPCConfiguration")
	logger.Info("ensuring elasticache subnet groups in vpc are as expected")
	// get subnet group id
	subnetGroupName, err := BuildInfraName(ctx, n.Client, defaultSubnetPostfix, DefaultAwsIdentifierLength)
	if err != nil {
		return errorUtil.Wrap(err, "error building subnet group name")
	}

	// build array list of all vpc private subnets
	var subnetIDs []*string
	for _, subnet := range privateVPCSubnets {
		subnetIDs = append(subnetIDs, subnet.SubnetId)
	}

	// check if group exists
	foundSubnetGroup, err := getElasticacheSubnetByGroup(n.ElasticacheApi, subnetGroupName)
	if err != nil {
		return errorUtil.Wrap(err, "error getting elasticache subnet group on reconcile")
	}

	if foundSubnetGroup != nil {
		logger.Infof("subnet group %s found, verifying it is in the expected state", *foundSubnetGroup.CacheSubnetGroupName)

		// ensure all subnets exist in subnet group
		subnetExists := true
		for _, subnet := range foundSubnetGroup.Subnets {
			if !contains(subnetIDs, subnet.SubnetIdentifier) {
				subnetExists = false
				break
			}
		}

		if !subnetExists || aws.StringValue(foundSubnetGroup.CacheSubnetGroupDescription) != defaultSubnetGroupDesc {
			logger.Infof("elasticache subnet group not as expected, updating.")
			if _, err = n.ElasticacheApi.ModifyCacheSubnetGroup(&elasticache.ModifyCacheSubnetGroupInput{
				CacheSubnetGroupDescription: aws.String(defaultSubnetGroupDesc),
				CacheSubnetGroupName:        foundSubnetGroup.CacheSubnetGroupName,
				SubnetIds:                   subnetIDs,
			}); err != nil {
				return errorUtil.Wrap(err, "error updating elasticache subnet group description")
			}
		}
		return nil
	}

	// in the case of no private subnets found, a less verbose error message compared to obscure aws error message is returned
	if len(privateVPCSubnets) == 0 {
		return errorUtil.New("no private subnets found, can not create subnet group for rds")
	}

	subnetGroupInput := &elasticache.CreateCacheSubnetGroupInput{
		CacheSubnetGroupDescription: aws.String(defaultSubnetGroupDesc),
		CacheSubnetGroupName:        aws.String(subnetGroupName),
		SubnetIds:                   subnetIDs,
	}

	logger.Infof("creating resource subnet group %s", subnetGroupName)
	if _, err := n.ElasticacheApi.CreateCacheSubnetGroup(subnetGroupInput); err != nil {
		return errorUtil.Wrap(err, "unable to create cache subnet group")
	}
	return nil
}

//getStandaloneVpc will return a vpc type or error
//
//Standalone VPCs are tagged with with the name `<organizationTag>/clusterID`.
//By default, `integreatly.org/clusterID`.
//
//This tag is used to identify a standalone vpc
func getStandaloneVpc(ctx context.Context, client client.Client, ec2Svc ec2iface.EC2API, logger *logrus.Entry) (*ec2.Vpc, error) {
	// get all vpcs
	vpcs, err := ec2Svc.DescribeVpcs(&ec2.DescribeVpcsInput{})
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting vpcs")
	}

	// build cro owner tag for filtering vpcs
	croOwnerTag, err := getCloudResourceOperatorOwnerTag(ctx, client)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to build cro owner tag")
	}

	// find associated vpc to tag
	var foundVPC *ec2.Vpc
	for _, vpc := range vpcs.Vpcs {
		for _, tag := range vpc.Tags {
			if *tag.Key == croOwnerTag.key && *tag.Value == croOwnerTag.value {
				logger.Infof("found vpc: %s", *vpc.VpcId)
				foundVPC = vpc
			}
		}
	}
	return foundVPC, nil
}

/*
getVPCAssociatedSubnets will return a list of subnets or an error

this is used twice, to find all subnets associated with a vpc in order to remove all subnets on deletion
it is also used as a helper function when we filter private associated subnets
*/
func getVPCAssociatedSubnets(ec2Svc ec2iface.EC2API, logger *logrus.Entry, vpc *ec2.Vpc) ([]*ec2.Subnet, error) {
	logger.Info("gathering cluster vpc and subnet information")
	// poll subnets to ensure credentials have reconciled
	subs, err := getSubnets(ec2Svc)
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting subnets")
	}

	// in the rare chance no vpc is found we should return an error to avoid an unexpected nil pointer
	if vpc == nil {
		return nil, errorUtil.Wrap(err, "vpc is nil, need vpc to find associated subnets")
	}

	// find associated subnets
	var associatedSubs []*ec2.Subnet
	for _, sub := range subs {
		if *sub.VpcId == *vpc.VpcId {
			logger.Infof("found subnet: %s in vpc %s", *sub.SubnetId, *sub.VpcId)
			associatedSubs = append(associatedSubs, sub)
		}
	}
	return associatedSubs, nil
}

// getRDSSubnetGroup returns rds db subnet group by the group name or an error
func getRDSSubnetGroup(rdsApi rdsiface.RDSAPI, subnetGroupName string) (*rds.DBSubnetGroup, error) {
	// check if group exists
	groups, err := rdsApi.DescribeDBSubnetGroups(&rds.DescribeDBSubnetGroupsInput{})
	if err != nil {
		return nil, errorUtil.Wrap(err, "error describing subnet groups")
	}
	for _, sub := range groups.DBSubnetGroups {
		if *sub.DBSubnetGroupName == subnetGroupName {
			return sub, nil
		}
	}
	return nil, nil
}

// getElasticacheSubnetByGroup returns elasticache subnet group by the group name or an error
func getElasticacheSubnetByGroup(elasticacheApi elasticacheiface.ElastiCacheAPI, subnetGroupName string) (*elasticache.CacheSubnetGroup, error) {
	// check if group exists
	groups, err := elasticacheApi.DescribeCacheSubnetGroups(&elasticache.DescribeCacheSubnetGroupsInput{})
	if err != nil {
		return nil, errorUtil.Wrap(err, "error describing subnet groups")
	}
	for _, sub := range groups.CacheSubnetGroups {
		if *sub.CacheSubnetGroupName == subnetGroupName {
			return sub, nil
		}
	}
	return nil, nil
}

// ReconcileNetworkProviderConfig return parsed ipNet cidr block
// a _network resource type strategy, is expected to have the same tier as either postgres or redis resource type
// ie. for a postgres tier X there should be a corresponding _network tier X
//
// the _network strategy config is unmarshalled into a ec2 create vpc input struct
// from the struct the cidr block is parsed to ensure validity
// if there is no entry for cidrblock in the _network block a sensible default which doesn't overlap with the cluster vpc
// if cro is unable to find a valid non-overlapping cidr block it will return an error
func (n *NetworkProvider) ReconcileNetworkProviderConfig(ctx context.Context, configManager ConfigManager, tier string, logger *logrus.Entry) (*net.IPNet, error) {
	logger.Infof("fetching _network strategy config for tier %s", tier)

	stratCfg, err := configManager.ReadStorageStrategy(ctx, providers.NetworkResourceType, tier)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to read _network strategy config")
	}

	vpcCreateConfig := &ec2.CreateVpcInput{}
	if err := json.Unmarshal(stratCfg.CreateStrategy, vpcCreateConfig); err != nil {
		return nil, errorUtil.Wrap(err, "failed to unmarshal aws vpc create config")
	}

	// if the config map is found and the _network block contains an entry, that is returned for use in the network creation
	if vpcCreateConfig.CidrBlock != nil && *vpcCreateConfig.CidrBlock != "" {
		_, vpcCidr, err := net.ParseCIDR(*vpcCreateConfig.CidrBlock)
		if err != nil {
			return nil, errorUtil.Wrap(err, "failed to parse cidr block from _network strategy")
		}
		logger.Infof("found vpc cidr block %s in network strategy tier %s", vpcCidr.String(), tier)
		return vpcCidr, nil
	}

	//if vpcCreateConfig.CidrBlock is nil or an empty string we can go ahead and set a sensible non overlapping default with a mask size of /26
	defaultCIDR, err := n.getNonOverlappingDefaultCIDR(ctx)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to generate default CIDR")
	}

	// a default cidr is generated and updated to the config map, that is returned for use in the network creation
	return defaultCIDR, nil
}

// getNonOverlappingDefaultCIDR returns a non overlapping cidr block based on the OpenShift vpc cidr block
// the default mask is /26
// for other masks the user is required to provide their own via config
func (n *NetworkProvider) getNonOverlappingDefaultCIDR(ctx context.Context) (*net.IPNet, error) {
	clusterVpc, err := getClusterVpc(ctx, n.Client, n.Ec2Api, n.Logger)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to get cluster vpc for cidr block")
	}
	//parse the cidr to an ipnet cidr in order to manipulate the ip
	_, clusterNet, err := net.ParseCIDR(aws.StringValue(clusterVpc.CidrBlock))

	if err != nil {
		return nil, errorUtil.Wrap(err, "error parsing cluster cidr block")
	}

	// this map is used to loop through the available options for a vpc cidr range in aws
	// See aws docs https://docs.aws.amazon.com/vpc/latest/userguide/VPC_Subnets.html#vpc-sizing-ipv4
	cidrRanges := map[string]string{
		"10.255.255.255/8":  fmt.Sprintf("10.0.0.0/%s", defaultCIDRMask),
		"172.31.255.255/12": fmt.Sprintf("172.16.0.0/%s", defaultCIDRMask),
	}
	//getting the network cr called from the cluster
	networkConf := &v12.Network{}
	err = n.Client.Get(ctx, client.ObjectKey{Name: "cluster"}, networkConf)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to get network kind")
	}

	podCIDR, foundPodNet, err := getPodCIDR(networkConf)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to get pod CIDR")
	}

	serviceCIDR, foundServiceNet, err := getServiceCIDR(networkConf)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to get service CIDR")
	}

	// in this loop we loop through the available cidr ranges for vpcs in aws
	// in each range the ip of the cidr block is incremented only in the A and B range
	// the reason for this is that cro allows a maximum mask size of /16
	// the default cidr mask is set by defaultCIDRMask env and is currently 26
	//
	// the current logic checks that the cidr block does not overlap with the cluster machine,
	//  pod and service cidr range
	for cidrRange, potentialDefault := range cidrRanges {
		_, cidrRangeNet, err := net.ParseCIDR(cidrRange)
		if err != nil {
			fmt.Println("error parsing cidr range for default cidr block", err)
		}
		// our potential default
		potentialDefaultIP, _, err := net.ParseCIDR(potentialDefault)
		if err != nil {
			fmt.Println("error parsing potential default cidr range for default cidr block", err)
		}

		// loop for as long as potential default is a valid CIDR in range
		for cidrRangeNet.Contains(potentialDefaultIP) {

			// create CIDR adding the default /26 mask
			_, defaultNet, err := net.ParseCIDR(fmt.Sprintf("%s/26", potentialDefaultIP.String()))
			if err != nil {
				continue
			}
			// increment potential IP
			potentialDefaultIP = incrementIPForDefaultCIDR(potentialDefaultIP)

			if clusterNet.Contains(defaultNet.IP) || defaultNet.Contains(clusterNet.IP) {
				continue
			}

			if foundPodNet {
				if podCIDR.Contains(defaultNet.IP) || defaultNet.Contains(podCIDR.IP) {
					continue
				}

			}

			if foundServiceNet {
				if serviceCIDR.Contains(defaultNet.IP) || defaultNet.Contains(serviceCIDR.IP) {
					continue
				}

			}

			// return the first available option that does not overlap
			return defaultNet, nil
		}
	}

	// if the loop finishes, it means that we have gone through all available cidr blocks in the available ranges in aws
	// return an error that cro was unable to find an option
	return nil, errorUtil.New("could not find a default cidr block")
}

// Makes a call to the cluster to get the pod cidr from the network CR, parses it into a ip notation
//then checks to see if the value is actually being returned in case of a scenario where the cidr is empty in the CR
func getPodCIDR(networkConf *v12.Network) (*net.IPNet, bool, error) {
	var podNet *net.IPNet
	var err error

	for _, entry := range networkConf.Spec.ClusterNetwork {
		_, podNet, err = net.ParseCIDR(entry.CIDR)
		if err != nil {
			return nil, false, errorUtil.Wrap(err, "error parsing pod cidr")
		}
	}
	if podNet == nil {
		return nil, false, err
	}
	return podNet, true, nil
}

// Makes a call to the cluster to get the service cidr from the network CR, parses it into a ip notation
//then checks to see if the value is actually being returned in case of a scenario where the cidr is empty in the CR
func getServiceCIDR(networkConf *v12.Network) (*net.IPNet, bool, error) {
	var err error
	var serviceNet *net.IPNet

	for _, entry := range networkConf.Spec.ServiceNetwork {

		_, serviceNet, err = net.ParseCIDR(entry)
		if err != nil {
			return nil, false, errorUtil.Wrap(err, "error parsing service cidr")
		}
	}
	if serviceNet == nil {
		return nil, false, err
	}

	return serviceNet, true, nil
}

//subnetExists is a helper function for checking if a subnet exists with a specific cidr block
func subnetExists(subnets []*ec2.Subnet, cidr string) bool {
	for _, subnet := range subnets {
		if *subnet.CidrBlock == cidr {
			return true
		}
	}
	return false
}

//isValidCIDRRange returns a bool denoting if a cidr mask is valid
//
//we accept cidr mask ranges from \16 to \26
func isValidCIDRRange(CIDR *net.IPNet) bool {
	mask, _ := CIDR.Mask.Size()
	return mask > 15 && mask < 27
}

// Increment an IP address only by B and A class.
// as cro offers a maximum mask size of /16
// we reduce the number of increments by skipping the C and D class.
func incrementIPForDefaultCIDR(ip net.IP) net.IP {
	ipv4 := ip.To4()
	if ipv4[1] < 255 {
		ipv4[1] = ipv4[1] + byte(1)
	} else {
		ipv4[1] = byte(0)
		ipv4[0] = ipv4[0] + byte(1)
	}
	return ipv4
}

// validateStandaloneCidrBlock validates the standalone cidr block before creation, returning an error if the cidr is not valid
// checks carried out :
// 		* has a cidr range between \16 and \26
// 		* does not overlap with cluster vpc cidr block
func validateStandaloneCidrBlock(validateCIDR *net.IPNet, clusterCIDR *net.IPNet) error {
	// validate has a cidr range between \16 and \26
	if !isValidCIDRRange(validateCIDR) {
		return errorUtil.New(fmt.Sprintf("%s is out of range, block sizes must be between `/16` and `/26`, please update `_network` strategy", validateCIDR.String()))
	}

	// standalone vpc cidr block can not overlap with existing cluster vpc cidr block
	// issue arises when trying to peer both vpcs with invalid vpc error - `overlapping CIDR range`
	// this utility function returns true if either cidr range intersect
	if clusterCIDR.Contains(validateCIDR.IP) || validateCIDR.Contains(clusterCIDR.IP) {
		return errorUtil.New(fmt.Sprintf("standalone vpc creation failed: standalone cidr block %s overlaps with cluster vpc cidr block %s, update _network strategy to continue vpc creation", validateCIDR.String(), clusterCIDR.IP))
	}
	return nil
}

// all network provider resources are to be tagged with a cloud resource operator owner tag
// denoted -> `integreatly.org/clusterID=<infrastructure-id>`
// this utility function returns a build owner tag
func getCloudResourceOperatorOwnerTag(ctx context.Context, client client.Client) (*tag, error) {
	clusterID, err := resources.GetClusterID(ctx, client)
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting clusterID")
	}
	organizationTag := resources.GetOrganizationTag()
	genericTag := ec2TagToGeneric(&ec2.Tag{
		Key:   aws.String(fmt.Sprintf("%sclusterID", organizationTag)),
		Value: aws.String(clusterID),
	})
	return genericTag, nil
}

// resources such as cluster vpc route tables are tagged with a cluster owner tag
// denoted -> ` kubernetes.io/cluster/<cluster-id>=owned
// this utility function builds and returns the tag
func getClusterOwnerTag(ctx context.Context, client client.Client) (*ec2.Tag, error) {
	clusterID, err := resources.GetClusterID(ctx, client)
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting clusterID")
	}

	// a tag for identifying cluster owned vpcs
	return &ec2.Tag{
		Key:   aws.String(fmt.Sprintf("%s%s", clusterOwnedTagKeyPrefix, clusterID)),
		Value: aws.String(clusterOwnedTagValue),
	}, nil
}

// this utilty function verifys if a route already exists in a list of routes
// we require a route setup in both cluster vpc route table and standalone vpc route table
func routeExists(routes []*ec2.Route, checkRoute *ec2.Route) bool {
	for _, route := range routes {
		if route.DestinationCidrBlock == nil || route.VpcPeeringConnectionId == nil {
			continue
		}
		if aws.StringValue(route.DestinationCidrBlock) == aws.StringValue(checkRoute.DestinationCidrBlock) && aws.StringValue(route.VpcPeeringConnectionId) == aws.StringValue(checkRoute.VpcPeeringConnectionId) {
			return true
		}
	}
	return false
}

func contains(strs []*string, str *string) bool {
	for _, s := range strs {
		if aws.StringValue(str) == aws.StringValue(s) {
			return true
		}
	}
	return false
}
