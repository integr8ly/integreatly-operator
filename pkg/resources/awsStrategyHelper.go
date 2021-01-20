package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	"github.com/aws/aws-sdk-go/service/elasticache"

	"github.com/aws/aws-sdk-go/service/rds"

	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/client"

	awsSdk "github.com/aws/aws-sdk-go/aws"
	"github.com/integr8ly/cloud-resource-operator/pkg/providers/aws"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
)

const (
	configMapName    = "cloud-resources-aws-strategies"
	PostgresStratKey = "postgres"
	RedisStratKey    = "redis"
)

// At the moment this function only allows for overriding the createStrategy. It's unlikely that delete or region will need to be
// overridden. In the case that they do, the function can be extended.
func CreatePostgresTierFromProduction(ctx context.Context, client k8sclient.Client, ns string, useClusterStorage bool, tier string, createStrategy map[string]interface{}, logger l.Logger) error {

	if useClusterStorage {
		return nil
	}

	logger.Infof("Creating AWS Postgres Strategy", l.Fields{"tier": tier})

	err, data := getResourceTierData(ctx, client, ns, PostgresStratKey, croUtil.TierProduction, logger)
	if err != nil {
		return err
	}

	err, newCreateStrategy := overrideCreateDBStrategy(data.CreateStrategy, createStrategy)
	if err != nil {
		logger.Error("Error overriding AWS Strategy", err)
		return err
	}

	newCreateStrategyJSON, err := json.Marshal(newCreateStrategy)
	if err != nil {
		return err
	}

	data.CreateStrategy = newCreateStrategyJSON

	err = updateConfigMapWithTier(ctx, client, ns, PostgresStratKey, tier, data, true, logger)
	if err != nil {
		return err
	}

	return nil
}

// At the moment this function only allows for overriding the createStrategy. It's unlikely that delete or region will need to be
// overridden. In the case that they do, the function can be extended.
func CreateRedisTierFromProduction(ctx context.Context, client k8sclient.Client, ns string, useClusterStorage bool, tier string, createStrategy map[string]interface{}, logger l.Logger) error {

	if useClusterStorage {
		return nil
	}

	logger.Infof("Creating AWS Redis Strategy", l.Fields{"tier": tier})

	err, data := getResourceTierData(ctx, client, ns, RedisStratKey, croUtil.TierProduction, logger)
	if err != nil {
		return err
	}

	err, newCreateStrategy := overrideCreateCacheStrategy(data.CreateStrategy, createStrategy)
	if err != nil {
		logger.Error("Error overriding AWS Strategy", err)
		return err
	}
	newCreateStrategyJSON, err := json.Marshal(newCreateStrategy)
	if err != nil {
		logger.Error("Error marshalling json", err)
		return err
	}

	data.CreateStrategy = newCreateStrategyJSON

	err = updateConfigMapWithTier(ctx, client, ns, RedisStratKey, tier, data, true, logger)
	if err != nil {
		return err
	}

	return nil
}

func updateConfigMapWithTier(ctx context.Context, client k8sclient.Client, ns string, resourceType string, tier string, data *aws.StrategyConfig, addNetworkUpdate bool, logger l.Logger) error {

	// Get the current config map
	cfgMap := &corev1.ConfigMap{}
	if err := client.Get(ctx, k8sclient.ObjectKey{
		Name:      configMapName,
		Namespace: ns,
	}, cfgMap); err != nil {
		logger.Errorf("Error getting resource", l.Fields{"cm": configMapName}, err)
		return err
	}

	// Get the entire postgres as a string
	postgresData := cfgMap.Data[resourceType]

	// Convert the string to an object
	strategyConfig := map[string]*aws.StrategyConfig{}
	if err := json.Unmarshal([]byte(postgresData), &strategyConfig); err != nil {
		logger.Errorf("Error Unmarshaling strategy mapping", l.Fields{"resourceType": resourceType}, err)
		return err
	}

	// Add new tier
	strategyConfig[tier] = data

	// Convert the object back to JSON
	strategyConfigJSON, err := json.Marshal(strategyConfig)
	if err != nil {
		return err
	}

	// Update the resource type
	cfgMap.Data[resourceType] = string(strategyConfigJSON)

	// If adding a new Postgres tier it will look for a network tier of the same name
	if addNetworkUpdate {
		err, newNetworkData := addNewNetworkTier(cfgMap.Data["_network"], tier, logger)
		if err != nil {
			return err
		}
		cfgMap.Data["_network"] = newNetworkData
	}

	logger.Infof("Creating AWS Strategy Config Map", l.Fields{"tier": tier})

	return client.Patch(ctx, cfgMap, k8sclient.Merge)
}

// When adding a new postgres tier we also need to add a corresponding network tier
// Duplicate the existing prod tier
func addNewNetworkTier(networkData string, tier string, logger l.Logger) (error, string) {
	type TierCreateStrategy struct {
		CreateStrategy struct {
			CidrBlock *string `json:"CidrBlock"`
		} `json:"createStrategy"`
	}
	var network map[string]TierCreateStrategy

	if err := json.Unmarshal([]byte(networkData), &network); err != nil {
		logger.Error("Error unmarshalling network JSON", err)
		return err, ""
	}

	// add new Tier to array based on production
	network[tier] = network["production"]

	networkJSON, err := json.Marshal(network)
	if err != nil {
		logger.Error("Error marshalling network JSON", err)
		return err, ""
	}

	return nil, string(networkJSON)
}

func overrideCreateDBStrategy(origin json.RawMessage, overrides map[string]interface{}) (error, *rds.CreateDBInstanceInput) {

	originRdsCreateConfig := &rds.CreateDBInstanceInput{}
	if err := json.Unmarshal([]byte(origin), originRdsCreateConfig); err != nil {
		return fmt.Errorf("failed to unmarshal aws rds cluster config %v", err), nil
	}

	for key, value := range overrides {
		if key == "AllocatedStorage" {
			v, ok := value.(int)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.AllocatedStorage = awsSdk.Int64(int64(v))
		}
		if key == "AutoMinorVersionUpgrade" {
			v, ok := value.(bool)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.AutoMinorVersionUpgrade = awsSdk.Bool(v)
		}

		if key == "AvailabilityZone" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.AvailabilityZone = awsSdk.String(v)
		}

		if key == "BackupRetentionPeriod" {
			v, ok := value.(int)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.BackupRetentionPeriod = awsSdk.Int64(int64(v))
		}

		if key == "CharacterSetName" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.CharacterSetName = awsSdk.String(v)
		}

		if key == "CopyTagsToSnapshot" {
			v, ok := value.(bool)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.CopyTagsToSnapshot = awsSdk.Bool(v)
		}

		if key == "DBClusterIdentifier" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.DBClusterIdentifier = awsSdk.String(v)
		}

		if key == "DBInstanceClass" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.DBInstanceClass = awsSdk.String(v)
		}

		if key == "DBInstanceIdentifier" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.DBInstanceIdentifier = awsSdk.String(v)
		}

		if key == "DBName" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.DBName = awsSdk.String(v)
		}

		if key == "DBParameterGroupName" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.DBParameterGroupName = awsSdk.String(v)
		}

		if key == "DBSecurityGroups" {
			v, ok := value.([]string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.DBSecurityGroups = awsSdk.StringSlice(v)
		}

		if key == "DBSubnetGroupName" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.DBSubnetGroupName = awsSdk.String(v)
		}

		if key == "DeletionProtection" {
			v, ok := value.(bool)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.DeletionProtection = awsSdk.Bool(v)
		}

		if key == "Domain" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.Domain = awsSdk.String(v)
		}

		if key == "DomainIAMRoleName" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.DomainIAMRoleName = awsSdk.String(v)
		}

		if key == "EnableCloudwatchLogsExports" {
			v, ok := value.([]string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.EnableCloudwatchLogsExports = awsSdk.StringSlice(v)
		}

		if key == "EnableIAMDatabaseAuthentication" {
			v, ok := value.(bool)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.EnableIAMDatabaseAuthentication = awsSdk.Bool(v)
		}

		if key == "EnablePerformanceInsights" {
			v, ok := value.(bool)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.EnablePerformanceInsights = awsSdk.Bool(v)
		}

		if key == "Engine" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.Engine = awsSdk.String(v)
		}

		if key == "EngineVersion" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.EngineVersion = awsSdk.String(v)
		}

		if key == "Iops" {
			v, ok := value.(int)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.Iops = awsSdk.Int64(int64(v))
		}

		if key == "KmsKeyId" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.KmsKeyId = awsSdk.String(v)
		}

		if key == "LicenseModel" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.LicenseModel = awsSdk.String(v)
		}

		if key == "MasterUserPassword" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.MasterUserPassword = awsSdk.String(v)
		}

		if key == "MasterUsername" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.MasterUsername = awsSdk.String(v)
		}

		if key == "MaxAllocatedStorage" {
			v, ok := value.(int)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.MaxAllocatedStorage = awsSdk.Int64(int64(v))
		}

		if key == "MonitoringInterval" {
			v, ok := value.(int)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.MonitoringInterval = awsSdk.Int64(int64(v))
		}

		if key == "MonitoringRoleArn" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.MonitoringRoleArn = awsSdk.String(v)
		}

		if key == "MultiAZ" {
			v, ok := value.(bool)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.MultiAZ = awsSdk.Bool(v)
		}

		if key == "OptionGroupName" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.OptionGroupName = awsSdk.String(v)
		}

		if key == "PerformanceInsightsKMSKeyId" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.PerformanceInsightsKMSKeyId = awsSdk.String(v)
		}

		if key == "PerformanceInsightsRetentionPeriod" {
			v, ok := value.(int)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.PerformanceInsightsRetentionPeriod = awsSdk.Int64(int64(v))
		}

		if key == "Port" {
			v, ok := value.(int)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.Port = awsSdk.Int64(int64(v))
		}

		if key == "PreferredBackupWindow" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.PreferredBackupWindow = awsSdk.String(v)
		}

		if key == "PreferredMaintenanceWindow" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.PreferredMaintenanceWindow = awsSdk.String(v)
		}

		if key == "PromotionTier" {
			v, ok := value.(int)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.PromotionTier = awsSdk.Int64(int64(v))
		}

		if key == "PubliclyAccessible" {
			v, ok := value.(bool)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.PubliclyAccessible = awsSdk.Bool(v)
		}

		if key == "StorageEncrypted" {
			v, ok := value.(bool)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.StorageEncrypted = awsSdk.Bool(v)
		}

		if key == "StorageType" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.StorageType = awsSdk.String(v)
		}

		if key == "TdeCredentialArn" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.TdeCredentialArn = awsSdk.String(v)
		}

		if key == "TdeCredentialPassword" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.TdeCredentialPassword = awsSdk.String(v)
		}

		if key == "Timezone" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.Timezone = awsSdk.String(v)
		}

		if key == "VpcSecurityGroupIds" {
			v, ok := value.([]string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originRdsCreateConfig.VpcSecurityGroupIds = awsSdk.StringSlice(v)
		}
	}

	return nil, originRdsCreateConfig
}

func overrideCreateCacheStrategy(origin json.RawMessage, overrides map[string]interface{}) (error, *elasticache.CreateCacheClusterInput) {

	originCacheCreateConfig := &elasticache.CreateCacheClusterInput{}
	if err := json.Unmarshal([]byte(origin), originCacheCreateConfig); err != nil {
		return fmt.Errorf("failed to unmarshal aws rds cluster config %v", err), nil
	}

	for key, value := range overrides {

		if key == "AZMode" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.AZMode = awsSdk.String(v)
		}
		if key == "AuthToken" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.AuthToken = awsSdk.String(v)
		}
		if key == "AutoMinorVersionUpgrade" {
			v, ok := value.(bool)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.AutoMinorVersionUpgrade = awsSdk.Bool(v)
		}
		if key == "CacheClusterId" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.CacheClusterId = awsSdk.String(v)
		}
		if key == "CacheNodeType" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.CacheNodeType = awsSdk.String(v)
		}
		if key == "CacheParameterGroupName" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.CacheParameterGroupName = awsSdk.String(v)
		}
		if key == "CacheSecurityGroupNames" {
			v, ok := value.([]string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.CacheSecurityGroupNames = awsSdk.StringSlice(v)
		}
		if key == "CacheSubnetGroupName" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.CacheSubnetGroupName = awsSdk.String(v)
		}
		if key == "Engine" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.Engine = awsSdk.String(v)
		}
		if key == "EngineVersion" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.EngineVersion = awsSdk.String(v)
		}
		if key == "NotificationTopicArn" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.NotificationTopicArn = awsSdk.String(v)
		}
		if key == "NumCacheNodes" {
			v, ok := value.(int)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.NumCacheNodes = awsSdk.Int64(int64(v))
		}
		if key == "Port" {
			v, ok := value.(int)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.Port = awsSdk.Int64(int64(v))
		}
		if key == "PreferredAvailabilityZone" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.PreferredAvailabilityZone = awsSdk.String(v)
		}
		if key == "PreferredAvailabilityZones" {
			v, ok := value.([]string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.PreferredAvailabilityZones = awsSdk.StringSlice(v)
		}
		if key == "PreferredMaintenanceWindow" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.PreferredMaintenanceWindow = awsSdk.String(v)
		}
		if key == "ReplicationGroupId" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.ReplicationGroupId = awsSdk.String(v)
		}
		if key == "SecurityGroupIds" {
			v, ok := value.([]string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.SecurityGroupIds = awsSdk.StringSlice(v)
		}
		if key == "SnapshotArns" {
			v, ok := value.([]string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.SnapshotArns = awsSdk.StringSlice(v)
		}
		if key == "SnapshotName" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.SnapshotName = awsSdk.String(v)
		}
		if key == "SnapshotRetentionLimit" {
			v, ok := value.(int)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.SnapshotRetentionLimit = awsSdk.Int64(int64(v))
		}
		if key == "SnapshotWindow" {
			v, ok := value.(string)
			if !ok {
				return errors.New(fmt.Sprintf("Unable to parse key: %s", key)), nil
			}
			originCacheCreateConfig.SnapshotWindow = awsSdk.String(v)
		}

	}

	return nil, originCacheCreateConfig
}

func getResourceTierData(ctx context.Context, client k8sclient.Client, ns string, resourceType string, tier string, logger l.Logger) (error, *aws.StrategyConfig) {
	croStrategyConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: ns,
		},
	}

	err := client.Get(ctx, k8sclient.ObjectKey{Name: configMapName, Namespace: ns}, croStrategyConfig)
	if err != nil {
		logger.Errorf("Failed to get resource", l.Fields{"cm": configMapName}, err)
		return err, nil
	}

	// for example postgres block as string
	resource := croStrategyConfig.Data[resourceType]

	strategyConfig := map[string]*aws.StrategyConfig{}
	if err := json.Unmarshal([]byte(resource), &strategyConfig); err != nil {
		logger.Errorf("Failed to unmarshal strategy mapping", l.Fields{"resourceType": resourceType}, err)
		return err, nil
	}
	if strategyConfig[tier] == nil {
		logger.Errorf("Invalid tier for strategy", l.Fields{"tier": tier}, err)
		return err, nil
	}
	return nil, strategyConfig[tier]
}
