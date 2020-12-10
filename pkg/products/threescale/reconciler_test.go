package threescale

import (
	"context"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"net/http"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	openshiftv1 "github.com/openshift/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"

	threescalev1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"

	oauthv1 "github.com/openshift/api/oauth/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	appsv1Client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	fakeoauthClient "github.com/openshift/client-go/oauth/clientset/versioned/fake"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	marketplacev2 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v2"

	consolev1 "github.com/openshift/api/console/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	integreatlyOperatorNamespace = "integreatly-operator-ns"
)

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := threescalev1.SchemeBuilder.AddToScheme(scheme)
	err = rbacv1.SchemeBuilder.AddToScheme(scheme)
	err = keycloak.SchemeBuilder.AddToScheme(scheme)
	err = integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	err = operatorsv1alpha1.AddToScheme(scheme)
	err = marketplacev1.SchemeBuilder.AddToScheme(scheme)
	err = marketplacev2.SchemeBuilder.AddToScheme(scheme)
	err = corev1.SchemeBuilder.AddToScheme(scheme)
	err = coreosv1.SchemeBuilder.AddToScheme(scheme)
	err = crov1.SchemeBuilder.AddToScheme(scheme)
	err = usersv1.AddToScheme(scheme)
	err = oauthv1.AddToScheme(scheme)
	err = routev1.AddToScheme(scheme)
	err = projectv1.AddToScheme(scheme)
	err = appsv1.AddToScheme(scheme)
	err = monitoringv1.AddToScheme(scheme)
	err = consolev1.AddToScheme(scheme)
	err = openshiftv1.AddToScheme(scheme)
	return scheme, err
}

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
}

type ThreeScaleTestScenario struct {
	Name                 string
	Installation         *integreatlyv1alpha1.RHMI
	FakeSigsClient       k8sclient.Client
	FakeAppsV1Client     appsv1Client.AppsV1Interface
	FakeOauthClient      oauthClient.OauthV1Interface
	FakeThreeScaleClient *ThreeScaleInterfaceMock
	ExpectedStatus       integreatlyv1alpha1.StatusPhase
	Assert               AssertFunc
	MPM                  marketplace.MarketplaceInterface
	Product              *integreatlyv1alpha1.RHMIProductStatus
	Recorder             record.EventRecorder
}

func getTestInstallation() *integreatlyv1alpha1.RHMI {
	return &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhmi",
			Namespace: "test",
		},
		Spec: integreatlyv1alpha1.RHMISpec{
			Type: "managed",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       integreatlyv1alpha1.SchemaGroupVersionKind.Kind,
			APIVersion: integreatlyv1alpha1.SchemeGroupVersion.String(),
		},
	}
}

func getTestBlobStorage() *crov1.BlobStorage {
	return &crov1.BlobStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "threescale-blobstorage-rhmi",
			Namespace: "test",
		},
		Status: crov1.BlobStorageStatus{
			Phase: types.PhaseComplete,
			SecretRef: &types.SecretRef{
				Name:      "test",
				Namespace: "test",
			},
		},
		Spec: crov1.BlobStorageSpec{
			SecretRef: &types.SecretRef{
				Name:      "test",
				Namespace: "test",
			},
		},
	}
}

func TestThreeScale(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("Error creating build scheme")
	}

	scenarios := []ThreeScaleTestScenario{
		{
			Name:                 "Test successful installation without errors",
			FakeSigsClient:       getSigClient(getSuccessfullTestPreReqs(integreatlyOperatorNamespace, defaultInstallationNamespace), scheme),
			FakeAppsV1Client:     getAppsV1Client(successfulTestAppsV1Objects),
			FakeOauthClient:      fakeoauthClient.NewSimpleClientset([]runtime.Object{}...).OauthV1(),
			FakeThreeScaleClient: getThreeScaleClient(),
			Assert:               assertInstallationSuccessfull,
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-installation",
					Namespace:  "integreatly-operator-ns",
					Finalizers: []string{"finalizer.3scale.integreatly.org"},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       integreatlyv1alpha1.SchemaGroupVersionKind.Kind,
					APIVersion: integreatlyv1alpha1.SchemeGroupVersion.String(),
				},
				Spec: integreatlyv1alpha1.RHMISpec{
					MasterURL:        "https://console.apps.example.com",
					RoutingSubdomain: "apps.example.com",
					SMTPSecret:       "test-smtp",
				},
			},
			MPM:            marketplace.NewManager(),
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Product:        &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:       setupRecorder(),
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			ctx := context.TODO()
			configManager, err := config.NewManager(ctx, scenario.FakeSigsClient, configManagerConfigMap.Namespace, configManagerConfigMap.Name, scenario.Installation)
			if err != nil {
				t.Fatalf("Error creating config manager")
			}

			err = configManager.Client.Create(ctx, smtpSec)
			if err != nil {
				t.Fatalf("Error creating config manager")
			}

			tsReconciler, err := NewReconciler(configManager, scenario.Installation, scenario.FakeAppsV1Client, scenario.FakeOauthClient, scenario.FakeThreeScaleClient, scenario.MPM, scenario.Recorder, getLogger())
			if err != nil {
				t.Fatalf("Error creating new reconciler %s: %v", constants.ThreeScaleSubscriptionName, err)
			}
			status, err := tsReconciler.Reconcile(ctx, scenario.Installation, scenario.Product, scenario.FakeSigsClient)
			if err != nil {
				t.Fatalf("Error reconciling %s: %v", constants.ThreeScaleSubscriptionName, err)
			}

			if status != scenario.ExpectedStatus {
				t.Fatalf("unexpected status: %v, expected: %v", status, scenario.ExpectedStatus)
			}

			err = scenario.Assert(scenario, configManager)
			if err != nil {
				t.Fatal(err.Error())
			}
		})
	}
}

func TestReconciler_reconcileBlobStorage(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		tsClient      ThreeScaleInterface
		appsv1Client  appsv1Client.AppsV1Interface
		oauthv1Client oauthClient.OauthV1Interface
		Reconciler    *resources.Reconciler
	}
	type args struct {
		ctx          context.Context
		serverClient k8sclient.Client
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "test successful reconcile",
			fields: fields{
				ConfigManager: nil,
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				mpm:           nil,
				installation:  getTestInstallation(),
				tsClient:      nil,
				appsv1Client:  nil,
				oauthv1Client: nil,
				Reconciler:    nil,
			},
			args: args{
				ctx:          context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme, getTestBlobStorage()),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				log:           getLogger(),
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
			}
			got, err := r.reconcileBlobStorage(tt.args.ctx, tt.args.serverClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileBlobStorage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("reconcileBlobStorage() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_reconcileComponents(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		tsClient      ThreeScaleInterface
		appsv1Client  appsv1Client.AppsV1Interface
		oauthv1Client oauthClient.OauthV1Interface
		Reconciler    *resources.Reconciler
	}
	type args struct {
		ctx          context.Context
		serverClient k8sclient.Client
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "test successful reconcile of s3 blob storage",
			fields: fields{
				ConfigManager: nil,
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				mpm:           nil,
				installation:  getTestInstallation(),
				tsClient:      nil,
				appsv1Client:  nil,
				oauthv1Client: nil,
				Reconciler:    nil,
			},
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme, getTestBlobStorage(), &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Data: map[string][]byte{
						"credentialKeyID":     []byte("test"),
						"credentialSecretKey": []byte("test"),
					},
				}),
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				log:           getLogger(),
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
			}
			got, err := r.reconcileComponents(tt.args.ctx, tt.args.serverClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileComponents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("reconcileComponents() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_syncOpenshiftAdmimMembership(t *testing.T) {
	calledSetUserAsAdmin := false

	tsClientMock := ThreeScaleInterfaceMock{
		SetUserAsAdminFunc: func(userID int, accessToken string) (*http.Response, error) {
			if userID != 1 {
				t.Fatalf("Unexpected user promoted to admin. Expected User with ID 1, got user with ID %d", userID)
			} else {
				calledSetUserAsAdmin = true
			}

			return &http.Response{
				StatusCode: 200,
			}, nil
		},
		SetUserAsMemberFunc: func(userID int, accessToken string) (*http.Response, error) {
			t.Fatalf("Unexpected call to `SetUserAsMember`. Called with userID %d", userID)

			return &http.Response{
				StatusCode: 200,
			}, nil
		},
	}

	openshiftAdminGroup := &usersv1.Group{
		Users: usersv1.OptionalNames{
			"user1",
			"user2",
		},
	}

	newTsUsers := &Users{
		Users: []*User{
			{
				UserDetails: UserDetails{
					Id:   1,
					Role: memberRole,
					// User is in OS admin group. Should be promoted
					Username: "User1",
				},
			},
			{
				UserDetails: UserDetails{
					Id:   2,
					Role: adminRole,
					// User is in OS admin group and admin in 3scale. Should
					// be ignored
					Username: "User2",
				},
			},
			{
				UserDetails{
					Id:   3,
					Role: adminRole,
					// User is not in OS admin group but is already admin.
					// Should NOT be demoted
					Username: "User3",
				},
			},
		},
	}

	err := syncOpenshiftAdminMembership(openshiftAdminGroup, newTsUsers, "", false, &tsClientMock, "")

	if err != nil {
		t.Fatalf("Unexpected error when reconcilling openshift admin membership: %s", err)
	}

	if !calledSetUserAsAdmin {
		t.Fatal("Expected user with ID 1 to be promoted as admin, but no promotion was invoked")
	}
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.Product3Scale})
}
