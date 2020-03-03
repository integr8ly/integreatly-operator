package common

import (
	"testing"

	"github.com/integr8ly/integreatly-operator/test/metadata"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIntegreatlyCRDExists(t *testing.T, ctx *TestingContext) {
	_, err := ctx.ExtensionClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get("rhmis.integreatly.org", v1.GetOptions{})
	if err != nil {
		t.Fatal(err)
		metadata.Instance.FoundCRD = false
	} else {
		metadata.Instance.FoundCRD = true
	}
}
