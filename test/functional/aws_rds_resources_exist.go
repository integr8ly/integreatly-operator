package functional

import (
	goctx "context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	croTypes "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/test/common"
	"github.com/integr8ly/integreatly-operator/test/resources"
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

var (
	expectedPostgres = []string{
		fmt.Sprintf("%s%s", constants.CodeReadyPostgresPrefix, common.InstallationName),
		fmt.Sprintf("%s%s", constants.ThreeScalePostgresPrefix, common.InstallationName),
		fmt.Sprintf("%s%s", constants.RHSSOPostgresPrefix, common.InstallationName),
		fmt.Sprintf("%s%s", constants.RHSSOUserProstgresPrefix, common.InstallationName),
		fmt.Sprintf("%s%s", constants.UPSPostgresPrefix, common.InstallationName),
		// TODO - Add check for Fuse postgres here when task for supporting external resources is done - https://issues.redhat.com/browse/INTLY-3239
	}
)

func AWSRDSResourcesExistTest(t *testing.T, ctx *common.TestingContext) {
	goContext := goctx.TODO()
	var testErrors []string

	// build an array of postgres resources to check
	var rdsResourceIDs []string
	for _, p := range expectedPostgres {
		// get postgres cr
		postgres := &crov1.Postgres{}
		if err := ctx.Client.Get(goContext, types.NamespacedName{Name: p, Namespace: common.RHMIOperatorNamespace}, postgres); err != nil {
			testErrors = append(testErrors, fmt.Sprintf("\nfailed to find %s postgres cr : %v", p, err))
		}
		// ensure cr phase is completed
		if postgres.Status.Phase != croTypes.PhaseComplete {
			testErrors = append(testErrors, fmt.Sprintf("\nfound %s postgres not ready with phase: %s, message: %s", p, postgres.Status.Phase, postgres.Status.Message))
		}
		// return the resource id
		resourceID, err := GetCROAnnotation(postgres)
		if err != nil {
			testErrors = append(testErrors, fmt.Sprintf("\n%s postgres does not contain a resource id annotation: %v", p, err))
		}
		// populate the array
		rdsResourceIDs = append(rdsResourceIDs, resourceID)
	}

	if len(testErrors) != 0 {
		t.Fatalf("test cro postgres exists failed with the following errors : %s", testErrors)
	}
	sess, err := resources.CreateAWSSession(goContext, ctx.Client)
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
