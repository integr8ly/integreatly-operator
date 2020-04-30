package common

import (
	"testing"

	"github.com/integr8ly/integreatly-operator/test/metadata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIntegreatlyCRDExists(t *testing.T, ctx *TestingContext) {
	testCrdExists(t, ctx, "rhmis.integreatly.org")
}

func TestRHMIConfigCRDExists(t *testing.T, ctx *TestingContext) {
	testCrdExists(t, ctx, "rhmiconfigs.integreatly.org")
}

func testCrdExists(t *testing.T, ctx *TestingContext, name string) {
	_, err := ctx.ExtensionClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
		metadata.Instance.FoundCRD = false
	} else {
		metadata.Instance.FoundCRD = true
	}
}
