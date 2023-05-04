package gcpiface

import (
	"context"
	"errors"

	"cloud.google.com/go/iam"
	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type StorageAPI interface {
	CreateBucket(ctx context.Context, bucket, projectID string, attrs *storage.BucketAttrs) error
	GetBucket(ctx context.Context, bucket string) (*storage.BucketAttrs, error)
	DeleteBucket(ctx context.Context, bucket string) error
	SetBucketPolicy(ctx context.Context, bucket, identity, role string) error
	HasBucketPolicy(ctx context.Context, bucket, identity, role string) (bool, error)
	ListObjects(ctx context.Context, bucket string, query *storage.Query) ([]*storage.ObjectAttrs, error)
	GetObjectMetadata(ctx context.Context, bucket, object string) (*storage.ObjectAttrs, error)
	DeleteObject(ctx context.Context, bucket, object string) error
}

type storageClient struct {
	StorageAPI
	storageService *storage.Client
	logger         *logrus.Entry
}

func NewStorageAPI(ctx context.Context, opt option.ClientOption, logger *logrus.Entry) (StorageAPI, error) {
	cloudStorageClient, err := storage.NewClient(ctx, opt)
	if err != nil {
		return nil, err
	}
	return &storageClient{
		storageService: cloudStorageClient,
		logger:         logger,
	}, nil
}

func (c *storageClient) CreateBucket(ctx context.Context, bucket, projectID string, attrs *storage.BucketAttrs) error {
	c.logger.Infof("creating bucket %q", bucket)
	bucketHandle := c.storageService.Bucket(bucket)
	err := bucketHandle.Create(ctx, projectID, attrs)
	if err != nil {
		return err
	}
	return nil
}

func (c *storageClient) GetBucket(ctx context.Context, bucket string) (*storage.BucketAttrs, error) {
	c.logger.Infof("getting bucket %s", bucket)
	bucketHandle := c.storageService.Bucket(bucket)
	return bucketHandle.Attrs(ctx)
}

func (c *storageClient) DeleteBucket(ctx context.Context, bucket string) error {
	c.logger.Infof("deleting bucket %q", bucket)
	bucketHandle := c.storageService.Bucket(bucket)
	err := bucketHandle.Delete(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (c *storageClient) SetBucketPolicy(ctx context.Context, bucket, identity, role string) error {
	c.logger.Infof("setting policy on bucket %q", bucket)
	bucketHandle := c.storageService.Bucket(bucket)
	policy, err := bucketHandle.IAM().Policy(ctx)
	if err != nil {
		return err
	}
	policy.Add(identity, iam.RoleName(role))
	if err = bucketHandle.IAM().SetPolicy(ctx, policy); err != nil {
		return err
	}
	return nil
}

func (c *storageClient) HasBucketPolicy(ctx context.Context, bucket, identity, role string) (bool, error) {
	c.logger.Infof("checking policy on bucket %q", bucket)
	bucketHandle := c.storageService.Bucket(bucket)
	policy, err := bucketHandle.IAM().Policy(ctx)
	if err != nil {
		return false, err
	}
	return policy.HasRole(identity, iam.RoleName(role)), nil
}

func (c *storageClient) ListObjects(ctx context.Context, bucket string, query *storage.Query) ([]*storage.ObjectAttrs, error) {
	c.logger.Infof("listing objects from bucket %q", bucket)
	objectIterator := c.storageService.Bucket(bucket).Objects(ctx, query)
	var objects []*storage.ObjectAttrs
	for {
		oa, err := objectIterator.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		objects = append(objects, oa)
	}
	return objects, nil
}

func (c *storageClient) GetObjectMetadata(ctx context.Context, bucket, object string) (*storage.ObjectAttrs, error) {
	c.logger.Infof("fetching object %q from bucket %q", object, bucket)
	objectHandle := c.storageService.Bucket(bucket).Object(object)
	attrs, err := objectHandle.Attrs(ctx)
	if err != nil {
		return nil, err
	}
	return attrs, nil
}

func (c *storageClient) DeleteObject(ctx context.Context, bucket, object string) error {
	c.logger.Infof("deleting object %q from bucket %q", object, bucket)
	objectHandle := c.storageService.Bucket(bucket).Object(object)
	err := objectHandle.Delete(ctx)
	if err != nil {
		return err
	}
	return nil
}

type MockStorageClient struct {
	StorageAPI
	CreateBucketFn      func(context.Context, string, string, *storage.BucketAttrs) error
	GetBucketFn         func(context.Context, string) (*storage.BucketAttrs, error)
	DeleteBucketFn      func(context.Context, string) error
	SetBucketPolicyFn   func(context.Context, string, string, string) error
	HasBucketPolicyFn   func(context.Context, string, string, string) (bool, error)
	ListObjectsFn       func(context.Context, string, *storage.Query) ([]*storage.ObjectAttrs, error)
	GetObjectMetadataFn func(context.Context, string, string) (*storage.ObjectAttrs, error)
	DeleteObjectFn      func(context.Context, string, string) error
}

func GetMockStorageClient(modifyFn func(storageClient *MockStorageClient)) *MockStorageClient {
	mock := &MockStorageClient{
		CreateBucketFn: func(ctx context.Context, bucket, projectID string, attrs *storage.BucketAttrs) error {
			return nil
		},
		GetBucketFn: func(ctx context.Context, bucket string) (*storage.BucketAttrs, error) {
			return nil, nil
		},
		DeleteBucketFn: func(ctx context.Context, bucket string) error {
			return nil
		},
		SetBucketPolicyFn: func(ctx context.Context, bucket, identity, role string) error {
			return nil
		},
		HasBucketPolicyFn: func(ctx context.Context, bucket, identity, role string) (bool, error) {
			return false, nil
		},
		ListObjectsFn: func(ctx context.Context, bucket string, query *storage.Query) ([]*storage.ObjectAttrs, error) {
			return []*storage.ObjectAttrs{}, nil
		},
		GetObjectMetadataFn: func(ctx context.Context, bucket, object string) (*storage.ObjectAttrs, error) {
			return &storage.ObjectAttrs{}, nil
		},
		DeleteObjectFn: func(ctx context.Context, bucket, object string) error {
			return nil
		},
	}
	if modifyFn != nil {
		modifyFn(mock)
	}
	return mock
}

func (m *MockStorageClient) CreateBucket(ctx context.Context, bucket, projectID string, attrs *storage.BucketAttrs) error {
	return m.CreateBucketFn(ctx, bucket, projectID, attrs)
}

func (m *MockStorageClient) GetBucket(ctx context.Context, bucket string) (*storage.BucketAttrs, error) {
	return m.GetBucketFn(ctx, bucket)
}

func (m *MockStorageClient) DeleteBucket(ctx context.Context, bucket string) error {
	return m.DeleteBucketFn(ctx, bucket)
}

func (m *MockStorageClient) SetBucketPolicy(ctx context.Context, bucket, identity, role string) error {
	return m.SetBucketPolicyFn(ctx, bucket, identity, role)
}

func (m *MockStorageClient) HasBucketPolicy(ctx context.Context, bucket, identity, role string) (bool, error) {
	return m.HasBucketPolicyFn(ctx, bucket, identity, role)
}

func (m *MockStorageClient) ListObjects(ctx context.Context, bucket string, query *storage.Query) ([]*storage.ObjectAttrs, error) {
	return m.ListObjectsFn(ctx, bucket, query)
}

func (m *MockStorageClient) GetObjectMetadata(ctx context.Context, bucket, object string) (*storage.ObjectAttrs, error) {
	return m.GetObjectMetadataFn(ctx, bucket, object)
}

func (m *MockStorageClient) DeleteObject(ctx context.Context, bucket, object string) error {
	return m.DeleteObjectFn(ctx, bucket, object)
}
