package common

import (
	goctx "context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

	"github.com/integr8ly/integreatly-operator/test/resources"
	projectv1 "github.com/openshift/api/project/v1"
	v1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	if err := createTestingIDP(t, goctx.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
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
	if err := resources.DoAuthOpenshiftUser(fmt.Sprintf("%s/auth/login", masterURL), "customer-admin-1", DefaultPassword, ctx.HttpClient, TestingIDPRealm, t); err != nil {
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
			t.Fatalf("error occurred while executing oc get request: %v", err)
		}
		if resp.StatusCode != 403 {
			t.Fatalf("test-failed - status code found : %d expected status code : 403 RHMI dedicated admin should be forbidden from %s secrets", resp.StatusCode, namespace)
		}
	}

	// Verify Dedicated admin permissions around RHMI Config
	verifyDedicatedAdminRHMIConfigPermissions(t, openshiftClient)
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

func verifyDedicatedAdminRHMIConfigPermissions(t *testing.T, openshiftClient *resources.OpenshiftClient) {
	// Dedicated admin can LIST RHMI Config CR
	resp, err := openshiftClient.DoOpenshiftGetRequest(resources.PathListRHMIConfig)

	if err != nil {
		t.Errorf("failed to perform LIST request for rhmi config with error : %s", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("unexpected response from LIST request for rhmi config : %v", resp)
	}

	// Dedicated admin can GET RHMI Config CR
	path := fmt.Sprintf(resources.PathGetRHMIConfig, "rhmi-config")

	resp, err = openshiftClient.DoOpenshiftGetRequest(path)

	if err != nil {
		t.Errorf("failed to perform GET request for rhmi config with error : %s", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("unexpected response from GET request for rhmi config : %v", resp)
	}

	// Dedicated admin can UPDATE RHMI Config CR
	bodyBytes, err := ioutil.ReadAll(resp.Body) // Use response from GET

	resp, err = openshiftClient.DoOpenshiftPutRequest(path, bodyBytes)

	if err != nil {
		t.Errorf("failed to perform UPDATE request for rhmi config with error : %s", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("unexpected response from UPDATE request for rhmi config : %v", resp)
	}

	// Dedicate admin can not CREATE new RHMI config
	rhmiConfig := &integreatlyv1alpha1.RHMIConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1alpha1",
			Kind:       "RHMIConfig",
		},
	}
	bodyBytes, err = json.Marshal(rhmiConfig)

	resp, err = openshiftClient.DoOpenshiftPostRequest(path, bodyBytes)

	if err != nil {
		t.Errorf("failed to perform CREATE request for rhmi config with error : %s", err)
	}

	if resp.StatusCode != 403 {
		t.Errorf("unexpected response from CREATE request for rhmi config : %v", resp)
	}

	// Dedicate admin can not DELETE RHMI config
	resp, err = openshiftClient.DoOpenshiftDeleteRequest(path)

	if err != nil {
		t.Errorf("failed to perform DELETE request for rhmi config with error : %s", err)
	}

	if resp.StatusCode != 403 {
		t.Errorf("unexpected response from DELETE request for rhmi config : %v", resp)
	}
}
