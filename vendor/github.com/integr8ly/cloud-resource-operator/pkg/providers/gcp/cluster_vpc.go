package gcp

import (
	"context"
	"fmt"

	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers/gcp/gcpiface"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	errorUtil "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	utils "k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getClusterVpc(ctx context.Context, c client.Client, networkClient gcpiface.NetworksAPI, projectID string, logger *logrus.Entry) (*computepb.Network, error) {
	// get cluster id
	clusterID, err := resources.GetClusterID(ctx, c)
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting clusterID")
	}

	// get networks with a name that matches clusterID
	networks, err := networkClient.List(ctx, &computepb.ListNetworksRequest{
		Project: projectID,
		Filter:  utils.To(fmt.Sprintf("name = \"%s-*\"", clusterID)),
	})
	if err != nil {
		return nil, errorUtil.Wrap(err, "error getting networks from gcp")
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

	logger.Infof("found cluster %s vpc %s", clusterID, *network.Name)
	return network, nil
}
