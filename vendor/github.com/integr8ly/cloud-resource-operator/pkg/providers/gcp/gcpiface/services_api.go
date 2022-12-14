package gcpiface

import (
	"context"
	"fmt"

	"google.golang.org/api/option"
	"google.golang.org/api/servicenetworking/v1"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
)

type ServicesAPI interface {
	ConnectionsList(*computepb.Network, string, string) (*servicenetworking.ListConnectionsResponse, error)
	ConnectionsCreate(string, *servicenetworking.Connection) (*servicenetworking.Operation, error)
	ConnectionsDelete(string, *servicenetworking.DeleteConnectionRequest) (*servicenetworking.Operation, error)
}

// GCP Client code below
type servicesClient struct {
	ServicesAPI
	servicenetworkingService *servicenetworking.APIService
}

func NewServicesAPI(ctx context.Context, opt option.ClientOption) (ServicesAPI, error) {
	servicenetworkingService, err := servicenetworking.NewService(ctx, opt)
	if err != nil {
		return nil, err
	}
	return &servicesClient{
		servicenetworkingService: servicenetworkingService,
	}, nil
}

func (c *servicesClient) ConnectionsList(clusterVpc *computepb.Network, projectID string, parent string) (*servicenetworking.ListConnectionsResponse, error) {
	call := c.servicenetworkingService.Services.Connections.List(parent)
	call.Network(fmt.Sprintf("projects/%s/global/networks/%s", projectID, clusterVpc.GetName()))
	return call.Do()
}

func (c *servicesClient) ConnectionsCreate(parent string, connection *servicenetworking.Connection) (*servicenetworking.Operation, error) {
	return c.servicenetworkingService.Services.Connections.Create(
		parent,
		connection,
	).Do()
}

func (c *servicesClient) ConnectionsDelete(name string, deleteconnectionrequest *servicenetworking.DeleteConnectionRequest) (*servicenetworking.Operation, error) {
	return c.servicenetworkingService.Services.Connections.DeleteConnection(
		name,
		deleteconnectionrequest,
	).Do()
}

// Mock Client code below
type MockServicesClient struct {
	ServicesAPI
	ConnectionsListFn    func(*computepb.Network, string, string) (*servicenetworking.ListConnectionsResponse, error)
	ConnectionsListFnTwo func(*computepb.Network, string, string) (*servicenetworking.ListConnectionsResponse, error)
	ConnectionsCreateFn  func(string, *servicenetworking.Connection) (*servicenetworking.Operation, error)
	ConnectionsDeleteFn  func(string, *servicenetworking.DeleteConnectionRequest) (*servicenetworking.Operation, error)
	call                 int
	Done                 bool
}

func GetMockServicesClient(modifyFn func(servicesClient *MockServicesClient)) *MockServicesClient {
	mock := &MockServicesClient{
		Done: true,
	}
	if modifyFn != nil {
		modifyFn(mock)
	}
	return mock
}

func (m *MockServicesClient) ConnectionsList(clusterVpc *computepb.Network, projectID string, parent string) (*servicenetworking.ListConnectionsResponse, error) {
	m.call++
	if m.ConnectionsListFn != nil && m.call == 1 {
		return m.ConnectionsListFn(clusterVpc, projectID, parent)
	}
	if m.ConnectionsListFnTwo != nil && m.call > 1 {
		return m.ConnectionsListFnTwo(clusterVpc, projectID, parent)
	}
	return &servicenetworking.ListConnectionsResponse{
		Connections: []*servicenetworking.Connection{},
	}, nil
}

func (m *MockServicesClient) ConnectionsCreate(parent string, connection *servicenetworking.Connection) (*servicenetworking.Operation, error) {
	if m.ConnectionsCreateFn != nil {
		return m.ConnectionsCreateFn(parent, connection)
	}
	return &servicenetworking.Operation{
		Done: m.Done,
	}, nil
}

func (m *MockServicesClient) ConnectionsDelete(name string, deleteconnectionrequest *servicenetworking.DeleteConnectionRequest) (*servicenetworking.Operation, error) {
	if m.ConnectionsDeleteFn != nil {
		return m.ConnectionsDeleteFn(name, deleteconnectionrequest)
	}
	return &servicenetworking.Operation{
		Done: m.Done,
	}, nil
}
