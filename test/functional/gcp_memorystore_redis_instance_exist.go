package functional

import (
	redis "cloud.google.com/go/redis/apiv1"
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/test/common"
	"google.golang.org/api/iterator"
	//redispb "cloud.google.com/go/redis/apiv1/redispb"
	redispb "google.golang.org/genproto/googleapis/cloud/redis/v1"
)

const (
	redisGetInstanceRequestFormat = "projects/%s/locations/%s/instances/%s"
	//`projects/{project_id}/locations/{location_id}/instances/{instance_id}`
)

func TestGCPMemorystoreRedisInstanceExist(t common.TestingTB, testingContext *common.TestingContext) {

	ctx := context.Background()
	c, err := redis.NewCloudRedisClient(ctx)
	if err != nil {
		t.Fatal("error create new cloud redis client %w", err)
	}
	defer c.Close()

	projectID, err := getDefaultProjectID(ctx, testingContext.Client)
	if err != nil {
		t.Fatal("error get Default Project ID %w", err)
	}
	region, err := getDefaultRegion(ctx, testingContext.Client)
	if err != nil {
		t.Fatal("error get Default Region %w", err)
	}

	req := &redispb.ListInstancesRequest{
		// TODO: Fill request struct fields.
		// See https://pkg.go.dev/cloud.google.com/go/redis/apiv1/redispb#ListInstancesRequest.
	}
	var redisInstanceList []string
	it := c.ListInstances(ctx, req)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatal("error get Redis instance %w", err)
		}
		_ = resp
		fmt.Printf("%v\n", resp.Name)
		redisInstanceList = append(redisInstanceList, resp.Name)
	}
	goContext := context.TODO()
	rhmi, err := common.GetRHMI(testingContext.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	// build an array of redis resources to check and test error array
	redisInstanceIDs, testErrors := GetRedisInstancesIDs(goContext, testingContext.Client, rhmi)
	if len(testErrors) != 0 {
		t.Fatalf("test cro redis exists failed with the following errors : %s", testErrors)
	}
	for _, redisId := range redisInstanceIDs {
		if !verifyRedisInstances(redisId, redisInstanceIDs) {
			t.Fatal("Redis Instance %s defined in CR, but missing in Google Cloud", redisId)
		}
		req := &redispb.GetInstanceRequest{
			// See https://pkg.go.dev/cloud.google.com/go/redis/apiv1/redispb#GetInstanceRequest.
			Name: fmt.Sprintf(redisGetInstanceRequestFormat, projectID, region, redisId),
		}
		resp, err := c.GetInstance(ctx, req)
		if err != nil {
			testErrors = append(testErrors, fmt.Sprintf("error get Redis Instance %s, error: %v", redisId, err))
			continue
		}
		if resp.State != redispb.Instance_READY {
			testErrors = append(testErrors, fmt.Sprintf("Redis Instance %s is not Ready. State: %s", redisId, resp.State.String()))
		}
		//if resp.ReplicaCount > 1 {
		//	//check that replicas in different zones
		//	//resp.get TODO
		//}

	}
	if len(testErrors) != 0 {
		t.Fatalf("test Redis instances exists failed with the following errors : %s", testErrors)
	}

}

func verifyRedisInstances(redisInstanceName string, databaseInstanceList []string) bool {
	for _, v := range databaseInstanceList {
		if v == redisInstanceName {
			return true
		}
	}
	return false
}
