package aws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/integr8ly/cloud-resource-operator/pkg/resources"

	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	errorUtil "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultConfigMapName      = "cloud-resources-aws-strategies"
	DefaultConfigMapNamespace = "kube-system"

	DefaultFinalizer = "finalizers.aws.cloud-resources-operator.integreatly.org"
	DefaultRegion    = "eu-west-1"

	regionUSEast1 = "us-east-1"
	regionUSWest2 = "us-west-2"
	regionEUWest1 = "eu-west-1"

	sesSMTPEndpointUSEast1 = "email-smtp.us-east-1.amazonaws.com"
	sesSMTPEndpointUSWest2 = "email-smtp.us-west-2.amazonaws.com"
	sesSMTPEndpointEUWest1 = "email-smtp.eu-west-1.amazonaws.com"
)

//go:generate moq -out config_moq.go . ConfigManager
type ConfigManager interface {
	ReadStorageStrategy(ctx context.Context, rt providers.ResourceType, tier string) (*StrategyConfig, error)
	ReadSMTPCredentialSetStrategy(ctx context.Context, tier string) (*StrategyConfig, error)
	GetDefaultRegionSMTPServerMapping() map[string]string
}

var _ ConfigManager = (*ConfigMapConfigManager)(nil)

type ConfigMapConfigManager struct {
	configMapName      string
	configMapNamespace string
	client             client.Client
}

type StrategyConfig struct {
	Region         string          `json:"region"`
	CreateStrategy json.RawMessage `json:"createStrategy"`
	DeleteStrategy json.RawMessage `json:"deleteStrategy"`
}

func NewConfigMapConfigManager(cm string, namespace string, client client.Client) *ConfigMapConfigManager {
	if cm == "" {
		cm = DefaultConfigMapName
	}
	if namespace == "" {
		namespace = DefaultConfigMapNamespace
	}
	return &ConfigMapConfigManager{
		configMapName:      cm,
		configMapNamespace: namespace,
		client:             client,
	}
}

func NewDefaultConfigMapConfigManager(client client.Client) *ConfigMapConfigManager {
	return NewConfigMapConfigManager(DefaultConfigMapName, DefaultConfigMapNamespace, client)
}

func (m *ConfigMapConfigManager) ReadStorageStrategy(ctx context.Context, rt providers.ResourceType, tier string) (*StrategyConfig, error) {
	stratCfg, err := m.getTierStrategyForProvider(ctx, string(rt), tier)
	if err != nil {
		return nil, errorUtil.Wrapf(err, "failed to get tier to strategy mapping for resource type %s", string(rt))
	}
	return stratCfg, nil
}

func (m *ConfigMapConfigManager) ReadSMTPCredentialSetStrategy(ctx context.Context, tier string) (*StrategyConfig, error) {
	stratCfg, err := m.getTierStrategyForProvider(ctx, string(providers.SMTPCredentialResourceType), tier)
	if err != nil {
		return nil, errorUtil.Wrapf(err, "failed to get tier to strategy mapping for resource type %s", string(providers.BlobStorageResourceType))
	}
	return stratCfg, nil
}

func (m *ConfigMapConfigManager) GetDefaultRegionSMTPServerMapping() map[string]string {
	return map[string]string{
		regionUSEast1: sesSMTPEndpointUSEast1,
		regionUSWest2: sesSMTPEndpointUSWest2,
		regionEUWest1: sesSMTPEndpointEUWest1,
	}
}

func (m *ConfigMapConfigManager) getTierStrategyForProvider(ctx context.Context, rt string, tier string) (*StrategyConfig, error) {
	cm, err := resources.GetConfigMapOrDefault(ctx, m.client, types.NamespacedName{Name: m.configMapName, Namespace: m.configMapNamespace}, m.buildDefaultConfigMap())
	if err != nil {
		return nil, errorUtil.Wrapf(err, "failed to get aws strategy config map %s in namespace %s", m.configMapName, m.configMapNamespace)
	}
	rawStrategyMapping := cm.Data[rt]
	if rawStrategyMapping == "" {
		return nil, errorUtil.New(fmt.Sprintf("aws strategy for resource type %s is not defined", rt))
	}
	var strategyMapping map[string]*StrategyConfig
	if err = json.Unmarshal([]byte(rawStrategyMapping), &strategyMapping); err != nil {
		return nil, errorUtil.Wrapf(err, "failed to unmarshal strategy mapping for resource type %s", rt)
	}
	return strategyMapping[tier], nil
}

func (m *ConfigMapConfigManager) buildDefaultConfigMap() *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: controllerruntime.ObjectMeta{
			Name:      m.configMapName,
			Namespace: m.configMapNamespace,
		},
		Data: map[string]string{
			"blobstorage":     "{\"development\": { \"region\": \"eu-west-1\", \"strategy\": {} }}",
			"smtpcredentials": "{\"development\": { \"region\": \"eu-west-1\", \"strategy\": {} }}",
			"redis":           "{\"development\": { \"region\": \"eu-west-1\", \"strategy\": {} }}",
			"postgres":        "{\"development\": { \"region\": \"eu-west-1\", \"strategy\": {} }}",
		},
	}
}

func buildInfraNameFromObject(ctx context.Context, c client.Client, om controllerruntime.ObjectMeta, n int) (string, error) {
	clusterId, err := resources.GetClusterId(ctx, c)
	if err != nil {
		return "", errorUtil.Wrap(err, "failed to retrieve cluster identifier")
	}
	return resources.ShortenString(fmt.Sprintf("%s-%s-%s", clusterId, om.Namespace, om.Name), n), nil
}
