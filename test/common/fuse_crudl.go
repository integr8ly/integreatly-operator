package common

import (
	goctx "context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
	"testing"
)

var (
	fuseLoginUser     = fmt.Sprintf("%v-%d", DefaultTestUserName, 0)
	fuseLoginPassword = DefaultPassword
)

func loginOpenshift(t *testing.T, ctx *TestingContext, masterUrl, username, password, namespacePrefix string) error {
	authUrl := fmt.Sprintf("%s/auth/login", masterUrl)

	if err := resources.DoAuthOpenshiftUser(authUrl, username, password, ctx.HttpClient, "testing-idp", t); err != nil {
		return err
	}

	openshiftClient := resources.NewOpenshiftClient(ctx.HttpClient, masterUrl)
	if err := resources.OpenshiftUserReconcileCheck(openshiftClient, ctx.Client, namespacePrefix, username); err != nil {
		return err
	}
	return nil
}

// Tests that a user in group rhmi-developers can log into fuse and
// create an integration
func TestFuseCrudlPermissions(t *testing.T, ctx *TestingContext) {
	if err := createTestingIDP(t, goctx.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Fatalf("error while creating testing idp: %v", err)
	}

	// get console master url
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	masterURL := rhmi.Spec.MasterURL

	// Get the fuse host url from the rhmi status
	fuseHost := rhmi.Status.Stages[v1alpha1.ProductsStage].Products[v1alpha1.ProductFuse].Host

	err = loginOpenshift(t, ctx, masterURL, fuseLoginUser, fuseLoginPassword, rhmi.Spec.NamespacePrefix)
	if err != nil {
		t.Fatalf("error logging into openshift: %v", err)
	}

	// Get a client that authenticated to fuse via oauth
	authenticatedFuseClient, err := resources.ProxyOAuth(ctx.HttpClient, fuseHost, fuseLoginUser, DefaultPassword)
	if err != nil {
		t.Fatalf("error authenticating with fuse: %v", err)
	}

	fuseApi := resources.NewFuseApiClient(fuseHost, authenticatedFuseClient)

	err = fuseApi.Ping()
	if err != nil {
		t.Fatalf("error pinging fuse: %v", err)
	}

	// Make sure there are no integrations present
	count, err := fuseApi.CountIntegrations()
	if err != nil {
		t.Fatalf("error counting integrations: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no fuse integrations, but %v found", count)
	}

	// Create one integration
	integrationId, err := fuseApi.CreateIntegration(resources.FuseIntegrationPayload)
	if err != nil {
		t.Fatalf("error creating integration: %v", err)
	}

	// Now there should be exactly one integration
	count, err = fuseApi.CountIntegrations()
	if err != nil {
		t.Fatalf("error counting integrations: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 fuse integration,  but %v found", count)
	}

	// Delete the previously created integration
	err = fuseApi.DeleteIntegration(integrationId)
	if err != nil {
		t.Fatalf("error deleting integration: %v", err)
	}
}
