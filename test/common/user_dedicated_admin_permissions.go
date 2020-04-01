package common

import (
	goctx "context"
	"crypto/tls"
	"fmt"
	"github.com/integr8ly/integreatly-operator/test/resources"
	projectv1 "github.com/openshift/api/project/v1"
	v1 "github.com/openshift/api/route/v1"
	"golang.org/x/net/publicsuffix"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
)

var productNamespaces = []string{
	"3scale",
	"3scale-operator",
	"amq-online",
	"amq-online-operator",
	"apicurito",
	"apicurito-operator",
	"cloud-resources-operator",
	"codeready-workspaces",
	"codeready-workspaces-operator",
	"fuse",
	"fuse-operator",
	"middleware-monitoring",
	"middleware-monitoring-operator",
	"operator",
	"rhsso",
	"rhsso-operator",
	"solution-explorer",
	"solution-explorer-operator",
	"ups",
	"ups-operator",
	"user-sso",
	"user-sso-operator",
}

func TestDedicatedAdminUserPermissions(t *testing.T, ctx *TestingContext) {
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
	masterURL := rhmi.Spec.MasterURL

	// get oauth route
	oauthRoute := &v1.Route{}
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: resources.OpenshiftOAuthRouteName, Namespace: resources.OpenshiftAuthenticationNamespace}, oauthRoute); err != nil {
		t.Fatal("error getting Openshift Oauth Route: ", err)
	}

	// get dedicated admin token
	if err := resources.DoAuthOpenshiftUser(fmt.Sprintf("%s/auth/login", masterURL), "customer-admin-1", DefaultPassword, httpClient); err != nil {
		t.Fatalf("error occured trying to get token : %v", err)
	}

	openshiftClient := &resources.OpenshiftClient{HTTPClient: httpClient}

	// get projects for dedicated admin
	dedicatedAdminFoundProjects, err := openshiftClient.DoOpenshiftGetProjects(masterURL)
	if err != nil {
		t.Fatalf("error occured while getting user projects : %v", err)
	}

	// check if projects are as expected for rhmi developer
	if result := verifyDedicatedAdminProjectPermissions(dedicatedAdminFoundProjects.Items); !result {
		t.Fatal("test-failed - projects missing for dedicated-admins")
	}

	// build array of rhmi namespaces
	var rhmiNamespaces []string
	for _, product := range productNamespaces {
		rhmiNamespaces = append(rhmiNamespaces, fmt.Sprintf("%s%s", NamespacePrefix, product))
	}

	// check to ensure dedicated admin is forbidden from rhmi namespace secrets
	for _, namespace := range rhmiNamespaces {
		path := fmt.Sprintf("/api/kubernetes/api/v1/namespaces/%s/secrets", namespace)
		resp, err := openshiftClient.DoOpenshiftGetRequest(masterURL, path)
		if err != nil {
			t.Fatalf("error occurred while executing oc get request: %v", err)
		}
		if resp.StatusCode != 403 {
			t.Fatalf("test-failed - status code found : %d expected status code : 403 RHMI dedicated admin should be forbidden from %s secrets", resp.StatusCode, namespace)
		}
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
