package functional

import (
	"context"
	"fmt"

	redis "cloud.google.com/go/redis/apiv1"
	croResources "github.com/integr8ly/cloud-resource-operator/pkg/resources"

	"github.com/integr8ly/integreatly-operator/test/common"
	"google.golang.org/api/option"

	"cloud.google.com/go/redis/apiv1/redispb"
)

const (
	redisGetInstanceRequestFormat = "projects/%s/locations/%s/instances/%s"
	//`projects/{project_id}/locations/{location_id}/instances/{instance_id}`
)

func TestGCPMemorystoreRedisInstanceExist(t common.TestingTB, testingContext *common.TestingContext) {
	ctx := context.Background()
	serviceAccountJson, err := getGCPCredentials(ctx, testingContext.Client)
	if err != nil {
		t.Fatal("failed to retrieve gcp credentials %v", err)
	}
	c, err := redis.NewCloudRedisClient(ctx, option.WithCredentialsJSON(serviceAccountJson))
	if err != nil {
		t.Fatal("error create new cloud redis client %w", err)
	}
	defer c.Close()

	projectID, err := croResources.GetGCPProject(ctx, testingContext.Client)
	if err != nil {
		t.Fatal("error get Default Project ID %w", err)
	}
	region, err := croResources.GetGCPRegion(ctx, testingContext.Client)
	if err != nil {
		t.Fatal("error get Default Region %w", err)
	}

	goContext := context.TODO()
	rhmi, err := common.GetRHMI(testingContext.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	// build an array of redis resources to check and test error array
	redisInstanceData, testErrors := GetRedisInstanceData(goContext, testingContext.Client, rhmi)
	if len(testErrors) != 0 {
		t.Fatalf("test cro redis exists failed with the following errors : %s", testErrors)
	}
	for redisId, redisVersion := range redisInstanceData {
		if !verifyRedisInstances(redisId, redisInstanceData) {
			t.Fatal("Redis Instance %s defined in CR, but missing in Google Cloud", redisId)
		}
		req := &redispb.GetInstanceRequest{
			// See https://pkg.go.dev/cloud.google.com/go/redis/apiv1/redispb#GetInstanceRequest.
			Name: fmt.Sprintf(redisGetInstanceRequestFormat, projectID, region, redisId),
		}
		resp, err := c.GetInstance(ctx, req)
		if err != nil {
			testErrors = append(testErrors, fmt.Errorf("error getting Redis Instance %s, error: %v", redisId, err))
			continue
		}
		if !verifyRedisInstanceConfig(resp, redisVersion) {
			t.Fatal("failed as resource is not as expected")
		}
	}
	if len(testErrors) != 0 {
		t.Fatalf("test Redis instances exists failed with the following errors : %s", testErrors)
	}

}

func verifyRedisInstances(redisInstanceName string, databaseInstanceList map[string]string) bool {
	for k, _ := range databaseInstanceList {
		if k == redisInstanceName {
			return true
		}
	}
	return false
}

func verifyRedisInstanceConfig(instance *redispb.Instance, redisVersion string) bool {
	return labelsContain(instance.Labels, managedLabelKey, managedLabelValue) &&
		instance.RedisVersion == redisVersion && instance.State == redispb.Instance_READY &&
		instance.Tier == redispb.Instance_STANDARD_HA && instance.MaintenancePolicy != nil

}
