package resources

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultOriginPullSecretName      = "samples-registry-credentials"
	DefaultOriginPullSecretNamespace = "openshift"
)

// CopyDefaultPullSecretToNamespace copies the default pull secret to a target namespace
func CopyDefaultPullSecretToNameSpace(context context.Context, destNamespace, destName string, inst *integreatlyv1alpha1.RHMI, client k8sclient.Client) error {
	if inst.Spec.PullSecret.Name == "" {
		inst.Spec.PullSecret.Name = DefaultOriginPullSecretName
	}
	if inst.Spec.PullSecret.Namespace == "" {
		inst.Spec.PullSecret.Namespace = DefaultOriginPullSecretNamespace
	}

	return CopySecret(context, client, inst.Spec.PullSecret.Name, inst.Spec.PullSecret.Namespace, destName, destNamespace)
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
