package gcpiface

import (
	redis "cloud.google.com/go/redis/apiv1"
	"context"
	"errors"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	redispb "google.golang.org/genproto/googleapis/cloud/redis/v1"
)

type RedisAPI interface {
	ListInstances(context.Context, *redispb.ListInstancesRequest, ...gax.CallOption) ([]*redispb.Instance, error)
	DeleteInstance(context.Context, *redispb.DeleteInstanceRequest, ...gax.CallOption) (*redis.DeleteInstanceOperation, error)
	CreateInstance(context.Context, *redispb.CreateInstanceRequest, ...gax.CallOption) (*redis.CreateInstanceOperation, error)
	GetInstance(context.Context, *redispb.GetInstanceRequest, ...gax.CallOption) (*redispb.Instance, error)
}

type redisClient struct {
	RedisAPI
	redisService *redis.CloudRedisClient
}

func NewRedisAPI(ctx context.Context, opt option.ClientOption) (RedisAPI, error) {
	cloudRedisClient, err := redis.NewCloudRedisClient(ctx, opt)
	if err != nil {
		return nil, err
	}
	return &redisClient{
		redisService: cloudRedisClient,
	}, nil
}

func (c *redisClient) ListInstances(ctx context.Context, req *redispb.ListInstancesRequest, opts ...gax.CallOption) ([]*redispb.Instance, error) {
	redisIterator := c.redisService.ListInstances(ctx, req, opts...)
	var instances []*redispb.Instance
	for {
		instance, err := redisIterator.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		instances = append(instances, instance)
	}
	return instances, nil
}

func (c *redisClient) DeleteInstance(ctx context.Context, req *redispb.DeleteInstanceRequest, opts ...gax.CallOption) (*redis.DeleteInstanceOperation, error) {
	return c.redisService.DeleteInstance(ctx, req, opts...)
}

func (c *redisClient) CreateInstance(ctx context.Context, req *redispb.CreateInstanceRequest, opts ...gax.CallOption) (*redis.CreateInstanceOperation, error) {
	return c.redisService.CreateInstance(ctx, req, opts...)
}

func (c *redisClient) GetInstance(ctx context.Context, req *redispb.GetInstanceRequest, opts ...gax.CallOption) (*redispb.Instance, error) {
	return c.redisService.GetInstance(ctx, req, opts...)
}

type MockRedisClient struct {
	RedisAPI
	ListInstancesFn  func(context.Context, *redispb.ListInstancesRequest, ...gax.CallOption) ([]*redispb.Instance, error)
	DeleteInstanceFn func(context.Context, *redispb.DeleteInstanceRequest, ...gax.CallOption) (*redis.DeleteInstanceOperation, error)
	CreateInstanceFn func(context.Context, *redispb.CreateInstanceRequest, ...gax.CallOption) (*redis.CreateInstanceOperation, error)
	GetInstanceFn    func(context.Context, *redispb.GetInstanceRequest, ...gax.CallOption) (*redispb.Instance, error)
}

func GetMockRedisClient(modifyFn func(redisClient *MockRedisClient)) *MockRedisClient {
	mock := &MockRedisClient{
		ListInstancesFn: func(ctx context.Context, request *redispb.ListInstancesRequest, opts ...gax.CallOption) ([]*redispb.Instance, error) {
			return []*redispb.Instance{}, nil
		},
		DeleteInstanceFn: func(ctx context.Context, request *redispb.DeleteInstanceRequest, opts ...gax.CallOption) (*redis.DeleteInstanceOperation, error) {
			return &redis.DeleteInstanceOperation{}, nil
		},
		CreateInstanceFn: func(ctx context.Context, request *redispb.CreateInstanceRequest, opts ...gax.CallOption) (*redis.CreateInstanceOperation, error) {
			return &redis.CreateInstanceOperation{}, nil
		},
		GetInstanceFn: func(ctx context.Context, request *redispb.GetInstanceRequest, callOption ...gax.CallOption) (*redispb.Instance, error) {
			return &redispb.Instance{}, nil
		},
	}
	if modifyFn != nil {
		modifyFn(mock)
	}
	return mock
}

func (m *MockRedisClient) ListInstances(ctx context.Context, req *redispb.ListInstancesRequest, opts ...gax.CallOption) ([]*redispb.Instance, error) {
	if m.ListInstancesFn != nil {
		return m.ListInstancesFn(ctx, req, opts...)
	}
	return []*redispb.Instance{}, nil
}

func (m *MockRedisClient) DeleteInstance(ctx context.Context, req *redispb.DeleteInstanceRequest, opts ...gax.CallOption) (*redis.DeleteInstanceOperation, error) {
	if m.DeleteInstanceFn != nil {
		return m.DeleteInstanceFn(ctx, req, opts...)
	}
	return &redis.DeleteInstanceOperation{}, nil
}

func (m *MockRedisClient) CreateInstance(ctx context.Context, req *redispb.CreateInstanceRequest, opts ...gax.CallOption) (*redis.CreateInstanceOperation, error) {
	if m.CreateInstanceFn != nil {
		return m.CreateInstanceFn(ctx, req, opts...)
	}
	return &redis.CreateInstanceOperation{}, nil
}

func (m *MockRedisClient) GetInstance(ctx context.Context, req *redispb.GetInstanceRequest, opts ...gax.CallOption) (*redispb.Instance, error) {
	if m.GetInstanceFn != nil {
		return m.GetInstanceFn(ctx, req, opts...)
	}
	return &redispb.Instance{}, nil
}
