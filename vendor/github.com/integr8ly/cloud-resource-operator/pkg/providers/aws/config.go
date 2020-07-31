package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	"github.com/integr8ly/cloud-resource-operator/pkg/resources"

	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	errorUtil "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultConfigMapName = "cloud-resources-aws-strategies"

	DefaultFinalizer = "finalizers.cloud-resources-operator.integreatly.org"

	defaultReconcileTime = time.Second * 30

	resourceIdentifierAnnotation = "resourceIdentifier"
)

//DefaultConfigMapNamespace is the default namespace that Configmaps will be created in
var DefaultConfigMapNamespace, _ = k8sutil.GetWatchNamespace()

type StrategyConfig struct {
	Region         string          `json:"region"`
	CreateStrategy json.RawMessage `json:"createStrategy"`
	DeleteStrategy json.RawMessage `json:"deleteStrategy"`
}

//go:generate moq -out config_moq.go . ConfigManager
type ConfigManager interface {
	ReadStorageStrategy(ctx context.Context, rt providers.ResourceType, tier string) (*StrategyConfig, error)
}

var _ ConfigManager = (*ConfigMapConfigManager)(nil)

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

func (m *ConfigMapConfigManager) getTierStrategyForProvider(ctx context.Context, rt string, tier string) (*StrategyConfig, error) {
	cm, err := resources.GetConfigMapOrDefault(ctx, m.client, types.NamespacedName{Name: m.configMapName, Namespace: m.configMapNamespace}, BuildDefaultConfigMap(m.configMapName, m.configMapNamespace))
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
	if strategyMapping[tier] == nil {
		return nil, errorUtil.New(fmt.Sprintf("no strategy found for deployment type %s and deployment tier %s", rt, tier))
	}
	return strategyMapping[tier], nil
}

func BuildDefaultConfigMap(name, namespace string) *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: controllerruntime.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"blobstorage": "{\"development\": { \"region\": \"\", \"createStrategy\": {}, \"deleteStrategy\": {} }, \"production\": { \"region\": \"\", \"createStrategy\": {}, \"deleteStrategy\": {} }}",
			"redis":       "{\"development\": { \"region\": \"\", \"createStrategy\": {}, \"deleteStrategy\": {} }, \"production\": { \"region\": \"\", \"createStrategy\": {}, \"deleteStrategy\": {} }}",
			"postgres":    "{\"development\": { \"region\": \"\", \"createStrategy\": {}, \"deleteStrategy\": {} }, \"production\": { \"region\": \"\", \"createStrategy\": {}, \"deleteStrategy\": {} }}",
		},
	}
}

// BuildSubnetGroupName builds and returns an id used for infra resources
func BuildInfraName(ctx context.Context, c client.Client, postfix string, n int) (string, error) {
	// get cluster id
	clusterID, err := resources.GetClusterID(ctx, c)
	if err != nil {
		return "", errorUtil.Wrap(err, "error getting clusterID")
	}
	return resources.ShortenString(fmt.Sprintf("%s-%s", clusterID, postfix), n), nil
}

func BuildInfraNameFromObject(ctx context.Context, c client.Client, om controllerruntime.ObjectMeta, n int) (string, error) {
	clusterID, err := resources.GetClusterID(ctx, c)
	if err != nil {
		return "", errorUtil.Wrap(err, "failed to retrieve cluster identifier")
	}
	return resources.ShortenString(fmt.Sprintf("%s-%s-%s", clusterID, om.Namespace, om.Name), n), nil
}

func buildTimestampedInfraNameFromObject(ctx context.Context, c client.Client, om controllerruntime.ObjectMeta, n int) (string, error) {
	clusterID, err := resources.GetClusterID(ctx, c)
	if err != nil {
		return "", errorUtil.Wrap(err, "failed to retrieve timestamped cluster identifier")
	}
	curTime := time.Now().Unix()
	return resources.ShortenString(fmt.Sprintf("%s-%s-%s-%d", clusterID, om.Namespace, om.Name, curTime), n), nil
}

func BuildTimestampedInfraNameFromObjectCreation(ctx context.Context, c client.Client, om controllerruntime.ObjectMeta, n int) (string, error) {
	clusterID, err := resources.GetClusterID(ctx, c)
	if err != nil {
		return "", errorUtil.Wrap(err, "failed to retrieve timestamped cluster identifier")
	}
	return resources.ShortenString(fmt.Sprintf("%s-%s-%s-%s", clusterID, om.Namespace, om.Name, om.GetObjectMeta().GetCreationTimestamp()), n), nil
}

func CreateSessionFromStrategy(ctx context.Context, c client.Client, keyID, secretKey string, strategy *StrategyConfig) (*session.Session, error) {
	region, err := GetRegionFromStrategyOrDefault(ctx, c, strategy)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to get region from strategy while creating aws session")
	}
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(keyID, secretKey, ""),
	})
	if err != nil {
		return nil, errorUtil.Wrapf(err, "failed to create aws session from strategy, region=%s keyID=%s", region, keyID)
	}
	return sess, nil
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
	region, err := resources.GetAWSRegion(ctx, c)
	if err != nil {
		return "", errorUtil.Wrap(err, "failed to retrieve region from cluster")
	}
	if region == "" {
		return "", errorUtil.New("failed to retrieve region from cluster, region is not defined")
	}
	return region, nil
}
