package common

import (
	goctx "context"
	"fmt"
	"math/rand"
	"testing"
	"time"

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
	rhmi, err := GetRHMI(ctx.Client, true)
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
		dumpAuthResources(ctx.Client, t)
		// t.Skip("Skipping due to known flaky behavior, to be fixed ASAP.\nJIRA: https://issues.redhat.com/browse/INTLY-10087")
		t.Fatalf("[%s] error ocurred: %v", getTimeStampPrefix(), err)
	}

	// Make sure 3Scale is available
	err = tsClient.Ping()
	if err != nil {
		t.Fatal(err)
	}

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	// Create a product
	productId, err := tsClient.CreateProduct(fmt.Sprintf("dummy-product-%v", r1.Intn(100000)))
	if err != nil {
		t.Fatal(err)
	}

	// Delete the product
	err = tsClient.DeleteProduct(productId)
	if err != nil {
		t.Fatal(err)
	}
}
