package common

import (
	"encoding/json"
	"fmt"
	"testing"

	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	croTypes "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
)

const (
	requestNameSpace  = "redhat-rhmi-operator"
	requestUrl        = "/apis/integreatly.org/v1alpha1"
	openShiftProvider = "openshift"
	externalProvider  = "aws"
)

var (
	postgresToCheck = []string{
		fmt.Sprintf("%s%s", constants.CodeReadyPostgresPrefix, InstallationName),
		fmt.Sprintf("%s%s", constants.ThreeScalePostgresPrefix, InstallationName),
		fmt.Sprintf("%s%s", constants.RHSSOPostgresPrefix, InstallationName),
		fmt.Sprintf("%s%s", constants.RHSSOUserProstgresPrefix, InstallationName),
		fmt.Sprintf("%s%s", constants.UPSPostgresPrefix, InstallationName),
		// TODO - Add check for Fuse postgres here when task for supporting external resources is done - https://issues.redhat.com/browse/INTLY-3239
		constants.AMQAuthServicePostgres,
	}
	redisToCheck = []string{
		fmt.Sprintf("%s%s", constants.ThreeScaleBackendRedisPrefix, InstallationName),
		fmt.Sprintf("%s%s", constants.ThreeScaleSystemRedisPrefix, InstallationName),
	}
	blobStorageToCheck = []string{
		fmt.Sprintf("%s%s", constants.BackupsBlobStoragePrefix, InstallationName),
		fmt.Sprintf("%s%s", constants.ThreeScaleBlobStoragePrefix, InstallationName),
	}
)

func TestCROPostgresSuccessfulState(t *testing.T, ctx *TestingContext) {
	var testErrors []string
	originalStrategy := getResourceStrategy(t, ctx)

	for _, postgresName := range postgresToCheck {
		// AMQAuthService postgres is always in cluster
		strategy := originalStrategy
		if postgresName == constants.AMQAuthServicePostgres {
			strategy = openShiftProvider
		}

		postgres := &crov1.Postgres{}
		err := getResourceAndUnMarshalJsonToResource(ctx, "postgres", postgresName, postgres, testErrors)

		if err == nil && postgres.Status.Phase != croTypes.PhaseComplete && postgres.Status.Strategy != strategy {
			testErrors = append(testErrors, fmt.Sprintf("\n%s Postgres not ready with phase: %s, message: %s, provider, %s", postgresName, postgres.Status.Phase, postgres.Status.Message, postgres.Status.Provider))
		}
	}

	if len(testErrors) != 0 {
		t.Fatalf("Test CRO Postgress Succesful failed with the following errors: %s", testErrors)
	}
}

func TestCRORedisSuccessfulState(t *testing.T, ctx *TestingContext) {
	var testErrors []string
	strategy := getResourceStrategy(t, ctx)

	for _, redisName := range redisToCheck {
		redis := &crov1.Redis{}
		err := getResourceAndUnMarshalJsonToResource(ctx, "redis", redisName, redis, testErrors)

		if err == nil && redis.Status.Phase != croTypes.PhaseComplete && redis.Status.Strategy != strategy {
			testErrors = append(testErrors, fmt.Sprintf("\n%s redis not ready with phase: %s, message: %s, provider, %s", redisName, redis.Status.Phase, redis.Status.Message, redis.Status.Provider))
		}
	}

	if len(testErrors) != 0 {
		t.Fatalf("Test CRO Redis Succesful failed with the following errors: %s", testErrors)
	}
}

func TestCROBlobStorageSuccessfulState(t *testing.T, ctx *TestingContext) {
	var testErrors []string
	strategy := getResourceStrategy(t, ctx)

	for _, blobStorageName := range blobStorageToCheck {
		blobStorage := &crov1.BlobStorage{}
		err := getResourceAndUnMarshalJsonToResource(ctx, "blobstorages", blobStorageName, blobStorage, testErrors)

		if err == nil && blobStorage.Status.Phase != croTypes.PhaseComplete && blobStorage.Status.Strategy != strategy {
			testErrors = append(testErrors, fmt.Sprintf("\n%s blob storage not ready with phase: %s, message: %s, provider, %s", blobStorageName, blobStorage.Status.Phase, blobStorage.Status.Message, blobStorage.Status.Provider))
		}
	}

	if len(testErrors) != 0 {
		t.Fatalf("Test CRO BlobStorage Succesful failed with the following errors: %s", testErrors)
	}
}

// Function to get a custom resource and unmarshal the json to a resource type
func getResourceAndUnMarshalJsonToResource(ctx *TestingContext, resource string, resourceName string, resourceType interface{}, testErrors []string) error {
	requestBody, err := getCustomResourceJson(ctx, resource, resourceName)

	if err != nil {
		testErrors = append(testErrors, fmt.Sprintf("\nFailed to get custom resource: %s", err))
		return err
	}

	err = json.Unmarshal(requestBody, resourceType)

	if err != nil {
		testErrors = append(testErrors, fmt.Sprintf("\nFailed to unmarshall json: %s", err))
		return err
	}

	return nil
}

// Function to get a custom resource json without needing to depend on operator-sdk
func getCustomResourceJson(ctx *TestingContext, resource string, resourceName string) ([]byte, error) {
	request := ctx.ExtensionClient.RESTClient().Get().Resource(resource).Name(resourceName).Namespace(requestNameSpace).RequestURI(requestUrl).Do()
	requestBody, err := request.Raw()

	if err != nil {
		return nil, err
	}

	return requestBody, nil
}

// Get resource provision strategy
func getResourceStrategy(t *testing.T, ctx *TestingContext) string {
	isClusterStorage, err := isClusterStorage(ctx)
	if err != nil {
		t.Fatal("error getting isClusterStorage:", err)
	}

	if !isClusterStorage {
		return externalProvider
	}

	return openShiftProvider
}
