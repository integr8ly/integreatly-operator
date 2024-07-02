package aws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"

	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/integr8ly/cloud-resource-operator/pkg/annotations"

	croType "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"

	"github.com/aws/aws-sdk-go/service/elasticache/elasticacheiface"
	"github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/integr8ly/cloud-resource-operator/pkg/providers"

	errorUtil "github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultAtRestEncryption = true
	defaultCacheNodeType    = "cache.t3.micro"
	defaultDescription      = "A Redis replication group"
	defaultEngineVersion    = "7.1"
	// 3scale does not support in transit encryption (redis with tls)
	defaultInTransitEncryption = false
	defaultNumCacheClusters    = 2
	defaultSnapshotRetention   = 31
	redisProviderName          = "aws-elasticache"
)

type ServiceUpdate struct {
	updates []string
}

var healthyAWSReplicationGroupStatuses = []string{
	"creating",
	"available",
	"modifying",
	"deleting",
	"snapshotting",
}

var _ providers.RedisProvider = (*RedisProvider)(nil)

// RedisProvider implementation for AWS Elasticache
type RedisProvider struct {
	Client            client.Client
	Logger            *logrus.Entry
	CredentialManager CredentialManager
	ConfigManager     ConfigManager
	CacheSvc          elasticacheiface.ElastiCacheAPI
	TCPPinger         resources.ConnectionTester
}

func NewAWSRedisProvider(client client.Client, logger *logrus.Entry) (*RedisProvider, error) {
	cm, err := NewCredentialManager(client)
	if err != nil {
		return nil, err
	}
	return &RedisProvider{
		Client:            client,
		Logger:            logger.WithFields(logrus.Fields{"provider": redisProviderName}),
		CredentialManager: cm,
		ConfigManager:     NewDefaultConfigMapConfigManager(client),
		TCPPinger:         resources.NewConnectionTestManager(),
	}, nil
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
	logger := p.Logger.WithField("action", "CreateRedis")
	logger.Infof("reconciling redis %s", r.Name)
	// handle provider-specific finalizer
	if err := resources.CreateFinalizer(ctx, p.Client, r, DefaultFinalizer); err != nil {
		return nil, "failed to set finalizer", err
	}

	// info about the elasticache cluster to be created
	elasticacheCreateConfig, _, serviceUpdates, stratCfg, err := p.getElasticacheConfig(ctx, r)
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
	sess, err := CreateSessionFromStrategy(ctx, p.Client, providerCreds, stratCfg)
	if err != nil {
		errMsg := "failed to create aws session to create elasticache replication group"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	maintenanceWindow, err := resources.VerifyRedisMaintenanceWindow(ctx, p.Client, r.Namespace, r.Name)
	if err != nil {
		msg := "failed to verify if redis updates are allowed"
		return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}

	// check if a standalone network is required
	networkManager := NewNetworkManager(sess, p.Client, logger, isSTSCluster(ctx, p.Client))
	isEnabled, err := networkManager.IsEnabled(ctx)
	if err != nil {
		errMsg := "failed to check cluster vpc subnets"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	//networkManager isEnabled checks for the presence of valid CRO subnets in the cluster vpc
	//when CRO subnets are present in a cluster vpc it indicates that the vpc configuration
	//was created in a cluster with a cluster version <= 4.4.5
	//
	//when CRO subnets are absent in a cluster vpc it indicates that the vpc configuration has not been created
	//and a new vpc is created for all resources to be deployed in and peered with the cluster vpc
	if isEnabled {
		// get cidr block from _network strat map, based on tier from redis cr
		vpcCidrBlock, err := networkManager.ReconcileNetworkProviderConfig(ctx, p.ConfigManager, r.Spec.Tier, logger)
		if err != nil {
			errMsg := "failed to get _network strategy config"
			return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
		logger.Debug("standalone network provider enabled, reconciling standalone vpc")

		// create the standalone vpc, subnets and subnet groups
		standaloneNetwork, err := networkManager.CreateNetwork(ctx, vpcCidrBlock)
		if err != nil {
			errMsg := "failed to create resource network"
			return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
		// we've created the standalone vpc, now we peer it to the cluster vpc
		logger.Infof("creating network peering")
		networkPeering, err := networkManager.CreateNetworkPeering(ctx, standaloneNetwork)
		if err != nil {
			errMsg := "failed to peer standalone network"
			return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
		logger.Infof("created network peering %s", aws.StringValue(networkPeering.PeeringConnection.VpcPeeringConnectionId))

		// we have created the peering connection we must now create the security groups and update the route tables
		securityGroup, err := networkManager.CreateNetworkConnection(ctx, standaloneNetwork)
		if err != nil {
			errMsg := "failed to create standalone network"
			return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
		logger.Infof("created security group %s", aws.StringValue(securityGroup.StandaloneSecurityGroup.GroupName))
	}

	// create the aws elasticache cluster
	redis, reconcileStatus, err := p.createElasticacheCluster(ctx, r, elasticache.New(sess), sts.New(sess), ec2.New(sess), elasticacheCreateConfig, stratCfg, serviceUpdates, isEnabled, maintenanceWindow)
	if err != nil {
		errMsg := "failed to reconcile redis instance"
		return nil, reconcileStatus, errorUtil.Wrap(err, errMsg)
	}
	if redis == nil {
		return nil, reconcileStatus, nil
	}

	// set updates allowed to false on the CR after successful reconcile
	if maintenanceWindow {
		r.Spec.MaintenanceWindow = false
		if err := p.Client.Update(ctx, r); err != nil {
			return nil, "failed to set redis allowUpdates to false", err
		}
	}

	return redis, reconcileStatus, nil
}

func (p *RedisProvider) createElasticacheCluster(ctx context.Context, r *v1alpha1.Redis, cacheSvc elasticacheiface.ElastiCacheAPI, stsSvc stsiface.STSAPI, ec2Svc ec2iface.EC2API, elasticacheConfig *elasticache.CreateReplicationGroupInput, _ *StrategyConfig, serviceUpdates *ServiceUpdate, standaloneNetworkExists bool, maintenanceWindow bool) (*providers.RedisCluster, croType.StatusMessage, error) {
	logger := p.Logger.WithField("action", "createElasticacheCluster")
	// the aws access key can sometimes still not be registered in aws on first try, so loop
	rgs, err := getReplicationGroups(cacheSvc)
	if err != nil {
		// return nil error so this function can be requeueed
		errMsg := "error getting replication groups"
		logger.Info(errMsg, err)
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	// we handle standalone networking in CreateRedis() for installs on >= 4.4.6 openshift clusters
	// this check is to ensure backward compatibility with <= 4.4.5 openshift clusters
	// creating bundled (in cluster vpc) subnets, subnet groups, security groups
	//
	// standaloneNetworkExists if no bundled subnets (created by this operator) are found in the cluster vpc
	if !standaloneNetworkExists {
		// setup networking in cluster vpc
		if err := p.configureElasticacheVpc(ctx, cacheSvc, ec2Svc); err != nil {
			errMsg := "error setting up resource vpc"
			return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}

		// setup security group for cluster vpc
		if err := configureSecurityGroup(ctx, p.Client, ec2Svc, logger); err != nil {
			errMsg := "error setting up security group"
			return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
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

	// check if we are running in STS mode
	_, isSTS := p.CredentialManager.(*STSCredentialManager)

	// create elasticache cluster if it doesn't exist
	if foundCache == nil {
		if annotations.Has(r, ResourceIdentifierAnnotation) {
			errMsg := fmt.Sprintf("Redis CR %s in %s namespace has %s annotation with value %s, but no corresponding Elasticache cluster was found",
				r.Name, r.Namespace, ResourceIdentifierAnnotation, r.ObjectMeta.Annotations[ResourceIdentifierAnnotation])
			return nil, croType.StatusMessage(errMsg), fmt.Errorf(errMsg)
		}
		if isSTS {
			// the tag should be added to the create strategy in cases where sts is enabled
			// and in the same api request of the first creation of the postgres to allow
			msg, err := p.buildRedisTagCreateStrategy(ctx, r, elasticacheConfig)
			if err != nil {
				errMsg := fmt.Sprintf("failed to add tags to rds: %s", msg)
				return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
			}
		}

		logrus.Info("creating elasticache cluster")
		if _, err := cacheSvc.CreateReplicationGroup(elasticacheConfig); err != nil {
			errMsg := fmt.Sprintf("error creating elasticache cluster %s", err)
			return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}

		annotations.Add(r, ResourceIdentifierAnnotation, *elasticacheConfig.ReplicationGroupId)
		if err := p.Client.Update(ctx, r); err != nil {
			return nil, croType.StatusMessage("failed to add annotation"), err
		}
		return nil, "started elasticache provision", nil
	}
	logger.Infof("found existing elasticache cluster %s", *foundCache.ReplicationGroupId)

	cacheClustersOutput, err := cacheSvc.DescribeCacheClusters(&elasticache.DescribeCacheClustersInput{})
	if err != nil {
		errMsg := "failed to describe clusters"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}
	var replicationGroupClusters []elasticache.CacheCluster
	for _, checkedCluster := range cacheClustersOutput.CacheClusters {
		cluster := *checkedCluster
		if resources.SafeStringDereference(cluster.ReplicationGroupId) == *foundCache.ReplicationGroupId {
			replicationGroupClusters = append(replicationGroupClusters, *checkedCluster)

			if checkedCluster.EngineVersion != nil && r.Status.Version != *checkedCluster.EngineVersion {
				r.Status.Version = *checkedCluster.EngineVersion
			}
		}
	}

	// check elasticache phase
	if *foundCache.Status != "available" {
		logger.Infof("found instance %s current status %s", *foundCache.ReplicationGroupId, *foundCache.Status)
		return nil, croType.StatusMessage(fmt.Sprintf("createReplicationGroup() in progress, current aws elasticache status is %s", *foundCache.Status)), nil
	}
	logger.Infof("found existing elasticache cluster %s", *foundCache.ReplicationGroupId)

	if maintenanceWindow {
		// check if any modifications are required to bring the elasticache instance up to date with the strategy map.
		modifyInput, err := buildElasticacheUpdateStrategy(ec2Svc, elasticacheConfig, foundCache, replicationGroupClusters, logger, r)
		if err != nil {
			errMsg := "failed to build elasticache modify strategy"
			return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
		if modifyInput == nil {
			logger.Infof("elasticache replication group %s is as expected", *foundCache.ReplicationGroupId)
		}

		// modifications are required to bring the elasticache instance up to date with the strategy map, perform updates.
		if modifyInput != nil {
			logger.Infof("%s differs from expected strategy, applying pending modifications :\n%s", *foundCache.ReplicationGroupId, modifyInput)
			if _, err := cacheSvc.ModifyReplicationGroup(modifyInput); err != nil {
				errMsg := "failed to modify elasticache cluster"
				return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
			}
			logger.Infof("set pending modifications to elasticache replication group %s", *foundCache.ReplicationGroupId)
		}
	}

	if serviceUpdates != nil && len(serviceUpdates.updates) > 0 {
		err = p.applySpecifiedSecurityUpdates(cacheSvc, foundCache, serviceUpdates)
		if err != nil {
			errMsg := "there was an error applying critical security updates"
			return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
	}

	if !isSTS {
		// add tags to cache nodes
		cacheInstance := *foundCache.NodeGroups[0]
		if *cacheInstance.Status != "available" {
			logger.Infof("elasticache node %s current status is %s", *cacheInstance.NodeGroupId, *cacheInstance.Status)
			return nil, croType.StatusMessage(fmt.Sprintf("cache node status not available, current status:  %s", *foundCache.Status)), nil
		}

		for _, cache := range cacheInstance.NodeGroupMembers {
			msg, err := p.TagElasticacheNode(ctx, cacheSvc, stsSvc, r, cache)
			if err != nil {
				errMsg := fmt.Sprintf("failed to add tags to elasticache: %s", msg)
				return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
			}
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

// buildRedisTagCreateStrategy Tags RDS resources
func (p *RedisProvider) buildRedisTagCreateStrategy(ctx context.Context, cr *v1alpha1.Redis, elasticacheCreateConfig *elasticache.CreateReplicationGroupInput) (croType.StatusMessage, error) {
	redisTags, _, err := p.getDefaultElasticacheTags(ctx, cr)
	if err != nil {
		msg := "Failed to build default tags"
		return croType.StatusMessage(msg), errorUtil.Wrapf(err, msg)
	}

	if cr.ObjectMeta.Labels["addonName"] != "" {
		addonTag := &elasticache.Tag{
			Key:   aws.String("add-on-name"),
			Value: aws.String(cr.ObjectMeta.Labels["addonName"]),
		}
		redisTags = append(redisTags, addonTag)
	}

	// adding tags to rds postgres create strategy instance
	elasticacheCreateConfig.SetTags(redisTags)
	return "", nil
}

// TagElasticacheNode Add Tags to AWS Elasticache
func (p *RedisProvider) TagElasticacheNode(ctx context.Context, cacheSvc elasticacheiface.ElastiCacheAPI, stsSvc stsiface.STSAPI, r *v1alpha1.Redis, cache *elasticache.NodeGroupMember) (croType.StatusMessage, error) {
	logrus.Info("creating or updating tags on elasticache nodes and snapshots")

	// check the node to make sure it is available before applying the tag
	// this is needed as the cluster may be available while a node is not
	cacheClusterOutput, err := cacheSvc.DescribeCacheClusters(&elasticache.DescribeCacheClustersInput{
		CacheClusterId: cache.CacheClusterId,
	})
	if err != nil {
		errMsg := "failed to get cache cluster output"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	clusterStatus := *cacheClusterOutput.CacheClusters[0].CacheClusterStatus
	if clusterStatus != "available" {
		errMsg := fmt.Sprintf("%s status is %s, skipping adding tags", *cache.CacheClusterId, clusterStatus)
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// get account identity
	identityInput := &sts.GetCallerIdentityInput{}
	id, err := stsSvc.GetCallerIdentity(identityInput)
	if err != nil {
		errMsg := "failed to get account identity"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// trim availability zone to return cache region
	region := (*cache.PreferredAvailabilityZone)[:len(*cache.PreferredAvailabilityZone)-1]

	// build cluster arn
	// need arn in the following format arn:aws:elasticache:us-east-1:1234567890:cluster:my-mem-cluster
	arn := fmt.Sprintf("arn:aws:elasticache:%s:%s:cluster:%s", region, *id.Account, *cache.CacheClusterId)

	cacheTags, clusterID, err := p.getDefaultElasticacheTags(ctx, r)
	if err != nil {
		msg := "Failed to build default tags"
		return croType.StatusMessage(msg), errorUtil.Wrapf(err, msg)
	}

	// add tags
	_, err = cacheSvc.AddTagsToResource(&elasticache.AddTagsToResourceInput{
		ResourceName: aws.String(arn),
		Tags:         cacheTags,
	})
	if err != nil {
		msg := "failed to add tags to aws elasticache :"
		return croType.StatusMessage(msg), err
	}

	// if snapshots exist add tags to them
	inputDescribe := &elasticache.DescribeSnapshotsInput{
		CacheClusterId: aws.String(*cache.CacheClusterId),
	}

	// loop snapshots adding tags per found snapshot
	snapshotList, _ := cacheSvc.DescribeSnapshots(inputDescribe)
	if snapshotList.Snapshots != nil && len(snapshotList.Snapshots) > 0 {
		metricName := getMetricName(r.Name)
		// We need to reset before recreating so that metrics for deleted snapshots are not orphaned
		resources.ResetMetric(metricName)
		for _, snapshot := range snapshotList.Snapshots {
			snapshotArn := fmt.Sprintf("arn:aws:elasticache:%s:%s:snapshot:%s", region, *id.Account, *snapshot.SnapshotName)
			logrus.Infof("Adding operator tags to snapshot : %s", *snapshot.SnapshotName)
			snapshotInput := &elasticache.AddTagsToResourceInput{
				ResourceName: aws.String(snapshotArn),
				Tags:         cacheTags,
			}
			labels := buildCacheSnapshotNotFoundLabels(clusterID, snapshotArn, snapshot.SnapshotName, cache.CacheClusterId, arn)
			_, err = cacheSvc.AddTagsToResource(snapshotInput)
			if err != nil {
				cacheErr, isAwsErr := err.(awserr.Error)
				if isAwsErr && cacheErr.Code() == elasticache.ErrCodeSnapshotNotFoundFault {
					// SnapshotNotFoundFault. this can happen when Status of Snapshot != "Available"
					logrus.Warningf("SnapshotNotFoundFault error trying tag aws elasticache snapshot")
					resources.SetMetric(metricName, labels, 1)
				} else {
					msg := "failed to add tags to aws elasticache snapshot"
					return croType.StatusMessage(msg), err
				}
			} else {
				resources.SetMetric(metricName, labels, 0)
			}
		}
	}

	logrus.Infof("successfully created or updated tags to elasticache node %s", *cache.CacheClusterId)
	return "successfully created and tagged", nil
}

func getMetricName(redisName string) string {
	// Convention for CRs is - but _ for prom metrics
	name := strings.ReplaceAll(redisName, "-", "_")
	return resources.DefaultRedisSnapshotNotAvailable + "_" + strings.ToLower(name)
}

func buildCacheSnapshotNotFoundLabels(clusterID string, arn string, snapshotName *string, cacheClusterID *string, cacheArn string) map[string]string {
	labels := map[string]string{}
	labels[resources.LabelClusterIDKey] = clusterID
	labels["arn"] = arn
	labels["cacheClusterId"] = resources.SafeStringDereference(cacheClusterID)
	labels["snapshotName"] = resources.SafeStringDereference(snapshotName)
	labels["cacheArn"] = cacheArn
	return labels
}

// DeleteRedis Delete elasticache replication group
func (p *RedisProvider) DeleteRedis(ctx context.Context, r *v1alpha1.Redis) (croType.StatusMessage, error) {
	// resolve elasticache information for elasticache created by provider
	logger := p.Logger.WithField("action", "DeleteRedis")
	logger.Infof("reconciling delete redis %s", r.Name)

	// expose metrics about the redis being deleted
	p.setRedisDeletionTimestampMetric(ctx, r)

	elasticacheCreateConfig, elasticacheDeleteConfig, _, stratCfg, err := p.getElasticacheConfig(ctx, r)
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
	sess, err := CreateSessionFromStrategy(ctx, p.Client, providerCreds, stratCfg)
	if err != nil {
		errMsg := "failed to create aws session to delete elasticache replication group"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// network manager required for cleaning up network.
	networkManager := NewNetworkManager(sess, p.Client, logger, isSTSCluster(ctx, p.Client))

	isEnabled, err := networkManager.IsEnabled(ctx)
	if err != nil {
		errMsg := "failed to check cluster vpc subnets"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	isLastResource, err := resources.IsLastResource(ctx, p.Client)
	if err != nil {
		errMsg := "failed to check if this cr is the last cr of type postgres and redis"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// delete the elasticache cluster
	return p.deleteElasticacheCluster(ctx, networkManager, elasticache.New(sess), ec2.New(sess), elasticacheCreateConfig, elasticacheDeleteConfig, r, isEnabled, isLastResource)
}

func (p *RedisProvider) deleteElasticacheCluster(ctx context.Context, networkManager NetworkManager, cacheSvc elasticacheiface.ElastiCacheAPI, ec2Svc ec2iface.EC2API, elasticacheCreateConfig *elasticache.CreateReplicationGroupInput, elasticacheDeleteConfig *elasticache.DeleteReplicationGroupInput, r *v1alpha1.Redis, isEnabled bool, isLastResource bool) (croType.StatusMessage, error) {
	logger := p.Logger.WithField("action", "deleteElasticacheCluster")

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
	if foundCache != nil {
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
	// isEnabled is true if no bundled resources are found in the cluster vpc
	if isEnabled && isLastResource {
		saVPC, err := getStandaloneVpc(ctx, p.Client, ec2Svc, logger)
		if err != nil {
			msg := "failed to get standalone VPC"
			return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
		}
		// Remove all networking resources if standalone vpc exists
		if saVPC != nil {
			logger.Info("found the last instance of types postgres and redis so deleting the standalone network")
			networkPeering, err := networkManager.GetClusterNetworkPeering(ctx)
			if err != nil {
				msg := "failed to get cluster network peering"
				return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
			}

			if err = networkManager.DeleteNetworkConnection(ctx, networkPeering); err != nil {
				msg := "failed to delete network connection"
				return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
			}

			if err = networkManager.DeleteNetworkPeering(networkPeering); err != nil {
				msg := "failed to delete cluster network peering"
				return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
			}

			if err = networkManager.DeleteNetwork(ctx); err != nil {
				msg := "failed to delete aws networking"
				return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
			}
		}
	}

	// in the case of standalone network not existing and the last resource is being deleted the
	// bundled networking resources should be cleaned up similarly to standalone networking resources
	// this involves the deletion of bundled elasticache and rds subnet group and ec2 security group
	if !isEnabled && isLastResource {
		err := networkManager.DeleteBundledCloudResources(ctx)
		if err != nil {
			msg := "failed to delete bundled networking resources"
			return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
		}
	}
	// remove the finalizer added by the provider
	resources.RemoveFinalizer(&r.ObjectMeta, DefaultFinalizer)
	if err := p.Client.Update(ctx, r); err != nil {
		errMsg := "failed to update instance as part of finalizer reconcile"
		return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}
	return croType.StatusEmpty, nil
}

// poll for replication groups
func getReplicationGroups(cacheSvc elasticacheiface.ElastiCacheAPI) ([]*elasticache.ReplicationGroup, error) {
	var rgs []*elasticache.ReplicationGroup
	err := wait.PollUntilContextTimeout(context.TODO(), time.Second*5, timeOut, true, func(ctx context.Context) (done bool, err error) {
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
func (p *RedisProvider) getElasticacheConfig(ctx context.Context, r *v1alpha1.Redis) (*elasticache.CreateReplicationGroupInput, *elasticache.DeleteReplicationGroupInput, *ServiceUpdate, *StrategyConfig, error) {
	stratCfg, err := p.ConfigManager.ReadStorageStrategy(ctx, providers.RedisResourceType, r.Spec.Tier)
	if err != nil {
		return nil, nil, nil, nil, errorUtil.Wrap(err, "failed to read aws strategy config")
	}
	defRegion, err := GetRegionFromStrategyOrDefault(ctx, p.Client, stratCfg)
	if err != nil {
		return nil, nil, nil, nil, errorUtil.Wrap(err, "failed to get default region")
	}
	if stratCfg.Region == "" {
		p.Logger.Debugf("region not set in deployment strategy configuration, using default region %s", defRegion)
		stratCfg.Region = defRegion
	}

	// unmarshal the elasticache cluster config
	elasticacheCreateConfig := &elasticache.CreateReplicationGroupInput{}
	if err := json.Unmarshal(stratCfg.CreateStrategy, elasticacheCreateConfig); err != nil {
		return nil, nil, nil, nil, errorUtil.Wrap(err, "failed to unmarshal aws elasticache cluster configuration")
	}

	// Override if node size is defined in the CR spec
	if r.Spec.Size != "" {
		elasticacheCreateConfig.CacheNodeType = aws.String(r.Spec.Size)
	}

	elasticacheDeleteConfig := &elasticache.DeleteReplicationGroupInput{}
	if err := json.Unmarshal(stratCfg.DeleteStrategy, elasticacheDeleteConfig); err != nil {
		return nil, nil, nil, nil, errorUtil.Wrap(err, "failed to unmarshal aws elasticache cluster configuration")
	}

	elasticacheServiceUpdates := &ServiceUpdate{}
	if stratCfg.ServiceUpdates != nil {
		if err := json.Unmarshal(stratCfg.ServiceUpdates, &elasticacheServiceUpdates.updates); err != nil {
			return nil, nil, nil, nil, errorUtil.Wrap(err, "failed to unmarshal aws elasticache cluster configuration (serviceUpdates field)")
		}
	}

	return elasticacheCreateConfig, elasticacheDeleteConfig, elasticacheServiceUpdates, stratCfg, nil
}

func (p *RedisProvider) getDefaultElasticacheTags(ctx context.Context, cr *v1alpha1.Redis) ([]*elasticache.Tag, string, error) {
	tags, clusterID, err := resources.GetDefaultResourceTags(ctx, p.Client, cr.Spec.Type, cr.Name, cr.ObjectMeta.Labels["productName"])
	if err != nil {
		msg := "Failed to get default redis tags"
		return nil, "", errorUtil.Wrapf(err, msg)
	}
	return genericListToElasticacheTagList(tags), clusterID, nil
}

// buildElasticacheUpdateStrategy compare the current elasticache state to the proposed elasticache state from the
// strategy map.
//
// if modifications are required, a modify input struct will be returned with all proposed changes.
//
// if no modifications are required, nil will be returned.
func buildElasticacheUpdateStrategy(ec2Client ec2iface.EC2API, elasticacheConfig *elasticache.CreateReplicationGroupInput, foundConfig *elasticache.ReplicationGroup, replicationGroupClusters []elasticache.CacheCluster, logger *logrus.Entry, r *v1alpha1.Redis) (*elasticache.ModifyReplicationGroupInput, error) {
	// setup logger.
	actionLogger := resources.NewActionLogger(logger, "buildElasticacheUpdateStrategy")
	actionLogger.Infof("verifying that %s configuration is as expected", resources.SafeStringDereference(foundConfig.ReplicationGroupId))

	// indicates whether an update should be attempted or not.
	updateFound := false

	// contains the proposed modifications to be made.
	modifyInput := &elasticache.ModifyReplicationGroupInput{}
	modifyInput.ReplicationGroupId = foundConfig.ReplicationGroupId

	// check to see if the cache node type requires a modification.
	if foundConfig.CacheNodeType != nil {
		if elasticacheConfig.CacheNodeType != nil && *elasticacheConfig.CacheNodeType != *foundConfig.CacheNodeType {
			// we need to determine if the proposed cache node type is supported in the availability zones that the instance is
			// deployed into.
			//
			// get the availability zones that support the proposed instance type.
			describeInstanceTypeOfferingOutput, err := ec2Client.DescribeInstanceTypeOfferings(&ec2.DescribeInstanceTypeOfferingsInput{
				Filters: []*ec2.Filter{
					{
						Name:   aws.String("instance-type"),
						Values: aws.StringSlice([]string{strings.Replace(*elasticacheConfig.CacheNodeType, "cache.", "", 1)}),
					},
				},
				LocationType: aws.String(ec2.LocationTypeAvailabilityZone),
			})
			if err != nil {
				return nil, errorUtil.Wrapf(err, "failed to get instance type offerings for type %s", aws.StringValue(foundConfig.CacheNodeType))
			}

			// normalise returned instance type offerings to a list of availability zones, to make comparison easier.
			var supportedAvailabilityZones []string
			for _, instanceTypeOffering := range describeInstanceTypeOfferingOutput.InstanceTypeOfferings {
				supportedAvailabilityZones = append(supportedAvailabilityZones, aws.StringValue(instanceTypeOffering.Location))
			}

			// get the availability zones of the instance.
			var usedAvailabilityZones []string
			for _, replicationGroupCluster := range replicationGroupClusters {
				usedAvailabilityZones = append(usedAvailabilityZones, aws.StringValue(replicationGroupCluster.PreferredAvailabilityZone))
			}

			// ensure the availability zones of the instance support the instance type.
			instanceTypeSupported := true
			for _, usedAvailabilityZone := range usedAvailabilityZones {
				if !resources.Contains(supportedAvailabilityZones, usedAvailabilityZone) {
					instanceTypeSupported = false
					break
				}
			}

			// the instance type is supported, go ahead with the modification.
			if instanceTypeSupported {
				modifyInput.CacheNodeType = elasticacheConfig.CacheNodeType
				modifyInput.ApplyImmediately = aws.Bool(r.Spec.ApplyImmediately)
				updateFound = true
			} else {
				// the instance type isn't supported, log and skip.
				actionLogger.Infof("cache node type %s is not supported, skipping cache node type modification", *elasticacheConfig.CacheNodeType)
			}
		}
	}

	// check if the amount of time snapshots should be kept for requires an update.
	if foundConfig.SnapshotRetentionLimit != nil {
		if *elasticacheConfig.SnapshotRetentionLimit != *foundConfig.SnapshotRetentionLimit {
			modifyInput.SnapshotRetentionLimit = elasticacheConfig.SnapshotRetentionLimit
			updateFound = true
		}
	}

	// elasticache replication groups consist of a group of cache clusters. some information can only be retrieved from
	// these cache clusters instead of the replication group itself.
	//
	// if any cache cluster requires an update, then the replication group itself requires an update. this will update
	// the underlying cache clusters.
	for _, foundCacheCluster := range replicationGroupClusters {
		// check if the redis compatibility version requires an update.
		if elasticacheConfig.EngineVersion != nil {

			engineUpgradeNeeded, err := resources.VerifyVersionUpgradeNeeded(*foundCacheCluster.EngineVersion, *elasticacheConfig.EngineVersion)
			if err != nil {
				return nil, errorUtil.Wrap(err, "invalid redis version")
			}
			if engineUpgradeNeeded {
				modifyInput.SetApplyImmediately(true)
				modifyInput.EngineVersion = elasticacheConfig.EngineVersion
				updateFound = true
			}
		}

		// check if the maintenance window requires an update.
		if elasticacheConfig.PreferredMaintenanceWindow != nil && *elasticacheConfig.PreferredMaintenanceWindow != *foundCacheCluster.PreferredMaintenanceWindow {
			modifyInput.PreferredMaintenanceWindow = elasticacheConfig.PreferredMaintenanceWindow
			updateFound = true
		}

		// check if the time window in which elasticache snapshots can be taken requires an update.
		if elasticacheConfig.SnapshotWindow != nil && *elasticacheConfig.SnapshotWindow != *foundCacheCluster.SnapshotWindow {
			modifyInput.SnapshotWindow = elasticacheConfig.SnapshotWindow
			updateFound = true
		}
	}

	if updateFound {
		return modifyInput, nil
	}
	return nil, nil
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
	cacheName, err := resources.BuildInfraNameFromObject(ctx, p.Client, r.ObjectMeta, defaultAwsIdentifierLength)
	if err != nil {
		return errorUtil.Wrapf(err, "failed to retrieve elasticache config")
	}
	if elasticacheConfig.ReplicationGroupId == nil {
		elasticacheConfig.ReplicationGroupId = aws.String(cacheName)
	}

	subGroup, err := resources.BuildInfraName(ctx, p.Client, defaultSubnetPostfix, defaultAwsIdentifierLength)
	if err != nil {
		return errorUtil.Wrap(err, "failed to build subnet group name")
	}
	if elasticacheConfig.CacheSubnetGroupName == nil {
		elasticacheConfig.CacheSubnetGroupName = aws.String(subGroup)
	}
	// build security group name
	secName, err := resources.BuildInfraName(ctx, p.Client, defaultSecurityGroupPostfix, defaultAwsIdentifierLength)
	if err != nil {
		return errorUtil.Wrap(err, "error building subnet group name")
	}
	foundSecGroup, err := getSecurityGroup(ec2Svc, secName)
	if err != nil {
		return errorUtil.Wrap(err, "")
	}

	if foundSecGroup != nil && elasticacheConfig.SecurityGroupIds == nil {
		elasticacheConfig.SecurityGroupIds = []*string{
			aws.String(*foundSecGroup.GroupId),
		}
	}

	return nil
}

// buildElasticacheDeleteConfig checks redis config, if none exists sets values to defaults
func (p *RedisProvider) buildElasticacheDeleteConfig(ctx context.Context, r v1alpha1.Redis, elasticacheCreateConfig *elasticache.CreateReplicationGroupInput, elasticacheDeleteConfig *elasticache.DeleteReplicationGroupInput) error {
	cacheName, err := resources.BuildInfraNameFromObject(ctx, p.Client, r.ObjectMeta, defaultAwsIdentifierLength)
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
	snapshotIdentifier, err := resources.BuildTimestampedInfraNameFromObject(ctx, p.Client, r.ObjectMeta, defaultAwsIdentifierLength)
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
	sgName, err := resources.BuildInfraName(ctx, p.Client, defaultSubnetPostfix, defaultAwsIdentifierLength)
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
	subIDs, err := GetPrivateSubnetIDS(ctx, p.Client, ec2Svc, p.Logger)
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
	var status string
	if instance != nil {
		status = resources.SafeStringDereference(instance.Status)
	}
	infoLabels := resources.BuildInfoMetricLabels(cr.ObjectMeta, status, clusterID, cacheName, redisProviderName)

	// build generic metrics
	genericLabels := resources.BuildGenericMetricLabels(cr.ObjectMeta, clusterID, cacheName, redisProviderName)

	// set status gauge
	resources.SetMetricCurrentTime(resources.DefaultRedisInfoMetricName, infoLabels)

	// set generic status metrics
	// a single metric should be exposed for each possible phase
	// the value of the metric should be 1.0 when the resource is in that phase
	// the value of the metric should be 0.0 when the resource is not in that phase
	// this follows the approach that pod status
	for _, phase := range []croType.StatusPhase{croType.PhaseFailed, croType.PhaseDeleteInProgress, croType.PhasePaused, croType.PhaseComplete, croType.PhaseInProgress} {
		labelsFailed := resources.BuildStatusMetricsLabels(cr.ObjectMeta, clusterID, cacheName, redisProviderName, phase)
		resources.SetMetric(resources.DefaultRedisStatusMetricName, labelsFailed, resources.Btof64(cr.Status.Phase == phase))
	}

	// set availability metric, based on the status flag on the elasticache replication group in aws.
	// 0 is a failure status, 1 is a success status.
	// consider available and snapshotting as non-failure states.
	// see .ReplicationGroups.Status in https://docs.aws.amazon.com/cli/latest/reference/elasticache/describe-replication-groups.html#output
	// for more details on possible status values.
	if instance == nil || !replicationGroupStatusIsHealthy(instance) {
		resources.SetMetric(resources.DefaultRedisAvailMetricName, genericLabels, 0)
	} else {
		resources.SetMetric(resources.DefaultRedisAvailMetricName, genericLabels, 1)
	}
}

// set metrics about the redis instance being deleted
// works in a similar way to kube_pod_deletion_timestamp
// https://github.com/kubernetes/kube-state-metrics/blob/0bfc2981f9c281c78e33052abdc2d621630562b9/internal/store/pod.go#L200-L218
func (p *RedisProvider) setRedisDeletionTimestampMetric(ctx context.Context, cr *v1alpha1.Redis) {
	if cr.DeletionTimestamp != nil && !cr.DeletionTimestamp.IsZero() {
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

		labels := resources.BuildStatusMetricsLabels(cr.ObjectMeta, clusterID, cacheName, redisProviderName, cr.Status.Phase)
		resources.SetMetric(resources.DefaultRedisDeletionMetricName, labels, float64(cr.DeletionTimestamp.Unix()))
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
	output, err := cacheSvc.DescribeUpdateActions(&elasticache.DescribeUpdateActionsInput{
		ReplicationGroupIds: []*string{instance.ReplicationGroupId},
	})
	if err != nil {
		logrus.Errorf("failed to get update actions information while exposing maintenance metric for %s : %v", *instance.ReplicationGroupId, err)
		return
	}
	if output == nil {
		logrus.Errorf("failed to get update actions(output = nil) information while exposing maintenance metric for %s", *instance.ReplicationGroupId)
		return
	}

	logrus.Infof("there are elasticache service update actions %d available : %s", len(output.UpdateActions), output.UpdateActions)
	for _, updateAction := range output.UpdateActions {
		metricLabels := map[string]string{}
		metricLabels[resources.LabelClusterIDKey] = clusterID

		metricLabels["ReplicationGroupId"] = resources.SafeStringDereference(updateAction.ReplicationGroupId)
		metricLabels["CacheClusterId"] = resources.SafeStringDereference(updateAction.CacheClusterId)
		metricLabels["Engine"] = resources.SafeStringDereference(updateAction.Engine)
		metricLabels["EstimatedUpdateTime"] = resources.SafeStringDereference(updateAction.EstimatedUpdateTime)
		metricLabels["NodesUpdated"] = resources.SafeStringDereference(updateAction.NodesUpdated)
		metricLabels["ServiceUpdateName"] = resources.SafeStringDereference(updateAction.ServiceUpdateName)
		metricLabels["ServiceUpdateRecommendedApplyByDate"] = strconv.FormatInt((resources.SafeTimeDereference(updateAction.ServiceUpdateRecommendedApplyByDate)).Unix(), 10)
		metricLabels["ServiceUpdateReleaseDate"] = strconv.FormatInt((resources.SafeTimeDereference(updateAction.ServiceUpdateReleaseDate)).Unix(), 10)
		metricLabels["ServiceUpdateSeverity"] = resources.SafeStringDereference(updateAction.ServiceUpdateSeverity)
		metricLabels["ServiceUpdateStatus"] = resources.SafeStringDereference(updateAction.ServiceUpdateStatus)
		metricLabels["ServiceUpdateType"] = resources.SafeStringDereference(updateAction.ServiceUpdateType)
		metricLabels["SlaMet"] = resources.SafeStringDereference(updateAction.SlaMet)
		metricLabels["UpdateActionAvailableDate"] = strconv.FormatInt((resources.SafeTimeDereference(updateAction.UpdateActionAvailableDate)).Unix(), 10)
		metricLabels["UpdateActionStatus"] = resources.SafeStringDereference(updateAction.UpdateActionStatus)
		metricLabels["UpdateActionStatusModifiedDate"] = strconv.FormatInt((resources.SafeTimeDereference(updateAction.UpdateActionStatusModifiedDate)).Unix(), 10)

		metricEpochTimestamp := (resources.SafeTimeDereference(updateAction.ServiceUpdateRecommendedApplyByDate)).Unix()

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
	genericLabels := resources.BuildGenericMetricLabels(cr.ObjectMeta, clusterID, cacheName, redisProviderName)

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
	cacheName, err := resources.BuildInfraNameFromObject(ctx, p.Client, rd.ObjectMeta, defaultAwsIdentifierLength)
	if err != nil {
		return "", errorUtil.Errorf("error occurred building cache name: %v", err)
	}
	return cacheName, nil
}

func replicationGroupStatusIsHealthy(cache *elasticache.ReplicationGroup) bool {
	return resources.Contains(healthyAWSReplicationGroupStatuses, *cache.Status)
}

// this function is responsible for checking if there are any critical updates which are specified in the config map
// it gets the updateactions for a given Elasticache from AWS
// it will loop through them and check if they are specified
// if they are it will apply service update
// if the applied update is critical security update, it will apply it immediately
func (p *RedisProvider) applySpecifiedSecurityUpdates(cacheSvc elasticacheiface.ElastiCacheAPI, replicationGroup *elasticache.ReplicationGroup, specifiedUpdates *ServiceUpdate) error {
	logger := p.Logger.WithField("action", "applySpecifiedSecurityUpdates")
	ServiceUpdateStatusAvailable := elasticache.ServiceUpdateStatusAvailable

	updateActions, err := cacheSvc.DescribeUpdateActions(&elasticache.DescribeUpdateActionsInput{
		ReplicationGroupIds: []*string{replicationGroup.ReplicationGroupId},
		ServiceUpdateStatus: []*string{&ServiceUpdateStatusAvailable},
	})

	if err != nil {
		logger.Errorf("failed to get elasticache service updates: %v", err)
		return err
	}

	encounteredError := false
	// Filter list of available updates down to Available Critical Security updates only
	for _, update := range updateActions.UpdateActions {
		for _, specifiedUpdate := range specifiedUpdates.updates {
			if specifiedUpdate == *update.ServiceUpdateName {
				logger.Infof("found specified ServiceUpdate '%s' which matches '%s' ServiceUpdate in aws", specifiedUpdate, *update.ServiceUpdateName)
				logger.Infof("Checking status of Update Action = %s ", resources.SafeStringDereference(update.ServiceUpdateName))
				logger.Infof("UpdateActionStatus = %s ", resources.SafeStringDereference(update.UpdateActionStatus))
				logger.Infof("ServiceUpdateSeverity = %s ", resources.SafeStringDereference(update.ServiceUpdateSeverity))
				logger.Infof("ServiceUpdateStatus = %s ", resources.SafeStringDereference(update.ServiceUpdateStatus))

				if *update.ServiceUpdateStatus == elasticache.ServiceUpdateStatusAvailable &&
					validServiceUpdateStates(resources.SafeStringDereference(update.UpdateActionStatus)) {
					logger.Warnf("Commencing service update %s of Elasticache (Redis) instance %s", resources.SafeStringDereference(update.ServiceUpdateName), *replicationGroup.ReplicationGroupId)
					err := p.applyServiceUpdate(cacheSvc, replicationGroup.ReplicationGroupId, update.ServiceUpdateName)
					if err != nil {
						encounteredError = true
						logger.Errorf("error returned when running batchApplyUpdate function for update %s with err %v", *update.ServiceUpdateName, err)
						break
					}

					if *update.ServiceUpdateSeverity == elasticache.ServiceUpdateSeverityCritical &&
						*update.ServiceUpdateType == elasticache.ServiceUpdateTypeSecurityUpdate {
						// this should push the changes out immediately rather than waiting for the maintenance window.
						logger.Warnf("Setting 'ApplyImmediately' flag to 'true' on the Elasticache instance %s in order to apply the service update immediately", *replicationGroup.ReplicationGroupId)
						if _, err := cacheSvc.ModifyReplicationGroup(&elasticache.ModifyReplicationGroupInput{
							ApplyImmediately:   aws.Bool(true),
							ReplicationGroupId: replicationGroup.ReplicationGroupId}); err != nil {
							logger.Errorf("error returned when running ModifyReplicationGroup function for update %s with err %v", *update.ServiceUpdateName, err)
							encounteredError = true
							break
						}
					}

				}
			}
		}
	}
	if encounteredError {
		return errors.New("encountered an error - check the logs for error messages")
	}
	return nil
}

func (p *RedisProvider) applyServiceUpdate(cacheSvc elasticacheiface.ElastiCacheAPI, replicationgroupid, serviceupdateName *string) error {
	logger := p.Logger.WithField("action", "applyServiceUpdate")

	logger.Warnf("Commencing critical security update of Redis instance %s Service update name: %s", resources.SafeStringDereference(replicationgroupid), resources.SafeStringDereference(serviceupdateName))
	updateOutput, err := cacheSvc.BatchApplyUpdateAction(&elasticache.BatchApplyUpdateActionInput{
		ReplicationGroupIds: []*string{replicationgroupid},
		ServiceUpdateName:   serviceupdateName,
	})
	if err != nil {
		logger.Errorf("Encountered an error when applying Service Update via BatchApplyUpdateAction: %v", err)
		return err
	}
	if len(updateOutput.UnprocessedUpdateActions) > 0 {
		for _, failure := range updateOutput.UnprocessedUpdateActions {
			logger.Errorf("Encountered a %s error when applying Service Update: %s", resources.SafeStringDereference(failure.ErrorType), resources.SafeStringDereference(failure.ErrorMessage))
		}
		return fmt.Errorf("encountered unprocessedupdateaction while applying '%s'", resources.SafeStringDereference(serviceupdateName))
	}
	return nil
}

func validServiceUpdateStates(status string) bool {
	if status == elasticache.UpdateActionStatusNotApplied ||
		status == elasticache.UpdateActionStatusStopping ||
		status == elasticache.UpdateActionStatusStopped ||
		status == elasticache.UpdateActionStatusScheduling {
		return true
	}
	return false
}
