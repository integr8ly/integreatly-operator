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

type AddressAPI interface {
	Get(context.Context, *computepb.GetGlobalAddressRequest, ...gax.CallOption) (*computepb.Address, error)
	Insert(context.Context, *computepb.InsertGlobalAddressRequest, ...gax.CallOption) error
	Delete(context.Context, *computepb.DeleteGlobalAddressRequest, ...gax.CallOption) error
}

// GCP Client code below
type addressClient struct {
	AddressAPI
	addressService *compute.GlobalAddressesClient
}

func NewAddressAPI(ctx context.Context, opt option.ClientOption) (AddressAPI, error) {
	globalAddressesRestClient, err := compute.NewGlobalAddressesRESTClient(ctx, opt)
	if err != nil {
		return nil, err
	}
	return &addressClient{
		addressService: globalAddressesRestClient,
	}, nil
}

func (c *addressClient) Get(ctx context.Context, req *computepb.GetGlobalAddressRequest, opts ...gax.CallOption) (*computepb.Address, error) {
	return c.addressService.Get(ctx, req, opts...)
}

func (c *addressClient) Insert(ctx context.Context, req *computepb.InsertGlobalAddressRequest, opts ...gax.CallOption) error {
	op, err := c.addressService.Insert(ctx, req, opts...)
	if err != nil {
		return err
	}
	return op.Wait(ctx)
}

func (c *addressClient) Delete(ctx context.Context, req *computepb.DeleteGlobalAddressRequest, opts ...gax.CallOption) error {
	op, err := c.addressService.Delete(ctx, req, opts...)
	if err != nil {
		return err
	}
	return op.Wait(ctx)
}

type MockAddressClient struct {
	AddressAPI
	GetFn    func(*computepb.GetGlobalAddressRequest) (*computepb.Address, error)
	GetFnTwo func(*computepb.GetGlobalAddressRequest) (*computepb.Address, error)
	InsertFn func(*computepb.InsertGlobalAddressRequest) error
	DeleteFn func(*computepb.DeleteGlobalAddressRequest) error
	call     int
}

func GetMockAddressClient(modifyFn func(addressClient *MockAddressClient)) *MockAddressClient {
	mock := &MockAddressClient{
		InsertFn: func(req *computepb.InsertGlobalAddressRequest) error {
			return nil
		},
		DeleteFn: func(req *computepb.DeleteGlobalAddressRequest) error {
			return nil
		},
	}
	if modifyFn != nil {
		modifyFn(mock)
	}
	return mock
}

func (m *MockAddressClient) Get(ctx context.Context, req *computepb.GetGlobalAddressRequest, opts ...gax.CallOption) (*computepb.Address, error) {
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

func (m *MockAddressClient) Insert(ctx context.Context, req *computepb.InsertGlobalAddressRequest, opts ...gax.CallOption) error {
	return m.InsertFn(req)
}

func (m *MockAddressClient) Delete(ctx context.Context, req *computepb.DeleteGlobalAddressRequest, opts ...gax.CallOption) error {
	return m.DeleteFn(req)
}
