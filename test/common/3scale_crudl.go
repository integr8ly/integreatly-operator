package common

import (
	goctx "context"
	"fmt"
	"math/rand"
	"time"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
)

var (
	threescaleLoginUser = fmt.Sprintf("%v%02d", defaultDedicatedAdminName, 1)
)

// Tests that a user in group rhmi-developers can log into fuse and
// create an integration
func Test3ScaleCrudlPermissions(t TestingTB, ctx *TestingContext) {
	if err := createTestingIDP(t, goctx.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Fatalf("error while creating testing idp: %v", err)
	}

	// get console master url
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)

	}

	// Get the fuse host url from the rhmi status
	host := rhmi.Status.Stages[rhmiv1alpha1.ProductsStage].Products[rhmiv1alpha1.Product3Scale].Host
	if host == "" {
		host = fmt.Sprintf("https://3scale-admin.%v", rhmi.Spec.RoutingSubdomain)
	}
	keycloakHost := rhmi.Status.Stages[rhmiv1alpha1.AuthenticationStage].Products[rhmiv1alpha1.ProductRHSSO].Host
	redirectUrl := fmt.Sprintf("%v/p/admin/dashboard", host)

	tsClient := resources.NewThreeScaleAPIClient(host, keycloakHost, redirectUrl, ctx.HttpClient, ctx.Client, t)

	// Login to 3Scale
	err = loginToThreeScale(t, host, threescaleLoginUser, DefaultPassword, "testing-idp", ctx.HttpClient)
	if err != nil {
		// t.Fatalf("[%s] error occurred: %v", getTimeStampPrefix(), err)
		t.Skipf("flakey test [%s] error ocurred: %v jira https://issues.redhat.com/browse/MGDAPI-557 ", getTimeStampPrefix(), err)
	}

	// Make sure 3Scale is available
	err = tsClient.Ping()
	if err != nil {
		t.Log("Error during making sure 3Scale is available")
		t.Fatal(err)
	}

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	// Create a product
	productId, err := tsClient.CreateProduct(fmt.Sprintf("dummy-product-%v", r1.Intn(100000)))
	if err != nil {
		t.Log("Error during create the product")
		t.Fatal(err)
	}

	// Delete the product
	err = tsClient.DeleteProduct(productId)
	if err != nil {
		t.Log("Error during deleting the product")
		t.Fatal(err)
	}
}
