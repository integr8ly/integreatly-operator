package functional

import (
	goctx "context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"testing"

	"github.com/integr8ly/integreatly-operator/test/common"
)

const (
	expectedBucketEncryption     = "AES256"
	resourceNameTag              = "integreatly.org/resource-name"
	backupBucketResourceName     = "backups-blobstorage-rhmi"
	threeScaleBucketResourceName = "threescale-blobstorage-rhmi"
)

func TestAWSs3BlobStorageResourcesExist(t *testing.T, ctx *common.TestingContext) {
	goContext := goctx.TODO()

	rhmi, err := common.GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	s3ResourceIDs, testErrors := GetS3BlobStorageResourceIDs(goContext, ctx.Client, rhmi)

	if len(testErrors) != 0 {
		t.Fatalf("test cro blob storage exists failed with the following errors : %s", testErrors)
	}

	if len(s3ResourceIDs) != 2 {
		t.Fatalf("There should be exactly 2 blob resources: actual: %d", len(s3ResourceIDs))
	}

	sess, err := CreateAWSSession(goContext, ctx.Client)
	if err != nil {
		t.Fatalf("failed to create aws session: %v", err)
	}

	s3api := s3.New(sess)

	backupsFound := new(bool)
	threeScaleFound := new(bool)

	for _, resourceIdentifier := range s3ResourceIDs {

		err := verifyEncryption(s3api, resourceIdentifier)
		if err != nil {
			testErrors = append(testErrors, err.Error())
		}

		err = verifyPublicAccessBlock(s3api, resourceIdentifier)
		if err != nil {
			testErrors = append(testErrors, err.Error())
		}

		err = verifyResourceNames(s3api, resourceIdentifier, backupsFound, threeScaleFound)
		if err != nil {
			testErrors = append(testErrors, err.Error())
		}
	}

	if *backupsFound == false || *threeScaleFound == false {
		testErrors = append(testErrors, fmt.Sprintf("Failed to find appropriate resource names for buckets"))
	}

	if len(testErrors) != 0 {
		t.Fatalf("test s3 blob storage failed with the following errors : %s", testErrors)
	}

}

func verifyEncryption(s3api *s3.S3, identifier string) error {

	enc, err := s3api.GetBucketEncryption(&s3.GetBucketEncryptionInput{Bucket: aws.String(identifier)})
	if err != nil {
		return fmt.Errorf("Error getting bucket encryption, bucket :%s, %w", identifier, err)
	}
	if enc.ServerSideEncryptionConfiguration == nil {
		return fmt.Errorf("Server Side Encryption does not exist for bucket :%s", identifier)
	}
	rules := enc.ServerSideEncryptionConfiguration.Rules
	for _, rule := range rules {
		if *rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm == expectedBucketEncryption {
			return nil
		}
	}

	return fmt.Errorf("Server Side Encryption does not exist for bucket :%s, %w", identifier, err)
}

func verifyPublicAccessBlock(s3api *s3.S3, identifier string) error {

	pab, err := s3api.GetPublicAccessBlock(&s3.GetPublicAccessBlockInput{Bucket: aws.String(identifier)})
	if err != nil {
		return fmt.Errorf("Error getting bucket public access block, bucket :%s, %w", identifier, err)
	}
	if pab.PublicAccessBlockConfiguration == nil {
		return fmt.Errorf("Public Access is not defined for bucket :%s", identifier)
	}
	if *pab.PublicAccessBlockConfiguration.BlockPublicPolicy == true {
		return nil
	} else {
		return fmt.Errorf("Public Access is not blocked for Bucket :%s", identifier)
	}
}

func verifyResourceNames(s3api *s3.S3, identifier string, backupsFound *bool, threeScaleFound *bool) error {

	tags, err := s3api.GetBucketTagging(&s3.GetBucketTaggingInput{Bucket: aws.String(identifier)})
	if err != nil {
		return fmt.Errorf("Error getting bucket tags, bucket :%s, %w", identifier, err)
	}
	if tags.TagSet == nil {
		return fmt.Errorf("Tags are not defined for bucket :%s", identifier)
	}
	for i := range tags.TagSet {
		tag := tags.TagSet[i]
		if *tag.Key == resourceNameTag {
			if *tag.Value == backupBucketResourceName {
				*backupsFound = true
				return nil
			}
			if *tag.Value == threeScaleBucketResourceName {
				*threeScaleFound = true
				return nil
			}
			return fmt.Errorf("Unexpected resource name for bucket :%s, %s", identifier, *tag.Value)
		}
	}
	return fmt.Errorf("No resource name tag for bucket :%s", identifier)
}
