package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"

	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/integr8ly/cloud-resource-operator/pkg/annotations"
	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"

	croType "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"

	"github.com/aws/aws-sdk-go/service/elasticache/elasticacheiface"
	"github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/integr8ly/cloud-resource-operator/pkg/providers"

	errorUtil "github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	redisProviderName = "aws-elasticache"
	// default create params
	defaultCacheNodeType = "cache.t2.micro"
	// required for at rest encryption, see https://docs.aws.amazon.com/AmazonElastiCache/latest/red-ug/at-rest-encryption.html
	defaultEngineVersion     = "3.2.6"
	defaultDescription       = "A Redis replication group"
	defaultNumCacheClusters  = 2
	defaultSnapshotRetention = 31
	defaultAtRestEncryption  = true
	// 3scale does not support in transit encryption (redis with tls)
	defaultInTransitEncryption = false
)

var _ providers.RedisProvider = (*RedisProvider)(nil)

// RedisProvider implementation for AWS Elasticache
type RedisProvider struct {
	Client            client.Client
	Logger            *logrus.Entry
	CredentialManager CredentialManager
	ConfigManager     ConfigManager
	CacheSvc          elasticacheiface.ElastiCacheAPI
	TCPPinger         ConnectionTester
}

func NewAWSRedisProvider(client client.Client, logger *logrus.Entry) *RedisProvider {
	return &RedisProvider{
		Client:            client,
		Logger:            logger.WithFields(logrus.Fields{"provider": redisProviderName}),
		CredentialManager: NewCredentialMinterCredentialManager(client),
		ConfigManager:     NewDefaultConfigMapConfigManager(client),
		TCPPinger:         NewConnectionTestManager(),
	}
}

func (p *RedisProvider) GetName() string {
	return redisProviderName
}

func (p *RedisProvider) SupportsStrategy(d string) bool {
	return d == providers.AWSDeploymentStrategy
}

func (p *RedisProvider) GetReconcileTime(r *v1alpha1.Redis) time.Duration {
	if r.Status.Phase != croType.PhaseComplete {
		return time.Second * 60
	}
	return resources.GetForcedReconcileTimeOrDefault(defaultReconcileTime)
}

// CreateRedis Create an Elasticache Replication Group from strategy config
func (p *RedisProvider) CreateRedis(ctx context.Context, r *v1alpha1.Redis) (*providers.RedisCluster, croType.StatusMessage, error) {
	// handle provider-specific finalizer
	if err := resources.CreateFinalizer(ctx, p.Client, r, DefaultFinalizer); err != nil {
		return nil, "failed to set finalizer", err
	}

	// info about the elasticache cluster to be created
	elasticacheCreateConfig, _, stratCfg, err := p.getElasticacheConfig(ctx, r)
	if err != nil {
		errMsg := fmt.Sprintf("failed to retrieve aws elasticache cluster config %s", r.Name)
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	// create the credentials to be used by the aws resource providers, not to be used by end-user
	providerCreds, err := p.CredentialManager.ReconcileProviderCredentials(ctx, r.Namespace)
	if err != nil {
		msg := "failed to reconcile elasticache credentials"
		return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}

	// setup aws elasticache cluster sdk session
	sess, err := CreateSessionFromStrategy(ctx, p.Client, providerCreds.AccessKeyID, providerCreds.SecretAccessKey, stratCfg)
	if err != nil {
		errMsg := "failed to create aws session to create elasticache replication group"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	// create the aws elasticache cluster
	return p.createElasticacheCluster(ctx, r, elasticache.New(sess), sts.New(sess), ec2.New(sess), elasticacheCreateConfig, stratCfg)
}

func (p *RedisProvider) createElasticacheCluster(ctx context.Context, r *v1alpha1.Redis, cacheSvc elasticacheiface.ElastiCacheAPI, stsSvc stsiface.STSAPI, ec2Svc ec2iface.EC2API, elasticacheConfig *elasticache.CreateReplicationGroupInput, stratCfg *StrategyConfig) (*providers.RedisCluster, types.StatusMessage, error) {
	// the aws access key can sometimes still not be registered in aws on first try, so loop
	rgs, err := getReplicationGroups(cacheSvc)
	if err != nil {
		// return nil error so this function can be requeueed
		errMsg := "error getting replication groups"
		logrus.Info(errMsg, err)
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	// setup vpc
	if err := p.configureElasticacheVpc(ctx, cacheSvc, ec2Svc); err != nil {
		errMsg := "error setting up resource vpc"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// setup security group
	if err := configureSecurityGroup(ctx, p.Client, ec2Svc); err != nil {
		errMsg := "error setting up security group"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// verify and build elasticache create config
	if err := p.buildElasticacheCreateStrategy(ctx, r, ec2Svc, elasticacheConfig); err != nil {
		errMsg := "failed to build and verify aws elasticache create strategy"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// check if the cluster has already been created
	var foundCache *elasticache.ReplicationGroup
	for _, c := range rgs {
		if *c.ReplicationGroupId == *elasticacheConfig.ReplicationGroupId {
			foundCache = c
			break
		}
	}

	// expose elasticache maintenance metric
	defer p.setRedisServiceMaintenanceMetric(ctx, cacheSvc, foundCache)

	// expose status metrics
	defer p.exposeRedisMetrics(ctx, r, foundCache)

	// expose a connection metric
	defer p.createElasticacheConnectionMetric(ctx, r, foundCache)

	// create elasticache cluster if it doesn't exist
	if foundCache == nil {
		if annotations.Has(r, resourceIdentifierAnnotation) {
			errMsg := fmt.Sprintf("Redis CR %s in %s namespace has %s annotation with value %s, but no corresponding Elasticache instance was found",
				r.Name, r.Namespace, resourceIdentifierAnnotation, r.ObjectMeta.Annotations[resourceIdentifierAnnotation])
			return nil, croType.StatusMessage(errMsg), fmt.Errorf(errMsg)
		}

		logrus.Info("creating elasticache cluster")
		if _, err := cacheSvc.CreateReplicationGroup(elasticacheConfig); err != nil {
			errMsg := fmt.Sprintf("error creating elasticache cluster %s", err)
			return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}

		annotations.Add(r, resourceIdentifierAnnotation, *elasticacheConfig.ReplicationGroupId)
		if err := p.Client.Update(ctx, r); err != nil {
			return nil, croType.StatusMessage("failed to add annotation"), err
		}
		return nil, "started elasticache provision", nil
	}

	// check elasticache phase
	if *foundCache.Status != "available" {
		logrus.Infof("found instance %s current status %s", *foundCache.ReplicationGroupId, *foundCache.Status)
		return nil, croType.StatusMessage(fmt.Sprintf("createReplicationGroup() in progress, current aws elasticache status is %s", *foundCache.Status)), nil
	}
	logrus.Infof("found existing elasticache instance %s", *foundCache.ReplicationGroupId)

	cacheClustersOutput, err := cacheSvc.DescribeCacheClusters(&elasticache.DescribeCacheClustersInput{})
	if err != nil {
		errMsg := "failed to describe clusters"
		return nil, croType.StatusMessage(fmt.Sprintf(errMsg)), errorUtil.Wrapf(err, errMsg)
	}
	var replicationGroupClusters []elasticache.CacheCluster
	for _, checkedCluster := range cacheClustersOutput.CacheClusters {
		if *checkedCluster.ReplicationGroupId == *foundCache.ReplicationGroupId {
			replicationGroupClusters = append(replicationGroupClusters, *checkedCluster)
		}
	}
	//check if found cluster and user strategy differs, and modify instance
	modifyInput := buildElasticacheUpdateStrategy(elasticacheConfig, foundCache, replicationGroupClusters)
	if modifyInput == nil {
		logrus.Infof("elasticache replication group %s is as expected", *foundCache.ReplicationGroupId)
	}
	if modifyInput != nil {
		logrus.Infof("%s differs from expected strategy, applying pending modifications :\n%s", *foundCache.ReplicationGroupId, modifyInput)
		if _, err := cacheSvc.ModifyReplicationGroup(modifyInput); err != nil {
			errMsg := "failed to modify elasticache cluster"
			return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
		logrus.Infof("set pending modifications to elasticache replication group %s", *foundCache.ReplicationGroupId)
	}

	// add tags to cache nodes
	cacheInstance := *foundCache.NodeGroups[0]
	if *cacheInstance.Status != "available" {
		logrus.Infof("elasticache node %s current status is %s", *cacheInstance.NodeGroupId, *cacheInstance.Status)
		return nil, croType.StatusMessage(fmt.Sprintf("cache node status not available, current status:  %s", *foundCache.Status)), nil
	}

	for _, cache := range cacheInstance.NodeGroupMembers {
		msg, err := p.TagElasticacheNode(ctx, cacheSvc, stsSvc, r, cache)
		if err != nil {
			errMsg := fmt.Sprintf("failed to add tags to elasticache: %s", msg)
			return nil, types.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
	}

	primaryEndpoint := foundCache.NodeGroups[0].PrimaryEndpoint
	rdd := &providers.RedisDeploymentDetails{
		URI:  *primaryEndpoint.Address,
		Port: *primaryEndpoint.Port,
	}

	// return secret information
	return &providers.RedisCluster{DeploymentDetails: rdd}, croType.StatusMessage(fmt.Sprintf("successfully created and tagged, aws elasticache status is %s", *foundCache.Status)), nil
}

// TagElasticacheNode Add Tags to AWS Elasticache
func (p *RedisProvider) TagElasticacheNode(ctx context.Context, cacheSvc elasticacheiface.ElastiCacheAPI, stsSvc stsiface.STSAPI, r *v1alpha1.Redis, cache *elasticache.NodeGroupMember) (types.StatusMessage, error) {
	logrus.Info("creating or updating tags on elasticache nodes and snapshots")

	// check the node to make sure it is available before applying the tag
	// this is needed as the cluster may be available while a node is not
	cacheClusterOutput, err := cacheSvc.DescribeCacheClusters(&elasticache.DescribeCacheClustersInput{
		CacheClusterId: cache.CacheClusterId,
	})
	if err != nil {
		errMsg := "failed to get cache cluster output"
		return types.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	clusterStatus := *cacheClusterOutput.CacheClusters[0].CacheClusterStatus
	if clusterStatus != "available" {
		errMsg := fmt.Sprintf("%s status is %s, skipping adding tags", *cache.CacheClusterId, clusterStatus)
		return types.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// get account identity
	identityInput := &sts.GetCallerIdentityInput{}
	id, err := stsSvc.GetCallerIdentity(identityInput)
	if err != nil {
		errMsg := "failed to get account identity"
		return types.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// trim availability zone to return cache region
	region := (*cache.PreferredAvailabilityZone)[:len(*cache.PreferredAvailabilityZone)-1]

	// build cluster arn
	// need arn in the following format arn:aws:elasticache:us-east-1:1234567890:cluster:my-mem-cluster
	arn := fmt.Sprintf("arn:aws:elasticache:%s:%s:cluster:%s", region, *id.Account, *cache.CacheClusterId)

	// set the tag values that will always be added
	organizationTag := resources.GetOrganizationTag()
	clusterID, err := resources.GetClusterID(ctx, p.Client)
	if err != nil {
		errMsg := "failed to get cluster id"
		return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}
	cacheTags := []*elasticache.Tag{
		{
			Key:   aws.String(organizationTag + "clusterID"),
			Value: aws.String(clusterID),
		},
		{
			Key:   aws.String(organizationTag + "resource-type"),
			Value: aws.String(r.Spec.Type),
		},
		{
			Key:   aws.String(organizationTag + "resource-name"),
			Value: aws.String(r.Name),
		},
	}

	// check is product name exists on cr
	if r.ObjectMeta.Labels["productName"] != "" {
		productTag := &elasticache.Tag{
			Key:   aws.String(organizationTag + "product-name"),
			Value: aws.String(r.ObjectMeta.Labels["productName"]),
		}
		cacheTags = append(cacheTags, productTag)
	}

	// add tags
	_, err = cacheSvc.AddTagsToResource(&elasticache.AddTagsToResourceInput{
		ResourceName: aws.String(arn),
		Tags:         cacheTags,
	})
	if err != nil {
		msg := "failed to add tags to aws elasticache :"
		return types.StatusMessage(msg), err
	}

	// if snapshots exist add tags to them
	inputDescribe := &elasticache.DescribeSnapshotsInput{
		CacheClusterId: aws.String(*cache.CacheClusterId),
	}

	// loop snapshots adding tags per found snapshot
	snapshotList, _ := cacheSvc.DescribeSnapshots(inputDescribe)
	if snapshotList.Snapshots != nil {
		for _, snapshot := range snapshotList.Snapshots {
			snapshotArn := fmt.Sprintf("arn:aws:elasticache:%s:%s:snapshot:%s", region, *id.Account, *snapshot.SnapshotName)
			logrus.Infof("Adding operator tags to snapshot : %s", *snapshot.SnapshotName)
			snapshotInput := &elasticache.AddTagsToResourceInput{
				ResourceName: aws.String(snapshotArn),
				Tags:         cacheTags,
			}
			_, err = cacheSvc.AddTagsToResource(snapshotInput)
			if err != nil {
				msg := "failed to add tags to aws elasticache snapshot"
				return types.StatusMessage(msg), err
			}
		}
	}

	logrus.Infof("successfully created or updated tags to elasticache node %s", *cache.CacheClusterId)
	return "successfully created and tagged", nil
}

//DeleteRedis Delete elasticache replication group
func (p *RedisProvider) DeleteRedis(ctx context.Context, r *v1alpha1.Redis) (croType.StatusMessage, error) {
	// resolve elasticache information for elasticache created by provider
	p.Logger.Info("getting cluster id from infrastructure for redis naming")
	elasticacheCreateConfig, elasticacheDeleteConfig, stratCfg, err := p.getElasticacheConfig(ctx, r)
	if err != nil {
		errMsg := fmt.Sprintf("failed to retrieve aws elasticache config for instance %s", r.Name)
		return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	// get provider aws creds so the elasticache cluster can be deleted
	providerCreds, err := p.CredentialManager.ReconcileProviderCredentials(ctx, r.Namespace)
	if err != nil {
		errMsg := "failed to reconcile aws provider credentials"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// setup aws elasticache cluster sdk session
	sess, err := CreateSessionFromStrategy(ctx, p.Client, providerCreds.AccessKeyID, providerCreds.SecretAccessKey, stratCfg)
	if err != nil {
		errMsg := "failed to create aws session to delete elasticache replication group"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// delete the elasticache cluster
	return p.deleteElasticacheCluster(ctx, elasticache.New(sess), elasticacheCreateConfig, elasticacheDeleteConfig, r)
}

func (p *RedisProvider) deleteElasticacheCluster(ctx context.Context, cacheSvc elasticacheiface.ElastiCacheAPI, elasticacheCreateConfig *elasticache.CreateReplicationGroupInput, elasticacheDeleteConfig *elasticache.DeleteReplicationGroupInput, r *v1alpha1.Redis) (croType.StatusMessage, error) {
	// the aws access key can sometimes still not be registered in aws on first try, so loop
	rgs, err := getReplicationGroups(cacheSvc)
	if err != nil {
		return "error getting replication groups", err
	}

	// check and verify delete config
	if err := p.buildElasticacheDeleteConfig(ctx, *r, elasticacheCreateConfig, elasticacheDeleteConfig); err != nil {
		errMsg := "failed to verify aws rds instance configuration"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// check if the cluster has already been deleted
	var foundCache *elasticache.ReplicationGroup
	for _, c := range rgs {
		if *c.ReplicationGroupId == *elasticacheCreateConfig.ReplicationGroupId {
			foundCache = c
			break
		}
	}

	// check if replication group does not exist and delete finalizer
	if foundCache == nil {
		// remove the finalizer added by the provider
		resources.RemoveFinalizer(&r.ObjectMeta, DefaultFinalizer)
		if err := p.Client.Update(ctx, r); err != nil {
			errMsg := "failed to update instance as part of finalizer reconcile"
			return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
		}
		return croType.StatusEmpty, nil
	}

	// set status metric
	p.exposeRedisMetrics(ctx, r, foundCache)

	// if status is not available return
	if *foundCache.Status != "available" {
		return croType.StatusMessage(fmt.Sprintf("delete detected, deleteReplicationGroup() in progress, current aws elasticache status is %s", *foundCache.Status)), nil
	}

	// delete elasticache cluster
	_, err = cacheSvc.DeleteReplicationGroup(elasticacheDeleteConfig)
	elasticacheErr, isAwsErr := err.(awserr.Error)
	if err != nil && (!isAwsErr || elasticacheErr.Code() != elasticache.ErrCodeReplicationGroupNotFoundFault) {
		errMsg := fmt.Sprintf("failed to delete elasticache cluster : %s", err)
		return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	return "delete detected, deleteReplicationGroup started", nil
}

// poll for replication groups
func getReplicationGroups(cacheSvc elasticacheiface.ElastiCacheAPI) ([]*elasticache.ReplicationGroup, error) {
	var rgs []*elasticache.ReplicationGroup
	err := wait.PollImmediate(time.Second*5, time.Minute*5, func() (done bool, err error) {
		listOutput, err := cacheSvc.DescribeReplicationGroups(&elasticache.DescribeReplicationGroupsInput{})
		if err != nil {
			return false, nil
		}
		rgs = listOutput.ReplicationGroups
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return rgs, nil
}

// getElasticacheConfig retrieves the elasticache config from the cloud-resources-aws-strategies configmap
func (p *RedisProvider) getElasticacheConfig(ctx context.Context, r *v1alpha1.Redis) (*elasticache.CreateReplicationGroupInput, *elasticache.DeleteReplicationGroupInput, *StrategyConfig, error) {
	stratCfg, err := p.ConfigManager.ReadStorageStrategy(ctx, providers.RedisResourceType, r.Spec.Tier)
	if err != nil {
		return nil, nil, nil, errorUtil.Wrap(err, "failed to read aws strategy config")
	}
	defRegion, err := GetRegionFromStrategyOrDefault(ctx, p.Client, stratCfg)
	if err != nil {
		return nil, nil, nil, errorUtil.Wrap(err, "failed to get default region")
	}
	if stratCfg.Region == "" {
		p.Logger.Debugf("region not set in deployment strategy configuration, using default region %s", defRegion)
		stratCfg.Region = defRegion
	}

	// unmarshal the elasticache cluster config
	elasticacheCreateConfig := &elasticache.CreateReplicationGroupInput{}
	if err := json.Unmarshal(stratCfg.CreateStrategy, elasticacheCreateConfig); err != nil {
		return nil, nil, nil, errorUtil.Wrap(err, "failed to unmarshal aws elasticache cluster configuration")
	}

	elasticacheDeleteConfig := &elasticache.DeleteReplicationGroupInput{}
	if err := json.Unmarshal(stratCfg.DeleteStrategy, elasticacheDeleteConfig); err != nil {
		return nil, nil, nil, errorUtil.Wrap(err, "failed to unmarshal aws elasticache cluster configuration")
	}
	return elasticacheCreateConfig, elasticacheDeleteConfig, stratCfg, nil
}

// checks found config vs user strategy for changes, if found returns a modify replication group
func buildElasticacheUpdateStrategy(elasticacheConfig *elasticache.CreateReplicationGroupInput, foundConfig *elasticache.ReplicationGroup, replicationGroupClusters []elasticache.CacheCluster) *elasticache.ModifyReplicationGroupInput {
	logrus.Infof("verifying that %s configuration is as expected", *foundConfig.ReplicationGroupId)
	updateFound := false
	modifyInput := &elasticache.ModifyReplicationGroupInput{}
	modifyInput.ReplicationGroupId = foundConfig.ReplicationGroupId

	if *elasticacheConfig.CacheNodeType != *foundConfig.CacheNodeType {
		modifyInput.CacheNodeType = elasticacheConfig.CacheNodeType
		updateFound = true
	}
	if *elasticacheConfig.SnapshotRetentionLimit != *foundConfig.SnapshotRetentionLimit {
		modifyInput.SnapshotRetentionLimit = elasticacheConfig.SnapshotRetentionLimit
		updateFound = true
	}

	for _, foundCacheCluster := range replicationGroupClusters {
		if elasticacheConfig.PreferredMaintenanceWindow != nil && *elasticacheConfig.PreferredMaintenanceWindow != *foundCacheCluster.PreferredMaintenanceWindow {
			modifyInput.PreferredMaintenanceWindow = elasticacheConfig.PreferredMaintenanceWindow
			updateFound = true
		}
		if elasticacheConfig.SnapshotWindow != nil && *elasticacheConfig.SnapshotWindow != *foundCacheCluster.SnapshotWindow {
			modifyInput.SnapshotWindow = elasticacheConfig.SnapshotWindow
			updateFound = true
		}
	}
	if updateFound {
		return modifyInput
	}
	return nil
}

// verifyRedisConfig checks elasticache config, if none exist sets values to default
func (p *RedisProvider) buildElasticacheCreateStrategy(ctx context.Context, r *v1alpha1.Redis, ec2Svc ec2iface.EC2API, elasticacheConfig *elasticache.CreateReplicationGroupInput) error {

	elasticacheConfig.AutomaticFailoverEnabled = aws.Bool(true)
	elasticacheConfig.Engine = aws.String("redis")

	if elasticacheConfig.CacheNodeType == nil {
		elasticacheConfig.CacheNodeType = aws.String(defaultCacheNodeType)
	}
	if elasticacheConfig.ReplicationGroupDescription == nil {
		elasticacheConfig.ReplicationGroupDescription = aws.String(defaultDescription)
	}
	if elasticacheConfig.EngineVersion == nil {
		elasticacheConfig.EngineVersion = aws.String(defaultEngineVersion)
	}
	if elasticacheConfig.NumCacheClusters == nil {
		elasticacheConfig.NumCacheClusters = aws.Int64(defaultNumCacheClusters)
	}
	if elasticacheConfig.SnapshotRetentionLimit == nil {
		elasticacheConfig.SnapshotRetentionLimit = aws.Int64(defaultSnapshotRetention)
	}
	if elasticacheConfig.AtRestEncryptionEnabled == nil {
		elasticacheConfig.AtRestEncryptionEnabled = aws.Bool(defaultAtRestEncryption)
	}
	if elasticacheConfig.TransitEncryptionEnabled == nil {
		elasticacheConfig.TransitEncryptionEnabled = aws.Bool(defaultInTransitEncryption)
	}
	cacheName, err := BuildInfraNameFromObject(ctx, p.Client, r.ObjectMeta, DefaultAwsIdentifierLength)
	if err != nil {
		return errorUtil.Wrapf(err, "failed to retrieve elasticache config")
	}
	if elasticacheConfig.ReplicationGroupId == nil {
		elasticacheConfig.ReplicationGroupId = aws.String(cacheName)
	}

	subGroup, err := BuildInfraName(ctx, p.Client, defaultSubnetPostfix, DefaultAwsIdentifierLength)
	if err != nil {
		return errorUtil.Wrap(err, "failed to build subnet group name")
	}
	if elasticacheConfig.CacheSubnetGroupName == nil {
		elasticacheConfig.CacheSubnetGroupName = aws.String(subGroup)
	}

	// build security group name
	secName, err := BuildInfraName(ctx, p.Client, defaultSecurityGroupPostfix, DefaultAwsIdentifierLength)
	if err != nil {
		return errorUtil.Wrap(err, "error building subnet group name")
	}
	// get security group
	foundSecGroup, err := getSecurityGroup(ec2Svc, secName)
	if err != nil {
		return errorUtil.Wrap(err, "")
	}

	if elasticacheConfig.SecurityGroupIds == nil {
		elasticacheConfig.SecurityGroupIds = []*string{
			aws.String(*foundSecGroup.GroupId),
		}
	}

	return nil
}

// buildElasticacheDeleteConfig checks redis config, if none exists sets values to defaults
func (p *RedisProvider) buildElasticacheDeleteConfig(ctx context.Context, r v1alpha1.Redis, elasticacheCreateConfig *elasticache.CreateReplicationGroupInput, elasticacheDeleteConfig *elasticache.DeleteReplicationGroupInput) error {
	cacheName, err := BuildInfraNameFromObject(ctx, p.Client, r.ObjectMeta, DefaultAwsIdentifierLength)
	if err != nil {
		return errorUtil.Wrapf(err, "failed to retrieve elasticache config")
	}
	if elasticacheDeleteConfig.ReplicationGroupId == nil {
		if elasticacheCreateConfig.ReplicationGroupId == nil {
			elasticacheCreateConfig.ReplicationGroupId = aws.String(cacheName)
		}
		elasticacheDeleteConfig.ReplicationGroupId = elasticacheCreateConfig.ReplicationGroupId
	}
	if elasticacheDeleteConfig.RetainPrimaryCluster == nil {
		elasticacheDeleteConfig.RetainPrimaryCluster = aws.Bool(false)
	}
	snapshotIdentifier, err := buildTimestampedInfraNameFromObject(ctx, p.Client, r.ObjectMeta, DefaultAwsIdentifierLength)
	if err != nil {
		return errorUtil.Wrapf(err, "failed to retrieve rds config")
	}
	if elasticacheDeleteConfig.FinalSnapshotIdentifier == nil {
		elasticacheDeleteConfig.FinalSnapshotIdentifier = aws.String(snapshotIdentifier)
	}
	return nil
}

// ensures a subnet group is in place to configure the resource, so that it is in the same vpc as the cluster
func (p *RedisProvider) configureElasticacheVpc(ctx context.Context, cacheSvc elasticacheiface.ElastiCacheAPI, ec2Svc ec2iface.EC2API) error {
	logrus.Info("configuring cluster vpc for redis resource")
	// get subnet group id
	sgName, err := BuildInfraName(ctx, p.Client, defaultSubnetPostfix, DefaultAwsIdentifierLength)
	if err != nil {
		return errorUtil.Wrap(err, "error building subnet group name")
	}

	// check if group exists
	groups, err := cacheSvc.DescribeCacheSubnetGroups(&elasticache.DescribeCacheSubnetGroupsInput{})
	if err != nil {
		return errorUtil.Wrap(err, "error describing subnet groups")
	}
	var foundSubnet *elasticache.CacheSubnetGroup
	for _, sub := range groups.CacheSubnetGroups {
		if *sub.CacheSubnetGroupName == sgName {
			foundSubnet = sub
			break
		}
	}
	if foundSubnet != nil {
		logrus.Infof("%s resource subnet group found", *foundSubnet.CacheSubnetGroupName)
		return nil
	}

	// get cluster vpc subnets
	subIDs, err := GetPrivateSubnetIDS(ctx, p.Client, ec2Svc)
	if err != nil {
		return errorUtil.Wrap(err, "error getting vpc subnets")
	}

	// build subnet group input
	subnetGroupInput := &elasticache.CreateCacheSubnetGroupInput{
		CacheSubnetGroupDescription: aws.String("Subnet group created by the cloud resource operator"),
		CacheSubnetGroupName:        aws.String(sgName),
		SubnetIds:                   subIDs,
	}

	logrus.Info("creating resource subnet group")
	if _, err := cacheSvc.CreateCacheSubnetGroup(subnetGroupInput); err != nil {
		return errorUtil.Wrap(err, "unable to create cache subnet group")
	}

	return nil
}

// returns generic labels to be added to every metric
func buildRedisGenericMetricLabels(r *v1alpha1.Redis, clusterID, cacheName string) map[string]string {
	labels := map[string]string{}
	labels["clusterID"] = clusterID
	labels["resourceID"] = r.Name
	labels["namespace"] = r.Namespace
	labels["instanceID"] = cacheName
	labels["productName"] = r.Labels["productName"]
	labels["strategy"] = redisProviderName
	return labels
}

// adds extra information to labels around resource
func buildRedisInfoMetricLabels(r *v1alpha1.Redis, group *elasticache.ReplicationGroup, clusterID, cacheName string) map[string]string {
	labels := buildRedisGenericMetricLabels(r, clusterID, cacheName)
	if group != nil {
		labels["status"] = *group.Status
		return labels
	}
	labels["status"] = "nil"
	return labels
}

func buildRedisStatusMetricsLabels(r *v1alpha1.Redis, clusterID, cacheName string, phase croType.StatusPhase) map[string]string {
	labels := buildRedisGenericMetricLabels(r, clusterID, cacheName)
	labels["statusPhase"] = string(phase)
	return labels
}

// used to expose an available and information metrics during reconcile
func (p *RedisProvider) exposeRedisMetrics(ctx context.Context, cr *v1alpha1.Redis, instance *elasticache.ReplicationGroup) {
	// build cache name
	cacheName, err := p.buildCacheName(ctx, cr)
	if err != nil {
		logrus.Errorf("error occurred while building instance name while exposing redis metrics: %v", err)
	}

	logrus.Info("setting redis information metric")
	clusterID, err := resources.GetClusterID(ctx, p.Client)
	if err != nil {
		logrus.Errorf("failed to get cluster id while exposing information metrics for %s : %v", cacheName, err)
		return
	}

	// build metric labels
	infoLabels := buildRedisInfoMetricLabels(cr, instance, clusterID, cacheName)

	// build generic metrics
	genericLabels := buildRedisGenericMetricLabels(cr, clusterID, cacheName)

	// set status gauge
	resources.SetMetricCurrentTime(resources.DefaultRedisInfoMetricName, infoLabels)

	// set generic status metrics
	// a single metric should be exposed for each possible phase
	// the value of the metric should be 1.0 when the resource is in that phase
	// the value of the metric should be 0.0 when the resource is not in that phase
	// this follows the approach that pod status
	for _, phase := range []croType.StatusPhase{croType.PhaseFailed, croType.PhaseDeleteInProgress, croType.PhasePaused, croType.PhaseComplete, croType.PhaseInProgress} {
		labelsFailed := buildRedisStatusMetricsLabels(cr, clusterID, cacheName, phase)
		resources.SetMetric(resources.DefaultRedisStatusMetricName, labelsFailed, resources.Btof64(cr.Status.Phase == phase))
	}

	// set availability metric, based on the status flag on the elasticache replication group in aws.
	// 0 is a failure status, 1 is a success status.
	// consider available and snapshotting as non-failure states.
	// see .ReplicationGroups.Status in https://docs.aws.amazon.com/cli/latest/reference/elasticache/describe-replication-groups.html#output
	// for more details on possible status values.
	if instance == nil || !resources.Contains([]string{"available", "snapshotting"}, *instance.Status) {
		resources.SetMetric(resources.DefaultRedisAvailMetricName, genericLabels, 0)
	} else {
		resources.SetMetric(resources.DefaultRedisAvailMetricName, genericLabels, 1)
	}
}

// sets maintenance metric
func (p *RedisProvider) setRedisServiceMaintenanceMetric(ctx context.Context, cacheSvc elasticacheiface.ElastiCacheAPI, instance *elasticache.ReplicationGroup) {
	// if the instance is nil skip this metric
	if instance == nil {
		logrus.Error("foundInstance is nil, skipping setRedisServiceMaintenanceMetric")
		return
	}

	// info about the elasticache cluster to be created
	logrus.Info("checking for pending redis service updates")
	clusterID, err := resources.GetClusterID(ctx, p.Client)
	if err != nil {
		logrus.Errorf("failed to get cluster information while exposing maintenance metric for %s : %v", *instance.ReplicationGroupId, err)
		return
	}

	// Retrieve service maintenance updates, create and export Prometheus metrics
	output, err := cacheSvc.DescribeServiceUpdates(&elasticache.DescribeServiceUpdatesInput{})
	if err != nil {
		logrus.Errorf("failed to get maintenance information while exposing maintenance metric for %s : %v", *instance.ReplicationGroupId, err)
		return
	}

	logrus.Infof("there are elasticache service updates: %d available", len(output.ServiceUpdates))
	for _, su := range output.ServiceUpdates {
		metricLabels := map[string]string{}
		metricLabels["clusterID"] = clusterID

		metricLabels["AutoUpdateAfterRecommendedApplyByDate"] = strconv.FormatBool(*su.AutoUpdateAfterRecommendedApplyByDate)
		metricLabels["Engine"] = *su.Engine
		metricLabels["EstimatedUpdateTime"] = *su.EstimatedUpdateTime
		metricLabels["ServiceUpdateDescription"] = *su.ServiceUpdateDescription
		metricLabels["ServiceUpdateEndDate"] = strconv.FormatInt((*su.ServiceUpdateEndDate).Unix(), 10)
		metricLabels["ServiceUpdateName"] = *su.ServiceUpdateName
		metricLabels["ServiceUpdateRecommendedApplyByDate"] = strconv.FormatInt((*su.ServiceUpdateRecommendedApplyByDate).Unix(), 10)
		metricLabels["ServiceUpdateReleaseDate"] = strconv.FormatInt((*su.ServiceUpdateReleaseDate).Unix(), 10)
		metricLabels["ServiceUpdateSeverity"] = *su.ServiceUpdateSeverity
		metricLabels["ServiceUpdateStatus"] = *su.ServiceUpdateStatus
		metricLabels["ServiceUpdateType"] = *su.ServiceUpdateType

		metricEpochTimestamp := (*su.ServiceUpdateRecommendedApplyByDate).Unix()

		resources.SetMetric(resources.DefaultRedisMaintenanceMetricName, metricLabels, float64(metricEpochTimestamp))
	}
}

func (p *RedisProvider) createElasticacheConnectionMetric(ctx context.Context, cr *v1alpha1.Redis, cache *elasticache.ReplicationGroup) {
	// build cache name
	cacheName, err := p.buildCacheName(ctx, cr)
	if err != nil {
		logrus.Errorf("error occurred while building instance name while exposing redis metrics: %v", err)
	}

	// return cluster id needed for metric labels
	logrus.Infof("testing and exposing redis connection metric for: %s", cacheName)
	clusterID, err := resources.GetClusterID(ctx, p.Client)
	if err != nil {
		logrus.Errorf("failed to get cluster id while exposing connection metric for %v", cacheName)
	}

	// build generic labels to be added to metric
	genericLabels := buildRedisGenericMetricLabels(cr, clusterID, cacheName)

	// check if the node group is available
	if cache == nil || cache.NodeGroups == nil {
		logrus.Infof("%s cache is nil and not yet available", cacheName)
		resources.SetMetric(resources.DefaultRedisConnectionMetricName, genericLabels, 0)
		return
	}

	// test the connection
	conn := p.TCPPinger.TCPConnection(*cache.NodeGroups[0].PrimaryEndpoint.Address, int(*cache.NodeGroups[0].PrimaryEndpoint.Port))
	if !conn {
		// create failed connection metric
		resources.SetMetric(resources.DefaultRedisConnectionMetricName, genericLabels, 0)
		return
	}
	// create successful connection metric
	resources.SetMetric(resources.DefaultRedisConnectionMetricName, genericLabels, 1)
}

func (p *RedisProvider) buildCacheName(ctx context.Context, rd *v1alpha1.Redis) (string, error) {
	cacheName, err := BuildInfraNameFromObject(ctx, p.Client, rd.ObjectMeta, DefaultAwsIdentifierLength)
	if err != nil {
		return "", errorUtil.Errorf("error occurred building cache name: %v", err)
	}
	return cacheName, nil
}
