package common

import (
	"context"
	"fmt"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	userv1 "github.com/openshift/api/user/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

const testUserName = "test-user99"
const dedicatedAdminsGroupName = "dedicated-admins"
const dedicatedAdminsRealmManagersGroupName = dedicatedAdminsGroupName + "/realm-managers"

var (
	testUser = &userv1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: testUserName,
		},
	}
	testUsersToCreate = []TestUser{
		{
			UserName:  testUserName,
			FirstName: "Test",
			LastName:  "User99",
		},
	}
)

func TestDedicatedAdminUsersSyncedSSO(t TestingTB, ctx *TestingContext) {
	goCtx := context.TODO()

	defer cleanUpTestDedicatedAdminUsersSyncedSSO(t, ctx)

	// Create Testing IDP
	if err := createTestingIDP(t, goCtx, ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Fatalf("Error while creating testing IDP: %v", err)
	}

	// Get RHMI CR details for test
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("Error getting RHMI CR: %v", err)
	}

	// Create OpenShift user
	if err := ctx.Client.Create(goCtx, testUser); err != nil {
		t.Fatalf("Error creating openshift user: %s", err)
	}
	t.Logf("OpenShift user %s created", testUser.Name)

	// List KeycloakUser CRs and validate there is no CR with username
	validateUserNotListedInKeyCloakCR(t, ctx, goCtx, testUser.Name)

	// Create user in RHSSO
	if err := createOrUpdateKeycloakUserCR(goCtx, ctx.Client, testUsersToCreate, rhmi.Name); err != nil {
		t.Fatalf("Failed to create KeycloakUser CR: %v", err)
	}
	t.Logf("Created KeycloakUser CR %s", testUserName)

	// Add user to the dedicated-admins group
	if err := createOrUpdateDedicatedAdminGroupCR(goCtx, ctx.Client, []string{testUserName}); err != nil {
		t.Fatalf("Failed to add user %s to group %s: %v", testUserName, dedicatedAdminsGroupName, err)
	}
	t.Logf("Added user %s to group %s", testUserName, dedicatedAdminsGroupName)

	generatedKU := &keycloak.KeycloakUser{}
	err = wait.Poll(time.Second*10, time.Minute*2, func() (done bool, err error) {
		err = ctx.Client.Get(
			goCtx,
			types.NamespacedName{
				Namespace: RHSSOUserProductNamespace,
				Name:      fmt.Sprintf("generated-%s-%s", testUser.Name, testUser.UID),
			},
			generatedKU,
		)
		if err != nil {
			switch err.(type) {
			case *errors.StatusError:
				statusErr := err.(*errors.StatusError)
				if statusErr.ErrStatus.Code == http.StatusNotFound {
					return false, nil
				}
				return true, statusErr
			default:
				return true, err
			}
		}
		userGroups := generatedKU.Spec.User.Groups
		if !contains(userGroups, dedicatedAdminsGroupName) ||
			!contains(userGroups, dedicatedAdminsRealmManagersGroupName) {
			t.Fatalf("Expected user with ID %s to be part of groups [%s, %s], got %s",
				testUser.UID,
				dedicatedAdminsGroupName,
				dedicatedAdminsRealmManagersGroupName,
				userGroups,
			)
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("Failed to retrieve generated KeycloakUser CR: %v", err)
	}
}

func cleanUpTestDedicatedAdminUsersSyncedSSO(t TestingTB, ctx *TestingContext) {
	t.Logf("Cleaning up resources created from B09 test")
	goCtx := context.TODO()

	// Ensure OpenShift user is deleted
	ctx.Client.Delete(goCtx, testUser)

	// Ensure KeycloakUser CR is deleted
	ctx.Client.Delete(goCtx, &keycloak.KeycloakUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", TestingIDPRealm, testUserName),
			Namespace: RHSSOProductNamespace,
		},
	})

	dedicatedAdminGroup := &userv1.Group{}
	ctx.Client.Get(goCtx, types.NamespacedName{Name: dedicatedAdminsGroupName}, dedicatedAdminGroup)

	controllerutil.CreateOrUpdate(goCtx, ctx.Client, dedicatedAdminGroup, func() error {
		for i, user := range dedicatedAdminGroup.Users {
			if user == testUserName {
				dedicatedAdminGroup.Users = removeIndex(dedicatedAdminGroup.Users, i)
			}
		}
		return nil
	})

	t.Logf("Finished cleaning up B09 test resources")
}

func removeIndex(slice []string, s int) []string {
	return append(slice[:s], slice[s+1:]...)
}
