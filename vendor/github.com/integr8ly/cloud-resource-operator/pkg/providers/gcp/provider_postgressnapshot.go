package gcp

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"cloud.google.com/go/storage"
	"github.com/integr8ly/cloud-resource-operator/pkg/annotations"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers/gcp/gcpiface"
	errorUtil "github.com/pkg/errors"
	str2duration "github.com/xhit/go-str2duration/v2"
	"google.golang.org/api/option"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"

	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	croType "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	_ providers.PostgresSnapshotProvider = (*PostgresSnapshotProvider)(nil)
)

const (
	postgresSnapshotProviderName = postgresProviderName + "-snapshots"
	bucketPolicy                 = "roles/storage.objectAdmin"
	labelLatest                  = "latest"
	labelBucketName              = "bucketName"
	labelObjectName              = "objectName"
	lifecycleAdditionalDays      = 10
)

type PostgresSnapshotProvider struct {
	client            client.Client
	logger            *logrus.Entry
	CredentialManager CredentialManager
	ConfigManager     ConfigManager
}

func NewGCPPostgresSnapshotProvider(client client.Client, logger *logrus.Entry) *PostgresSnapshotProvider {
	return &PostgresSnapshotProvider{
		client:            client,
		logger:            logger.WithFields(logrus.Fields{"provider": postgresProviderName}),
		CredentialManager: NewCredentialMinterCredentialManager(client),
		ConfigManager:     NewDefaultConfigManager(client),
	}
}

func (p *PostgresSnapshotProvider) GetName() string {
	return postgresSnapshotProviderName
}

func (p *PostgresSnapshotProvider) SupportsStrategy(deploymentStrategy string) bool {
	return deploymentStrategy == providers.GCPDeploymentStrategy
}

func (p *PostgresSnapshotProvider) GetReconcileTime(snapshot *v1alpha1.PostgresSnapshot) time.Duration {
	if snapshot.Status.Phase != croType.PhaseComplete {
		return time.Second * 60
	}
	return resources.GetForcedReconcileTimeOrDefault(defaultReconcileTime)
}

func (p *PostgresSnapshotProvider) CreatePostgresSnapshot(ctx context.Context, snap *v1alpha1.PostgresSnapshot, pg *v1alpha1.Postgres) (*providers.PostgresSnapshotInstance, croType.StatusMessage, error) {
	logger := p.logger.WithField("action", "CreatePostgresSnapshot")
	if err := resources.CreateFinalizer(ctx, p.client, snap, DefaultFinalizer); err != nil {
		msg := "failed to set finalizer"
		return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}
	strategyConfig, err := p.ConfigManager.ReadStorageStrategy(ctx, providers.PostgresResourceType, pg.Spec.Tier)
	if err != nil {
		msg := "failed to retrieve postgres strategy config"
		return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}
	creds, err := p.CredentialManager.ReconcileProviderCredentials(ctx, pg.Namespace)
	if err != nil {
		msg := fmt.Sprintf("failed to reconcile gcp provider credentials for postgres instance %s", pg.Name)
		return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}
	storageClient, err := gcpiface.NewStorageAPI(ctx, option.WithCredentialsJSON(creds.ServiceAccountJson), logger)
	if err != nil {
		msg := "could not initialise storage client"
		return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}
	sqlClient, err := gcpiface.NewSQLAdminService(ctx, option.WithCredentialsJSON(creds.ServiceAccountJson), p.logger)
	if err != nil {
		errMsg := "could not initialise sql admin service"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	return p.reconcilePostgresSnapshot(ctx, snap, pg, strategyConfig, storageClient, sqlClient)
}

func (p *PostgresSnapshotProvider) DeletePostgresSnapshot(ctx context.Context, snap *v1alpha1.PostgresSnapshot, pg *v1alpha1.Postgres) (croType.StatusMessage, error) {
	logger := p.logger.WithField("action", "DeletePostgresSnapshot")
	creds, err := p.CredentialManager.ReconcileProviderCredentials(ctx, pg.Namespace)
	if err != nil {
		msg := fmt.Sprintf("failed to reconcile gcp provider credentials for postgres instance %s", pg.Name)
		return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}
	storageClient, err := gcpiface.NewStorageAPI(ctx, option.WithCredentialsJSON(creds.ServiceAccountJson), logger)
	if err != nil {
		msg := "could not initialise storage client"
		return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}
	return p.deletePostgresSnapshot(ctx, snap, pg, storageClient)
}

func (p *PostgresSnapshotProvider) reconcilePostgresSnapshot(ctx context.Context, snap *v1alpha1.PostgresSnapshot, pg *v1alpha1.Postgres, config *StrategyConfig, storageClient gcpiface.StorageAPI, sqlClient gcpiface.SQLAdminService) (*providers.PostgresSnapshotInstance, croType.StatusMessage, error) {
	instanceName := annotations.Get(pg, ResourceIdentifierAnnotation)
	if instanceName == "" {
		errMsg := fmt.Sprintf("failed to find %s annotation for postgres cr %s", ResourceIdentifierAnnotation, pg.Name)
		return nil, croType.StatusMessage(errMsg), errorUtil.New(errMsg)
	}
	snapshotID := snap.Name
	snap.Status.SnapshotID = snapshotID
	if err := p.client.Status().Update(ctx, snap); err != nil {
		errMsg := fmt.Sprintf("failed to update snapshot %s in namespace %s", snap.Name, snap.Namespace)
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	objectMeta, err := storageClient.GetObjectMetadata(ctx, instanceName, snapshotID)
	if err != nil && err != storage.ErrObjectNotExist {
		errMsg := fmt.Sprintf("failed to retrieve object metadata for bucket %s and object %s", instanceName, snapshotID)
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	if objectMeta == nil {
		statusMessage, err := p.createPostgresSnapshot(ctx, snap, pg, config, storageClient, sqlClient)
		return nil, statusMessage, err
	}
	snapshotRetention, err := str2duration.ParseDuration(string(pg.Spec.SnapshotRetention))
	if err != nil {
		errMsg := fmt.Sprintf("failed to parse %q into go duration", pg.Spec.SnapshotRetention)
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	lifecycleDays := int64(math.Ceil(snapshotRetention.Hours()/24) + lifecycleAdditionalDays)
	hasLifecycle, err := storageClient.HasBucketLifecycle(ctx, instanceName, lifecycleDays)
	if err != nil {
		errMsg := fmt.Sprintf("failed to check object lifecycle for bucket %s", instanceName)
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	if !hasLifecycle {
		err = storageClient.SetBucketLifecycle(ctx, instanceName, lifecycleDays)
		if err != nil {
			errMsg := fmt.Sprintf("failed to set object lifecycle for bucket %s", instanceName)
			return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
	}
	statusMessage, err := p.reconcileSkipDelete(ctx, snap)
	if err != nil {
		return nil, statusMessage, err
	}
	return &providers.PostgresSnapshotInstance{
		Name: objectMeta.Name,
	}, statusMessage, nil
}

func (p *PostgresSnapshotProvider) createPostgresSnapshot(ctx context.Context, snap *v1alpha1.PostgresSnapshot, pg *v1alpha1.Postgres, config *StrategyConfig, storageClient gcpiface.StorageAPI, sqlClient gcpiface.SQLAdminService) (croType.StatusMessage, error) {
	instanceName := annotations.Get(pg, ResourceIdentifierAnnotation)
	if pg.Status.Phase != croType.PhaseComplete {
		errMsg := fmt.Sprintf("waiting for postgres instance %s to be complete, status %s", instanceName, pg.Status.Phase)
		return croType.StatusMessage(errMsg), errorUtil.New(errMsg)
	}
	bucketAttrs, err := storageClient.GetBucket(ctx, instanceName)
	if err != nil && err != storage.ErrBucketNotExist {
		errMsg := fmt.Sprintf("failed to retrieve bucket metadata for bucket %s", instanceName)
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	if bucketAttrs == nil {
		err = storageClient.CreateBucket(ctx, instanceName, config.ProjectID, &storage.BucketAttrs{
			Location: config.Region,
		})
		if err != nil && !resources.IsConflictError(err) {
			errMsg := fmt.Sprintf("failed to create bucket with name %s", instanceName)
			return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
	}
	instance, err := sqlClient.GetInstance(ctx, config.ProjectID, instanceName)
	if err != nil {
		errMsg := fmt.Sprintf("failed to find postgres instance with name %s", instanceName)
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	serviceAccount := fmt.Sprintf("serviceAccount:%s", instance.ServiceAccountEmailAddress)
	hasPolicy, err := storageClient.HasBucketPolicy(ctx, instanceName, serviceAccount, bucketPolicy)
	if err != nil {
		errMsg := fmt.Sprintf("failed to check bucket policy for %s", instanceName)
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	if !hasPolicy {
		err = storageClient.SetBucketPolicy(ctx, instanceName, serviceAccount, bucketPolicy)
		if err != nil {
			errMsg := fmt.Sprintf("failed to set policy on bucket %s", instanceName)
			return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
	}
	_, err = sqlClient.ExportDatabase(ctx, config.ProjectID, instanceName, &sqladmin.InstancesExportRequest{
		ExportContext: &sqladmin.ExportContext{
			Databases: []string{"postgres"},
			FileType:  "SQL",
			Uri:       fmt.Sprintf("gs://%s/%s", instanceName, snap.Status.SnapshotID),
		},
	})
	if err != nil && !resources.IsConflictError(err) {
		errMsg := fmt.Sprintf("failed to export database from postgres instance %s", instanceName)
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	msg := fmt.Sprintf("snapshot creation started for %s", snap.Name)
	return croType.StatusMessage(msg), nil
}

func (p *PostgresSnapshotProvider) reconcileSkipDelete(ctx context.Context, snap *v1alpha1.PostgresSnapshot) (croType.StatusMessage, error) {
	latestSnapshot, err := getLatestPostgresSnapshot(ctx, p.client, snap.Spec.ResourceName, snap.Namespace)
	if err != nil {
		errMsg := fmt.Sprintf("failed to determine latest snapshot for %s", snap.Spec.ResourceName)
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	if latestSnapshot == nil {
		msg := fmt.Sprintf("no complete snapshots found for %s", snap.Spec.ResourceName)
		return croType.StatusMessage(msg), nil
	}
	if snap.Name == latestSnapshot.Name {
		snap.Spec.SkipDelete = true
		if err = p.client.Update(ctx, snap); err != nil {
			errMsg := fmt.Sprintf("failed to update postgres snapshot %s", snap.Name)
			return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
	}
	if latestSnapshot.Spec.SkipDelete {
		snapshots, err := getAllSnapshotsForInstance(ctx, p.client, snap.Spec.ResourceName, snap.Namespace)
		if err != nil {
			errMsg := "failed to list postgres snapshots"
			return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
		for i := range snapshots {
			if snapshots[i].Name == latestSnapshot.Name {
				continue
			}
			snapshots[i].Spec.SkipDelete = false
			if err = p.client.Update(ctx, snapshots[i]); err != nil {
				errMsg := "failed to remove skipDelete from postgres snapshot cr"
				return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
			}
		}
	}
	msg := fmt.Sprintf("snapshot %s successfully reconciled", snap.Name)
	return croType.StatusMessage(msg), nil
}

func (p *PostgresSnapshotProvider) deletePostgresSnapshot(ctx context.Context, snap *v1alpha1.PostgresSnapshot, pg *v1alpha1.Postgres, storageClient gcpiface.StorageAPI) (croType.StatusMessage, error) {
	if !snap.Spec.SkipDelete {
		instanceName := annotations.Get(pg, ResourceIdentifierAnnotation)
		err := storageClient.DeleteObject(ctx, instanceName, snap.Name)
		if err != nil {
			errMsg := fmt.Sprintf("failed to delete snapshot %s from bucket %s", snap.Name, instanceName)
			return croType.StatusMessage(errMsg), err
		}
		objects, err := storageClient.ListObjects(ctx, instanceName, nil)
		if err != nil {
			errMsg := fmt.Sprintf("failed to list objects from bucket %s", instanceName)
			return croType.StatusMessage(errMsg), err
		}
		for i := range objects {
			if objects[i].Name == snap.Name {
				msg := fmt.Sprintf("object %s deletion in progress", snap.Name)
				return croType.StatusMessage(msg), nil
			}
		}
		if len(objects) == 0 {
			err = storageClient.DeleteBucket(ctx, instanceName)
			if err != nil {
				errMsg := fmt.Sprintf("failed to delete bucket %s", instanceName)
				return croType.StatusMessage(errMsg), err
			}
		}
	}
	resources.RemoveFinalizer(&snap.ObjectMeta, DefaultFinalizer)
	if err := p.client.Update(ctx, snap); err != nil {
		errMsg := "failed to update snapshot as part of finalizer reconcile"
		return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}
	msg := fmt.Sprintf("snapshot %s deleted", snap.Name)
	return croType.StatusMessage(msg), nil
}

func getLatestPostgresSnapshot(ctx context.Context, k8sClient client.Client, resourceName string, namespace string) (*v1alpha1.PostgresSnapshot, error) {
	snapshots, err := getAllSnapshotsForInstance(ctx, k8sClient, resourceName, namespace)
	if err != nil {
		return nil, err
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].GetCreationTimestamp().After(snapshots[j].GetCreationTimestamp().Time)
	})
	var latest *v1alpha1.PostgresSnapshot
	for i := range snapshots {
		if snapshots[i].Status.Phase == croType.PhaseComplete {
			latest = snapshots[i]
			break
		}
	}
	return latest, nil
}

func getAllSnapshotsForInstance(ctx context.Context, k8sClient client.Client, resourceName string, namespace string) ([]*v1alpha1.PostgresSnapshot, error) {
	allSnapshots := &v1alpha1.PostgresSnapshotList{}
	err := k8sClient.List(ctx, allSnapshots, &client.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		return nil, err
	}
	instanceSnapshots := []*v1alpha1.PostgresSnapshot{}
	for i := range allSnapshots.Items {
		if allSnapshots.Items[i].Spec.ResourceName == resourceName {
			instanceSnapshots = append(instanceSnapshots, &allSnapshots.Items[i])
		}
	}
	return instanceSnapshots, nil
}
