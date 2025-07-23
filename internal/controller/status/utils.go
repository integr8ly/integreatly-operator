package status

import (
	"context"

	"github.com/integr8ly/integreatly-operator/pkg/resources/k8s"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	addonInstanceCRDName = "addoninstances.addons.managed.openshift.io"
)

func IsAddonOperatorInstalled(client client.Client) (bool, error) {
	addonInstanceCRD := &apiextensionv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: addonInstanceCRDName,
		},
	}

	return k8s.Exists(context.Background(), client, addonInstanceCRD)
}
