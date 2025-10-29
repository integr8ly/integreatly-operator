package common

import (
	goctx "context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
	"k8s.io/apimachinery/pkg/util/wait"
)

func Test3ScaleUserPromotion(t TestingTB, ctx *TestingContext) {
	// To run this testcase for multiple test-users, set USER_NUMBERS to a string
	// in a "1,2,3" format
	userNumbers := os.Getenv("USER_NUMBERS")
	testUserNumbers := strings.Split(userNumbers, ",")

	if userNumbers == "" {
		t.Logf("env var USER_NUMBERS was not set, defaulting to 1")
		testUserNumbers = []string{"1"}
	}
	var (
		developerUsers     []string
		dedicatedAdminUser = fmt.Sprintf("%v%02d", defaultDedicatedAdminName, 1)
	)
	for _, numberString := range testUserNumbers {
		number, err := strconv.Atoi(numberString)
		if err != nil {
			t.Fatalf("`USER_NUMBERS` env variable doesn't have the proper format (e.g. '1,2,3') %v", err)
		}
		developerUsers = append(developerUsers, fmt.Sprintf("%v%02d", DefaultTestUserName, number))
	}

	if err := createTestingIDP(t, goctx.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Fatalf("error while creating testing idp: %v", err)
	}

	// get console master url
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	// Get the 3Scale host url from the rhmi status
	host := rhmi.Status.Stages[rhmiv1alpha1.InstallStage].Products[rhmiv1alpha1.Product3Scale].Host
	if host == "" {
		host = fmt.Sprintf("https://3scale-admin.%v", rhmi.Spec.RoutingSubdomain)
	}

	keycloakHost := rhmi.Status.Stages[rhmiv1alpha1.InstallStage].Products[rhmiv1alpha1.ProductRHSSO].Host

	if keycloakHost == "" {
		t.Fatalf("Failed to retrieve keycloak host from RHMI CR: %v", rhmi)
	}

	redirectUrl := fmt.Sprintf("%v/p/admin/dashboard", host)

	httpClient, err := NewTestingHTTPClient(ctx.KubeConfig)
	if err != nil {
		t.Fatal(err)
	}

	err = loginToThreeScale(t, host, dedicatedAdminUser, TestingIdpPassword, TestingIDPRealm, httpClient)
	if err != nil {
		t.Fatalf("Failed to log into 3Scale: %v", err)
	}

	err = waitForUserToBecome3ScaleAdmin(t, ctx, host, threescaleLoginUser)
	if err != nil {
		t.Fatalf("timout asserting 3scale user, %s, is admin for performing test: %s", threescaleLoginUser, err)
	}

	tsClient := resources.NewThreeScaleAPIClient(host, keycloakHost, redirectUrl, httpClient, ctx.Client, t)

	for _, developerUser := range developerUsers {
		t.Logf("Trying to log into 3scale for user: %s", developerUser)
		loginTo3ScaleAsDeveloper(t, developerUser, host, ctx)

		userId, err := tsClient.GetUserId(developerUser)
		if err != nil || userId == "" {
			t.Fatal("Failed to retrieve user id for ", developerUser, "userId: ", userId, err)
		}

		err = tsClient.SetUserAsAdmin(developerUser, fmt.Sprintf("%v@example.com", developerUser), userId)
		if err != nil {
			t.Fatal("Failed to set user as admin ", err)
		}

		// TODO: Waiting an arbitrary amount of time to verify that a 3Scale reconcile was complete
		// and did not result in reverting the promotion of test-user to admin
		// Change this when https://issues.redhat.com/browse/INTLY-7770 implemented
		err = wait.PollUntilContextTimeout(goctx.TODO(), time.Second*350, time.Minute*7, false, func(ctx goctx.Context) (done bool, err error) {

			isAdmin, err := tsClient.VerifyUserIsAdmin(userId)
			if err != nil {
				t.Fatal("Error attempting to verify that the user is an admin ", err)
			}
			if !isAdmin {
				t.Fatal("User reverted from admin back to member")
			}

			return true, nil
		})
		if err != nil {
			t.Fatal("Failed to verify that the user is an admin ", err)
		}
	}

}

// Login as a developer and create a separate HTTP client. This will mimic what would happen in reality
// i.e. 2 users using separate browsers.
func loginTo3ScaleAsDeveloper(t TestingTB, user string, host string, ctx *TestingContext) {

	httpClient, err := NewTestingHTTPClient(ctx.KubeConfig)
	if err != nil {
		t.Fatal(err)
	}

	err = loginToThreeScale(t, host, user, TestingIdpPassword, TestingIDPRealm, httpClient)
	if err != nil {
		t.Fatalf("Failed to log into 3Scale: %v", err)
	}
}
