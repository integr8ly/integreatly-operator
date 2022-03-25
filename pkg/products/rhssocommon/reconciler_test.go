package rhssocommon

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	appsv1 "k8s.io/api/apps/v1"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	controllerruntime "sigs.k8s.io/controller-runtime"

	keycloakCommon "github.com/integr8ly/keycloak-client/pkg/common"

	threescalev1 "github.com/3scale/3scale-operator/apis/apps/v1alpha1"
	kafkav1alpha1 "github.com/integr8ly/integreatly-operator/apis-products/kafka.strimzi.io/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"

	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"

	oauthv1 "github.com/openshift/api/oauth/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	fakeoauthClient "github.com/openshift/client-go/oauth/clientset/versioned/fake"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"

	crov1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	croTypes "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Were not tested before:

//CleanUpKeycloakResources
//CreateKeyCloakRoute
//SetupOpenshiftIDP !!
// GetOAuthClientName - very simple function
// IdentityProviderExists
// deleteKeyCloakUsers
// getKeyCloakUsers

const (
	defaultOperatorNamespace    = "integreatly-operator"
	defaultNamespace            = "user-sso"
	ssoType                     = "user sso"
	keycloakName                = "rhssouser"
	masterRealmName             = "master"
	firstBrokerLoginFlowAlias   = "first broker login"
	reviewProfileExecutionAlias = "review profile config"
)

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := threescalev1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = keycloak.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = operatorsv1alpha1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = corev1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = coreosv1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = kafkav1alpha1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = usersv1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = oauthv1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = routev1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = projectv1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = crov1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = monitoringv1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	return scheme, err
}

func TestReconciler_reconcileCloudResources(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	installation := &integreatlyv1alpha1.RHMI{
		ObjectMeta: controllerruntime.ObjectMeta{
			Name:      "test",
			Namespace: defaultNamespace,
		},
	}

	//completed postgres that points at the secret croPostgresSecret
	croPostgres := &crov1.Postgres{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("rhssouser-postgres-%s", installation.Name),
			Namespace: defaultNamespace,
		},
		Status: croTypes.ResourceTypeStatus{
			Phase: croTypes.PhaseComplete,
			SecretRef: &croTypes.SecretRef{
				Name:      "test",
				Namespace: defaultNamespace,
			},
		},
	}

	//secret created by the cloud resource operator postgres reconciler
	croPostgresSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: defaultNamespace,
		},
		Data: map[string][]byte{},
		Type: corev1.SecretTypeOpaque,
	}

	tests := []struct {
		name         string
		installation *integreatlyv1alpha1.RHMI
		fakeClient   func() k8sclient.Client
		want         integreatlyv1alpha1.StatusPhase
		wantErr      bool
	}{
		{
			name:         "error creating postgres cr causes state failed",
			installation: &integreatlyv1alpha1.RHMI{},
			fakeClient: func() k8sclient.Client {
				mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, croPostgres, croPostgresSecret)
				mockClient.GetFunc = func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return errors.New("test error")
				}
				return mockClient
			},
			wantErr: true,
			want:    integreatlyv1alpha1.PhaseFailed,
		},
		{
			name:         "nil secret causes state awaiting",
			installation: installation,
			fakeClient: func() k8sclient.Client {
				pendingCroPostgres := croPostgres.DeepCopy()
				pendingCroPostgres.Status.Phase = croTypes.PhaseInProgress
				return moqclient.NewSigsClientMoqWithScheme(scheme, croPostgresSecret, pendingCroPostgres)
			},
			want: integreatlyv1alpha1.PhaseAwaitingCloudResources,
		},
		{
			name:         "defined secret causes state completed",
			installation: installation,
			fakeClient: func() k8sclient.Client {
				return moqclient.NewSigsClientMoqWithScheme(scheme, croPostgres, croPostgresSecret)
			},
			want: integreatlyv1alpha1.PhaseCompleted,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &config.RHSSOCommon{
				Config: config.ProductConfig(map[string]string{
					"NAMESPACE": defaultNamespace,
				}),
			}
			r := &Reconciler{
				Log: getLogger(),
			}

			got, err := r.ReconcileCloudResources(constants.RHSSOUserProstgresPrefix, defaultNamespace, ssoType, config, context.TODO(), tt.installation, tt.fakeClient())
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileCloudResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("reconcileCloudResources() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_handleProgress(t *testing.T) {
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

	kcr := getKcr(keycloak.KeycloakRealmStatus{
		Phase: keycloak.PhaseReconciling,
	})

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: defaultNamespace,
		},
	}

	githubOauthSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "github-oauth-secret",
			Namespace: defaultOperatorNamespace,
		},
	}

	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oauth-client-secrets",
			Namespace: defaultOperatorNamespace,
		},
		Data: map[string][]byte{
			"rhssouser": bytes.NewBufferString("test").Bytes(),
		},
	}

	cases := []struct {
		Name                  string
		ExpectError           bool
		ExpectedStatus        integreatlyv1alpha1.StatusPhase
		ExpectedError         string
		Logger                l.Logger
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
			Name:                  "test ready kcr returns phase complete",
			ExpectedStatus:        integreatlyv1alpha1.PhaseCompleted,
			Logger:                getLogger(),
			FakeClient:            moqclient.NewSigsClientMoqWithScheme(scheme, secret, kc, kcr, githubOauthSecret, oauthClientSecrets),
			FakeOauthClient:       fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:            basicConfigMock(),
			Installation:          &integreatlyv1alpha1.RHMI{},
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
		},
		{
			Name:                  "test unready kcr cr returns phase in progress",
			ExpectedStatus:        integreatlyv1alpha1.PhaseInProgress,
			Logger:                getLogger(),
			FakeClient:            moqclient.NewSigsClientMoqWithScheme(scheme, kc, secret, getKcr(keycloak.KeycloakRealmStatus{Phase: keycloak.PhaseFailing}), githubOauthSecret, oauthClientSecrets),
			FakeOauthClient:       fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:            basicConfigMock(),
			Installation:          &integreatlyv1alpha1.RHMI{},
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
		},
		{
			Name:                  "test missing kc cr returns phase failed",
			ExpectedStatus:        integreatlyv1alpha1.PhaseFailed,
			ExpectError:           true,
			FakeClient:            moqclient.NewSigsClientMoqWithScheme(scheme, secret, kcr, githubOauthSecret, oauthClientSecrets),
			FakeOauthClient:       fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:            basicConfigMock(),
			Installation:          &integreatlyv1alpha1.RHMI{},
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
		},
		{
			Name:                  "test missing kcr cr returns phase failed",
			ExpectedStatus:        integreatlyv1alpha1.PhaseFailed,
			ExpectError:           true,
			Logger:                getLogger(),
			FakeClient:            moqclient.NewSigsClientMoqWithScheme(scheme, secret, kc, githubOauthSecret, oauthClientSecrets),
			FakeOauthClient:       fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig:            basicConfigMock(),
			Installation:          &integreatlyv1alpha1.RHMI{},
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
		},
		{
			Name:            "test failed config write",
			ExpectedStatus:  integreatlyv1alpha1.PhaseFailed,
			ExpectError:     true,
			FakeClient:      moqclient.NewSigsClientMoqWithScheme(scheme, secret, kc, kcr, githubOauthSecret, oauthClientSecrets),
			FakeOauthClient: fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeConfig: &config.ConfigReadWriterMock{
				ReadRHSSOUserFunc: func() (*config.RHSSOUser, error) {
					return config.NewRHSSOUser(config.ProductConfig{
						"NAMESPACE": "user-sso",
						"REALM":     "openshift",
						"URL":       "rhsso.openshift-cluster.com",
					}), nil
				},
				WriteConfigFunc: func(config config.ConfigReadable) error {
					return errors.New("error writing config")
				},
			},
			Installation:          &integreatlyv1alpha1.RHMI{},
			Recorder:              setupRecorder(),
			ApiUrl:                "https://serverurl",
			KeycloakClientFactory: getMoqKeycloakClientFactory(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			config := &config.RHSSOUser{
				RHSSOCommon: &config.RHSSOCommon{
					Config: config.ProductConfig(map[string]string{
						"NAMESPACE": defaultNamespace,
					}),
				},
			}
			testReconciler := NewReconciler(
				tc.FakeConfig,
				tc.FakeMPM,
				tc.Installation,
				tc.Logger,
				tc.FakeOauthClient,
				tc.Recorder,
				tc.ApiUrl,
				tc.KeycloakClientFactory,
				*marketplace.LocalProductDeclaration("integreatly-rhsso"),
			)
			if err != nil && err.Error() != tc.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
			}

			status, err := testReconciler.HandleProgressPhase(context.TODO(), tc.FakeClient, keycloakName, masterRealmName, config, config.RHSSOCommon, string(integreatlyv1alpha1.VersionRHSSOUser), string(integreatlyv1alpha1.OperatorVersionRHSSOUser))

			if err != nil && !tc.ExpectError {
				t.Fatalf("expected error but got one: %v", err)
			}

			if err == nil && tc.ExpectError {
				t.Fatal("expected error but got none")
			}

			if status != tc.ExpectedStatus {
				t.Fatalf("Expected status: '%v', got: '%v'", tc.ExpectedStatus, status)
			}
		})
	}
}

func getKcr(status keycloak.KeycloakRealmStatus) *keycloak.KeycloakRealm {
	return &keycloak.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      masterRealmName,
			Namespace: defaultNamespace,
		},
		Spec: keycloak.KeycloakRealmSpec{
			Realm: &keycloak.KeycloakAPIRealm{
				ID:          masterRealmName,
				Realm:       masterRealmName,
				DisplayName: masterRealmName,
				Enabled:     true,
				EventsListeners: []string{
					"metrics-listener",
				},
			},
		},
		Status: status,
	}
}

func basicConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		GetOperatorNamespaceFunc: func() string {
			return defaultOperatorNamespace
		},
		ReadRHSSOUserFunc: func() (*config.RHSSOUser, error) {
			return config.NewRHSSOUser(config.ProductConfig{
				"NAMESPACE": "user-sso",
				"REALM":     "openshift",
				"URL":       "rhsso.openshift-cluster.com",
				"HOST":      "edge/route",
			}), nil
		},
		WriteConfigFunc: func(config config.ConfigReadable) error {
			return nil
		},
		GetOauthClientsSecretNameFunc: func() string {
			return "oauth-client-secrets"
		},
		ReadMonitoringFunc: func() (*config.Monitoring, error) {
			return config.NewMonitoring(config.ProductConfig{
				"NAMESPACE": "middleware-monitoring",
			}), nil
		},
	}
}

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
}

func getMoqKeycloakClientFactory() keycloakCommon.KeycloakClientFactory {
	exInfo := []*keycloak.AuthenticationExecutionInfo{
		{
			ProviderID: "identity-provider-redirector",
			ID:         "123-123-123",
		},
	}

	keycloakInterfaceMock, context := createKeycloakInterfaceMock()

	// Add the browser flow execution mock to the context in order to test
	// the reconcileComponents phase
	context.AuthenticationFlowsExecutions["browser"] = exInfo

	return &keycloakCommon.KeycloakClientFactoryMock{AuthenticatedClientFunc: func(kc keycloak.Keycloak) (keycloakInterface keycloakCommon.KeycloakInterface, err error) {
		return &keycloakCommon.KeycloakInterfaceMock{CreateIdentityProviderFunc: func(identityProvider *keycloak.KeycloakIdentityProvider, realmName string) (string, error) {
			return "", nil
		}, GetIdentityProviderFunc: func(alias string, realmName string) (provider *keycloak.KeycloakIdentityProvider, err error) {
			return nil, nil
		}, CreateAuthenticatorConfigFunc: func(authenticatorConfig *keycloak.AuthenticatorConfig, realmName string, executionID string) (string, error) {
			return "", nil
		},
			ListRealmsFunc:                           keycloakInterfaceMock.ListRealms,
			FindGroupByNameFunc:                      keycloakInterfaceMock.FindGroupByName,
			CreateGroupFunc:                          keycloakInterfaceMock.CreateGroup,
			SetGroupChildFunc:                        keycloakInterfaceMock.SetGroupChild,
			MakeGroupDefaultFunc:                     keycloakInterfaceMock.MakeGroupDefault,
			ListUsersInGroupFunc:                     keycloakInterfaceMock.ListUsersInGroup,
			ListDefaultGroupsFunc:                    keycloakInterfaceMock.ListDefaultGroups,
			CreateGroupClientRoleFunc:                keycloakInterfaceMock.CreateGroupClientRole,
			ListGroupClientRolesFunc:                 keycloakInterfaceMock.ListGroupClientRoles,
			FindGroupClientRoleFunc:                  keycloakInterfaceMock.FindGroupClientRole,
			ListAvailableGroupClientRolesFunc:        keycloakInterfaceMock.ListAvailableGroupClientRoles,
			FindAvailableGroupClientRoleFunc:         keycloakInterfaceMock.FindAvailableGroupClientRole,
			ListGroupRealmRolesFunc:                  keycloakInterfaceMock.ListGroupRealmRoles,
			ListAvailableGroupRealmRolesFunc:         keycloakInterfaceMock.ListAvailableGroupRealmRoles,
			CreateGroupRealmRoleFunc:                 keycloakInterfaceMock.CreateGroupRealmRole,
			ListAuthenticationExecutionsForFlowFunc:  keycloakInterfaceMock.ListAuthenticationExecutionsForFlow,
			FindAuthenticationExecutionForFlowFunc:   keycloakInterfaceMock.FindAuthenticationExecutionForFlow,
			UpdateAuthenticationExecutionForFlowFunc: keycloakInterfaceMock.UpdateAuthenticationExecutionForFlow,
			ListClientsFunc:                          keycloakInterfaceMock.ListClients,
		}, nil
	}}
}

func createKeycloakInterfaceMock() (keycloakCommon.KeycloakInterface, *mockClientContext) {
	context := mockClientContext{
		Groups:        []*keycloakCommon.Group{},
		DefaultGroups: []*keycloakCommon.Group{},
		ClientRoles:   map[string][]*keycloak.KeycloakUserRole{},
		RealmRoles:    map[string][]*keycloak.KeycloakUserRole{},
		AuthenticationFlowsExecutions: map[string][]*keycloak.AuthenticationExecutionInfo{
			firstBrokerLoginFlowAlias: []*keycloak.AuthenticationExecutionInfo{
				&keycloak.AuthenticationExecutionInfo{
					Requirement: "REQUIRED",
					Alias:       reviewProfileExecutionAlias,
				},
				// dummy ones
				&keycloak.AuthenticationExecutionInfo{
					Requirement: "REQUIRED",
					Alias:       "dummy execution",
				},
			},
		},
	}

	availableGroupClientRoles := []*keycloak.KeycloakUserRole{
		&keycloak.KeycloakUserRole{
			ID:   "create-client",
			Name: "create-client",
		},
		&keycloak.KeycloakUserRole{
			ID:   "manage-authorization",
			Name: "manage-authorization",
		},
		&keycloak.KeycloakUserRole{
			ID:   "manage-clients",
			Name: "manage-clients",
		},
		&keycloak.KeycloakUserRole{
			ID:   "manage-events",
			Name: "manage-events",
		},
		&keycloak.KeycloakUserRole{
			ID:   "manage-identity-providers",
			Name: "manage-identity-providers",
		},
		&keycloak.KeycloakUserRole{
			ID:   "manage-realm",
			Name: "manage-realm",
		},
		&keycloak.KeycloakUserRole{
			ID:   "manage-users",
			Name: "manage-users",
		},
		&keycloak.KeycloakUserRole{
			ID:   "query-clients",
			Name: "query-clients",
		},
		&keycloak.KeycloakUserRole{
			ID:   "query-groups",
			Name: "query-groups",
		},
		&keycloak.KeycloakUserRole{
			ID:   "query-realms",
			Name: "query-realms",
		},
		&keycloak.KeycloakUserRole{
			ID:   "query-users",
			Name: "query-users",
		},
		&keycloak.KeycloakUserRole{
			ID:   "view-authorization",
			Name: "view-authorization",
		},
		&keycloak.KeycloakUserRole{
			ID:   "view-clients",
			Name: "view-clients",
		},
		&keycloak.KeycloakUserRole{
			ID:   "view-events",
			Name: "view-events",
		},
		&keycloak.KeycloakUserRole{
			ID:   "view-identity-providers",
			Name: "view-identity-providers",
		},
		&keycloak.KeycloakUserRole{
			ID:   "view-realm",
			Name: "view-realm",
		},
		&keycloak.KeycloakUserRole{
			ID:   "view-users",
			Name: "view-users",
		},
	}

	availableGroupRealmRoles := []*keycloak.KeycloakUserRole{
		&keycloak.KeycloakUserRole{
			ID:   "mock-role-3",
			Name: "mock-role-3",
		},
		&keycloak.KeycloakUserRole{
			ID:   "create-realm",
			Name: "create-realm",
		},
		&keycloak.KeycloakUserRole{
			ID:   "mock-role-4",
			Name: "mock-role-4",
		},
	}

	listRealmsFunc := func() ([]*keycloak.KeycloakAPIRealm, error) {
		return []*keycloak.KeycloakAPIRealm{
			&keycloak.KeycloakAPIRealm{
				Realm: "master",
			},
			&keycloak.KeycloakAPIRealm{
				Realm: "test",
			},
		}, nil
	}

	findGroupByNameFunc := func(groupName string, realmName string) (*keycloakCommon.Group, error) {
		for _, group := range context.Groups {
			if group.Name == groupName {
				return group, nil
			}
		}

		return nil, nil
	}

	createGroupFunc := func(groupName string, realmName string) (string, error) {
		nextID := fmt.Sprintf("group-%d", len(context.Groups))

		newGroup := &keycloakCommon.Group{
			ID:   string(nextID),
			Name: groupName,
		}

		context.Groups = append(context.Groups, newGroup)

		context.ClientRoles[nextID] = []*keycloak.KeycloakUserRole{}
		context.RealmRoles[nextID] = []*keycloak.KeycloakUserRole{}

		return nextID, nil
	}

	setGroupChildFunc := func(groupID, realmName string, childGroup *keycloakCommon.Group) error {
		var childGroupToAppend *keycloakCommon.Group
		for _, group := range context.Groups {
			if group.ID == childGroup.ID {
				childGroupToAppend = group
			}
		}

		if childGroupToAppend == nil {
			childGroupToAppend = childGroup
		}

		found := false
		for _, group := range context.Groups {
			if group.ID == groupID {
				group.SubGroups = append(group.SubGroups, childGroupToAppend)
				found = true
			}
		}

		if !found {
			return fmt.Errorf("Group %s not found", groupID)
		}

		return nil
	}

	listUsersInGroupFunc := func(realmName, groupID string) ([]*keycloak.KeycloakAPIUser, error) {
		return []*keycloak.KeycloakAPIUser{}, nil
	}

	makeGroupDefaultFunc := func(groupID string, realmName string) error {
		var group *keycloakCommon.Group

		for _, existingGroup := range context.Groups {
			if existingGroup.ID == groupID {
				group = existingGroup
				break
			}
		}

		if group == nil {
			return fmt.Errorf("Referenced group not found")
		}

		context.DefaultGroups = append(context.DefaultGroups, group)
		return nil
	}

	listDefaultGroupsFunc := func(realmName string) ([]*keycloakCommon.Group, error) {
		return context.DefaultGroups, nil
	}

	createGroupClientRoleFunc := func(role *keycloak.KeycloakUserRole, realmName, clientID, groupID string) (string, error) {
		groupClientRoles, ok := context.ClientRoles[groupID]

		if !ok {
			return "", fmt.Errorf("Referenced group not found")
		}

		context.ClientRoles[groupID] = append(groupClientRoles, role)
		return "dummy-group-client-role-id", nil
	}

	listGroupClientRolesFunc := func(realmName, clientID, groupID string) ([]*keycloak.KeycloakUserRole, error) {
		groupRoles, ok := context.ClientRoles[groupID]

		if !ok {
			return nil, fmt.Errorf("Referenced group not found")
		}

		return groupRoles, nil
	}

	listAvailableGroupClientRolesFunc := func(realmName, clientID, groupID string) ([]*keycloak.KeycloakUserRole, error) {
		_, ok := context.ClientRoles[groupID]

		if !ok {
			return nil, fmt.Errorf("Referenced group not found")
		}

		return availableGroupClientRoles, nil
	}

	findGroupClientRoleFunc := func(realmName, clientID, groupID string, predicate func(*keycloak.KeycloakUserRole) bool) (*keycloak.KeycloakUserRole, error) {
		all, err := listGroupClientRolesFunc(realmName, clientID, groupID)

		if err != nil {
			return nil, err
		}

		for _, role := range all {
			if predicate(role) {
				return role, nil
			}
		}

		return nil, nil
	}

	findAvailableGroupClientRoleFunc := func(realmName, clientID, groupID string, predicate func(*keycloak.KeycloakUserRole) bool) (*keycloak.KeycloakUserRole, error) {
		all, err := listAvailableGroupClientRolesFunc(realmName, clientID, groupID)

		if err != nil {
			return nil, err
		}

		for _, role := range all {
			if predicate(role) {
				return role, nil
			}
		}

		return nil, nil
	}

	listGroupRealmRolesFunc := func(realmName, groupID string) ([]*keycloak.KeycloakUserRole, error) {
		groupRoles, ok := context.RealmRoles[groupID]

		if !ok {
			return nil, fmt.Errorf("Referenced group not found")
		}

		return groupRoles, nil
	}

	listAvailableGroupRealmRolesFunc := func(realmName, groupID string) ([]*keycloak.KeycloakUserRole, error) {
		_, ok := context.RealmRoles[groupID]

		if !ok {
			return nil, fmt.Errorf("Referenced group not found")
		}

		return availableGroupRealmRoles, nil
	}

	createGroupRealmRoleFunc := func(role *keycloak.KeycloakUserRole, realmName, groupID string) (string, error) {
		groupRealmRoles, ok := context.RealmRoles[groupID]

		if !ok {
			return "", fmt.Errorf("Referenced group not found")
		}

		context.RealmRoles[groupID] = append(groupRealmRoles, role)
		return "dummy-group-realm-role-id", nil
	}

	listClientsFunc := func(realmName string) ([]*keycloak.KeycloakAPIClient, error) {
		return []*keycloak.KeycloakAPIClient{
			&keycloak.KeycloakAPIClient{
				ClientID: "test-realm",
				ID:       "test-realm",
				Name:     "test-realm",
			},
			&keycloak.KeycloakAPIClient{
				ClientID: "master-realm",
				ID:       "master-realm",
				Name:     "master-realm",
			},
		}, nil
	}

	listAuthenticationExecutionsForFlowFunc := func(flowAlias, realmName string) ([]*keycloak.AuthenticationExecutionInfo, error) {
		executions, ok := context.AuthenticationFlowsExecutions[flowAlias]

		if !ok {
			return nil, errors.New("Authentication flow not found")
		}

		return executions, nil
	}

	findAuthenticationExecutionForFlowFunc := func(flowAlias, realmName string, predicate func(*keycloak.AuthenticationExecutionInfo) bool) (*keycloak.AuthenticationExecutionInfo, error) {
		executions, err := listAuthenticationExecutionsForFlowFunc(flowAlias, realmName)

		if err != nil {
			return nil, err
		}

		for _, execution := range executions {
			if predicate(execution) {
				return execution, nil
			}
		}

		return nil, nil
	}

	updateAuthenticationExecutionForFlowFunc := func(flowAlias, realmName string, execution *keycloak.AuthenticationExecutionInfo) error {
		executions, ok := context.AuthenticationFlowsExecutions[flowAlias]

		if !ok {
			return fmt.Errorf("Authentication flow %s not found", flowAlias)
		}

		for i, currentExecution := range executions {
			if currentExecution.Alias != execution.Alias {
				continue
			}

			context.AuthenticationFlowsExecutions[flowAlias][i] = execution
			break
		}

		return nil
	}

	return &keycloakCommon.KeycloakInterfaceMock{
		ListRealmsFunc:                           listRealmsFunc,
		FindGroupByNameFunc:                      findGroupByNameFunc,
		CreateGroupFunc:                          createGroupFunc,
		SetGroupChildFunc:                        setGroupChildFunc,
		ListUsersInGroupFunc:                     listUsersInGroupFunc,
		MakeGroupDefaultFunc:                     makeGroupDefaultFunc,
		ListDefaultGroupsFunc:                    listDefaultGroupsFunc,
		CreateGroupClientRoleFunc:                createGroupClientRoleFunc,
		ListGroupClientRolesFunc:                 listGroupClientRolesFunc,
		ListAvailableGroupClientRolesFunc:        listAvailableGroupClientRolesFunc,
		FindGroupClientRoleFunc:                  findGroupClientRoleFunc,
		FindAvailableGroupClientRoleFunc:         findAvailableGroupClientRoleFunc,
		ListGroupRealmRolesFunc:                  listGroupRealmRolesFunc,
		ListAvailableGroupRealmRolesFunc:         listAvailableGroupRealmRolesFunc,
		CreateGroupRealmRoleFunc:                 createGroupRealmRoleFunc,
		ListAuthenticationExecutionsForFlowFunc:  listAuthenticationExecutionsForFlowFunc,
		FindAuthenticationExecutionForFlowFunc:   findAuthenticationExecutionForFlowFunc,
		UpdateAuthenticationExecutionForFlowFunc: updateAuthenticationExecutionForFlowFunc,
		ListClientsFunc:                          listClientsFunc,
	}, &context
}

type mockClientContext struct {
	Groups                        []*keycloakCommon.Group
	DefaultGroups                 []*keycloakCommon.Group
	ClientRoles                   map[string][]*keycloak.KeycloakUserRole
	RealmRoles                    map[string][]*keycloak.KeycloakUserRole
	AuthenticationFlowsExecutions map[string][]*keycloak.AuthenticationExecutionInfo
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.ProductRHSSO})
}

func TestReconciler_CleanupKeycloakResources(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}
	err = apiextensionv1.AddToScheme(scheme)
	if err != nil {
		t.Fatalf("Error setting up scheme: %s", err)
	}

	type fields struct {
		ConfigManager         config.ConfigReadWriter
		mpm                   *marketplace.MarketplaceInterfaceMock
		Installation          *integreatlyv1alpha1.RHMI
		Log                   l.Logger
		Oauthv1Client         oauthClient.OauthV1Interface
		APIURL                string
		Reconciler            *resources.Reconciler
		Recorder              record.EventRecorder
		KeycloakClientFactory keycloakCommon.KeycloakClientFactory
	}
	type args struct {
		ctx          context.Context
		inst         *integreatlyv1alpha1.RHMI
		serverClient k8sclient.Client
		ns           string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "Test uninstallation: missing Keycloak CRD returns phaseCompleted",
			fields: fields{
				ConfigManager:         basicConfigMock(),
				Log:                   getLogger(),
				KeycloakClientFactory: getMoqKeycloakClientFactory(),
			},
			args: args{
				ctx: context.TODO(),
				inst: &integreatlyv1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						DeletionTimestamp: &metav1.Time{},
					},
				},
				serverClient: moqclient.NewSigsClientMoqWithScheme(scheme),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "Test uninstallation: missing v1 API returns phaseFailed",
			fields: fields{
				ConfigManager:         basicConfigMock(),
				Log:                   getLogger(),
				KeycloakClientFactory: getMoqKeycloakClientFactory(),
			},
			args: args{
				ctx: context.TODO(),
				inst: &integreatlyv1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						DeletionTimestamp: &metav1.Time{},
					},
				},
				serverClient: moqclient.NewSigsClientMoqWithScheme(runtime.NewScheme()),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager:         tt.fields.ConfigManager,
				mpm:                   tt.fields.mpm,
				Installation:          tt.fields.Installation,
				Log:                   tt.fields.Log,
				Oauthv1Client:         tt.fields.Oauthv1Client,
				APIURL:                tt.fields.APIURL,
				Reconciler:            tt.fields.Reconciler,
				Recorder:              tt.fields.Recorder,
				KeycloakClientFactory: tt.fields.KeycloakClientFactory,
			}
			got, err := r.CleanupKeycloakResources(tt.args.ctx, tt.args.inst, tt.args.serverClient, tt.args.ns)
			if (err != nil) != tt.wantErr {
				t.Errorf("CleanupKeycloakResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CleanupKeycloakResources() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsUpgrade(t *testing.T) {
	type args struct {
		config         *config.RHSSOCommon
		productVersion integreatlyv1alpha1.ProductVersion
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test - False when empty version in config - fresh install",
			args: args{
				config: config.NewRHSSOCommon(config.ProductConfig{}),
			},
			want: false,
		},
		{
			name: "Test - False when version in config equals constant product version",
			args: args{
				config: config.NewRHSSOCommon(config.ProductConfig{
					"VERSION": "1.1.1",
				}),
				productVersion: integreatlyv1alpha1.ProductVersion("1.1.1"),
			},
			want: false,
		},
		{
			name: "Test - True when version in config does not equal constant product version",
			args: args{
				config: config.NewRHSSOCommon(config.ProductConfig{
					"VERSION": "1.1.1",
				}),
				productVersion: integreatlyv1alpha1.ProductVersion("1.1.2"),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUpgrade(tt.args.config, tt.args.productVersion); got != tt.want {
				t.Errorf("IsUpgrade() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_SetRollingStrategyForUpgrade(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ConfigManager         config.ConfigReadWriter
		mpm                   marketplace.MarketplaceInterface
		Installation          *integreatlyv1alpha1.RHMI
		Log                   l.Logger
		Oauthv1Client         oauthClient.OauthV1Interface
		APIURL                string
		Reconciler            *resources.Reconciler
		Recorder              record.EventRecorder
		KeycloakClientFactory keycloakCommon.KeycloakClientFactory
	}
	type args struct {
		isUpgrade      bool
		ctx            context.Context
		serverClient   k8sclient.Client
		config         *config.RHSSOCommon
		productVersion integreatlyv1alpha1.ProductVersion
		keycloakName   string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "Test - Phase complete when not an upgrade ",
			args: args{
				isUpgrade: false,
			},
			want: integreatlyv1alpha1.PhaseCompleted,
		},
		{
			name: "Test - Phase failed when non semantic version is used in config",
			args: args{
				isUpgrade: true,
				config: &config.RHSSOCommon{Config: map[string]string{
					"VERSION": "NonSemantic",
				}},
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "Test - Phase failed when non semantic version is used in version constant",
			args: args{
				isUpgrade: true,
				config: &config.RHSSOCommon{Config: map[string]string{
					"VERSION": "7.0",
				}},
				productVersion: integreatlyv1alpha1.ProductVersion("NonSemantic"),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "Test - Phase Complete - Patch Version Bump",
			args: args{
				isUpgrade: true,
				config: &config.RHSSOCommon{Config: map[string]string{
					"VERSION": "7.4",
				}},
				productVersion: integreatlyv1alpha1.ProductVersion("7.4.1"),
			},
			want: integreatlyv1alpha1.PhaseCompleted,
		},
		{
			name: "Test - Phase Complete - Minor Version Bump",
			args: args{
				isUpgrade: true,
				config: &config.RHSSOCommon{Config: map[string]string{
					"VERSION": "7.4",
				}},
				productVersion: integreatlyv1alpha1.ProductVersion("7.5"),
				serverClient:   moqclient.NewSigsClientMoqWithScheme(scheme),
				keycloakName:   keycloakName,
			},
			fields: fields{Log: getLogger()},
			want:   integreatlyv1alpha1.PhaseCompleted,
		},
		{
			name: "Test - Phase Complete - Major Version Bump",
			args: args{
				isUpgrade: true,
				config: &config.RHSSOCommon{Config: map[string]string{
					"VERSION": "7.4",
				}},
				productVersion: integreatlyv1alpha1.ProductVersion("8.4"),
				serverClient:   moqclient.NewSigsClientMoqWithScheme(scheme),
				keycloakName:   keycloakName,
			},
			fields: fields{Log: getLogger()},
			want:   integreatlyv1alpha1.PhaseCompleted,
		},
		{
			name: "Test - Phase Failed - Keycloak CR get error",
			args: args{
				isUpgrade: true,
				config: &config.RHSSOCommon{Config: map[string]string{
					"VERSION": "7.4",
				}},
				productVersion: integreatlyv1alpha1.ProductVersion("8.4"),
				serverClient: &moqclient.SigsClientInterfaceMock{GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
					return fmt.Errorf("GetError")
				}},
				keycloakName: keycloakName,
			},
			fields:  fields{Log: getLogger()},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager:         tt.fields.ConfigManager,
				mpm:                   tt.fields.mpm,
				Installation:          tt.fields.Installation,
				Log:                   tt.fields.Log,
				Oauthv1Client:         tt.fields.Oauthv1Client,
				APIURL:                tt.fields.APIURL,
				Reconciler:            tt.fields.Reconciler,
				Recorder:              tt.fields.Recorder,
				KeycloakClientFactory: tt.fields.KeycloakClientFactory,
			}
			got, err := r.SetRollingStrategyForUpgrade(tt.args.isUpgrade, tt.args.ctx, tt.args.serverClient, tt.args.config, tt.args.productVersion, tt.args.keycloakName)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetRollingStrategyForUpgrade() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SetRollingStrategyForUpgrade() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_IsOperatorInstallComplete(t *testing.T) {
	type fields struct {
		ConfigManager         config.ConfigReadWriter
		mpm                   marketplace.MarketplaceInterface
		Installation          *integreatlyv1alpha1.RHMI
		Log                   l.Logger
		Oauthv1Client         oauthClient.OauthV1Interface
		APIURL                string
		Reconciler            *resources.Reconciler
		Recorder              record.EventRecorder
		KeycloakClientFactory keycloakCommon.KeycloakClientFactory
	}
	type args struct {
		kc              *keycloak.Keycloak
		operatorVersion integreatlyv1alpha1.OperatorVersion
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Test - false when status version not equal operator version",
			args: args{
				kc: &keycloak.Keycloak{
					Status: keycloak.KeycloakStatus{
						Version: "1.0.0",
					},
				},
				operatorVersion: integreatlyv1alpha1.OperatorVersion("1.1.0"),
			},
			want: false,
		},
		{
			name: "Test - false when status not ready",
			args: args{
				kc: &keycloak.Keycloak{
					Status: keycloak.KeycloakStatus{
						Version: "1.1.0",
						Ready:   false,
					},
				},
				operatorVersion: integreatlyv1alpha1.OperatorVersion("1.1.0"),
			},
			want: false,
		},
		{
			name: "Test - true status version equal operator version and ready",
			args: args{
				kc: &keycloak.Keycloak{
					Status: keycloak.KeycloakStatus{
						Version: "1.1.0",
						Ready:   true,
					},
				},
				operatorVersion: integreatlyv1alpha1.OperatorVersion("1.1.0"),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager:         tt.fields.ConfigManager,
				mpm:                   tt.fields.mpm,
				Installation:          tt.fields.Installation,
				Log:                   tt.fields.Log,
				Oauthv1Client:         tt.fields.Oauthv1Client,
				APIURL:                tt.fields.APIURL,
				Reconciler:            tt.fields.Reconciler,
				Recorder:              tt.fields.Recorder,
				KeycloakClientFactory: tt.fields.KeycloakClientFactory,
			}
			if got := r.IsOperatorInstallComplete(tt.args.kc, tt.args.operatorVersion); got != tt.want {
				t.Errorf("IsOperatorInstallComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_ReconcileCSVEnvVars(t *testing.T) {
	type fields struct {
		ConfigManager         config.ConfigReadWriter
		mpm                   marketplace.MarketplaceInterface
		Installation          *integreatlyv1alpha1.RHMI
		Log                   l.Logger
		Oauthv1Client         oauthClient.OauthV1Interface
		APIURL                string
		Reconciler            *resources.Reconciler
		Recorder              record.EventRecorder
		KeycloakClientFactory keycloakCommon.KeycloakClientFactory
	}
	type args struct {
		kc              *keycloak.Keycloak
		operatorVersion integreatlyv1alpha1.OperatorVersion
	}
	scenarios := []struct {
		name     string
		CSV      *olmv1alpha1.ClusterServiceVersion
		EnvVars  map[string]string
		fields   fields
		args     args
		validate func(csv *olmv1alpha1.ClusterServiceVersion, updated bool, err error) error
	}{
		{
			name: "new env var is created",
			args: args{
				kc: &keycloak.Keycloak{
					Status: keycloak.KeycloakStatus{
						Version: "1.0.0",
					},
				},
				operatorVersion: integreatlyv1alpha1.OperatorVersion("1.1.0"),
			},
			CSV: &operatorsv1alpha1.ClusterServiceVersion{
				Spec: operatorsv1alpha1.ClusterServiceVersionSpec{
					InstallStrategy: olmv1alpha1.NamedInstallStrategy{
						StrategySpec: olmv1alpha1.StrategyDetailsDeployment{
							DeploymentSpecs: []olmv1alpha1.StrategyDeploymentSpec{
								{
									Name: "keycloak-operator",
									Spec: appsv1.DeploymentSpec{
										Template: corev1.PodTemplateSpec{
											Spec: corev1.PodSpec{
												Containers: []corev1.Container{
													{
														Env: []corev1.EnvVar{},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			EnvVars: map[string]string{
				"PHILS_COOL_ENV_VAR": "yesboi",
			},
			validate: func(csv *operatorsv1alpha1.ClusterServiceVersion, updated bool, err error) error {
				for _, e := range csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs[0].Spec.Template.Spec.Containers[0].Env {
					if e.Name != "PHILS_COOL_ENV_VAR" || e.Value != "yesboi" {
						return errors.New(fmt.Sprintf("Expected env var {PHILS_COOL_ENV_VAR: yesboi}, got %+v", e))
					}
				}
				if !updated {
					return errors.New(fmt.Sprintf("Expected updated to be true, but got: %v", updated))
				}
				return nil
			},
		},
		{
			name: "existing env var is preserved",
			args: args{
				kc: &keycloak.Keycloak{
					Status: keycloak.KeycloakStatus{
						Version: "1.0.0",
					},
				},
				operatorVersion: integreatlyv1alpha1.OperatorVersion("1.1.0"),
			},
			CSV: &operatorsv1alpha1.ClusterServiceVersion{
				Spec: operatorsv1alpha1.ClusterServiceVersionSpec{
					InstallStrategy: olmv1alpha1.NamedInstallStrategy{
						StrategySpec: olmv1alpha1.StrategyDetailsDeployment{
							DeploymentSpecs: []olmv1alpha1.StrategyDeploymentSpec{
								{
									Name: "keycloak-operator",
									Spec: appsv1.DeploymentSpec{
										Template: corev1.PodTemplateSpec{
											Spec: corev1.PodSpec{
												Containers: []corev1.Container{
													{
														Env: []corev1.EnvVar{
															{
																Name:  "PAULS_AMAZEBALLS_ENV_VAR",
																Value: "yeaaah",
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},

			EnvVars: map[string]string{
				"PHILS_COOL_ENV_VAR": "yesboi",
			},
			validate: func(csv *operatorsv1alpha1.ClusterServiceVersion, updated bool, err error) error {
				envs := csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs[0].Spec.Template.Spec.Containers[0].Env
				for _, e := range envs {
					if e.Name == "PHILS_COOL_ENV_VAR" {
						if e.Value != "yesboi" {
							return errors.New(fmt.Sprintf("Expected env var {PHILS_COOL_ENV_VAR: yesboi}, got %+v", envs))
						}
					} else if e.Name == "PAULS_AMAZEBALLS_ENV_VAR" {
						if e.Value != "yeaaah" {
							return errors.New(fmt.Sprintf("Expected env var {PAULS_AMAZEBALLS_ENV_VAR: yeaaah}, got %+v", envs))
						}
					} else {
						return errors.New(fmt.Sprintf("unexpected env var (should only have PHILS_COOL_ENV_VAR and PAULS_AMAZEBALLS_ENV_VAR): %+v", envs))
					}
				}
				if !updated {
					return errors.New(fmt.Sprintf("Expected updated to be true, but got: %v", updated))
				}
				return nil
			},
		},
		{
			name: "existing env var is updated",
			args: args{
				kc: &keycloak.Keycloak{
					Status: keycloak.KeycloakStatus{
						Version: "1.0.0",
					},
				},
				operatorVersion: integreatlyv1alpha1.OperatorVersion("1.1.0"),
			},
			CSV: &operatorsv1alpha1.ClusterServiceVersion{
				Spec: operatorsv1alpha1.ClusterServiceVersionSpec{
					InstallStrategy: olmv1alpha1.NamedInstallStrategy{
						StrategySpec: olmv1alpha1.StrategyDetailsDeployment{
							DeploymentSpecs: []olmv1alpha1.StrategyDeploymentSpec{
								{
									Name: "keycloak-operator",
									Spec: appsv1.DeploymentSpec{
										Template: corev1.PodTemplateSpec{
											Spec: corev1.PodSpec{
												Containers: []corev1.Container{
													{
														Env: []corev1.EnvVar{
															{
																Name:  "PAULS_AMAZEBALLS_ENV_VAR",
																Value: "yeaaah",
															},
															{
																Name:  "PHILS_COOL_ENV_VAR",
																Value: "noboi",
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},

			EnvVars: map[string]string{
				"PHILS_COOL_ENV_VAR": "yesboi",
			},
			validate: func(csv *operatorsv1alpha1.ClusterServiceVersion, updated bool, err error) error {
				envs := csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs[0].Spec.Template.Spec.Containers[0].Env
				for _, e := range envs {
					if e.Name == "PHILS_COOL_ENV_VAR" {
						if e.Value != "yesboi" {
							return errors.New(fmt.Sprintf("Expected env var {PHILS_COOL_ENV_VAR: yesboi}, got %+v", envs))
						}
					} else if e.Name == "PAULS_AMAZEBALLS_ENV_VAR" {
						if e.Value != "yeaaah" {
							return errors.New(fmt.Sprintf("Expected env var {PAULS_AMAZEBALLS_ENV_VAR: yeaaah}, got %+v", envs))
						}
					} else {
						return errors.New(fmt.Sprintf("unexpected env var (should only have PHILS_COOL_ENV_VAR and PAULS_AMAZEBALLS_ENV_VAR): %+v", envs))
					}
				}
				if !updated {
					return errors.New(fmt.Sprintf("Expected updated to be true, but got: %v", updated))
				}
				return nil
			},
		},
		{
			name: "updated set to false with no changes",
			args: args{
				kc: &keycloak.Keycloak{
					Status: keycloak.KeycloakStatus{
						Version: "1.0.0",
					},
				},
				operatorVersion: integreatlyv1alpha1.OperatorVersion("1.1.0"),
			},
			CSV: &operatorsv1alpha1.ClusterServiceVersion{
				Spec: operatorsv1alpha1.ClusterServiceVersionSpec{
					InstallStrategy: olmv1alpha1.NamedInstallStrategy{
						StrategySpec: olmv1alpha1.StrategyDetailsDeployment{
							DeploymentSpecs: []olmv1alpha1.StrategyDeploymentSpec{
								{
									Name: "keycloak-operator",
									Spec: appsv1.DeploymentSpec{
										Template: corev1.PodTemplateSpec{
											Spec: corev1.PodSpec{
												Containers: []corev1.Container{
													{
														Env: []corev1.EnvVar{
															{
																Name:  "PAULS_AMAZEBALLS_ENV_VAR",
																Value: "yeaaah",
															},
															{
																Name:  "PHILS_COOL_ENV_VAR",
																Value: "yesboi",
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},

			EnvVars: map[string]string{
				"PHILS_COOL_ENV_VAR": "yesboi",
			},
			validate: func(csv *operatorsv1alpha1.ClusterServiceVersion, updated bool, err error) error {
				envs := csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs[0].Spec.Template.Spec.Containers[0].Env
				for _, e := range envs {
					if e.Name == "PHILS_COOL_ENV_VAR" {
						if e.Value != "yesboi" {
							return errors.New(fmt.Sprintf("Expected env var {PHILS_COOL_ENV_VAR: yesboi}, got %+v", envs))
						}
					} else if e.Name == "PAULS_AMAZEBALLS_ENV_VAR" {
						if e.Value != "yeaaah" {
							return errors.New(fmt.Sprintf("Expected env var {PAULS_AMAZEBALLS_ENV_VAR: yeaaah}, got %+v", envs))
						}
					} else {
						return errors.New(fmt.Sprintf("unexpected env var (should only have PHILS_COOL_ENV_VAR and PAULS_AMAZEBALLS_ENV_VAR): %+v", envs))
					}
				}
				if updated {
					return errors.New(fmt.Sprintf("Expected updated to be false, but got: %v", updated))
				}
				return nil
			},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager:         scenario.fields.ConfigManager,
				mpm:                   scenario.fields.mpm,
				Installation:          scenario.fields.Installation,
				Log:                   scenario.fields.Log,
				Oauthv1Client:         scenario.fields.Oauthv1Client,
				APIURL:                scenario.fields.APIURL,
				Reconciler:            scenario.fields.Reconciler,
				Recorder:              scenario.fields.Recorder,
				KeycloakClientFactory: scenario.fields.KeycloakClientFactory,
			}
			err := scenario.validate(r.ReconcileCSVEnvVars(scenario.CSV, scenario.EnvVars))
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
