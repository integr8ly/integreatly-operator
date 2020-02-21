package rhsso

import (
	"context"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	userv1 "github.com/openshift/api/user/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
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
		ExpectedEmail string
		ExpectedError bool
	}{
		{
			Name: "Test get email from identity",
			FakeClient: fake.NewFakeClientWithScheme(scheme, &userv1.Identity{
				ObjectMeta: v1.ObjectMeta{
					Name: testIdentity,
				},
				Extra: map[string]string{"email": testEmail},
			}),
			User: userv1.User{
				Identities: []string{testIdentity},
			},
			ExpectedEmail: testEmail,
			ExpectedError: false,
		},
		{
			Name:       "Test error getting identity",
			FakeClient: fake.NewFakeClientWithScheme(scheme),
			User: userv1.User{
				Identities: []string{testIdentity},
			},
			ExpectedEmail: "",
			ExpectedError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			got, err := GetUserEmailFromIdentity(context.TODO(), tt.FakeClient, tt.User)
			if (err != nil) != tt.ExpectedError {
				t.Errorf("GetUserEmailFromIdentity() error = %v, ExpectedErr %v", err, tt.ExpectedError)
				return
			}
			if got != tt.ExpectedEmail {
				t.Errorf("GetUserEmailFromIdentity() got = %v, want %v", got, tt.ExpectedEmail)
			}
		})
	}
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
