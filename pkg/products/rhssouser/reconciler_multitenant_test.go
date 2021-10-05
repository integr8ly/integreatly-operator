package rhssouser

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	keycloakCommon "github.com/integr8ly/keycloak-client/pkg/common"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	consolev1 "github.com/openshift/api/console/v1"
	userv1 "github.com/openshift/api/user/v1"
	fakeoauthClient "github.com/openshift/client-go/oauth/clientset/versioned/fake"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconciler_full_multitenant_reconcile(t *testing.T) {

	installation := &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{},
		TypeMeta:   metav1.TypeMeta{},
	}

	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	kc := &keycloak.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: defaultNamespace,
		},
	}

	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tenant-oauth-client-secrets",
			Namespace: defaultOperatorNamespace,
		},
		Data: map[string][]byte{
			"staging":    bytes.NewBufferString("test").Bytes(),
			"test-user1": bytes.NewBufferString("test").Bytes(),
			"test-user2": bytes.NewBufferString("test").Bytes(),
			"test-user3": bytes.NewBufferString("test").Bytes(),
		},
	}

	cases := []struct {
		Name                  string
		ExpectError           bool
		ExpectedStatus        integreatlyv1alpha1.StatusPhase
		ExpectedError         string
		AssertFunc            assertFunc
		FakeConfig            *config.ConfigReadWriterMock
		FakeClient            k8sclient.Client
		FakeOauthClient       oauthClient.OauthV1Interface
		FakeMPM               *marketplace.MarketplaceInterfaceMock
		Installation          *integreatlyv1alpha1.RHMI
		Recorder              record.EventRecorder
		ApiUrl                string
		KeycloakClientFactory keycloakCommon.KeycloakClientFactory
	}{
		{
			Name:                  "Confirm staging users and realm are created",
			ExpectedStatus:        integreatlyv1alpha1.PhaseCompleted,
			ExpectError:           false,
			AssertFunc:            confirmStagingUsers,
			FakeClient:            fakeclient.NewFakeClientWithScheme(scheme, kc, getIdentities("1"), getIdentities("2"), getIdentities("3"), getKCRealmList(), oauthClientSecrets),
			FakeOauthClient:       fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:            basicConfigMock(),
			FakeMPM:               &marketplace.MarketplaceInterfaceMock{},
			Installation:          installation,
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
		},
		{
			Name:                  "Confirm console links are present for users",
			ExpectedStatus:        integreatlyv1alpha1.PhaseCompleted,
			ExpectError:           false,
			AssertFunc:            confirmStagingConsoleLinkExist,
			FakeClient:            fakeclient.NewFakeClientWithScheme(scheme, kc, getIdentities("1"), getIdentities("2"), getIdentities("3"), getKCRealmList(), oauthClientSecrets),
			FakeOauthClient:       fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:            basicConfigMock(),
			FakeMPM:               &marketplace.MarketplaceInterfaceMock{},
			Installation:          installation,
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
		},
		{
			Name:                  "Confirm user realms are created",
			ExpectedStatus:        integreatlyv1alpha1.PhaseCompleted,
			ExpectError:           false,
			AssertFunc:            confirmUserRealmsAreCreated,
			FakeClient:            fakeclient.NewFakeClientWithScheme(scheme, kc, getIdentities("1"), getIdentities("2"), getIdentities("3"), getKCUsersList(), getKCRealmList(), oauthClientSecrets),
			FakeOauthClient:       fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:            basicConfigMock(),
			FakeMPM:               &marketplace.MarketplaceInterfaceMock{},
			Installation:          installation,
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
		},
		{
			Name:                  "Confirm users belong to their realms and not to staging realm after reconcile",
			ExpectedStatus:        integreatlyv1alpha1.PhaseCompleted,
			ExpectError:           false,
			AssertFunc:            confirmUserUsersBelongToCorrectRealm,
			FakeClient:            fakeclient.NewFakeClientWithScheme(scheme, kc, getIdentities("1"), getIdentities("2"), getIdentities("3"), getKCUsersList(), getKCRealmList(), oauthClientSecrets),
			FakeOauthClient:       fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:            basicConfigMock(),
			FakeMPM:               &marketplace.MarketplaceInterfaceMock{},
			Installation:          installation,
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
		},
		{
			Name:                  "Confirm console links after user creation are correct",
			ExpectedStatus:        integreatlyv1alpha1.PhaseCompleted,
			ExpectError:           false,
			AssertFunc:            confirmTenantsConsoleLinks,
			FakeClient:            fakeclient.NewFakeClientWithScheme(scheme, kc, getIdentities("1"), getIdentities("2"), getIdentities("3"), getKCUsersList(), getKCRealmList(), oauthClientSecrets),
			FakeOauthClient:       fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:            basicConfigMock(),
			FakeMPM:               &marketplace.MarketplaceInterfaceMock{},
			Installation:          installation,
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			testReconciler, err := NewReconciler(
				tc.FakeConfig,
				tc.Installation,
				tc.FakeOauthClient,
				tc.FakeMPM,
				tc.Recorder,
				tc.ApiUrl,
				tc.KeycloakClientFactory,
				getLogger(),
				localProductDeclaration,
			)
			if err != nil && err.Error() != tc.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
			}

			status, err := testReconciler.ReconcileMultiTenantUserSSO(context.TODO(), tc.FakeClient, kc)

			if err != nil && !tc.ExpectError {
				t.Fatalf("expected no errors, but got one: %v", err)
			}

			if err == nil && tc.ExpectError {
				t.Fatal("expected error but got none")
			}

			if status != tc.ExpectedStatus {
				t.Fatalf("Expected status: '%v', got: '%v'", tc.ExpectedStatus, status)
			}

			err = tc.AssertFunc(context.TODO(), tc.FakeClient)
			if err != nil {
				t.Fatal(err.Error())
			}
		})
	}
}

func getIdentities(userID string) *userv1.Identity {
	return &userv1.Identity{
		ObjectMeta: v1.ObjectMeta{
			Name: fmt.Sprintf("test-user%v", userID),
			Labels: map[string]string{
				"sso": StagingRealmName,
			},
		},
		User: corev1.ObjectReference{
			Name: fmt.Sprintf("test-user%v", userID),
			UID:  types.UID(userID),
		},
		ProviderName: "devsandbox",
	}
}

func getKCUsersList() *keycloak.KeycloakUserList {
	return &keycloak.KeycloakUserList{
		ListMeta: v1.ListMeta{},
		Items: []keycloak.KeycloakUser{
			{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-user1",
					Namespace: defaultNamespace,
					Labels: map[string]string{
						"sso": StagingRealmName,
					},
				},
				Spec: keycloak.KeycloakUserSpec{
					RealmSelector: &v1.LabelSelector{
						MatchLabels: map[string]string{
							"sso": StagingRealmName,
						},
					},
					User: keycloak.KeycloakAPIUser{
						ID:       "1",
						UserName: "test-user1",
						FederatedIdentities: []keycloak.FederatedIdentity{
							{IdentityProvider: "devsandbox", UserID: "1"},
						},
					},
				},
			},
			{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-user2",
					Namespace: defaultNamespace,
					Labels: map[string]string{
						"sso": StagingRealmName,
					},
				},
				Spec: keycloak.KeycloakUserSpec{
					RealmSelector: &v1.LabelSelector{
						MatchLabels: map[string]string{
							"sso": StagingRealmName,
						},
					},
					User: keycloak.KeycloakAPIUser{
						ID:       "2",
						UserName: "test-user2",
						FederatedIdentities: []keycloak.FederatedIdentity{
							{IdentityProvider: "devsandbox", UserID: "2"},
						},
					},
				},
			},
			{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-user3",
					Namespace: defaultNamespace,
					Labels: map[string]string{
						"sso": StagingRealmName,
					},
				},
				Spec: keycloak.KeycloakUserSpec{
					RealmSelector: &v1.LabelSelector{
						MatchLabels: map[string]string{
							"sso": StagingRealmName,
						},
					},
					User: keycloak.KeycloakAPIUser{
						ID:       "3",
						UserName: "test-user3",
						FederatedIdentities: []keycloak.FederatedIdentity{
							{IdentityProvider: "devsandbox", UserID: "3"},
						},
					},
				},
			},
		},
	}
}

func getKCRealmList() *keycloak.KeycloakRealmList {
	return &keycloak.KeycloakRealmList{
		Items: []keycloak.KeycloakRealm{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "master",
					Namespace: defaultNamespace,
				},
				Spec: keycloak.KeycloakRealmSpec{
					Realm: &keycloak.KeycloakAPIRealm{
						ID:            "master",
						Realm:         "master",
						Enabled:       true,
						DisplayName:   "master",
						EventsEnabled: boolPtr(true),
					},
				},
			},
		},
	}
}

type assertFunc func(context.Context, k8sclient.Client) error

var confirmStagingUsers = func(ctx context.Context, client k8sclient.Client) error {
	var users keycloak.KeycloakUserList
	var found = false

	listOptions := []k8sclient.ListOption{
		k8sclient.MatchingLabels(map[string]string{
			"sso": StagingRealmName,
		}),
	}

	// 3 users are to be expected
	err := client.List(ctx, &users, listOptions...)
	if err != nil {
		return err
	}

	size := len(users.Items)
	if size != 3 {
		return fmt.Errorf("Not all users have been created in staging realm")
	}

	kcRealms := &keycloak.KeycloakRealmList{}

	err = client.List(ctx, kcRealms)
	if err != nil {
		return err
	}

	// Staging realm should be present
	for i := 1; i < len(kcRealms.Items); i++ {
		if kcRealms.Items[i].Spec.Realm.ID == "staging" {
			found = true
		}
	}

	if found != true {
		return fmt.Errorf("Staging realm not present")
	}

	return nil
}

var confirmStagingConsoleLinkExist = func(ctx context.Context, client k8sclient.Client) error {
	amountOfUsers := 3

	for i := 1; i <= amountOfUsers; i++ {
		cl := &consolev1.ConsoleLink{}

		err := client.Get(ctx, k8sclient.ObjectKey{Name: fmt.Sprintf("test-user%v", i) + "-usersso"}, cl)
		if err != nil {
			return err
		}

		nsName := cl.Spec.NamespaceDashboard.Namespaces[0]
		if nsName != fmt.Sprintf("test-user%v", i)+"-stage" {
			return fmt.Errorf("Console link missing")
		}
		stageNs := cl.Spec.NamespaceDashboard.Namespaces[1]
		if stageNs != fmt.Sprintf("test-user%v", i)+"-dev" {
			return fmt.Errorf("Console link missing")
		}
	}

	return nil
}

var confirmUserRealmsAreCreated = func(ctx context.Context, client k8sclient.Client) error {
	var masterRealm = false
	var stagingRealm = false
	var user1Realm = false
	var user2Realm = false
	var user3Realm = false

	// Confirm staging realm, master realm and 3 tenant realms exists
	kcRealms := &keycloak.KeycloakRealmList{}

	err := client.List(ctx, kcRealms)
	if err != nil {
		return err
	}

	for i := 0; i < len(kcRealms.Items); i++ {
		if kcRealms.Items[i].Spec.Realm.ID == "master" {
			masterRealm = true
		}
		if kcRealms.Items[i].Spec.Realm.ID == "staging" {
			stagingRealm = true
		}
		if kcRealms.Items[i].Spec.Realm.ID == "test-user1" {
			user1Realm = true
		}
		if kcRealms.Items[i].Spec.Realm.ID == "test-user2" {
			user2Realm = true
		}
		if kcRealms.Items[i].Spec.Realm.ID == "test-user3" {
			user3Realm = true
		}
	}

	if masterRealm != true || stagingRealm != true || user1Realm != true || user2Realm != true || user3Realm != true {
		return fmt.Errorf("Not all realms exist")
	}
	return nil
}

var confirmUserUsersBelongToCorrectRealm = func(ctx context.Context, client k8sclient.Client) error {

	realmsLabels := []string{StagingRealmName, "master", "test-user1", "test-user2", "test-user3"}

	for i := 0; i < len(realmsLabels); i++ {
		users, err := getUsersListPerLabel(ctx, client, realmsLabels[i])
		if err != nil {
			return fmt.Errorf("Cannot get master realm")
		}
		if realmsLabels[i] == "master" || realmsLabels[i] == StagingRealmName {
			if len(users.Items) > 0 {
				return fmt.Errorf("There are users left on %v", realmsLabels[i])
			}
		} else {
			if len(users.Items) != 1 {
				return fmt.Errorf("There are %v registered users on %v realm", len(users.Items), realmsLabels[i])
			}
		}
	}

	return nil
}

var confirmTenantsConsoleLinks = func(ctx context.Context, client k8sclient.Client) error {
	amountOfUsers := 3

	for i := 1; i <= amountOfUsers; i++ {
		cl := &consolev1.ConsoleLink{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("test-user%v-usersso", i),
			},
		}

		err := client.Get(ctx, k8sclient.ObjectKey{Name: fmt.Sprintf("test-user%v", i) + "-usersso"}, cl)
		if err != nil {
			return err
		}

		nsName := cl.Spec.NamespaceDashboard.Namespaces[0]
		if nsName != fmt.Sprintf("test-user%v", i)+"-stage" {
			return fmt.Errorf("Console link missing")
		}
		stageNs := cl.Spec.NamespaceDashboard.Namespaces[1]
		if stageNs != fmt.Sprintf("test-user%v", i)+"-dev" {
			return fmt.Errorf("Console link missing")
		}

		if cl.Spec.Link.Href != fmt.Sprintf("edge/route/auth/admin/test-user%v/console/", i) {
			return fmt.Errorf(fmt.Sprintf("Console link for test-user%v are incorrect", i))
		}
	}

	return nil
}

func getUsersListPerLabel(ctx context.Context, client k8sclient.Client, label string) (keycloak.KeycloakUserList, error) {
	var users keycloak.KeycloakUserList

	listOptions := []k8sclient.ListOption{
		k8sclient.MatchingLabels(map[string]string{
			"sso": label,
		}),
	}

	err := client.List(ctx, &users, listOptions...)
	if err != nil {
		return users, err
	}

	return users, nil
}
