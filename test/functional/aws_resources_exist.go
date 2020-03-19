package functional

import (
	goctx "context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3control"
	"github.com/integr8ly/integreatly-operator/test/common"
	configv1 "github.com/openshift/api/config/v1"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"testing"
)

var (
	expectedRDSResources = []*rds.DBInstance{
		{
			DBName:             aws.String(ThreeScaleResourceName),
			DeletionProtection: aws.Bool(true),
			MultiAZ:            aws.Bool(true),
			StorageEncrypted:   aws.Bool(true),
		},
		{
			DBName:             aws.String(UPSResourceName),
			DeletionProtection: aws.Bool(true),
			MultiAZ:            aws.Bool(true),
			StorageEncrypted:   aws.Bool(true),
		},
		{
			DBName:             aws.String(CRResourceName),
			DeletionProtection: aws.Bool(true),
			MultiAZ:            aws.Bool(true),
			StorageEncrypted:   aws.Bool(true),
		},
		{
			DBName:             aws.String(CRResourceName),
			DeletionProtection: aws.Bool(true),
			MultiAZ:            aws.Bool(true),
			StorageEncrypted:   aws.Bool(true),
		},

		//TODO implement Fuse here when it's ready
		{
			DBName:             aws.String(ClusterSSOResourceName),
			DeletionProtection: aws.Bool(true),
			MultiAZ:            aws.Bool(true),
			StorageEncrypted:   aws.Bool(true),
		},
		{
			DBName:             aws.String(ApplicationSSOResourceName),
			DeletionProtection: aws.Bool(true),
			MultiAZ:            aws.Bool(true),
			StorageEncrypted:   aws.Bool(true),
		},
	}
	expectedElasticacheResources = []*elasticache.ReplicationGroup{
		{
			AtRestEncryptionEnabled:  aws.Bool(true),
			TransitEncryptionEnabled: aws.Bool(true),
		},
	}
	expectedS3Resources = []*s3.PublicAccessBlockConfiguration{
		{
			BlockPublicAcls:       aws.Bool(true),
			BlockPublicPolicy:     aws.Bool(true),
			IgnorePublicAcls:      aws.Bool(true),
			RestrictPublicBuckets: aws.Bool(true),
		},
	}
)

func AWSResourcesExistTest(t *testing.T, ctx *common.TestingContext) {
	context := goctx.TODO()

	//retrieve aws credentials for creating an aws session
	awsSecretAccessKey, awsAccessKeyID, err := getAWSCredentials(context, ctx.Client)
	if err != nil {
		t.Fatalf("failed to get AWS credentials : %v", err)
	}

	//retrieve aws region for creating an aws session
	region, err := getAWSRegion(context, ctx.Client)
	if err != nil {
		t.Fatalf("failed to get AWS cluster region : %v", err)
	}

	//retrieve clusterID for filtering resources
	clusterID, err := getClusterID(context, ctx.Client)
	if err != nil {
		t.Errorf("failed to get clusterID : %v", err)
	}
	t.Log(clusterID)

	//create new session for aws api's
	session, err := createAWSSession(awsSecretAccessKey, awsAccessKeyID, region)
	if err != nil {
		t.Fatalf("failed to create session : %v", err)
	}

	//create new rds api
	rdsapi := rds.New(session)
	foundRDSInstances, err := rdsapi.DescribeDBInstances(&rds.DescribeDBInstancesInput{})
	if err != nil {
		t.Fatalf("failed to retrieve rds instances : %v", err)
	}

	fmt.Print(foundRDSInstances)

	//instances, instanceErr := getRDSResources(*createAWSSession())
	//elasticacheResourcesMap, elasticacheErr := getElasticacheResources(*createAWSSession(), clusterID)
	//s3ResourcesMap, s3Err := getS3Resources(*createAWSSession(), clusterID)
	//if instanceErr != nil || elasticacheErr != nil || s3Err != nil {
	//	t.Errorf("error getting one or more aws resource")
	//}
	//resourceMatchesExpectedState := make(map[string]bool)
	////Check RDS
	//for _, expectedInstance := range expectedRDSResources {
	//	for _, returnedInstance := range instances {
	//		if expectedInstance.DBName != returnedInstance.DBName {
	//			continue
	//		}
	//		if !common.EqualResourceBool(
	//			[]bool{aws.BoolValue(expectedInstance.DeletionProtection), aws.BoolValue(expectedInstance.MultiAZ), aws.BoolValue(expectedInstance.StorageEncrypted)},
	//			[]bool{aws.BoolValue(returnedInstance.DeletionProtection), aws.BoolValue(returnedInstance.MultiAZ), aws.BoolValue(returnedInstance.StorageEncrypted)}) || returnedInstance.PreferredBackupWindow == "" {
	//			t.Logf("RDS resources are not in the expected state")
	//			resourceMatchesExpectedState[aws.StringValue(expectedInstance.DBName)] = false
	//		}
	//		resourceMatchesExpectedState[aws.StringValue(expectedInstance.DBName)] = true
	//
	//	}
	//}
	t.Fatal("Aiden is ...")
	////Check Elasticache
	//for _, expectedReplicationGroup := range expectedElasticacheResources {
	//	for arn, replicationgroups := range elasticacheResourcesMap {
	//		if strings.Contains(arn, ThreeScaleNameElement) {
	//			for _, repgroup := range replicationgroups {
	//				if !common.EqualResourceBool(
	//					[]bool{aws.BoolValue(expectedReplicationGroup.AtRestEncryptionEnabled), aws.BoolValue(expectedReplicationGroup.TransitEncryptionEnabled)},
	//					[]bool{aws.BoolValue(repgroup.TransitEncryptionEnabled), aws.BoolValue(repgroup.TransitEncryptionEnabled)}) || repgroup.SnapshotWindow == "" || len(replicationgroups) != expectedElasticacheReplicationGroupCount {
	//					t.Errorf("Elasticache replicationGroups are not in the expected state, replicationGroups are missing or they aren't configured correctly")
	//				}
	//
	//			}
	//		}
	//
	//	}
	//}
	////Check S3
	//for _, expectedAccessBlockConfig := range expectedS3Resources {
	//	for arn, accessBlockConfig := range s3ResourcesMap {
	//		if strings.Contains(arn, ThreeScaleNameElement) || strings.Contains(arn, S3Backup) {
	//			if !common.EqualResourceBool(
	//				[]bool{aws.BoolValue(expectedAccessBlockConfig.BlockPublicAcls), aws.BoolValue(expectedAccessBlockConfig.BlockPublicPolicy), aws.BoolValue(expectedAccessBlockConfig.IgnorePublicAcls), aws.BoolValue(expectedAccessBlockConfig.RestrictPublicBuckets)},
	//				[]bool{aws.BoolValue(accessBlockConfig.BlockPublicAcls), aws.BoolValue(accessBlockConfig.BlockPublicPolicy), aws.BoolValue(accessBlockConfig.IgnorePublicAcls), aws.BoolValue(accessBlockConfig.RestrictPublicBuckets)}) {
	//				t.Errorf("S3 Resources are not in the expected state, buckets are missing or accessBlockConfigs are incorrect")
	//			}
	//
	//		}
	//	}
	//}
}

// createAWSSession returns a new session from aws
func createAWSSession(awsAccessKeyID, awsSecretAccessKey, region string) (*session.Session, error) {
	session, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, ""),
		Region:      aws.String(region),
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create new session with aws : %w", err)
	}
	return session, nil
}

//getAWSRegion retrieves region from cluster infrastructure
func getAWSRegion(ctx context.Context, client dynclient.Client) (string, error) {
	infra := &configv1.Infrastructure{}
	err := client.Get(ctx, types.NamespacedName{Name: "cluster"}, infra)
	if err != nil {
		return "", fmt.Errorf("failed to get aws region : %w", err)
	}
	if infra.Status.PlatformStatus.Type != configv1.AWSPlatformType {
		return "", fmt.Errorf("platform status %s is not %s", infra.Status.PlatformStatus.Type, configv1.AWSPlatformType)
	}
	return infra.Status.PlatformStatus.AWS.Region, nil
}

//getClusterID retrieves cluster id from cluster infrastructure
func getClusterID(ctx context.Context, client dynclient.Client) (string, error) {
	infra := &configv1.Infrastructure{}
	if err := client.Get(ctx, types.NamespacedName{Name: "cluster"}, infra); err != nil {
		return "", fmt.Errorf("failed to retreive cluster infrastructure : %w", err)
	}
	return infra.Status.InfrastructureName, nil

}

//getAWSCredentials retrieves credentials from secret namespace
func getAWSCredentials(ctx context.Context, client dynclient.Client) (string, string, error) {
	secret := &corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{Name: awsCredsSecretName, Namespace: awsCredsNamespace}, secret); err != nil {
		return "", "", fmt.Errorf("failed getting secret: %v from cluster: %w ", awsCredsSecretName, err)
	}
	awsAccessKeyID := string(secret.Data["aws_access_key_id"])
	awsSecretAccessKey := string(secret.Data["aws_secret_access_key"])
	if awsAccessKeyID == "" && awsSecretAccessKey == "" {
		return "", "", errors.New("aws credentials secret can't be empty")
	}
	return awsAccessKeyID, awsSecretAccessKey, nil
}

func getRDSResources(client awsClients) ([]*rds.DBInstance, error) {
	clusterDescribeOutput, err := client.rdsClient.DescribeDBInstances(&rds.DescribeDBInstancesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to describe database clusters : %w", err)
	}
	return clusterDescribeOutput.DBInstances, nil
}

func getElasticacheResources(client awsClients, cID string) (map[string][]*elasticache.ReplicationGroup, error) {
	resourcesOutput, err := client.taggingClient.GetResources(&resourcegroupstaggingapi.GetResourcesInput{
		ResourceTypeFilters: aws.StringSlice([]string{"elasticache:cluster"}),
		TagFilters: []*resourcegroupstaggingapi.TagFilter{
			{
				Key:    aws.String(tagKeyClusterId),
				Values: aws.StringSlice([]string{cID}),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe resources output :%w", err)
	}
	elasticacheResources := make(map[string][]*elasticache.ReplicationGroup)
	for _, resourceTagMapping := range resourcesOutput.ResourceTagMappingList {
		arn := aws.StringValue(resourceTagMapping.ResourceARN)
		arnSplit := strings.Split(arn, ":")
		cacheClusterOutput, err := client.elasticacheClient.DescribeCacheClusters(&elasticache.DescribeCacheClustersInput{
			CacheClusterId: aws.String(arnSplit[len(arnSplit)-1]),
		})
		if err != nil {
			return nil, fmt.Errorf("failed get cacheCluster output :%w", err)
		}
		for _, cacheCluster := range cacheClusterOutput.CacheClusters {
			replicationGroups, err := client.elasticacheClient.DescribeReplicationGroups(&elasticache.DescribeReplicationGroupsInput{
				ReplicationGroupId: cacheCluster.ReplicationGroupId,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to describe replicationGroups :%w", err)
			}
			elasticacheResources[arn] = replicationGroups.ReplicationGroups
		}

	}

	return elasticacheResources, nil
}

func getS3Resources(client awsClients, cID string) (map[string]*s3control.PublicAccessBlockConfiguration, error) {
	getBucketOutput, err := client.taggingClient.GetResources(&resourcegroupstaggingapi.GetResourcesInput{
		ResourceTypeFilters: aws.StringSlice([]string{"s3"}),
		TagFilters: []*resourcegroupstaggingapi.TagFilter{
			{
				Key:    aws.String(tagKeyClusterId),
				Values: aws.StringSlice([]string{cID}),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets : %w", err)
	}
	s3Resources := make(map[string]*s3control.PublicAccessBlockConfiguration)
	for _, tagMappingList := range getBucketOutput.ResourceTagMappingList {
		arn := aws.StringValue(tagMappingList.ResourceARN)
		//get account id from arn, should be the second last element
		bucketARNElements := strings.Split(arn, ":")
		accountID := bucketARNElements[len(bucketARNElements)-2]
		accessBlockOutput, err := client.s3ControlClient.GetPublicAccessBlock(&s3control.GetPublicAccessBlockInput{AccountId: aws.String(accountID)})
		if err != nil {
			return nil, fmt.Errorf("failed get public access blocks for s3 bucket :%w", err)
		}
		s3Resources[arn] = accessBlockOutput.PublicAccessBlockConfiguration
	}

	return s3Resources, nil
}
