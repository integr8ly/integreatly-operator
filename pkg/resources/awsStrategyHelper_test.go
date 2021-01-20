package resources

import (
	"context"
	"log"
	"testing"

	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getAWSStrategyBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := corev1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	return scheme, err
}

func getPostgresOverrides() map[string]interface{} {
	return map[string]interface{}{
		"DBInstanceClass":         "db.t3.large",
		"AutoMinorVersionUpgrade": true,
		"AllocatedStorage":        10,
	}
}

func getRedisOverrides() map[string]interface{} {
	return map[string]interface{}{
		"CacheNodeType":           "large",
		"AutoMinorVersionUpgrade": true,
		"NumCacheNodes":           10,
	}
}

func TestCreatePostgresTier_fromProduction(t *testing.T) {

	scheme, err := getAWSStrategyBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	mockConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cloud-resources-aws-strategies",
			Namespace: "mockNamespace",
		},
		Data: map[string]string{
			"postgres": "{\"production\":{\"region\":\"\",\"createStrategy\":{\"Timezone\":\"Timezone\",\"AutoMinorVersionUpgrade\":false,\"AllocatedStorage\":5,\"DBInstanceClass\":null},\"deleteStrategy\":{}}}",
			"_network": "{\"production\":{\"createStrategy\":{\"CidrBlock\":\"\"}}}",
		},
	}
	expectedPostgres := "{\"newtier\":{\"region\":\"\",\"createStrategy\":{\"AllocatedStorage\":10,\"AutoMinorVersionUpgrade\":true,\"AvailabilityZone\":null,\"BackupRetentionPeriod\":null,\"CharacterSetName\":null,\"CopyTagsToSnapshot\":null,\"DBClusterIdentifier\":null,\"DBInstanceClass\":\"db.t3.large\",\"DBInstanceIdentifier\":null,\"DBName\":null,\"DBParameterGroupName\":null,\"DBSecurityGroups\":null,\"DBSubnetGroupName\":null,\"DeletionProtection\":null,\"Domain\":null,\"DomainIAMRoleName\":null,\"EnableCloudwatchLogsExports\":null,\"EnableIAMDatabaseAuthentication\":null,\"EnablePerformanceInsights\":null,\"Engine\":null,\"EngineVersion\":null,\"Iops\":null,\"KmsKeyId\":null,\"LicenseModel\":null,\"MasterUserPassword\":null,\"MasterUsername\":null,\"MaxAllocatedStorage\":null,\"MonitoringInterval\":null,\"MonitoringRoleArn\":null,\"MultiAZ\":null,\"NcharCharacterSetName\":null,\"OptionGroupName\":null,\"PerformanceInsightsKMSKeyId\":null,\"PerformanceInsightsRetentionPeriod\":null,\"Port\":null,\"PreferredBackupWindow\":null,\"PreferredMaintenanceWindow\":null,\"ProcessorFeatures\":null,\"PromotionTier\":null,\"PubliclyAccessible\":null,\"StorageEncrypted\":null,\"StorageType\":null,\"Tags\":null,\"TdeCredentialArn\":null,\"TdeCredentialPassword\":null,\"Timezone\":\"Timezone\",\"VpcSecurityGroupIds\":null},\"deleteStrategy\":{}},\"production\":{\"region\":\"\",\"createStrategy\":{\"Timezone\":\"Timezone\",\"AutoMinorVersionUpgrade\":false,\"AllocatedStorage\":5,\"DBInstanceClass\":null},\"deleteStrategy\":{}}}"
	expectedNetwork := "{\"newtier\":{\"createStrategy\":{\"CidrBlock\":\"\"}},\"production\":{\"createStrategy\":{\"CidrBlock\":\"\"}}}"

	fakeClient := moqclient.NewSigsClientMoqWithScheme(scheme, []runtime.Object{mockConfigMap}...)
	fakeClient.PatchFunc = func(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {

		cm := obj.(*corev1.ConfigMap)
		if cm.Data["postgres"] != expectedPostgres {
			log.Println(cm.Data["postgres"])
			log.Println(expectedPostgres)
			t.Fatal("CM postgres data not updated as expected")
		}
		if cm.Data["_network"] != expectedNetwork {
			//log.Println(cm.Data["_network"])
			t.Fatal("CM network data not updated as expected")
		}
		return nil
	}

	CreatePostgresTierFromProduction(context.TODO(), fakeClient, "mockNamespace", false, "newtier", getPostgresOverrides(), getLogger())
}

func TestCreateRedisTier_fromProduction(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	mockConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cloud-resources-aws-strategies",
			Namespace: "mockNamespace",
		},
		Data: map[string]string{
			"redis": "{\"production\":{\"region\":\"\",\"createStrategy\":{\"EngineVersion\":\"1.1.1\",\"CacheNodeType\":\"micro\",\"NumCacheNodes\":5},\"deleteStrategy\":{}}}",
		},
	}
	expectedRedis := "{\"newtier\":{\"region\":\"\",\"createStrategy\":{\"AZMode\":null,\"AuthToken\":null,\"AutoMinorVersionUpgrade\":true,\"CacheClusterId\":null,\"CacheNodeType\":\"large\",\"CacheParameterGroupName\":null,\"CacheSecurityGroupNames\":null,\"CacheSubnetGroupName\":null,\"Engine\":null,\"EngineVersion\":\"1.1.1\",\"NotificationTopicArn\":null,\"NumCacheNodes\":10,\"Port\":null,\"PreferredAvailabilityZone\":null,\"PreferredAvailabilityZones\":null,\"PreferredMaintenanceWindow\":null,\"ReplicationGroupId\":null,\"SecurityGroupIds\":null,\"SnapshotArns\":null,\"SnapshotName\":null,\"SnapshotRetentionLimit\":null,\"SnapshotWindow\":null,\"Tags\":null},\"deleteStrategy\":{}},\"production\":{\"region\":\"\",\"createStrategy\":{\"EngineVersion\":\"1.1.1\",\"CacheNodeType\":\"micro\",\"NumCacheNodes\":5},\"deleteStrategy\":{}}}"

	fakeClient := moqclient.NewSigsClientMoqWithScheme(scheme, []runtime.Object{mockConfigMap}...)
	fakeClient.PatchFunc = func(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {

		cm := obj.(*corev1.ConfigMap)
		if cm.Data["redis"] != expectedRedis {
			t.Fatal("CM reids data not updated as expected")
		}
		return nil
	}

	CreateRedisTierFromProduction(context.TODO(), fakeClient, "mockNamespace", false, "newtier", getRedisOverrides(), getLogger())
}
