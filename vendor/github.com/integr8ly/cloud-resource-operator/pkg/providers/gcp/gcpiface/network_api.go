package gcpiface

import (
	"context"
	"errors"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/googleapis/gax-go/v2"
	"github.com/googleapis/gax-go/v2/apierror"
	"google.golang.org/api/option"
)

type NetworksAPI interface {
	List(context.Context, *computepb.ListNetworksRequest, ...gax.CallOption) ([]*computepb.Network, error)
	RemovePeering(context.Context, *computepb.RemovePeeringNetworkRequest, ...gax.CallOption) error
}

// GCP Client code below
type networksClient struct {
	NetworksAPI
	networksService *compute.NetworksClient
}

func NewNetworksAPI(ctx context.Context, opt option.ClientOption) (NetworksAPI, error) {
	networksRestClient, err := compute.NewNetworksRESTClient(ctx, opt)
	if err != nil {
		return nil, err
	}
	return &networksClient{
		networksService: networksRestClient,
	}, nil
}

func (c *networksClient) List(ctx context.Context, req *computepb.ListNetworksRequest, opts ...gax.CallOption) ([]*computepb.Network, error) {
	netIterator := c.networksService.List(ctx, req, opts...)
	var networks []*computepb.Network
	for {
		n, err := netIterator.Next()
		if err != nil {
			var ae *apierror.APIError
			if errors.As(err, &ae) {
				return nil, err
			}
			break
		}
		networks = append(networks, n)
	}
	return networks, nil
}

func (c *networksClient) RemovePeering(ctx context.Context, req *computepb.RemovePeeringNetworkRequest, opts ...gax.CallOption) error {
	op, err := c.networksService.RemovePeering(ctx, req, opts...)
	if err != nil {
		return err
	}
	return op.Wait(ctx)
}

// Mock Client code below
type MockNetworksClient struct {
	NetworksAPI
	ListFn          func(*computepb.ListNetworksRequest) ([]*computepb.Network, error)
	RemovePeeringFn func(*computepb.RemovePeeringNetworkRequest) error
}

func GetMockNetworksClient(modifyFn func(networksClient *MockNetworksClient)) *MockNetworksClient {
	mock := &MockNetworksClient{}
	if modifyFn != nil {
		modifyFn(mock)
	}
	return mock
}

func (m *MockNetworksClient) List(ctx context.Context, req *computepb.ListNetworksRequest, opts ...gax.CallOption) ([]*computepb.Network, error) {
	if m.ListFn != nil {
		return m.ListFn(req)
	}
	return []*computepb.Network{}, nil
}

func (m *MockNetworksClient) RemovePeering(ctx context.Context, req *computepb.RemovePeeringNetworkRequest, opts ...gax.CallOption) error {
	if m.RemovePeeringFn != nil {
		return m.RemovePeeringFn(req)
	}
	return nil
}
