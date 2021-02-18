package functional

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/common"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

/*
	This test is to verify that changes made to the RHMI config are reflected in the
	Cloud Resource Operator (CRO) strategy override config map
	And further reflected in the AWS resource
	RHMI config allows for Maintenance and Backup times
	We need to ensure these windows are built and updated in the config map

	We currently have tests A18 and A21 where we test the validation of the RHMI config
	And the updating of the CRO strategy override config map
	We will be using this test to full test e2e from updating the config to changing the values in AWS
*/

type strategytestCase struct {
	RHMIConfigValues  common.MaintenanceBackup
	backupWindow      string
	maintenanceWindow string
}

var strategyTestCases = []strategytestCase{
	{
		RHMIConfigValues: common.MaintenanceBackup{
			Backup: v1alpha1.Backup{
				ApplyOn: "20:00",
			},
			Maintenance: v1alpha1.Maintenance{
				ApplyFrom: "sun 21:01",
			},
		},
		backupWindow:      "20:00-21:00",
		maintenanceWindow: "sun:21:01-sun:22:01",
	},
}

// tests e2e cro strategy override
func CROStrategyOverrideAWSResourceTest(t common.TestingTB, testingContext *common.TestingContext) {
	ctx := context.TODO()
	var testErrors []string

	// rhmi config we need to use is the rhmi config provisioned in the RHMI install
	// this to avoid a conflict of having multiple rhmi configs
	rhmiConfig := common.RHMIConfigTemplate()

	for _, test := range strategyTestCases {
		// update rhmi config Backup and Maintenance times to valid expected times
		if err := common.UpdateRHMIConfigBackupAndMaintenance(ctx, testingContext.Client, rhmiConfig, test.RHMIConfigValues); err != nil {
			testErrors = append(testErrors, fmt.Sprintf("\nunable to update rhmi config : %v", err))
			continue
		}

		// ensure strategy config map is as expected
		// we have added this poll to allow the operator reconcile on the cr and update the strategy override
		// we expect the change to be immediate, the poll is help with any potential test flake
		// continue to poll until no error or timeout - on timeout we handle the polling error
		var lastPollError error
		if err := wait.PollImmediate(time.Second*5, time.Second*30, func() (done bool, err error) {
			lastPollError = common.VerifyCROStrategyMap(ctx, testingContext.Client, test.backupWindow, test.maintenanceWindow)
			if lastPollError == nil {
				return true, nil
			}
			return false, nil
		}); err != nil {
			testErrors = append(testErrors, fmt.Sprintf("\ntest failure : %s : %v", lastPollError, err))
		}

		// we need to verify if the rds instance have been updated to reflect the new backup and maintenance windows
		// as we need to wait for the Cloud Resource Operator to reconcile on each RDS resource it may take some time for the windows to be updated
		// continue to poll until no error or timeout - on timeout we handle the polling error
		if err := wait.PollImmediate(time.Second*30, time.Second*300, func() (done bool, err error) {
			lastPollError = verifyRDSMaintenanceBackupWindows(ctx, testingContext.Client, test.backupWindow, test.maintenanceWindow)
			if lastPollError == nil {
				return true, nil
			}
			return false, nil

		}); err != nil {
			testErrors = append(testErrors, fmt.Sprintf("\ntest failure : %s : %v", lastPollError, err))
		}

		// we need to verify if the elasticache instance have been updated to reflect the new backup and maintenance windows
		// as we need to wait for the Cloud Resource Operator to reconcile on each RDS resource it may take some time for the windows to be updated
		// continue to poll until no error or timeout - on timeout we handle the polling error
		if err := wait.PollImmediate(time.Second*30, time.Second*300, func() (done bool, err error) {
			lastPollError = verifyElasticacheMaintenanceBackupWindows(ctx, testingContext.Client, test.backupWindow, test.maintenanceWindow)
			if lastPollError == nil {
				return true, nil
			}
			return false, nil
		}); err != nil {
			testErrors = append(testErrors, fmt.Sprintf("\ntest failure : %s : %v", lastPollError, err))
		}
	}

	if len(testErrors) != 0 {
		t.Fatalf("strategy override test failed : \n%s", testErrors)
	}
}

/*
	From a list of ResourceID's we iterate through them
	For every resource we describe the resource, checking the backup and maintenance windows are as expected
	We build a list of errors, to ensure we catch every verification
	If there is an error we return the list as an error
*/
func verifyRDSMaintenanceBackupWindows(ctx context.Context, client client.Client, expectedBackupWindow, expectedMaintenanceWindow string) error {

	rhmi, err := common.GetRHMI(client, true)
	if err != nil {
		return err
	}

	// build an array of postgres resources to check and an array of test errors
	rdsResourceIDs, testErrors := GetRDSResourceIDs(ctx, client, rhmi)

	// check for errors from getting rds resource ids
	// if there is any errors we return as we can not continue the test with out all expected resources
	if len(testErrors) != 0 {
		return fmt.Errorf("verify rds maintenance and backup windows failed with the following errors : \n%s", testErrors)
	}

	// create aws session
	sess, err := CreateAWSSession(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to create aws session: %v", err)
	}
	rdsapi := rds.New(sess)

	// check every rds instance backup and maintenance windows are as expected resource
	for _, resourceIdentifier := range rdsResourceIDs {
		// get the rds instance
		foundRDSInstances, err := rdsapi.DescribeDBInstances(&rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(resourceIdentifier),
		})
		if err != nil {
			testErrors = append(testErrors, fmt.Sprintf("failed to get rds instance :%s with error : %v", resourceIdentifier, err))
			continue
		}
		// verify the rds instance is as expected
		instance := *foundRDSInstances.DBInstances[0]
		if *instance.PreferredMaintenanceWindow != expectedMaintenanceWindow {
			testErrors = append(testErrors, fmt.Sprintf("\tresource found : %s : found maintenance window : %s : expected maintenance window : %s\n", *instance.DBInstanceIdentifier, *instance.PreferredMaintenanceWindow, expectedMaintenanceWindow))
		}
		if *instance.PreferredBackupWindow != expectedBackupWindow {
			testErrors = append(testErrors, fmt.Sprintf("\tresource found : %s : found backup window : %s : expected backup window : %s\n", *instance.DBInstanceIdentifier, *instance.PreferredBackupWindow, expectedBackupWindow))

		}
	}

	// check for any errors
	if len(testErrors) != 0 {
		return fmt.Errorf("verify rds maintenance and backup windows failed with the following errors : \n%s", testErrors)
	}
	return nil
}

/*
	From a list of ResourceID's we iterate through them
	For every resource we describe the resource, checking the backup and maintenance windows are as expected
	We build a list of errors, to ensure we catch every verification
	If there is an error we return the list as an error
*/
func verifyElasticacheMaintenanceBackupWindows(ctx context.Context, client client.Client, expectedBackupWindow, expectedMaintenanceWindow string) error {
	rhmi, err := common.GetRHMI(client, true)
	if err != nil {
		return err
	}

	// build an array of redis resources to check and test error array
	elasticacheResourceIDs, testErrors := GetElasticacheResourceIDs(ctx, client, rhmi)

	// check for errors from getting rds resource ids
	// if there is any errors we return as we can not continue the test with out all expected resources
	if len(testErrors) != 0 {
		return fmt.Errorf("test elasticache maintenance and backup windows failed with the following errors : \n%s", testErrors)
	}

	// create AWS session
	sess, err := CreateAWSSession(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to create aws session: %v", err)
	}
	elasticacheapi := elasticache.New(sess)

	// check each elasticache maintenance and backup windows are as expected
	// we need to used elasticache clusters as it is the only way we can return both maintenance and snapshot windows
	// see * https://docs.aws.amazon.com/sdk-for-go/api/service/elasticache/#DescribeCacheClustersOutput
	// The cloud resource operator (CRO) provisions a replication group
	// A replication group is made up of two nodes, each node is a cache cluster
	// Describing cache clusters we get more fine grained information about each node
	foundElasticacheClusters, err := elasticacheapi.DescribeCacheClusters(&elasticache.DescribeCacheClustersInput{})
	if err != nil {
		testErrors = append(testErrors, fmt.Sprintf("failed to get elasticache cache clusters with error : %v", err))
	}

	// we need to iterate through each resource ID
	for _, resourceID := range elasticacheResourceIDs {
		// we need to iterate through each cache cluster
		for _, cacheCluster := range foundElasticacheClusters.CacheClusters {
			// if a cache cluster does not match our resource id we can break out
			if *cacheCluster.ReplicationGroupId != resourceID {
				continue
			}
			// check cache cluster maintenance and backup windows are as expected
			if *cacheCluster.PreferredMaintenanceWindow != expectedMaintenanceWindow {
				testErrors = append(testErrors, fmt.Sprintf("\tresource found : %s : found maintenance window : %s : expected maintenance window : %s\n", *cacheCluster.ReplicationGroupId, *cacheCluster.PreferredMaintenanceWindow, expectedMaintenanceWindow))
			}
			if *cacheCluster.SnapshotWindow != expectedBackupWindow {
				testErrors = append(testErrors, fmt.Sprintf("\tresource found : %s : found snapshot window : %s : expected snapshot window : %s\n", *cacheCluster.ReplicationGroupId, *cacheCluster.SnapshotWindow, expectedBackupWindow))
			}
		}
	}

	// check for any errors
	if len(testErrors) != 0 {
		return fmt.Errorf("verify elasticache maintenance and snapshot windows failed with the following errors : \n%s", testErrors)
	}
	return nil
}
