package common

import (
	goctx "context"
	"fmt"
	"testing"
	"time"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
	"k8s.io/apimachinery/pkg/util/wait"
)

func Test3ScaleUserPromotion(t *testing.T, ctx *TestingContext) {
	var (
		developerUser      = fmt.Sprintf("%v%02d", DefaultTestUserName, 1)
		dedicatedAdminUser = fmt.Sprintf("%v%02d", defaultDedicatedAdminName, 1)
	)

	if err := createTestingIDP(t, goctx.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		// t.Fatalf("error while creating testing idp: %v", err)
		t.Skipf("flakey test error while creating testing idp: %v jira https://issues.redhat.com/browse/MGDAPI-935", err)
	}

	// get console master url
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		// t.Fatalf("error getting RHMI CR: %v", err)
		t.Skipf("flakey test error getting RHMI CR: %v jira https://issues.redhat.com/browse/MGDAPI-935", err)
	}

	// Get the 3Scale host url from the rhmi status
	host := rhmi.Status.Stages[v1alpha1.ProductsStage].Products[v1alpha1.Product3Scale].Host
	if host == "" {
		host = fmt.Sprintf("https://3scale-admin.%v", rhmi.Spec.RoutingSubdomain)
	}

	keycloakHost := rhmi.Status.Stages[v1alpha1.AuthenticationStage].Products[v1alpha1.ProductRHSSO].Host

	if keycloakHost == "" {
		// t.Fatalf("Failed to retrieve keycloak host from RHMI CR: %v", rhmi)
		t.Skipf("flakey test Failed to retrieve keycloak host from RHMI CR: %v skipping jira https://issues.redhat.com/browse/MGDAPI-935", rhmi)
	}

	redirectUrl := fmt.Sprintf("%v/p/admin/dashboard", host)

	loginTo3ScaleAsDeveloper(t, developerUser, host, ctx)

	httpClient, err := NewTestingHTTPClient(ctx.KubeConfig)
	if err != nil {
		// t.Fatal(err)
		t.Skipf("flakey test failed to create testing http client error: %v jira https://issues.redhat.com/browse/MGDAPI-935", err)
	}

	err = loginToThreeScale(t, host, dedicatedAdminUser, DefaultPassword, TestingIDPRealm, httpClient)
	if err != nil {
		t.Skip("Skipping due to known flaky behavior error, to be fixed ASAP.\nJIRA: https://issues.redhat.com/browse/INTLY-10087", err)
	}

	tsClient := resources.NewThreeScaleAPIClient(host, keycloakHost, redirectUrl, httpClient, ctx.Client, t)

	// Make sure 3Scale is available
	err = tsClient.Ping()
	if err != nil {
		// t.Fatal(err)
		t.Skipf("flakey test 3scale not available error: %v jira https://issues.redhat.com/browse/MGDAPI-935 ", err)
	}

	userId, err := tsClient.GetUserId(developerUser)
	if err != nil || userId == "" {
		// t.Fatal("Failed to retrieve user id for ", developerUser, "userId: ", userId, err)
		t.Skip("flakey test Failed to retrieve user id for ", developerUser, "userId: ", userId, err, "jira https://issues.redhat.com/browse/MGDAPI-935 ")
	}

	err = tsClient.SetUserAsAdmin(developerUser, fmt.Sprintf("%v@example.com", developerUser), userId)
	if err != nil {
		// t.Fatal("Failed to set user as admin ", err)
		t.Skip("flakey test failed to set user as admin: error", err, "jira https://issues.redhat.com/browse/MGDAPI-935 ")
	}

	// TODO: Waiting an arbitrary amount of time to verify that a 3Scale reconcile was complete
	// and did not result in reverting the promotion of test-user to admin
	// Change this when https://issues.redhat.com/browse/INTLY-7770 implemented
	_ = wait.Poll(time.Second*350, time.Minute*7, func() (done bool, err error) {

		isAdmin, err := tsClient.VerifyUserIsAdmin(userId)
		if err != nil {
			// t.Fatal("Error attempting to verify that the user is an admin ", err)
			t.Skip("flakey test Error attempting to verify that the user is an admin ", err, " jira https://issues.redhat.com/browse/MGDAPI-935 ")
		}
		if !isAdmin {
			// t.Fatal("User reverted from admin back to member")
			t.Skip("flakey test User reverted from admin back to member jira https://issues.redhat.com/browse/MGDAPI-935 ")
		}

		return true, nil
	})

}

// Login as a developer and create a separate HTTP client. This will mimic what would happen in reality
// i.e. 2 users using separate browsers.
func loginTo3ScaleAsDeveloper(t *testing.T, user string, host string, ctx *TestingContext) {

	httpClient, err := NewTestingHTTPClient(ctx.KubeConfig)
	if err != nil {
		t.Fatal(err)
	}

	err = loginToThreeScale(t, host, user, DefaultPassword, TestingIDPRealm, httpClient)
	if err != nil {
		t.Fatalf("Failed to log into 3Scale: %v", err)
	}
}
