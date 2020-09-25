package common

import (
	goctx "context"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	enmasseadminv1beta1 "github.com/integr8ly/integreatly-operator/pkg/apis-products/enmasse/admin/v1beta1"
	enmassev1beta1 "github.com/integr8ly/integreatly-operator/pkg/apis-products/enmasse/v1beta1"
	enmassev1beta2 "github.com/integr8ly/integreatly-operator/pkg/apis-products/enmasse/v1beta2"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

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
	"user-sso",
	"user-sso-operator",
}

// Applicable to install types used in 2.X
var rhmi2NamespacesPermissions = []string{
	"amq-online",
	"amq-online-operator",
	"apicurito",
	"apicurito-operator",
	"cloud-resources-operator",
	"codeready-workspaces",
	"codeready-workspaces-operator",
	"fuse",
	"fuse-operator",
	"solution-explorer",
	"solution-explorer-operator",
	"ups",
	"ups-operator",
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

	// Verify Dedicated admins permissions around secrets
	verifyDedicatedAdminSecretPermissions(t, openshiftClient, rhmi.Spec.Type)

	// Verify Dedicated admin permissions around RHMI Config
	verifyDedicatedAdminRHMIConfigPermissions(t, openshiftClient)

	verifyDedicatedAdmin3ScaleRoutePermissions(t, openshiftClient)

	if rhmi.Spec.Type != string(integreatlyv1alpha1.InstallationTypeManagedApi) {

		// Verify dedicated admin permissions around StandardInfraConfig
		verifyDedicatedAdminStandardInfraConfigPermissions(t, openshiftClient)

		// Verify dedicated admin permissions around BrokeredInfraConfig
		verifyDedicatedAdminBrokeredInfraConfigPermissions(t, openshiftClient)

		// Verify dedicated admin permissions around AddressSpacePlan
		verifyDedicatedAdminAddressSpacePlanPermissions(t, openshiftClient)

		// Verify dedicated admin permissions around AddressPlan
		verifyDedicatedAdminAddressPlanPermissions(t, openshiftClient)

		// Verify dedicated admin permissions around AuthenticationService
		verifyDedicatedAdminAuthenticationServicePermissions(t, openshiftClient)

		// Verify dedicated admin Role / Role binding for AMQ Online resources
		verifyDedicatedAdminAMQOnlineRolePermissions(t, ctx)
	}
}

// Verify that a dedicated admin can edit routes in the 3scale namespace
func verifyDedicatedAdmin3ScaleRoutePermissions(t *testing.T, client *resources.OpenshiftClient) {
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

func verifyDedicatedAdminSecretPermissions(t *testing.T, openshiftClient *resources.OpenshiftClient, installType string) {
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
	resp, err := openshiftClient.GetRequest(fmt.Sprintf(resources.OpenshiftPathGetSecret, RHMIOperatorNamespace) + "/github-oauth-secret")
	if err != nil {
		t.Errorf("error occurred while executing oc get request: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("test-failed - status code found : %d expected status code : 200 - RHMI dedicated admin should have access to github oauth secret in %s", resp.StatusCode, RHMIOperatorNamespace)
	}
}

func getProductNamespaces(installType string) []string {
	if installType == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		return commonNamespaces
	} else {
		return append(commonNamespaces, rhmi2NamespacesPermissions...)
	}
}

// Verify Dedicated admin permissions for RHMIConfig Resource - CRUDL
func verifyDedicatedAdminRHMIConfigPermissions(t *testing.T, openshiftClient *resources.OpenshiftClient) {
	t.Log("Verifying Dedicated admin permissions for RHMIConfig Resource")

	expectedPermission := ExpectedPermissions{
		ExpectedCreateStatusCode: 403,
		ExpectedReadStatusCode:   403,
		ExpectedUpdateStatusCode: 403,
		ExpectedDeleteStatusCode: 403,
		ExpectedListStatusCode:   403,
		ListPath:                 fmt.Sprintf(resources.PathListRHMIConfig, RHMIOperatorNamespace),
		GetPath:                  fmt.Sprintf(resources.PathGetRHMIConfig, RHMIOperatorNamespace, "rhmi-config"),
		ObjectToCreate: &integreatlyv1alpha1.RHMIConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-rhmi-config",
				Namespace: RHMIOperatorNamespace,
			},
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1alpha1",
				Kind:       "RHMIConfig",
			},
		},
	}

	verifyCRUDLPermissions(t, openshiftClient, expectedPermission)
}

// Verify Dedicated admin permissions for StandardInfraConfig Resource - CRUDL
func verifyDedicatedAdminStandardInfraConfigPermissions(t *testing.T, openshiftClient *resources.OpenshiftClient) {
	t.Log("Verifying Dedicated admin permissions for StandardInfraConfig Resource")

	resourceName := "test-standard-infra-config"

	expectedPermission := ExpectedPermissions{
		ExpectedCreateStatusCode: 201,
		ExpectedReadStatusCode:   200,
		ExpectedUpdateStatusCode: 200,
		ExpectedDeleteStatusCode: 200,
		ExpectedListStatusCode:   200,
		ListPath:                 fmt.Sprintf(resources.PathListStandardInfraConfig, AMQOnlineOperatorNamespace),
		GetPath:                  fmt.Sprintf(resources.PathGetStandardInfraConfig, AMQOnlineOperatorNamespace, resourceName),
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
	t.Log("Verifying Dedicated admin permissions for BrokeredInfraConfig Resource")

	resourceName := "test-brokered-infra-config"

	expectedPermission := ExpectedPermissions{
		ExpectedCreateStatusCode: 201,
		ExpectedReadStatusCode:   200,
		ExpectedUpdateStatusCode: 200,
		ExpectedDeleteStatusCode: 200,
		ExpectedListStatusCode:   200,
		ListPath:                 fmt.Sprintf(resources.PathListBrokeredInfraConfig, AMQOnlineOperatorNamespace),
		GetPath:                  fmt.Sprintf(resources.PathGetBrokeredInfraConfig, AMQOnlineOperatorNamespace, resourceName),
		ObjectToCreate: enmassev1beta1.BrokeredInfraConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
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

// Verify Dedicated admin permissions for AddressSpacePlan Resource - CRUDL
func verifyDedicatedAdminAddressSpacePlanPermissions(t *testing.T, openshiftClient *resources.OpenshiftClient) {
	t.Log("Verifying Dedicated admin permissions for AddressSpacePlan Resource")

	resourceName := "test-address-plan-space"

	expectedPermission := ExpectedPermissions{
		ExpectedCreateStatusCode: 201,
		ExpectedReadStatusCode:   200,
		ExpectedUpdateStatusCode: 200,
		ExpectedDeleteStatusCode: 200,
		ExpectedListStatusCode:   200,
		ListPath:                 fmt.Sprintf(resources.PathListAddressSpacePlan, AMQOnlineOperatorNamespace),
		GetPath:                  fmt.Sprintf(resources.PathGetAddressSpacePlan, AMQOnlineOperatorNamespace, resourceName),
		ObjectToCreate: enmassev1beta2.AddressSpacePlan{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: AMQOnlineOperatorNamespace,
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "AddressSpacePlan",
				APIVersion: "admin.enmasse.io/v1beta2",
			},
			Spec: enmassev1beta2.AddressSpacePlanSpec{
				AddressPlans:     []string{"standard-small-queue"},
				AddressSpaceType: "standard",
				InfraConfigRef:   "default-minimal",
				ResourceLimits: enmassev1beta2.AddressSpacePlanResourceLimits{
					Router:    1,
					Broker:    1,
					Aggregate: 1,
				},
			},
		},
	}

	verifyCRUDLPermissions(t, openshiftClient, expectedPermission)
}

// Verify Dedicated admin permissions for AddressPlan Resource - CRUDL
func verifyDedicatedAdminAddressPlanPermissions(t *testing.T, openshiftClient *resources.OpenshiftClient) {
	t.Log("Verifying Dedicated admin permissions for AddressPlan Resource")

	resourceName := "test-address-plan"

	expectedPermission := ExpectedPermissions{
		ExpectedCreateStatusCode: 201,
		ExpectedReadStatusCode:   200,
		ExpectedUpdateStatusCode: 200,
		ExpectedDeleteStatusCode: 200,
		ExpectedListStatusCode:   200,
		ListPath:                 fmt.Sprintf(resources.PathListAddressPlan, AMQOnlineOperatorNamespace),
		GetPath:                  fmt.Sprintf(resources.PathGetAddressPlan, AMQOnlineOperatorNamespace, resourceName),
		ObjectToCreate: enmassev1beta2.AddressPlan{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: AMQOnlineOperatorNamespace,
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "AddressPlan",
				APIVersion: "admin.enmasse.io/v1beta2",
			},
			Spec: enmassev1beta2.AddressPlanSpec{
				AddressType: "queue",
				Resources: enmassev1beta2.AddressPlanResources{
					Router: 0.01,
					Broker: 0.001,
				},
			},
		},
	}

	verifyCRUDLPermissions(t, openshiftClient, expectedPermission)
}

// Verify Dedicated admin permissions for AuthenticationService Resource - CRUDL
func verifyDedicatedAdminAuthenticationServicePermissions(t *testing.T, openshiftClient *resources.OpenshiftClient) {
	t.Log("Verifying Dedicated admin permissions for AuthenticationService Resource")

	resourceName := "test-authentication-service"
	expectedPermission := ExpectedPermissions{
		ExpectedCreateStatusCode: 201,
		ExpectedReadStatusCode:   200,
		ExpectedUpdateStatusCode: 200,
		ExpectedDeleteStatusCode: 200,
		ExpectedListStatusCode:   200,
		ListPath:                 fmt.Sprintf(resources.PathListAuthenticationService, AMQOnlineOperatorNamespace),
		GetPath:                  fmt.Sprintf(resources.PathGetAuthenticationService, AMQOnlineOperatorNamespace, resourceName),
		ObjectToCreate: enmasseadminv1beta1.AuthenticationService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: AMQOnlineOperatorNamespace,
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "AuthenticationService",
				APIVersion: "admin.enmasse.io/v1beta1",
			},
			Spec: enmasseadminv1beta1.AuthenticationServiceSpec{
				Type: enmasseadminv1beta1.External,
				External: &enmasseadminv1beta1.AuthenticationServiceSpecExternal{
					Host: "test",
					Port: 0,
				},
			},
			Status: enmasseadminv1beta1.AuthenticationServiceStatus{
				Host:  "test",
				Phase: enmasseadminv1beta1.AuthenticationServiceActive,
			},
		},
	}

	verifyCRUDLPermissions(t, openshiftClient, expectedPermission)
}

func verifyDedicatedAdminAMQOnlineRolePermissions(t *testing.T, ctx *TestingContext) {
	t.Log("Verifying Dedicated admin AMQ Online resource role / role binding")

	roleBinding := &rbacv1.RoleBinding{}
	if err := ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: "dedicated-admins-service-admin", Namespace: AMQOnlineOperatorNamespace}, roleBinding); err != nil {
		t.Fatalf("error getting dedicated-admins-service-admin role binding in %s namespace: %s", AMQOnlineOperatorNamespace, err)
	}

	// Verify dedicated admin group is in role binding
	found := false
	for _, subject := range roleBinding.Subjects {
		if subject.Name == "dedicated-admins" && subject.Kind == "Group" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("Did not find dedicated admin group in %s rolebinding in %s namespace", roleBinding.Name, roleBinding.Namespace)
	}

	// Verify permissions given by the role
	role := &rbacv1.Role{}
	if err := ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: roleBinding.RoleRef.Name, Namespace: AMQOnlineOperatorNamespace}, role); err != nil {
		t.Fatalf("error %s role in %s namespace: %s", roleBinding.RoleRef.Name, AMQOnlineOperatorNamespace, err)
	}

	haveCorrectPermission := false
	expectedRule := rbacv1.PolicyRule{
		Verbs:     []string{"create", "get", "update", "delete", "list", "watch", "patch"},
		APIGroups: []string{"admin.enmasse.io"},
		Resources: []string{"addressplans", "addressspaceplans", "brokeredinfraconfigs", "standardinfraconfigs", "authenticationservices"},
	}

	for _, rule := range role.Rules {
		if reflect.DeepEqual(rule, expectedRule) {
			haveCorrectPermission = true
			break
		}
	}

	if !haveCorrectPermission {
		t.Fatalf("Incorrect permissions found for %s role in %s namespace. Excpected %s as a policy rule ", role.Name, role.Namespace, expectedRule)
	}
}
