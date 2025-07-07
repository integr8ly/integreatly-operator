package user

import (
	"context"
	"fmt"
	"testing"

	"github.com/integr8ly/integreatly-operator/utils"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	keycloak "github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"
	configv1 "github.com/openshift/api/config/v1"
	userv1 "github.com/openshift/api/user/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testIdentity = "test-identity"
	testEmail    = "test@email.com"
)

func TestGetUserEmailFromIdentity(t *testing.T) {

	tests := []struct {
		Name          string
		User          userv1.User
		Identities    userv1.IdentityList
		ExpectedEmail string
		ExpectedError bool
	}{
		{
			Name: "Test get email from identity",
			User: userv1.User{
				Identities: []string{testIdentity},
			},
			Identities: userv1.IdentityList{
				Items: []userv1.Identity{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: testIdentity,
						},
						Extra: map[string]string{"email": testEmail},
					},
				},
			},
			ExpectedEmail: testEmail,
			ExpectedError: false,
		},
		{
			Name: "Test user email not in identity",
			User: userv1.User{
				Identities: []string{testIdentity},
			},
			Identities: userv1.IdentityList{
				Items: []userv1.Identity{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "notTestIdentity",
						},
						Extra: map[string]string{"email": testEmail},
					},
				},
			},
			ExpectedEmail: "",
			ExpectedError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			got := GetEmailFromIdentity(tt.User, tt.Identities)

			if got != tt.ExpectedEmail {
				t.Errorf("GetEmailFromIdentity() got = %v, want %v", got, tt.ExpectedEmail)
			}
		})
	}
}

func TestGetUsersInActiveIDPs(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		Name          string
		FakeClient    k8sclient.Client
		FakeLogger    l.Logger
		ExpectedUsers *userv1.UserList
		ExpectError   bool
	}{
		{
			Name:       "Test get user with no associated identity",
			FakeLogger: getLogger(),
			FakeClient: utils.NewTestClient(scheme,
				&userv1.Identity{
					ObjectMeta: v1.ObjectMeta{
						Name: "active-idp",
					},
					ProviderName: "exists",
				},
				&userv1.User{
					ObjectMeta: v1.ObjectMeta{
						Name: "exists",
					},
					Identities: []string{"active-idp"},
				},
				&userv1.User{
					ObjectMeta: v1.ObjectMeta{
						Name: "exits with no identity",
					},
					Identities: []string{"missing-identity"},
				},
				&configv1.OAuth{
					ObjectMeta: v1.ObjectMeta{
						Name: "cluster",
					},
					Spec: configv1.OAuthSpec{
						IdentityProviders: []configv1.IdentityProvider{
							{Name: "exists"},
						},
					},
				},
			),
			ExpectedUsers: &userv1.UserList{
				TypeMeta: v1.TypeMeta{},
				ListMeta: v1.ListMeta{},
				Items: []userv1.User{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "exists",
						},
						Identities: []string{"active-idp"},
					},
				},
			},
			ExpectError: false,
		},
		{
			Name:       "Test orphaned identities have no side affect",
			FakeLogger: getLogger(),
			FakeClient: utils.NewTestClient(scheme,
				&userv1.Identity{
					ObjectMeta: v1.ObjectMeta{
						Name: "active-idp",
					},
					ProviderName: "exists",
				},
				&userv1.User{
					ObjectMeta: v1.ObjectMeta{
						Name: "exists",
					},
					Identities: []string{"active-idp"},
				},
				&userv1.Identity{
					ObjectMeta: v1.ObjectMeta{
						Name: "inactive-idp",
					},
					ProviderName: "non-existent",
				},
				&userv1.User{
					ObjectMeta: v1.ObjectMeta{
						Name: "non-existent",
					},
					Identities: []string{"inactive-idp"},
				},
				&userv1.Identity{
					ObjectMeta: v1.ObjectMeta{
						Name: "orphaned-identity-1",
					},
					ProviderName: "exist",
				},
				&userv1.Identity{
					ObjectMeta: v1.ObjectMeta{
						Name: "orphaned-identity-2",
					},
					ProviderName: "exist",
				},
				&configv1.OAuth{
					ObjectMeta: v1.ObjectMeta{
						Name: "cluster",
					},
					Spec: configv1.OAuthSpec{
						IdentityProviders: []configv1.IdentityProvider{
							{Name: "exists"},
						},
					},
				}),
			ExpectedUsers: &userv1.UserList{
				TypeMeta: v1.TypeMeta{},
				ListMeta: v1.ListMeta{},
				Items: []userv1.User{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "exists",
						},
						Identities: []string{"active-idp"},
					},
				},
			},
			ExpectError: false,
		},
		{
			Name:       "Test get user with active idp",
			FakeLogger: getLogger(),
			FakeClient: utils.NewTestClient(scheme, &userv1.Identity{
				ObjectMeta: v1.ObjectMeta{
					Name: "active-idp",
				},
				ProviderName: "exists",
			},
				&userv1.User{
					ObjectMeta: v1.ObjectMeta{
						Name: "exists",
					},
					Identities: []string{"active-idp"},
				},
				&userv1.Identity{
					ObjectMeta: v1.ObjectMeta{
						Name: "inactive-idp",
					},
					ProviderName: "non-existant",
				},
				&userv1.User{
					ObjectMeta: v1.ObjectMeta{
						Name: "non-existant",
					},
					Identities: []string{"inactive-idp"},
				},
				&configv1.OAuth{
					ObjectMeta: v1.ObjectMeta{
						Name: "cluster",
					},
					Spec: configv1.OAuthSpec{
						IdentityProviders: []configv1.IdentityProvider{
							{Name: "exists"},
						},
					},
				}),
			ExpectedUsers: &userv1.UserList{
				TypeMeta: v1.TypeMeta{},
				ListMeta: v1.ListMeta{},
				Items: []userv1.User{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "exists",
						},
						Identities: []string{"active-idp"},
					},
				},
			},
			ExpectError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			got, err := GetUsersInActiveIDPs(context.TODO(), tt.FakeClient, tt.FakeLogger)
			if (err != nil) != tt.ExpectError {
				t.Errorf("GetUsersInActiveIDPs() error = %v, ExpectedErr %v", err, tt.ExpectError)
				return
			}
			if len(tt.ExpectedUsers.Items) != len(got.Items) {
				t.Errorf("unexpected amount of found users, got %v expected %v", len(got.Items), len(tt.ExpectedUsers.Items))
			}
			for _, expectedUser := range tt.ExpectedUsers.Items {
				if !contains(got.Items, expectedUser) {
					t.Errorf("expected user: %v not found", expectedUser)
				}
			}
		})
	}
}

func contains(s []userv1.User, e userv1.User) bool {
	for _, a := range s {
		if a.Name == e.Name {
			return true
		}
	}
	return false
}

func TestAppendUpdateProfileActionForUserWithoutEmail(t *testing.T) {

	tests := []struct {
		Name                string
		KeyCloakUser        keycloak.KeycloakAPIUser
		AddedRequiredAction bool
	}{
		{
			Name: "Test Update Profile action is added for user with empty email",
			KeyCloakUser: keycloak.KeycloakAPIUser{
				Email:           "",
				RequiredActions: []string{},
			},
			AddedRequiredAction: true,
		},
		{
			Name: "Test Update Profile action is not added for user with email",
			KeyCloakUser: keycloak.KeycloakAPIUser{
				Email:           testEmail,
				RequiredActions: []string{},
			},
			AddedRequiredAction: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			AppendUpdateProfileActionForUserWithoutEmail(&tt.KeyCloakUser)
			if tt.AddedRequiredAction && len(tt.KeyCloakUser.RequiredActions) != 1 {
				t.Fatal("Expected user to be updated with required action but wasn't")
			}

			if !tt.AddedRequiredAction && len(tt.KeyCloakUser.RequiredActions) != 0 {
				t.Fatal("Expected user to not be updated with required action but was")
			}
		})
	}
}

func TestGetValidGeneratedUserName(t *testing.T) {

	tests := []struct {
		Name                  string
		KeyCloakUser          keycloak.KeycloakAPIUser
		ExpectedGeneratedName string
	}{
		{
			Name: "Test - Username is lower cased",
			KeyCloakUser: keycloak.KeycloakAPIUser{
				UserName: "TEST",
			},
			ExpectedGeneratedName: fmt.Sprintf("%s%s", GeneratedNamePrefix, "test"),
		},
		{
			Name: "Test - Username is lower cased and invalid characters replaced",
			KeyCloakUser: keycloak.KeycloakAPIUser{
				UserName: "TEST_USER@Example.com",
			},
			ExpectedGeneratedName: fmt.Sprintf("%s%s%s%s%s%s%s%s", GeneratedNamePrefix, "test", invalidCharacterReplacement, "user", invalidCharacterReplacement, "example", invalidCharacterReplacement, "com"),
		},
		{
			Name: "Test - Username replacement character is not added to the end of generated name",
			KeyCloakUser: keycloak.KeycloakAPIUser{
				UserName: "Tester01#",
			},
			ExpectedGeneratedName: fmt.Sprintf("%s%s", GeneratedNamePrefix, "tester01"),
		},
		{
			Name: "Test - UserId is added to generated name",
			KeyCloakUser: keycloak.KeycloakAPIUser{
				UserName: "Tester.01#",
				FederatedIdentities: []keycloak.FederatedIdentity{
					{
						UserID: "54d19771-aab6-49bb-913f-ce94e0ae5600",
					},
				},
			},
			ExpectedGeneratedName: fmt.Sprintf("%s%s%s%s", GeneratedNamePrefix, "tester", invalidCharacterReplacement, "01-54d19771-aab6-49bb-913f-ce94e0ae5600"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			if got := GetValidGeneratedUserName(tt.KeyCloakUser); got != tt.ExpectedGeneratedName {
				t.Errorf("GetValidGeneratedUserName() = %v, want %v", got, tt.ExpectedGeneratedName)
			}
		})
	}
}

func TestSetUserNameAsEmail(t *testing.T) {
	type args struct {
		userName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test: username returned if already contains @ character",
			args: args{userName: "test@test.com"},
			want: "test@test.com",
		},
		{
			name: "Test: username with default domain appended if does not contain @ character",
			args: args{userName: "test"},
			want: fmt.Sprintf("test%s", defaultEmailDomain),
		},
		{
			name: "Test: sanitise of user name with default domain",
			args: args{userName: "test&sanitise"},
			want: fmt.Sprintf("test-sanitise%s", defaultEmailDomain),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SetUserNameAsEmail(tt.args.userName)
			if err != nil {
				t.Fatalf("Failed test with: %v", err)
			}

			if got != tt.want {
				t.Errorf("SetUserNameAsEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUsersReturnedByProvider(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		Name       string
		FakeClient k8sclient.Client
		Assertion  func(userv1.UserList) error
	}{
		{
			Name: "Test that users are returned correctly",
			FakeClient: utils.NewTestClient(scheme,
				&userv1.UserList{
					Items: []userv1.User{
						{
							ObjectMeta: v1.ObjectMeta{
								Name: "test-1",
							},
							Identities: []string{
								"rhd:1243634215613",
								"someAwesomeIdentity:123456788",
							},
						},
						{
							ObjectMeta: v1.ObjectMeta{
								Name: "test-2",
							},
							Identities: []string{
								"someAwesomeIdentity:123456788",
							},
						},
						{
							ObjectMeta: v1.ObjectMeta{
								Name: "test-3",
							},
							Identities: []string{
								"someAwesomeIdentity:123456788",
								"someAwesomeIdentity2:421453151",
								"rhd:1243634215613",
							},
						},
					},
				},
			),
			Assertion: assertThatUsersAreReturnedCorrectly,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			usersList, err := GetUsersByProviderName(context.TODO(), tt.FakeClient, "rhd")
			if err != nil {
				t.Fatalf("Failed test with: %v", err)
			}

			if err := tt.Assertion(*usersList); err != nil {
				t.Fatalf("Failed assertion: %v", err)
			}

		})
	}
}

func TestGetIdentitiesByProviderName(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		Name                  string
		ProviderName          string
		ExpectedNumIdentities int
		FakeClient            k8sclient.Client
	}{
		{
			Name:                  "Test that identities are returned correctly when given correct provider name",
			ProviderName:          "testing-idp",
			ExpectedNumIdentities: 2,
			FakeClient: utils.NewTestClient(scheme,
				&userv1.UserList{
					Items: []userv1.User{
						{
							ObjectMeta: v1.ObjectMeta{
								Name: "test-1",
								UID:  "test-1",
							},
						},
						{
							ObjectMeta: v1.ObjectMeta{
								Name: "test-2",
								UID:  "test-2",
							},
						},
					},
				},
				&userv1.IdentityList{
					Items: []userv1.Identity{
						{
							ObjectMeta: v1.ObjectMeta{
								Name: "testing-idp:test-1",
							},
							User: corev1.ObjectReference{
								Name: "test-1",
								UID:  "test-1",
							},
							ProviderName: "testing-idp",
						},
						{
							ObjectMeta: v1.ObjectMeta{
								Name: "testing-idp:test-2",
							},
							User: corev1.ObjectReference{
								Name: "test-2",
								UID:  "test-2",
							},
							ProviderName: "testing-idp",
						},
					},
				},
			),
		},
		{
			Name:                  "Test that identities are returned correctly when given incorrect provider name",
			ProviderName:          "bad-provider-name",
			ExpectedNumIdentities: 0,
			FakeClient: utils.NewTestClient(scheme,
				&userv1.UserList{
					Items: []userv1.User{
						{
							ObjectMeta: v1.ObjectMeta{
								Name: "test-1",
								UID:  "test-1",
							},
						},
						{
							ObjectMeta: v1.ObjectMeta{
								Name: "test-2",
								UID:  "test-2",
							},
						},
					},
				},
				&userv1.IdentityList{
					Items: []userv1.Identity{
						{
							ObjectMeta: v1.ObjectMeta{
								Name: "testing-idp:test-1",
							},
							User: corev1.ObjectReference{
								Name: "test-1",
								UID:  "test-1",
							},
							ProviderName: "testing-idp",
						},
						{
							ObjectMeta: v1.ObjectMeta{
								Name: "testing-idp:test-2",
							},
							User: corev1.ObjectReference{
								Name: "test-2",
								UID:  "test-2",
							},
							ProviderName: "testing-idp",
						},
					},
				},
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			identitiesList, err := GetIdentitiesByProviderName(context.TODO(), tt.FakeClient, tt.ProviderName)
			if err != nil {
				t.Fatalf("Failed test with: %v", err)
			}

			if len(identitiesList.Items) != tt.ExpectedNumIdentities {
				t.Fatalf("incorrect number of Identities was returned, expected %v, got %v", tt.ExpectedNumIdentities, len(identitiesList.Items))
			}

		})
	}
}

func assertThatUsersAreReturnedCorrectly(usersList userv1.UserList) error {
	if len(usersList.Items) != 2 {
		return fmt.Errorf("not all users have been found, expected amount of users is 2, actual is %v", len(usersList.Items))
	}

	return nil
}

func TestGetMultitenantUsers(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		Name       string
		FakeClient k8sclient.Client
		Assertion  func(users []MultiTenantUser) error
	}{
		{
			Name: "Test that users are returned correctly",
			FakeClient: utils.NewTestClient(scheme,
				&userv1.UserList{
					Items: []userv1.User{
						{
							ObjectMeta: v1.ObjectMeta{
								Name:        "test-1",
								UID:         "test-1",
								Annotations: map[string]string{"tenant": "yes"},
							},
						},
						{
							ObjectMeta: v1.ObjectMeta{
								Name:        "test-2",
								UID:         "test-2",
								Annotations: map[string]string{"tenant": "yes"},
							},
						},
						{
							ObjectMeta: v1.ObjectMeta{
								Name:        "test-3",
								UID:         "test-3",
								Annotations: map[string]string{"tenant": "yes"},
							},
						},
					},
				},
			),
			Assertion: confirmThatCorrectNumberOfUsersIsReturned,
		},
		{
			Name: "Test that users email addresses are setup correctly",
			FakeClient: utils.NewTestClient(scheme,
				&userv1.UserList{
					Items: []userv1.User{
						{
							ObjectMeta: v1.ObjectMeta{
								Name:        "test-1",
								UID:         "test-1",
								Annotations: map[string]string{"tenant": "yes"},
							},
						},
						{
							ObjectMeta: v1.ObjectMeta{
								Name:        "test-2",
								UID:         "test-2",
								Annotations: map[string]string{"tenant": "yes"},
							},
						},
						{
							ObjectMeta: v1.ObjectMeta{
								Name:        "test-3",
								UID:         "test-3",
								Annotations: map[string]string{"tenant": "yes"},
							},
						},
					},
				},
				&userv1.IdentityList{
					Items: []userv1.Identity{
						{
							ObjectMeta: v1.ObjectMeta{
								Name:        "testIdp:test-1",
								Annotations: map[string]string{"tenant": "yes"},
							},
							User: corev1.ObjectReference{
								Name: "test-1",
								UID:  "test-1",
							},
							Extra: map[string]string{
								"email": "test1email@email.com",
							},
							ProviderName: "devsandbox",
						},
						{
							ObjectMeta: v1.ObjectMeta{
								Name:        "testIdp:test-2",
								Annotations: map[string]string{"tenant": "yes"},
							},
							User: corev1.ObjectReference{
								Name: "test-2",
								UID:  "test-2",
							},
							ProviderName: "devsandbox",
						},
					},
				},
			),
			Assertion: confirmThatUsersHaveCorrectEmailAddressesSet,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			multitenantUsers, err := GetMultiTenantUsers(context.TODO(), tt.FakeClient)
			if err != nil {
				t.Fatalf("Failed test with: %v", err)
			}

			if err := tt.Assertion(multitenantUsers); err != nil {
				t.Fatalf("Failed assertion: %v", err)
			}

		})
	}
}

func confirmThatCorrectNumberOfUsersIsReturned(users []MultiTenantUser) error {
	if len(users) != 3 {
		return fmt.Errorf("incorrect number of users returned, expected 3, got %v", len(users))
	}

	return nil
}

func confirmThatUsersHaveCorrectEmailAddressesSet(users []MultiTenantUser) error {
	user1Email := "test1email@email.com"
	user2Email := "test-2@rhmi.io"
	user3Email := "test-3@rhmi.io"

	if len(users) != 3 {
		return fmt.Errorf("incorrect number of users returned, expected 3, got %v", len(users))
	}

	for _, user := range users {
		if user.TenantName == "test-1" {
			if user.Email != user1Email {
				return fmt.Errorf("%v does not have correct email set, got: %v, expected: %v", user.Username, user.Email, user1Email)
			}
		}
		if user.TenantName == "test-2" {
			if user.Email != user2Email {
				return fmt.Errorf("%v does not have correct email set, got: %v, expected: %v", user.Username, user.Email, user2Email)
			}
		}
		if user.TenantName == "test-3" {
			if user.Email != user3Email {
				return fmt.Errorf("%v does not have correct email set, got: %v, expected: %v", user.Username, user.Email, user2Email)
			}
		}
	}

	return nil
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.ProductRHSSO})
}
