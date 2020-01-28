package resources

import (
	"context"

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

// Gets the default pull secret for pulling container images from registry
func GetDefaultPullSecret(context context.Context, client k8sclient.Client, inst *integreatlyv1alpha1.Installation) (corev1.Secret, error) {
	if inst.Spec.PullSecret.Name == "" {
		inst.Spec.PullSecret.Name = DefaultOriginPullSecretName
	}
	if inst.Spec.PullSecret.Namespace == "" {
		inst.Spec.PullSecret.Namespace = DefaultOriginPullSecretNamespace
	}

	openshiftSecret := corev1.Secret{}

	err := client.Get(context, types.NamespacedName{Name: inst.Spec.PullSecret.Name, Namespace: inst.Spec.PullSecret.Namespace}, &openshiftSecret)

	return openshiftSecret, err
}

// Copys the default pull secret to a target namespace
func CopyDefaultPullSecretToNameSpace(context context.Context, nameSpaceToCopy string, inst *integreatlyv1alpha1.Installation, client k8sclient.Client, nameOfSecret string) error {
	openshiftSecret, err := GetDefaultPullSecret(context, client, inst)

	if err != nil {
		return err
	}

	componentSecret := &corev1.Secret{
		Type: corev1.SecretTypeDockerConfigJson,
		ObjectMeta: metav1.ObjectMeta{
			Name:      nameOfSecret,
			Namespace: nameSpaceToCopy,
		},
		Data: openshiftSecret.Data,
	}

	err = CreateOrUpdate(context, client, componentSecret)

	return err
}
