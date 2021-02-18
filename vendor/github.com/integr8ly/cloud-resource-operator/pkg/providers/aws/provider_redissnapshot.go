// utility to manage the creation and deletion of snapshots (backups) of Redis clusters in AWS Elasticache.
//
// used by the redis snapshot controller to reconcile RedisSnapshot custom resources
// A snapshot CR must reference an existing Redis CR

package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/elasticache/elasticacheiface"
	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	croType "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	errorUtil "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ providers.RedisSnapshotProvider = (*RedisSnapshotProvider)(nil)

const redisSnapshotProviderName = "aws-redis-snapshots"

type RedisSnapshotProvider struct {
	client            client.Client
	logger            *logrus.Entry
	CredentialManager CredentialManager
	ConfigManager     ConfigManager
}

func NewAWSRedisSnapshotProvider(client client.Client, logger *logrus.Entry) *RedisSnapshotProvider {
	return &RedisSnapshotProvider{
		client:            client,
		logger:            logger.WithFields(logrus.Fields{"provider": redisProviderName}),
		CredentialManager: NewCredentialMinterCredentialManager(client),
		ConfigManager:     NewDefaultConfigMapConfigManager(client),
	}
}

func (p *RedisSnapshotProvider) GetName() string {
	return redisSnapshotProviderName
}

func (p *RedisSnapshotProvider) SupportsStrategy(s string) bool {
	return s == providers.AWSDeploymentStrategy
}

func (p *RedisSnapshotProvider) GetReconcileTime(snapshot *v1alpha1.RedisSnapshot) time.Duration {
	if snapshot.Status.Phase != croType.PhaseComplete {
		return time.Second * 60
	}
	return resources.GetForcedReconcileTimeOrDefault(defaultReconcileTime)
}

func (p *RedisSnapshotProvider) CreateRedisSnapshot(ctx context.Context, snapshot *v1alpha1.RedisSnapshot, redis *v1alpha1.Redis) (*providers.RedisSnapshotInstance, croType.StatusMessage, error) {
	// add finalizer to the snapshot cr
	if err := resources.CreateFinalizer(ctx, p.client, snapshot, DefaultFinalizer); err != nil {
		errMsg := fmt.Sprintf("failed to set finalizer for snapshot %s", snapshot.Name)
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	session, err := p.createSessionForResource(ctx, redis.Namespace, providers.RedisResourceType, redis.Spec.Tier)

	if err != nil {
		errMsg := "failed to create AWS session"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	cacheSvc := elasticache.New(session)

	return p.createRedisSnapshot(ctx, snapshot, redis, cacheSvc)
}

func (p *RedisSnapshotProvider) createRedisSnapshot(ctx context.Context, snapshot *v1alpha1.RedisSnapshot, redis *v1alpha1.Redis, cacheSvc elasticacheiface.ElastiCacheAPI) (*providers.RedisSnapshotInstance, croType.StatusMessage, error) {
	logger := resources.NewActionLogger(p.logger, "createRedisSnapshot")
	// generate snapshot name
	snapshotName, err := BuildTimestampedInfraNameFromObjectCreation(ctx, p.client, snapshot.ObjectMeta, DefaultAwsIdentifierLength)
	if err != nil {
		errMsg := "failed to generate snapshot name"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// update cr with snapshot name
	snapshot.Status.SnapshotID = snapshotName

	if err = p.client.Status().Update(ctx, snapshot); err != nil {
		errMsg := fmt.Sprintf("failed to update instance %s in namespace %s", snapshot.Name, snapshot.Namespace)
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// generate cache cluster name
	clusterName, err := BuildInfraNameFromObject(ctx, p.client, redis.ObjectMeta, DefaultAwsIdentifierLength)
	if err != nil {
		errMsg := "failed to get cluster name"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	foundSnapshot, err := p.findSnapshotInstance(cacheSvc, snapshotName)

	if err != nil {
		errMsg := "failed to describe snaphots in AWS"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// get replication group
	cacheOutput, err := cacheSvc.DescribeReplicationGroups(&elasticache.DescribeReplicationGroupsInput{
		ReplicationGroupId: aws.String(clusterName),
	})

	if cacheOutput == nil {
		errMsg := "snapshot failed, no replication group found"
		return nil, croType.StatusMessage(errMsg), errorUtil.New(errMsg)
	}

	// ensure replication group is available
	if *cacheOutput.ReplicationGroups[0].Status != "available" {
		errMsg := fmt.Sprintf("current replication group status is %s", *cacheOutput.ReplicationGroups[0].Status)
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// find primary cache node
	var cacheName string
	for _, i := range cacheOutput.ReplicationGroups[0].NodeGroups[0].NodeGroupMembers {
		if *i.CurrentRole == "primary" {
			cacheName = *i.CacheClusterId
			break
		}
	}

	// create snapshot of the redis instance
	if foundSnapshot == nil {
		logger.Info("creating redis snapshot")
		_, err = cacheSvc.CreateSnapshot(&elasticache.CreateSnapshotInput{
			CacheClusterId: aws.String(cacheName),
			SnapshotName:   aws.String(snapshotName),
		})
		if err != nil {
			errMsg := "error creating elasticache snapshot"
			return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
		return nil, "snapshot started", nil
	}

	// if snapshot status complete update status
	if *foundSnapshot.SnapshotStatus == "available" {
		return &providers.RedisSnapshotInstance{
			Name: *foundSnapshot.SnapshotName,
		}, "snapshot created", nil
	}

	// creation in progress
	msg := fmt.Sprintf("current snapshot status : %s", *foundSnapshot.SnapshotStatus)
	logger.Info(msg)
	return nil, croType.StatusMessage(msg), nil
}

func (p *RedisSnapshotProvider) DeleteRedisSnapshot(ctx context.Context, snapshot *v1alpha1.RedisSnapshot, redis *v1alpha1.Redis) (croType.StatusMessage, error) {

	session, err := p.createSessionForResource(ctx, redis.Namespace, providers.RedisResourceType, redis.Spec.Tier)

	if err != nil {
		errMsg := "failed to create AWS session"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	cacheSvc := elasticache.New(session)

	return p.deleteRedisSnapshot(ctx, snapshot, redis, cacheSvc)
}

func (p *RedisSnapshotProvider) deleteRedisSnapshot(ctx context.Context, snapshot *v1alpha1.RedisSnapshot, redis *v1alpha1.Redis, cacheSvc elasticacheiface.ElastiCacheAPI) (croType.StatusMessage, error) {
	snapshotName := snapshot.Status.SnapshotID
	foundSnapshot, err := p.findSnapshotInstance(cacheSvc, snapshotName)

	if err != nil {
		errMsg := "failed to describe snaphots in AWS"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// snapshot is deleted
	if foundSnapshot == nil {
		resources.RemoveFinalizer(&snapshot.ObjectMeta, DefaultFinalizer)

		if err := p.client.Update(ctx, snapshot); err != nil {
			msg := "failed to update instance as part of finalizer reconcile"
			return croType.StatusMessage(msg), errorUtil.Wrapf(err, msg)
		}
		return "snapshot deleted", nil
	}

	deleteSnapshotInput := &elasticache.DeleteSnapshotInput{
		SnapshotName: aws.String(snapshotName),
	}

	_, err = cacheSvc.DeleteSnapshot(deleteSnapshotInput)

	if err != nil {
		errMsg := fmt.Sprintf("failed to delete snapshot %s in aws", snapshotName)
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	return "snapshot deletion started", nil
}

func (p *RedisSnapshotProvider) findSnapshotInstance(cacheSvc elasticacheiface.ElastiCacheAPI, snapshotName string) (*elasticache.Snapshot, error) {
	// check snapshot exists
	listOutput, err := cacheSvc.DescribeSnapshots(&elasticache.DescribeSnapshotsInput{
		SnapshotName: aws.String(snapshotName),
	})
	if err != nil {
		elasticacheErr, isAwsErr := err.(awserr.Error)
		if isAwsErr && elasticacheErr.Code() == "SnapshotNotFound" {
			return nil, nil
		}
		return nil, err
	}
	var foundSnapshot *elasticache.Snapshot
	for _, c := range listOutput.Snapshots {
		if *c.SnapshotName == snapshotName {
			foundSnapshot = c
			break
		}
	}
	return foundSnapshot, nil
}

func (p *RedisSnapshotProvider) createSessionForResource(ctx context.Context, namespace string, resourceType providers.ResourceType, tier string) (*session.Session, error) {

	// create the credentials to be used by the aws resource providers, not to be used by end-user
	providerCreds, err := p.CredentialManager.ReconcileProviderCredentials(ctx, namespace)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to reconcile aws credentials")
	}

	// get resource region
	stratCfg, err := p.ConfigManager.ReadStorageStrategy(ctx, resourceType, tier)

	if err != nil {
		return nil, err
	}

	return CreateSessionFromStrategy(ctx, p.client, providerCreds.AccessKeyID, providerCreds.SecretAccessKey, stratCfg)
}
