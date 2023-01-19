package gcp

import (
	"context"
	"fmt"
	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const blobstorageProviderName = "gcp-storage"

type BlobStorageProvider struct {
	Client            client.Client
	CredentialManager CredentialManager
	ConfigManager     ConfigManager
}

func NewGCPBlobStorageProvider(client client.Client) *BlobStorageProvider {
	return &BlobStorageProvider{
		Client:            client,
		CredentialManager: NewCredentialMinterCredentialManager(client),
		ConfigManager:     NewDefaultConfigManager(client),
	}
}

func (bsp BlobStorageProvider) GetName() string {
	return blobstorageProviderName
}

func (bsp BlobStorageProvider) SupportsStrategy(deploymentStrategy string) bool {
	return deploymentStrategy == providers.GCPDeploymentStrategy
}

func (bsp BlobStorageProvider) GetReconcileTime(bs *v1alpha1.BlobStorage) time.Duration {
	if bs.Status.Phase != types.PhaseComplete {
		return time.Second * 60
	}
	return resources.GetForcedReconcileTimeOrDefault(defaultReconcileTime)
}

func (bsp BlobStorageProvider) CreateStorage(ctx context.Context, bs *v1alpha1.BlobStorage) (*providers.BlobStorageInstance, types.StatusMessage, error) {
	_, err := bsp.CredentialManager.ReconcileProviderCredentials(ctx, bs.Namespace)
	if err != nil {
		errMsg := fmt.Sprintf("failed to reconcile gcp blob storage provider credentials for blob storage instance %s", bs.Name)
		return nil, types.StatusMessage(errMsg), fmt.Errorf("%s: %w", errMsg, err)
	}
	// TODO implement me
	return nil, "", nil
}

func (bsp BlobStorageProvider) DeleteStorage(ctx context.Context, bs *v1alpha1.BlobStorage) (types.StatusMessage, error) {
	_, err := bsp.CredentialManager.ReconcileProviderCredentials(ctx, bs.Namespace)
	if err != nil {
		errMsg := fmt.Sprintf("failed to reconcile gcp blob storage provider credentials for blob storage instance %s", bs.Name)
		return types.StatusMessage(errMsg), fmt.Errorf("%s: %w", errMsg, err)
	}
	// TODO implement me
	return "", nil
}

var _ providers.BlobStorageProvider = (*BlobStorageProvider)(nil)
