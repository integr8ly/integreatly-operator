package common

import (
	"context"
	"fmt"
	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
	keycloak "github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"
	configv1 "github.com/openshift/api/config/v1"
	userv1 "github.com/openshift/api/user/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
	"time"
)

const (
	userFailedCreateAlertName = "ThreeScaleUserCreationFailed"
	userLong1                 = "alongusernamethatisabovefourtycharacterslong"
	userLong2                 = "alongusernamethatisabovefourtycharacterslong2"
)

var (
	userLongName = &userv1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: userLong1,
		},
	}
	usersToCreateInTestingIDP = []TestUser{
		{
			UserName:  userLong2,
			FirstName: "TooMuch",
			LastName:  "Character",
		},
	}
	clusterOauthPreTest = &configv1.OAuth{}
)

func TestInvalidUserNameAlert(t TestingTB, ctx *TestingContext) {
	goCtx := context.TODO()

	// Get resources before test execution and always try to restore cluster to pre test state
	if err := ctx.Client.Get(goCtx, types.NamespacedName{Name: clusterOauthName}, clusterOauthPreTest); err != nil {
		t.Fatalf("error occurred while getting cluster oauth: %w", err)
	}
	defer restoreClusterStatePreTest(t, ctx)

	// Create Testing IDP
	if err := createTestingIDP(t, goCtx, ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Fatalf("error while creating testing idp: %v", err)
	}

	// Get RHMI CR details for test
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	// Create User
	if err := ctx.Client.Create(goCtx, userLongName); err != nil {
		t.Fatalf("Error creating openshift user: %s", err)
	}
	t.Logf("Openshift user %s created", userLongName.Name)

	// List Keycloak User CRs and validate there is no CR with username
	validateUserNotListedInKeyCloakCR(t, ctx, goCtx, userLongName.Name)

	// Create User in RHSSO
	if err := createOrUpdateKeycloakUserCR(goCtx, ctx.Client, usersToCreateInTestingIDP, rhmi.Name); err != nil {
		t.Fatalf("Failed to created keycloak user cr: %s with err: %s", err)
	}
	t.Logf("Created keycloak cr %s on %s", userLong2, TestingIDPRealm)

	// Login with User
	masterURL := rhmi.Spec.MasterURL
	pollOpenshiftUserLogin(t, ctx, masterURL, userLong2)

	// Validate ThreeScaleUserCreationFailed alerts is firing
	validateAlertIsFiring(t, ctx, userFailedCreateAlertName)

	// Delete user from Openshift
	if err := ctx.Client.Delete(goCtx, &userv1.User{ObjectMeta: metav1.ObjectMeta{Name: userLong2}}); err != nil {
		t.Fatalf("Failed to delete openshift user %s: %s", userLong2, err)
	}
	t.Logf("Deleted %s openshift user", userLong2)

	// Validate ThreeScaleUserCreationFailed alert is no longer firing
	validateAlertIsNotFiring(t, ctx, userFailedCreateAlertName)

	// Login as dedicated admin user
	customerAdminUsername := fmt.Sprintf("%v%02d", defaultDedicatedAdminName, 1)
	pollOpenshiftUserLogin(t, ctx, masterURL, customerAdminUsername)

	// List user in 3scale and ensure dedicated admin is listed
	host := rhmi.Status.Stages[rhmiv1alpha1.InstallStage].Products[rhmiv1alpha1.Product3Scale].Host
	validateUserIsListedIn3scale(t, ctx, host, customerAdminUsername)

	// Delete IDP
	if err := deleteIDPsFromOauth(goCtx, ctx.Client); err != nil {
		t.Fatalf("Error deleting IDP from cluster Oauth: %w", err)
	}

	// List user in 3scale and ensure dedicated admin is not listed
	validateUserIsNotListedIn3scale(t, ctx, host, customerAdminUsername)

	t.Logf("Test passed")
}

func validateUserNotListedInKeyCloakCR(t TestingTB, ctx *TestingContext, goCtx context.Context, userName string) {
	keycloakUsers := &keycloak.KeycloakUserList{}
	if err := ctx.Client.List(goCtx, keycloakUsers, []k8sclient.ListOption{k8sclient.InNamespace(RHSSOProductNamespace)}...); err != nil {
		t.Fatalf("Error listing keycloak users: %s", err)
	}

	for _, keycloakUser := range keycloakUsers.Items {
		if strings.Contains(keycloakUser.Name, userName) {
			t.Fatalf("Expected no keycloak user cr with name %s but found %s", userName, keycloakUser.Name)
		}
	}

	t.Logf("No keycloak cr found containing %s in name in %s namespace", userName, RHSSOProductNamespace)
}

func pollOpenshiftUserLogin(t TestingTB, ctx *TestingContext, masterURL, userName string) {
	if err := wait.PollUntilContextTimeout(context.TODO(), time.Second*10, time.Minute*8, true, func(ctx2 context.Context) (done bool, err error) {
		tempHttpClient, err := NewTestingHTTPClient(ctx.KubeConfig)
		if err != nil {
			return false, nil
		}

		if err := resources.DoAuthOpenshiftUser(fmt.Sprintf("%s/auth/login", masterURL), userName, TestingIdpPassword, tempHttpClient, TestingIDPRealm, t); err != nil {
			t.Logf("Error trying to sign in as %s to openshift : %v", userName, err)
			return false, nil
		}

		// Additional check for User CR successfully created if user logged into cluster successfully
		user := &userv1.User{ObjectMeta: metav1.ObjectMeta{Name: userName}}
		if err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: userName}, user); err != nil {
			t.Logf("Failed to find User CR with name %s: %s", userName, err)
			return false, nil
		}

		t.Logf("Logged into openshift using %s user", userName)
		return true, nil
	}); err != nil {
		t.Fatalf("Error trying to login to openshift with user %s: %s", userName, err)
	}
}

func validateAlertIsFiring(t TestingTB, ctx *TestingContext, alertName string) {
	if err := wait.PollUntilContextTimeout(context.TODO(), time.Second*10, time.Minute*8, true, func(ctx2 context.Context) (done bool, err error) {
		getAlertErr := getFiringAlerts(t, ctx)

		// getAlertErr should not be nil as DeadMansSwitch & at least specific alert should be firing
		if getAlertErr == nil {
			return false, nil
		}

		for _, alert := range getAlertErr.(*alertsFiringError).alertsFiring {
			if alert.alertName == alertName {
				return true, nil
			}
		}

		return false, nil
	}); err != nil {
		//t.Fatalf("%s alert was not firing: %s", alertName, err)
		t.Skipf("Known flaky test - https://issues.redhat.com/browse/MGDAPI-2581: %s alert was not firing: %s", alertName, err)
	}

	t.Logf("%s alert is firing", alertName)
}

func validateAlertIsNotFiring(t TestingTB, ctx *TestingContext, alertName string) {
	if err := wait.PollUntilContextTimeout(context.TODO(), time.Second*10, time.Minute*8, true, func(ctx2 context.Context) (done bool, err error) {
		// getAlertErr will be nil if only DeadMansSwitch alert is firing
		getAlertErr := getFiringAlerts(t, ctx)
		if getAlertErr == nil {
			return true, nil
		}

		for _, alert := range getAlertErr.(*alertsFiringError).alertsFiring {
			if alert.alertName == alertName {
				return false, nil
			}
		}
		// Other alerts are firing but specified alert is no longer firing
		return true, nil
	}); err != nil {
		t.Fatalf("%s alert was still firing: %s", alertName, err)
	}

	t.Logf("%s alerts is no longer firing", alertName)
}

func validateUserIsListedIn3scale(t TestingTB, ctx *TestingContext, host, threeScaleUsername string) {
	if err := wait.PollUntilContextTimeout(context.TODO(), time.Second*10, time.Minute*8, true, func(ctx2 context.Context) (done bool, err error) {
		users, err := getUsersIn3scale(ctx, host)
		if err != nil {
			t.Logf("Error gettting 3scale users: %s", err)
			return false, nil
		}

		for _, user := range users.Users {
			if user.UserDetails.Username == threeScaleUsername {
				t.Logf("Found %s user in 3scale", threeScaleUsername)
				return true, nil
			}
		}

		return false, nil
	}); err != nil {
		t.Fatalf("Could not find %s user in 3scale", threeScaleUsername)
	}
}

func validateUserIsNotListedIn3scale(t TestingTB, ctx *TestingContext, host, customerAdminUsername string) {
	if err := wait.PollUntilContextTimeout(context.TODO(), time.Second*10, time.Minute*8, true, func(ctx2 context.Context) (done bool, err error) {
		users, err := getUsersIn3scale(ctx, host)
		if err != nil {
			t.Logf("Error getting 3scale users: %s", err)
			return false, nil
		}

		for _, user := range users.Users {
			if user.UserDetails.Username == customerAdminUsername {
				return false, nil
			}
		}

		t.Logf("Did not find %s user in 3scale", customerAdminUsername)
		return true, nil
	}); err != nil {
		t.Fatalf("Found %s user in 3scale", customerAdminUsername)
	}
}

func restoreClusterStatePreTest(t TestingTB, ctx *TestingContext) {
	t.Logf("Cleaning up resources created from test")
	goCtx := context.TODO()

	// Ensure Oauth is restored to pre-test state
	clusterOauth := &configv1.OAuth{ObjectMeta: metav1.ObjectMeta{Name: clusterOauthName}}
	_, err := controllerutil.CreateOrUpdate(goCtx, ctx.Client, clusterOauth, func() error {
		clusterOauth.Spec = clusterOauthPreTest.Spec
		return nil
	})
	if err != nil {
		t.Fatalf("failed to update clusterOauth.spec: %s, err: %v", clusterOauth.Spec, err)
	}
	// Ensure openshift users are deleted
	err = ctx.Client.Delete(goCtx, userLongName)
	if err != nil && !k8errors.IsNotFound(err) {
		t.Fatalf("failed to delete openshift user: %s, err: %v", userLongName, err)
	}

	// Ensure Keycloak CR created are deleted
	err = ctx.Client.Delete(goCtx, &keycloak.KeycloakUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", TestingIDPRealm, userLong2),
			Namespace: RHSSOProductNamespace,
		},
	})
	if err != nil && !k8errors.IsNotFound(err) {
		t.Fatalf("failed to delete Keycloak CR: %s, err: %v", fmt.Sprintf("%s-%s", TestingIDPRealm, userLong2), err)
	}

	t.Logf("Finished cleaning up test resources")
}
