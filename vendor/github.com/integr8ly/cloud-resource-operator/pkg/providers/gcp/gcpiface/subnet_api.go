package gcpiface

import (
	"context"
	"net/http"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

type SubnetsApi interface {
	Get(context.Context, *computepb.GetSubnetworkRequest, ...gax.CallOption) (*computepb.Subnetwork, error)
}

// GCP Client code below
type subnetsClient struct {
	SubnetsApi
	subnetsService *compute.SubnetworksClient
}

func NewSubnetsAPI(ctx context.Context, opt option.ClientOption) (SubnetsApi, error) {
	subnetsRestClient, err := compute.NewSubnetworksRESTClient(ctx, opt)
	if err != nil {
		return nil, err
	}
	return &subnetsClient{
		subnetsService: subnetsRestClient,
	}, nil
}

func (c *subnetsClient) Get(ctx context.Context, req *computepb.GetSubnetworkRequest, opts ...gax.CallOption) (*computepb.Subnetwork, error) {
	return c.subnetsService.Get(ctx, req, opts...)
}

// Mock Client code below
type MockSubnetsClient struct {
	SubnetsApi
	GetFn    func(*computepb.GetSubnetworkRequest) (*computepb.Subnetwork, error)
	GetFnTwo func(*computepb.GetSubnetworkRequest) (*computepb.Subnetwork, error)
	call     int
}

func GetMockSubnetsClient(modifyFn func(subnetClient *MockSubnetsClient)) *MockSubnetsClient {
	mock := &MockSubnetsClient{}
	if modifyFn != nil {
		modifyFn(mock)
	}
	return mock
}

func (m *MockSubnetsClient) Get(ctx context.Context, req *computepb.GetSubnetworkRequest, opts ...gax.CallOption) (*computepb.Subnetwork, error) {
	m.call++
	if m.GetFn != nil && m.call == 1 {
		return m.GetFn(req)
	}
	if m.GetFnTwo != nil && m.call > 1 {
		return m.GetFnTwo(req)
	}
	return nil, &googleapi.Error{
		Code: http.StatusNotFound,
	}
}
