package functional

import (
	goctx "context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/integr8ly/integreatly-operator/test/common"
)

func AWSRDSResourcesExistTest(t common.TestingTB, ctx *common.TestingContext) {
	goContext := goctx.TODO()

	rhmi, err := common.GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	// build an array of postgres resources to check and an array of test errors
	rdsData, testErrors := GetPostgresInstanceData(goContext, ctx.Client, rhmi)

	if len(testErrors) != 0 {
		t.Fatalf("test cro postgres exists failed with the following errors : %s", testErrors)
	}
	sess, isSTS, err := CreateAWSSession(goContext, ctx.Client)
	if err != nil {
		t.Fatalf("failed to create aws session: %v", err)
	}

	// check ever expected resource
	rdsapi := rds.New(sess)
	for resourceIdentifier, rdsVersion := range rdsData {
		// get the rds instance
		foundRDSInstances, err := rdsapi.DescribeDBInstances(&rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(resourceIdentifier),
		})
		if err != nil {
			testErrors = append(testErrors, fmt.Errorf("failed to get rds instance :%s with error : %v", resourceIdentifier, err))
			continue
		}
		// verify the rds instance is as expected
		if !verifyRDSInstanceConfig(*foundRDSInstances.DBInstances[0], isSTS, rdsVersion) {
			testErrors = append(testErrors, fmt.Errorf("failed as rds %s resource is not as expected", resourceIdentifier))
		}
	}

	if len(testErrors) != 0 {
		t.Fatalf("test cro postgres exists failed with the following errors : %s", testErrors)
	}

}

// return expected resource variables
func verifyRDSInstanceConfig(instance rds.DBInstance, isSTS bool, rdsVersion string) bool {
	// if managed tag is present, and we are either not running on STS, or we are running on STS
	// and the rosa cluster type is present, and the rest of the config is expected
	return rdsTagsContains(instance.TagList, awsManagedTagKey, awsManagedTagValue) &&
		(!isSTS || rdsTagsContains(instance.TagList, awsClusterTypeKey, awsClusterTypeRosaValue)) &&
		*instance.MultiAZ && *instance.DeletionProtection && *instance.StorageEncrypted &&
		!*instance.AutoMinorVersionUpgrade && *instance.EngineVersion == rdsVersion
}
