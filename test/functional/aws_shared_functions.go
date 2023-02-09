package functional

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/integr8ly/integreatly-operator/pkg/resources/k8s"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/sts"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	stsSvc "github.com/aws/aws-sdk-go/service/sts"
	crov1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	croTypes "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/test/common"
	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	awsCredsNamespace       = "kube-system"
	awsCredsSecretName      = "aws-creds"
	awsManagedTagKey        = "red-hat-managed"
	awsManagedTagValue      = "true"
	awsClusterTypeKey       = "red-hat-clustertype"
	awsClusterTypeRosaValue = "rosa"
)

func getExpectedPostgres(installType string, installationName string) []string {
	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installType)) {
		// expected postgres resources provisioned per product
		return []string{
			fmt.Sprintf("%s%s", constants.ThreeScalePostgresPrefix, installationName),
			fmt.Sprintf("%s%s", constants.RHSSOPostgresPrefix, installationName),
		}
	} else {
		// expected postgres resources provisioned per product
		return []string{
			fmt.Sprintf("%s%s", constants.ThreeScalePostgresPrefix, installationName),
			fmt.Sprintf("%s%s", constants.RHSSOPostgresPrefix, installationName),
			fmt.Sprintf("%s%s", constants.RHSSOUserProstgresPrefix, installationName),
		}
	}
}

func getExpectedRedis(installType string, installationName string) []string {
	// expected redis resources provisioned per product
	commonRedis := []string{
		fmt.Sprintf("%s%s", constants.ThreeScaleBackendRedisPrefix, installationName),
		fmt.Sprintf("%s%s", constants.ThreeScaleSystemRedisPrefix, installationName),
		fmt.Sprintf("%s%s", constants.RateLimitRedisPrefix, installationName),
	}
	return commonRedis
}

func getExpectedBlobStorage(installType string, installationName string) []string {

	// 3scale blob storage
	threescaleBlobStorage := []string{
		fmt.Sprintf("%s%s", constants.ThreeScaleBlobStoragePrefix, installationName),
	}

	return threescaleBlobStorage
}

/*
Each resource provisioned contains an annotation with the resource ID
This function iterates over a list of expected resource CR's
Returns a list of resource ID's, these ID's can be used when testing AWS resources
*/
func GetElasticacheResourceIDs(ctx context.Context, client client.Client, rhmi *integreatlyv1alpha1.RHMI) ([]string, []string) {
	var foundErrors []string
	var foundResourceIDs []string

	expectedRedis := getExpectedRedis(rhmi.Spec.Type, rhmi.Name)

	for _, r := range expectedRedis {
		// get elasticache cr
		redis := &crov1.Redis{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: common.RHOAMOperatorNamespace, Name: r}, redis); err != nil {
			foundErrors = append(foundErrors, fmt.Sprintf("\nfailed to find %s redis cr : %v", r, err))
		}
		// ensure phase is completed
		if redis.Status.Phase != croTypes.PhaseComplete {
			foundErrors = append(foundErrors, fmt.Sprintf("\nfound %s redis not ready with phase: %s, message: %s", r, redis.Status.Phase, redis.Status.Message))
		}
		// return resource id
		resourceID, err := getCROAnnotation(redis)
		if err != nil {
			foundErrors = append(foundErrors, fmt.Sprintf("\n%s redis cr does not contain a resource id annotation: %v", r, err))
		}
		// populate the array
		foundResourceIDs = append(foundResourceIDs, resourceID)
	}
	return foundResourceIDs, foundErrors
}

/*
Each resource provisioned contains an annotation with the resource ID
This function iterates over a list of expected resource CR's
Returns a list of resource ID's, these ID's can be used when testing AWS resources
*/
func GetRDSResourceIDs(ctx context.Context, client client.Client, rhmi *integreatlyv1alpha1.RHMI) ([]string, []string) {
	var foundErrors []string
	var foundResourceIDs []string

	expectedPostgres := getExpectedPostgres(rhmi.Spec.Type, rhmi.Name)

	for _, r := range expectedPostgres {
		// get rds cr
		postgres := &crov1.Postgres{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: common.RHOAMOperatorNamespace, Name: r}, postgres); err != nil {
			foundErrors = append(foundErrors, fmt.Sprintf("\nfailed to find %s postgres cr : %v", r, err))
		}
		// ensure phase is completed
		if postgres.Status.Phase != croTypes.PhaseComplete {
			foundErrors = append(foundErrors, fmt.Sprintf("\nfound %s postgres not ready with phase: %s, message: %s", r, postgres.Status.Phase, postgres.Status.Message))
		}
		// return resource id
		resourceID, err := getCROAnnotation(postgres)
		if err != nil {
			foundErrors = append(foundErrors, fmt.Sprintf("\n%s postgres cr does not contain a resource id annotation: %v", r, err))
		}
		// populate the array
		foundResourceIDs = append(foundResourceIDs, resourceID)
	}
	return foundResourceIDs, foundErrors
}

func GetS3BlobStorageResourceIDs(ctx context.Context, client client.Client, rhmi *integreatlyv1alpha1.RHMI) ([]string, []string) {
	var foundErrors []string
	var foundResourceIDs []string

	expectedBlobStorage := getExpectedBlobStorage(rhmi.Spec.Type, rhmi.Name)

	for _, r := range expectedBlobStorage {
		// get rds cr
		blobStorage := &crov1.BlobStorage{}
		if err := client.Get(ctx, types.NamespacedName{Namespace: common.RHOAMOperatorNamespace, Name: r}, blobStorage); err != nil {
			foundErrors = append(foundErrors, fmt.Sprintf("\nfailed to find %s blobStorage cr : %v", r, err))
		}
		// ensure phase is completed
		if blobStorage.Status.Phase != croTypes.PhaseComplete {
			foundErrors = append(foundErrors, fmt.Sprintf("\nfound %s blobStorage not ready with phase: %s, message: %s", r, blobStorage.Status.Phase, blobStorage.Status.Message))
		}
		// return resource id
		resourceID, err := getCROAnnotation(blobStorage)
		if err != nil {
			foundErrors = append(foundErrors, fmt.Sprintf("\n%s blobStorage cr does not contain a resource id annotation: %v", r, err))
		}
		// populate the array
		foundResourceIDs = append(foundResourceIDs, resourceID)
	}

	return foundResourceIDs, foundErrors
}

// CreateAWSSession creates a session to be used in getting an api instance for aws
func CreateAWSSession(ctx context.Context, client client.Client) (*session.Session, bool, error) {
	region, err := getAWSRegion(ctx, client)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get AWS cluster region : %w", err)
	}
	awsConfig := aws.Config{
		Region: aws.String(region),
	}
	isSTS, err := sts.IsClusterSTS(ctx, client, logger.NewLogger())
	if err != nil {
		return nil, false, err
	}
	if isSTS {
		roleARN, tokenPath, err := sts.GetSTSCredentialsFromEnvVar()
		if err != nil {
			return nil, isSTS, fmt.Errorf("failed to get STS credentials: %w", err)
		}
		if k8s.IsRunLocally() {
			sess := session.Must(session.NewSession(&awsConfig))
			awsConfig.Credentials = stscreds.NewCredentials(sess, roleARN)
		} else {
			svc := stsSvc.New(session.Must(session.NewSession(&awsConfig)))
			credentialsProvider := stscreds.NewWebIdentityRoleProviderWithOptions(svc, roleARN, sts.RoleSessionName, stscreds.FetchTokenPath(tokenPath))
			awsConfig.Credentials = credentials.NewCredentials(credentialsProvider)
		}
	} else {
		awsAccessKeyID, awsSecretAccessKey, err := getAWSCredentials(ctx, client)
		if err != nil {
			return nil, isSTS, fmt.Errorf("failed to get AWS credentials: %w", err)
		}
		awsConfig.Credentials = credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, "")
	}
	sess := session.Must(session.NewSession(&awsConfig))
	return sess, isSTS, nil
}

// getAWSRegion retrieves region from cluster infrastructure
func getAWSRegion(ctx context.Context, client client.Client) (string, error) {
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

// getAWSCredentials retrieves aws credentials from secret namespace
func getAWSCredentials(ctx context.Context, client client.Client) (string, string, error) {
	secret := &corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{Name: awsCredsSecretName, Namespace: awsCredsNamespace}, secret); err != nil {
		return "", "", fmt.Errorf("failed getting secret %s from ns %s: %w", awsCredsSecretName, awsCredsNamespace, err)
	}
	awsAccessKeyID := string(secret.Data["aws_access_key_id"])
	awsSecretAccessKey := string(secret.Data["aws_secret_access_key"])
	if awsAccessKeyID == "" && awsSecretAccessKey == "" {
		return "", "", errors.New("aws credentials secret can't be empty")
	}
	return awsAccessKeyID, awsSecretAccessKey, nil
}

// return resource identifier annotation from cr
func getCROAnnotation(instance metav1.Object) (string, error) {
	annotations := instance.GetAnnotations()
	if annotations == nil {
		return "", errors.New(fmt.Sprintf("annotations for %s can not be nil", instance.GetName()))
	}

	for k, v := range annotations {
		if "resourceIdentifier" == k {
			return v, nil
		}
	}
	return "", errors.New(fmt.Sprintf("no resource identifier found for resource %s", instance.GetName()))
}

func getStrategyForResource(configMap *v1.ConfigMap, resourceType, tier string) (*strategyMap, error) {
	rawStrategyMapping := configMap.Data[resourceType]
	if rawStrategyMapping == "" {
		return nil, fmt.Errorf("aws strategy for resource type: %s is not defined", resourceType)
	}
	var strategyMapping map[string]*strategyMap
	if err := json.Unmarshal([]byte(rawStrategyMapping), &strategyMapping); err != nil {
		return nil, fmt.Errorf("failed to unmarshal strategy mapping for resource type %s: %v", resourceType, err)
	}
	if strategyMapping[tier] == nil {
		return nil, fmt.Errorf("no strategy found for deployment type: %s and deployment tier: %s", resourceType, tier)
	}
	return strategyMapping[tier], nil
}

// GetClustersAvailableZones returns a map containing zone names that are currently available
func GetClustersAvailableZones(nodes *v1.NodeList) map[string]bool {
	zones := make(map[string]bool)
	for _, node := range nodes.Items {
		if isNodeWorkerAndReady(node) {
			for labelName, labelValue := range node.Labels {
				if labelName == "topology.kubernetes.io/zone" {
					zones[labelValue] = true
				}
			}
		}
	}
	return zones
}

func elasticacheTagsContains(tags []*elasticache.Tag, key, value string) bool {
	for _, tag := range tags {
		if *tag.Key == key && *tag.Value == value {
			return true
		}
	}
	return false
}

func rdsTagsContains(tags []*rds.Tag, key, value string) bool {
	for _, tag := range tags {
		if *tag.Key == key && *tag.Value == value {
			return true
		}
	}
	return false
}

func s3TagsContains(tags []*s3.Tag, key, value string) bool {
	for _, tag := range tags {
		if *tag.Key == key && *tag.Value == value {
			return true
		}
	}
	return false
}

func ec2TagsContains(tags []*ec2.Tag, key, value string) bool {
	for _, tag := range tags {
		if *tag.Key == key && *tag.Value == value {
			return true
		}
	}
	return false
}

func describeSubnets(ec2svc *ec2.EC2, input *ec2.DescribeSubnetsInput) ([]*ec2.Subnet, error) {
	describeSubnetsOutput, err := ec2svc.DescribeSubnets(input)
	if err != nil {
		return nil, fmt.Errorf("could not describe subnets: %w", err)
	}
	if len(describeSubnetsOutput.Subnets) == 0 {
		return nil, fmt.Errorf("could not find any subnets")
	}
	return describeSubnetsOutput.Subnets, nil
}

func getClusterSubnets(ec2svc *ec2.EC2, clusterID string) ([]*ec2.Subnet, error) {
	subnets, err := describeSubnets(ec2svc, &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("tag:" + clusterResourceTagKeyPrefix + clusterID),
				Values: []*string{
					aws.String(clusterSharedTagValue),
					aws.String(clusterOwnedTagValue),
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not fetch cluster subnets: %v", err)
	}
	return subnets, nil
}

func getStandaloneSubnets(ec2svc *ec2.EC2, clusterID string) ([]*ec2.Subnet, error) {
	subnets, err := describeSubnets(ec2svc, &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:" + standaloneResourceTagKey),
				Values: []*string{&clusterID},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not fetch standalone subnets: %v", err)
	}
	return subnets, nil
}

func describeVpcs(ec2svc *ec2.EC2, input *ec2.DescribeVpcsInput) ([]*ec2.Vpc, error) {
	describeVpcsOutput, err := ec2svc.DescribeVpcs(input)
	if err != nil {
		return nil, fmt.Errorf("could not describe vpcs: %w", err)
	}
	if len(describeVpcsOutput.Vpcs) == 0 {
		return nil, fmt.Errorf("could not find any vpcs")
	}
	return describeVpcsOutput.Vpcs, nil
}

func getClusterVpc(ec2svc *ec2.EC2, clusterID string) (*ec2.Vpc, error) {
	clusterSubnets, err := getClusterSubnets(ec2svc, clusterID)
	if err != nil {
		return nil, fmt.Errorf("could not fetch cluster vpc: %w", err)
	}
	clusterTagKey := fmt.Sprintf("%s%s", clusterResourceTagKeyPrefix, clusterID)
	var vpcID *string
	for _, subnet := range clusterSubnets {
		for _, tag := range subnet.Tags {
			if tag != nil && *tag.Key == clusterTagKey && (*tag.Value == clusterOwnedTagValue || *tag.Value == clusterSharedTagValue) {
				vpcID = subnet.VpcId
				break
			}
		}
	}
	if vpcID == nil {
		return nil, fmt.Errorf("could not fetch cluster vpc: no subnet tags matched key %s with value %s or %s", clusterTagKey, clusterOwnedTagValue, clusterSharedTagValue)
	}
	vpcs, err := describeVpcs(ec2svc, &ec2.DescribeVpcsInput{VpcIds: []*string{vpcID}})
	if err != nil {
		return nil, fmt.Errorf("could not fetch cluster vpc: %w", err)
	}
	if len(vpcs) > 1 {
		return nil, fmt.Errorf("found more than one vpc associated with cluster subnets")
	}
	return vpcs[0], nil
}

func getStandaloneVpc(ec2svc *ec2.EC2, clusterID string) (*ec2.Vpc, error) {
	vpcs, err := describeVpcs(ec2svc, &ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:" + standaloneResourceTagKey),
				Values: []*string{&clusterID},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not fetch standalone vpc: %w", err)
	}
	if len(vpcs) > 1 {
		return nil, fmt.Errorf("found more than one vpc associated with tag %s", standaloneResourceTagKey)
	}
	return vpcs[0], nil
}

func describeRouteTables(ec2svc *ec2.EC2, input *ec2.DescribeRouteTablesInput) ([]*ec2.RouteTable, error) {
	describeRouteTablesOutput, err := ec2svc.DescribeRouteTables(input)
	if err != nil {
		return nil, fmt.Errorf("could not describe route tables: %w", err)
	}
	routeTables := describeRouteTablesOutput.RouteTables
	if len(routeTables) == 0 {
		return nil, fmt.Errorf("could not find any route tables")
	}
	return routeTables, nil
}

func getClusterRouteTables(ec2svc *ec2.EC2, vpcId *string, subnets []*ec2.Subnet) ([]*ec2.RouteTable, error) {
	routeTables, err := describeRouteTables(ec2svc, &ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{vpcId},
			},
			{
				Name: aws.String("association.subnet-id"),
				Values: func() (subnetIds []*string) {
					for i := range subnets {
						subnetIds = append(subnetIds, subnets[i].SubnetId)
					}
					return
				}(),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not fetch cluster route tables: %w", err)
	}
	return routeTables, nil
}

func getStandaloneRouteTables(ec2svc *ec2.EC2, vpcId *string) ([]*ec2.RouteTable, error) {
	routeTables, err := describeRouteTables(ec2svc, &ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{vpcId},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not fetch standalone route tables: %w", err)
	}
	return routeTables, nil
}
