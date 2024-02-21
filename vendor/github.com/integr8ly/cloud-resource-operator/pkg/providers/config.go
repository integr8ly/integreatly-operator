package providers

import (
	"context"
	"encoding/json"

	"github.com/integr8ly/cloud-resource-operator/internal/k8sutil"

	"github.com/integr8ly/cloud-resource-operator/pkg/resources"

	controllerruntime "sigs.k8s.io/controller-runtime"

	"k8s.io/apimachinery/pkg/types"

	errorUtil "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DefaultProviderConfigMapName represents the default config map name
const (
	DefaultProviderConfigMapName = "cloud-resource-config"
)

// DefaultConfigNamespace get default namespace
var DefaultConfigNamespace, _ = k8sutil.GetWatchNamespace()

// DeploymentStrategyMapping Deployment Strategy Map
type DeploymentStrategyMapping struct {
	BlobStorage string `json:"blobstorage"`
	Redis       string `json:"redis"`
	Postgres    string `json:"postgres"`
}

//go:generate moq -out config_moq.go . ConfigManager
type ConfigManager interface {
	GetStrategyMappingForDeploymentType(ctx context.Context, t string) (*DeploymentStrategyMapping, error)
}

var _ ConfigManager = (*ConfigMapConfigManager)(nil)

type ConfigMapConfigManager struct {
	client                     client.Client
	providerConfigMapName      string
	providerConfigMapNamespace string
}

func NewConfigManager(cm string, namespace string, client client.Client) *ConfigMapConfigManager {
	if cm == "" {
		cm = DefaultProviderConfigMapName
	}
	if namespace == "" {
		namespace = DefaultConfigNamespace
	}
	return &ConfigMapConfigManager{
		client:                     client,
		providerConfigMapName:      cm,
		providerConfigMapNamespace: namespace,
	}
}

// GetStrategyMappingForDeploymentType Get high-level information about the strategy used in a deployment type
func (m *ConfigMapConfigManager) GetStrategyMappingForDeploymentType(ctx context.Context, t string) (*DeploymentStrategyMapping, error) {
	cm, err := resources.GetConfigMapOrDefault(ctx, m.client, types.NamespacedName{Name: m.providerConfigMapName, Namespace: m.providerConfigMapNamespace}, m.buildDefaultConfigMap())
	if err != nil {
		return nil, errorUtil.Wrapf(err, "failed to read provider config from configmap %s in namespace %s", m.providerConfigMapName, m.providerConfigMapNamespace)
	}
	dsm := &DeploymentStrategyMapping{}
	if err = json.Unmarshal([]byte(cm.Data[t]), dsm); err != nil {
		return nil, errorUtil.Wrapf(err, "failed to unmarshal config for deployment type %s", t)
	}
	return dsm, nil
}

func (m *ConfigMapConfigManager) buildDefaultConfigMap() *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: controllerruntime.ObjectMeta{
			Name:      m.providerConfigMapName,
			Namespace: m.providerConfigMapNamespace,
		},
		Data: map[string]string{
			"openshift": "{\"blobstorage\":\"openshift\", \"redis\":\"openshift\", \"postgres\":\"openshift\"}",
			"aws":       "{\"blobstorage\":\"aws\", \"redis\":\"aws\", \"postgres\":\"aws\"}",
			"gcp":       "{\"blobstorage\":\"gcp\", \"redis\":\"gcp\", \"postgres\":\"gcp\"}",
		},
	}
}
