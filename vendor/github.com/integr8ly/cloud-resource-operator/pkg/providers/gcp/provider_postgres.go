package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/utils/ptr"
	"reflect"
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/integr8ly/cloud-resource-operator/pkg/providers/gcp/gcpiface"

	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	croType "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	"github.com/integr8ly/cloud-resource-operator/pkg/annotations"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	errorUtil "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	str2duration "github.com/xhit/go-str2duration/v2"
	"google.golang.org/api/option"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
	v1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
)

const (
	postgresProviderName                          = "gcp-cloudsql"
	ResourceIdentifierAnnotation                  = "resourceIdentifier"
	defaultCredSecSuffix                          = "-gcp-sql-credentials" // #nosec G101 -- false positive (ref: https://securego.io/docs/rules/g101.html)
	defaultGCPCLoudSQLDatabaseVersion             = "POSTGRES_13"
	defaultGCPPostgresUser                        = "postgres"
	defaultPostgresPasswordKey                    = "password"
	defaultPostgresUserKey                        = "user"
	defaultTier                                   = "db-custom-2-3840"
	defaultAvailabilityType                       = "REGIONAL"
	defaultStorageAutoResizeLimit                 = 100
	defaultStorageAutoResize                      = true
	defaultBackupConfigEnabled                    = true
	defaultPointInTimeRecoveryEnabled             = true
	defaultBackupRetentionSettingsRetentionUnit   = "COUNT"
	defaultBackupRetentionSettingsRetainedBackups = 30
	defaultDataDiskSizeGb                         = 20
	defaultDeleteProtectionEnabled                = true
	defaultIPConfigIPV4Enabled                    = false
	defaultGCPPostgresPort                        = 5432
	defaultDeploymentDatabase                     = "postgres"
)

type PostgresProvider struct {
	Client            client.Client
	Logger            *logrus.Entry
	CredentialManager CredentialManager
	ConfigManager     ConfigManager
	TCPPinger         resources.ConnectionTester
}

type CreateInstanceRequest struct {
	Instance *gcpiface.DatabaseInstance `json:"instance,omitempty"`
}

func NewGCPPostgresProvider(client client.Client, logger *logrus.Entry) *PostgresProvider {
	return &PostgresProvider{
		Client:            client,
		Logger:            logger.WithFields(logrus.Fields{"provider": postgresProviderName}),
		CredentialManager: NewCredentialMinterCredentialManager(client),
		ConfigManager:     NewDefaultConfigManager(client),
		TCPPinger:         resources.NewConnectionTestManager(),
	}
}

func (p *PostgresProvider) GetName() string {
	return postgresProviderName
}

func (p *PostgresProvider) SupportsStrategy(deploymentStrategy string) bool {
	return deploymentStrategy == providers.GCPDeploymentStrategy
}

func (p *PostgresProvider) GetReconcileTime(pg *v1alpha1.Postgres) time.Duration {
	if pg.Status.Phase != croType.PhaseComplete {
		return time.Second * 60
	}
	return resources.GetForcedReconcileTimeOrDefault(defaultReconcileTime)
}

func (p *PostgresProvider) ReconcilePostgres(ctx context.Context, pg *v1alpha1.Postgres) (*providers.PostgresInstance, croType.StatusMessage, error) {
	logger := p.Logger.WithField("action", "CreatePostgres")
	if err := resources.CreateFinalizer(ctx, p.Client, pg, DefaultFinalizer); err != nil {
		return nil, "failed to set finalizer", err
	}
	strategyConfig, err := p.ConfigManager.ReadStorageStrategy(ctx, providers.PostgresResourceType, pg.Spec.Tier)
	if err != nil {
		msg := "failed to retrieve postgres strategy config"
		return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}

	creds, err := p.CredentialManager.ReconcileProviderCredentials(ctx, pg.Namespace)
	if err != nil {
		errMsg := fmt.Sprintf("failed to reconcile gcp postgres provider credentials for postgres instance %s", pg.Name)
		return nil, croType.StatusMessage(errMsg), fmt.Errorf("%s: %w", errMsg, err)
	}

	sqlClient, err := gcpiface.NewSQLAdminService(ctx, option.WithCredentialsJSON(creds.ServiceAccountJson), p.Logger)
	if err != nil {
		errMsg := "could not initialise new SQL Admin Service"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	networkManager, err := NewNetworkManager(ctx, strategyConfig.ProjectID, option.WithCredentialsJSON(creds.ServiceAccountJson), p.Client, logger)
	if err != nil {
		errMsg := "failed to initialise network manager"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	// get cidr block from _network strat map, based on tier from postgres cr
	ipRangeCidr, err := networkManager.ReconcileNetworkProviderConfig(ctx, p.ConfigManager, pg.Spec.Tier)
	if err != nil {
		errMsg := "failed to reconcile network provider config"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	address, msg, err := networkManager.CreateNetworkIpRange(ctx, ipRangeCidr)
	if err != nil || msg != "" {
		return nil, msg, err
	}
	_, msg, err = networkManager.CreateNetworkService(ctx)
	if err != nil || msg != "" {
		return nil, msg, err
	}

	instance, statusMessage, err := p.reconcileCloudSQLInstance(ctx, pg, sqlClient, strategyConfig, address)
	if err != nil || instance == nil {
		return nil, statusMessage, err
	}
	if pg.Spec.SnapshotFrequency != "" && pg.Spec.SnapshotRetention != "" {
		statusMessage, err = p.reconcileCloudSqlInstanceSnapshots(ctx, pg)
		if err != nil {
			return nil, statusMessage, err
		}
	} else if pg.Spec.SnapshotFrequency != "" || pg.Spec.SnapshotRetention != "" {
		p.Logger.Warn("postgres instance has only one snapshot field present, skipping snapshotting")
	}
	return instance, statusMessage, err
}

func (p *PostgresProvider) reconcileCloudSQLInstance(ctx context.Context, pg *v1alpha1.Postgres, sqladminService gcpiface.SQLAdminService, strategyConfig *StrategyConfig, address *computepb.Address) (*providers.PostgresInstance, croType.StatusMessage, error) {
	logger := p.Logger.WithField("action", "reconcileCloudSQLInstance")
	logger.Infof("reconciling cloudSQL instance")

	sec, err := buildDefaultCloudSQLSecret(pg)
	if err != nil {
		msg := "failed to build default cloudSQL secret"
		return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}
	result, err := controllerutil.CreateOrUpdate(ctx, p.Client, sec, func() error {
		return nil
	})
	if err != nil {
		errMsg := fmt.Sprintf("failed to create or update secret %s, action was %s", sec.Name, result)
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	gcpInstanceConfig, err := p.buildCloudSQLCreateStrategy(ctx, pg, strategyConfig, sec, address)
	if err != nil {
		msg := "failed to build and verify gcp cloudSQL instance configuration"
		return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}

	foundInstance, err := sqladminService.GetInstance(ctx, strategyConfig.ProjectID, gcpInstanceConfig.Name)
	if err != nil && !resources.IsNotFoundError(err) {
		msg := "cannot retrieve sql instance from gcp"
		return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}
	defer p.exposePostgresInstanceMetrics(ctx, pg, foundInstance)

	if foundInstance != nil {
		if !annotations.Has(pg, ResourceIdentifierAnnotation) {
			annotations.Add(pg, ResourceIdentifierAnnotation, foundInstance.Name)
			if err := p.Client.Update(ctx, pg); err != nil {
				msg := "failed to add annotation to postgres cr"
				return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
			}
		}
		if foundInstance.DatabaseVersion != "" && pg.Status.Version != foundInstance.DatabaseVersion {
			pg.Status.Version = foundInstance.DatabaseVersion
		}
	}

	if foundInstance == nil {
		logger.Infof("no instance found, creating one")
		_, err := sqladminService.CreateInstance(ctx, strategyConfig.ProjectID, gcpInstanceConfig.MapToGcpDatabaseInstance())
		if err != nil && !resources.IsNotFoundError(err) {
			msg := "failed to create cloudSQL instance"
			return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
		}
		annotations.Add(pg, ResourceIdentifierAnnotation, gcpInstanceConfig.Name)
		if err := p.Client.Update(ctx, pg); err != nil {
			msg := "failed to add annotation"
			return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
		}
		msg := "started cloudSQL provision"
		return nil, croType.StatusMessage(msg), nil
	}

	if foundInstance.State == "PENDING_CREATE" {
		msg := fmt.Sprintf("creation of %s cloudSQL instance in progress", foundInstance.Name)
		return nil, croType.StatusMessage(msg), nil
	}

	logger.Infof("building cloudSQL update config for: %s", foundInstance.Name)
	modifiedInstance, err := p.buildCloudSQLUpdateStrategy(gcpInstanceConfig, foundInstance)
	if err != nil {
		msg := "error building update config for cloudsql instance"
		return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}

	if modifiedInstance != nil {
		logger.Infof("modifying cloudSQL instance: %s", foundInstance.Name)
		_, err := sqladminService.ModifyInstance(ctx, strategyConfig.ProjectID, foundInstance.Name, modifiedInstance)
		if err != nil && !resources.IsConflictError(err) {
			msg := fmt.Sprintf("failed to modify cloudsql instance: %s", foundInstance.Name)
			return nil, croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
		}
	}

	var host string
	for i := range foundInstance.IpAddresses {
		if foundInstance.IpAddresses[i].Type == "PRIVATE" {
			host = foundInstance.IpAddresses[i].IpAddress
		}
	}
	pdd := &providers.PostgresDeploymentDetails{
		Username: string(sec.Data[defaultPostgresUserKey]),
		Password: string(sec.Data[defaultPostgresPasswordKey]),
		Host:     host,
		Database: defaultDeploymentDatabase,
		Port:     defaultGCPPostgresPort,
	}
	msg := fmt.Sprintf("successfully reconciled cloudsql instance %s", foundInstance.Name)
	p.Logger.Info(msg)
	return &providers.PostgresInstance{DeploymentDetails: pdd}, croType.StatusMessage(msg), nil
}

// DeletePostgres will set the postgres deletion timestamp, reconcile provider credentials so that the postgres instance
// can be accessed, build the cloudSQL service using these credentials and call the deleteCloudSQLInstance function to
// perform the delete action.
func (p *PostgresProvider) DeletePostgres(ctx context.Context, pg *v1alpha1.Postgres) (croType.StatusMessage, error) {
	logger := p.Logger.WithField("action", "DeletePostgres")
	logger.Infof("reconciling postgres %s", pg.Name)

	p.setPostgresDeletionTimestampMetric(ctx, pg)

	strategyConfig, err := p.ConfigManager.ReadStorageStrategy(ctx, providers.PostgresResourceType, pg.Spec.Tier)
	if err != nil {
		msg := "failed to retrieve postgres strategy config"
		return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}

	creds, err := p.CredentialManager.ReconcileProviderCredentials(ctx, pg.Namespace)
	if err != nil {
		errMsg := fmt.Sprintf("failed to reconcile gcp postgres provider credentials for postgres instance %s", pg.Name)
		return croType.StatusMessage(errMsg), fmt.Errorf("%s: %w", errMsg, err)
	}

	sqlClient, err := gcpiface.NewSQLAdminService(ctx, option.WithCredentialsJSON(creds.ServiceAccountJson), p.Logger)
	if err != nil {
		errMsg := "could not initialise new SQL Admin Service"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	isLastResource, err := resources.IsLastResource(ctx, p.Client)
	if err != nil {
		errMsg := "failed to check if this cr is the last cr of type postgres and redis"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	networkManager, err := NewNetworkManager(ctx, strategyConfig.ProjectID, option.WithCredentialsJSON(creds.ServiceAccountJson), p.Client, logger)
	if err != nil {
		errMsg := "failed to initialise network manager"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	return p.deleteCloudSQLInstance(ctx, networkManager, sqlClient, strategyConfig, pg, isLastResource)
}

// deleteCloudSQLInstance will retrieve the instance required using the cloudSQLDeleteConfig
// and delete this instance if it is not already pending delete. The credentials and finalizer are then removed.
func (p *PostgresProvider) deleteCloudSQLInstance(ctx context.Context, networkManager NetworkManager, sqladminService gcpiface.SQLAdminService, strategyConfig *StrategyConfig, pg *v1alpha1.Postgres, isLastResource bool) (croType.StatusMessage, error) {
	logger := p.Logger.WithField("action", "deleteCloudSQLInstance")

	cloudSQLDeleteConfig, err := p.buildCloudSQLDeleteStrategy(ctx, pg, strategyConfig)
	if err != nil {
		msg := "failed to retrieve postgres strategy config"
		return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}

	foundInstance, err := sqladminService.GetInstance(ctx, strategyConfig.ProjectID, cloudSQLDeleteConfig.Name)
	if err != nil && !resources.IsNotFoundError(err) {
		msg := "cannot retrieve sql instance from gcp"
		return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}

	if foundInstance != nil && foundInstance.Name != "" {
		p.exposePostgresInstanceMetrics(ctx, pg, foundInstance)
		if foundInstance.State == "PENDING_DELETE" {
			statusMessage := fmt.Sprintf("postgres instance %s is already deleting", cloudSQLDeleteConfig.Name)
			p.Logger.Info(statusMessage)
			return croType.StatusMessage(statusMessage), nil
		}
		if foundInstance.Settings.DeletionProtectionEnabled {
			update := &sqladmin.DatabaseInstance{
				Settings: &sqladmin.Settings{
					DeletionProtectionEnabled: false,
					ForceSendFields:           []string{"DeletionProtectionEnabled"},
				},
			}
			_, err := sqladminService.ModifyInstance(ctx, strategyConfig.ProjectID, foundInstance.Name, update)
			if err != nil && !resources.IsConflictError(err) {
				msg := fmt.Sprintf("failed to disable deletion protection for cloudsql instance: %s", foundInstance.Name)
				return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
			}
			msg := fmt.Sprintf("disabling deletion protection for cloudsql instance %s", foundInstance.Name)
			return croType.StatusMessage(msg), nil
		}
		_, err = sqladminService.DeleteInstance(ctx, strategyConfig.ProjectID, foundInstance.Name)
		if err != nil && !resources.IsConflictError(err) {
			msg := fmt.Sprintf("failed to delete cloudsql instance: %s", foundInstance.Name)
			return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
		}
		msg := fmt.Sprintf("deletion in progress for cloudsql instance %s", foundInstance.Name)
		return croType.StatusMessage(msg), nil
	}

	logger.Info("deleting cloudSQL secret")
	sec := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pg.Name + defaultCredSecSuffix,
			Namespace: pg.Namespace,
		},
	}
	err = p.Client.Delete(ctx, sec)
	if err != nil && !k8serr.IsNotFound(err) {
		msg := "failed to delete cloudSQL secrets"
		return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}

	// remove networking components
	if isLastResource {
		if err := networkManager.DeleteNetworkPeering(ctx); err != nil {
			msg := "failed to delete cluster network peering"
			return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
		}
		if err := networkManager.DeleteNetworkService(ctx); err != nil {
			msg := "failed to delete cluster network service"
			return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
		}
		if err := networkManager.DeleteNetworkIpRange(ctx); err != nil {
			msg := "failed to delete network IP range"
			return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
		}
		if exist, err := networkManager.ComponentsExist(ctx); err != nil || exist {
			if exist {
				return croType.StatusMessage("network component deletion in progress"), nil
			}
			msg := "failed to check if components exist"
			return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
		}
	}

	resources.RemoveFinalizer(&pg.ObjectMeta, DefaultFinalizer)
	if err := p.Client.Update(ctx, pg); err != nil {
		msg := "failed to update instance as part of finalizer reconcile"
		return croType.StatusMessage(msg), errorUtil.Wrap(err, msg)
	}
	msg := fmt.Sprintf("successfully deleted gcp postgres instance %s", cloudSQLDeleteConfig.Name)
	p.Logger.Info(msg)

	return croType.StatusMessage(msg), nil
}

// set metrics about the postgres instance being deleted
// works in a similar way to kube_pod_deletion_timestamp
// https://github.com/kubernetes/kube-state-metrics/blob/0bfc2981f9c281c78e33052abdc2d621630562b9/internal/store/pod.go#L200-L218
func (p *PostgresProvider) setPostgresDeletionTimestampMetric(ctx context.Context, pg *v1alpha1.Postgres) {
	if pg.DeletionTimestamp != nil && !pg.DeletionTimestamp.IsZero() {

		instanceName, err := resources.BuildInfraNameFromObject(ctx, p.Client, pg.ObjectMeta, defaultGcpIdentifierLength)
		if err != nil || instanceName == "" {
			p.Logger.Errorf("unable to build instance name")
			return
		}

		p.Logger.Info("setting postgres information metric")
		clusterID, err := resources.GetClusterID(ctx, p.Client)
		if err != nil {
			p.Logger.Errorf("failed to get cluster id while exposing information metric for %v", instanceName)
			return
		}

		labels := buildPostgresStatusMetricsLabels(pg, clusterID, instanceName, pg.Status.Phase)
		resources.SetMetric(resources.DefaultPostgresDeletionMetricName, labels, float64(pg.DeletionTimestamp.Unix()))
	}
}

func buildPostgresGenericMetricLabels(pg *v1alpha1.Postgres, clusterID, instanceName string) map[string]string {
	labels := map[string]string{}
	labels[resources.LabelClusterIDKey] = clusterID
	labels[resources.LabelResourceIDKey] = pg.Name
	labels[resources.LabelNamespaceKey] = pg.Namespace
	labels[resources.LabelInstanceIDKey] = instanceName
	labels[resources.LabelProductNameKey] = pg.Labels["productName"]
	labels[resources.LabelStrategyKey] = postgresProviderName
	return labels
}

func buildPostgresStatusMetricsLabels(cr *v1alpha1.Postgres, clusterID, instanceName string, phase croType.StatusPhase) map[string]string {
	labels := buildPostgresGenericMetricLabels(cr, clusterID, instanceName)
	labels[resources.LabelStatusPhaseKey] = string(phase)
	return labels
}

func (p *PostgresProvider) buildCloudSQLCreateStrategy(ctx context.Context, pg *v1alpha1.Postgres, strategyConfig *StrategyConfig, sec *v1.Secret, address *computepb.Address) (*gcpiface.DatabaseInstance, error) {
	createStrategy := &CreateInstanceRequest{
		Instance: &gcpiface.DatabaseInstance{},
	}
	if err := json.Unmarshal(strategyConfig.CreateStrategy, &createStrategy); err != nil {
		return nil, errorUtil.Wrap(err, "failed to unmarshal gcp postgres create request")
	}
	instance := createStrategy.Instance
	if instance.Name == "" {
		instanceID, err := resources.BuildInfraNameFromObject(ctx, p.Client, pg.ObjectMeta, defaultGcpIdentifierLength)
		if err != nil {
			return nil, errorUtil.Wrap(err, "failed to build gcp postgres instance id from object")
		}
		instance.Name = instanceID
	}
	if instance.DatabaseVersion == "" {
		instance.DatabaseVersion = defaultGCPCLoudSQLDatabaseVersion
	}
	if instance.Region == "" {
		instance.Region = strategyConfig.Region
	}
	if instance.RootPassword == "" {
		instance.RootPassword = string(sec.Data[defaultPostgresPasswordKey])
	}

	tags, err := buildDefaultPostgresTags(ctx, p.Client, pg)
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to build gcp postgres instance tags")
	}

	if instance.Settings == nil {
		instance.Settings = &gcpiface.Settings{}
	}
	if instance.Settings.UserLabels == nil {
		instance.Settings.UserLabels = map[string]string{}
	}
	for key, value := range tags {
		instance.Settings.UserLabels[key] = value
	}

	if instance.Settings.Tier == "" {
		instance.Settings.Tier = defaultTier
	}
	if instance.Settings.AvailabilityType == "" {
		instance.Settings.AvailabilityType = defaultAvailabilityType
	}
	if instance.Settings.StorageAutoResizeLimit == 0 {
		instance.Settings.StorageAutoResizeLimit = defaultStorageAutoResizeLimit
	}
	if instance.Settings.BackupConfiguration == nil {
		instance.Settings.BackupConfiguration = &gcpiface.BackupConfiguration{}
	}
	if instance.Settings.BackupConfiguration.Enabled == nil {
		instance.Settings.BackupConfiguration.Enabled = ptr.To(defaultBackupConfigEnabled)
	}
	if instance.Settings.BackupConfiguration.PointInTimeRecoveryEnabled == nil {
		instance.Settings.BackupConfiguration.PointInTimeRecoveryEnabled = ptr.To(defaultPointInTimeRecoveryEnabled)
	}
	if instance.Settings.BackupConfiguration.BackupRetentionSettings == nil {
		instance.Settings.BackupConfiguration.BackupRetentionSettings = &gcpiface.BackupRetentionSettings{}
	}
	if instance.Settings.BackupConfiguration.BackupRetentionSettings.RetentionUnit == "" {
		instance.Settings.BackupConfiguration.BackupRetentionSettings.RetentionUnit = defaultBackupRetentionSettingsRetentionUnit
	}
	if instance.Settings.BackupConfiguration.BackupRetentionSettings.RetainedBackups == 0 {
		instance.Settings.BackupConfiguration.BackupRetentionSettings.RetainedBackups = defaultBackupRetentionSettingsRetainedBackups
	}
	if instance.Settings.DataDiskSizeGb == 0 {
		instance.Settings.DataDiskSizeGb = defaultDataDiskSizeGb
	}
	if instance.Settings.DeletionProtectionEnabled == nil {
		instance.Settings.DeletionProtectionEnabled = ptr.To(defaultDeleteProtectionEnabled)
	}
	if instance.Settings.IpConfiguration == nil {
		instance.Settings.IpConfiguration = &gcpiface.IpConfiguration{}
	}
	if instance.Settings.IpConfiguration.Ipv4Enabled == nil {
		instance.Settings.IpConfiguration.Ipv4Enabled = ptr.To(defaultIPConfigIPV4Enabled)
	}
	if instance.Settings.IpConfiguration.AllocatedIpRange == "" {
		instance.Settings.IpConfiguration.AllocatedIpRange = address.GetName()
	}
	if instance.Settings.IpConfiguration.PrivateNetwork == "" {
		instance.Settings.IpConfiguration.PrivateNetwork = address.GetNetwork()
	}
	return instance, nil
}

func (p *PostgresProvider) buildCloudSQLDeleteStrategy(ctx context.Context, pg *v1alpha1.Postgres, strategyConfig *StrategyConfig) (*sqladmin.DatabaseInstance, error) {
	deleteStrategy := &sqladmin.DatabaseInstance{}
	if err := json.Unmarshal(strategyConfig.DeleteStrategy, deleteStrategy); err != nil {
		return nil, errorUtil.Wrap(err, "failed to unmarshal gcp postgres delete request")
	}
	if deleteStrategy.Name == "" {
		instanceID, err := resources.BuildInfraNameFromObject(ctx, p.Client, pg.ObjectMeta, defaultGcpIdentifierLength)
		if err != nil {
			return nil, errorUtil.Wrap(err, "failed to build gcp postgres instance id from object")
		}
		deleteStrategy.Name = instanceID
	}
	return deleteStrategy, nil
}

func (p *PostgresProvider) buildCloudSQLUpdateStrategy(cloudSQLConfig *gcpiface.DatabaseInstance, foundInstance *sqladmin.DatabaseInstance) (*sqladmin.DatabaseInstance, error) {
	p.Logger.Debugf("verifying that %s configuration is as expected", foundInstance.Name)

	updateFound := false
	modifiedInstance := &sqladmin.DatabaseInstance{}

	if cloudSQLConfig.Settings != nil && foundInstance.Settings != nil {
		modifiedInstance.Settings = &sqladmin.Settings{
			ForceSendFields: []string{},
		}

		if cloudSQLConfig.Settings.DeletionProtectionEnabled != nil && *cloudSQLConfig.Settings.DeletionProtectionEnabled != foundInstance.Settings.DeletionProtectionEnabled {
			modifiedInstance.Settings.DeletionProtectionEnabled = *cloudSQLConfig.Settings.DeletionProtectionEnabled
			modifiedInstance.Settings.ForceSendFields = append(modifiedInstance.Settings.ForceSendFields, "DeletionProtectionEnabled")
			updateFound = true
		}

		if cloudSQLConfig.Settings.StorageAutoResize != nil && *cloudSQLConfig.Settings.StorageAutoResize != *foundInstance.Settings.StorageAutoResize {
			modifiedInstance.Settings.StorageAutoResize = cloudSQLConfig.Settings.StorageAutoResize
			modifiedInstance.Settings.ForceSendFields = append(modifiedInstance.Settings.ForceSendFields, "StorageAutoResize")
			updateFound = true
		}

		if cloudSQLConfig.Settings.Tier != foundInstance.Settings.Tier {
			modifiedInstance.Settings.Tier = cloudSQLConfig.Settings.Tier
			updateFound = true
		}
		if cloudSQLConfig.Settings.AvailabilityType != foundInstance.Settings.AvailabilityType {
			modifiedInstance.Settings.AvailabilityType = cloudSQLConfig.Settings.AvailabilityType
			updateFound = true
		}
		if cloudSQLConfig.Settings.StorageAutoResizeLimit != foundInstance.Settings.StorageAutoResizeLimit {
			modifiedInstance.Settings.StorageAutoResizeLimit = cloudSQLConfig.Settings.StorageAutoResizeLimit
			updateFound = true
		}
		if cloudSQLConfig.Settings.DataDiskSizeGb != foundInstance.Settings.DataDiskSizeGb {
			modifiedInstance.Settings.DataDiskSizeGb = cloudSQLConfig.Settings.DataDiskSizeGb
			updateFound = true
		}

		userLabelsMatch := reflect.DeepEqual(cloudSQLConfig.Settings.UserLabels, foundInstance.Settings.UserLabels)
		if !userLabelsMatch {
			modifiedInstance.Settings.UserLabels = cloudSQLConfig.Settings.UserLabels
			updateFound = true
		}
	}

	if cloudSQLConfig.Settings.BackupConfiguration != nil && foundInstance.Settings.BackupConfiguration != nil {
		modifiedInstance.Settings.BackupConfiguration = &sqladmin.BackupConfiguration{
			ForceSendFields: []string{},
		}
		if cloudSQLConfig.Settings.BackupConfiguration.Enabled != nil && *cloudSQLConfig.Settings.BackupConfiguration.Enabled != foundInstance.Settings.BackupConfiguration.Enabled {
			modifiedInstance.Settings.BackupConfiguration.Enabled = *cloudSQLConfig.Settings.BackupConfiguration.Enabled
			modifiedInstance.Settings.BackupConfiguration.ForceSendFields = append(modifiedInstance.Settings.BackupConfiguration.ForceSendFields, "Enabled")
			updateFound = true
		}
		if cloudSQLConfig.Settings.BackupConfiguration.PointInTimeRecoveryEnabled != nil && *cloudSQLConfig.Settings.BackupConfiguration.PointInTimeRecoveryEnabled != foundInstance.Settings.BackupConfiguration.PointInTimeRecoveryEnabled {
			modifiedInstance.Settings.BackupConfiguration.PointInTimeRecoveryEnabled = *cloudSQLConfig.Settings.BackupConfiguration.PointInTimeRecoveryEnabled
			modifiedInstance.Settings.BackupConfiguration.ForceSendFields = append(modifiedInstance.Settings.BackupConfiguration.ForceSendFields, "PointInTimeRecoveryEnabled")
			updateFound = true
		}
	}

	if cloudSQLConfig.Settings.BackupConfiguration.BackupRetentionSettings != nil && foundInstance.Settings.BackupConfiguration.BackupRetentionSettings != nil {
		modifiedInstance.Settings.BackupConfiguration.BackupRetentionSettings = &sqladmin.BackupRetentionSettings{}

		if cloudSQLConfig.Settings.BackupConfiguration.BackupRetentionSettings.RetentionUnit != foundInstance.Settings.BackupConfiguration.BackupRetentionSettings.RetentionUnit {
			modifiedInstance.Settings.BackupConfiguration.BackupRetentionSettings.RetentionUnit = cloudSQLConfig.Settings.BackupConfiguration.BackupRetentionSettings.RetentionUnit
			updateFound = true
		}
		if cloudSQLConfig.Settings.BackupConfiguration.BackupRetentionSettings.RetainedBackups != foundInstance.Settings.BackupConfiguration.BackupRetentionSettings.RetainedBackups {
			modifiedInstance.Settings.BackupConfiguration.BackupRetentionSettings.RetainedBackups = cloudSQLConfig.Settings.BackupConfiguration.BackupRetentionSettings.RetainedBackups
			updateFound = true
		}
	}

	if cloudSQLConfig.Settings.IpConfiguration != nil && foundInstance.Settings.IpConfiguration != nil {
		modifiedInstance.Settings.IpConfiguration = &sqladmin.IpConfiguration{
			ForceSendFields: []string{},
		}
		if cloudSQLConfig.Settings.IpConfiguration.Ipv4Enabled != nil && *cloudSQLConfig.Settings.IpConfiguration.Ipv4Enabled != foundInstance.Settings.IpConfiguration.Ipv4Enabled {
			modifiedInstance.Settings.IpConfiguration.Ipv4Enabled = *cloudSQLConfig.Settings.IpConfiguration.Ipv4Enabled
			modifiedInstance.Settings.IpConfiguration.ForceSendFields = append(modifiedInstance.Settings.IpConfiguration.ForceSendFields, "Ipv4Enabled")
			updateFound = true
		}
	}

	if cloudSQLConfig.Settings.MaintenanceWindow != nil && foundInstance.Settings.MaintenanceWindow != nil {
		modifiedInstance.Settings.MaintenanceWindow = &sqladmin.MaintenanceWindow{
			ForceSendFields: []string{},
		}
		if cloudSQLConfig.Settings.MaintenanceWindow.Day != nil && *cloudSQLConfig.Settings.MaintenanceWindow.Day != foundInstance.Settings.MaintenanceWindow.Day {
			modifiedInstance.Settings.MaintenanceWindow.Day = *cloudSQLConfig.Settings.MaintenanceWindow.Day
			updateFound = true
		}
		if cloudSQLConfig.Settings.MaintenanceWindow.Hour != nil && *cloudSQLConfig.Settings.MaintenanceWindow.Hour != foundInstance.Settings.MaintenanceWindow.Hour {
			modifiedInstance.Settings.MaintenanceWindow.Hour = *cloudSQLConfig.Settings.MaintenanceWindow.Hour
			modifiedInstance.Settings.MaintenanceWindow.ForceSendFields = append(modifiedInstance.Settings.MaintenanceWindow.ForceSendFields, "Hour")
			updateFound = true
		}
	}

	if cloudSQLConfig.DatabaseVersion != foundInstance.DatabaseVersion {
		modifiedInstance.DatabaseVersion = cloudSQLConfig.DatabaseVersion
		updateFound = true
	}

	if !updateFound {
		return nil, nil
	}

	return modifiedInstance, nil
}

func (p *PostgresProvider) exposePostgresInstanceMetrics(ctx context.Context, pg *v1alpha1.Postgres, instance *sqladmin.DatabaseInstance) {
	if instance == nil {
		return
	}
	clusterID, err := resources.GetClusterID(ctx, p.Client)
	if err != nil {
		p.Logger.Errorf("failed to get cluster id while exposing metrics for postgres instance %s", instance.Name)
		return
	}
	genericLabels := resources.BuildGenericMetricLabels(pg.ObjectMeta, clusterID, instance.Name, postgresProviderName)
	instanceState := instance.State
	infoLabels := resources.BuildInfoMetricLabels(pg.ObjectMeta, instanceState, clusterID, instance.Name, postgresProviderName)
	resources.SetMetricCurrentTime(resources.DefaultPostgresInfoMetricName, infoLabels)
	// a single metric should be exposed for each possible status phase
	// the value of the metric should be 1.0 when the resource is in that phase
	// the value of the metric should be 0.0 when the resource is not in that phase
	for _, phase := range []croType.StatusPhase{croType.PhaseFailed, croType.PhaseDeleteInProgress, croType.PhasePaused, croType.PhaseComplete, croType.PhaseInProgress} {
		labelsFailed := resources.BuildStatusMetricsLabels(pg.ObjectMeta, clusterID, instance.Name, postgresProviderName, phase)
		resources.SetMetric(resources.DefaultPostgresStatusMetricName, labelsFailed, resources.Btof64(pg.Status.Phase == phase))
	}
	// set availability metric, based on the status flag on the cloudsql postgres instance in gcp
	// the value of the metric should be 0 when the instance state is unhealthy
	// the value of the metric should be 1 when the instance state is healthy
	// more details on possible state values here: https://pkg.go.dev/google.golang.org/api/sqladmin/v1beta4@v0.105.0#DatabaseInstance.State
	var instanceHealthy float64
	var instanceConnectable float64
	if resources.Contains(healthyPostgresInstanceStates(), instanceState) {
		instanceHealthy = 1
		if len(instance.IpAddresses) > 0 {
			var host string
			for i := range instance.IpAddresses {
				if instance.IpAddresses[i].Type == "PRIVATE" {
					host = instance.IpAddresses[i].IpAddress
				}
			}
			if success := p.TCPPinger.TCPConnection(host, defaultGCPPostgresPort); success {
				instanceConnectable = 1
			}
		}
	}
	resources.SetMetric(resources.DefaultPostgresAvailMetricName, genericLabels, instanceHealthy)
	resources.SetMetric(resources.DefaultPostgresConnectionMetricName, genericLabels, instanceConnectable)
}

func healthyPostgresInstanceStates() []string {
	return []string{
		"PENDING_CREATE",
		"RUNNABLE",
		"PENDING_DELETE",
	}
}

func buildDefaultCloudSQLSecret(p *v1alpha1.Postgres) (*v1.Secret, error) {
	password, err := resources.GeneratePassword()
	if err != nil {
		errMsg := "failed to generate password"
		return nil, errorUtil.Wrap(err, errMsg)
	}
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name + defaultCredSecSuffix,
			Namespace: p.Namespace,
		},
		StringData: map[string]string{
			defaultPostgresUserKey:     defaultGCPPostgresUser,
			defaultPostgresPasswordKey: password,
		},
		Type: v1.SecretTypeOpaque,
	}, nil
}

func (p *PostgresProvider) reconcileCloudSqlInstanceSnapshots(ctx context.Context, pg *v1alpha1.Postgres) (croType.StatusMessage, error) {
	snapshotRetention, err := str2duration.ParseDuration(string(pg.Spec.SnapshotRetention))
	if err != nil {
		errMsg := fmt.Sprintf("failed to parse %q into go duration", pg.Spec.SnapshotRetention)
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	snapshots, err := getAllSnapshotsForInstance(ctx, p.Client, pg.Name, pg.Namespace)
	if err != nil {
		errMsg := fmt.Sprintf("failed to fetch all snapshots associated with postgres instance %s", pg.Name)
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	if len(snapshots) == 0 {
		err := p.createSnapshot(ctx, pg)
		if err != nil {
			errMsg := fmt.Sprintf("failed to create postgres snapshot for %s", pg.Name)
			return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
		msg := fmt.Sprintf("created postgres snapshot CR for instance %s", pg.Name)
		return croType.StatusMessage(msg), nil
	}
	latestSnapshot, err := getLatestPostgresSnapshot(ctx, p.Client, pg.Name, pg.Namespace)
	if err != nil {
		errMsg := fmt.Sprintf("failed to determine latest snapshot id for instance %s", pg.Name)
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	if latestSnapshot == nil {
		msg := fmt.Sprintf("latest snapshot creation in progress for instance %s", pg.Name)
		return croType.StatusMessage(msg), nil
	}
	for i := range snapshots {
		if snapshots[i].Name == latestSnapshot.Name {
			continue
		}
		retainUntil := snapshots[i].CreationTimestamp.Add(snapshotRetention)
		if time.Now().After(retainUntil) {
			p.Logger.Infof("deleting snapshot %s because its retention has expired", snapshots[i].Name)
			err := p.Client.Delete(ctx, snapshots[i])
			if err != nil {
				errMsg := fmt.Sprintf("failed to delete postgres snapshot %s", snapshots[i].Name)
				return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
			}
		}
	}
	snapshotFrequency, err := str2duration.ParseDuration(string(pg.Spec.SnapshotFrequency))
	if err != nil {
		errMsg := fmt.Sprintf("failed to parse %q into go duration", pg.Spec.SnapshotFrequency)
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].GetCreationTimestamp().After(snapshots[j].GetCreationTimestamp().Time)
	})
	nextSnapshotTime := snapshots[0].CreationTimestamp.Add(snapshotFrequency)
	if time.Now().After(nextSnapshotTime) {
		err = p.createSnapshot(ctx, pg)
		if err != nil {
			errMsg := fmt.Sprintf("failed to create postgres snapshot for %s", pg.Name)
			return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
		}
	}
	msg := fmt.Sprintf("successfully reconciled postgres instance %s snapshots", pg.Name)
	p.Logger.Info(msg)
	return croType.StatusMessage(msg), nil
}

func (p *PostgresProvider) createSnapshot(ctx context.Context, pg *v1alpha1.Postgres) error {
	instanceName := annotations.Get(pg, ResourceIdentifierAnnotation)
	p.Logger.Infof("creating new snapshot for postgres instance %s", instanceName)
	snapshot := &v1alpha1.PostgresSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: pg.Name,
			Namespace:    pg.Namespace,
		},
		Spec: v1alpha1.PostgresSnapshotSpec{
			ResourceName: pg.Name,
		},
	}
	return p.Client.Create(ctx, snapshot)
}
