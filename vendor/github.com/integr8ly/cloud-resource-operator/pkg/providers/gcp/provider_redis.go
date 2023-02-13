package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	TCPPinger         resources.ConnectionTester
}

func NewGCPRedisProvider(client client.Client, logger *logrus.Entry) *RedisProvider {
	return &RedisProvider{
		Client:            client,
		Logger:            logger.WithFields(logrus.Fields{"provider": redisProviderName}),
		CredentialManager: NewCredentialMinterCredentialManager(client),
		ConfigManager:     NewDefaultConfigManager(client),
		TCPPinger:         resources.NewConnectionTestManager(),
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
	redisClient, err := gcpiface.NewRedisAPI(ctx, clientOption, logger)
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
	_, err = networkManager.CreateNetworkService(ctx)
	if err != nil {
		statusMessage := "failed to create network service"
		return nil, croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	createInstanceRequest, err := p.buildCreateInstanceRequest(ctx, r, strategyConfig, address)
	if err != nil {
		statusMessage := "failed to build gcp create redis instance request"
		return nil, croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	foundInstance, err := redisClient.GetInstance(ctx, &redispb.GetInstanceRequest{Name: createInstanceRequest.Instance.Name})
	if err != nil && !resources.IsNotFoundError(err) {
		statusMessage := fmt.Sprintf("failed to fetch gcp redis instance %s", createInstanceRequest.Instance.Name)
		return nil, croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	defer p.exposeRedisInstanceMetrics(ctx, r, foundInstance)
	if foundInstance == nil {
		_, err = redisClient.CreateInstance(ctx, createInstanceRequest)
		if err != nil {
			statusMessage := fmt.Sprintf("failed to create gcp redis instance %s", createInstanceRequest.Instance.Name)
			return nil, croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
		}
		_, err = controllerutil.CreateOrUpdate(ctx, p.Client, r, func() error {
			annotations.Add(r, ResourceIdentifierAnnotation, createInstanceRequest.InstanceId)
			return nil
		})
		if err != nil {
			statusMessage := "failed to add annotation to redis cr"
			return nil, croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
		}
		statusMessage := fmt.Sprintf("started creation of gcp redis instance %s", createInstanceRequest.Instance.Name)
		return nil, croType.StatusMessage(statusMessage), nil
	}
	if foundInstance.State != redispb.Instance_READY {
		statusMessage := fmt.Sprintf("gcp redis instance %s is not ready yet, current state is %s", createInstanceRequest.Instance.Name, foundInstance.State.String())
		return nil, croType.StatusMessage(statusMessage), nil
	}
	maintenanceWindowEnabled, err := resources.VerifyRedisMaintenanceWindow(ctx, p.Client, r.Namespace, r.Name)
	if err != nil {
		statusMessage := "failed to verify if redis updates are allowed"
		return nil, croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	if maintenanceWindowEnabled {
		if updateInstanceRequest := p.buildUpdateInstanceRequest(createInstanceRequest.Instance, foundInstance); updateInstanceRequest != nil {
			_, err = redisClient.UpdateInstance(ctx, updateInstanceRequest)
			if err != nil {
				statusMessage := fmt.Sprintf("failed to update gcp redis instance %s", createInstanceRequest.Instance.Name)
				return nil, croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
			}
		}
		if upgradeInstanceRequest := p.buildUpgradeInstanceRequest(createInstanceRequest.Instance, foundInstance); upgradeInstanceRequest != nil {
			_, err = redisClient.UpgradeInstance(ctx, upgradeInstanceRequest)
			if err != nil {
				statusMessage := fmt.Sprintf("failed to upgrade gcp redis instance %s", createInstanceRequest.Instance.Name)
				return nil, croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
			}
		}
		_, err = controllerutil.CreateOrUpdate(ctx, p.Client, r, func() error {
			r.Spec.MaintenanceWindow = false
			return nil
		})
		if err != nil {
			statusMessage := "failed to set redis maintenance window to false"
			return nil, croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
		}
	}
	rdd := &providers.RedisDeploymentDetails{
		URI:  foundInstance.Host,
		Port: int64(foundInstance.Port),
	}
	statusMessage := fmt.Sprintf("successfully reconciled gcp redis instance %s", createInstanceRequest.Instance.Name)
	p.Logger.Info(statusMessage)
	return &providers.RedisCluster{DeploymentDetails: rdd}, croType.StatusMessage(statusMessage), nil
}

func (p *RedisProvider) DeleteRedis(ctx context.Context, r *v1alpha1.Redis) (croType.StatusMessage, error) {
	logger := p.Logger.WithField("action", "DeleteRedis")
	logger.Infof("reconciling delete redis %s", r.Name)
	p.setRedisDeletionTimestampMetric(ctx, r)
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
	redisClient, err := gcpiface.NewRedisAPI(ctx, clientOption, logger)
	if err != nil {
		statusMessage := "could not initialise redis client"
		return croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	return p.deleteRedisInstance(ctx, networkManager, redisClient, strategyConfig, r, isLastResource)
}

func (p *RedisProvider) deleteRedisInstance(ctx context.Context, networkManager NetworkManager, redisClient gcpiface.RedisAPI, strategyConfig *StrategyConfig, r *v1alpha1.Redis, isLastResource bool) (croType.StatusMessage, error) {
	deleteInstanceRequest, err := p.buildDeleteInstanceRequest(r, strategyConfig)
	if err != nil {
		statusMessage := "failed to build delete gcp redis instance request"
		return croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	foundInstance, err := redisClient.GetInstance(ctx, &redispb.GetInstanceRequest{Name: deleteInstanceRequest.Name})
	if err != nil && !resources.IsNotFoundError(err) {
		statusMessage := fmt.Sprintf("failed to fetch gcp redis instance %s", deleteInstanceRequest.Name)
		return croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
	}
	if foundInstance != nil {
		p.exposeRedisInstanceMetrics(ctx, r, foundInstance)
		if foundInstance.State == redispb.Instance_DELETING {
			statusMessage := fmt.Sprintf("deletion in progress for gcp redis instance %s", deleteInstanceRequest.Name)
			return croType.StatusMessage(statusMessage), nil
		}
		_, err = redisClient.DeleteInstance(ctx, deleteInstanceRequest)
		if err != nil {
			statusMessage := fmt.Sprintf("failed to delete gcp redis instance %s", deleteInstanceRequest.Name)
			return croType.StatusMessage(statusMessage), errorUtil.Wrap(err, statusMessage)
		}
		statusMessage := fmt.Sprintf("delete detected, gcp redis instance %s deletion started", deleteInstanceRequest.Name)
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
	statusMessage := fmt.Sprintf("successfully deleted gcp redis instance %s", deleteInstanceRequest.Name)
	p.Logger.Info(statusMessage)
	return croType.StatusMessage(statusMessage), nil
}

func (p *RedisProvider) exposeRedisInstanceMetrics(ctx context.Context, r *v1alpha1.Redis, instance *redispb.Instance) {
	if instance == nil {
		return
	}
	instanceName := annotations.Get(r, ResourceIdentifierAnnotation)
	if instanceName == "" {
		p.Logger.Errorf("failed to find %s annotation while exposing metrics for redis instance %s", ResourceIdentifierAnnotation, instanceName)
	}
	clusterID, err := resources.GetClusterID(ctx, p.Client)
	if err != nil {
		p.Logger.Errorf("failed to get cluster id while exposing metrics for redis instance %s", instanceName)
		return
	}
	genericLabels := resources.BuildGenericMetricLabels(r.ObjectMeta, clusterID, instanceName, redisProviderName)
	instanceState := instance.State.String()
	infoLabels := resources.BuildInfoMetricLabels(r.ObjectMeta, instanceState, clusterID, instanceName, redisProviderName)
	resources.SetMetricCurrentTime(resources.DefaultRedisInfoMetricName, infoLabels)
	// a single metric should be exposed for each possible status phase
	// the value of the metric should be 1.0 when the resource is in that phase
	// the value of the metric should be 0.0 when the resource is not in that phase
	for _, phase := range []croType.StatusPhase{croType.PhaseFailed, croType.PhaseDeleteInProgress, croType.PhasePaused, croType.PhaseComplete, croType.PhaseInProgress} {
		labelsFailed := resources.BuildStatusMetricsLabels(r.ObjectMeta, clusterID, instanceName, redisProviderName, phase)
		resources.SetMetric(resources.DefaultRedisStatusMetricName, labelsFailed, resources.Btof64(r.Status.Phase == phase))
	}
	// set availability metric, based on the status flag on the memorystore redis instance in gcp
	// the value of the metric should be 0 when the instance state is unhealthy
	// the value of the metric should be 1 when the instance state is healthy
	// more details on possible state values here: https://pkg.go.dev/cloud.google.com/go/redis@v1.10.0/apiv1beta1/redispb#Instance_State
	var instanceHealthy float64
	var instanceConnectable float64
	if resources.Contains(healthyRedisInstanceStates(), instanceState) {
		instanceHealthy = 1
		if success := p.TCPPinger.TCPConnection(instance.Host, int(instance.Port)); success {
			instanceConnectable = 1
		}
	}
	resources.SetMetric(resources.DefaultRedisAvailMetricName, genericLabels, instanceHealthy)
	resources.SetMetric(resources.DefaultRedisConnectionMetricName, genericLabels, instanceConnectable)
}

func healthyRedisInstanceStates() []string {
	return []string{
		redispb.Instance_CREATING.String(),
		redispb.Instance_READY.String(),
		redispb.Instance_UPDATING.String(),
		redispb.Instance_DELETING.String(),
	}
}

// set metrics about the redis instance being deleted
// works in a similar way to kube_pod_deletion_timestamp
// https://github.com/kubernetes/kube-state-metrics/blob/0bfc2981f9c281c78e33052abdc2d621630562b9/internal/store/pod.go#L200-L218
func (p *RedisProvider) setRedisDeletionTimestampMetric(ctx context.Context, r *v1alpha1.Redis) {
	if r.DeletionTimestamp != nil && !r.DeletionTimestamp.IsZero() {
		instanceName := annotations.Get(r, ResourceIdentifierAnnotation)
		if instanceName == "" {
			p.Logger.Errorf("failed to find %s annotation while exposing metrics for redis cr %s", ResourceIdentifierAnnotation, r.Name)
			return
		}
		clusterID, err := resources.GetClusterID(ctx, p.Client)
		if err != nil {
			p.Logger.Errorf("failed to get cluster id while exposing metrics for redis instance %s", instanceName)
			return
		}
		labels := resources.BuildStatusMetricsLabels(r.ObjectMeta, clusterID, instanceName, redisProviderName, r.Status.Phase)
		resources.SetMetric(resources.DefaultRedisDeletionMetricName, labels, float64(r.DeletionTimestamp.Unix()))
	}
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
			return nil, errorUtil.Wrap(err, "failed to build gcp redis instance id from object")
		}
		createInstanceRequest.InstanceId = instanceID
	}
	tags, err := buildDefaultRedisTags(ctx, p.Client, r)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to build gcp redis instance tags")
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
		Labels:            tags,
	}
	if createInstanceRequest.Instance == nil {
		createInstanceRequest.Instance = defaultInstance
		return createInstanceRequest, nil
	}
	if createInstanceRequest.Instance.Labels == nil {
		createInstanceRequest.Instance.Labels = map[string]string{}
	}
	for key, value := range tags {
		createInstanceRequest.Instance.Labels[key] = value
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
			return nil, fmt.Errorf("failed to find gcp redis instance name from annotations")
		}
		deleteInstanceRequest.Name = fmt.Sprintf(redisInstanceNameFormat, strategyConfig.ProjectID, strategyConfig.Region, resourceID)
	}
	return deleteInstanceRequest, nil
}

func (p *RedisProvider) buildUpdateInstanceRequest(instanceConfig *redispb.Instance, instance *redispb.Instance) *redispb.UpdateInstanceRequest {
	updateFound := false
	updateInstanceReq := &redispb.UpdateInstanceRequest{
		Instance: &redispb.Instance{
			Name: instance.Name,
		},
		UpdateMask: &fieldmaskpb.FieldMask{},
	}
	if instance.MemorySizeGb != instanceConfig.MemorySizeGb {
		updateFound = true
		updateInstanceReq.Instance.MemorySizeGb = instanceConfig.MemorySizeGb
		updateInstanceReq.UpdateMask.Paths = append(updateInstanceReq.UpdateMask.Paths, "memory_size_gb")
	}
	if !reflect.DeepEqual(instance.Labels, instanceConfig.Labels) {
		updateFound = true
		updateInstanceReq.Instance.Labels = instanceConfig.Labels
		updateInstanceReq.UpdateMask.Paths = append(updateInstanceReq.UpdateMask.Paths, "labels")
	}
	if !reflect.DeepEqual(instance.RedisConfigs, instanceConfig.RedisConfigs) {
		updateFound = true
		updateInstanceReq.Instance.RedisConfigs = instanceConfig.RedisConfigs
		updateInstanceReq.UpdateMask.Paths = append(updateInstanceReq.UpdateMask.Paths, "redis_configs")
	}
	if isMaintenancePolicyOutdated(instance.MaintenancePolicy, instanceConfig.MaintenancePolicy) {
		updateFound = true
		updateInstanceReq.Instance.MaintenancePolicy = instanceConfig.MaintenancePolicy
		updateInstanceReq.UpdateMask.Paths = append(updateInstanceReq.UpdateMask.Paths, "maintenance_policy")
	}
	if !updateFound {
		return nil
	}
	return updateInstanceReq
}

func isMaintenancePolicyOutdated(a *redispb.MaintenancePolicy, b *redispb.MaintenancePolicy) bool {
	if a == nil && b == nil {
		return false
	}
	if (a == nil && b != nil) || (a != nil && b == nil) {
		return true
	}
	return !reflect.DeepEqual(a.WeeklyMaintenanceWindow, b.WeeklyMaintenanceWindow)
}

func (p *RedisProvider) buildUpgradeInstanceRequest(instanceConfig *redispb.Instance, instance *redispb.Instance) *redispb.UpgradeInstanceRequest {
	if instance.RedisVersion != instanceConfig.RedisVersion {
		return &redispb.UpgradeInstanceRequest{
			Name:         instance.Name,
			RedisVersion: instanceConfig.RedisVersion,
		}
	}
	return nil
}
