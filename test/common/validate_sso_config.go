package common

import (
	goctx "context"
	"crypto/tls"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/headzoo/surf"
	brow "github.com/headzoo/surf/browser"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"
	"golang.org/x/net/context"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"net/http"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSSOconfig(t TestingTB, ctx *TestingContext) {
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	// get Keycloak CR
	keycloak := &v1alpha1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name: string(integreatlyv1alpha1.ProductRHSSOUser),
		},
	}
	err = ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: keycloak.Name, Namespace: RHSSOUserProductNamespace}, keycloak)
	if err != nil {
		t.Fatalf("Couldn't get RHSSO config: %v", err)
	}

	// get Keycloak pods
	keycloakPods := &v1.PodList{}
	selector, err := labels.Parse("component=keycloak")
	if err != nil {
		t.Fatal(err)
	}

	keycloakListOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(RHSSOUserProductNamespace),
		k8sclient.MatchingLabelsSelector{
			Selector: selector,
		},
	}

	err = ctx.Client.List(context.TODO(), keycloakPods, keycloakListOpts...)
	if err != nil {
		t.Fatalf("failed to get pods for Keycloak: %v", err)
	}

	t.Log("Checking requested resources present in Keycloak pods")
	expectedResources := keycloak.Spec.KeycloakDeploymentSpec.Resources
	for pod := range keycloakPods.Items {
		podResources := keycloakPods.Items[pod].Spec.Containers[0].Resources
		if !cmp.Equal(expectedResources.Requests.Cpu(), podResources.Requests.Cpu()) {
			t.Fatalf("Wrong requested CPU. Expected requests: %v but got: %v", expectedResources.Requests.Cpu().String(), podResources.Requests.Cpu().String())
		}
		if !cmp.Equal(expectedResources.Limits.Cpu(), podResources.Limits.Cpu()) {
			t.Fatalf("Wrong limits for CPU. Expected requests: %v but got: %v", expectedResources.Limits.Cpu().String(), podResources.Limits.Cpu().String())
		}
		if !cmp.Equal(expectedResources.Requests.Memory(), podResources.Requests.Memory()) {
			t.Fatalf("Wrong requested Memory. Expected requests: %v but got: %v", expectedResources.Requests.Memory().String(), podResources.Requests.Memory().String())
		}
		if !cmp.Equal(expectedResources.Limits.Memory(), podResources.Limits.Memory()) {
			t.Fatalf("Wrong limits for Memory. Expected requests: %v but got: %v", expectedResources.Limits.Memory().String(), podResources.Limits.Memory().String())
		}
	}

	// Validate RHSSO URL from RHOAM CR

	adminRoute := rhmi.Status.Stages[integreatlyv1alpha1.InstallStage].Products[integreatlyv1alpha1.ProductRHSSO].Host
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Note: Use with caution in production
	}
	browser := surf.NewBrowser()
	browser.SetCookieJar(ctx.HttpClient.Jar)
	browser.SetTransport(tr)
	browser.SetAttribute(brow.FollowRedirects, true)

	t.Log("Checking the link for admin is available")
	// open login page
	err = browser.Open(fmt.Sprintf("%s/auth", adminRoute))
	if err != nil {
		t.Errorf("failed to open browser url: %w", err)
	}
	// Validate USER-SSO URL from RHOAM CR

	customerAdminRoute := fmt.Sprintf("%s/auth/", rhmi.Status.Stages[integreatlyv1alpha1.InstallStage].Products[integreatlyv1alpha1.ProductRHSSOUser].Host)

	t.Log("Checking the link for dedicated admin is available")
	// open User SSO route
	err = browser.Open(customerAdminRoute)
	if err != nil {
		t.Errorf("failed to open browser url: %w", err)
	}
	t.Log("A32 - Validate SSO config test succeeded")
}
