package functional

import (
	goctx "context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/integr8ly/integreatly-operator/test/common"
	"testing"
)

func AWSElasticacheResourcesExistTest(t *testing.T, ctx *common.TestingContext) {
	goContext := goctx.TODO()

	// build an array of redis resources to check and test error array
	elasticacheResourceIDs, testErrors := GetElasticacheResourceIDs(goContext, ctx.Client)

	if len(testErrors) != 0 {
		t.Fatalf("test cro redis exists failed with the following errors : %s", testErrors)
	}

	// create AWS session
	sess, err := CreateAWSSession(goContext, ctx.Client)
	if err != nil {
		t.Fatalf("failed to create aws session: %v", err)
	}

	// create new elasticache api with retrieved session
	elasticacheapi := elasticache.New(sess)

	// iterate through returned resource ID's
	for _, resourceID := range elasticacheResourceIDs {
		//get elasticache resources through new elasticacheapi
		foundElasticacheReplicationGroups, err := elasticacheapi.DescribeReplicationGroups(&elasticache.DescribeReplicationGroupsInput{
			ReplicationGroupId: aws.String(resourceID),
		})
		if err != nil {
			testErrors = append(testErrors, fmt.Sprintf("failed to get %s elasticache replicationgroups with error : %v", resourceID, err))
			continue
		}
		if len(foundElasticacheReplicationGroups.ReplicationGroups[0].NodeGroups) > 1 {
			testErrors = append(testErrors, fmt.Sprintf("insufficient number of nodes in elasticache group"))
			continue
		}
		replicationGroup := foundElasticacheReplicationGroups.ReplicationGroups[0]
		nodeGroup := replicationGroup.NodeGroups[0]

		// perform checks to verify state is as expected
		if !verifyMultiAZ(nodeGroup.NodeGroupMembers) {
			testErrors = append(testErrors, fmt.Sprintf("elasticache resource %s multiAZ failure %v", resourceID, err))
		}
		if !aws.BoolValue(replicationGroup.AtRestEncryptionEnabled) {
			testErrors = append(testErrors, fmt.Sprintf("elasticache resource %s does not have encryption enabled", resourceID))
		}
		if replicationGroup.SnapshotWindow == nil {
			testErrors = append(testErrors, fmt.Sprintf("elasticache resource %s does not have automatic snapshotting enabled", resourceID))
		}
	}

	if len(testErrors) != 0 {
		t.Fatalf("test elasticache instances exists failed with the following errors : %s", testErrors)
	}

}

// helper method for verifying nodes are in different availability zones
func verifyMultiAZ(member []*elasticache.NodeGroupMember) bool {
	if member[0].PreferredAvailabilityZone == member[1].PreferredAvailabilityZone {
		return false
	}
	return true
}
