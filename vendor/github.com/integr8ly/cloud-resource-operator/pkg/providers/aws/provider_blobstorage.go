package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/integr8ly/cloud-resource-operator/pkg/annotations"
	croType "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	v1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/integr8ly/cloud-resource-operator/pkg/resources"

	"github.com/integr8ly/cloud-resource-operator/pkg/providers"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	errorUtil "github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//provider name and default create options
const (
	blobstorageProviderName               = "aws-s3"
	defaultAwsBucketNameLength            = 40
	DetailsBlobStorageBucketName          = "bucketName"
	DetailsBlobStorageBucketRegion        = "bucketRegion"
	DetailsBlobStorageCredentialKeyID     = "credentialKeyID"
	DetailsBlobStorageCredentialSecretKey = "credentialSecretKey"
	defaultForceBucketDeletion            = false

	// bucket accessibility defaults
	defaultBlockPublicAcls       = true
	defaultBlockPublicPolicy     = true
	defaultIgnorePublicAcls      = true
	defaultRestrictPublicBuckets = true

	// bucket encryption defaults
	defaultEncryptionSSEAlgorithm = s3.ServerSideEncryptionAes256
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

func NewAWSBlobStorageProvider(client client.Client, logger *logrus.Entry) *BlobStorageProvider {
	return &BlobStorageProvider{
		Client:            client,
		Logger:            logger.WithFields(logrus.Fields{"provider": blobstorageProviderName}),
		CredentialManager: NewCredentialMinterCredentialManager(client),
		ConfigManager:     NewDefaultConfigMapConfigManager(client),
	}
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

//S3DeleteStrat custom s3 delete strat
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

	// create the credentials to be used by the end-user, whoever created the blobstorage instance
	endUserCredsName := buildEndUserCredentialsNameFromBucket(*bucketCreateCfg.Bucket)
	p.Logger.Infof("creating end-user credentials with name %s for managing s3 bucket %s", endUserCredsName, *bucketCreateCfg.Bucket)
	endUserCreds, _, err := p.CredentialManager.ReoncileBucketOwnerCredentials(ctx, endUserCredsName, bs.Namespace, *bucketCreateCfg.Bucket)
	if err != nil {
		errMsg := fmt.Sprintf("failed to reconcile s3 end-user credentials for blob storage instance %s", bs.Name)
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	// create the credentials to be used by the aws resource providers, not to be used by end-user
	p.Logger.Infof("creating provider credentials for creating s3 buckets, in namespace %s", bs.Namespace)
	providerCreds, err := p.CredentialManager.ReconcileProviderCredentials(ctx, bs.Namespace)
	if err != nil {
		errMsg := fmt.Sprintf("failed to reconcile aws blob storage provider credentials for blob storage instance %s", bs.Name)
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	// setup aws s3 sdk session
	p.Logger.Infof("creating new aws sdk session in region %s", stratCfg.Region)
	sess, err := CreateSessionFromStrategy(ctx, p.Client, providerCreds.AccessKeyID, providerCreds.SecretAccessKey, stratCfg)
	if err != nil {
		errMsg := "failed to create aws session to create s3 bucket"
		return nil, croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}
	s3Client := s3.New(sess)

	// create bucket if it doesn't already exist, if it does exist then use the existing bucket
	p.Logger.Infof("reconciling aws s3 bucket %s", *bucketCreateCfg.Bucket)
	msg, err := p.reconcileBucketCreate(ctx, bs, s3Client, bucketCreateCfg)
	if err != nil {
		return nil, msg, errorUtil.Wrapf(err, string(msg))
	}

	// blobstorageinstance that will be returned if everything is successful
	bsi := &providers.BlobStorageInstance{
		DeploymentDetails: &BlobStorageDeploymentDetails{
			BucketName:          *bucketCreateCfg.Bucket,
			BucketRegion:        stratCfg.Region,
			CredentialKeyID:     endUserCreds.AccessKeyID,
			CredentialSecretKey: endUserCreds.SecretAccessKey,
		},
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

func (p *BlobStorageProvider) TagBlobStorage(ctx context.Context, bucketName string, bs *v1alpha1.BlobStorage, stratCfgRegion string, s3svc s3iface.S3API) (croType.StatusMessage, error) {
	p.Logger.Infof("bucket %s found, Adding tags to bucket", bucketName)

	// set tag values that will always be added
	defaultOrganizationTag := resources.GetOrganizationTag()
	clusterID, err := resources.GetClusterID(ctx, p.Client)
	if err != nil {
		errMsg := "failed to get cluster id"
		return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}
	bucketTags := []*s3.Tag{
		{
			Key:   aws.String(defaultOrganizationTag + "clusterID"),
			Value: aws.String(clusterID),
		},
		{
			Key:   aws.String(defaultOrganizationTag + "resource-type"),
			Value: aws.String(bs.Spec.Type),
		},
		{
			Key:   aws.String(defaultOrganizationTag + "resource-name"),
			Value: aws.String(bs.Name),
		},
	}

	// check if product name exists and append label
	if bs.ObjectMeta.Labels["productName"] != "" {
		productTag := &s3.Tag{
			Key:   aws.String(defaultOrganizationTag + "product-name"),
			Value: aws.String(bs.ObjectMeta.Labels["productName"]),
		}
		bucketTags = append(bucketTags, productTag)
	}

	// adding the tags to S3
	_, err = s3svc.PutBucketTagging(&s3.PutBucketTaggingInput{
		Bucket: aws.String(bucketName),
		Tagging: &s3.Tagging{
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

	// create new s3 session
	p.Logger.Infof("creating new aws sdk session in region %s", stratCfg.Region)
	sess, err := CreateSessionFromStrategy(ctx, p.Client, providerCreds.AccessKeyID, providerCreds.SecretAccessKey, stratCfg)
	if err != nil {
		errMsg := "failed to create aws session to delete s3 bucket"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// delete the bucket that was created by the provider
	return p.reconcileBucketDelete(ctx, bs, s3.New(sess), bucketCreateCfg, bucketDeleteCfg)
}

func (p *BlobStorageProvider) reconcileBucketDelete(ctx context.Context, bs *v1alpha1.BlobStorage, s3svc s3iface.S3API, bucketCfg *s3.CreateBucketInput, bucketDeleteCfg *S3DeleteStrat) (croType.StatusMessage, error) {
	buckets, err := getS3buckets(s3svc)
	if err != nil {
		return "error getting s3 buckets", err
	}

	// check if the bucket has already been deleted
	var foundBucket *s3.Bucket
	for _, i := range buckets {
		if *i.Name == *bucketCfg.Bucket {
			foundBucket = i
			break
		}
	}

	if foundBucket == nil {
		if err := p.removeCredsAndFinalizer(ctx, bs, s3svc, bucketCfg, bucketDeleteCfg); err != nil {
			errMsg := fmt.Sprintf("unable to remove credential secrets and finalizer for %s", *bucketCfg.Bucket)
			return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
		}
	}

	bucketSize, err := getBucketSize(s3svc, bucketCfg)
	if err != nil {
		errMsg := fmt.Sprintf("unable to get bucket size : %s", *bucketCfg.Bucket)
		return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	if *bucketDeleteCfg.ForceBucketDeletion || bucketSize == 0 {
		if err := emptyBucket(s3svc, bucketCfg); err != nil {
			errMsg := fmt.Sprintf("unable to empty bucket : %q", *bucketCfg.Bucket)
			return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
		}

		if err := deleteBucket(s3svc, bucketCfg); err != nil {
			errMsg := fmt.Sprintf("unable to delete bucket : %s", *bucketCfg.Bucket)
			return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
		}
	}

	if err := p.removeCredsAndFinalizer(ctx, bs, s3svc, bucketCfg, bucketDeleteCfg); err != nil {
		errMsg := fmt.Sprintf("unable to remove credential secrets and finalizer for %s", *bucketCfg.Bucket)
		return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	return croType.StatusEmpty, nil
}

func (p *BlobStorageProvider) removeCredsAndFinalizer(ctx context.Context, bs *v1alpha1.BlobStorage, s3svc s3iface.S3API, bucketCfg *s3.CreateBucketInput, bucketDeleteCfg *S3DeleteStrat) error {
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

func deleteBucket(s3svc s3iface.S3API, bucketCfg *s3.CreateBucketInput) error {
	_, err := s3svc.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: bucketCfg.Bucket,
	})
	s3err, isAwsErr := err.(awserr.Error)
	if err != nil && (!isAwsErr || s3err.Code() != s3.ErrCodeNoSuchBucket) {
		return errorUtil.Wrapf(err, fmt.Sprintf("failed to delete s3 bucket: %s", err))
	}
	return nil
}

func emptyBucket(s3svc s3iface.S3API, bucketCfg *s3.CreateBucketInput) error {
	size, err := getBucketSize(s3svc, bucketCfg)
	if err != nil {
		return err
	}

	if size == 0 {
		return nil
	}

	// Setup BatchDeleteIterator to iterate through a list of objects.
	iter := s3manager.NewDeleteListIterator(s3svc, &s3.ListObjectsInput{
		Bucket: aws.String(*bucketCfg.Bucket),
	})

	// Traverse iterator deleting each object
	if err := s3manager.NewBatchDeleteWithClient(s3svc).Delete(aws.BackgroundContext(), iter); err != nil {
		errMsg := fmt.Sprintf("unable to delete objects from bucket")
		return errorUtil.Wrapf(err, errMsg)
	}

	return nil
}

func getBucketSize(s3svc s3iface.S3API, bucketCfg *s3.CreateBucketInput) (int, error) {
	// get bucket items
	resp, err := s3svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(*bucketCfg.Bucket)})
	if err != nil {
		errMsg := fmt.Sprintf("unable to list items in bucket %q", *bucketCfg.Bucket)
		return 0, errorUtil.Wrapf(err, errMsg)
	}
	return len(resp.Contents), nil
}

func (p *BlobStorageProvider) reconcileBucketCreate(ctx context.Context, bs *v1alpha1.BlobStorage, s3svc s3iface.S3API, bucketCfg *s3.CreateBucketInput) (croType.StatusMessage, error) {
	// the aws access key can sometimes still not be registered in aws on first try, so loop
	p.Logger.Infof("listing existing aws s3 buckets")
	buckets, err := getS3buckets(s3svc)
	if err != nil {
		errMsg := "failed to list existing aws s3 buckets, credentials could be reconciling"
		return croType.StatusMessage(errMsg), errorUtil.Wrap(err, errMsg)
	}

	// check if bucket already exists
	p.Logger.Infof("checking if aws s3 bucket %s already exists", *bucketCfg.Bucket)
	var foundBucket *s3.Bucket
	for _, b := range buckets {
		if *b.Name == *bucketCfg.Bucket {
			foundBucket = b
			break
		}
	}

	defer p.exposeBlobStorageMetrics(ctx, bs)

	if foundBucket != nil {
		if err = reconcileS3BucketSettings(aws.StringValue(foundBucket.Name), s3svc); err != nil {
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
	_, err = s3svc.CreateBucket(bucketCfg)
	if err != nil {
		errMsg := fmt.Sprintf("failed to create s3 bucket %s", *bucketCfg.Bucket)
		return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	annotations.Add(bs, ResourceIdentifierAnnotation, *bucketCfg.Bucket)
	if err := p.Client.Update(ctx, bs); err != nil {
		errMsg := "failed to add annotation"
		return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}

	if err = reconcileS3BucketSettings(aws.StringValue(bucketCfg.Bucket), s3svc); err != nil {
		errMsg := fmt.Sprintf("failed to set s3 bucket settings on bucket creation %s", aws.StringValue(bucketCfg.Bucket))
		return croType.StatusMessage(errMsg), errorUtil.Wrapf(err, errMsg)
	}
	p.Logger.Infof("reconcile for aws s3 bucket completed successfully")
	return "successfully reconciled", nil
}

// function to get s3 buckets, used to check/wait on AWS credentials
func getS3buckets(s3svc s3iface.S3API) ([]*s3.Bucket, error) {
	var existingBuckets []*s3.Bucket
	err := wait.PollImmediate(time.Second*5, time.Minute*5, func() (done bool, err error) {
		listOutput, err := s3svc.ListBuckets(nil)
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

func reconcileS3BucketSettings(bucket string, s3svc s3iface.S3API) error {
	_, err := s3svc.PutPublicAccessBlock(&s3.PutPublicAccessBlockInput{
		Bucket: aws.String(bucket),
		PublicAccessBlockConfiguration: &s3.PublicAccessBlockConfiguration{
			BlockPublicAcls:       aws.Bool(defaultBlockPublicAcls),
			BlockPublicPolicy:     aws.Bool(defaultBlockPublicPolicy),
			IgnorePublicAcls:      aws.Bool(defaultIgnorePublicAcls),
			RestrictPublicBuckets: aws.Bool(defaultRestrictPublicBuckets),
		},
	})
	if err != nil {
		return errorUtil.Wrapf(err, "failed to set client access settings on bucket %s", bucket)
	}
	_, err = s3svc.PutBucketEncryption(&s3.PutBucketEncryptionInput{
		Bucket: aws.String(bucket),
		ServerSideEncryptionConfiguration: &s3.ServerSideEncryptionConfiguration{
			Rules: []*s3.ServerSideEncryptionRule{
				{
					ApplyServerSideEncryptionByDefault: &s3.ServerSideEncryptionByDefault{
						SSEAlgorithm: aws.String(defaultEncryptionSSEAlgorithm),
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
	bucketName, err := BuildInfraNameFromObject(ctx, p.Client, bs.ObjectMeta, defaultAwsBucketNameLength)
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
	labels["clusterID"] = clusterID
	labels["resourceID"] = cr.Name
	labels["namespace"] = cr.Namespace
	labels["instanceID"] = bucketName
	labels["productName"] = cr.Labels["productName"]
	labels["strategy"] = blobstorageProviderName
	labels["statusPhase"] = string(phase)
	return labels
}

func (p *BlobStorageProvider) exposeBlobStorageMetrics(ctx context.Context, cr *v1alpha1.BlobStorage) {
	// build instance name
	bucketName, err := BuildInfraNameFromObject(ctx, p.Client, cr.ObjectMeta, defaultAwsBucketNameLength)
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
