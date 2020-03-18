package common

import (
	goctx "context"
	"fmt"
	"github.com/google/go-querystring/query"
	"github.com/integr8ly/integreatly-operator/test/resources"
	v1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

const (
	expectedRhmiDeveloperNamespace    = "redhat-rhmi-fuse"
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
	// get console master url
	rhmi, err := GetRHMI(ctx)
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
	rhmiDevUserToken, err := resources.DoAuthOpenshiftUser(oauthRoute.Spec.Host, masterURL, resources.DefaultIDP, "test-user01", "Password1")
	if err != nil {
		t.Fatalf("error occured trying to get token : %v", err)
	}

	// get projects for rhmi developer
	rhmiDevfoundProjects, err := resources.DoOpenshiftGetProjects(masterURL, rhmiDevUserToken)
	if err != nil {
		t.Fatalf("error occured while getting user projects : %v", err)
	}

	// check if projects are as expected for rhmi developer
	for projectCount, p := range rhmiDevfoundProjects.Items {
		if projectCount >= expectedRhmiDeveloperProjectCount {
			t.Fatalf("test failed - found rhmi developer project count : %s expected rhmi-developer project count : %s", projectCount, expectedRhmiDeveloperProjectCount)
		}
		if p.Name != expectedRhmiDeveloperNamespace {
			t.Fatalf("test failed - found rhmi developer project: %s expected rhmi developer project : %s", p.Name, expectedRhmiDeveloperNamespace)
		}
	}

	// get fuse pods for rhmi developer
	fuseNamespace := fmt.Sprintf("%s-fuse", NamespacePrefix)
	podlist, err := resources.DoOpenshiftGetPodsForNamespacePods(masterURL, fuseNamespace, rhmiDevUserToken)
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
			resp, err := resources.DoOpenshiftGetRequest(fmt.Sprintf("%s/%s/%s/log?%s", masterURL, resources.PathFusePods, p.Name, lv.Encode()), "", rhmiDevUserToken)
			if err != nil {
				t.Fatalf("error occurred making oc get request: %v", err)
			}
			if resp.StatusCode != 200 {
				t.Fatalf("test-failed - status code %d RHMI developer unable to access fuse logs in pod %s : %v", resp.StatusCode, p.Name, err)
			}
		}
	}
}
