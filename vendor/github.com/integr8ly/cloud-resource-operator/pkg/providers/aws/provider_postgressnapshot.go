// utility to manage the creation and deletion of snapshots (backups) of Postgres instances in AWS RDS.
//
// used by the postgres snapshot controller to reconcile PostgresSnapshot custom resources
// A snapshot CR must reference an existing Postgres CR

package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	croType "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	errorUtil "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ providers.PostgresSnapshotProvider = (*PostgresSnapshotProvider)(nil)

const postgresSnapshotProviderName = "aws-rds-snapshots"

type PostgresSnapshotProvider struct {
	client            client.Client
	logger            *logrus.Entry
	CredentialManager CredentialManager
	ConfigManager     ConfigManager
}

func NewAWSPostgresSnapshotProvider(client client.Client, logger *logrus.Entry) *PostgresSnapshotProvider {
	return &PostgresSnapshotProvider{
		client:            client,
		logger:            logger.WithFields(logrus.Fields{"provider": postgresSnapshotProviderName}),
		CredentialManager: NewCredentialMinterCredentialManager(client),
		ConfigManager:     NewDefaultConfigMapConfigManager(client),
	}
}

func (p *PostgresSnapshotProvider) GetName() string {
	return postgresSnapshotProviderName
}

func (p *PostgresSnapshotProvider) SupportsStrategy(s string) bool {
	return s == providers.AWSDeploymentStrategy
}

func (p *PostgresSnapshotProvider) GetReconcileTime(snapshot *v1alpha1.PostgresSnapshot) time.Duration {
	if snapshot.Status.Phase != croType.PhaseComplete {
		return time.Second * 60
	}
	return resources.GetForcedReconcileTimeOrDefault(defaultReconcileTime)
}

func (p *PostgresSnapshotProvider) CreatePostgresSnapshot(ctx context.Context, snapshot *v1alpha1.PostgresSnapshot, postgres *v1alpha1.Postgres) (*providers.PostgresSnapshotInstance, croType.StatusMessage, error) {
	// add finalizer to the snapshot cr
	if err := resources.CreateFinalizer(ctx, p.client, snapshot, DefaultFinalizer); err != nil {
		errMsg := fmt.Sprintf("failed to set finalizer for snapshot %s", snapshot.Name)
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	session, err := p.createSessionForResource(ctx, postgres.Namespace, providers.PostgresResourceType, postgres.Spec.Tier)

	if err != nil {
		errMsg := "failed to create AWS session"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	rdsSvc := rds.New(session)

	return p.createPostgresSnapshot(ctx, snapshot, postgres, rdsSvc)
}

func (p *PostgresSnapshotProvider) DeletePostgresSnapshot(ctx context.Context, snapshot *v1alpha1.PostgresSnapshot, postgres *v1alpha1.Postgres) (croType.StatusMessage, error) {

	// create the credentials to be used by the aws resource providers, not to be used by end-user
	session, err := p.createSessionForResource(ctx, postgres.Namespace, providers.PostgresResourceType, postgres.Spec.Tier)

	if err != nil {
		errMsg := "failed to create AWS session"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	rdsSvc := rds.New(session)

	return p.deletePostgresSnapshot(ctx, snapshot, postgres, rdsSvc)
}

func (p *PostgresSnapshotProvider) createPostgresSnapshot(ctx context.Context, snapshot *v1alpha1.PostgresSnapshot, postgres *v1alpha1.Postgres, rdsSvc rdsiface.RDSAPI) (*providers.PostgresSnapshotInstance, croType.StatusMessage, error) {
	logger := resources.NewActionLogger(p.logger, "createPostgresSnapshot")

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

	// get instance name
	instanceName, err := BuildInfraNameFromObject(ctx, p.client, postgres.ObjectMeta, DefaultAwsIdentifierLength)
	if err != nil {
		errMsg := "failed to get cluster name"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	foundSnapshot, err := p.findSnapshotInstance(rdsSvc, snapshotName)

	if err != nil {
		errMsg := "failed to describe snaphots in AWS"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// create snapshot of the rds instance
	if foundSnapshot == nil {
		// postgres instance has either just been created
		// or is already backing up
		if postgres.Status.Phase == croType.PhaseInProgress {
			return nil, croType.StatusMessage("waiting for postgres instance to be available"), nil
		}
		// postgres instance is being deleted
		// impossible to create a snapshot
		if postgres.Status.Phase == croType.PhaseDeleteInProgress {
			errMsg := "cannot create snapshot when instance deletion is in progress"
			return nil, croType.StatusMessage(errMsg), errorUtil.New(errMsg)
		}
		logger.Info("creating rds snapshot")
		_, err = rdsSvc.CreateDBSnapshot(&rds.CreateDBSnapshotInput{
			DBInstanceIdentifier: aws.String(instanceName),
			DBSnapshotIdentifier: aws.String(snapshotName),
		})
		if err != nil {
			errMsg := "error creating rds snapshot"
			return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
		return nil, "snapshot started", nil
	}

	// if snapshot status complete update status
	if *foundSnapshot.Status == "available" {
		return &providers.PostgresSnapshotInstance{
			Name: *foundSnapshot.DBSnapshotIdentifier,
		}, "snapshot created", nil
	}

	// creation in progress
	msg := fmt.Sprintf("current snapshot status : %s", *foundSnapshot.Status)
	logger.Info(msg)
	return nil, croType.StatusMessage(msg), nil
}

func (p *PostgresSnapshotProvider) deletePostgresSnapshot(ctx context.Context, snapshot *v1alpha1.PostgresSnapshot, postgres *v1alpha1.Postgres, rdsSvc rdsiface.RDSAPI) (croType.StatusMessage, error) {
	snapshotName := snapshot.Status.SnapshotID
	foundSnapshot, err := p.findSnapshotInstance(rdsSvc, snapshotName)

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

	deleteSnapshotInput := &rds.DeleteDBSnapshotInput{
		DBSnapshotIdentifier: foundSnapshot.DBSnapshotIdentifier,
	}

	_, err = rdsSvc.DeleteDBSnapshot(deleteSnapshotInput)

	if err != nil {
		errMsg := fmt.Sprintf("failed to delete snapshot %s in aws", snapshotName)
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	return "snapshot deletion started", nil
}

func (p *PostgresSnapshotProvider) findSnapshotInstance(rdsSvc rdsiface.RDSAPI, snapshotName string) (*rds.DBSnapshot, error) {
	// check snapshot exists
	listOutput, err := rdsSvc.DescribeDBSnapshots(&rds.DescribeDBSnapshotsInput{
		DBSnapshotIdentifier: aws.String(snapshotName),
	})
	if err != nil {
		rdsErr, isAwsErr := err.(awserr.Error)
		if isAwsErr && rdsErr.Code() == "DBSnapshotNotFound" {
			return nil, nil
		}
		return nil, err
	}
	var foundSnapshot *rds.DBSnapshot
	for _, c := range listOutput.DBSnapshots {
		if *c.DBSnapshotIdentifier == snapshotName {
			foundSnapshot = c
			break
		}
	}
	return foundSnapshot, nil
}

func (p *PostgresSnapshotProvider) createSessionForResource(ctx context.Context, namespace string, resourceType providers.ResourceType, tier string) (*session.Session, error) {

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
