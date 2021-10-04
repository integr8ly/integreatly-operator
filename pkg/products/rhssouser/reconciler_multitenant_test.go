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
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

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

	// Setup master realm - staging realm expected to be created
	kcRealmList := &keycloak.KeycloakRealmList{
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

	// Create tenant secret
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

	kcUser1 := &userv1.Identity{
		ObjectMeta: v1.ObjectMeta{
			Name: "test-user1",
			Labels: map[string]string{
				"sso": StagingRealmName,
			},
		},
		Extra: map[string]string{
			"email": "testuser1@rhmi.io",
		},
		User: corev1.ObjectReference{
			Name: "test-user1",
			UID:  "9604284c-c30e-4e38-93c0-f34775eda9ef",
		},
		ProviderName: "devsandbox",
	}

	kcUser2 := &userv1.Identity{
		ObjectMeta: v1.ObjectMeta{
			Name: "test-user2",
			Labels: map[string]string{
				"sso": StagingRealmName,
			},
		},
		User: corev1.ObjectReference{
			Name: "test-user2",
			UID:  "4ef04b07-1d6d-4368-9726-778c95467e24",
		},
		ProviderName: "devsandbox",
	}

	kcUser3 := &userv1.Identity{
		ObjectMeta: v1.ObjectMeta{
			Name: "test-user3",
			Labels: map[string]string{
				"sso": StagingRealmName,
			},
		},
		User: corev1.ObjectReference{
			Name: "test-user3",
			UID:  "3",
		},
		ProviderName: "devsandbox",
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
			FakeClient:            fakeclient.NewFakeClientWithScheme(scheme, kc, kcUser1, kcUser2, kcUser3, kcRealmList, oauthClientSecrets),
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
			FakeClient:            fakeclient.NewFakeClientWithScheme(scheme, kc, kcUser1, kcUser2, kcUser3, kcRealmList, oauthClientSecrets),
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

			// Confirm tenants realms got created

			// Confirm tenants links got updated
		})
	}
}
