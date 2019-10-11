package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/aws/aws-sdk-go/aws/awserr"

	v1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"

	"github.com/integr8ly/cloud-resource-operator/pkg/providers"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	errorUtil "github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	blobstorageProviderName = "aws-s3"

	dataBucketName          = "bucketName"
	dataCredentialKeyID     = "credentialKeyID"
	dataCredentialSecretKey = "credentialSecretKey"

	bucketNameLen = 40
)

// BlobStorageDeploymentDetails Provider-specific details about the AWS S3 bucket created
type BlobStorageDeploymentDetails struct {
	BucketName          string
	CredentialKeyID     string
	CredentialSecretKey string
}

func (d *BlobStorageDeploymentDetails) Data() map[string][]byte {
	return map[string][]byte{
		dataBucketName:          []byte(d.BucketName),
		dataCredentialKeyID:     []byte(d.CredentialKeyID),
		dataCredentialSecretKey: []byte(d.CredentialSecretKey),
	}
}

var _ providers.BlobStorageProvider = (*BlobStorageProvider)(nil)

// BlobStorageProvider BlobStorageProvider implementation for AWS S3
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

// CreateStorage Create S3 bucket from strategy config and credentials to interact with it
func (p *BlobStorageProvider) CreateStorage(ctx context.Context, bs *v1alpha1.BlobStorage) (*providers.BlobStorageInstance, v1alpha1.StatusMessage, error) {
	p.Logger.Infof("creating blob storage instance %s via aws s3", bs.Name)

	// handle provider-specific finalizer
	if err := resources.CreateFinalizer(ctx, p.Client, bs, DefaultFinalizer); err != nil {
		return nil, "failed to set finalizer", err
	}

	// info about the bucket to be created
	p.Logger.Infof("getting aws s3 bucket config for blob storage instance %s", bs.Name)
	bucketCreateCfg, stratCfg, err := p.getS3BucketConfig(ctx, bs)
	// cluster infra info
	p.Logger.Info("getting cluster id from infrastructure for bucket naming")
	bucketName, err := buildInfraNameFromObject(ctx, p.Client, bs.ObjectMeta, bucketNameLen)
	if err != nil {
		return nil, "failed to retrieve aws s3 bucket config", errorUtil.Wrapf(err, "failed to retrieve aws s3 bucket config for blob storage instance %s", bs.Name)
	}
	if bucketCreateCfg.Bucket == nil {
		bucketCreateCfg.Bucket = aws.String(bucketName)
	}

	// create the credentials to be used by the end-user, whoever created the blobstorage instance
	endUserCredsName := buildEndUserCredentialsNameFromBucket(bucketName)
	p.Logger.Infof("creating end-user credentials with name %s for managing s3 bucket %s", endUserCredsName, *bucketCreateCfg.Bucket)
	endUserCreds, _, err := p.CredentialManager.ReoncileBucketOwnerCredentials(ctx, endUserCredsName, bs.Namespace, *bucketCreateCfg.Bucket)
	if err != nil {
		return nil, "failed to reconcile s3 end-user credentials", errorUtil.Wrapf(err, "failed to reconcile s3 end-user credentials for blob storage instance %s", bs.Name)
	}

	// create the credentials to be used by the aws resource providers, not to be used by end-user
	p.Logger.Infof("creating provider credentials for creating s3 buckets, in namespace %s", bs.Namespace)
	providerCreds, err := p.CredentialManager.ReconcileProviderCredentials(ctx, bs.Namespace)
	if err != nil {
		return nil, "failed to reconcile aws blob storage provider credentials", errorUtil.Wrapf(err, "failed to reconcile aws blob storage provider credentials for blob storage instance %s", bs.Name)
	}

	// setup aws s3 sdk session
	p.Logger.Infof("creating new aws sdk session in region %s", stratCfg.Region)
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String(stratCfg.Region),
		Credentials: credentials.NewStaticCredentials(providerCreds.AccessKeyID, providerCreds.SecretAccessKey, ""),
	}))
	s3svc := s3.New(sess)

	// pre-create the blobstorageinstance that will be returned if everything is successful
	bsi := &providers.BlobStorageInstance{
		DeploymentDetails: &BlobStorageDeploymentDetails{
			BucketName:          *bucketCreateCfg.Bucket,
			CredentialKeyID:     endUserCreds.AccessKeyID,
			CredentialSecretKey: endUserCreds.SecretAccessKey,
		},
	}

	// create bucket if it doesn't already exist, if it does exist then use the existing bucket
	p.Logger.Infof("reconciling aws s3 bucket %s", *bucketCreateCfg.Bucket)
	if err := p.reconcileBucketCreate(ctx, s3svc, bucketCreateCfg); err != nil {
		return nil, "failed to reconcile aws s3 bucket", errorUtil.Wrapf(err, "failed to reconcile aws s3 bucket %s", *bucketCreateCfg.Bucket)
	}

	p.Logger.Infof("creation handler for blob storage instance %s in namespace %s finished successfully", bs.Name, bs.Namespace)
	return bsi, "blob storage successfully created", nil
}

// DeleteStorage Delete S3 bucket and credentials to add objects to it
func (p *BlobStorageProvider) DeleteStorage(ctx context.Context, bs *v1alpha1.BlobStorage) (v1alpha1.StatusMessage, error) {
	p.Logger.Infof("deleting blob storage instance %s via aws s3", bs.Name)

	// cluster infra info
	p.Logger.Info("getting cluster id from infrastructure for bucket naming")
	bucketName, err := buildInfraNameFromObject(ctx, p.Client, bs.ObjectMeta, bucketNameLen)
	if err != nil {
		return "failed to construct name for s3 bucket from cluster infrastructure", errorUtil.Wrap(err, "failed to build bucket name")
	}

	// resolve bucket information for bucket created by provider
	p.Logger.Infof("getting aws s3 bucket config for blob storage instance %s", bs.Name)
	bucketCreateCfg, stratCfg, err := p.getS3BucketConfig(ctx, bs)
	if err != nil {
		return "failed to retrieve aws s3 bucket config", errorUtil.Wrapf(err, "failed to retrieve aws s3 bucket config for blob storage instance %s", bs.Name)
	}
	if bucketCreateCfg.Bucket == nil {
		bucketCreateCfg.Bucket = aws.String(bucketName)
	}

	// get provider aws creds so the bucket can be deleted
	p.Logger.Infof("creating provider credentials for creating s3 buckets, in namespace %s", bs.Namespace)
	providerCreds, err := p.CredentialManager.ReconcileProviderCredentials(ctx, bs.Namespace)
	if err != nil {
		return "failed to reconcile aws provider credentials", errorUtil.Wrapf(err, "failed to reconcile aws provider credentials for blob storage instance %s", bs.Name)
	}
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String(stratCfg.Region),
		Credentials: credentials.NewStaticCredentials(providerCreds.AccessKeyID, providerCreds.SecretAccessKey, ""),
	}))

	// delete the bucket that was created by the provider
	p.Logger.Infof("creating new aws sdk session in region %s", stratCfg.Region)
	s3svc := s3.New(sess)

	if err = p.reconcileBucketDelete(ctx, s3svc, bucketCreateCfg); err != nil {
		return "failed to delete aws s3 bucket", errorUtil.Wrapf(err, "failed to delete aws s3 bucket %s", *bucketCreateCfg.Bucket)
	}

	// remove the credentials request created by the provider
	endUserCredsName := buildEndUserCredentialsNameFromBucket(bucketName)
	p.Logger.Infof("deleting end-user credential request %s in namespace %s", endUserCredsName, bs.Namespace)
	endUserCredsReq := &v1.CredentialsRequest{
		ObjectMeta: controllerruntime.ObjectMeta{
			Name:      endUserCredsName,
			Namespace: bs.Namespace,
		},
	}
	if err := p.Client.Delete(ctx, endUserCredsReq); err != nil {
		if !errors.IsNotFound(err) {
			return "failed to delete credential request", errorUtil.Wrapf(err, "failed to delete credential request %s", endUserCredsName)
		}
		p.Logger.Infof("could not find credential request %s, already deleted, continuing", endUserCredsName)
	}

	// remove the finalizer added by the provider
	p.Logger.Infof("deleting finalizer %s from blob storage instance %s in namespace %s", DefaultFinalizer, bs.Name, bs.Namespace)
	resources.RemoveFinalizer(&bs.ObjectMeta, DefaultFinalizer)
	if err := p.Client.Update(ctx, bs); err != nil {
		return "failed to update instance as part of finalizer reconcile", errorUtil.Wrapf(err, "failed to update instance %s as part of finalizer reconcile", bs.Name)
	}

	p.Logger.Infof("deletion handler for blob storage instance %s in namespace %s finished successfully", bs.Name, bs.Namespace)
	return "blob storage successfully deleted", nil
}

func (p *BlobStorageProvider) reconcileBucketDelete(ctx context.Context, s3svc s3iface.S3API, bucketCfg *s3.CreateBucketInput) error {
	_, err := s3svc.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: bucketCfg.Bucket,
	})
	s3err, isAWSErr := err.(awserr.Error)
	if err != nil && !isAWSErr {
		return errorUtil.Wrapf(err, "failed to delete s3 bucket %s", *bucketCfg.Bucket)
	}
	if err != nil && isAWSErr {
		if s3err.Code() != s3.ErrCodeNoSuchBucket {
			return errorUtil.Wrapf(err, "failed to delete aws s3 bucket %s, aws error", *bucketCfg.Bucket)
		}
		p.Logger.Info("failed to find s3 bucket, it may have already been deleted, continuing")
	}
	err = s3svc.WaitUntilBucketNotExists(&s3.HeadBucketInput{
		Bucket: bucketCfg.Bucket,
	})
	if err != nil {
		return errorUtil.Wrapf(err, "failed to wait for s3 bucket deletion, %s", *bucketCfg.Bucket)
	}
	return nil
}

func (p *BlobStorageProvider) reconcileBucketCreate(ctx context.Context, s3svc s3iface.S3API, bucketCfg *s3.CreateBucketInput) error {
	// the aws access key can sometimes still not be registered in aws on first try, so loop
	p.Logger.Infof("listing existing aws s3 buckets")
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
		return errorUtil.Wrap(err, "timed out waiting to list s3 buckets")
	}

	// create bucket if it doesn't already exist, if it does exist then use the existing bucket
	p.Logger.Infof("checking if aws s3 bucket %s already exists", *bucketCfg.Bucket)
	var foundBucket *s3.Bucket
	for _, b := range existingBuckets {
		if *b.Name == *bucketCfg.Bucket {
			foundBucket = b
			break
		}
	}
	if foundBucket != nil {
		p.Logger.Infof("bucket %s already exists, using that", *foundBucket.Name)
		return nil
	}
	p.Logger.Infof("bucket %s not found, creating bucket", *bucketCfg.Bucket)
	_, err = s3svc.CreateBucket(bucketCfg)
	if err != nil {
		return errorUtil.Wrapf(err, "failed to create s3 bucket %s", *bucketCfg.Bucket)
	}
	p.Logger.Infof("reconcile for aws s3 bucket completed successfully, bucket created")
	return nil
}

func (p *BlobStorageProvider) getS3BucketConfig(ctx context.Context, bs *v1alpha1.BlobStorage) (*s3.CreateBucketInput, *StrategyConfig, error) {
	stratCfg, err := p.ConfigManager.ReadStorageStrategy(ctx, providers.BlobStorageResourceType, bs.Spec.Tier)
	if err != nil {
		return nil, nil, errorUtil.Wrap(err, "failed to read aws strategy config")
	}
	if stratCfg.Region == "" {
		p.Logger.Debugf("region not set in deployment strategy configuration, using default region %s", DefaultRegion)
		stratCfg.Region = DefaultRegion
	}

	// delete the s3 bucket created by the provider
	s3cbi := &s3.CreateBucketInput{}
	if err = json.Unmarshal(stratCfg.CreateStrategy, s3cbi); err != nil {
		return nil, nil, errorUtil.Wrap(err, "failed to unmarshal aws s3 configuration")
	}
	return s3cbi, stratCfg, nil
}

func buildEndUserCredentialsNameFromBucket(b string) string {
	return fmt.Sprintf("cro-aws-s3-%s-creds", b)
}
