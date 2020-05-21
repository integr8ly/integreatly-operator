package common

import (
	goctx "context"
	"encoding/json"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"io/ioutil"
	"testing"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/integr8ly/integreatly-operator/test/resources"
	projectv1 "github.com/openshift/api/project/v1"
	v1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	expectedRhmiDeveloperProjectCount = 1
	expectedFusePodCount              = 6
)

// struct used to create query string for fuse logs endpoint
type LogOptions struct {
	Container string `url:"container"`
	Follow    string `url:"follow"`
	TailLines string `url:"tailLines"`
}

func TestRHMIDeveloperUserPermissions(t *testing.T, ctx *TestingContext) {
	if err := createTestingIDP(t, goctx.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Fatalf("error while creating testing idp: %v", err)
	}

	// get console master url
	rhmi, err := getRHMI(ctx.Client)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	masterURL := rhmi.Spec.MasterURL
	t.Logf("retrieved console master URL %v", masterURL)

	// get oauth route
	oauthRoute := &v1.Route{}
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: resources.OpenshiftOAuthRouteName, Namespace: resources.OpenshiftAuthenticationNamespace}, oauthRoute); err != nil {
		t.Fatal("error getting Openshift Oauth Route: ", err)
	}
	t.Log("retrieved openshift-Oauth route")

	// get rhmi developer user tokens
	if err := resources.DoAuthOpenshiftUser(fmt.Sprintf("%s/auth/login", masterURL), "test-user-1", DefaultPassword, ctx.HttpClient, TestingIDPRealm, t); err != nil {
		t.Fatalf("error occured trying to get token : %v", err)
	}
	t.Log("retrieved rhmi developer user tokens")
	openshiftClient := resources.NewOpenshiftClient(ctx.HttpClient, masterURL)

	// test rhmi developer projects are as expected
	t.Log("testing rhmi developer projects")
	fuseNamespace := fmt.Sprintf("%sfuse", NamespacePrefix)
	if err := testRHMIDeveloperProjects(masterURL, fuseNamespace, openshiftClient); err != nil {
		t.Fatalf("test failed - %v", err)
	}

	// get fuse pods for rhmi developer
	podlist, err := openshiftClient.ListPods(fuseNamespace)
	if err != nil {
		t.Fatalf("error occured while getting pods : %v", err)
	}

	// check if six pods are running
	runningCount := 0
	for _, p := range podlist.Items {
		if p.Status.Phase == "Running" {
			runningCount++
		}
	}
	if runningCount != expectedFusePodCount {
		t.Fatalf("test-failed - expected fuse pod count : %d found fuse pod count: %d", expectedFusePodCount, runningCount)
	}

	// log through rhmi developer fuse podlist
	for _, p := range podlist.Items {
		if p.Status.Phase == "Running" {
			logOpt := LogOptions{p.Spec.Containers[0].Name, "false", "10"}
			lv, err := query.Values(logOpt)
			if err != nil {
				t.Fatal(err)
			}
			// verify an rhmi developer can access the pods logs
			podPath := fmt.Sprintf(resources.OpenshiftPathListPods, p.Namespace)
			resp, err := openshiftClient.GetRequest(fmt.Sprintf("%s/%s/log?%s", podPath, p.Name, lv.Encode()))
			if err != nil {
				t.Fatalf("error occurred making oc get request: %v", err)
			}
			if resp.StatusCode != 200 {
				t.Fatalf("test-failed - status code %d RHMI developer unable to access fuse logs in pod %s : %v", resp.StatusCode, p.Name, err)
			}
		}
	}

	// Verify RHMI Developer permissions around RHMI Config
	verifyRHMIDeveloperRHMIConfigPermissions(t, openshiftClient)

	verifyRHMIDeveloper3ScaleRoutePermissions(t, openshiftClient)
}

// Verify that a dedicated admin can edit routes in the 3scale namespace
func verifyRHMIDeveloper3ScaleRoutePermissions(t *testing.T, client *resources.OpenshiftClient) {
	ns := "redhat-rhmi-3scale"
	route := "backend"

	path := fmt.Sprintf(resources.PathGetRoute, ns, route)
	resp, err := client.DoOpenshiftGetRequest(path)
	if err != nil {
		t.Errorf("Failed to get route : %s", err)
	}
	if resp.StatusCode != 403 {
		t.Errorf("RHMI Developer was incorrectly able to get route : %v", resp)
	}
}

func testRHMIDeveloperProjects(masterURL, fuseNamespace string, openshiftClient *resources.OpenshiftClient) error {
	var rhmiDevfoundProjects *projectv1.ProjectList
	// five minute time out needed to ensure users have been reconciled by RHMI operator
	err := wait.PollImmediate(time.Second*5, time.Minute*5, func() (done bool, err error) {
		rhmiDevfoundProjects, err = openshiftClient.ListProjects()
		if err != nil {
			return false, fmt.Errorf("error occured while getting user projects : %w", err)
		}

		// check if projects are as expected for rhmi developer
		if len(rhmiDevfoundProjects.Items) != expectedRhmiDeveloperProjectCount {
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		if rhmiDevfoundProjects != nil {
			return fmt.Errorf("unexpected developer project count : %d expected project count : %d , error occurred - %w", len(rhmiDevfoundProjects.Items), expectedRhmiDeveloperProjectCount, err)
		} else {
			return fmt.Errorf("unexpected error occurred when retrieving projects list - %w", err)
		}

	}

	foundNamespace := rhmiDevfoundProjects.Items[0].Name
	if foundNamespace != fuseNamespace {
		return fmt.Errorf("found rhmi developer project: %s expected rhmi developer project : %s", foundNamespace, fuseNamespace)
	}

	return nil
}

func verifyRHMIDeveloperRHMIConfigPermissions(t *testing.T, openshiftClient *resources.OpenshiftClient) {
	// RHMI Developer can not LIST RHMI Config CR
	resp, err := openshiftClient.DoOpenshiftGetRequest(resources.PathListRHMIConfig)

	if err != nil {
		t.Errorf("failed to perform LIST request for rhmi config with error : %s", err)
	}

	if resp.StatusCode != 403 {
		t.Errorf("unexpected response from LIST request for rhmi config : %v", resp)
	}

	// RHMI Developer can not GET RHMI Config CR
	path := fmt.Sprintf(resources.PathGetRHMIConfig, "rhmi-config")

	resp, err = openshiftClient.DoOpenshiftGetRequest(path)

	if err != nil {
		t.Errorf("failed to perform GET request for rhmi config with error : %s", err)
	}

	if resp.StatusCode != 403 {
		t.Errorf("unexpected response from GET request for rhmi config : %v", resp)
	}

	// RHMI Developer can not UPDATE RHMI Config CR
	bodyBytes, err := ioutil.ReadAll(resp.Body) // Use response from GET

	resp, err = openshiftClient.DoOpenshiftPutRequest(path, bodyBytes)

	if err != nil {
		t.Errorf("failed to perform UPDATE request for rhmi config with error : %s", err)
	}

	if resp.StatusCode != 403 {
		t.Errorf("unexpected response from UPDATE request for rhmi config : %v", resp)
	}

	// RHMI Developer can not CREATE new RHMI config
	rhmiConfig := &integreatlyv1alpha1.RHMIConfig{}
	bodyBytes, err = json.Marshal(rhmiConfig)

	resp, err = openshiftClient.DoOpenshiftPostRequest(path, bodyBytes)

	if err != nil {
		t.Errorf("failed to perform CREATE request for rhmi config with error : %s", err)
	}

	if resp.StatusCode != 403 {
		t.Errorf("unexpected response from CREATE request for rhmi config : %v", resp)
	}

	// RHMI Developer can not DELETE RHMI config
	resp, err = openshiftClient.DoOpenshiftDeleteRequest(path)

	if err != nil {
		t.Errorf("failed to perform DELETE request for rhmi config with error : %s", err)
	}

	if resp.StatusCode != 403 {
		t.Errorf("unexpected response from DELETE request for rhmi config : %v", resp)
	}
}
