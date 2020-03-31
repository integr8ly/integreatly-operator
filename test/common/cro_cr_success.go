package common

import (
	"encoding/json"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/products/amqonline"
	"github.com/integr8ly/integreatly-operator/pkg/products/ups"
	"testing"

	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	croTypes "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
)

const (
	requestUrl        = "/apis/integreatly.org/v1alpha1"
	openShiftProvider = "openshift"
	externalProvider  = "aws"
)

var (
	postgresToCheck = []string{
		//fmt.Sprintf("%s%s", codeready.PostgresPrefix, InstallationName),
		//fmt.Sprintf("%s%s", threescale.PostgresPrefix, InstallationName),
		//fmt.Sprintf("%s%s", rhsso.PostgresPrefix, InstallationName),
		//fmt.Sprintf("%s%s", rhssouser.PostgresPrefix, InstallationName),
		fmt.Sprintf("%s%s", ups.PostgresPrefix, InstallationName),
		// TODO - Add check for Fuse postgres here when task for supporting external resources is done - https://issues.redhat.com/browse/INTLY-3239
		amqonline.AuthServicePostgres,
	}
	redisToCheck = []string{
		//fmt.Sprintf("%s%s", threescale.BackendRedisPrefix, InstallationName),
		//fmt.Sprintf("%s%s", threescale.SystemRedisPrefix, InstallationName),
	}
	blobStorageToCheck = []string{
		//fmt.Sprintf("%s%s", cloudresources.BackupsBlobStoragePrefix, InstallationName),
		//fmt.Sprintf("%s%s", threescale.BlobStoragePrefix, InstallationName),
	}
)

func TestCROPostgresSuccessfulState(t *testing.T, ctx *TestingContext) {
	originalStrategy := getResourceStrategy(t, ctx)

	for _, postgresName := range postgresToCheck {
		// AMQAuthService postgres is always in cluster
		strategy := originalStrategy
		if postgresName == amqonline.AuthServicePostgres {
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
	request := ctx.ExtensionClient.RESTClient().Get().Resource(resource).Name(resourceName).Namespace(RHMIOperatorNameSpace).RequestURI(requestUrl).Do()
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
