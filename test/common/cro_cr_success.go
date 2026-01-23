package common

import (
	"context"
	"encoding/json"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"

	crov1 "github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1"
	croTypes "github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1/types"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
)

const (
	requestUrl        = "/apis/integreatly.org/v1alpha1"
	openShiftProvider = "openshift"
	externalProvider  = "cloud"
)

func getPostgres(installType string, installationName string) []string {
	// Common to all install types including managed api
	commonPostgresToCheck := []string{
		fmt.Sprintf("%s%s", constants.ThreeScalePostgresPrefix, installationName),
		fmt.Sprintf("%s%s", constants.RHSSOPostgresPrefix, installationName),
	}

	rhoamPostgresToCheck := []string{
		fmt.Sprintf("%s%s", constants.RHSSOUserProstgresPrefix, installationName),
	}

	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installType)) {
		return commonPostgresToCheck
	} else {
		return append(commonPostgresToCheck, rhoamPostgresToCheck...)
	}
}

func getRedisToCheck(installType string, installationName string) []string {
	commonRedis := []string{
		fmt.Sprintf("%s%s", constants.ThreeScaleBackendRedisPrefix, installationName),
		fmt.Sprintf("%s%s", constants.ThreeScaleSystemRedisPrefix, installationName),
		fmt.Sprintf("%s%s", constants.RateLimitRedisPrefix, installationName),
	}

	return commonRedis
}

func getBlobStorageToCheck(installType, installationName string) []string {
	common := []string{
		fmt.Sprintf("%s%s", constants.ThreeScaleBlobStoragePrefix, installationName),
	}

	return common
}

func TestCROPostgresSuccessfulState(t TestingTB, ctx *TestingContext) {
	originalStrategy := getResourceStrategy(t, ctx)

	// get console master url
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	postgresToCheck := getPostgres(rhmi.Spec.Type, rhmi.Name)

	for _, postgresName := range postgresToCheck {
		// AMQAuthService postgres is always in cluster
		strategy := originalStrategy

		postgres := &crov1.Postgres{}
		err := getResourceAndUnMarshalJsonToResource(ctx, "postgres", postgresName, postgres)

		if err != nil {
			t.Errorf("Failed to retrieve postgres custom resource: %s", err)
			continue
		}

		if postgres.Status.Phase != croTypes.PhaseComplete && postgres.Status.Strategy != strategy {
			t.Errorf("%s Postgres not ready with phase: %s, message: %s, provider, %s", postgresName, postgres.Status.Phase, postgres.Status.Message, postgres.Status.Provider)
		}
	}
}

func TestCRORedisSuccessfulState(t TestingTB, ctx *TestingContext) {
	strategy := getResourceStrategy(t, ctx)

	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	redisToCheck := getRedisToCheck(rhmi.Spec.Type, rhmi.Name)

	for _, redisName := range redisToCheck {
		redis := &crov1.Redis{}
		err := getResourceAndUnMarshalJsonToResource(ctx, "redis", redisName, redis)

		if err != nil {
			t.Errorf("Failed to retrieve redis custom resource: %s", err)
			continue
		}

		if redis.Status.Phase != croTypes.PhaseComplete && redis.Status.Strategy != strategy {
			t.Errorf("%s redis not ready with phase: %s, message: %s, provider, %s", redisName, redis.Status.Phase, redis.Status.Message, redis.Status.Provider)
		}
	}
}

func TestCROBlobStorageSuccessfulState(t TestingTB, ctx *TestingContext) {
	strategy := getResourceStrategy(t, ctx)

	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	blobStorageToCheck := getBlobStorageToCheck(rhmi.Spec.Type, rhmi.Name)

	for _, blobStorageName := range blobStorageToCheck {
		blobStorage := &crov1.BlobStorage{}
		err := getResourceAndUnMarshalJsonToResource(ctx, "blobstorages", blobStorageName, blobStorage)

		if err != nil {
			t.Errorf("Failed to retrieve blobstorage custom resource: %s", err)
			continue
		}

		if blobStorage.Status.Phase != croTypes.PhaseComplete && blobStorage.Status.Strategy != strategy {
			t.Errorf("%s blob storage not ready with phase: %s, message: %s, provider, %s", blobStorageName, blobStorage.Status.Phase, blobStorage.Status.Message, blobStorage.Status.Provider)
		}
	}
}

// Function to get a custom resource and unmarshal the json to a resource type
func getResourceAndUnMarshalJsonToResource(ctx *TestingContext, resource string, resourceName string, resourceType interface{}) error {
	requestBody, err := getCustomResourceJson(ctx, resource, resourceName)

	if err != nil {
		return err
	}

	err = json.Unmarshal(requestBody, resourceType)

	if err != nil {
		return err
	}

	return nil
}

// Function to get a custom resource json without needing to depend on operator-sdk
func getCustomResourceJson(ctx *TestingContext, resource string, resourceName string) ([]byte, error) {
	request := ctx.ExtensionClient.RESTClient().Get().Resource(resource).Name(resourceName).Namespace(RHOAMOperatorNamespace).RequestURI(requestUrl).Do(context.TODO())
	requestBody, err := request.Raw()

	if err != nil {
		return nil, err
	}

	return requestBody, nil
}

// Get resource provision strategy
func getResourceStrategy(t TestingTB, ctx *TestingContext) string {
	isClusterStorage, err := isClusterStorage(ctx)
	if err != nil {
		t.Fatal("error getting isClusterStorage:", err)
	}

	if !isClusterStorage {
		return externalProvider
	}

	return openShiftProvider
}
