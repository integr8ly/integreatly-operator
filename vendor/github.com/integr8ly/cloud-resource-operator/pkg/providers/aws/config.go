package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsCredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/integr8ly/cloud-resource-operator/internal/k8sutil"

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

	DefaultFinalizer = "cloud-resources-operator.integreatly.org/finalizers"

	defaultReconcileTime = time.Second * 30

	ResourceIdentifierAnnotation = "resourceIdentifier"
)

// DefaultConfigMapNamespace is the default namespace that Configmaps will be created in
var DefaultConfigMapNamespace, _ = k8sutil.GetWatchNamespace()

/*
StrategyConfig provides the configuration necessary to create/modify/delete aws resources
Region -> required to create aws sessions, if no region is provided we default to cluster infrastructure
CreateStrategy -> maps to resource specific create parameters, uses as a source of truth to the state we expect the resource to be in
DeleteStrategy -> maps to resource specific delete parameters
*/
type StrategyConfig struct {
	Region         string          `json:"region"`
	CreateStrategy json.RawMessage `json:"createStrategy"`
	DeleteStrategy json.RawMessage `json:"deleteStrategy"`
	ServiceUpdates json.RawMessage `json:"serviceUpdates"`
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
			"blobstorage": "{\"development\": { \"region\": \"\", \"_network\": \"\", \"createStrategy\": {}, \"deleteStrategy\": {} }, \"production\": { \"region\": \"\", \"_network\": \"\", \"createStrategy\": {}, \"deleteStrategy\": {} }}",
			"redis":       "{\"development\": { \"region\": \"\", \"_network\": \"\", \"createStrategy\": {}, \"deleteStrategy\": {} }, \"production\": { \"region\": \"\", \"_network\": \"\",\"createStrategy\": {}, \"deleteStrategy\": {} }}",
			"postgres":    "{\"development\": { \"region\": \"\", \"_network\": \"\", \"createStrategy\": {}, \"deleteStrategy\": {} }, \"production\": { \"region\": \"\", \"_network\": \"\",\"createStrategy\": {}, \"deleteStrategy\": {} }}",
			"_network":    "{\"development\": { \"region\": \"\", \"_network\": \"\", \"createStrategy\": {}, \"deleteStrategy\": {} }, \"production\": { \"region\": \"\", \"_network\": \"\",\"createStrategy\": {}, \"deleteStrategy\": {} }}",
		},
	}
}

func CreateSessionFromStrategy(ctx context.Context, c client.Client, credentials *Credentials, strategy *StrategyConfig) (*session.Session, error) {
	region, err := GetRegionFromStrategyOrDefault(ctx, c, strategy)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to get region from strategy while creating aws session")
	}

	awsConfig := aws.Config{
		Region: aws.String(region),
	}
	// Check if STS credentials are passed
	if len(credentials.RoleArn) > 0 {
		// If running locally and STS role to assume is created, assume this role locally
		// Local IAM user must be a principle in the role created with the sts:AssumeRole action
		// Otherwise assume running in a pod in STS cluster
		if k8sutil.IsRunModeLocal() {
			sess := session.Must(session.NewSession(&awsConfig))
			awsConfig.Credentials = stscreds.NewCredentials(sess, credentials.RoleArn)
		} else {
			svc := sts.New(session.Must(session.NewSession(&awsConfig)))
			credentialsProvider := stscreds.NewWebIdentityRoleProviderWithOptions(svc, credentials.RoleArn, "Red-Hat-cloud-resources-operator", stscreds.FetchTokenPath(credentials.TokenFilePath))
			awsConfig.Credentials = awsCredentials.NewCredentials(credentialsProvider)
		}
	} else {
		awsConfig.Credentials = awsCredentials.NewStaticCredentials(credentials.AccessKeyID, credentials.SecretAccessKey, "")
	}
	sess := session.Must(session.NewSession(&awsConfig))
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
