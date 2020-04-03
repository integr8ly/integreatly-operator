package resources

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/sirupsen/logrus"
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
func ReconcileSecretToProductNamespace(ctx context.Context, client k8sclient.Client, configManager config.ConfigReadWriter, secretName string, namespace string) (integreatlyv1alpha1.StatusPhase, error) {
	err := CopySecret(ctx, client, secretName, configManager.GetOperatorNamespace(), secretName, namespace)

	if err != nil {
		// Secret may not initially exist - log warning without blocking a reconcile
		if k8serr.IsNotFound(err) {
			logrus.Warnf("Could not find %s secret in RHMI operator namespace to copy to %s", secretName, namespace)
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

func LinkSecretToServiceAccounts(ctx context.Context, client k8sclient.Client, namespace string, secretName string) error {
	serviceAccounts := &corev1.ServiceAccountList{}
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(namespace),
	}
	err := client.List(ctx, serviceAccounts, listOpts...)
	if err != nil {
		return err
	}

	for _, sa := range serviceAccounts.Items {
		currentSa := &corev1.ServiceAccount{}
		err = client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: sa.Name}, currentSa)
		if err != nil {
			return err
		}

		pullSecretFound := false
		for _, ips := range currentSa.ImagePullSecrets {
			if ips.Name == secretName {
				pullSecretFound = true
				break
			}

		}

		if !pullSecretFound {
			newPullSecret := corev1.LocalObjectReference{Name: secretName}
			_, err = controllerutil.CreateOrUpdate(ctx, client, currentSa, func() error {
				currentSa.ImagePullSecrets = append(currentSa.ImagePullSecrets, newPullSecret)
				return nil
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
