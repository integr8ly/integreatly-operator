package common

import (
	"context"

	"github.com/integr8ly/integreatly-operator/test/metadata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIntegreatlyCRDExists(t TestingTB, ctx *TestingContext) {
	testCrdExists(t, ctx, "rhmis.integreatly.org")
}

func TestRHMIConfigCRDExists(t TestingTB, ctx *TestingContext) {
	testCrdExists(t, ctx, "rhmiconfigs.integreatly.org")
}

func testCrdExists(t TestingTB, ctx *TestingContext, name string) {
	_, err := ctx.ExtensionClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
		metadata.Instance.FoundCRD = false
	} else {
		metadata.Instance.FoundCRD = true
	}
}
