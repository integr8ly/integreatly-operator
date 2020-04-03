package common

import (
	goctx "context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
	v1 "k8s.io/api/core/v1"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

const (
	threescaleLoginUser = "customer-admin-0"
)

func lookup3ScaleClientSecret(client dynclient.Client, clientId string) (string, error) {
	secretName := fmt.Sprintf("keycloak-client-secret-%v", clientId)
	selector := dynclient.ObjectKey{
		Namespace: "redhat-rhmi-rhsso",
		Name:      secretName,
	}

	secret := &v1.Secret{}
	err := client.Get(goctx.TODO(), selector, secret)
	if err != nil {
		return "", err
	}

	return string(secret.Data["CLIENT_SECRET"]), nil
}

// Tests that a user in group rhmi-developers can log into fuse and
// create an integration
func Test3ScaleCrudlPermissions(t *testing.T, ctx *TestingContext) {
	if err := createTestingIDP(goctx.TODO(), ctx.Client, ctx.HttpClient, ctx.SelfSignedCerts); err != nil {
		t.Fatalf("error while creating testing idp: %v", err)
	}

	// get console master url
	rhmi, err := getRHMI(ctx.Client)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	masterURL := rhmi.Spec.MasterURL

	clientSecret, err := lookup3ScaleClientSecret(ctx.Client, "3scale")
	if err != nil {
		t.Fatal(err)
	}

	// Get the fuse host url from the rhmi status
	host := rhmi.Status.Stages[v1alpha1.ProductsStage].Products[v1alpha1.Product3Scale].Host
	keycloakHost := rhmi.Status.Stages[v1alpha1.AuthenticationStage].Products[v1alpha1.ProductRHSSO].Host
	redirectUrl := fmt.Sprintf("%v/p/admin/dashboard", host)

	tsClient := resources.NewThreeScaleAPIClient(host, keycloakHost, redirectUrl, ctx.HttpClient, ctx.Client)

	// First login to OpenShift
	err = tsClient.LoginOpenshift(masterURL, threescaleLoginUser, DefaultPassword, rhmi.Spec.NamespacePrefix)
	if err != nil {
		t.Fatal(err)
	}

	// Login to 3Scale via rhsso
	err = tsClient.Login3Scale(clientSecret)
	if err != nil {
		t.Fatal(err)
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
