package functional

import (
	goctx "context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/integr8ly/integreatly-operator/test/common"
)

func AWSRDSResourcesExistTest(t *testing.T, ctx *common.TestingContext) {
	goContext := goctx.TODO()

	// build an array of postgres resources to check and an array of test errors
	rdsResourceIDs, testErrors := GetRDSResourceIDs(goContext, ctx.Client)

	if len(testErrors) != 0 {
		t.Fatalf("test cro postgres exists failed with the following errors : %s", testErrors)
	}
	sess, err := CreateAWSSession(goContext, ctx.Client)
	if err != nil {
		t.Fatalf("failed to create aws session: %v", err)
	}

	// check ever expected resource
	rdsapi := rds.New(sess)
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
		if !verifyRDSInstanceConfig(*foundRDSInstances.DBInstances[0]) {
			testErrors = append(testErrors, fmt.Sprintf("failed as rds %s resource is not as expected", resourceIdentifier))
		}
	}

	if len(testErrors) != 0 {
		t.Fatalf("test cro postgres exists failed with the following errors : %s", testErrors)
	}

}

// return expected resource variables
func verifyRDSInstanceConfig(instance rds.DBInstance) bool {
	return *instance.MultiAZ && *instance.DeletionProtection && *instance.StorageEncrypted
}
