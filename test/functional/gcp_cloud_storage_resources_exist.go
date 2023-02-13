package functional

import (
	goctx "context"
	"fmt"
	"log"
	"sort"

	"cloud.google.com/go/storage"
	croResources "github.com/integr8ly/cloud-resource-operator/pkg/resources"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/integr8ly/integreatly-operator/test/common"
)

const (
	resourceLabel = "integreatly.org/resource-name"
)

// This Test will be revised, as the plan is to use NooBaa PV's for on-cluster s3 support.
// We won't be installing buckets into Google Cloud Storage for RHOAM

func TestGCPCloudStorageBlobStorageResourcesExist(t common.TestingTB, testingCtx *common.TestingContext) {
	ctx := goctx.Background()
	serviceAccountJson, err := getGCPCredentials(ctx, testingCtx.Client)
	if err != nil {
		t.Fatal("failed to retrieve gcp credentials %v", err)
	}
	projectID, err := croResources.GetGCPProject(ctx, testingCtx.Client)
	if err != nil {
		t.Fatal("error get Default Project ID %w", err)
	}
	storageClient, err := storage.NewClient(ctx, option.WithCredentialsJSON(serviceAccountJson))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer storageClient.Close()

	cloudBucketsList := storageClient.Buckets(ctx, projectID)

	goContext := goctx.TODO()

	rhmi, err := common.GetRHMI(testingCtx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	// buckets to which the blob belongs. As defined in RHMI Blob CR (?)
	bucketsList, testErrors := GetCloudObjectStorageBlobStorageResourceIDs(goContext, testingCtx.Client, rhmi)
	if len(testErrors) != 0 {
		t.Fatalf("test cro blob storage exists failed with the following errors : %s", testErrors)
	}
	if len(bucketsList) != 1 { // legacy - in aws s3 - number of buckets was changed 1/2/1, see history of aws_s3_resource_exist.go;
		// below we are working with list, although only one bucket (to allow future changes)
		t.Fatalf("There should be exactly 1 blob resources for %s install type: actual: %d", rhmi.Spec.Type, len(bucketsList))
	}

	if !verifyBucketsList(bucketsList, cloudBucketsList) {
		t.Fatalf("inconsistency found between buckets defined in Rhmi Block CR and buckets available in Cloud Storage")
	}

	backupsFound := new(bool)
	threeScaleFound := new(bool)

	for _, bucketName := range bucketsList {
		bucket := storageClient.Bucket(bucketName)
		attrs, err := bucket.Attrs(ctx)
		if err != nil {
			t.Fatalf("error getting bucket arributes, bucket :%s, %w", bucketName, err)
		}

		if err := verifyEncryptionOfGcpCloudStoreBucket(attrs, bucketName); err != nil {
			testErrors = append(testErrors, err.Error())
		}

		if err := verifyPublicAccessOfGcpCloudStoreBucket(attrs, bucketName); err != nil {
			testErrors = append(testErrors, err.Error())
		}

		if err := verifyBucketResourceNames(attrs, bucketName, backupsFound, threeScaleFound, rhmi.Name); err != nil {
			testErrors = append(testErrors, err.Error())
		}
	}

	// Expect just three scale bucket for managed api install
	if !*threeScaleFound {
		testErrors = append(testErrors, "Failed to find appropriate resource names for buckets for managed api install")
	}

	if len(testErrors) != 0 {
		t.Fatalf("test s3 blob storage failed with the following errors : %s", testErrors)
	}

}

// check that list of list1 (bucketNames from CR) is a subset of in list2 (buckets in Google cloud storage)
func verifyBucketsList(list []string, list2 *storage.BucketIterator) bool {
	sort.Strings(list)
	for {
		bucketAttrs, err := list2.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return false
		}
		ind := sort.SearchStrings(list, bucketAttrs.Name)
		if ind >= len(list) {
			return false
		}
	}
	return true
}

func verifyEncryptionOfGcpCloudStoreBucket(attrs *storage.BucketAttrs, identifier string) error {
	// Cloud Storage always encrypts your data on the server side, before it is written to disk, at no additional charge.
	// Besides this standard, Google-managed behavior, there are additional ways to encrypt your data when using Cloud Storage
	// https://cloud.google.com/storage/docs/encryption
	// DefaultKMSKeyName - a Cloud KMS key name, in the form projects/P/locations/L/keyRings/R/cryptoKeys/K,
	// that will be used to encrypt objects inserted into this bucket, if no encryption method is specified.
	// The key's location must be the same as the bucket's.
	bucketEncryption := attrs.Encryption
	if bucketEncryption.DefaultKMSKeyName == "" {
		return fmt.Errorf("bucket Encryption issue - DefaultKMSKeyName is not set in bucket's attributes :%s", identifier)
	}
	return nil
}

func verifyPublicAccessOfGcpCloudStoreBucket(attrs *storage.BucketAttrs, identifier string) error {
	if attrs.PublicAccessPrevention.String() == "" {
		return fmt.Errorf("bucket arributes PublicAccessPrevention is not set :%s", identifier)
	}
	return nil
}

func verifyBucketResourceNames(attrs *storage.BucketAttrs, identifier string,
	backupsFound *bool, threeScaleFound *bool, installationName string) error {
	labelsMap := attrs.Labels
	if len(labelsMap) == 0 {
		return fmt.Errorf("no labels found for bucket :%s", identifier)
	}
	val, found := labelsMap[managedLabelKey]
	if !found || val != managedLabelValue {
		return fmt.Errorf("expected label for bucket missing :%s, %s", identifier, managedLabelKey)
	}
	val, found = labelsMap[resourceLabel]
	// If the key exists
	if found {
		if val == getExpectedBackupBucketResourceName(installationName) {
			*backupsFound = true
			return nil
		}
		if val == getExpectedThreeScaleBucketResourceName(installationName) {
			*threeScaleFound = true
			return nil
		}
	}
	return fmt.Errorf("no resource name label for bucket :%s", identifier)
}
