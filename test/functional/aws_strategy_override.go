package functional

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/integr8ly/integreatly-operator/test/common"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
	sess, _, err := CreateAWSSession(ctx, client)
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
	sess, _, err := CreateAWSSession(ctx, client)
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
