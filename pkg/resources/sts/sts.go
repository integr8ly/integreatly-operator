package sts

import (
	"context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/addon"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	cloudcredentialv1 "github.com/openshift/api/operator/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterCloudCredentialName  = "cluster"
	RoleArnParameterName        = "sts-role-arn"
	RoleSessionName             = "Red-Hat-cloud-resources-operator"
	CredsSecretName             = "sts-credentials"
	CredsSecretRoleARNKeyName   = "role_arn"
	CredsSecretTokenPathKeyName = "web_identity_token_file"
	CredsRoleEnvKey             = "ROLE_ARN"
	CredsTokenPathEnvKey        = "TOKEN_PATH"
	CredsS3BucketUsr			= "s3BucketCredentialUsr"
	CredsS3BucketPsw			= "s3BucketCredentialPsw"
)

func IsClusterSTS(ctx context.Context, client k8sclient.Client, log logger.Logger) (bool, error) {
	cloudCredential := &cloudcredentialv1.CloudCredential{}
	if err := client.Get(ctx, k8sclient.ObjectKey{Name: clusterCloudCredentialName}, cloudCredential); err != nil {
		log.Error("failed to get cloudCredential whle checking if STS mode", err)
		return false, err
	}

	if cloudCredential.Spec.CredentialsMode == cloudcredentialv1.CloudCredentialsModeManual {
		log.Info("STS mode")
		return true, nil
	}
	log.Info("non STS mode")
	return false, nil
}

// GetSTSRoleARN retrieves the role ARN addon parameter to be used by CRO
func GetSTSRoleARN(ctx context.Context, client k8sclient.Client, namespace string) (string, error) {
	stsRoleArn, stsFound, err := addon.GetStringParameterByInstallType(
		ctx,
		client,
		integreatlyv1alpha1.InstallationTypeManagedApi,
		namespace,
		RoleArnParameterName,
	)
	if err != nil {
		return "", fmt.Errorf("failed while retrieving addon parameter %w", err)
	}
	if !stsFound || stsRoleArn == "" {
		return "", fmt.Errorf("no STS configuration found")
	}

	return stsRoleArn, nil
}

// GetSTSCredentials retrieves the STS secret used by CRO
func GetSTSCredentials(ctx context.Context, client k8sclient.Client, namespace string) (string, string, error) {
	secret := &corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{Name: CredsSecretName, Namespace: namespace}, secret); err != nil {
		return "", "", fmt.Errorf("failed getting secret %s from ns %s: %w", CredsSecretName, namespace, err)
	}
	roleARN := string(secret.Data[CredsSecretRoleARNKeyName])
	tokenPath := string(secret.Data[CredsSecretTokenPathKeyName])
	if roleARN == "" || tokenPath == "" {
		return "", "", fmt.Errorf("sts credentials secret can't be empty")
	}
	return roleARN, tokenPath, nil
}

// GetSTSCredentialsFromEnvVar Gets the role arn and token file path from environment variable
// Should only be used in functional test container
func GetSTSCredentialsFromEnvVar() (string, string, error) {
	roleARN, found := os.LookupEnv(CredsRoleEnvKey)
	if !found {
		return "", "", fmt.Errorf("%s key should not be empty", CredsRoleEnvKey)
	}

	tokenPath, found := os.LookupEnv(CredsTokenPathEnvKey)
	if !found {
		return "", "", fmt.Errorf("%s key should not be empty", CredsRoleEnvKey)
	}

	return roleARN, tokenPath, nil
}

// GetS3BucketCredentials retrieves Credential addon parameters to be used by 3scale
// Using S3 Bucket Credentiol addon parameters is temporary solution,
// workaround for the lack of STS support in 3Scale,
// as noted in https://issues.redhat.com/browse/MGDAPI-1905,
// until our feature request https://issues.redhat.com/browse/THREESCALE-7132 gets implemented
func GetS3BucketCredentials(ctx context.Context, client k8sclient.Client, namespace string) (string, string, error) {
	s3BucketUsr, stsFound, err := addon.GetStringParameterByInstallType(
		ctx,
		client,
		integreatlyv1alpha1.InstallationTypeManagedApi,
		namespace,
		CredsS3BucketUsr,
	)
	if err != nil {
		return "", "", fmt.Errorf("failed while retrieving addon parameter %w", err)
	}
	if !stsFound || CredsS3BucketUsr == "" {
		return "", "", fmt.Errorf("no S3 Bucket Credential addon parameter found")
	}

	s3BucketPsw, stsFound, err := addon.GetStringParameterByInstallType(
		ctx,
		client,
		integreatlyv1alpha1.InstallationTypeManagedApi,
		namespace,
		CredsS3BucketPsw,
	)
	if err != nil {
		return "", "", fmt.Errorf("failed while retrieving addon parameter %w", err)
	}
	if !stsFound || CredsS3BucketUsr == "" {
		return "", "", fmt.Errorf("no S3 Bucket Credential addon parameter found")
	}

	return s3BucketUsr, s3BucketPsw, nil
}