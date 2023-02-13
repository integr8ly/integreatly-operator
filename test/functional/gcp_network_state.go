package functional

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"cloud.google.com/go/compute/apiv1/computepb"
	croGCP "github.com/integr8ly/cloud-resource-operator/pkg/providers/gcp"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers/gcp/gcpiface"
	croResources "github.com/integr8ly/cloud-resource-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/test/common"
	"google.golang.org/api/option"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	utils "k8s.io/utils/pointer"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	defaultGcpServiceConnectionURI = "servicenetworking.googleapis.com"
	defaultGcpServicesFormat       = "services/%s"
	defaultIpRangePostfix          = "ip-range"
	defaultGcpIdentifierLength     = 40
	gcpTier                        = "production"
	gcpAllowedCidrRanges           = []string{
		"10.255.255.255/8",
		"172.31.255.255/12",
	}
)

// TestGCPNetworkState tests GCP cloud network components
func TestGCPNetworkState(t common.TestingTB, testingCtx *common.TestingContext) {
	ctx := context.Background()

	// get the strategy map to get the GCP Subnet cidr block
	strategyMap := &corev1.ConfigMap{}
	err := testingCtx.Client.Get(ctx, types.NamespacedName{
		Namespace: common.RHOAMOperatorNamespace,
		Name:      croGCP.DefaultConfigMapName,
	}, strategyMap)
	if err != nil {
		t.Fatal("could not get gcp strategy map", err)
	}

	strat, err := getStrategyForResource(strategyMap, networkResourceType, gcpTier)
	if err != nil {
		t.Skip("_network key does not exist in strategy configmap, skipping standalone vpc network test")
	}

	// get the cidr block from Strategy Map
	expectedCidr, err := verifyAndGetCidrBlockFromGCPStrategyMap(strat)
	if err != nil {
		t.Fatal(err)
	}

	serviceAccountJson, err := getGCPCredentials(ctx, testingCtx.Client)
	if err != nil {
		t.Fatalf("failed to retrieve gcp credentials %s", err)
	}
	opt := option.WithCredentialsJSON(serviceAccountJson)

	projectID, err := croResources.GetGCPProject(ctx, testingCtx.Client)
	if err != nil {
		t.Fatalf("error get Default Project ID %s", err)
	}

	address, err := verifyGcpAddressRange(ctx, testingCtx.Client, projectID, expectedCidr, opt)
	if err != nil {
		t.Fatalf("error verifying gcp address range %s", err)
	}
	t.Logf("address range %s verified as expected", address.GetName())

	network, err := getClusterVpc(ctx, testingCtx.Client, projectID, opt)
	if err != nil {
		t.Fatalf("failed to retrieve cluster network %s", err)
	}

	peeringName, err := verifyGcpServiceConnection(ctx, network, projectID, opt)
	if err != nil {
		t.Fatalf("error verifying gcp service connection %s", err)
	}
	t.Log("service connection verified as expected")

	err = verifyGcpPeeringConnection(network, peeringName)
	if err != nil {
		t.Fatalf("error verifying gcp peering connection %s", err)
	}
	t.Logf("peering connection %s verified  as expected", peeringName)
}

func verifyAndGetCidrBlockFromGCPStrategyMap(strat *strategyMap) (string, error) {
	vpcCreateConfig := &croGCP.CreateVpcInput{}
	if err := json.Unmarshal(strat.CreateStrategy, vpcCreateConfig); err != nil {
		return "", fmt.Errorf("failed to unmarshal vpc create config")
	}
	if vpcCreateConfig.CidrBlock != "" {
		if err := verifyCidrBlockIsInAllowedRange(vpcCreateConfig.CidrBlock, gcpAllowedCidrRanges); err != nil {
			return "", fmt.Errorf("cidr block %s is not within the allowed range %s", vpcCreateConfig.CidrBlock, err)
		}
	} else {
		fmt.Println("strategy map CIDR block is empty")
	}
	return vpcCreateConfig.CidrBlock, nil
}

func verifyGcpAddressRange(ctx context.Context, client k8sclient.Client, projectID string, expectedCidr string, opt option.ClientOption) (*computepb.Address, error) {
	addressClient, err := gcpiface.NewAddressAPI(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("error creating address client %w", err)
	}
	ipAddressName, err := croResources.BuildInfraName(ctx, client, defaultIpRangePostfix, defaultGcpIdentifierLength)
	if err != nil {
		return nil, fmt.Errorf("failed to create ip address range name %w", err)
	}

	address, err := addressClient.Get(ctx, &computepb.GetGlobalAddressRequest{
		Project: projectID,
		Address: ipAddressName,
	})
	if err != nil {
		return nil, fmt.Errorf("error retrieving address range %w", err)
	}
	if address.GetStatus() != computepb.Address_RESERVED.String() {
		return nil, fmt.Errorf("address range status expected RESERVED, but found %s", address.GetStatus())
	}
	if expectedCidr != "" {
		if cidr, err := strconv.ParseInt(expectedCidr, 10, 32); err != nil && address.GetPrefixLength() != int32(cidr) {
			return nil, fmt.Errorf("address range cidr %d, does not match expected %s", address.GetPrefixLength(), expectedCidr)
		}
	}
	return address, nil
}

func verifyGcpServiceConnection(ctx context.Context, network *computepb.Network, projectID string, opt option.ClientOption) (string, error) {
	serviceClient, err := gcpiface.NewServicesAPI(ctx, opt)
	if err != nil {
		return "", fmt.Errorf("error creating service client %w", err)
	}
	resp, err := serviceClient.ConnectionsList(network, projectID, fmt.Sprintf(defaultGcpServicesFormat, defaultGcpServiceConnectionURI))
	if err != nil || resp == nil {
		return "", fmt.Errorf("failed to retrieve service connection %w", err)
	}
	if len(resp.Connections) == 0 {
		return "", errors.New("expected 1 service connection, got 0")
	}
	peering := resp.Connections[0].Peering
	if peering == "" {
		return "", errors.New("empty peering associated with service connection")
	}
	return peering, nil
}

func verifyGcpPeeringConnection(network *computepb.Network, peeringName string) error {
	var peering *computepb.NetworkPeering
	for i, netPeering := range network.GetPeerings() {
		if netPeering.GetName() == peeringName {
			peering = network.GetPeerings()[i]
			break
		}
	}

	if peering == nil {
		return fmt.Errorf("unable to find peering with name %s", peeringName)
	}

	if peering.GetState() != computepb.NetworkPeering_ACTIVE.String() {
		return fmt.Errorf("peering is in invalid state %s", peering.GetState())
	}
	return nil
}

func getClusterVpc(ctx context.Context, client k8sclient.Client, projectID string, opt option.ClientOption) (*computepb.Network, error) {
	clusterID, err := getClusterID(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve cluster id %w", err)
	}
	networkClient, err := gcpiface.NewNetworksAPI(ctx, opt)
	if err != nil {
		return nil, err
	}
	// get networks with a name that matches clusterID
	networks, err := networkClient.List(ctx, &computepb.ListNetworksRequest{
		Project: projectID,
		Filter:  utils.String(fmt.Sprintf("name = \"%s-*\"", clusterID)),
	})
	if err != nil {
		return nil, fmt.Errorf("error getting networks from gcp %w", err)
	}
	// confirm only one network matched the clusterID
	if len(networks) != 1 {
		return nil, fmt.Errorf("cannot determine cluster vpc. matching networks found %d", len(networks))
	}
	network := networks[0]

	// check the network has at least two subnets
	if len(network.GetSubnetworks()) < defaultNumberOfExpectedSubnets {
		return nil, fmt.Errorf("found cluster vpc has only %d subnetworks, expected at least 2", len(network.Subnetworks))
	}
	return network, nil
}
