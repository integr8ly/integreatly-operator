package functional

import (
	"context"
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
	"github.com/integr8ly/integreatly-operator/test/common"
	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
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
			credentialsProvider := stscreds.NewWebIdentityRoleProvider(svc, roleARN, sts.RoleSessionName, tokenPath)
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
