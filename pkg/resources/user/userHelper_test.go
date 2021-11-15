package user

import (
	"context"
	"fmt"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	"time"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	userv1 "github.com/openshift/api/user/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testIdentity = "test-identity"
	testEmail    = "test@email.com"
)

func TestGetUserEmailFromIdentity(t *testing.T) {

	scheme := runtime.NewScheme()
	err := userv1.AddToScheme(scheme)

	if err != nil {
		t.Fatalf("Error creating build scheme")
	}

	tests := []struct {
		Name          string
		FakeClient    k8sclient.Client
		User          userv1.User
		Identities    userv1.IdentityList
		ExpectedEmail string
		ExpectedError bool
	}{
		{
			Name:       "Test get email from identity",
			FakeClient: fake.NewFakeClientWithScheme(scheme),
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
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			got := GetUserEmailFromIdentity(context.TODO(), tt.FakeClient, tt.User, tt.Identities)

			if got != tt.ExpectedEmail {
				t.Errorf("GetUserEmailFromIdentity() got = %v, want %v", got, tt.ExpectedEmail)
			}
		})
	}
}

func TestGetUsersInActiveIDPs(t *testing.T) {

	scheme := runtime.NewScheme()
	err := userv1.AddToScheme(scheme)
	err = configv1.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("Error creating build scheme")
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
			FakeClient: fake.NewFakeClientWithScheme(scheme,
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
					userv1.User{
						ObjectMeta: v1.ObjectMeta{
							Name: "exists",
						},
						Identities: []string{"active-idp"},
					},
					userv1.User{
						ObjectMeta: v1.ObjectMeta{
							Name: "exits with no identity",
						},
						Identities: []string{""},
					},
				},
			},
			ExpectError: false,
		},
		{
			Name:       "Test orphaned identities have no side affect",
			FakeLogger: getLogger(),
			FakeClient: fake.NewFakeClientWithScheme(scheme,
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
					userv1.User{
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
			FakeClient: fake.NewFakeClientWithScheme(scheme, &userv1.Identity{
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
					userv1.User{
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
			if got := SetUserNameAsEmail(tt.args.userName); got != tt.want {
				t.Errorf("SetUserNameAsEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUsersReturnedByProvider(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = userv1.AddToScheme(scheme)

	tests := []struct {
		Name       string
		FakeClient k8sclient.Client
		Assertion  func(userv1.UserList) error
	}{
		{
			Name: "Test that users are returned correctly",
			FakeClient: fake.NewFakeClientWithScheme(scheme,
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

func assertThatUsersAreReturnedCorrectly(usersList userv1.UserList) error {
	if len(usersList.Items) != 2 {
		return fmt.Errorf("not all users have been found, expected amount of users is 2, actual is %v", len(usersList.Items))
	}

	return nil
}

func TestGetMultitenantUsers(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = userv1.AddToScheme(scheme)

	tests := []struct {
		Name           string
		FakeClient     k8sclient.Client
		InstallationCR *integreatlyv1alpha1.RHMI
		Assertion      func(users []MultiTenantUser) error
	}{
		{
			Name: "Test that users are returned correctly",
			FakeClient: fake.NewFakeClientWithScheme(scheme,
				&userv1.UserList{
					Items: []userv1.User{
						{
							ObjectMeta: v1.ObjectMeta{
								CreationTimestamp: metav1.Time{Time: time.Date(2021, time.March, 01, 00, 01, 00, 00, time.UTC)},
								Name:              "test-1",
								UID:               types.UID("test-1"),
							},
						},
						{
							ObjectMeta: v1.ObjectMeta{
								CreationTimestamp: metav1.Time{Time: time.Date(2021, time.March, 01, 00, 01, 00, 00, time.UTC)},
								Name:              "test-2",
								UID:               types.UID("test-2"),
							},
						},
						{
							ObjectMeta: v1.ObjectMeta{
								CreationTimestamp: metav1.Time{Time: time.Date(2021, time.March, 01, 00, 01, 00, 00, time.UTC)},
								Name:              "test-3",
								UID:               types.UID("test-3"),
							},
						},
					},
				},
			),
			InstallationCR: &integreatlyv1alpha1.RHMI{
				ObjectMeta: v1.ObjectMeta{
					CreationTimestamp: metav1.Time{Time: time.Date(2021, time.March, 01, 00, 00, 00, 00, time.UTC)},
				},
			},
			Assertion: confirmThatCorrectNumberOfUsersIsReturned,
		},
		{
			Name: "Test that users email addresses are setup correctly",
			FakeClient: fake.NewFakeClientWithScheme(scheme,
				&userv1.UserList{
					Items: []userv1.User{
						{
							ObjectMeta: v1.ObjectMeta{
								CreationTimestamp: metav1.Time{Time: time.Date(2021, time.March, 01, 00, 01, 00, 00, time.UTC)},
								Name:              "test-1",
								UID:               types.UID("test-1"),
							},
						},
						{
							ObjectMeta: v1.ObjectMeta{
								CreationTimestamp: metav1.Time{Time: time.Date(2021, time.March, 01, 00, 01, 00, 00, time.UTC)},
								Name:              "test-2",
								UID:               types.UID("test-2"),
							},
						},
						{
							ObjectMeta: v1.ObjectMeta{
								CreationTimestamp: metav1.Time{Time: time.Date(2021, time.March, 01, 00, 01, 00, 00, time.UTC)},
								Name:              "test-3",
								UID:               types.UID("test-3"),
							},
						},
					},
				},
				&userv1.IdentityList{
					Items: []userv1.Identity{
						{
							ObjectMeta: v1.ObjectMeta{
								Name: "testIdp:test-1",
							},
							User: corev1.ObjectReference{
								Name: "test-1",
								UID:  types.UID("test-1"),
							},
							Extra: map[string]string{
								"email": "test1email@email.com",
							},
							ProviderName: "devsandbox",
						},
						{
							ObjectMeta: v1.ObjectMeta{
								Name: "testIdp:test-2",
							},
							User: corev1.ObjectReference{
								Name: "test-2",
								UID:  types.UID("test-2"),
							},
							ProviderName: "devsandbox",
						},
					},
				},
			),
			InstallationCR: &integreatlyv1alpha1.RHMI{
				ObjectMeta: v1.ObjectMeta{
					CreationTimestamp: metav1.Time{Time: time.Date(2021, time.March, 01, 00, 00, 00, 00, time.UTC)},
				},
			},
			Assertion: confirmThatUsersHaveCorrectEmailAddressesSet,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			multitenantUsers, err := GetMultiTenantUsers(context.TODO(), tt.FakeClient, tt.InstallationCR)
			if err != nil {
				t.Fatalf("Failed test with: %v", err)
			}

			if err := tt.Assertion(multitenantUsers); err != nil {
				t.Fatalf("Failed assertion: %v", err)
			}

		})
	}
}

func TestTenantCreationTimeLogic(t *testing.T) {

	tests := []struct {
		Name           string
		Installation   *integreatlyv1alpha1.RHMI
		User           userv1.User
		ExpectedStatus bool
	}{
		{
			Name: "Test that a user is created past RHOAM installation",
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: v1.ObjectMeta{
					CreationTimestamp: metav1.Time{Time: time.Date(2021, time.April, 01, 00, 00, 00, 00, time.UTC)},
				},
			},
			User: userv1.User{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{Time: time.Date(2021, time.April, 01, 00, 01, 00, 00, time.UTC)},
				},
			},
			ExpectedStatus: true,
		},
		{
			Name: "Test that a user is create pre installation of RHOAM",
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: v1.ObjectMeta{
					CreationTimestamp: metav1.Time{Time: time.Date(2021, time.April, 01, 00, 00, 00, 00, time.UTC)},
				},
			},
			User: userv1.User{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{Time: time.Date(2021, time.March, 01, 00, 01, 00, 00, time.UTC)},
				},
			},
			ExpectedStatus: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			isTenantCreatedAfterInstallation := isTenantCreatedAfterInstallation(&tt.User, tt.Installation)

			if isTenantCreatedAfterInstallation != tt.ExpectedStatus {
				t.Fatalf("Expected %v phase but got %v", tt.ExpectedStatus, isTenantCreatedAfterInstallation)
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
