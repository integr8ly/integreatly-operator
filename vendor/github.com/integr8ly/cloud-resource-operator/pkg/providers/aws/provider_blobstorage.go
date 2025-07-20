package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"time"

	croType "github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1/types"
	"github.com/integr8ly/cloud-resource-operator/pkg/annotations"

	v1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/integr8ly/cloud-resource-operator/pkg/resources"

	"github.com/integr8ly/cloud-resource-operator/pkg/providers"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1"
	errorUtil "github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// provider name and default create options
const (
	blobstorageProviderName               = "aws-s3"
	defaultAwsBucketNameLength            = 40
	DetailsBlobStorageBucketName          = "bucketName"
	DetailsBlobStorageBucketRegion        = "bucketRegion"
	DetailsBlobStorageCredentialKeyID     = "credentialKeyID" // #nosec G101 -- false positive (ref: https://securego.io/docs/rules/g101.html)
	DetailsBlobStorageCredentialSecretKey = "credentialSecretKey"
	defaultForceBucketDeletion            = false

	// bucket accessibility defaults
	defaultBlockPublicAcls       = true
	defaultBlockPublicPolicy     = true
	defaultIgnorePublicAcls      = true
	defaultRestrictPublicBuckets = true

	// bucket encryption defaults
	defaultEncryptionSSEAlgorithm = types.ServerSideEncryptionAes256
)

// BlobStorageDeploymentDetails Provider-specific details about the AWS S3 bucket created
type BlobStorageDeploymentDetails struct {
	BucketName          string
	BucketRegion        string
	CredentialKeyID     string
	CredentialSecretKey string
}

func (d *BlobStorageDeploymentDetails) Data() map[string][]byte {
	return map[string][]byte{
		DetailsBlobStorageBucketName:          []byte(d.BucketName),
		DetailsBlobStorageBucketRegion:        []byte(d.BucketRegion),
		DetailsBlobStorageCredentialKeyID:     []byte(d.CredentialKeyID),
		DetailsBlobStorageCredentialSecretKey: []byte(d.CredentialSecretKey),
	}
}

var _ providers.BlobStorageProvider = (*BlobStorageProvider)(nil)

// BlobStorageProvider implementation for AWS S3
type BlobStorageProvider struct {
	Client            client.Client
	Logger            *logrus.Entry
	CredentialManager CredentialManager
	ConfigManager     ConfigManager
}

func NewAWSBlobStorageProvider(client client.Client, logger *logrus.Entry) (*BlobStorageProvider, error) {
	cm, err := NewCredentialManager(client)
	if err != nil {
		return nil, err
	}
	return &BlobStorageProvider{
		Client:            client,
		Logger:            logger.WithFields(logrus.Fields{"provider": blobstorageProviderName}),
		CredentialManager: cm,
		ConfigManager:     NewDefaultConfigMapConfigManager(client),
	}, nil
}

func (p *BlobStorageProvider) GetName() string {
	return blobstorageProviderName
}

func (p *BlobStorageProvider) SupportsStrategy(d string) bool {
	return d == providers.AWSDeploymentStrategy
}

func (p *BlobStorageProvider) GetReconcileTime(bs *v1alpha1.BlobStorage) time.Duration {
	if bs.Status.Phase != croType.PhaseComplete {
		return time.Second * 60
	}
	return resources.GetForcedReconcileTimeOrDefault(defaultReconcileTime)
}

// S3DeleteStrat custom s3 delete strat
type S3DeleteStrat struct {
	_ struct{} `type:"structure"`

	ForceBucketDeletion *bool `json:"forceBucketDeletion"`
}

// CreateStorage Create S3 bucket from strategy config and credentials to interact with it
func (p *BlobStorageProvider) CreateStorage(ctx context.Context, bs *v1alpha1.BlobStorage) (*providers.BlobStorageInstance, croType.StatusMessage, error) {
	// handle provider-specific finalizer
	if err := resources.CreateFinalizer(ctx, p.Client, bs, DefaultFinalizer); err != nil {
		return nil, "failed to set finalizer", err
	}

	// info about the bucket to be created
	p.Logger.Infof("getting aws s3 bucket config for blob storage instance %s", bs.Name)
	bucketCreateCfg, _, stratCfg, err := p.buildS3BucketConfig(ctx, bs)
	if err != nil {
		errMsg := "failed to build s3 bucket config"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// create the credentials to be used by the aws resource providers, not to be used by end-user
	p.Logger.Infof("creating provider credentials for creating s3 buckets, in namespace %s", bs.Namespace)
	providerCreds, err := p.CredentialManager.ReconcileProviderCredentials(ctx, bs.Namespace)
	if err != nil {
		errMsg := fmt.Sprintf("failed to reconcile aws blob storage provider credentials for blob storage instance %s", bs.Name)
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	// setup aws s3 sdk session
	p.Logger.Infof("creating new aws sdk config in region %s", stratCfg.Region)
	cfg, err := CreateConfigFromStrategy(ctx, p.Client, providerCreds, stratCfg)
	if err != nil {
		errMsg := "failed to create aws config to create s3 bucket"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	s3Client := s3.NewFromConfig(*cfg)

	// create bucket if it doesn't already exist, if it does exist then use the existing bucket
	p.Logger.Infof("reconciling aws s3 bucket %s", *bucketCreateCfg.Bucket)
	msg, err := p.reconcileBucketCreate(ctx, bs, s3Client, bucketCreateCfg)
	if err != nil {
		return nil, msg, errorUtil.Wrapf(err, string(msg))
	}

	// create the credentials to be used by the end-user, whoever created the blobstorage instance
	endUserCredsName := buildEndUserCredentialsNameFromBucket(*bucketCreateCfg.Bucket)
	p.Logger.Infof("creating end-user credentials with name %s for managing s3 bucket %s", endUserCredsName, *bucketCreateCfg.Bucket)
	endUserCreds, err := p.CredentialManager.ReconcileBucketOwnerCredentials(ctx, endUserCredsName, bs.Namespace, *bucketCreateCfg.Bucket)
	if err != nil {
		errMsg := fmt.Sprintf("failed to reconcile s3 end-user credentials for blob storage instance %s", bs.Name)
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	// blobstorageinstance that will be returned if everything is successful
	var bsi *providers.BlobStorageInstance
	switch p.CredentialManager.(type) {
	case *STSCredentialManager:
		bsi = &providers.BlobStorageInstance{
			DeploymentDetails: &BlobStorageDeploymentDetails{
				BucketName:   *bucketCreateCfg.Bucket,
				BucketRegion: stratCfg.Region,
			},
		}
	default:
		bsi = &providers.BlobStorageInstance{
			DeploymentDetails: &BlobStorageDeploymentDetails{
				BucketName:          *bucketCreateCfg.Bucket,
				BucketRegion:        stratCfg.Region,
				CredentialKeyID:     endUserCreds.AccessKeyID,
				CredentialSecretKey: endUserCreds.SecretAccessKey,
			},
		}
	}

	// Adding tags to s3
	msg, err = p.TagBlobStorage(ctx, *bucketCreateCfg.Bucket, bs, stratCfg.Region, s3Client)
	if err != nil {
		errMsg := fmt.Sprintf("failed to add tags to bucket: %s", msg)
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	p.Logger.Infof("creation handler for blob storage instance %s in namespace %s finished successfully", bs.Name, bs.Namespace)
	return bsi, msg, nil
}

func (p *BlobStorageProvider) TagBlobStorage(ctx context.Context, bucketName string, bs *v1alpha1.BlobStorage, stratCfgRegion string, s3Client S3API) (croType.StatusMessage, error) {
	p.Logger.Infof("bucket %s found, Adding tags to bucket", bucketName)

	bucketTags, err := p.getDefaultS3Tags(ctx, bs)
	if err != nil {
		msg := "Failed to build default tags"
		return croType.StatusMessage(msg), errorUtil.Wrapf(err, msg)
	}

	// adding the tags to S3
	_, err = s3Client.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{
		Bucket: aws.String(bucketName),
		Tagging: &types.Tagging{
			TagSet: bucketTags,
		},
	})
	if err != nil {
		errMsg := fmt.Sprintf("failed to add tags to S3 bucket: %s", err)
		return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	logrus.Infof("successfully created or updated tags to s3 bucket %s", bucketName)
	return "successfully created and tagged", nil
}

// DeleteStorage Delete S3 bucket and credentials to add objects to it
func (p *BlobStorageProvider) DeleteStorage(ctx context.Context, bs *v1alpha1.BlobStorage) (croType.StatusMessage, error) {
	p.Logger.Infof("deleting blob storage instance %s via aws s3", bs.Name)

	// resolve bucket information for bucket created by provider
	p.Logger.Infof("getting aws s3 bucket config for blob storage instance %s", bs.Name)
	bucketCreateCfg, bucketDeleteCfg, stratCfg, err := p.buildS3BucketConfig(ctx, bs)
	if err != nil {
		errMsg := "failed to build s3 bucket config"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// get provider aws creds so the bucket can be deleted
	p.Logger.Infof("creating provider credentials for creating s3 buckets, in namespace %s", bs.Namespace)
	providerCreds, err := p.CredentialManager.ReconcileProviderCredentials(ctx, bs.Namespace)
	if err != nil {
		errMsg := fmt.Sprintf("failed to reconcile aws provider credentials for blob storage instance %s", bs.Name)
		return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	// create new s3 config
	p.Logger.Infof("creating new aws sdk config in region %s", stratCfg.Region)
	cfg, err := CreateConfigFromStrategy(ctx, p.Client, providerCreds, stratCfg)
	if err != nil {
		errMsg := "failed to create aws session to delete s3 bucket"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// delete the bucket that was created by the provider
	s3Client := s3.NewFromConfig(*cfg)
	return p.reconcileBucketDelete(ctx, bs, s3Client, bucketCreateCfg, bucketDeleteCfg)
}

func (p *BlobStorageProvider) reconcileBucketDelete(ctx context.Context, bs *v1alpha1.BlobStorage, s3Client S3API, bucketCfg *s3.CreateBucketInput, bucketDeleteCfg *S3DeleteStrat) (croType.StatusMessage, error) {
	buckets, err := getS3buckets(ctx, s3Client)
	if err != nil {
		return "error getting s3 buckets", err
	}

	// check if the bucket has already been deleted
	var foundBucket *types.Bucket
	found := false
	for _, i := range buckets {
		currentBucket := i //fix gosec error G601 (CWE-118): Implicit memory aliasing in for loop.
		if *currentBucket.Name == *bucketCfg.Bucket {
			foundBucket = &currentBucket
			found = true
			break
		}
	}
	logrus.Info("found bucket", foundBucket)

	if !found {
		if err := p.removeCredsAndFinalizer(ctx, bs, s3Client, bucketCfg, bucketDeleteCfg); err != nil {
			errMsg := fmt.Sprintf("unable to remove credential secrets and finalizer for %s", *bucketCfg.Bucket)
			return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
		}
	}

	bucketSize, err := getBucketSize(ctx, s3Client, bucketCfg)
	if err != nil {
		errMsg := fmt.Sprintf("unable to get bucket size : %s", *bucketCfg.Bucket)
		return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	if *bucketDeleteCfg.ForceBucketDeletion || bucketSize == 0 {
		if err := emptyBucket(ctx, s3Client, bucketCfg); err != nil {
			errMsg := fmt.Sprintf("unable to empty bucket : %q", *bucketCfg.Bucket)
			return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
		}

		if err := deleteBucket(ctx, s3Client, bucketCfg); err != nil {
			errMsg := fmt.Sprintf("unable to delete bucket : %s", *bucketCfg.Bucket)
			return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
		}
	}

	if err := p.removeCredsAndFinalizer(ctx, bs, s3Client, bucketCfg, bucketDeleteCfg); err != nil {
		errMsg := fmt.Sprintf("unable to remove credential secrets and finalizer for %s", *bucketCfg.Bucket)
		return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	return croType.StatusEmpty, nil
}

func (p *BlobStorageProvider) removeCredsAndFinalizer(ctx context.Context, bs *v1alpha1.BlobStorage, s3Client S3API, bucketCfg *s3.CreateBucketInput, bucketDeleteCfg *S3DeleteStrat) error {
	// build end user credential name
	endUserCredsName := buildEndUserCredentialsNameFromBucket(*bucketCfg.Bucket)

	// remove the credentials request created by the provider
	p.Logger.Infof("deleting end-user credential request %s in namespace %s", endUserCredsName, bs.Namespace)
	endUserCredsReq := &v1.CredentialsRequest{
		ObjectMeta: controllerruntime.ObjectMeta{
			Name:      endUserCredsName,
			Namespace: bs.Namespace,
		},
	}
	if err := p.Client.Delete(ctx, endUserCredsReq); err != nil {
		if !errors.IsNotFound(err) {
			errMsg := fmt.Sprintf("failed to delete credential request %s", endUserCredsName)
			return errorUtil.Wrapf(err, errMsg)
		}
		p.Logger.Infof("could not find credential request %s, already deleted, continuing", endUserCredsName)
	}

	// remove the finalizer
	resources.RemoveFinalizer(&bs.ObjectMeta, DefaultFinalizer)
	if err := p.Client.Update(ctx, bs); err != nil {
		errMsg := "failed to update blob storage cr as part of finalizer reconcile"
		return errorUtil.Wrapf(err, errMsg)
	}

	p.exposeBlobStorageMetrics(ctx, bs)

	return nil
}

func (p *BlobStorageProvider) getDefaultS3Tags(ctx context.Context, cr *v1alpha1.BlobStorage) ([]types.Tag, error) {
	tags, _, err := resources.GetDefaultResourceTags(ctx, p.Client, cr.Spec.Type, cr.Name, cr.ObjectMeta.Labels["productName"])
	if err != nil {
		msg := "Failed to get default s3 tags"
		return nil, errorUtil.Wrapf(err, msg)
	}
	return genericToS3Tags(tags), nil
}

func deleteBucket(ctx context.Context, s3Client S3API, bucketCfg *s3.CreateBucketInput) error {
	_, err := s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: bucketCfg.Bucket,
	})
	// Error handling changed in V2 of aws-go-sdk
	if err != nil {
		var apiErr smithy.APIError
		if errorUtil.As(err, &apiErr) {
			// Check if the error is an S3-specific error
			var s3Err *types.NoSuchBucket
			if errorUtil.As(err, &s3Err) {
				// This is a NoSuchBucket error, so we ignore it
				fmt.Println("Bucket does not exist, skipping deletion")
			} else {
				// Some other AWS API error
				return errorUtil.Wrapf(err, fmt.Sprintf("failed to delete s3 bucket: %s", err))
			}
		} else {
			// Other non-AWS errors
			return errorUtil.Wrapf(err, fmt.Sprintf("failed to delete s3 bucket: %s", err))
		}

	}
	return nil
}

func emptyBucket(ctx context.Context, s3Client S3API, bucketCfg *s3.CreateBucketInput) error {
	size, err := getBucketSize(ctx, s3Client, bucketCfg)
	if err != nil {
		return err
	}

	if size == 0 {
		return nil
	}

	paginator := s3.NewListObjectsV2Paginator(s3Client, &s3.ListObjectsV2Input{
		Bucket: bucketCfg.Bucket,
	})

	// paginator has replaced list,  Iterate through pages and collect object identifiers
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return fmt.Errorf("failed to list objects in bucket %q: %w", *bucketCfg.Bucket, err)
		}

		if len(page.Contents) == 0 {
			break
		}

		var objects []types.ObjectIdentifier
		for _, obj := range page.Contents {
			objects = append(objects, types.ObjectIdentifier{Key: obj.Key})
		}

		// Perform batch delete
		_, err = s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: bucketCfg.Bucket,
			Delete: &types.Delete{
				Objects: objects,
				Quiet:   aws.Bool(true),
			},
		})
		if err != nil {
			return fmt.Errorf("unable to delete objects from bucket %q: %w", *bucketCfg.Bucket, err)
		}
	}

	return nil
}

func getBucketSize(ctx context.Context, s3Client S3API, bucketCfg *s3.CreateBucketInput) (int, error) {
	// get bucket items
	resp, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: aws.String(*bucketCfg.Bucket)})
	if err != nil {
		errMsg := fmt.Sprintf("unable to list items in bucket %q", *bucketCfg.Bucket)
		return 0, errorUtil.Wrapf(err, errMsg)
	}
	return len(resp.Contents), nil
}

// getS3Buckets retrieves a list of S3 buckets using AWS SDK v2
func getS3Buckets(ctx context.Context, s3client S3API) ([]types.Bucket, error) {
	output, err := s3client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, errorUtil.Wrap(err, "failed to list AWS S3 buckets")
	}
	return output.Buckets, nil
}

func (p *BlobStorageProvider) reconcileBucketCreate(ctx context.Context, bs *v1alpha1.BlobStorage, s3Client S3API, bucketCfg *s3.CreateBucketInput) (croType.StatusMessage, error) {
	// the aws access key can sometimes still not be registered in aws on first try, so loop
	p.Logger.Infof("listing existing aws s3 buckets")
	buckets, err := getS3Buckets(ctx, s3Client)
	if err != nil {
		errMsg := "failed to list existing aws s3 buckets, credentials could be reconciling"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// check if bucket already exists
	p.Logger.Infof("checking if aws s3 bucket %s already exists", *bucketCfg.Bucket)
	found := false
	var foundBucket *types.Bucket
	for _, b := range buckets {
		currentBucket := b //fix gosec error G601 (CWE-118): Implicit memory aliasing in for loop.
		if *currentBucket.Name == *bucketCfg.Bucket {
			foundBucket = &currentBucket
			found = true
			break
		}
	}

	defer p.exposeBlobStorageMetrics(ctx, bs)

	if found {
		if err = reconcileS3BucketSettings(ctx, aws.ToString(foundBucket.Name), s3Client); err != nil {
			errMsg := fmt.Sprintf("failed to set s3 bucket settings %s", *foundBucket.Name)
			return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
		}
		msg := fmt.Sprintf("using bucket %s", *foundBucket.Name)
		return croType.StatusMessage(msg), nil
	}

	// foundBucket == nil at this point, so if the CR already has a resourceIdentifier
	// annotation, then we expect it to be there. We shouldn't create it again, it will require
	// manual intervention to restore from a backup.
	if annotations.Has(bs, ResourceIdentifierAnnotation) {
		errMsg := fmt.Sprintf("BlobStorage CR %s in %s namespace has %s annotation with value %s, but no corresponding S3 Bucket was found",
			bs.Name, bs.Namespace, ResourceIdentifierAnnotation, bs.ObjectMeta.Annotations[ResourceIdentifierAnnotation])
		return croType.StatusMessage(errMsg), fmt.Errorf(errMsg)
	}

	// create bucket
	p.Logger.Infof("bucket %s not found, creating bucket", *bucketCfg.Bucket)
	_, err = s3Client.CreateBucket(ctx, bucketCfg)
	if err != nil {
		errMsg := fmt.Sprintf("failed to create s3 bucket %s", *bucketCfg.Bucket)
		return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	annotations.Add(bs, ResourceIdentifierAnnotation, *bucketCfg.Bucket)
	if err := p.Client.Update(ctx, bs); err != nil {
		errMsg := "failed to add annotation"
		return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	if err = reconcileS3BucketSettings(ctx, aws.ToString(bucketCfg.Bucket), s3Client); err != nil {
		errMsg := fmt.Sprintf("failed to set s3 bucket settings on bucket creation %s", aws.ToString(bucketCfg.Bucket))
		return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}
	p.Logger.Infof("reconcile for aws s3 bucket completed successfully")
	return "successfully reconciled", nil
}

// function to get s3 buckets, used to check/wait on AWS credentials
func getS3buckets(ctx context.Context, s3Client S3API) ([]types.Bucket, error) {
	var existingBuckets []types.Bucket
	err := wait.PollUntilContextTimeout(ctx, time.Second*5, time.Minute*5, true, func(ctx context.Context) (bool, error) {
		listOutput, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
		if err != nil {
			return false, nil
		}
		existingBuckets = listOutput.Buckets
		return true, nil
	})
	if err != nil {
		return nil, errorUtil.Wrap(err, "timed out waiting to list s3 buckets")
	}
	return existingBuckets, nil
}

func reconcileS3BucketSettings(ctx context.Context, bucket string, s3Client S3API) error {
	_, err := s3Client.PutPublicAccessBlock(ctx, &s3.PutPublicAccessBlockInput{
		Bucket: aws.String(bucket),
		PublicAccessBlockConfiguration: &types.PublicAccessBlockConfiguration{
			BlockPublicAcls:       aws.Bool(defaultBlockPublicAcls),
			BlockPublicPolicy:     aws.Bool(defaultBlockPublicPolicy),
			IgnorePublicAcls:      aws.Bool(defaultIgnorePublicAcls),
			RestrictPublicBuckets: aws.Bool(defaultRestrictPublicBuckets),
		},
	})
	if err != nil {
		return errorUtil.Wrapf(err, "failed to set client access settings on bucket %s", bucket)
	}
	_, err = s3Client.PutBucketEncryption(ctx, &s3.PutBucketEncryptionInput{
		Bucket: aws.String(bucket),
		ServerSideEncryptionConfiguration: &types.ServerSideEncryptionConfiguration{
			Rules: []types.ServerSideEncryptionRule{
				{
					ApplyServerSideEncryptionByDefault: &types.ServerSideEncryptionByDefault{
						SSEAlgorithm: defaultEncryptionSSEAlgorithm,
					},
				},
			},
		},
	})
	if err != nil {
		return errorUtil.Wrapf(err, "failed to set encryption settings on bucket %s", bucket)
	}
	return nil
}

func (p *BlobStorageProvider) buildS3BucketConfig(ctx context.Context, bs *v1alpha1.BlobStorage) (*s3.CreateBucketInput, *S3DeleteStrat, *StrategyConfig, error) {
	// info about the bucket to be created
	p.Logger.Infof("getting aws s3 bucket config for blob storage instance %s", bs.Name)
	bucketCreateCfg, bucketDeleteCfg, stratCfg, err := p.getS3BucketConfig(ctx, bs)
	if err != nil {
		return nil, nil, nil, errorUtil.Wrapf(err, fmt.Sprintf("failed to retrieve aws s3 bucket config for blob storage instance %s", bs.Name))
	}

	// cluster infra info
	p.Logger.Info("getting cluster id from infrastructure for bucket naming")
	bucketName, err := resources.BuildInfraNameFromObject(ctx, p.Client, bs.ObjectMeta, defaultAwsBucketNameLength)
	if err != nil {
		return nil, nil, nil, errorUtil.Wrapf(err, fmt.Sprintf("failed to retrieve aws s3 bucket config for blob storage instance %s", bs.Name))
	}
	if bucketCreateCfg.Bucket == nil {
		bucketCreateCfg.Bucket = aws.String(bucketName)
	}

	if bucketDeleteCfg.ForceBucketDeletion == nil {
		bucketDeleteCfg.ForceBucketDeletion = aws.Bool(defaultForceBucketDeletion)
	}

	return bucketCreateCfg, bucketDeleteCfg, stratCfg, nil
}

func (p *BlobStorageProvider) getS3BucketConfig(ctx context.Context, bs *v1alpha1.BlobStorage) (*s3.CreateBucketInput, *S3DeleteStrat, *StrategyConfig, error) {
	stratCfg, err := p.ConfigManager.ReadStorageStrategy(ctx, providers.BlobStorageResourceType, bs.Spec.Tier)
	if err != nil {
		return nil, nil, nil, errorUtil.Wrap(err, "failed to read aws strategy config")
	}

	defRegion, err := GetRegionFromStrategyOrDefault(ctx, p.Client, stratCfg)
	if err != nil {
		return nil, nil, nil, errorUtil.Wrap(err, "failed to get default region")
	}
	if stratCfg.Region == "" {
		p.Logger.Debugf("region not set in deployment strategy configuration, using default region %s", defRegion)
		stratCfg.Region = defRegion
	}

	// create s3 bucket config created by the provider
	s3createConfig := &s3.CreateBucketInput{}
	if err = json.Unmarshal(stratCfg.CreateStrategy, s3createConfig); err != nil {
		return nil, nil, nil, errorUtil.Wrap(err, "failed to unmarshal aws s3 create strat configuration")
	}
	// setting Location Restraint required now for all regions outside of us-east-1 for s3 buckets
	// setting it equal to the default region for the cluster.
	if defRegion != "us-east-1" {
		s3createConfig.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(defRegion),
		}
	}

	// delete s3 bucket config created by the provider
	s3deleteConfig := &S3DeleteStrat{}
	if err = json.Unmarshal(stratCfg.DeleteStrategy, s3deleteConfig); err != nil {
		return nil, nil, nil, errorUtil.Wrapf(err, "failed to unmarshal aws s3 delete strat configuration")
	}

	return s3createConfig, s3deleteConfig, stratCfg, nil
}

func buildEndUserCredentialsNameFromBucket(b string) string {
	return fmt.Sprintf("cro-aws-s3-%s-creds", b)
}

func buildBlobStorageStatusMetricLabels(cr *v1alpha1.BlobStorage, clusterID, bucketName string, phase croType.StatusPhase) map[string]string {
	labels := map[string]string{}
	labels[resources.LabelClusterIDKey] = clusterID
	labels[resources.LabelResourceIDKey] = cr.Name
	labels[resources.LabelNamespaceKey] = cr.Namespace
	labels[resources.LabelInstanceIDKey] = bucketName
	labels[resources.LabelProductNameKey] = cr.Labels["productName"]
	labels[resources.LabelStrategyKey] = blobstorageProviderName
	labels[resources.LabelStatusPhaseKey] = string(phase)
	return labels
}

func (p *BlobStorageProvider) exposeBlobStorageMetrics(ctx context.Context, cr *v1alpha1.BlobStorage) {
	// build instance name
	bucketName, err := resources.BuildInfraNameFromObject(ctx, p.Client, cr.ObjectMeta, defaultAwsBucketNameLength)
	if err != nil {
		logrus.Errorf("error occurred while building instance name during blob storage metrics: %v", err)
	}

	// get Cluster Id
	logrus.Info("setting blob storage information metric")
	clusterID, err := resources.GetClusterID(ctx, p.Client)
	if err != nil {
		logrus.Errorf("failed to get cluster id while exposing information metric for %v", bucketName)
		return
	}

	// set generic status metrics
	// a single metric should be exposed for each possible phase
	// the value of the metric should be 1.0 when the resource is in that phase
	// the value of the metric should be 0.0 when the resource is not in that phase
	// this follows the approach that pod status
	for _, phase := range []croType.StatusPhase{croType.PhaseFailed, croType.PhaseDeleteInProgress, croType.PhasePaused, croType.PhaseComplete, croType.PhaseInProgress} {
		labelsFailed := buildBlobStorageStatusMetricLabels(cr, clusterID, bucketName, phase)
		resources.SetMetric(resources.DefaultBlobStorageStatusMetricName, labelsFailed, resources.Btof64(cr.Status.Phase == phase))
	}
}
