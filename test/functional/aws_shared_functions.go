package functional

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	croTypes "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
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
	awsCredsNamespace  = "kube-system"
	awsCredsSecretName = "aws-creds"
)

func getExpectedPostgres(installType string, installationName string) []string {
	if installType == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		// expected postgres resources provisioned per product
		return []string{
			fmt.Sprintf("%s%s", constants.ThreeScalePostgresPrefix, installationName),
			fmt.Sprintf("%s%s", constants.RHSSOPostgresPrefix, installationName),
			fmt.Sprintf("%s%s", constants.RHSSOUserProstgresPrefix, installationName),
		}
	} else {
		// expected postgres resources provisioned per product
		return []string{
			fmt.Sprintf("%s%s", constants.CodeReadyPostgresPrefix, installationName),
			fmt.Sprintf("%s%s", constants.ThreeScalePostgresPrefix, installationName),
			fmt.Sprintf("%s%s", constants.RHSSOPostgresPrefix, installationName),
			fmt.Sprintf("%s%s", constants.RHSSOUserProstgresPrefix, installationName),
			fmt.Sprintf("%s%s", constants.UPSPostgresPrefix, installationName),
			fmt.Sprintf("%s%s", constants.FusePostgresPrefix, installationName),
		}
	}
}

func getExpectedRedis(installType string, installationName string) []string {
	// expected redis resources provisioned per product
	commonRedis := []string{
		fmt.Sprintf("%s%s", constants.ThreeScaleBackendRedisPrefix, installationName),
		fmt.Sprintf("%s%s", constants.ThreeScaleSystemRedisPrefix, installationName),
	}

	managedApiRedis := []string{
		fmt.Sprintf("%s%s", constants.RateLimitRedisPrefix, installationName),
	}

	if installType == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		return append(commonRedis, managedApiRedis...)
	} else {
		return commonRedis
	}
}

func getExpectedBlobStorage(installType string, installationName string) []string {
	// common blob storage
	commonBlobStorage := []string{
		fmt.Sprintf("%s%s", constants.ThreeScaleBlobStoragePrefix, installationName),
	}

	// rhmi blob storage
	rhmiBlobStorage := []string{
		fmt.Sprintf("%s%s", constants.ThreeScaleBlobStoragePrefix, installationName),
	}

	if installType == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		return commonBlobStorage
	} else {
		return append(commonBlobStorage, rhmiBlobStorage...)
	}
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
		if err := client.Get(ctx, types.NamespacedName{Namespace: common.RHMIOperatorNamespace, Name: r}, redis); err != nil {
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
		if err := client.Get(ctx, types.NamespacedName{Namespace: common.RHMIOperatorNamespace, Name: r}, postgres); err != nil {
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
		if err := client.Get(ctx, types.NamespacedName{Namespace: common.RHMIOperatorNamespace, Name: r}, blobStorage); err != nil {
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

// creates a session to be used in getting an api instance for aws
func CreateAWSSession(ctx context.Context, client client.Client) (*session.Session, error) {
	//retrieve aws credentials for creating an aws session
	awsSecretAccessKey, awsAccessKeyID, err := getAWSCredentials(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS credentials : %w", err)
	}

	//retrieve aws region for creating an aws session
	region, err := getAWSRegion(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS cluster region : %w", err)
	}

	//create new session for aws api's
	sess, err := createAWSSession(awsSecretAccessKey, awsAccessKeyID, region)
	if err != nil {
		return nil, fmt.Errorf("failed to create session : %w", err)
	}
	return sess, nil
}

// createAWSSession returns a new session from aws
func createAWSSession(awsAccessKeyID, awsSecretAccessKey, region string) (*session.Session, error) {
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, ""),
		Region:      aws.String(region),
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create new session with aws : %w", err)
	}
	return sess, nil
}

//getAWSRegion retrieves region from cluster infrastructure
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

//getAWSCredentials retrieves credentials from secret namespace
func getAWSCredentials(ctx context.Context, client client.Client) (string, string, error) {
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

func putStrategyForResource(configMap *v1.ConfigMap, stratMap *strategyMap, resourceType, tier string) error {
	rawStrategyMapping := configMap.Data[resourceType]
	if rawStrategyMapping == "" {
		return fmt.Errorf("aws strategy for resource type: %s is not defined", resourceType)
	}
	var strategyMapping map[string]*strategyMap
	if err := json.Unmarshal([]byte(rawStrategyMapping), &strategyMapping); err != nil {
		return fmt.Errorf("failed to unmarshal strategy mapping for resource type %s: %v", resourceType, err)
	}
	strategyMapping[tier] = stratMap
	updatedRawStrategyMapping, err := json.Marshal(strategyMapping)
	if err != nil {
		return fmt.Errorf("failed to marshal strategy mapping for resource type %s: %v", resourceType, err)
	}
	configMap.Data[resourceType] = string(updatedRawStrategyMapping)
	return nil
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

// getVpcCidrBlock returns a cidr block using a key/value tag pairing
func getVpcCidrBlock(session *ec2.EC2, clusterTagName, clusterTagValue string) (string, error) {
	describeVpcs, err := session.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String(clusterTagName),
				Values: []*string{aws.String(clusterTagValue)},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("could not find vpc: %v", err)
	}

	// only one vpc is expected
	vpcs := describeVpcs.Vpcs
	if len(vpcs) != 1 {
		return "", fmt.Errorf("expected 1 vpc but found %d", len(vpcs))
	}

	return aws.StringValue(vpcs[0].CidrBlock), nil
}
