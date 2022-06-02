package aws

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/integr8ly/cloud-resource-operator/internal/k8sutil"
	errorUtil "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

const (
	defaultSTSCredentialSecretName = "sts-credentials"
	defaultRoleARNKeyName          = "role_arn"
	defaultTokenPathKeyName        = "web_identity_token_file"
)

var _ CredentialManager = (*STSCredentialManager)(nil)

// STSCredentialManager Implementation of CredentialManager for OpenShift Clusters that use STS
type STSCredentialManager struct {
	OperatorNamespace string
	Client            client.Client
}

func NewSTSCredentialManager(client client.Client) *STSCredentialManager {
	ns, _ := k8sutil.GetOperatorNamespace()
	return &STSCredentialManager{
		OperatorNamespace: ns,
		Client:            client,
	}
}

//ReconcileProviderCredentials Ensure the credentials the AWS provider requires are available
func (m *STSCredentialManager) ReconcileProviderCredentials(ctx context.Context, _ string) (*Credentials, error) {
	secret, err := getSTSCredentialsSecret(ctx, m.Client, m.OperatorNamespace)
	if err != nil {
		return nil, errorUtil.Wrapf(err, "failed to get aws sts credentials secret %s", defaultSTSCredentialSecretName)
	}
	credentials := &Credentials{
		RoleArn:       string(secret.Data[defaultRoleARNKeyName]),
		TokenFilePath: string(secret.Data[defaultTokenPathKeyName]),
	}
	if credentials.RoleArn == "" {
		return nil, errorUtil.New(fmt.Sprintf("%s key is undefined in secret %s", defaultRoleARNKeyName, secret.Name))
	}
	if credentials.TokenFilePath == "" {
		return nil, errorUtil.New(fmt.Sprintf("%s key is undefined in secret %s", defaultTokenPathKeyName, secret.Name))
	}
	return credentials, nil
}

func (m *STSCredentialManager) ReconcileBucketOwnerCredentials(_ context.Context, _, _, _ string) (*Credentials, error) {
	return nil, nil
}

func getSTSCredentialsSecret(ctx context.Context, client client.Client, ns string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := client.Get(ctx, types.NamespacedName{Name: defaultSTSCredentialSecretName, Namespace: ns}, secret)
	return secret, err
}
