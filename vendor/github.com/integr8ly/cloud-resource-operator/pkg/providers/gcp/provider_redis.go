package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/integr8ly/cloud-resource-operator/pkg/annotations"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers/gcp/gcpiface"

	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	croType "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	errorUtil "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	redispb "google.golang.org/genproto/googleapis/cloud/redis/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	redisInstanceNameFormat = "projects/%s/locations/%s/instances/%s"
	redisMemorySizeGB       = 1
	redisParentFormat       = "projects/%s/locations/%s"
	redisProviderName       = "gcp-memorystore"
	redisVersion            = "REDIS_6_X"
)

type RedisProvider struct {
	Client            client.Client
	Logger            *logrus.Entry
	CredentialManager CredentialManager
	ConfigManager     ConfigManager
}

func NewGCPRedisProvider(client client.Client, logger *logrus.Entry) *RedisProvider {
	return &RedisProvider{
		Client:            client,
		Logger:            logger.WithFields(logrus.Fields{"provider": redisProviderName}),
		CredentialManager: NewCredentialMinterCredentialManager(client),
		ConfigManager:     NewDefaultConfigManager(client),
	}
}

func (p *RedisProvider) GetName() string {
	return redisProviderName
}

func (p *RedisProvider) SupportsStrategy(deploymentStrategy string) bool {
	return deploymentStrategy == providers.GCPDeploymentStrategy
}

func (p *RedisProvider) GetReconcileTime(r *v1alpha1.Redis) time.Duration {
	if r.Status.Phase != croType.PhaseComplete {
		return time.Second * 60
	}
	return resources.GetForcedReconcileTimeOrDefault(defaultReconcileTime)
}

var _ providers.RedisProvider = (*RedisProvider)(nil)

func (p *RedisProvider) CreateRedis(ctx context.Context, r *v1alpha1.Redis) (*providers.RedisCluster, croType.StatusMessage, error) {
	logger := p.Logger.WithField("action", "CreateRedis")
	logger.Infof("reconciling redis %s", r.Name)

	strategyConfig, err := p.getRedisStrategyConfig(ctx, r.Spec.Tier)
	if err != nil {
		statusMessage := "failed to retrieve redis strategy config"
		return nil, croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	if err := resources.CreateFinalizer(ctx, p.Client, r, DefaultFinalizer); err != nil {
		statusMessage := "failed to set finalizer"
		return nil, croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	creds, err := p.CredentialManager.ReconcileProviderCredentials(ctx, r.Namespace)
	if err != nil {
		statusMessage := fmt.Sprintf("failed to reconcile gcp redis provider credentials for redis instance %s", r.Name)
		return nil, croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	clientOption := option.WithCredentialsJSON(creds.ServiceAccountJson)
	networkManager, err := NewNetworkManager(ctx, strategyConfig.ProjectID, clientOption, p.Client, logger)
	if err != nil {
		statusMessage := "failed to initialise network manager"
		return nil, croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	redisClient, err := gcpiface.NewRedisAPI(ctx, clientOption)
	if err != nil {
		statusMessage := "could not initialise redis client"
		return nil, croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	return p.createRedisInstance(ctx, networkManager, redisClient, strategyConfig, r)
}

func (p *RedisProvider) createRedisInstance(ctx context.Context, networkManager NetworkManager, redisClient gcpiface.RedisAPI, strategyConfig *StrategyConfig, r *v1alpha1.Redis) (*providers.RedisCluster, croType.StatusMessage, error) {
	ipRangeCidr, err := networkManager.ReconcileNetworkProviderConfig(ctx, p.ConfigManager, r.Spec.Tier)
	if err != nil {
		errMsg := "failed to reconcile network provider config"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	address, err := networkManager.CreateNetworkIpRange(ctx, ipRangeCidr)
	if err != nil {
		statusMessage := "failed to create network service"
		return nil, croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	if address == nil || address.GetStatus() == computepb.Address_RESERVING.String() {
		statusMessage := "network ip address range creation in progress"
		return nil, croType.StatusMessage(statusMessage), nil
	}
	p.Logger.Infof("created ip address range %s: %s/%d", address.GetName(), address.GetAddress(), address.GetPrefixLength())
	p.Logger.Infof("creating network service connection")
	service, err := networkManager.CreateNetworkService(ctx)
	if err != nil {
		statusMessage := "failed to create network service"
		return nil, croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	if service == nil {
		statusMessage := "network service connection creation in progress"
		return nil, croType.StatusMessage(statusMessage), nil
	}
	p.Logger.Infof("created network service connection %s", service.Service)

	createInstanceRequest, err := p.buildCreateInstanceRequest(ctx, r, strategyConfig, address)
	if err != nil {
		statusMessage := "failed to build create redis instance request"
		return nil, croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	foundInstance, err := redisClient.GetInstance(ctx, &redispb.GetInstanceRequest{Name: createInstanceRequest.Instance.Name})
	if err != nil && !resources.IsNotFoundError(err) {
		statusMessage := fmt.Sprintf("failed to fetch redis instance %s", createInstanceRequest.InstanceId)
		return nil, croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	if foundInstance == nil {
		_, err = redisClient.CreateInstance(ctx, createInstanceRequest)
		if err != nil {
			statusMessage := fmt.Sprintf("failed to create redis instance %s", createInstanceRequest.InstanceId)
			return nil, croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
		}
		annotations.Add(r, ResourceIdentifierAnnotation, createInstanceRequest.InstanceId)
		if err := p.Client.Update(ctx, r); err != nil {
			statusMessage := "failed to add annotation to redis cr"
			return nil, croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
		}
		statusMessage := "started creation of gcp redis instance"
		return nil, croType.StatusMessage(statusMessage), nil
	}
	if foundInstance.State != redispb.Instance_READY {
		statusMessage := fmt.Sprintf("creation in progress for redis instance %s", r.Name)
		return nil, croType.StatusMessage(statusMessage), nil
	}
	rdd := &providers.RedisDeploymentDetails{
		URI:  foundInstance.Host,
		Port: int64(foundInstance.Port),
	}
	statusMessage := fmt.Sprintf("successfully created gcp redis instance %s", r.Name)
	p.Logger.Info(statusMessage)
	return &providers.RedisCluster{DeploymentDetails: rdd}, croType.StatusMessage(statusMessage), nil
}

func (p *RedisProvider) DeleteRedis(ctx context.Context, r *v1alpha1.Redis) (croType.StatusMessage, error) {
	logger := p.Logger.WithField("action", "DeleteRedis")
	logger.Infof("reconciling delete redis %s", r.Name)

	strategyConfig, err := p.getRedisStrategyConfig(ctx, r.Spec.Tier)
	if err != nil {
		statusMessage := "failed to retrieve redis strategy config"
		return croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	creds, err := p.CredentialManager.ReconcileProviderCredentials(ctx, r.Namespace)
	if err != nil {
		statusMessage := fmt.Sprintf("failed to reconcile gcp redis provider credentials for redis instance %s", r.Name)
		return croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	isLastResource, err := resources.IsLastResource(ctx, p.Client)
	if err != nil {
		statusMessage := "failed to check if this cr is the last cr of type postgres and redis"
		return croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	clientOption := option.WithCredentialsJSON(creds.ServiceAccountJson)
	networkManager, err := NewNetworkManager(ctx, strategyConfig.ProjectID, clientOption, p.Client, logger)
	if err != nil {
		statusMessage := "failed to initialise network manager"
		return croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	redisClient, err := gcpiface.NewRedisAPI(ctx, clientOption)
	if err != nil {
		statusMessage := "could not initialise redis client"
		return croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	return p.deleteRedisInstance(ctx, networkManager, redisClient, strategyConfig, r, isLastResource)
}

func (p *RedisProvider) deleteRedisInstance(ctx context.Context, networkManager NetworkManager, redisClient gcpiface.RedisAPI, strategyConfig *StrategyConfig, r *v1alpha1.Redis, isLastResource bool) (croType.StatusMessage, error) {
	deleteInstanceRequest, err := p.buildDeleteInstanceRequest(r, strategyConfig)
	if err != nil {
		statusMessage := "failed to build delete redis instance request"
		return croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	foundInstance, err := redisClient.GetInstance(ctx, &redispb.GetInstanceRequest{Name: deleteInstanceRequest.Name})
	if err != nil && !resources.IsNotFoundError(err) {
		statusMessage := fmt.Sprintf("failed to fetch redis instance %s", deleteInstanceRequest.Name)
		return croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	if foundInstance != nil {
		if foundInstance.State == redispb.Instance_DELETING {
			statusMessage := fmt.Sprintf("deletion in progress for redis instance %s", r.Name)
			return croType.StatusMessage(statusMessage), nil
		}
		_, err = redisClient.DeleteInstance(ctx, deleteInstanceRequest)
		if err != nil {
			statusMessage := fmt.Sprintf("failed to delete redis instance %s", r.Name)
			return croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
		}
		statusMessage := fmt.Sprintf("delete detected, redis instance %s started", r.Name)
		return croType.StatusMessage(statusMessage), nil
	}

	// remove networking components
	if isLastResource {
		if err = networkManager.DeleteNetworkPeering(ctx); err != nil {
			statusMessage := "failed to delete cluster network peering"
			return croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
		}
		if err = networkManager.DeleteNetworkService(ctx); err != nil {
			statusMessage := "failed to delete network service"
			return croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
		}
		if err = networkManager.DeleteNetworkIpRange(ctx); err != nil {
			statusMessage := "failed to delete network ip range"
			return croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
		}
		if exist, err := networkManager.ComponentsExist(ctx); err != nil || exist {
			if exist {
				statusMessage := "network component deletion in progress"
				return croType.StatusMessage(statusMessage), nil
			}
			statusMessage := "failed to check if components exist"
			return croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
		}
	}

	// remove the finalizer added by the provider
	resources.RemoveFinalizer(&r.ObjectMeta, DefaultFinalizer)
	if err = p.Client.Update(ctx, r); err != nil {
		statusMessage := fmt.Sprintf("failed to update instance %s as part of finalizer reconcile", r.Name)
		return croType.StatusMessage(statusMessage), errorUtil.Wrapf(err, statusMessage)
	}
	statusMessage := fmt.Sprintf("successfully deleted gcp redis instance %s", r.Name)
	return croType.StatusMessage(statusMessage), nil
}

func (p *RedisProvider) getRedisInstances(ctx context.Context, redisClient gcpiface.RedisAPI, projectID, region string) ([]*redispb.Instance, error) {
	request := redispb.ListInstancesRequest{
		Parent: fmt.Sprintf("projects/%v/locations/%v", projectID, region),
	}
	instances, err := redisClient.ListInstances(ctx, &request)
	if err != nil {
		return nil, err
	}
	return instances, nil
}

func (p *RedisProvider) getRedisStrategyConfig(ctx context.Context, tier string) (*StrategyConfig, error) {
	strategyConfig, err := p.ConfigManager.ReadStorageStrategy(ctx, providers.RedisResourceType, tier)
	if err != nil {
		errMsg := "failed to read gcp strategy config"
		return nil, errorUtil.Wrap(err, errMsg)
	}
	defaultProject, err := GetProjectFromStrategyOrDefault(ctx, p.Client, strategyConfig)
	if err != nil {
		errMsg := "failed to get default gcp project"
		return nil, errorUtil.Wrap(err, errMsg)
	}
	if strategyConfig.ProjectID == "" {
		p.Logger.Debugf("project not set in deployment strategy configuration, using default project %s", defaultProject)
		strategyConfig.ProjectID = defaultProject
	}
	defaultRegion, err := GetRegionFromStrategyOrDefault(ctx, p.Client, strategyConfig)
	if err != nil {
		errMsg := "failed to get default gcp region"
		return nil, errorUtil.Wrap(err, errMsg)
	}
	if strategyConfig.Region == "" {
		p.Logger.Debugf("region not set in deployment strategy configuration, using default region %s", defaultRegion)
		strategyConfig.Region = defaultRegion
	}
	return strategyConfig, nil
}

func (p *RedisProvider) buildCreateInstanceRequest(ctx context.Context, r *v1alpha1.Redis, strategyConfig *StrategyConfig, address *computepb.Address) (*redispb.CreateInstanceRequest, error) {
	createInstanceRequest := &redispb.CreateInstanceRequest{}
	if err := json.Unmarshal(strategyConfig.CreateStrategy, createInstanceRequest); err != nil {
		return nil, errorUtil.Wrap(err, "failed to unmarshal gcp redis create strategy")
	}
	if createInstanceRequest.Parent == "" {
		createInstanceRequest.Parent = fmt.Sprintf(redisParentFormat, strategyConfig.ProjectID, strategyConfig.Region)
	}
	if createInstanceRequest.InstanceId == "" {
		instanceID, err := resources.BuildInfraNameFromObject(ctx, p.Client, r.ObjectMeta, defaultGcpIdentifierLength)
		if err != nil {
			return nil, errorUtil.Wrap(err, "failed to build redis instance id from object")
		}
		createInstanceRequest.InstanceId = instanceID
	}
	defaultInstance := &redispb.Instance{
		Name:              fmt.Sprintf(redisInstanceNameFormat, strategyConfig.ProjectID, strategyConfig.Region, createInstanceRequest.InstanceId),
		Tier:              redispb.Instance_STANDARD_HA,
		ReadReplicasMode:  redispb.Instance_READ_REPLICAS_DISABLED,
		MemorySizeGb:      redisMemorySizeGB,
		AuthorizedNetwork: strings.Split(address.GetNetwork(), "v1/")[1],
		ConnectMode:       redispb.Instance_PRIVATE_SERVICE_ACCESS,
		ReservedIpRange:   address.GetName(),
		RedisVersion:      redisVersion,
	}
	if createInstanceRequest.Instance == nil {
		createInstanceRequest.Instance = defaultInstance
		return createInstanceRequest, nil
	}
	if createInstanceRequest.Instance.Name == "" {
		createInstanceRequest.Instance.Name = defaultInstance.Name
	}
	if createInstanceRequest.Instance.Tier == 0 {
		createInstanceRequest.Instance.Tier = defaultInstance.Tier
	}
	if createInstanceRequest.Instance.ReadReplicasMode == 0 {
		createInstanceRequest.Instance.ReadReplicasMode = defaultInstance.ReadReplicasMode
	}
	if createInstanceRequest.Instance.MemorySizeGb == 0 {
		createInstanceRequest.Instance.MemorySizeGb = defaultInstance.MemorySizeGb
	}
	if createInstanceRequest.Instance.AuthorizedNetwork == "" {
		createInstanceRequest.Instance.AuthorizedNetwork = defaultInstance.AuthorizedNetwork
	}
	if createInstanceRequest.Instance.ConnectMode == 0 {
		createInstanceRequest.Instance.ConnectMode = defaultInstance.ConnectMode
	}
	if createInstanceRequest.Instance.ReservedIpRange == "" {
		createInstanceRequest.Instance.ReservedIpRange = defaultInstance.ReservedIpRange
	}
	if createInstanceRequest.Instance.RedisVersion == "" {
		createInstanceRequest.Instance.RedisVersion = defaultInstance.RedisVersion
	}
	return createInstanceRequest, nil
}

func (p *RedisProvider) buildDeleteInstanceRequest(r *v1alpha1.Redis, strategyConfig *StrategyConfig) (*redispb.DeleteInstanceRequest, error) {
	deleteInstanceRequest := &redispb.DeleteInstanceRequest{}
	if err := json.Unmarshal(strategyConfig.DeleteStrategy, deleteInstanceRequest); err != nil {
		return nil, errorUtil.Wrap(err, "failed to unmarshal gcp redis delete strategy")
	}
	if deleteInstanceRequest.Name == "" {
		resourceID := annotations.Get(r, ResourceIdentifierAnnotation)
		if resourceID == "" {
			return nil, fmt.Errorf("failed to find redis instance name from annotations")
		}
		deleteInstanceRequest.Name = fmt.Sprintf(redisInstanceNameFormat, strategyConfig.ProjectID, strategyConfig.Region, resourceID)
	}
	return deleteInstanceRequest, nil
}
