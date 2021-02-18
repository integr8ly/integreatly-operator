package common

import (
	"context"

	apicurioregistry "github.com/Apicurio/apicurio-registry-operator/pkg/apis/apicur/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
	"k8s.io/apimachinery/pkg/types"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	artifactID   = "share-price"
	artifactType = resources.Avro
	artifactData = "{\"type\":\"record\",\"name\":\"price\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"symbol\",\"type\":\"string\"},{\"name\":\"price\",\"type\":\"string\"}]}"
)

// TestApicurioRegistryAPI tests the ApicurioRegistry API
func TestApicurioRegistryAPI(t TestingTB, ctx *TestingContext) {
	host, err := getHost(ctx.Client)
	if err != nil {
		t.Fatalf("error getting ApicurioRegistry CR: %v", err)
	}

	apiClient := resources.NewApicurioRegistryApiClient(host, ctx.HttpClient)

	err = apiClient.CreateArtifact(artifactID, artifactType, artifactData)
	if err != nil {
		t.Fatalf("error creating artifact: %v", err)
	}

	data, err := apiClient.ReadArtifact(artifactID)
	if err != nil {
		t.Fatalf("error reading artifact: %v", err)
	}
	if data != artifactData {
		t.Fatalf("unexpected artifact data: %v", data)
	}

	err = apiClient.DeleteArtifact(artifactID)
	if err != nil {
		t.Fatalf("error deleting artifact: %v", err)
	}
}

func getHost(client dynclient.Client) (string, error) {
	apicurioRegistry := &apicurioregistry.ApicurioRegistry{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: string(integreatlyv1alpha1.ProductApicurioRegistry), Namespace: ApicurioRegistryProductNamespace}, apicurioRegistry)
	if err != nil {
		return "", err
	}
	return apicurioRegistry.Status.Host, nil
}
