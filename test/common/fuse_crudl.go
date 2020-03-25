package common

import (
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
	"testing"
)

const (
	fuseLoginUser = "test-user01"
)

// Tests that a user in group rhmi-developers can log into fuse and
// create an integration
func TestFuseCrudlPermissions(t *testing.T, ctx *TestingContext) {

	rhmi, err := getRHMI(ctx)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	// Get the fuse host url from the rhmi status
	fuseHost := rhmi.Status.Stages[v1alpha1.ProductsStage].Products[v1alpha1.ProductFuse].Host

	// Get a client that authenticated to fuse via oauth
	authenticatedFuseClient, err := resources.ProxyOAuth(fuseHost, fuseLoginUser, DefaultPassword)
	if err != nil {
		t.Fatalf("error authenticating with fuse: %v", err)
	}

	fuseApi := resources.NewFuseApiClient(fuseHost, authenticatedFuseClient)

	err = fuseApi.Ping()
	if err != nil {
		t.Fatal(err)
	}

	// Make sure there are no integrations present
	count, err := fuseApi.CountIntegrations()
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected no fuse integrations, but %v found", count)
	}

	// Create one integration
	integrationId, err := fuseApi.CreateIntegration()
	if err != nil {
		t.Fatal(err)
	}

	// Now there should be exactly one integration
	count, err = fuseApi.CountIntegrations()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 fuse integration,  but %v found", count)
	}

	// Delete the previously created integration
	err = fuseApi.DeleteIntegration(integrationId)
	if err != nil {
		t.Fatal(err)
	}
}
