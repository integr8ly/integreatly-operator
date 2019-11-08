package openshift

import (
	"context"
	"time"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers/aws"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	varPlaceholder = "REPLACE_ME"
)

var _ providers.BlobStorageProvider = (*BlobStorageProvider)(nil)

type BlobStorageProvider struct {
	Client client.Client
	Logger *logrus.Entry
}

func NewBlobStorageProvider(c client.Client, l *logrus.Entry) *BlobStorageProvider {
	return &BlobStorageProvider{
		Client: c,
		Logger: l,
	}
}

func (b BlobStorageProvider) GetName() string {
	return "openshift-blobstorage"
}

func (b BlobStorageProvider) SupportsStrategy(s string) bool {
	return providers.OpenShiftDeploymentStrategy == s
}

func (b BlobStorageProvider) GetReconcileTime(bs *v1alpha1.BlobStorage) time.Duration {
	return time.Second * 10
}

func (b BlobStorageProvider) CreateStorage(ctx context.Context, bs *v1alpha1.BlobStorage) (*providers.BlobStorageInstance, types.StatusMessage, error) {
	// default to an empty s3 set of credentials for now. in the future. this should determine the cloud provider being
	// used by checking the infrastructure cr.
	dd := &aws.BlobStorageDeploymentDetails{
		BucketName:          varPlaceholder,
		BucketRegion:        varPlaceholder,
		CredentialKeyID:     varPlaceholder,
		CredentialSecretKey: varPlaceholder,
	}

	if bs.Spec.SecretRef.Namespace == "" {
		bs.Spec.SecretRef.Namespace = bs.Namespace
	}

	if bs.Status.Phase != types.PhaseComplete || bs.Status.SecretRef.Name == "" || bs.Status.SecretRef.Namespace == "" {
		return &providers.BlobStorageInstance{
			DeploymentDetails: dd,
		}, "reconcile complete", nil
	}

	sec := &v1.Secret{}
	if err := b.Client.Get(ctx, client.ObjectKey{Name: bs.Status.SecretRef.Name, Namespace: bs.Status.SecretRef.Namespace}, sec); err != nil {
		return nil, "failed to reconcile", err
	}

	dd.BucketName = resources.StringOrDefault(string(sec.Data[aws.DetailsBlobStorageBucketName]), varPlaceholder)
	dd.BucketRegion = resources.StringOrDefault(string(sec.Data[aws.DetailsBlobStorageBucketRegion]), varPlaceholder)
	dd.CredentialKeyID = resources.StringOrDefault(string(sec.Data[aws.DetailsBlobStorageCredentialKeyID]), varPlaceholder)
	dd.CredentialSecretKey = resources.StringOrDefault(string(sec.Data[aws.DetailsBlobStorageCredentialSecretKey]), varPlaceholder)
	return &providers.BlobStorageInstance{
		DeploymentDetails: dd,
	}, "reconcile complete", nil
}

func (b BlobStorageProvider) DeleteStorage(ctx context.Context, bs *v1alpha1.BlobStorage) (types.StatusMessage, error) {
	return "deletion complete", nil
}
