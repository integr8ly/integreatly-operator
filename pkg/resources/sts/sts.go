package sts

import (
	"context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/addon"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	cloudcredentialv1 "github.com/openshift/api/operator/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"regexp"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// #nosec G101 -- This is a false positive
const (
	ClusterCloudCredentialName  = "cluster"
	RoleArnParameterName        = "sts-role-arn"
	RoleSessionName             = "Red-Hat-cloud-resources-operator"
	CredsSecretName             = "sts-credentials"
	CredsSecretRoleARNKeyName   = "role_arn"
	CredsSecretTokenPathKeyName = "web_identity_token_file"
	CredsRoleEnvKey             = "ROLE_ARN"
	CredsTokenPathEnvKey        = "TOKEN_PATH"
	CredsS3AccessKeyId          = "s3-access-key-id"
	CredsS3SecretAccessKey      = "s3-secret-access-key"
)

func IsClusterSTS(ctx context.Context, client k8sclient.Client, log logger.Logger) (bool, error) {
	cloudCredential := &cloudcredentialv1.CloudCredential{}
	if err := client.Get(ctx, k8sclient.ObjectKey{Name: ClusterCloudCredentialName}, cloudCredential); err != nil {
		log.Error("failed to get cloudCredential while checking if STS mode", err)
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
	stsRoleArn, stsFound, err := addon.GetStringParameter(
		ctx,
		client,
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

// ValidateAddOnStsRoleArnParameterPattern is checking if STS addon parameter Pattern is valid
// Parameter is Valid only in case:
// 1.	Parameter exists and value matching AWS Role ARN pattern
// Parameter is Not valid  in other cases:
// 2.	parameter exists and value is NOT matching AWS Role ARN pattern
// 3.	parameter exists and value is empty
// 4.	parameter does not exist
func ValidateAddOnStsRoleArnParameterPattern(client k8sclient.Client, namespace string) (bool, error) {
	stsRoleArn, err := GetSTSRoleARN(context.TODO(), client, namespace)
	if err != nil {
		return false, fmt.Errorf("failed while retrieving addon parameter: %v", err)
	}

	awsArnPattern := "arn:aws(?:-us-gov)?:iam:\\S*:\\d+:role\\/\\S+"
	r, err := regexp.Compile(awsArnPattern)
	if err != nil {
		return false, fmt.Errorf("regexp Compile error: %v", err)
	}

	// Not a regex match
	if !r.MatchString(stsRoleArn) {
		return false, fmt.Errorf("AWS STS role ARN parameter validation failed - parameter pattern is not matching to AWS ARN standard")
	}

	return true, nil
}

// CreateSTSARNSecret create the STS arn secret - should be already validated in preflight checks
func CreateSTSARNSecret(ctx context.Context, client k8sclient.Client, installationNamespace, operatorNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	stsRoleArn, err := GetSTSRoleARN(ctx, client, installationNamespace)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("STS role ARN parameter pattern validation failed: %w", err)
	}

	// create CRO credentials secret
	credSec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CredsSecretName,
			Namespace: operatorNamespace,
		},
		Data: map[string][]byte{},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, client, credSec, func() error {
		credSec.Data[CredsSecretRoleARNKeyName] = []byte(stsRoleArn)
		credSec.Data[CredsSecretTokenPathKeyName] = []byte("/var/run/secrets/openshift/serviceaccount/token")
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to update CRO credentials Secret. Failed to pass ARN into secret: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
