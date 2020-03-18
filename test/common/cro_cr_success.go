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
	originalStrategy := getResourceStrategy(t, ctx)

	for _, postgresName := range postgresToCheck {
		// AMQAuthService postgres is always in cluster
		strategy := originalStrategy
		if postgresName == constants.AMQAuthServicePostgres {
			strategy = openShiftProvider
		}

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

func TestCRORedisSuccessfulState(t *testing.T, ctx *TestingContext) {
	strategy := getResourceStrategy(t, ctx)

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

func TestCROBlobStorageSuccessfulState(t *testing.T, ctx *TestingContext) {
	strategy := getResourceStrategy(t, ctx)

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
	request := ctx.ExtensionClient.RESTClient().Get().Resource(resource).Name(resourceName).Namespace(RHMIOperatorNamespace).RequestURI(requestUrl).Do()
	requestBody, err := request.Raw()

	if err != nil {
		return nil, err
	}

	return requestBody, nil
}

// Get resource provision strategy
func getResourceStrategy(t *testing.T, ctx *TestingContext) string {
	isClusterStorage, err := IsClusterStorage(ctx)
	if err != nil {
		t.Fatal("error getting isClusterStorage:", err)
	}

	if !isClusterStorage {
		return externalProvider
	}

	return openShiftProvider
}
