package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	awsCredsNamespace  = "redhat-rhmi-operator"
	awsCredsSecretName = "cloud-resources-aws-credentials"
)

func CreateAWSSession(ctx context.Context, client dynclient.Client) (*session.Session, error) {
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
