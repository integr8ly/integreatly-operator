package common

import (
	goctx "context"
	"fmt"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
)

var (
	threescaleLoginUser = fmt.Sprintf("%v-%d", defaultDedicatedAdminName, 0)
)

// Tests that a user in group rhmi-developers can log into fuse and
// create an integration
func Test3ScaleCrudlPermissions(t *testing.T, ctx *TestingContext) {
	if err := createTestingIDP(t, goctx.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Fatalf("error while creating testing idp: %v", err)
	}

	// get console master url
	rhmi, err := getRHMI(ctx.Client)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	// Get the fuse host url from the rhmi status
	host := rhmi.Status.Stages[v1alpha1.ProductsStage].Products[v1alpha1.Product3Scale].Host
	if host == "" {
		host = fmt.Sprintf("https://3scale-admin.%v", rhmi.Spec.RoutingSubdomain)
	}
	keycloakHost := rhmi.Status.Stages[v1alpha1.AuthenticationStage].Products[v1alpha1.ProductRHSSO].Host
	redirectUrl := fmt.Sprintf("%v/p/admin/dashboard", host)

	tsClient := resources.NewThreeScaleAPIClient(host, keycloakHost, redirectUrl, ctx.HttpClient, ctx.Client, t)

	// Login to 3Scale
	err = loginToThreeScale(t, host, threescaleLoginUser, DefaultPassword, "testing-idp", ctx.HttpClient)
	if err != nil {
		// t.Skip("Skipping due to known flaky behavior, to be fixed ASAP.\nJIRA: https://issues.redhat.com/browse/INTLY-8433")
		dumpAuthResources(ctx.Client)
		t.Fatalf("[%s] error ocurred: %v", getTimeStampPrefix(), err)
	}

	// Make sure 3Scale is available
	err = tsClient.Ping()
	if err != nil {
		t.Fatal(err)
	}

	// Create a product
	productId, err := tsClient.CreateProduct("dummy-product")
	if err != nil {
		t.Fatal(err)
	}

	// Delete the product
	err = tsClient.DeleteProduct(productId)
	if err != nil {
		t.Fatal(err)
	}
}
