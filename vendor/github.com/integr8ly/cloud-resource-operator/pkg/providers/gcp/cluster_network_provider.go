package gcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"cloud.google.com/go/compute/apiv1/computepb"
	croType "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers/gcp/gcpiface"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	errorUtil "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	"google.golang.org/api/servicenetworking/v1"
	utils "k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultServiceConnectionName   = "servicenetworking-googleapis-com"
	defaultServiceConnectionURI    = "servicenetworking.googleapis.com"
	defaultIpRangePostfix          = "ip-range"
	defaultIpRangeCIDRMask         = 22
	defaultNumberOfExpectedSubnets = 2
	defaultServicesFormat          = "services/%s"
	defaultServiceConnectionFormat = defaultServicesFormat + "/connections/%s"
	defaultNetworksFormat          = "projects/%s/global/networks/%s"
	defaultIpv4Length              = 8 * net.IPv4len
)

//go:generate moq -out cluster_network_provider_moq.go . NetworkManager
type NetworkManager interface {
	CreateNetworkIpRange(context.Context, *net.IPNet) (*computepb.Address, croType.StatusMessage, error)
	CreateNetworkService(context.Context) (*servicenetworking.Connection, croType.StatusMessage, error)
	DeleteNetworkPeering(context.Context) error
	DeleteNetworkService(context.Context) error
	DeleteNetworkIpRange(context.Context) error
	ComponentsExist(context.Context) (bool, error)
	ReconcileNetworkProviderConfig(ctx context.Context, configManager ConfigManager, tier string) (*net.IPNet, error)
}

var (
	_ NetworkManager = (*NetworkProvider)(nil)
)

type NetworkProvider struct {
	Client      client.Client
	NetworkApi  gcpiface.NetworksAPI
	SubnetApi   gcpiface.SubnetsApi
	ServicesApi gcpiface.ServicesAPI
	AddressApi  gcpiface.AddressAPI
	Logger      *logrus.Entry
	ProjectID   string
}

type CreateVpcInput struct {
	CidrBlock string
}

// NewNetworkManager initialises all required clients
func NewNetworkManager(ctx context.Context, projectID string, opt option.ClientOption, client client.Client, logger *logrus.Entry) (*NetworkProvider, error) {
	networksApi, err := gcpiface.NewNetworksAPI(ctx, opt)
	if err != nil {
		return nil, errorUtil.Wrap(err, "Failed to initialise network client")
	}
	subnetsApi, err := gcpiface.NewSubnetsAPI(ctx, opt)
	if err != nil {
		return nil, errorUtil.Wrap(err, "Failed to initialise subnetworks client")
	}
	servicesApi, err := gcpiface.NewServicesAPI(ctx, opt)
	if err != nil {
		return nil, errorUtil.Wrap(err, "Failed to initialise servicenetworking client")
	}
	addressApi, err := gcpiface.NewAddressAPI(ctx, opt)
	if err != nil {
		return nil, errorUtil.Wrap(err, "Failed to initialise addresses client")
	}
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger())
	}
	return &NetworkProvider{
		Client:      client,
		NetworkApi:  networksApi,
		SubnetApi:   subnetsApi,
		ServicesApi: servicesApi,
		AddressApi:  addressApi,
		Logger:      logger.WithField("provider", "gcp_network_provider"),
		ProjectID:   projectID,
	}, nil
}

func (n *NetworkProvider) CreateNetworkIpRange(ctx context.Context, cidrRange *net.IPNet) (*computepb.Address, croType.StatusMessage, error) {
	// build ip address range name
	ipRangeName, err := resources.BuildInfraName(ctx, n.Client, defaultIpRangePostfix, defaultGcpIdentifierLength)
	if err != nil {
		return nil, croType.StatusNetworkCreateError, errorUtil.Wrap(err, "failed to build ip address range infra name")
	}
	address, err := n.getAddressRange(ctx, ipRangeName)
	if err != nil {
		return nil, croType.StatusNetworkCreateError, errorUtil.Wrap(err, "failed to retrieve ip address range")
	}
	// if it does not exist, create it
	if address == nil {
		clusterVpc, err := getClusterVpc(ctx, n.Client, n.NetworkApi, n.ProjectID, n.Logger)
		if err != nil {
			return nil, croType.StatusNetworkCreateError, errorUtil.Wrap(err, "failed to get cluster vpc")
		}
		subnets, err := n.getClusterSubnets(ctx, clusterVpc)
		if err != nil {
			return nil, croType.StatusNetworkCreateError, errorUtil.Wrap(err, "failed to get cluster subnetworks")
		}
		err = validateCidrBlock(cidrRange, subnets)
		if err != nil {
			return nil, croType.StatusNetworkCreateError, errorUtil.Wrap(err, "ip range validation failure")
		}
		err = n.createAddressRange(ctx, clusterVpc, ipRangeName, cidrRange)
		if err != nil {
			return nil, croType.StatusNetworkCreateError, errorUtil.Wrap(err, "failed to create ip address range")
		}

		return nil, croType.StatusNetworkIPRangePendingCreation, nil
	}
	n.Logger.Infof("created ip address range %s: %s/%d", address.GetName(), address.GetAddress(), address.GetPrefixLength())
	return address, "", nil
}

// CreateNetworkService Creates the network service connection and will return the service if it has been created successfully
// This automatically creates a peering connection to the clusterVpc named after the service connection
func (n *NetworkProvider) CreateNetworkService(ctx context.Context) (*servicenetworking.Connection, croType.StatusMessage, error) {
	clusterVpc, err := getClusterVpc(ctx, n.Client, n.NetworkApi, n.ProjectID, n.Logger)
	if err != nil {
		return nil, croType.StatusNetworkCreateError, errorUtil.Wrap(err, "failed to get cluster vpc")
	}
	service, err := n.getServiceConnection(clusterVpc)
	if err != nil {
		return nil, croType.StatusNetworkCreateError, errorUtil.Wrap(err, "failed to retrieve service connection")
	}
	// if it does not exist, create it
	if service == nil {
		// build ip address range name
		ipRange, err := resources.BuildInfraName(ctx, n.Client, defaultIpRangePostfix, defaultGcpIdentifierLength)
		if err != nil {
			return nil, croType.StatusNetworkCreateError, errorUtil.Wrap(err, "failed to build ip address range infra name")
		}
		address, err := n.getAddressRange(ctx, ipRange)
		if err != nil {
			return nil, croType.StatusNetworkCreateError, errorUtil.Wrap(err, "failed to retrieve ip address range")
		}
		// if the ip range is present, and is ready for use
		// possible states for address are RESERVING, RESERVED, IN_USE
		if address == nil || address.GetStatus() == computepb.Address_RESERVING.String() {
			return nil, croType.StatusNetworkIPRangeNotExistOrPendingCreation, nil
		}
		if address != nil && address.GetStatus() == computepb.Address_RESERVED.String() {
			err = n.createServiceConnection(clusterVpc, ipRange)
			if err != nil {
				return nil, croType.StatusNetworkCreateError, errorUtil.Wrap(err, "failed to create service connection")
			}
			return nil, croType.StatusNetworkServiceConnectionPendingCreation, nil
		}
	}
	n.Logger.Infof("created network service connection %s", service)
	return service, "", nil
}

// DeleteNetworkPeering Removes the peering connection from the cluster vpc
// The service connection removal can get stuck if this is not performed first
func (n *NetworkProvider) DeleteNetworkPeering(ctx context.Context) error {
	clusterVpc, err := getClusterVpc(ctx, n.Client, n.NetworkApi, n.ProjectID, n.Logger)
	if err != nil {
		return errorUtil.Wrap(err, "failed to get cluster vpc")
	}
	peering := n.getPeeringConnection(clusterVpc)
	// if it exists, delete it
	if peering != nil {
		err = n.deletePeeringConnection(ctx, clusterVpc)
		if err != nil {
			return errorUtil.Wrap(err, "failed to delete peering connection")
		}
	}
	return nil
}

// DeleteNetworkService This deletes the network service connection, but can get stuck if peering
// has not been removed
func (n *NetworkProvider) DeleteNetworkService(ctx context.Context) error {
	clusterVpc, err := getClusterVpc(ctx, n.Client, n.NetworkApi, n.ProjectID, n.Logger)
	if err != nil {
		return errorUtil.Wrap(err, "failed to get cluster vpc")
	}
	service, err := n.getServiceConnection(clusterVpc)
	if err != nil {
		return err
	}
	// if the service exists, delete it
	if service != nil {
		err := n.deleteServiceConnection(service)
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *NetworkProvider) DeleteNetworkIpRange(ctx context.Context) error {
	// build ip address range name
	ipRange, err := resources.BuildInfraName(ctx, n.Client, defaultIpRangePostfix, defaultGcpIdentifierLength)
	if err != nil {
		return errorUtil.Wrap(err, "failed to build ip address range infra name")
	}
	address, err := n.getAddressRange(ctx, ipRange)
	if err != nil {
		return errorUtil.Wrap(err, "failed to retrieve ip address range")
	}
	// if the address exists, delete it
	if address != nil {
		clusterVpc, err := getClusterVpc(ctx, n.Client, n.NetworkApi, n.ProjectID, n.Logger)
		if err != nil {
			return errorUtil.Wrap(err, "failed to get cluster vpc")
		}
		service, err := n.getServiceConnection(clusterVpc)
		if err != nil {
			return err
		}
		if service != nil && service.ReservedPeeringRanges[0] == address.GetName() {
			return errors.New("failed to delete ip address range, service connection still present")
		}
		err = n.deleteAddressRange(ctx, ipRange)
		if err != nil {
			return errorUtil.Wrap(err, "failed to delete ip address range")
		}
	}
	return nil
}

func (n *NetworkProvider) ComponentsExist(ctx context.Context) (bool, error) {
	clusterVpc, err := getClusterVpc(ctx, n.Client, n.NetworkApi, n.ProjectID, n.Logger)
	if err != nil {
		return false, errorUtil.Wrap(err, "failed to get cluster vpc")
	}
	// build ip address range name
	ipRange, err := resources.BuildInfraName(ctx, n.Client, defaultIpRangePostfix, defaultGcpIdentifierLength)
	if err != nil {
		return false, errorUtil.Wrap(err, "failed to build ip address range infra name")
	}
	address, err := n.getAddressRange(ctx, ipRange)
	if err != nil {
		return false, errorUtil.Wrap(err, "failed to retrieve ip address range")
	}
	if address != nil {
		n.Logger.Infof("ip address range %s deletion in progress", address.GetName())
		return true, nil
	}
	service, err := n.getServiceConnection(clusterVpc)
	if err != nil {
		return false, err
	}
	if service != nil {
		n.Logger.Infof("service connection %s deletion in progress", service.Service)
		return true, nil
	}
	return false, nil
}

func (n *NetworkProvider) getServiceConnection(clusterVpc *computepb.Network) (*servicenetworking.Connection, error) {
	resp, err := n.ServicesApi.ConnectionsList(clusterVpc, n.ProjectID, fmt.Sprintf(defaultServicesFormat, defaultServiceConnectionURI))
	if err != nil {
		return nil, err
	}
	if len(resp.Connections) == 0 {
		return nil, nil
	}
	return resp.Connections[0], nil
}

func (n *NetworkProvider) createServiceConnection(clusterVpc *computepb.Network, ipRange string) error {
	n.Logger.Infof("creating service connection %s", defaultServiceConnectionName)
	_, err := n.ServicesApi.ConnectionsCreate(
		fmt.Sprintf(defaultServicesFormat, defaultServiceConnectionURI),
		&servicenetworking.Connection{
			Network: fmt.Sprintf(defaultNetworksFormat, n.ProjectID, clusterVpc.GetName()),
			ReservedPeeringRanges: []string{
				ipRange,
			},
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func (n *NetworkProvider) deleteServiceConnection(service *servicenetworking.Connection) error {
	n.Logger.Infof("deleting service connection %s", service.Service)
	resp, err := n.ServicesApi.ConnectionsDelete(
		fmt.Sprintf(defaultServiceConnectionFormat, defaultServiceConnectionURI, defaultServiceConnectionName),
		&servicenetworking.DeleteConnectionRequest{
			ConsumerNetwork: service.Network,
		})
	if err != nil {
		return err
	}
	if !resp.Done {
		return errors.New("service connection deletion in progress")
	}
	return nil
}

func (n *NetworkProvider) getAddressRange(ctx context.Context, ipRange string) (*computepb.Address, error) {
	address, err := n.AddressApi.Get(ctx, &computepb.GetGlobalAddressRequest{
		Address: ipRange,
		Project: n.ProjectID,
	})
	if err != nil && !resources.IsNotFoundError(err) {
		return nil, fmt.Errorf("unexpected error getting addresses from gcp: %w", err)
	}
	return address, nil
}

func (n *NetworkProvider) createAddressRange(ctx context.Context, clusterVpc *computepb.Network, name string, cidrRange *net.IPNet) error {
	n.Logger.Infof("creating address %s", name)
	prefixLength, _ := cidrRange.Mask.Size()
	req := &computepb.InsertGlobalAddressRequest{
		Project: n.ProjectID,
		AddressResource: &computepb.Address{
			AddressType:  utils.To(computepb.Address_INTERNAL.String()),
			IpVersion:    utils.To(computepb.Address_IPV4.String()),
			Name:         &name,
			Network:      clusterVpc.SelfLink,
			PrefixLength: utils.To(int32(prefixLength)),
			Purpose:      utils.To(computepb.Address_VPC_PEERING.String()),
		},
	}
	var msg string
	if cidrRange.IP != nil {
		req.AddressResource.Address = utils.To(cidrRange.IP.String())
		msg = fmt.Sprintf("using cidr %s", cidrRange.String())
	} else {
		msg = fmt.Sprintf("using cidr (gcp-generated)/%d", prefixLength)
	}
	n.Logger.Infof(msg)
	return n.AddressApi.Insert(ctx, req)
}

func (n *NetworkProvider) deleteAddressRange(ctx context.Context, ipRange string) error {
	n.Logger.Infof("deleting address %s", ipRange)
	return n.AddressApi.Delete(ctx, &computepb.DeleteGlobalAddressRequest{
		Project: n.ProjectID,
		Address: ipRange,
	})
}

func (n *NetworkProvider) getPeeringConnection(clusterVpc *computepb.Network) *computepb.NetworkPeering {
	peerings := clusterVpc.GetPeerings()
	if peerings == nil {
		return nil
	}
	for _, p := range peerings {
		if p.GetName() == defaultServiceConnectionName {
			peering := p
			return peering
		}
	}
	return nil
}

func (n *NetworkProvider) deletePeeringConnection(ctx context.Context, clusterVpc *computepb.Network) error {
	n.Logger.Infof("deleting peering %s", defaultServiceConnectionName)
	return n.NetworkApi.RemovePeering(ctx, &computepb.RemovePeeringNetworkRequest{
		Project: n.ProjectID,
		Network: clusterVpc.GetName(),
		NetworksRemovePeeringRequestResource: &computepb.NetworksRemovePeeringRequest{
			Name: utils.To(defaultServiceConnectionName),
		},
	})
}

func (n *NetworkProvider) getClusterSubnets(ctx context.Context, clusterVpc *computepb.Network) ([]*computepb.Subnetwork, error) {
	var subnets []*computepb.Subnetwork
	clusterSubnets := clusterVpc.GetSubnetworks()
	for i := range clusterSubnets {
		name, region, err := parseSubnetUrl(clusterSubnets[i])
		if err != nil {
			return nil, err
		}
		subnet, err := n.SubnetApi.Get(ctx, &computepb.GetSubnetworkRequest{
			Project:    n.ProjectID,
			Subnetwork: name,
			Region:     region,
		})
		if err != nil {
			return nil, errorUtil.Wrapf(err, "failed to retrieve cluster subnet %s", subnet)
		}
		subnets = append(subnets, subnet)
	}
	return subnets, nil
}

func (n *NetworkProvider) ReconcileNetworkProviderConfig(ctx context.Context, configManager ConfigManager, tier string) (*net.IPNet, error) {
	n.Logger.Infof("fetching _network strategy config for tier %s", tier)

	stratCfg, err := configManager.ReadStorageStrategy(ctx, providers.NetworkResourceType, tier)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to read _network strategy config")
	}

	vpcCreateConfig := &CreateVpcInput{}
	if err := json.Unmarshal(stratCfg.CreateStrategy, vpcCreateConfig); err != nil {
		return nil, errorUtil.Wrap(err, "failed to unmarshal gcp vpc create config")
	}

	// if the config map is found and the _network block contains an entry, that is returned for use in the network creation
	if vpcCreateConfig.CidrBlock != "" {
		_, vpcCidr, err := net.ParseCIDR(vpcCreateConfig.CidrBlock)
		if err != nil {
			return nil, errorUtil.Wrap(err, "failed to parse cidr block from _network strategy")
		}
		n.Logger.Infof("found vpc cidr block %s in network strategy tier %s", vpcCidr.String(), tier)
		return vpcCidr, nil
	}

	// if vpcCreateConfig.CidrBlock is an empty string we can go ahead and set a default with a mask size of /22
	defaultCidr := &net.IPNet{
		Mask: net.CIDRMask(defaultIpRangeCIDRMask, defaultIpv4Length),
	}

	// a default cidr is generated and updated to the config map, that is returned for use in the network creation
	return defaultCidr, nil
}

func validateCidrBlock(validateCIDR *net.IPNet, subnets []*computepb.Subnetwork) error {
	// validate has a cidr range lower than or equal to /22
	if !isValidCIDRRange(validateCIDR) {
		return fmt.Errorf("%s is out of range, block sizes must be `/22` or lower, please update `_network` strategy", validateCIDR.String())
	}
	for i := range subnets {
		_, clusterCIDR, err := net.ParseCIDR(resources.SafeStringDereference(subnets[i].IpCidrRange))
		if err != nil {
			return fmt.Errorf("failed to parse cluster subnet into cidr")
		}
		if clusterCIDR.Contains(validateCIDR.IP) || validateCIDR.Contains(clusterCIDR.IP) {
			return fmt.Errorf("ip range creation failed: cidr block %s overlaps with cluster vpc subnet block %s, update _network strategy to continue ip range creation", validateCIDR.String(), clusterCIDR.String())
		}
	}
	return nil
}

func isValidCIDRRange(validateCIDR *net.IPNet) bool {
	mask, _ := validateCIDR.Mask.Size()
	return mask <= defaultIpRangeCIDRMask
}

// parses a subnet URL in the format:
// https://www.googleapis.com/compute/v1/projects/my-project-1234/regions/my-region/subnetworks/my-subnet-name
func parseSubnetUrl(subnetUrl string) (string, string, error) {
	parsed, err := url.Parse(subnetUrl)
	if err != nil {
		return "", "", errorUtil.Wrapf(err, "failed to parse subnet url %s", subnetUrl)
	}
	var name, region string
	path := strings.Split(parsed.Path, "/")
	for i := range path {
		if path[i] == "regions" {
			region = path[i+1]
		}
		if path[i] == "subnetworks" {
			name = path[i+1]
			break
		}
	}
	if name == "" || region == "" {
		return "", "", errors.New("failed to retrieve subnetwork name from URL")
	}
	return name, region, nil
}
