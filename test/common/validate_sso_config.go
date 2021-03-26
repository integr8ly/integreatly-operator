package common

import (
	goctx "context"
	"fmt"
	"github.com/headzoo/surf"
	brow "github.com/headzoo/surf/browser"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSSOconfig(t TestingTB, ctx *TestingContext) {

	// Validate CPU value requested by SSO
	keycloak := &v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name: string(integreatlyv1alpha1.ProductRHSSO),
		},
	}
	err := ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: keycloak.Name, Namespace: NamespacePrefix + keycloak.Name}, keycloak)
	if err != nil {
		t.Fatalf("Couldn't get RHSSO config: %v", err)
	}

	expected := resource.MustParse("0.65") // needs to be updated once the dynamic SKU config is up. Otherwise will fail
	received := keycloak.Spec.KeycloakDeploymentSpec.Resources.Requests.Cpu()
	t.Log("Checking that requested 650m VCPU for a Keycloak pod")
	if !(expected.Equal(*received)) {
		t.Fatalf("Wrong requested CPU. Expected: %v but got: %v", expected, received)
	}

	// Validate RHSSO URL from RHOAM CR
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	adminRoute := rhmi.Status.Stages["authentication"].Products["rhsso"].Host
	browser := surf.NewBrowser()
	browser.SetCookieJar(ctx.HttpClient.Jar)
	browser.SetTransport(ctx.HttpClient.Transport)
	browser.SetAttribute(brow.FollowRedirects, true)

	t.Log("Checking the link for admin is available")
	// open login page
	err = browser.Open(fmt.Sprintf("%s/auth", adminRoute))
	if err != nil {
		t.Errorf("failed to open browser url: %w", err)
	}
	// Validate USER-SSO URL from RHOAM CR

	customerAdminRoute := fmt.Sprintf("%s/auth/", rhmi.Status.Stages["products"].Products["rhssouser"].Host)

	t.Log("Checking the link for dedicated admin is available")
	// open User SSO route
	err = browser.Open(customerAdminRoute)
	if err != nil {
		t.Errorf("failed to open browser url: %w", err)
	}
	t.Log("A32 - Validate SSO config test succeeded")
}
