package common

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	CROStrategyConfigMap = "cloud-resources-aws-strategies"
)

/*
  This test is to verify that changes made to the RHMI config are reflected in the
  Cloud Resource Operator (CRO) strategy override config map
  CRO uses this strategy to override the parameters to resources CRO provisions
  RHMI config allows for Maintenance and Backup times
  We need to ensure these windows are built and updated in the config map

  NOTE :
	* This test case is : A21
	* This is to be run as part of the common test cases
	* Some of the functions are shared with functional/aws_strategy_override.go
	* aws_strategy_override.go tests full e2e from updating RHMIConfig to updating AWS resource
*/
func TestRHMIConfigCROStrategyOverride(t TestingTB, testingCtx *TestingContext) {
	ctx := context.TODO()

	// rhmi config we need to use is the rhmi config provisioned in the RHMI install
	// this to avoid a conflict of having multiple rhmi configs
	rhmiConfig := RHMIConfigTemplate()

	// known valid Backup and Maintenance times
	expectedMaintenanceAndBackup := MaintenanceBackup{
		Backup: v1alpha1.Backup{
			ApplyOn: "12:05",
		},
		Maintenance: v1alpha1.Maintenance{
			ApplyFrom: "Thu 14:15",
		},
	}
	// expected windows built from `expectedMaintenanceAndBackup
	expectedBackupWindow := "12:05-13:05"
	expectedMaintenanceWindow := "thu:14:15-thu:15:15"

	// update rhmi config Backup and Maintenance times to valid expected times
	if err := UpdateRHMIConfigBackupAndMaintenance(ctx, testingCtx.Client, rhmiConfig, expectedMaintenanceAndBackup); err != nil {
		t.Fatalf("test failed - unable to update rhmi config : %v", err)
	}

	// ensure strategy config map is as expected
	// we have added this poll to allow the operator reconcile on the cr and update the strategy override
	// we expect the change to be immediate, the poll is help with any potential test flake
	var lastPollError error
	if err := wait.PollImmediate(time.Second*5, time.Second*30, func() (done bool, err error) {
		lastPollError = VerifyCROStrategyMap(ctx, testingCtx.Client, expectedBackupWindow, expectedMaintenanceWindow)
		// If lastPollError is now nil, we've succeeded, return true for 'done' to exit the poll
		return lastPollError == nil, nil
	}); err != nil {
		t.Fatalf("test failed : strategy map not as expected : %v : %v ", lastPollError, err)
	}
}

// gets cro strategy map, checks both redis and postgres values to be as expected.
func VerifyCROStrategyMap(context context.Context, client client.Client, expectedBackupWindow, expectedMaintenanceWindow string) error {
	var foundErrors []string

	croStrategyConfig := &v12.ConfigMap{}
	if err := client.Get(context, types.NamespacedName{Name: CROStrategyConfigMap, Namespace: RHMIOperatorNamespace}, croStrategyConfig); err != nil {
		return fmt.Errorf("unable to get cloud-resources-aws-strategies config map : %v", err)
	}

	redisExpectedBackupWindow := fmt.Sprintf("\"SnapshotWindow\":\"%s\"", expectedBackupWindow)
	if !strings.Contains(croStrategyConfig.Data["redis"], redisExpectedBackupWindow) {
		foundErrors = append(foundErrors, fmt.Sprintf("\n - expected '%s' not found in Redis strategy config map", redisExpectedBackupWindow))
	}
	redisExpectedMaintenaceWindow := fmt.Sprintf("\"PreferredMaintenanceWindow\":\"%s\"", expectedMaintenanceWindow)
	if !strings.Contains(croStrategyConfig.Data["redis"], redisExpectedMaintenaceWindow) {
		foundErrors = append(foundErrors, fmt.Sprintf("\n - expected '%s' not found in Redis strategy config map", redisExpectedMaintenaceWindow))
	}

	postgresExpectedBackupWindow := fmt.Sprintf("\"PreferredBackupWindow\":\"%s\"", expectedBackupWindow)
	if !strings.Contains(croStrategyConfig.Data["postgres"], postgresExpectedBackupWindow) {
		foundErrors = append(foundErrors, fmt.Sprintf("\n - expected '%s' not found in Postgres strategy config map", postgresExpectedBackupWindow))
	}
	postgresExpectedMaintenanceWindow := fmt.Sprintf("\"PreferredMaintenanceWindow\":\"%s\"", expectedMaintenanceWindow)
	if !strings.Contains(croStrategyConfig.Data["postgres"], postgresExpectedMaintenanceWindow) {
		foundErrors = append(foundErrors, fmt.Sprintf("\n - expected '%s' not found in Postgres strategy config map", postgresExpectedMaintenanceWindow))
	}

	if len(foundErrors) != 0 {
		return fmt.Errorf("\n - Redis data found : \n\t%s\n - Postgres data found : \n\t%s\n %s", croStrategyConfig.Data["redis"], croStrategyConfig.Data["postgres"], foundErrors)
	}

	return nil
}

// updates the rhmiconfig backup and maintenance values
func UpdateRHMIConfigBackupAndMaintenance(context context.Context, client client.Client, rhmiConfig *v1alpha1.RHMIConfig, maintenanceBackup MaintenanceBackup) error {
	if _, err := controllerutil.CreateOrUpdate(context, client, rhmiConfig, func() error {
		rhmiConfig.Spec.Maintenance.ApplyFrom = maintenanceBackup.Maintenance.ApplyFrom
		rhmiConfig.Spec.Backup.ApplyOn = maintenanceBackup.Backup.ApplyOn
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// expected redis value to be returned in cro strat map
func BuildExpectRedisStrat(backupWindow, maintenanceWindow string) string {
	return fmt.Sprintf("{\"development\":{\"region\":\"\",\"createStrategy\":{},\"deleteStrategy\":{}},\"production\":{\"region\":\"\",\"createStrategy\":{\"AtRestEncryptionEnabled\":null,\"AuthToken\":null,\"AutoMinorVersionUpgrade\":null,\"AutomaticFailoverEnabled\":null,\"CacheNodeType\":null,\"CacheParameterGroupName\":null,\"CacheSecurityGroupNames\":null,\"CacheSubnetGroupName\":null,\"Engine\":null,\"EngineVersion\":null,\"KmsKeyId\":null,\"NodeGroupConfiguration\":null,\"NotificationTopicArn\":null,\"NumCacheClusters\":null,\"NumNodeGroups\":null,\"Port\":null,\"PreferredCacheClusterAZs\":null,\"PreferredMaintenanceWindow\":\"%s\",\"PrimaryClusterId\":null,\"ReplicasPerNodeGroup\":null,\"ReplicationGroupDescription\":null,\"ReplicationGroupId\":null,\"SecurityGroupIds\":null,\"SnapshotArns\":null,\"SnapshotName\":null,\"SnapshotRetentionLimit\":null,\"SnapshotWindow\":\"%s\",\"Tags\":null,\"TransitEncryptionEnabled\":null},\"deleteStrategy\":{}}}", maintenanceWindow, backupWindow)
}

// expected postgres value to be returned in cro strat map
func buildExpectPostgresStrat(backupWindow, maintenanceWindow string) string {
	return fmt.Sprintf("{\"development\":{\"region\":\"\",\"createStrategy\":{},\"deleteStrategy\":{}},\"production\":{\"region\":\"\",\"createStrategy\":{\"AllocatedStorage\":null,\"AutoMinorVersionUpgrade\":null,\"AvailabilityZone\":null,\"BackupRetentionPeriod\":null,\"CharacterSetName\":null,\"CopyTagsToSnapshot\":null,\"DBClusterIdentifier\":null,\"DBInstanceClass\":null,\"DBInstanceIdentifier\":null,\"DBName\":null,\"DBParameterGroupName\":null,\"DBSecurityGroups\":null,\"DBSubnetGroupName\":null,\"DeletionProtection\":null,\"Domain\":null,\"DomainIAMRoleName\":null,\"EnableCloudwatchLogsExports\":null,\"EnableIAMDatabaseAuthentication\":null,\"EnablePerformanceInsights\":null,\"Engine\":null,\"EngineVersion\":null,\"Iops\":null,\"KmsKeyId\":null,\"LicenseModel\":null,\"MasterUserPassword\":null,\"MasterUsername\":null,\"MaxAllocatedStorage\":null,\"MonitoringInterval\":null,\"MonitoringRoleArn\":null,\"MultiAZ\":null,\"OptionGroupName\":null,\"PerformanceInsightsKMSKeyId\":null,\"PerformanceInsightsRetentionPeriod\":null,\"Port\":null,\"PreferredBackupWindow\":\"%s\",\"PreferredMaintenanceWindow\":\"%s\",\"ProcessorFeatures\":null,\"PromotionTier\":null,\"PubliclyAccessible\":null,\"StorageEncrypted\":null,\"StorageType\":null,\"Tags\":null,\"TdeCredentialArn\":null,\"TdeCredentialPassword\":null,\"Timezone\":null,\"VpcSecurityGroupIds\":null},\"deleteStrategy\":{}}}", backupWindow, maintenanceWindow)
}
