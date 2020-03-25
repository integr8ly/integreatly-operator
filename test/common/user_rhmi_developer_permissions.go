package common

import (
	goctx "context"
	"fmt"
	"github.com/google/go-querystring/query"
	"github.com/integr8ly/integreatly-operator/test/resources"
	projectv1 "github.com/openshift/api/project/v1"
	v1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	"testing"
	"time"
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
	if err := createTestingIDP(ctx, http.DefaultClient); err != nil {
		t.Fatalf("error while creating testing idp: %w", err)
	}

	// get console master url
	rhmi, err := getRHMI(ctx)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	masterURL := rhmi.Spec.MasterURL

	// get oauth route
	oauthRoute := &v1.Route{}
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: resources.OpenshiftOAuthRouteName, Namespace: resources.OpenshiftAuthenticationNamespace}, oauthRoute); err != nil {
		t.Fatal("error getting Openshift Oauth Route: ", err)
	}

	// get rhmi developer user tokens
	openshiftHTTPClient, err := resources.DoAuthOpenshiftUser(masterURL, "test-user-0", DefaultPassword)
	if err != nil {
		t.Fatalf("error occured trying to get token : %v", err)
	}

	openshiftClient := &resources.OpenshiftClient{HTTPClient:openshiftHTTPClient}

	// test rhmi developer projects are as expected
	fuseNamespace := fmt.Sprintf("%sfuse", NamespacePrefix)
	if err := testRHMIDeveloperProjects(masterURL, fuseNamespace, openshiftClient); err != nil {
		t.Fatalf("test failed - %v", err)
	}

	// get fuse pods for rhmi developer
	podlist, err := openshiftClient.DoOpenshiftGetPodsForNamespacePods(masterURL, fuseNamespace)
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
			resp, err := openshiftClient.DoOpenshiftGetRequest(fmt.Sprintf("%s/%s/%s/log?%s", masterURL, resources.PathFusePods, p.Name, lv.Encode()), "")
			if err != nil {
				t.Fatalf("error occurred making oc get request: %v", err)
			}
			if resp.StatusCode != 200 {
				t.Fatalf("test-failed - status code %d RHMI developer unable to access fuse logs in pod %s : %v", resp.StatusCode, p.Name, err)
			}
		}
	}
}

func testRHMIDeveloperProjects(masterURL, fuseNamespace string, openshiftClient *resources.OpenshiftClient) error {
	var rhmiDevfoundProjects *projectv1.ProjectList
	err := wait.PollImmediate(time.Second*5, time.Minute*1, func() (done bool, err error) {
		// get projects for rhmi developer
		rhmiDevfoundProjects, err = openshiftClient.DoOpenshiftGetProjects(masterURL)
		if err != nil {
			return false, fmt.Errorf("error occured while getting user projects : %w", err)
		}

		// check if projects are as expected for rhmi developer
		if len(rhmiDevfoundProjects.Items) != expectedRhmiDeveloperProjectCount {
			return false, fmt.Errorf("found rhmi developer project count : %d expected rhmi-developer project count : %d", len(rhmiDevfoundProjects.Items), expectedRhmiDeveloperProjectCount)
		}

		foundNamespace := rhmiDevfoundProjects.Items[0].Name
		if foundNamespace != fuseNamespace {
			return true, fmt.Errorf("found rhmi developer project: %s expected rhmi developer project : %s", foundNamespace, fuseNamespace)
		}

		return true, nil
	})
	if err != nil {
		return fmt.Errorf("rhmi developer projects failure - %w", err)
	}
	return nil
}
