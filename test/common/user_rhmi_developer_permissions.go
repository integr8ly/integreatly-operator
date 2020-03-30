package common

import (
	goctx "context"
	"crypto/tls"
	"fmt"
	"github.com/google/go-querystring/query"
	"github.com/integr8ly/integreatly-operator/test/resources"
	projectv1 "github.com/openshift/api/project/v1"
	v1 "github.com/openshift/api/route/v1"
	"golang.org/x/net/publicsuffix"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	"net/http/cookiejar"
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

	// get rhmi developer user tokens
	if err := resources.DoAuthOpenshiftUser(masterURL, "test-user-1", DefaultPassword, httpClient); err != nil {
		t.Fatalf("error occured trying to get token : %v", err)
	}

	openshiftClient := &resources.OpenshiftClient{HTTPClient: httpClient}

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
	// five minute time out needed to ensure users have been reconciled by RHMI operator
	err := wait.PollImmediate(time.Second*5, time.Minute*5, func() (done bool, err error) {
		// get projects for rhmi developer
		rhmiDevfoundProjects, err = openshiftClient.DoOpenshiftGetProjects(masterURL)
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
		return fmt.Errorf("unexpected developer project count : %d expected project count : %d , error occurred - %w", len(rhmiDevfoundProjects.Items), expectedRhmiDeveloperProjectCount, err)
	}

	foundNamespace := rhmiDevfoundProjects.Items[0].Name
	if foundNamespace != fuseNamespace {
		return fmt.Errorf("found rhmi developer project: %s expected rhmi developer project : %s", foundNamespace, fuseNamespace)
	}

	return nil
}
