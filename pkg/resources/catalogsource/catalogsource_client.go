package catalogsource

import (
	"context"
	"encoding/json"
	"fmt"

	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	"github.com/sirupsen/logrus"

	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	grpc "github.com/operator-framework/operator-registry/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate moq -out catalogsource_client_mock.go . CatalogSourceClientInterface
type CatalogSourceClientInterface interface {
	GetLatestCSV(catalogSourceKey k8sclient.ObjectKey, packageName, channelName string) (*olmv1alpha1.ClusterServiceVersion, error)
}

type CatalogSourceClient struct {
	ctx    context.Context
	client k8sclient.Client
}

var _ CatalogSourceClientInterface = &CatalogSourceClient{}

func NewClient(ctx context.Context, client k8sclient.Client) (*CatalogSourceClient, error) {
	return &CatalogSourceClient{
		client: client,
		ctx:    ctx,
	}, nil
}

func (client *CatalogSourceClient) GetLatestCSV(catalogSourceKey k8sclient.ObjectKey, packageName, channelName string) (*olmv1alpha1.ClusterServiceVersion, error) {

	catalogsource := &coreosv1alpha1.CatalogSource{}
	err := client.client.Get(client.ctx, catalogSourceKey, catalogsource)
	if err != nil {
		return nil, fmt.Errorf("failed to get catalogsource: %w", err)
	}

	clientGRPC, err := grpc.NewClient(catalogsource.Address())
	if err != nil {
		return nil, fmt.Errorf("failed to create a new GRPC client: %w", err)
	}

	defer clientGRPC.Close()

	bundle, err := clientGRPC.GetBundleInPackageChannel(client.ctx, packageName, channelName)
	if err != nil {
		return nil, fmt.Errorf("failed to get csv from catalogsource: %w", err)
	}

	csv := &olmv1alpha1.ClusterServiceVersion{}
	err = json.Unmarshal([]byte(bundle.GetCsvJson()), &csv)
	if err != nil {
		logrus.Errorf("failed to unmarshal json: %v", err)
	}
	return csv, nil
}
