package resources

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	k8serr "k8s.io/apimachinery/pkg/api/errors"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// CopyPullSecretToNamespace copies the default pull secret to a target namespace
func CopyPullSecretToNameSpace(context context.Context, secretSpec *integreatlyv1alpha1.PullSecretSpec, destNamespace, destName string, client k8sclient.Client) error {
	return CopySecret(context, client, secretSpec.Name, secretSpec.Namespace, destName, destNamespace)
}

//CopySecret will copy or update the destination secret from the source secret
func CopySecret(ctx context.Context, client k8sclient.Client, srcName, srcNamespace, destName, destNamespace string) error {
	srcSecret := corev1.Secret{}
	err := client.Get(ctx, types.NamespacedName{Name: srcName, Namespace: srcNamespace}, &srcSecret)
	if err != nil {
		return err
	}

	destSecret := &corev1.Secret{
		Type: corev1.SecretTypeDockerConfigJson,
		ObjectMeta: metav1.ObjectMeta{
			Name:      destName,
			Namespace: destNamespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, client, destSecret, func() error {
		destSecret.Data = srcSecret.Data
		destSecret.Type = srcSecret.Type
		return nil
	})

	return err
}

// Copies a secret from the RHMI operator namespace to a target namespace
func ReconcileSecretToProductNamespace(ctx context.Context, client k8sclient.Client, configManager config.ConfigReadWriter, secretName string, namespace string, log l.Logger) (integreatlyv1alpha1.StatusPhase, error) {
	err := CopySecret(ctx, client, secretName, configManager.GetOperatorNamespace(), secretName, namespace)

	if err != nil {
		// Secret may not initially exist - log warning without blocking a reconcile
		if k8serr.IsNotFound(err) {
			log.Warningf("Could not find secret in operator namespace to copy", l.Fields{"secretName": secretName, "ns": namespace})
			return integreatlyv1alpha1.PhaseCompleted, nil
		}
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to seed copy secret to product namespace: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

// Copies a secret from a target namespace to the RHMI operator namespace
func ReconcileSecretToRHMIOperatorNamespace(ctx context.Context, client k8sclient.Client, configManager config.ConfigReadWriter, secretName string, namespace string) (integreatlyv1alpha1.StatusPhase, error) {
	err := CopySecret(ctx, client, secretName, namespace, secretName, configManager.GetOperatorNamespace())

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to copy secret to RHMI Operator namespace: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
