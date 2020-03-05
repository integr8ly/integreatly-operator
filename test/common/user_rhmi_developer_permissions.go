package common

import (
	goctx "context"
	"fmt"
	"github.com/google/go-querystring/query"
	v1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

const (
	expectedRhmiDeveloperNamespace    = "redhat-rhmi-fuse"
	expectedRhmiDeveloperProjectCount = 1
	expectedFusePodCount              = 6
)

func TestRHMIDeveloperUserPermissions(t *testing.T, ctx *TestingContext) {
	// get console master url
	rhmi, err := getRHMI(ctx)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	masterURL := rhmi.Spec.MasterURL

	// get oauth route
	oauthRoute := &v1.Route{}
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: openshiftOAuthRouteName, Namespace: openshiftAuthenticationNamespace}, oauthRoute); err != nil {
		t.Fatal("error getting Openshift Oauth Route: ", err)
	}

	// get rhmi developer user tokens
	rhmiDevUserToken, err := doAuthOpenshiftUser(oauthRoute.Spec.Host, masterURL, defaultIDP, "test-user01", "Password1")
	if err != nil {
		t.Fatalf("error occured trying to get token : %v", err)
	}

	// get projects for rhmi developer
	rhmiDevfoundProjects, err := doOpenshiftGetProjects(masterURL, rhmiDevUserToken)
	if err != nil {
		t.Fatalf("error occured while getting user projects : %v", err)
	}

	// check if projects are as expected for rhmi developer
	for projectCount, p := range rhmiDevfoundProjects.Items {
		t.Log(fmt.Sprintf("found rhmi-developer project - %s", p.Name))
		if projectCount >= expectedRhmiDeveloperProjectCount {
			t.Fatal(fmt.Sprintf("test failed - project count for rhmi-developer exceeded expected 1"))
		}
		if p.Name != expectedRhmiDeveloperNamespace {
			t.Fatal(fmt.Sprintf("test failed - found project for rhmi-developer does not match expected : %s", expectedRhmiDeveloperNamespace))
		}
	}
	t.Log("test-passed - found projects for rhmi-developer are as expected")

	// get fuse pods for rhmi developer
	podlist, err := doOpenshiftGetNamespacePods(masterURL, pathFusePods, rhmiDevUserToken)
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
	t.Logf("test-passed - found expected %d running pods in fuse namespace", expectedFusePodCount)

	// log through rhmi developer fuse podlist
	for _, p := range podlist.Items {
		if p.Status.Phase == "Running" {
			logOpt := LogOptions{p.Spec.Containers[0].Name, "false", "10"}
			lv, err := query.Values(logOpt)
			if err != nil {
				t.Fatal(err)
			}
			// verify an rhmi developer can access the pods logs
			resp, err := doOpenshiftGetRequest(fmt.Sprintf("%s/%s/%s/log?%s", masterURL, pathFusePods, p.Name, lv.Encode()), "", rhmiDevUserToken)
			if err != nil {
				t.Fatalf("error occured while making Openshift request: %v", err)
			}
			if resp.StatusCode != 200 {
				t.Fatalf("test-failed - rhmi devolper unable to access fuse logs at %s, error : %v", p.Name, err)
			}
		}
	}
}
