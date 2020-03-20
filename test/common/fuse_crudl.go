package common

import (
	goctx "context"
	"crypto/tls"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
	"golang.org/x/net/publicsuffix"
	"net/http"
	"net/http/cookiejar"
	"testing"
)

const (
	fuseLoginUser = "test-user01"
)

// Tests that a user in group rhmi-developers can log into fuse and
// create an integration
func TestFuseCrudlPermissions(t *testing.T, ctx *TestingContext) {
	// declare transport
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: ctx.SelfSignedCerts},
	}

	// declare new cookie jar om nom nom
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		t.Fatal("error occurred creating a new cookie jar", err)
	}

	// declare http client
	httpClient := &http.Client{
		Transport: tr,
		Jar:       jar,
	}

	if err := createTestingIDP(goctx.TODO(), ctx.Client, httpClient, ctx.SelfSignedCerts); err != nil {
		t.Fatalf("error while creating testing idp: %v", err)
	}

	// get console master url
	rhmi, err := getRHMI(ctx.Client)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	// Get the fuse host url from the rhmi status
	fuseHost := rhmi.Status.Stages[v1alpha1.ProductsStage].Products[v1alpha1.ProductFuse].Host

	// Get a client that authenticated to fuse via oauth
	authenticatedFuseClient, err := resources.ProxyOAuth(httpClient, fuseHost, fuseLoginUser, DefaultPassword)
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
	integrationId, err := fuseApi.CreateIntegration(resources.FuseIntegrationPayload)
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
