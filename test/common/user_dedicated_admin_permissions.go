package common

import (
	goctx "context"
	"fmt"
	"io/ioutil"
	"strings"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"

	"github.com/integr8ly/integreatly-operator/test/resources"
	projectv1 "github.com/openshift/api/project/v1"
	v1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Common to all install types including managed api
var commonNamespaces = []string{
	"3scale",
	"3scale-operator",
	"middleware-monitoring",
	"middleware-monitoring-operator",
	"operator",
	"rhsso",
	"rhsso-operator",
}

var managedAPINamespacesPermissions = []string{
	"user-sso",
	"user-sso-operator",
}

type ExpectedPermissions struct {
	ExpectedCreateStatusCode int
	ExpectedReadStatusCode   int
	ExpectedUpdateStatusCode int
	ExpectedDeleteStatusCode int
	ExpectedListStatusCode   int
	ListPath                 string
	GetPath                  string
	ObjectToCreate           interface{}
}

func TestDedicatedAdminUserPermissions(t TestingTB, ctx *TestingContext) {
	customerAdminUsername := fmt.Sprintf("%v%02d", defaultDedicatedAdminName, 1)

	if err := createTestingIDP(t, goctx.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Fatalf("error while creating testing idp: %v", err)
	}

	// get console master url
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	masterURL := rhmi.Spec.MasterURL

	// get oauth route
	oauthRoute := &v1.Route{}
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: resources.OpenshiftOAuthRouteName, Namespace: resources.OpenshiftAuthenticationNamespace}, oauthRoute); err != nil {
		t.Fatal("error getting Openshift Oauth Route: ", err)
	}

	// get dedicated admin token
	if err := resources.DoAuthOpenshiftUser(fmt.Sprintf("%s/auth/login", masterURL), customerAdminUsername, TestingIdpPassword, ctx.HttpClient, TestingIDPRealm, t); err != nil {
		t.Fatalf("error occured trying to get token : %v", err)
	}

	openshiftClient := resources.NewOpenshiftClient(ctx.HttpClient, masterURL)

	// get projects for dedicated admin
	dedicatedAdminFoundProjects, err := openshiftClient.ListProjects()
	if err != nil {
		t.Fatalf("error occured while getting user projects : %v", err)
	}

	// check if projects are as expected for rhmi developer
	if result := verifyDedicatedAdminProjectPermissions(dedicatedAdminFoundProjects.Items); !result {
		t.Fatal("test-failed - projects missing for dedicated-admins")
	}

	// Verify Dedicated admins permissions around secrets
	verifyDedicatedAdminSecretPermissions(t, openshiftClient, rhmi.Spec.Type)

	verifyDedicatedAdmin3ScaleRoutePermissions(t, openshiftClient)
}

// Verify that a dedicated admin can edit routes in the 3scale namespace
func verifyDedicatedAdmin3ScaleRoutePermissions(t TestingTB, client *resources.OpenshiftClient) {
	ns := NamespacePrefix + "3scale"
	route := "backend"

	path := fmt.Sprintf(resources.PathGetRoute, ns, route)
	resp, err := client.DoOpenshiftGetRequest(path)
	if err != nil {
		t.Errorf("Failed to get route : %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("Unable to get 3scale route as dedicated admin : %v", resp)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body) // Use response from GET
	if err != nil {
		t.Errorf("failed to read response body from get route request : %s", err)
	}

	path = fmt.Sprintf(resources.PathGetRoute, ns, route)
	resp, err = client.DoOpenshiftPutRequest(path, bodyBytes)

	if err != nil {
		t.Errorf("Failed to update route : %s", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Failed to update route as dedicated admin : %v", resp)
	}
}

// verifies that there is at least 1 project with a prefix `openshift` , `redhat` and `kube`
func verifyDedicatedAdminProjectPermissions(projects []projectv1.Project) bool {
	var hasOpenshiftPrefix, hasRedhatPrefix, hasKubePrefix bool
	for _, ns := range projects {
		if strings.HasPrefix(ns.Name, "openshift") {
			hasOpenshiftPrefix = true
		}
		if strings.HasPrefix(ns.Name, "redhat") {
			hasRedhatPrefix = true
		}
		if strings.HasPrefix(ns.Name, "kube") {
			hasKubePrefix = true
		}
	}
	return hasKubePrefix && hasRedhatPrefix && hasOpenshiftPrefix
}

func verifyDedicatedAdminSecretPermissions(t TestingTB, openshiftClient *resources.OpenshiftClient, installType string) {
	t.Log("Verifying Dedicated admin permissions for Secrets Resource")

	productNamespaces := getProductNamespaces(installType)

	// build array of rhmi namespaces
	var rhmiNamespaces []string
	for _, product := range productNamespaces {
		rhmiNamespaces = append(rhmiNamespaces, fmt.Sprintf("%s%s", NamespacePrefix, product))
	}

	// check to ensure dedicated admin is forbidden from rhmi namespace secrets
	for _, namespace := range rhmiNamespaces {
		path := fmt.Sprintf(resources.OpenshiftPathGetSecret, namespace)
		resp, err := openshiftClient.GetRequest(path)
		if err != nil {
			t.Errorf("error occurred while executing oc get request: %v", err)
			continue
		}
		if resp.StatusCode != 403 {
			t.Errorf("test-failed - status code found : %d expected status code : 403 RHMI dedicated admin should be forbidden from %s secrets", resp.StatusCode, namespace)
		}
	}

	// check dedicated admin can get github oauth secret
	resp, err := openshiftClient.GetRequest(fmt.Sprintf(resources.OpenshiftPathGetSecret, RHOAMOperatorNamespace) + "/github-oauth-secret")
	if err != nil {
		t.Errorf("error occurred while executing oc get request: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("test-failed - status code found : %d expected status code : 200 - RHMI dedicated admin should have access to github oauth secret in %s", resp.StatusCode, RHOAMOperatorNamespace)
	}
}

func getProductNamespaces(installType string) []string {
	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installType)) {
		return commonNamespaces
	} else {
		return append(commonNamespaces, managedAPINamespacesPermissions...)
	}
}
