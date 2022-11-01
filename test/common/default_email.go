package common

import (
	"fmt"
	"strings"
	"time"

	goctx "context"

	keycloakv1 "github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"
	userv1 "github.com/openshift/api/user/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	usernameNoEmail              = "autotest-user01"
	usernameWithEmail            = "autotest-user02"
	usernameUsingEmail           = "autotest-user03@test.com"
	usernameRequireSanitiseEmail = "autotest$user04"
	existingEmail                = "autotest-nondefault-user02@hotmail.com"
	defaultDomain                = "@rhmi.io"
)

// TestDefaultUserEmail verifies that a user syncronized from the IDP have a
// default email adress if no email is present from the IDP.
//
// Verify that the email address is generated as <username>@rhmi.io
func TestDefaultUserEmail(t TestingTB, ctx *TestingContext) {
	err := createTestingIDP(t, goctx.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts)
	if err != nil {
		t.Fatalf("Error occurred creating testing IDP: %v", err)
	}

	// Create user with no email
	// the default email for this user will be <user name>@rhmi.io
	userNoEmail, identityNoEmail, err := createUserTestingIDP(ctx, usernameNoEmail, nil)
	if err != nil {
		t.Fatalf("Unexpected error creating User: %v", err)
	}
	// Clean up the user resource
	defer func(ctx *TestingContext, user *userv1.User, identity *userv1.Identity) {
		if err := deleteUser(ctx, user, identity); err != nil {
			t.Fatal(err)
		}
	}(ctx, userNoEmail, identityNoEmail)

	// Create user with email
	// different that the default generated email
	userWithEmail, identityWithEmail, err := createUserTestingIDP(ctx, usernameWithEmail, func(identity *userv1.Identity) {
		identity.Extra = map[string]string{
			"email": existingEmail,
		}
	})
	if err != nil {
		t.Fatalf("Unexpected error creating User: %v", err)
	}

	// Cleanup the user resource
	defer func(ctx *TestingContext, user *userv1.User, identity *userv1.Identity) {
		if err := deleteUser(ctx, user, identity); err != nil {
			t.Fatal(err)
		}
	}(ctx, userWithEmail, identityWithEmail)

	// Create user that uses email as username
	userUsingEmailAsUsername, identityWithEmailAsUserName, err := createUserTestingIDP(ctx, usernameUsingEmail, nil)
	if err != nil {
		t.Fatalf("Unexpected error creating User: %v", err)
	}

	// Cleanup the user resource
	defer func(ctx *TestingContext, user *userv1.User, identity *userv1.Identity) {
		if err := deleteUser(ctx, user, identity); err != nil {
			t.Fatal(err)
		}
	}(ctx, userUsingEmailAsUsername, identityWithEmailAsUserName)

	// Create user with username that requires sanitization
	userRequireSanitiseEmail, identityRequireSanitiseEmail, err := createUserTestingIDP(ctx, usernameRequireSanitiseEmail, nil)
	if err != nil {
		t.Fatalf("Unexpected error creating User: %v", err)
	}

	// Cleanup the user resource
	defer func(ctx *TestingContext, user *userv1.User, identity *userv1.Identity) {
		if err := deleteUser(ctx, user, identity); err != nil {
			t.Fatal(err)
		}
	}(ctx, userRequireSanitiseEmail, identityRequireSanitiseEmail)

	rhssoNamespace := fmt.Sprintf("%srhsso", NamespacePrefix)

	// Get the keycloak CR for each user
	keycloakUser1, err := waitForKeycloakUser(ctx, 5*time.Minute, rhssoNamespace, usernameNoEmail)
	if err != nil {
		t.Fatalf("Unexpected error querying KeycloakUser %s: %v", usernameNoEmail, err)
	}

	keycloakUser2, err := waitForKeycloakUser(ctx, 5*time.Minute, rhssoNamespace, usernameWithEmail)
	if err != nil {
		t.Fatalf("Unexpected error querying KeycloakUser %s: %v", usernameWithEmail, err)
		return
	}

	keycloakUser3, err := waitForKeycloakUser(ctx, 5*time.Minute, rhssoNamespace, usernameUsingEmail)
	if err != nil {
		t.Fatalf("Unexpected error querying KeycloakUser %s: %v", usernameUsingEmail, err)
		return
	}

	keycloakUser4, err := waitForKeycloakUser(ctx, 5*time.Minute, rhssoNamespace, usernameRequireSanitiseEmail)
	if err != nil {
		t.Fatalf("Unexpected error querying KeycloakUser %s: %v", usernameRequireSanitiseEmail, err)
		return
	}

	// Assert that the user with no email has the default generated email
	expectedEmail := fmt.Sprintf("%s%s", usernameNoEmail, defaultDomain)
	if keycloakUser1.Spec.User.Email != expectedEmail {
		t.Errorf("Unexpected email for generated KeycloakUser: Expected %s, got %s", expectedEmail, keycloakUser1.Spec.User.Email)
	}

	// Assert that the user with email has its own email
	if keycloakUser2.Spec.User.Email != existingEmail {
		t.Errorf("Unexpected email for generated KeycloakUser: Expected %s, got %s", existingEmail, keycloakUser2.Spec.User.Email)
	}

	// Assert that the user using email as username does not get appended with default domain and just uses username
	if keycloakUser3.Spec.User.Email != usernameUsingEmail {
		t.Errorf("Unexpected email for generated KeycloakUser: Expected %s, got %s", usernameUsingEmail, keycloakUser3.Spec.User.Email)
	}

	// Assert that the username is sanitised when using as email
	if keycloakUser4.Spec.User.Email != fmt.Sprintf("%s%s", strings.Replace(usernameRequireSanitiseEmail, "$", "-", -1), defaultDomain) {
		t.Errorf("Unexpected email for generated KeycloakUser: Expected %s, got %s", usernameUsingEmail, keycloakUser4.Spec.User.Email)
	}
}

func createUserTestingIDP(ctx *TestingContext, userName string, mutateIdentity func(*userv1.Identity)) (*userv1.User, *userv1.Identity, error) {
	identityName := fmt.Sprintf("%s:%s", TestingIDPRealm, userName)

	identity := &userv1.Identity{
		ObjectMeta: v1.ObjectMeta{
			Name: identityName,
		},
		ProviderName:     TestingIDPRealm,
		ProviderUserName: userName,
	}

	if mutateIdentity != nil {
		mutateIdentity(identity)
	}

	if err := ctx.Client.Create(goctx.TODO(), identity); err != nil {
		if !k8serr.IsAlreadyExists(err) {
			return nil, nil, err
		}
	}

	var user = &userv1.User{
		ObjectMeta: v1.ObjectMeta{
			Name: userName,
		},
		Identities: []string{
			identityName,
		},
	}

	if err := ctx.Client.Create(goctx.TODO(), user); err != nil {
		return nil, nil, err
	}

	return user, identity, nil
}

func deleteUser(ctx *TestingContext, user *userv1.User, identity *userv1.Identity) error {
	// Delete the user
	if err := ctx.Client.Delete(goctx.TODO(), user); err != nil {
		return err
	}

	// Delete the identity
	return ctx.Client.Delete(goctx.TODO(), identity)
}

func waitForKeycloakUser(ctx *TestingContext, timeout time.Duration, namespace, userName string) (*keycloakv1.KeycloakUser, error) {
	began := time.Now()

	for {
		// If it timed out, return an error
		if time.Now().After(began.Add(timeout)) {
			return nil, fmt.Errorf("Timeout after %v", timeout)
		}

		// Get the list of users in the RHSSO namespace
		list := &keycloakv1.KeycloakUserList{}
		err := ctx.Client.List(goctx.TODO(), list, k8sclient.InNamespace(namespace))

		// If an error occurred, return the error
		if err != nil {
			return nil, err
		}

		// Look for the matching user in the user list and send it if it's
		// found
		for _, keycloakUser := range list.Items {
			if keycloakUser.Spec.User.UserName == userName {
				return &keycloakUser, nil
			}
		}
	}
}
