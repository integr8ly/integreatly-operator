package functional

//some functions below were taken from CRO
import (
	"context"
	"errors"
	"fmt"

	"github.com/integr8ly/integreatly-operator/test/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultNumberOfExpectedSubnets = 2
	managedLabelKey                = "red-hat-managed"
	managedLabelValue              = "true"
	gcpCredsSecretName             = "cloud-resource-gcp-credentials"
)

func getGCPCredentials(ctx context.Context, client client.Client) ([]byte, error) {
	secret := &corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{Name: gcpCredsSecretName, Namespace: common.RHOAMOperatorNamespace}, secret); err != nil {
		return nil, fmt.Errorf("failed getting secret %s from ns %s: %w", gcpCredsSecretName, common.RHOAMOperatorNamespace, err)
	}
	serviceAccountJson := secret.Data["service_account.json"]
	if len(serviceAccountJson) == 0 {
		return nil, errors.New("gcp credentials secret can't be empty")
	}
	return serviceAccountJson, nil
}

func labelsContain(labels map[string]string, key, value string) bool {
	for k, v := range labels {
		if k == key && v == value {
			return true
		}
	}
	return false
}
