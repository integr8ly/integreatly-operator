package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	errorUtil "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"time"

	"github.com/integr8ly/cloud-resource-operator/internal/k8sutil"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultConfigMapName       = "cloud-resources-gcp-strategies"
	defaultReconcileTime       = time.Second * 30
	DefaultFinalizer           = "cloud-resources-operator.integreatly.org/finalizers"
	defaultGcpIdentifierLength = 40
)

// DefaultConfigMapNamespace is the default namespace that Configmaps will be created in
var DefaultConfigMapNamespace, _ = k8sutil.GetWatchNamespace()

type StrategyConfig struct {
	Region         string          `json:"region"`
	ProjectID      string          `json:"projectID"`
	CreateStrategy json.RawMessage `json:"createStrategy"`
	DeleteStrategy json.RawMessage `json:"deleteStrategy"`
}

//go:generate moq -out config_moq.go . ConfigManager
type ConfigManager interface {
	ReadStorageStrategy(ctx context.Context, rt providers.ResourceType, tier string) (*StrategyConfig, error)
}

type ConfigMapConfigManager struct {
	configMapName      string
	configMapNamespace string
	client             client.Client
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

func NewDefaultConfigManager(client client.Client) *ConfigMapConfigManager {
	return NewConfigMapConfigManager(DefaultConfigMapName, DefaultConfigMapNamespace, client)
}

func (cmm *ConfigMapConfigManager) ReadStorageStrategy(ctx context.Context, rt providers.ResourceType, tier string) (*StrategyConfig, error) {
	stratCfg, err := cmm.getTierStrategyForProvider(ctx, string(rt), tier)
	if err != nil {
		return nil, errorUtil.Wrapf(err, "failed to get tier to strategy mapping for resource type %s", string(rt))
	}
	return stratCfg, nil
}

func (cmm *ConfigMapConfigManager) getTierStrategyForProvider(ctx context.Context, rt string, tier string) (*StrategyConfig, error) {
	cm, err := resources.GetConfigMapOrDefault(ctx, cmm.client, types.NamespacedName{Name: cmm.configMapName, Namespace: cmm.configMapNamespace}, BuildDefaultConfigMap(cmm.configMapName, cmm.configMapNamespace))
	if err != nil {
		return nil, errorUtil.Wrapf(err, "failed to get gcp strategy config map %s in namespace %s", cmm.configMapName, cmm.configMapNamespace)
	}
	rawStrategyMapping := cm.Data[rt]
	if rawStrategyMapping == "" {
		return nil, errorUtil.New(fmt.Sprintf("gcp strategy for resource type %s is not defined", rt))
	}
	var strategyMapping map[string]*StrategyConfig
	if err = json.Unmarshal([]byte(rawStrategyMapping), &strategyMapping); err != nil {
		return nil, errorUtil.Wrapf(err, "failed to unmarshal strategy mapping for resource type %s", rt)
	}
	strategyConfig := strategyMapping[tier]
	if strategyConfig == nil {
		return nil, errorUtil.New(fmt.Sprintf("no strategy found for deployment type %s and deployment tier %s", rt, tier))
	}
	if strategyConfig.ProjectID == "" {
		defaultProject, err := GetProjectFromStrategyOrDefault(ctx, cmm.client, strategyConfig)
		if err != nil {
			return nil, errorUtil.Wrap(err, "failed to get default gcp project")
		}
		strategyConfig.ProjectID = defaultProject
	}
	if strategyConfig.Region == "" {
		defaultRegion, err := GetRegionFromStrategyOrDefault(ctx, cmm.client, strategyConfig)
		if err != nil {
			return nil, errorUtil.Wrap(err, "failed to get default gcp region")
		}
		strategyConfig.Region = defaultRegion
	}
	return strategyConfig, nil
}

func BuildDefaultConfigMap(name, namespace string) *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: controllerruntime.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"blobstorage": `{"development": { "region": "", "projectID": "", "createStrategy": {}, "deleteStrategy": {} }, "production": { "region": "", "projectID": "", "createStrategy": {}, "deleteStrategy": {} }}`,
			"redis":       `{"development": { "region": "", "projectID": "", "createStrategy": {}, "deleteStrategy": {} }, "production": { "region": "", "projectID": "", "createStrategy": {}, "deleteStrategy": {} }}`,
			"postgres":    `{"development": { "region": "", "projectID": "", "createStrategy": {}, "deleteStrategy": {} }, "production": { "region": "", "projectID": "", "createStrategy": {}, "deleteStrategy": {} }}`,
			"_network":    `{"development": { "region": "", "projectID": "", "createStrategy": {}, "deleteStrategy": {} }, "production": { "region": "", "projectID": "", "createStrategy": {}, "deleteStrategy": {} }}`,
		},
	}
}

func GetRegionFromStrategyOrDefault(ctx context.Context, c client.Client, strategy *StrategyConfig) (string, error) {
	defaultRegion, err := getDefaultRegion(ctx, c)
	if err != nil {
		return "", errorUtil.Wrap(err, "failed to get default region")
	}
	region := strategy.Region
	if region == "" {
		region = defaultRegion
	}
	return region, nil
}

func getDefaultRegion(ctx context.Context, c client.Client) (string, error) {
	region, err := resources.GetGCPRegion(ctx, c)
	if err != nil {
		return "", errorUtil.Wrap(err, "failed to retrieve region from cluster")
	}
	if region == "" {
		return "", errorUtil.New("failed to retrieve region from cluster, region is not defined")
	}
	return region, nil
}

func GetProjectFromStrategyOrDefault(ctx context.Context, c client.Client, strategy *StrategyConfig) (string, error) {
	defaultProject, err := getDefaultProject(ctx, c)
	if err != nil {
		return "", errorUtil.Wrap(err, "failed to get default project")
	}
	project := strategy.ProjectID
	if project == "" {
		project = defaultProject
	}
	return project, nil
}

func getDefaultProject(ctx context.Context, c client.Client) (string, error) {
	defaultProject, err := resources.GetGCPProject(ctx, c)
	if err != nil {
		return "", errorUtil.Wrap(err, "failed to retrieve project from cluster")
	}
	if defaultProject == "" {
		return "", errorUtil.New("failed to retrieve project from cluster, project ID is not defined")
	}
	return defaultProject, nil
}

var _ ConfigManager = (*ConfigMapConfigManager)(nil)
