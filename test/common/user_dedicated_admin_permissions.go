package common

import (
	goctx "context"
	"fmt"
	enmassev1beta1 "github.com/integr8ly/integreatly-operator/pkg/apis-products/enmasse/v1beta1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"strings"
	"testing"

	"github.com/integr8ly/integreatly-operator/test/resources"
	projectv1 "github.com/openshift/api/project/v1"
	v1 "github.com/openshift/api/route/v1"
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

	verifyDedicatedAdmin3ScaleRoutePermissions(t, openshiftClient)

	// Verify dedicated admin permissions around StandardInfraConfig
	verifyDedicatedAdminStandardInfraConfigPermissions(t, openshiftClient)

	// Verify dedicated admin permissions around BrokeredInfraConfig
	verifyDedicatedAdminBrokeredInfraConfigPermissions(t, openshiftClient)
}

// Verify that a dedicated admin can edit routes in the 3scale namespace
func verifyDedicatedAdmin3ScaleRoutePermissions(t *testing.T, client *resources.OpenshiftClient) {
	ns := "redhat-rhmi-3scale"
	route := "backend"

	path := fmt.Sprintf(resources.PathGetRoute, ns, route)
	resp, err := client.DoOpenshiftGetRequest(path)
	if err != nil {
		t.Errorf("Failed to get route : %s", err)
	}
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

// Verify Dedicated admin permissions for RHMIConfig Resource - CRUDL
func verifyDedicatedAdminRHMIConfigPermissions(t *testing.T, openshiftClient *resources.OpenshiftClient) {
	expectedPermission := ExpectedPermissions{
		ExpectedCreateStatusCode: 403,
		ExpectedReadStatusCode:   200,
		ExpectedUpdateStatusCode: 200,
		ExpectedDeleteStatusCode: 403,
		ExpectedListStatusCode:   200,
		ListPath:                 fmt.Sprintf(resources.PathListRHMIConfig, RHMIOperatorNamespace),
		GetPath:                  fmt.Sprintf(resources.PathGetRHMIConfig, RHMIOperatorNamespace, "rhmi-config"),
		ObjectToCreate:           &integreatlyv1alpha1.RHMIConfig{},
	}

	verifyCRUDLPermissions(t, openshiftClient, expectedPermission)
}

// Verify Dedicated admin permissions for StandardInfraConfig Resource - CRUDL
func verifyDedicatedAdminStandardInfraConfigPermissions(t *testing.T, openshiftClient *resources.OpenshiftClient) {
	expectedPermission := ExpectedPermissions{
		ExpectedCreateStatusCode: 201,
		ExpectedReadStatusCode:   200,
		ExpectedUpdateStatusCode: 200,
		ExpectedDeleteStatusCode: 200,
		ExpectedListStatusCode:   200,
		ListPath:                 fmt.Sprintf(resources.PathListStandardInfraConfig, AMQOnlineOperatorNamespace),
		GetPath:                  fmt.Sprintf(resources.PathGetStandardInfraConfig, AMQOnlineOperatorNamespace, "test-standard-infra-config"),
		ObjectToCreate: enmassev1beta1.StandardInfraConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-standard-infra-config",
				Namespace: AMQOnlineOperatorNamespace,
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "StandardInfraConfig",
				APIVersion: "admin.enmasse.io/v1beta1",
			},
			Spec: enmassev1beta1.StandardInfraConfigSpec{
				Broker: enmassev1beta1.InfraConfigBroker{
					AddressFullPolicy: "FAIL",
				},
			},
		},
	}

	verifyCRUDLPermissions(t, openshiftClient, expectedPermission)
}

// Verify Dedicated admin permissions for BrokeredInfraConfig Resource - CRUDL
func verifyDedicatedAdminBrokeredInfraConfigPermissions(t *testing.T, openshiftClient *resources.OpenshiftClient) {
	expectedPermission := ExpectedPermissions{
		ExpectedCreateStatusCode: 201,
		ExpectedReadStatusCode:   200,
		ExpectedUpdateStatusCode: 200,
		ExpectedDeleteStatusCode: 200,
		ExpectedListStatusCode:   200,
		ListPath:                 fmt.Sprintf(resources.PathListBrokeredInfraConfig, AMQOnlineOperatorNamespace),
		GetPath:                  fmt.Sprintf(resources.PathGetBrokeredInfraConfig, AMQOnlineOperatorNamespace, "test-brokered-infra-config"),
		ObjectToCreate: enmassev1beta1.BrokeredInfraConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-brokered-infra-config",
				Namespace: AMQOnlineOperatorNamespace,
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "BrokeredInfraConfig",
				APIVersion: "admin.enmasse.io/v1beta1",
			},
			Spec: enmassev1beta1.BrokeredInfraConfigSpec{
				Broker: enmassev1beta1.InfraConfigBroker{
					AddressFullPolicy: "FAIL",
				},
			},
		},
	}

	verifyCRUDLPermissions(t, openshiftClient, expectedPermission)
}
