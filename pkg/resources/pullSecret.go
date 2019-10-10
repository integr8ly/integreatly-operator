package resources

import (
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Gets the default pull secret for pulling container images from registry
func GetDefaultPullSecret(client pkgclient.Client, context context.Context) (corev1.Secret, error) {
	openshiftSecret := corev1.Secret{}

	err := client.Get(context, types.NamespacedName{Name: DefaultOriginPullSecretName, Namespace: DefaultOriginPullSecretNamespace}, &openshiftSecret)

	return openshiftSecret, err
}

// Copys the default pull secret to a target namespace
func CopyDefaultPullSecretToNameSpace(nameSpaceToCopy string, nameOfSecret string, client pkgclient.Client, context context.Context) error {
	openshiftSecret, err := GetDefaultPullSecret(client, context)

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
