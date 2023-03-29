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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	stsSvc "github.com/aws/aws-sdk-go/service/sts"
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

func getAwsClusterSubnets(ec2svc *ec2.EC2, clusterID string) ([]*ec2.Subnet, error) {
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

func getAwsClusterVpc(ec2svc *ec2.EC2, clusterID string) (*ec2.Vpc, error) {
	clusterSubnets, err := getAwsClusterSubnets(ec2svc, clusterID)
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
