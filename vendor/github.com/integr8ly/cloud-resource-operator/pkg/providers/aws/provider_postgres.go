package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/integr8ly/cloud-resource-operator/internal/k8sutil"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"

	croType "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	"github.com/integr8ly/cloud-resource-operator/pkg/annotations"

	"k8s.io/apimachinery/pkg/types"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"

	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	errorUtil "github.com/pkg/errors"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	postgresProviderName                 = "aws-rds"
	DefaultAwsIdentifierLength           = 40
	defaultAwsMultiAZ                    = true
	defaultAwsPostgresDeletionProtection = true
	defaultAwsPostgresPort               = 5432
	defaultAwsPostgresUser               = "postgres"
	defaultAwsAllocatedStorage           = 20
	defaultAwsMaxAllocatedStorage        = 100
	defaultAwsPostgresDatabase           = "postgres"
	defaultAwsBackupRetentionPeriod      = 31
	defaultAwsDBInstanceClass            = "db.t3.small"
	defaultAwsEngine                     = "postgres"
	defaultAwsEngineVersion              = "10.13"
	defaultAwsPubliclyAccessible         = false
	defaultAwsSkipFinalSnapshot          = false
	defaultAWSCopyTagsToSnapshot         = true
	defaultAwsDeleteAutomatedBackups     = true
	defaultCredSecSuffix                 = "-aws-rds-credentials"
	defaultPostgresUserKey               = "user"
	defaultPostgresPasswordKey           = "password"
	defaultStorageEncrypted              = true
)

var (
	defaultSupportedEngineVersions = []string{"10.13", "10.6", "9.6", "9.5"}
	healthyAWSDBInstanceStatuses   = []string{
		"backtracking",
		"available",
		"backing-up",
		"configuring-enhanced-monitoring",
		"configuring-iam-database-auth",
		"configuring-log-exports",
		"converting-to-vpc",
		"creating",
		"deleting",
		"maintenance",
		"modifying",
		"moving-to-vpc",
		"renaming",
		"resetting-master-credentials",
		"starting",
		"storage-optimization",
		"upgrading",
	}
)

var _ providers.PostgresProvider = (*PostgresProvider)(nil)

type PostgresProvider struct {
	Client            client.Client
	Logger            *logrus.Entry
	CredentialManager CredentialManager
	ConfigManager     ConfigManager
	TCPPinger         ConnectionTester
}

func NewAWSPostgresProvider(client client.Client, logger *logrus.Entry) *PostgresProvider {
	return &PostgresProvider{
		Client:            client,
		Logger:            logger.WithFields(logrus.Fields{"provider": postgresProviderName}),
		CredentialManager: NewCredentialMinterCredentialManager(client),
		ConfigManager:     NewDefaultConfigMapConfigManager(client),
		TCPPinger:         NewConnectionTestManager(),
	}
}

func (p *PostgresProvider) GetName() string {
	return postgresProviderName
}

func (p *PostgresProvider) SupportsStrategy(d string) bool {
	return d == providers.AWSDeploymentStrategy
}

func (p *PostgresProvider) GetReconcileTime(pg *v1alpha1.Postgres) time.Duration {
	if pg.Status.Phase != croType.PhaseComplete {
		return time.Second * 60
	}
	return resources.GetForcedReconcileTimeOrDefault(defaultReconcileTime)
}

// CreatePostgres creates an RDS Instance from strategy config
func (p *PostgresProvider) CreatePostgres(ctx context.Context, pg *v1alpha1.Postgres) (*providers.PostgresInstance, croType.StatusMessage, error) {
	logger := p.Logger.WithField("action", "CreatePostgres")
	logger.Infof("reconciling postgres %s", pg.Name)

	// handle provider-specific finalizer
	if err := resources.CreateFinalizer(ctx, p.Client, pg, DefaultFinalizer); err != nil {
		return nil, "failed to set finalizer", err
	}

	// info about the RDS instance to be created
	rdsCfg, _, strategyConfig, err := p.getRDSConfig(ctx, pg)
	if err != nil {
		msg := "failed to retrieve aws rds cluster config for instance"
		return nil, croType.StatusMessage(msg), errorUtil.Wrapf(err, msg)
	}

	// create the credentials to be used by the aws resource providers, not to be used by end-user
	providerCreds, err := p.CredentialManager.ReconcileProviderCredentials(ctx, pg.Namespace)
	if err != nil {
		msg := "failed to reconcile rds credentials"
		return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}

	// create credentials secret
	sec := buildDefaultRDSSecret(pg)
	or, err := controllerutil.CreateOrUpdate(ctx, p.Client, sec, func() error {
		return nil
	})
	if err != nil {
		errMsg := fmt.Sprintf("failed to create or update secret %s, action was %s", sec.Name, or)
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	// setup aws RDS instance sdk session
	sess, err := CreateSessionFromStrategy(ctx, p.Client, providerCreds.AccessKeyID, providerCreds.SecretAccessKey, strategyConfig)
	if err != nil {
		errMsg := "failed to create aws session to create rds db instance"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// check is a standalone network is required
	networkManager := NewNetworkManager(sess, p.Client, logger)
	isEnabled, err := networkManager.IsEnabled(ctx)
	if err != nil {
		errMsg := "failed to check cluster vpc subnets"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	//networkManager isEnabled checks for the presence of valid RHMI subnets in the cluster vpc
	//when rhmi subnets are present in a cluster vpc it indicates that the vpc configuration
	//was created in a cluster with a cluster version <= 4.4.5
	//
	//when rhmi subnets are absent in a cluster vpc it indicates that the vpc configuration has not been created
	//and a new vpc is created for all resources to be deployed in and peered with the cluster vpc
	if isEnabled {
		// get cidr block from _network strat map, based on tier from postgres cr
		vpcCidrBlock, err := networkManager.ReconcileNetworkProviderConfig(ctx, p.ConfigManager, pg.Spec.Tier, logger)
		if err != nil {
			errMsg := "failed to reconcile network provider config"
			return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
		logger.Debug("standalone network provider enabled, reconciling standalone vpc")

		// create the standalone vpc, subnets and subnet groups
		standaloneNetwork, err := networkManager.CreateNetwork(ctx, vpcCidrBlock)
		if err != nil {
			errMsg := "failed to create resource network"
			return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
		logger.Infof("created standalone network %s", aws.StringValue(standaloneNetwork.Vpc.VpcId))

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

	// create the aws RDS instance
	return p.createRDSInstance(ctx, pg, rds.New(sess), ec2.New(sess), rdsCfg, isEnabled)
}

func (p *PostgresProvider) createRDSInstance(ctx context.Context, cr *v1alpha1.Postgres, rdsSvc rdsiface.RDSAPI, ec2Svc ec2iface.EC2API, rdsCfg *rds.CreateDBInstanceInput, standaloneNetworkExists bool) (*providers.PostgresInstance, croType.StatusMessage, error) {
	logger := p.Logger.WithField("action", "createRDSInstance")
	// the aws access key can sometimes still not be registered in aws on first try, so loop
	pi, err := getRDSInstances(rdsSvc)
	if err != nil {
		// return nil error so this function can be requeued
		msg := "error getting replication groups"
		return nil, croType.StatusMessage(msg), err
	}

	// we handle standalone networking in CreatePostgres() for installs on >= 4.4.6 openshift cluster
	// this check is to ensure backward compatibility with <= 4.4.5 openshift cluster
	// creating bundled (in cluster vpc) subnets, subnet groups, security groups
	//
	// standaloneNetworkExists if no bundled resources are found in the cluster vpc
	if !standaloneNetworkExists {
		// setup networking in cluster vpc rds vpc
		if err := p.configureRDSVpc(ctx, rdsSvc, ec2Svc); err != nil {
			msg := "error setting up resource vpc"
			return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
		}

		// setup security group for cluster vpc
		if err := configureSecurityGroup(ctx, p.Client, ec2Svc, logger); err != nil {
			msg := "error setting up security group"
			return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
		}
	}

	// getting postgres user password from created secret
	credSec := &v1.Secret{}
	if err := p.Client.Get(ctx, types.NamespacedName{Name: cr.Name + defaultCredSecSuffix, Namespace: cr.Namespace}, credSec); err != nil {
		msg := "failed to retrieve rds credential secret"
		return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}

	postgresPass := string(credSec.Data[defaultPostgresPasswordKey])
	if postgresPass == "" {
		msg := "unable to retrieve rds password"
		return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}

	// verify and build rds create config
	if err := p.buildRDSCreateStrategy(ctx, cr, ec2Svc, rdsCfg, postgresPass); err != nil {
		msg := "failed to build and verify aws rds instance configuration"
		return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}

	// check if the cluster has already been created
	var foundInstance *rds.DBInstance
	for _, i := range pi {
		if *i.DBInstanceIdentifier == *rdsCfg.DBInstanceIdentifier {
			foundInstance = i
			break
		}
	}

	// expose pending maintenance metric
	defer p.setPostgresServiceMaintenanceMetric(ctx, rdsSvc, foundInstance)

	// set status metric
	defer p.exposePostgresMetrics(ctx, cr, foundInstance, ec2Svc)

	// create connection metric
	defer p.createRDSConnectionMetric(ctx, cr, foundInstance)

	// create rds instance if it doesn't exist
	if foundInstance == nil {
		if annotations.Has(cr, ResourceIdentifierAnnotation) {
			errMsg := fmt.Sprintf("Postgres CR %s in %s namespace has %s annotation with value %s, but no corresponding RDS instance was found",
				cr.Name, cr.Namespace, ResourceIdentifierAnnotation, cr.ObjectMeta.Annotations[ResourceIdentifierAnnotation])
			return nil, croType.StatusMessage(errMsg), fmt.Errorf(errMsg)
		}

		logger.Info("creating rds instance")
		if _, err := rdsSvc.CreateDBInstance(rdsCfg); err != nil {
			return nil, croType.StatusMessage(fmt.Sprintf("error creating rds instance %s", err)), err
		}

		annotations.Add(cr, ResourceIdentifierAnnotation, *rdsCfg.DBInstanceIdentifier)
		if err := p.Client.Update(ctx, cr); err != nil {
			return nil, croType.StatusMessage("failed to add annotation"), err
		}
		return nil, "started rds provision", nil
	}

	// check rds instance phase
	if *foundInstance.DBInstanceStatus != "available" {
		logger.Infof("found instance %s current status %s", *foundInstance.DBInstanceIdentifier, *foundInstance.DBInstanceStatus)
		return nil, croType.StatusMessage(fmt.Sprintf("createRDSInstance() in progress, current aws rds resource status is %s", *foundInstance.DBInstanceStatus)), nil
	}

	// check if found instance and user strategy differs, and modify instance
	logger.Infof("found existing rds instance: %s", *foundInstance.DBInstanceIdentifier)
	mi, err := buildRDSUpdateStrategy(rdsCfg, foundInstance)

	if err != nil {
		errMsg := fmt.Sprintf("error building update config for rds instance: %s", *foundInstance.DBInstanceIdentifier)
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	if mi == nil {
		logger.Infof("rds instance %s is as expected", *foundInstance.DBInstanceIdentifier)
	}
	if mi != nil {
		if _, err = rdsSvc.ModifyDBInstance(mi); err != nil {
			errMsg := fmt.Sprintf("error experienced trying to modify db instance: %s", *foundInstance.DBInstanceIdentifier)
			return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
		logger.Infof("set pending modifications for rds instance: %s", *foundInstance.DBInstanceIdentifier)
	}

	// Add Tags to Aws Postgres resources
	msg, err := p.TagRDSPostgres(ctx, cr, rdsSvc, foundInstance)
	if err != nil {
		errMsg := fmt.Sprintf("failed to add tags to rds: %s", msg)
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	pdd := &providers.PostgresDeploymentDetails{
		Username: *foundInstance.MasterUsername,
		Password: postgresPass,
		Host:     *foundInstance.Endpoint.Address,
		Database: *foundInstance.DBName,
		Port:     int(*foundInstance.Endpoint.Port),
	}

	// return secret information
	return &providers.PostgresInstance{DeploymentDetails: pdd}, croType.StatusMessage(fmt.Sprintf("%s, aws rds status is %s", msg, *foundInstance.DBInstanceStatus)), nil
}

// TagRDSPostgres Tags RDS resources
func (p *PostgresProvider) TagRDSPostgres(ctx context.Context, cr *v1alpha1.Postgres, rdsSvc rdsiface.RDSAPI, foundInstance *rds.DBInstance) (croType.StatusMessage, error) {
	logger := p.Logger.WithField("action", "TagRDSPostgres")
	logger.Infof("adding tags to rds instance %s", *foundInstance.DBInstanceIdentifier)
	// get the environment from the CR
	// set the tag values that will always be added
	defaultOrganizationTag := resources.GetOrganizationTag()

	//get Cluster Id
	clusterID, _ := resources.GetClusterID(ctx, p.Client)
	// Set the Tag values

	rdsTag := []*rds.Tag{
		{
			Key:   aws.String(defaultOrganizationTag + "clusterID"),
			Value: aws.String(clusterID),
		},
		{
			Key:   aws.String(defaultOrganizationTag + "resource-type"),
			Value: aws.String(cr.Spec.Type),
		},
		{
			Key:   aws.String(defaultOrganizationTag + "resource-name"),
			Value: aws.String(cr.Name),
		},
	}
	if cr.ObjectMeta.Labels["productName"] != "" {
		productTag := &rds.Tag{
			Key:   aws.String(defaultOrganizationTag + "product-name"),
			Value: aws.String(cr.ObjectMeta.Labels["productName"]),
		}
		rdsTag = append(rdsTag, productTag)
	}

	// adding tags to rds postgres instance
	_, err := rdsSvc.AddTagsToResource(&rds.AddTagsToResourceInput{
		ResourceName: aws.String(*foundInstance.DBInstanceArn),
		Tags:         rdsTag,
	})
	if err != nil {
		msg := "Failed to add Tags to RDS instance"
		return croType.StatusMessage(msg), errorUtil.Wrapf(err, msg)

	}

	// Get a list of Snapshot objects for the DB instance
	rdsSnapshotAttributeInput := &rds.DescribeDBSnapshotsInput{
		DBInstanceIdentifier: aws.String(*foundInstance.DBInstanceIdentifier),
	}
	rdsSnapshotList, err := rdsSvc.DescribeDBSnapshots(rdsSnapshotAttributeInput)
	if err != nil {
		msg := "Can't get Snapshot info"
		return croType.StatusMessage(msg), errorUtil.Wrapf(err, msg)
	}
	// Adding tags to each DB Snapshots from list on AWS
	for _, snapshotList := range rdsSnapshotList.DBSnapshots {
		inputRdsSnapshot := &rds.AddTagsToResourceInput{
			ResourceName: aws.String(*snapshotList.DBSnapshotArn),
			Tags:         rdsTag,
		}
		// Adding Tags to RDS Snapshot
		_, err = rdsSvc.AddTagsToResource(inputRdsSnapshot)
		if err != nil {
			msg := "Failed to add Tags to RDS Snapshot"
			return croType.StatusMessage(msg), errorUtil.Wrapf(err, msg)
		}
	}

	logger.Infof("tags were added successfully to the rds instance %s", *foundInstance.DBInstanceIdentifier)
	return "successfully created and tagged", nil
}

func (p *PostgresProvider) DeletePostgres(ctx context.Context, r *v1alpha1.Postgres) (croType.StatusMessage, error) {
	logger := p.Logger.WithField("action", "DeletePostgres")
	logger.Infof("reconciling postgres %s", r.Name)
	p.setPostgresDeletionTimestampMetric(ctx, r)

	// resolve postgres information for postgres created by provider
	rdsCreateConfig, rdsDeleteConfig, stratCfg, err := p.getRDSConfig(ctx, r)
	if err != nil {
		return "failed to retrieve aws rds config", err
	}

	// get provider aws creds so the postgres instance can be deleted
	providerCreds, err := p.CredentialManager.ReconcileProviderCredentials(ctx, r.Namespace)
	if err != nil {
		msg := "failed to reconcile aws provider credentials"
		return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}

	// setup aws postgres instance sdk session
	sess, err := CreateSessionFromStrategy(ctx, p.Client, providerCreds.AccessKeyID, providerCreds.SecretAccessKey, stratCfg)
	if err != nil {
		errMsg := "failed to create aws session to delete rds db instance"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// network manager required for cleaning up network vpc, subnet and subnet groups.
	networkManager := NewNetworkManager(sess, p.Client, logger)

	isEnabled, err := networkManager.IsEnabled(ctx)
	if err != nil {
		errMsg := "failed to check cluster vpc subnets"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		errMsg := "Failed to get watch namespace"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	isLastResource, err := p.isLastResource(ctx, namespace)
	if err != nil {
		errMsg := "failed to check if this cr is the last cr of type postgres and redis"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	return p.deleteRDSInstance(ctx, r, networkManager, rds.New(sess), ec2.New(sess), rdsCreateConfig, rdsDeleteConfig, isEnabled, isLastResource)
}

func (p *PostgresProvider) deleteRDSInstance(ctx context.Context, pg *v1alpha1.Postgres, networkManager NetworkManager, instanceSvc rdsiface.RDSAPI, ec2Svc ec2iface.EC2API, rdsCreateConfig *rds.CreateDBInstanceInput, rdsDeleteConfig *rds.DeleteDBInstanceInput, standaloneNetworkExists bool, isLastResource bool) (croType.StatusMessage, error) {
	logger := p.Logger.WithField("action", "deleteRDSInstance")

	// the aws access key can sometimes still not be registered in aws on first try, so loop
	pgs, err := getRDSInstances(instanceSvc)
	if err != nil {
		return "error getting aws rds instances", err
	}

	// check and verify delete config
	if err := p.buildRDSDeleteConfig(ctx, pg, rdsCreateConfig, rdsDeleteConfig); err != nil {
		msg := "failed to verify aws rds instance configuration"
		return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}

	// check if the instance has already been deleted
	var foundInstance *rds.DBInstance
	for _, i := range pgs {
		if *i.DBInstanceIdentifier == *rdsDeleteConfig.DBInstanceIdentifier {
			foundInstance = i
			break
		}
	}

	// check if instance exists, if it does attempt to delete it
	// if not delete finalizer and credential secret
	if foundInstance != nil {
		// set status metric
		p.exposePostgresMetrics(ctx, pg, foundInstance, ec2Svc)

		// return if rds instance is not available
		if *foundInstance.DBInstanceStatus != "available" {
			statusMessage := fmt.Sprintf("delete detected, deleteDBInstance() in progress, current aws rds status is %s", *foundInstance.DBInstanceStatus)
			logger.Info(statusMessage)
			return croType.StatusMessage(statusMessage), nil
		}

		// delete rds instance if deletion protection is false
		if !*foundInstance.DeletionProtection {
			_, err = instanceSvc.DeleteDBInstance(rdsDeleteConfig)
			rdsErr, isAwsErr := err.(awserr.Error)
			if err != nil && (!isAwsErr || rdsErr.Code() != rds.ErrCodeDBInstanceNotFoundFault) {
				msg := fmt.Sprintf("failed to delete rds instance : %s", err)
				return croType.StatusMessage(msg), errorUtil.Wrapf(err, msg)
			}
			return "delete detected, deleteDBInstance() started", nil
		}

		// modify rds instance to turn off deletion protection
		_, err = instanceSvc.ModifyDBInstance(&rds.ModifyDBInstanceInput{
			DBInstanceIdentifier: rdsDeleteConfig.DBInstanceIdentifier,
			DeletionProtection:   aws.Bool(false),
		})
		if err != nil {
			msg := "failed to remove deletion protection"
			return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
		}

		return croType.StatusMessage(fmt.Sprintf("deletion protection detected, modifyDBInstance() in progress, current aws rds status is %s", *foundInstance.DBInstanceStatus)), nil
	}

	// standaloneNetworkExists if no bundled resources are found in the cluster vpc
	if standaloneNetworkExists && isLastResource {
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

	// in the case of standalone network not existing and the last resource is being deleted the
	// bundled networking resources should be cleaned up similarly to standalone networking resources
	// this involves the deletion of bundled elasticace and rds subnet group and ec2 security group
	if !standaloneNetworkExists && isLastResource {
		err := networkManager.DeleteBundledCloudResources(ctx)
		if err != nil {
			msg := "failed to delete bundled networking resources"
			return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
		}
	}
	// delete credential secret
	logger.Info("deleting rds secret")
	sec := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pg.Name + defaultCredSecSuffix,
			Namespace: pg.Namespace,
		},
	}
	err = p.Client.Delete(ctx, sec)
	if err != nil && !k8serr.IsNotFound(err) {
		msg := "failed to deleted rds secrets"
		return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}

	resources.RemoveFinalizer(&pg.ObjectMeta, DefaultFinalizer)
	if err := p.Client.Update(ctx, pg); err != nil {
		msg := "failed to update instance as part of finalizer reconcile"
		return croType.StatusMessage(msg), errorUtil.Wrapf(err, msg)
	}
	return croType.StatusEmpty, nil
}

// function to get rds instances, used to check/wait on AWS credentials
func getRDSInstances(cacheSvc rdsiface.RDSAPI) ([]*rds.DBInstance, error) {
	var pi []*rds.DBInstance
	err := wait.PollImmediate(time.Second*5, time.Minute*5, func() (done bool, err error) {
		listOutput, err := cacheSvc.DescribeDBInstances(&rds.DescribeDBInstancesInput{})
		if err != nil {
			return false, nil
		}
		pi = listOutput.DBInstances
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return pi, nil
}

func (p *PostgresProvider) getRDSConfig(ctx context.Context, r *v1alpha1.Postgres) (*rds.CreateDBInstanceInput, *rds.DeleteDBInstanceInput, *StrategyConfig, error) {
	logger := p.Logger.WithField("action", "getRDSConfig")
	stratCfg, err := p.ConfigManager.ReadStorageStrategy(ctx, providers.PostgresResourceType, r.Spec.Tier)
	if err != nil {
		return nil, nil, nil, errorUtil.Wrap(err, "failed to read aws strategy config")
	}

	defRegion, err := GetRegionFromStrategyOrDefault(ctx, p.Client, stratCfg)
	if err != nil {
		return nil, nil, nil, errorUtil.Wrap(err, "failed to get default region")
	}
	if stratCfg.Region == "" {
		logger.Infof("region not set in deployment strategy configuration, using default region %s", defRegion)
		stratCfg.Region = defRegion
	}

	rdsCreateConfig := &rds.CreateDBInstanceInput{}
	if err := json.Unmarshal(stratCfg.CreateStrategy, rdsCreateConfig); err != nil {
		return nil, nil, nil, errorUtil.Wrap(err, "failed to unmarshal aws rds cluster configuration")
	}

	rdsDeleteConfig := &rds.DeleteDBInstanceInput{}
	if err := json.Unmarshal(stratCfg.DeleteStrategy, rdsDeleteConfig); err != nil {
		return nil, nil, nil, errorUtil.Wrap(err, "failed to unmarshal aws rds cluster configuration")
	}
	return rdsCreateConfig, rdsDeleteConfig, stratCfg, nil
}

func (p *PostgresProvider) isLastResource(ctx context.Context, namespace string) (bool, error) {
	listOptions := client.ListOptions{
		Namespace: namespace,
	}
	var postgresList = &v1alpha1.PostgresList{}
	if err := p.Client.List(ctx, postgresList, &listOptions); err != nil {
		msg := "failed to retrieve postgres cr(s)"
		return false, errorUtil.Wrap(err, msg)
	}
	var redisList = &v1alpha1.RedisList{}
	if err := p.Client.List(ctx, redisList, &listOptions); err != nil {
		msg := "failed to retrieve redis cr(s)"
		return false, errorUtil.Wrap(err, msg)
	}
	return len(postgresList.Items) == 1 && len(redisList.Items) == 0, nil
}

// verifies if there is a change between a found instance and the configuration from the instance strat and verified the changes are not pending
func buildRDSUpdateStrategy(rdsConfig *rds.CreateDBInstanceInput, foundConfig *rds.DBInstance) (*rds.ModifyDBInstanceInput, error) {
	logrus.Infof("verifying that %s configuration is as expected", *foundConfig.DBInstanceIdentifier)
	updateFound := false

	mi := &rds.ModifyDBInstanceInput{}
	mi.DBInstanceIdentifier = foundConfig.DBInstanceIdentifier

	if *rdsConfig.DeletionProtection != *foundConfig.DeletionProtection {
		mi.DeletionProtection = rdsConfig.DeletionProtection
		updateFound = true
	}
	if *rdsConfig.Port != *foundConfig.Endpoint.Port {
		mi.DBPortNumber = rdsConfig.Port
		updateFound = true
	}
	if *rdsConfig.BackupRetentionPeriod != *foundConfig.BackupRetentionPeriod {
		mi.BackupRetentionPeriod = rdsConfig.BackupRetentionPeriod
		updateFound = true
	}
	if *rdsConfig.DBInstanceClass != *foundConfig.DBInstanceClass {
		mi.DBInstanceClass = rdsConfig.DBInstanceClass
		updateFound = true
	}
	if *rdsConfig.PubliclyAccessible != *foundConfig.PubliclyAccessible {
		mi.PubliclyAccessible = rdsConfig.PubliclyAccessible
		updateFound = true
	}
	if *rdsConfig.MaxAllocatedStorage != *foundConfig.MaxAllocatedStorage {
		mi.MaxAllocatedStorage = rdsConfig.MaxAllocatedStorage
		updateFound = true
	}
	if *rdsConfig.MultiAZ != *foundConfig.MultiAZ {
		mi.MultiAZ = rdsConfig.MultiAZ
		updateFound = true
	}
	if rdsConfig.AutoMinorVersionUpgrade != nil && *rdsConfig.AutoMinorVersionUpgrade != *foundConfig.AutoMinorVersionUpgrade {
		mi.AutoMinorVersionUpgrade = rdsConfig.AutoMinorVersionUpgrade
		updateFound = true
	}
	if rdsConfig.PreferredBackupWindow != nil && *rdsConfig.PreferredBackupWindow != *foundConfig.PreferredBackupWindow {
		mi.PreferredBackupWindow = rdsConfig.PreferredBackupWindow
		updateFound = true
	}
	if rdsConfig.PreferredMaintenanceWindow != nil && *rdsConfig.PreferredMaintenanceWindow != *foundConfig.PreferredMaintenanceWindow {
		mi.PreferredMaintenanceWindow = rdsConfig.PreferredMaintenanceWindow
		updateFound = true
	}
	if rdsConfig.EngineVersion != nil {
		engineUpgradeNeeded, err := resources.VerifyVersionUpgradeNeeded(*foundConfig.EngineVersion, *rdsConfig.EngineVersion)
		if err != nil {
			return nil, errorUtil.Wrap(err, "invalid postgres version")
		}
		if engineUpgradeNeeded {
			mi.EngineVersion = rdsConfig.EngineVersion
			updateFound = true
		}
	}
	if !updateFound || !verifyPendingModification(mi, foundConfig.PendingModifiedValues) {
		return nil, nil
	}
	return mi, nil
}

// returns true if modify input is not pending
func verifyPendingModification(mi *rds.ModifyDBInstanceInput, pm *rds.PendingModifiedValues) bool {
	pendingModifications := true
	if pm == nil {
		return pendingModifications
	}
	if mi.DBPortNumber != nil && pm.Port != nil {
		if *mi.DBPortNumber == *pm.Port {
			pendingModifications = false
		}
	}
	if mi.BackupRetentionPeriod != nil && pm.BackupRetentionPeriod != nil {
		if *mi.BackupRetentionPeriod == *pm.BackupRetentionPeriod {
			pendingModifications = false
		}
	}
	if mi.DBInstanceClass != nil && pm.DBInstanceClass != nil {
		if *mi.DBInstanceClass == *pm.DBInstanceClass {
			pendingModifications = false
		}
	}
	if mi.EngineVersion != nil && pm.EngineVersion != nil {
		if *mi.EngineVersion == *pm.EngineVersion {
			pendingModifications = false
		}
	}
	if mi.MultiAZ != nil && pm.MultiAZ != nil {
		if *mi.MultiAZ == *pm.MultiAZ {
			pendingModifications = false
		}
	}
	return pendingModifications
}

// verify postgres create config
func (p *PostgresProvider) buildRDSCreateStrategy(ctx context.Context, pg *v1alpha1.Postgres, ec2Svc ec2iface.EC2API, rdsCreateConfig *rds.CreateDBInstanceInput, postgresPassword string) error {
	if rdsCreateConfig.DeletionProtection == nil {
		rdsCreateConfig.DeletionProtection = aws.Bool(defaultAwsPostgresDeletionProtection)
	}
	if rdsCreateConfig.MasterUsername == nil {
		rdsCreateConfig.MasterUsername = aws.String(defaultAwsPostgresUser)
	}
	if rdsCreateConfig.MasterUserPassword == nil {
		rdsCreateConfig.MasterUserPassword = aws.String(postgresPassword)
	}
	if rdsCreateConfig.Port == nil {
		rdsCreateConfig.Port = aws.Int64(defaultAwsPostgresPort)
	}
	if rdsCreateConfig.AutoMinorVersionUpgrade == nil {
		rdsCreateConfig.AutoMinorVersionUpgrade = aws.Bool(false)
	}
	if rdsCreateConfig.DBName == nil {
		rdsCreateConfig.DBName = aws.String(defaultAwsPostgresDatabase)
	}
	if rdsCreateConfig.BackupRetentionPeriod == nil {
		rdsCreateConfig.BackupRetentionPeriod = aws.Int64(defaultAwsBackupRetentionPeriod)
	}
	if rdsCreateConfig.DBInstanceClass == nil {
		rdsCreateConfig.DBInstanceClass = aws.String(defaultAwsDBInstanceClass)
	}
	if rdsCreateConfig.PubliclyAccessible == nil {
		rdsCreateConfig.PubliclyAccessible = aws.Bool(defaultAwsPubliclyAccessible)
	}
	if rdsCreateConfig.AllocatedStorage == nil {
		rdsCreateConfig.AllocatedStorage = aws.Int64(defaultAwsAllocatedStorage)
	}
	if rdsCreateConfig.MaxAllocatedStorage == nil {
		rdsCreateConfig.MaxAllocatedStorage = aws.Int64(defaultAwsMaxAllocatedStorage)
	}
	if rdsCreateConfig.EngineVersion == nil {
		rdsCreateConfig.EngineVersion = aws.String(defaultAwsEngineVersion)
	}
	if rdsCreateConfig.StorageEncrypted == nil {
		rdsCreateConfig.StorageEncrypted = aws.Bool(defaultStorageEncrypted)
	}
	if rdsCreateConfig.EngineVersion != nil {
		if !resources.Contains(defaultSupportedEngineVersions, *rdsCreateConfig.EngineVersion) {
			rdsCreateConfig.EngineVersion = aws.String(defaultAwsEngineVersion)
		}
	}
	instanceName, err := p.buildInstanceName(ctx, pg)
	if err != nil {
		return errorUtil.Wrapf(err, "failed to retrieve rds config")
	}
	if rdsCreateConfig.DBInstanceIdentifier == nil {
		rdsCreateConfig.DBInstanceIdentifier = aws.String(instanceName)
	}
	if rdsCreateConfig.MultiAZ == nil {
		rdsCreateConfig.MultiAZ = aws.Bool(defaultAwsMultiAZ)
	}
	if *rdsCreateConfig.MultiAZ {
		rdsCreateConfig.AvailabilityZone = nil
	}
	rdsCreateConfig.Engine = aws.String(defaultAwsEngine)
	subGroup, err := BuildInfraName(ctx, p.Client, defaultSubnetPostfix, DefaultAwsIdentifierLength)
	if err != nil {
		return errorUtil.Wrapf(err, "failed to build subnet group name")
	}
	if rdsCreateConfig.DBSubnetGroupName == nil {
		rdsCreateConfig.DBSubnetGroupName = aws.String(subGroup)
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

	if rdsCreateConfig.VpcSecurityGroupIds == nil {
		rdsCreateConfig.VpcSecurityGroupIds = []*string{
			aws.String(*foundSecGroup.GroupId),
		}
	}
	if rdsCreateConfig.CopyTagsToSnapshot == nil {
		rdsCreateConfig.CopyTagsToSnapshot = aws.Bool(defaultAWSCopyTagsToSnapshot)
	}
	return nil
}

// verify postgres delete config
func (p *PostgresProvider) buildRDSDeleteConfig(ctx context.Context, pg *v1alpha1.Postgres, rdsCreateConfig *rds.CreateDBInstanceInput, rdsDeleteConfig *rds.DeleteDBInstanceInput) error {
	instanceIdentifier, err := BuildInfraNameFromObject(ctx, p.Client, pg.ObjectMeta, DefaultAwsIdentifierLength)
	if err != nil {
		return errorUtil.Wrapf(err, "failed to retrieve rds config")
	}
	if rdsDeleteConfig.DBInstanceIdentifier == nil {
		if rdsCreateConfig.DBInstanceIdentifier == nil {
			rdsCreateConfig.DBInstanceIdentifier = aws.String(instanceIdentifier)
		}
		rdsDeleteConfig.DBInstanceIdentifier = rdsCreateConfig.DBInstanceIdentifier
	}
	if rdsDeleteConfig.DeleteAutomatedBackups == nil {
		rdsDeleteConfig.DeleteAutomatedBackups = aws.Bool(defaultAwsDeleteAutomatedBackups)
	}
	if rdsDeleteConfig.SkipFinalSnapshot == nil {
		rdsDeleteConfig.SkipFinalSnapshot = aws.Bool(defaultAwsSkipFinalSnapshot)
	}
	snapshotIdentifier, err := buildTimestampedInfraNameFromObject(ctx, p.Client, pg.ObjectMeta, DefaultAwsIdentifierLength)
	if err != nil {
		return errorUtil.Wrap(err, "failed to retrieve timestamped rds config")
	}
	if rdsDeleteConfig.FinalDBSnapshotIdentifier == nil && !*rdsDeleteConfig.SkipFinalSnapshot {
		rdsDeleteConfig.FinalDBSnapshotIdentifier = aws.String(snapshotIdentifier)
	}
	return nil
}

func buildDefaultRDSSecret(ps *v1alpha1.Postgres) *v1.Secret {
	password, err := resources.GeneratePassword()
	if err != nil {
		return nil
	}
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ps.Name + defaultCredSecSuffix,
			Namespace: ps.Namespace,
		},
		StringData: map[string]string{
			defaultPostgresUserKey:     defaultAwsPostgresUser,
			defaultPostgresPasswordKey: password,
		},
		Type: v1.SecretTypeOpaque,
	}
}

// ensures a subnet group is in place to configure the resource to be in the same vpc as the cluster
func (p *PostgresProvider) configureRDSVpc(ctx context.Context, rdsSvc rdsiface.RDSAPI, ec2Svc ec2iface.EC2API) error {
	logger := p.Logger.WithField("action", "configureRDSVpc")
	logger.Info("ensuring vpc is as expected for resource")
	// get subnet group id
	sgID, err := BuildInfraName(ctx, p.Client, defaultSubnetPostfix, DefaultAwsIdentifierLength)
	if err != nil {
		return errorUtil.Wrap(err, "error building subnet group name")
	}

	// check if group exists
	groups, err := rdsSvc.DescribeDBSubnetGroups(&rds.DescribeDBSubnetGroupsInput{})
	if err != nil {
		return errorUtil.Wrap(err, "error describing subnet groups")
	}
	var foundSubnet *rds.DBSubnetGroup
	for _, sub := range groups.DBSubnetGroups {
		if *sub.DBSubnetGroupName == sgID {
			foundSubnet = sub
			break
		}
	}
	if foundSubnet != nil {
		logger.Infof("subnet group %s found", *foundSubnet.DBSubnetGroupName)
		return nil
	}
	defaultOrganizationTag := resources.GetOrganizationTag()

	// get cluster id
	clusterID, err := resources.GetClusterID(ctx, p.Client)
	if err != nil {
		return errorUtil.Wrap(err, "error getting cluster id")
	}

	// get cluster vpc subnets
	subIDs, err := GetPrivateSubnetIDS(ctx, p.Client, ec2Svc, logger)
	if err != nil {
		return errorUtil.Wrap(err, "error getting vpc subnets")
	}

	// build subnet group input
	subnetGroupInput := &rds.CreateDBSubnetGroupInput{
		DBSubnetGroupDescription: aws.String(defaultSubnetGroupDesc),
		DBSubnetGroupName:        aws.String(sgID),
		SubnetIds:                subIDs,
		Tags: []*rds.Tag{
			{
				Key:   aws.String(defaultOrganizationTag + "clusterID"),
				Value: aws.String(clusterID),
			},
		},
	}

	// create db subnet group
	logger.Infof("creating resource subnet group %s", *subnetGroupInput.DBSubnetGroupName)
	if _, err := rdsSvc.CreateDBSubnetGroup(subnetGroupInput); err != nil {
		return errorUtil.Wrap(err, "unable to create db subnet group")
	}

	return nil
}

func buildPostgresInfoMetricLabels(cr *v1alpha1.Postgres, instance *rds.DBInstance, clusterID, instanceName string) map[string]string {
	labels := buildPostgresGenericMetricLabels(cr, clusterID, instanceName)
	if instance != nil {
		labels["status"] = *instance.DBInstanceStatus
		return labels
	}
	labels["status"] = "nil"
	return labels
}

func buildPostgresGenericMetricLabels(cr *v1alpha1.Postgres, clusterID, instanceName string) map[string]string {
	labels := map[string]string{}
	labels["clusterID"] = clusterID
	labels["resourceID"] = cr.Name
	labels["namespace"] = cr.Namespace
	labels["instanceID"] = instanceName
	labels["productName"] = cr.Labels["productName"]
	labels["strategy"] = postgresProviderName
	return labels
}

func buildPostgresStatusMetricsLabels(cr *v1alpha1.Postgres, clusterID, instanceName string, phase croType.StatusPhase) map[string]string {
	labels := buildPostgresGenericMetricLabels(cr, clusterID, instanceName)
	labels["statusPhase"] = string(phase)
	return labels
}

func (p *PostgresProvider) exposePostgresMetrics(ctx context.Context, cr *v1alpha1.Postgres, instance *rds.DBInstance, ec2Svc ec2iface.EC2API) {
	// build instance name
	instanceName, err := p.buildInstanceName(ctx, cr)
	if err != nil {
		logrus.Errorf("error occurred while building instance name during postgres metrics: %v", err)
	}

	// get Cluster Id
	logrus.Info("setting postgres information metric")
	clusterID, err := resources.GetClusterID(ctx, p.Client)
	if err != nil {
		logrus.Errorf("failed to get cluster id while exposing information metric for %v", instanceName)
		return
	}

	// build metric labels
	infoLabels := buildPostgresInfoMetricLabels(cr, instance, clusterID, instanceName)
	// build available mertic labels
	genericLabels := buildPostgresGenericMetricLabels(cr, clusterID, instanceName)

	// set status gauge
	resources.SetMetricCurrentTime(resources.DefaultPostgresInfoMetricName, infoLabels)

	// set generic status metrics
	// a single metric should be exposed for each possible phase
	// the value of the metric should be 1.0 when the resource is in that phase
	// the value of the metric should be 0.0 when the resource is not in that phase
	// this follows the approach that pod status
	for _, phase := range []croType.StatusPhase{croType.PhaseFailed, croType.PhaseDeleteInProgress, croType.PhasePaused, croType.PhaseComplete, croType.PhaseInProgress} {
		labelsFailed := buildPostgresStatusMetricsLabels(cr, clusterID, instanceName, phase)
		resources.SetMetric(resources.DefaultPostgresStatusMetricName, labelsFailed, resources.Btof64(cr.Status.Phase == phase))
	}

	// set availability metric, based on the status flag on the rds instance in aws.
	// 0 is a failure status, 1 is a success status.
	// consider available and backing-up as non-failure states as they don't cause connection failures.
	// see https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Overview.DBInstance.Status.html for possible status
	// values.
	if instance == nil || !rdsInstanceStatusIsHealthy(instance) {
		resources.SetMetric(resources.DefaultPostgresAvailMetricName, genericLabels, 0)
	} else {
		resources.SetMetric(resources.DefaultPostgresAvailMetricName, genericLabels, 1)
	}

	// cloud watch only provides us with free storage space, we need to expose more metrics to allow for more accurate alerting
	// as `predict_linear` is hard to predict on non-linear growth & results in false positives
	// we should follow the approach AWS take to auto scaling, and alert when free storage space is less than 10%
	if instance != nil && instance.AllocatedStorage != nil {
		// convert allocated storage to bytes and expose as a metric
		resources.SetMetric(resources.DefaultPostgresAllocatedStorageMetricName, genericLabels, float64(*instance.AllocatedStorage*resources.BytesInGibiBytes))
	}

	if instance != nil {
		//rds instance types are prefixed ex: db.t3.small
		//need to remove db. prefix for DescribeInstanceTypes
		instanceType := strings.TrimPrefix(*instance.DBInstanceClass, "db.")

		input := &ec2.DescribeInstanceTypesInput{
			InstanceTypes: []*string{&instanceType},
		}
		result, err := ec2Svc.DescribeInstanceTypes(input)
		if err != nil {
			logrus.Errorf("error occurred while describing instance types %v", err)
			return
		}
		instanceTypes := result.InstanceTypes
		if len(instanceTypes) > 0 {
			MemorySize := instanceTypes[0].MemoryInfo.SizeInMiB
			resources.SetMetric(resources.DefaultPostgresMaxMemoryMetricName, genericLabels, float64(*MemorySize))
		}
	}
}

// set metrics about the postgres instance being deleted
// works in a similar way to kube_pod_deletion_timestamp
// https://github.com/kubernetes/kube-state-metrics/blob/0bfc2981f9c281c78e33052abdc2d621630562b9/internal/store/pod.go#L200-L218
func (p *PostgresProvider) setPostgresDeletionTimestampMetric(ctx context.Context, cr *v1alpha1.Postgres) {
	if cr.DeletionTimestamp != nil && !cr.DeletionTimestamp.IsZero() {
		// build instance name
		instanceName, err := p.buildInstanceName(ctx, cr)
		if err != nil {
			logrus.Errorf("error occurred while building instance name during postgres metrics: %v", err)
		}

		// get Cluster Id
		logrus.Info("setting postgres information metric")
		clusterID, err := resources.GetClusterID(ctx, p.Client)
		if err != nil {
			logrus.Errorf("failed to get cluster id while exposing information metric for %v", instanceName)
			return
		}

		labels := buildPostgresStatusMetricsLabels(cr, clusterID, instanceName, cr.Status.Phase)
		resources.SetMetric(resources.DefaultPostgresDeletionMetricName, labels, float64(cr.DeletionTimestamp.Unix()))
	}
}

func (p *PostgresProvider) setPostgresServiceMaintenanceMetric(ctx context.Context, rdsSession rdsiface.RDSAPI, instance *rds.DBInstance) {
	// if the instance is nil skip this metric
	if instance == nil {
		logrus.Error("foundInstance is nil, skipping setPostgresServiceMaintenanceMetric")
		return
	}

	logrus.Info("checking for pending postgres service updates")
	clusterID, err := resources.GetClusterID(ctx, p.Client)
	if err != nil {
		logrus.Errorf("failed to get cluster id while exposing information metric for %s : %v", *instance.DBInstanceIdentifier, err)
		return
	}

	// Retrieve service maintenance updates, create and export Prometheus metrics
	output, err := rdsSession.DescribePendingMaintenanceActions(&rds.DescribePendingMaintenanceActionsInput{})
	if err != nil {
		logrus.Errorf("failed to get maintenance information while exposing maintenance metric for %s : %v", *instance.DBInstanceIdentifier, err)
		return
	}

	logrus.Infof("rds serviceupdates: %d available", len(output.PendingMaintenanceActions))
	for _, su := range output.PendingMaintenanceActions {
		metricLabels := map[string]string{}

		metricLabels["clusterID"] = clusterID
		metricLabels["ResourceIdentifier"] = *su.ResourceIdentifier

		for _, pma := range su.PendingMaintenanceActionDetails {

			metricEpochTimestamp := time.Now().Unix()

			if pma.AutoAppliedAfterDate != nil && !pma.AutoAppliedAfterDate.IsZero() {
				metricLabels["AutoAppliedAfterDate"] = strconv.FormatInt((*pma.AutoAppliedAfterDate).Unix(), 10)
				metricEpochTimestamp = (*pma.AutoAppliedAfterDate).Unix()
			}

			if pma.CurrentApplyDate != nil && !pma.CurrentApplyDate.IsZero() {
				metricLabels["CurrentApplyDate"] = strconv.FormatInt((*pma.CurrentApplyDate).Unix(), 10)
				metricEpochTimestamp = (*pma.CurrentApplyDate).Unix()
			}

			metricLabels["Description"] = *pma.Description

			resources.SetMetric(resources.DefaultPostgresMaintenanceMetricName, metricLabels, float64(metricEpochTimestamp))
		}
	}

}

// tests to see if a simple tcp connection can be made to rds and creates a metric based on this
func (p *PostgresProvider) createRDSConnectionMetric(ctx context.Context, cr *v1alpha1.Postgres, instance *rds.DBInstance) {
	// build instance name
	instanceName, err := p.buildInstanceName(ctx, cr)
	if err != nil {
		logrus.Errorf("error occurred while building instance name during postgres metrics: %v", err)
	}

	// return cluster id needed for metric labels
	logrus.Infof("testing and exposing postgres connection metric for: %s", instanceName)
	clusterID, err := resources.GetClusterID(ctx, p.Client)
	if err != nil {
		logrus.Errorf("failed to get cluster id while exposing connection metric for %v", instanceName)

	}

	// build generic labels to be added to metric
	genericLabels := buildPostgresGenericMetricLabels(cr, clusterID, instanceName)

	// check if the instance is available
	if instance == nil {
		logrus.Infof("foundInstance is nil, setting createRDSConnectionMetric to 0")
		resources.SetMetric(resources.DefaultPostgresConnectionMetricName, genericLabels, 0)
		return
	}

	// check if the endpoint is available
	if instance.Endpoint == nil {
		logrus.Infof("instance endpoint not yet available for: %s", *instance.DBInstanceIdentifier)
		resources.SetMetric(resources.DefaultPostgresConnectionMetricName, genericLabels, 0)
		return
	}

	// test the connection
	conn := p.TCPPinger.TCPConnection(*instance.Endpoint.Address, int(*instance.Endpoint.Port))
	if !conn {
		// create failed connection metric
		resources.SetMetric(resources.DefaultPostgresConnectionMetricName, genericLabels, 0)
		return
	}
	// create successful connection metric
	resources.SetMetric(resources.DefaultPostgresConnectionMetricName, genericLabels, 1)
}

// returns the name of the instance from build infra
func (p *PostgresProvider) buildInstanceName(ctx context.Context, pg *v1alpha1.Postgres) (string, error) {
	instanceName, err := BuildInfraNameFromObject(ctx, p.Client, pg.ObjectMeta, DefaultAwsIdentifierLength)
	if err != nil {
		return "", errorUtil.Errorf("error occurred building instance name: %v", err)
	}
	return instanceName, nil
}

func rdsInstanceStatusIsHealthy(instance *rds.DBInstance) bool {
	return resources.Contains(healthyAWSDBInstanceStatuses, *instance.DBInstanceStatus)
}
