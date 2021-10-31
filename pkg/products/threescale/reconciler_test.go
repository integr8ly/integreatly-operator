package threescale

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"

	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	k8sTypes "k8s.io/apimachinery/pkg/types"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	openshiftv1 "github.com/openshift/api/apps/v1"
	configv1 "github.com/openshift/api/config/v1"
	appsv1 "k8s.io/api/apps/v1"

	crov1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	threescalev1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
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

	openshiftappsv1 "github.com/openshift/api/apps/v1"
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
	localProductDeclaration      = marketplace.LocalProductDeclaration("integreatly-3scale")
)

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := threescalev1.SchemeBuilder.AddToScheme(scheme)
	err = rbacv1.SchemeBuilder.AddToScheme(scheme)
	err = keycloak.SchemeBuilder.AddToScheme(scheme)
	err = integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	err = operatorsv1alpha1.AddToScheme(scheme)
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
	err = configv1.AddToScheme(scheme)

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
	Uninstall            bool
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
			Kind:       "RHMI",
			APIVersion: integreatlyv1alpha1.GroupVersion.String(),
		},
	}
}

func getTestBlobStorage() *crov1.BlobStorage {
	return &crov1.BlobStorage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "threescale-blobstorage-rhmi",
			Namespace: "test",
		},
		Status: types.ResourceTypeStatus{
			Phase: types.PhaseComplete,
			SecretRef: &types.SecretRef{
				Name:      "test",
				Namespace: "test",
			},
		},
		Spec: types.ResourceTypeSpec{
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
					Kind:       "RHMI",
					APIVersion: integreatlyv1alpha1.GroupVersion.String(),
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
			Uninstall:      false,
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

			tsReconciler, err := NewReconciler(configManager, scenario.Installation, scenario.FakeAppsV1Client, scenario.FakeOauthClient, scenario.FakeThreeScaleClient, scenario.MPM, scenario.Recorder, getLogger(), localProductDeclaration)
			if err != nil {
				t.Fatalf("Error creating new reconciler %s: %v", constants.ThreeScaleSubscriptionName, err)
			}
			status, err := tsReconciler.Reconcile(ctx, scenario.Installation, scenario.Product, scenario.FakeSigsClient, &quota.ProductConfigMock{}, scenario.Uninstall)
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
				},
					&threescalev1.APIManager{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "3scale",
							Namespace: "test",
						},
						Spec:   threescalev1.APIManagerSpec{},
						Status: threescalev1.APIManagerStatus{},
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
			got, err := r.reconcileComponents(tt.args.ctx, tt.args.serverClient, &quota.ProductConfigMock{})
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

func TestReconciler_ensureDeploymentConfigsReady(t *testing.T) {
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
		extraParams   map[string]string
		recorder      record.EventRecorder
		log           l.Logger
	}
	type args struct {
		ctx              context.Context
		serverClient     k8sclient.Client
		productNamespace string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "Test - Unable to get deployment config - PhaseFailed",
			args: args{
				ctx: context.TODO(),
				serverClient: &moqclient.SigsClientInterfaceMock{GetFunc: func(ctx context.Context, key k8sTypes.NamespacedName, obj runtime.Object) error {
					return fmt.Errorf("fetch error")
				}},
				productNamespace: defaultInstallationNamespace,
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "Test - Deployment config has a 'False' condition - Rollout success - PhaseCreatingComponents",
			fields: fields{
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
				appsv1Client: getAppsV1Client(successfulTestAppsV1Objects),
				log:          getLogger(),
			},
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme,
					&openshiftappsv1.DeploymentConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "apicast-production",
							Namespace: defaultInstallationNamespace,
							Labels: map[string]string{
								"app": "apicast-production",
							},
						},
						Status: openshiftappsv1.DeploymentConfigStatus{
							Conditions: []openshiftappsv1.DeploymentCondition{
								{
									Status: corev1.ConditionFalse,
								},
							},
						},
					},
				),
				productNamespace: defaultInstallationNamespace,
			},
			want: integreatlyv1alpha1.PhaseCreatingComponents,
		},
		{
			name: "Test - Waiting for replicas to be rolled out - Condition Unknown - PhaseInProgress",
			fields: fields{
				log: getLogger(),
			},
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme,
					&openshiftappsv1.DeploymentConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "apicast-production",
							Namespace: defaultInstallationNamespace,
							Labels: map[string]string{
								"app": "apicast-production",
							},
						},
						Status: openshiftappsv1.DeploymentConfigStatus{
							Conditions: []openshiftappsv1.DeploymentCondition{
								{
									Status: corev1.ConditionUnknown,
								},
							},
						},
					},
				),
				productNamespace: defaultInstallationNamespace,
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: true,
		},
		{
			name: "Test - Waiting for replicas to be rolled out - Replicas != AvailableReplicas - PhaseInProgress",
			fields: fields{
				log: getLogger(),
			},
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme,
					&openshiftappsv1.DeploymentConfig{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "apicast-production",
							Namespace: defaultInstallationNamespace,
							Labels: map[string]string{
								"app": "apicast-production",
							},
						},
						Status: openshiftappsv1.DeploymentConfigStatus{
							Conditions: []openshiftappsv1.DeploymentCondition{
								{
									Status: corev1.ConditionTrue,
								},
							},
							Replicas:      1,
							ReadyReplicas: 0,
						},
					},
				),
				productNamespace: defaultInstallationNamespace,
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: true,
		},
		{
			name: "Test - Waiting for replicas to be rolled out - ReadyReplicas != UpdatedReplicas - PhaseInProgress",
			fields: fields{
				log: getLogger(),
			},
			args: args{
				ctx: context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme,
					&openshiftappsv1.DeploymentConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "apicast-production",
							Namespace: defaultInstallationNamespace,
							Labels: map[string]string{
								"app": "apicast-production",
							},
						},
						Status: openshiftappsv1.DeploymentConfigStatus{
							Conditions: []openshiftappsv1.DeploymentCondition{
								{
									Status: corev1.ConditionTrue,
								},
							},
							ReadyReplicas:   1,
							UpdatedReplicas: 0,
						},
					},
				),
				productNamespace: defaultInstallationNamespace,
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: true,
		},
		{
			name: "Test - Replicas all rolled out - PhaseComplete",
			args: args{
				ctx:              context.TODO(),
				serverClient:     fake.NewFakeClientWithScheme(scheme, getSuccessfullTestPreReqs(integreatlyOperatorNamespace, defaultInstallationNamespace)...),
				productNamespace: defaultInstallationNamespace,
			},
			want: integreatlyv1alpha1.PhaseCompleted,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
				extraParams:   tt.fields.extraParams,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			got, err := r.ensureDeploymentConfigsReady(tt.args.ctx, tt.args.serverClient, tt.args.productNamespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureDeploymentConfigsReady() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ensureDeploymentConfigsReady() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_reconcileOpenshiftUsers(t *testing.T) {
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
		extraParams   map[string]string
		recorder      record.EventRecorder
		log           l.Logger
	}
	type args struct {
		ctx          context.Context
		installation *integreatlyv1alpha1.RHMI
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
			name: "Test - Read RHSSO Config failed - PhaseFailed",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return nil, fmt.Errorf("read error")
				}},
				log: getLogger(),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "Test - System seed secret failed - PhaseFailed",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
				log: getLogger(),
			},
			args: args{
				serverClient: &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key k8sTypes.NamespacedName, obj runtime.Object) error {
						return fmt.Errorf("get error")
					},
				},
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "Test - Get Keycloak users failed - PhaseFailed",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
				log: getLogger(),
			},
			args: args{
				serverClient: &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key k8sTypes.NamespacedName, obj runtime.Object) error {
						return nil
					},
					ListFunc: func(ctx context.Context, list runtime.Object, opts ...k8sclient.ListOption) error {
						return fmt.Errorf("list error")
					},
				},
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "Test - Get 3scale Users failed - PhaseInProgress",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
				log: getLogger(),
				tsClient: &ThreeScaleInterfaceMock{
					GetUsersFunc: func(accessToken string) (*Users, error) {
						return nil, fmt.Errorf("get error")
					},
				},
			},
			args: args{
				serverClient: &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key k8sTypes.NamespacedName, obj runtime.Object) error {
						return nil
					},
					ListFunc: func(ctx context.Context, list runtime.Object, opts ...k8sclient.ListOption) error {
						return nil
					},
				},
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: true,
		},
		{
			name: "Test - Reconcile Successful - PhaseComplete",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
				log: getLogger(),
				tsClient: &ThreeScaleInterfaceMock{
					GetUsersFunc: func(accessToken string) (*Users, error) {
						return &Users{
							Users: []*User{
								{
									UserDetails: UserDetails{
										Username: "updated-3scale",
										Id:       1,
									},
								},
								{
									UserDetails: UserDetails{
										Username: "notInKeyCloak",
									},
								},
							},
						}, nil
					},
					AddUserFunc: func(username string, email string, password string, accessToken string) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
						}, nil
					},
					DeleteUserFunc: func(userID int, accessToken string) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
						}, nil
					},
					UpdateUserFunc: func(userID int, username string, email string, accessToken string) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
						}, nil
					},
					GetUserFunc: func(username string, accessToken string) (*User, error) {
						return &User{
							UserDetails: UserDetails{
								Username: defaultInstallationNamespace,
								Id:       1,
							},
						}, nil
					},
				},
			},
			args: args{
				serverClient: fake.NewFakeClientWithScheme(scheme,
					append(getSuccessfullTestPreReqs(integreatlyOperatorNamespace, defaultInstallationNamespace),
						&keycloak.KeycloakUser{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "generated-3scale",
								Namespace: "rhsso",
								Labels: map[string]string{
									"sso": "integreatly",
								},
							},
							Spec: keycloak.KeycloakUserSpec{
								User: keycloak.KeycloakAPIUser{
									UserName: defaultInstallationNamespace,
									Attributes: map[string][]string{
										user3ScaleID: {fmt.Sprint(1)},
									},
								},
							},
						},
					)...),
				installation: &integreatlyv1alpha1.RHMI{
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: string(integreatlyv1alpha1.InstallationTypeManagedApi),
					},
				},
			},
			want: integreatlyv1alpha1.PhaseCompleted,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
				extraParams:   tt.fields.extraParams,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			got, err := r.reconcileOpenshiftUsers(tt.args.ctx, tt.args.installation, tt.args.serverClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileOpenshiftUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("reconcileOpenshiftUsers() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_updateKeycloakUsersAttributeWith3ScaleUserId(t *testing.T) {
	accessToken := "accessToken"

	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		tsClient      ThreeScaleInterface
		appsv1Client  appsv1Client.AppsV1Interface
		oauthv1Client oauthClient.OauthV1Interface
		Reconciler    *resources.Reconciler
		extraParams   map[string]string
		recorder      record.EventRecorder
		log           l.Logger
	}
	type args struct {
		ctx          context.Context
		serverClient k8sclient.Client
		kcu          []keycloak.KeycloakAPIUser
		accessToken  *string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "Test - Read RHSSO Config failed - PhaseFailed",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return nil, fmt.Errorf("read error")
				}},
				log: getLogger(),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "Test - Read RHSSO Config failed - PhaseFailed",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return nil, fmt.Errorf("read error")
				}},
				log: getLogger(),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "Test - Get 3scale User failed - Continued - PhaseCompleted",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
				log: getLogger(),
				tsClient: &ThreeScaleInterfaceMock{
					GetUserFunc: func(username string, accessToken string) (*User, error) {
						return nil, fmt.Errorf("get error")
					},
				},
			},
			args: args{
				kcu: []keycloak.KeycloakAPIUser{
					{
						UserName: defaultInstallationNamespace,
					},
				},
				accessToken: &accessToken,
			},
			want: integreatlyv1alpha1.PhaseCompleted,
		},
		{
			name: "Test - Update Keycloak User failed - PhaseInProgress",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
				log: getLogger(),
				tsClient: &ThreeScaleInterfaceMock{
					GetUserFunc: func(username string, accessToken string) (*User, error) {
						return &User{UserDetails: UserDetails{Id: 1}}, nil
					},
				},
			},
			args: args{
				kcu: []keycloak.KeycloakAPIUser{
					{
						UserName: defaultInstallationNamespace,
					},
				},
				accessToken: &accessToken,
				serverClient: &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key k8sTypes.NamespacedName, obj runtime.Object) error {
						return fmt.Errorf("get error")
					},
				},
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: true,
		},
		{
			name: "Test - Update Keycloak User successful - PhaseComplete",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
				log: getLogger(),
				tsClient: &ThreeScaleInterfaceMock{
					GetUserFunc: func(username string, accessToken string) (*User, error) {
						return &User{UserDetails: UserDetails{Id: 1}}, nil
					},
				},
			},
			args: args{
				kcu: []keycloak.KeycloakAPIUser{
					{
						UserName: "test",
					},
				},
				accessToken: &accessToken,
				serverClient: &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key k8sTypes.NamespacedName, obj runtime.Object) error {
						return nil
					},
					UpdateFunc: func(ctx context.Context, obj runtime.Object, opts ...k8sclient.UpdateOption) error {
						return nil
					},
				},
			},
			want: integreatlyv1alpha1.PhaseCompleted,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
				extraParams:   tt.fields.extraParams,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			got, err := r.updateKeycloakUsersAttributeWith3ScaleUserId(tt.args.ctx, tt.args.serverClient, tt.args.kcu, tt.args.accessToken)
			if (err != nil) != tt.wantErr {
				t.Errorf("updateKeycloakUsersAttributeWith3ScaleUserId() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("updateKeycloakUsersAttributeWith3ScaleUserId() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_getUserDiff(t *testing.T) {
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
		extraParams   map[string]string
		recorder      record.EventRecorder
		log           l.Logger
	}
	type args struct {
		ctx          context.Context
		serverClient k8sclient.Client
		kcUsers      []keycloak.KeycloakAPIUser
		tsUsers      []*User
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []keycloak.KeycloakAPIUser
		want1  []*User
		want2  []*User
	}{
		{
			name: "Test - Read RHSSO Config failed - PhaseFailed",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return nil, fmt.Errorf("read error")
				}},
				log: getLogger(),
			},
			want:  nil,
			want1: nil,
			want2: nil,
		},
		{
			name: "Test - Keycloak User not in 3scale appended to added",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				log: getLogger(),
			},
			args: args{
				tsUsers: []*User{
					{
						UserDetails: UserDetails{
							Username: defaultInstallationNamespace,
						},
					},
				},
				kcUsers: []keycloak.KeycloakAPIUser{
					{
						UserName: "NEW-3SCALE",
					},
					{
						UserName: defaultInstallationNamespace,
					},
				},
			},
			want: []keycloak.KeycloakAPIUser{
				{
					UserName: "NEW-3SCALE",
				},
			},
			want1: nil,
			want2: nil,
		},
		{
			name: "Test - 3scale User not in Keycloak appended to deleted & comparison is case sensitive",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				log: getLogger(),
			},
			args: args{
				tsUsers: []*User{
					{
						UserDetails: UserDetails{
							Username: defaultInstallationNamespace,
						},
					},
					{
						UserDetails: UserDetails{
							Username: "3SCALE",
						},
					},
				},
				kcUsers: []keycloak.KeycloakAPIUser{
					{
						UserName: defaultInstallationNamespace,
					},
				},
				serverClient: fake.NewFakeClientWithScheme(scheme, &keycloak.KeycloakUser{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "generated-3scale",
						Namespace: "rhsso",
					},
				}),
			},
			want: nil,
			want1: []*User{
				{
					UserDetails: UserDetails{
						Username: "3SCALE",
					},
				},
			},
			want2: nil,
		},
		{
			name: "Test - Get keycloak user for deletion failed - appended to deleted",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				log: getLogger(),
			},
			args: args{
				tsUsers: []*User{
					{
						UserDetails: UserDetails{
							Username: defaultInstallationNamespace,
						},
					},
					{
						UserDetails: UserDetails{
							Username: "notInKeyCloak",
						},
					},
				},
				kcUsers: []keycloak.KeycloakAPIUser{
					{
						UserName: defaultInstallationNamespace,
					},
				},
				serverClient: fake.NewFakeClientWithScheme(scheme),
			},
			want: nil,
			want1: []*User{
				{
					UserDetails: UserDetails{
						Username: "notInKeyCloak",
					},
				},
			},
			want2: nil,
		},
		{
			name: "Test - 3scale User updated appended to updated and to added",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				log: getLogger(),
			},
			args: args{
				tsUsers: []*User{
					{
						UserDetails: UserDetails{
							Username: fmt.Sprintf("Updated-%s", defaultInstallationNamespace),
							Id:       1,
						},
					},
				},
				kcUsers: []keycloak.KeycloakAPIUser{
					{
						UserName: defaultInstallationNamespace,
					},
				},
				serverClient: fake.NewFakeClientWithScheme(scheme, &keycloak.KeycloakUser{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("generated-%s", defaultInstallationNamespace),
						Namespace: "rhsso",
					},
					Spec: keycloak.KeycloakUserSpec{
						User: keycloak.KeycloakAPIUser{
							Attributes: map[string][]string{
								user3ScaleID: {fmt.Sprint(1)},
							},
						},
					},
				}),
			},
			want: []keycloak.KeycloakAPIUser{
				{
					UserName: defaultInstallationNamespace,
				},
			},
			want1: nil,
			want2: []*User{
				{
					UserDetails: UserDetails{
						Username: fmt.Sprintf("Updated-%s", defaultInstallationNamespace),
						Id:       1,
					},
				},
			},
		},
		{
			name: "Test - 3scale User with same uppercase name in KeyCloak CR appended to updated and to added",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{ReadRHSSOFunc: func() (*config.RHSSO, error) {
					return config.NewRHSSO(config.ProductConfig{
						"NAMESPACE": "rhsso",
					}), nil
				}},
				log: getLogger(),
			},
			args: args{
				tsUsers: []*User{
					{
						UserDetails: UserDetails{
							Username: fmt.Sprintf("UpperCase-%s", defaultInstallationNamespace),
							Id:       1,
						},
					},
				},
				kcUsers: []keycloak.KeycloakAPIUser{
					{
						UserName: fmt.Sprintf("UpperCase-%s", defaultInstallationNamespace),
						Attributes: map[string][]string{
							user3ScaleID: {fmt.Sprint(1)},
						},
					},
				},
				serverClient: fake.NewFakeClientWithScheme(scheme, &keycloak.KeycloakUser{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("generated-uppercase-%s", defaultInstallationNamespace),
						Namespace: "rhsso",
					},
					Spec: keycloak.KeycloakUserSpec{
						User: keycloak.KeycloakAPIUser{
							Attributes: map[string][]string{
								user3ScaleID: {fmt.Sprint(1)},
							},
							UserName: fmt.Sprintf("UpperCase-%s", defaultInstallationNamespace),
						},
					},
				}),
			},
			want: []keycloak.KeycloakAPIUser{
				{
					UserName: fmt.Sprintf("UpperCase-%s", defaultInstallationNamespace),
					Attributes: map[string][]string{
						user3ScaleID: {fmt.Sprint(1)},
					},
				},
			},
			want1: nil,
			want2: []*User{
				{
					UserDetails: UserDetails{
						Username: fmt.Sprintf("UpperCase-%s", defaultInstallationNamespace),
						Id:       1,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				tsClient:      tt.fields.tsClient,
				appsv1Client:  tt.fields.appsv1Client,
				oauthv1Client: tt.fields.oauthv1Client,
				Reconciler:    tt.fields.Reconciler,
				extraParams:   tt.fields.extraParams,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			got, got1, got2 := r.getUserDiff(tt.args.ctx, tt.args.serverClient, tt.args.kcUsers, tt.args.tsUsers)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getUserDiff() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("getUserDiff() got1 = %v, want %v", got1, tt.want1)
			}
			if !reflect.DeepEqual(got2, tt.want2) {
				t.Errorf("getUserDiff() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func TestReconciler_reconcileBlackboxTargets(t *testing.T) {
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
			name: "Test - reconcileBlackboxTargets - PhaseCompleted",
			fields: fields{
				installation: getTestInstallation(),
				ConfigManager: &config.ConfigReadWriterMock{ReadMonitoringFunc: func() (*config.Monitoring, error) {
					return config.NewMonitoring(config.ProductConfig{
						"NAMESPACE": "3scale",
					}), nil
				}},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
			},
			args: args{
				ctx:          context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme, getSuccessfullTestPreReqs(integreatlyOperatorNamespace, defaultInstallationNamespace)...),
			},
			want: integreatlyv1alpha1.PhaseCompleted,
		},
		{
			name: "Test - reconcileBlackboxTargets - PhaseInProgress",
			fields: fields{
				installation: getTestInstallation(),
				ConfigManager: &config.ConfigReadWriterMock{ReadMonitoringFunc: func() (*config.Monitoring, error) {
					return config.NewMonitoring(config.ProductConfig{
						"NAMESPACE": "3scale",
					}), nil
				}},
				Config: config.NewThreeScale(config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				}),
			},
			args: args{
				ctx:          context.TODO(),
				serverClient: fake.NewFakeClientWithScheme(scheme, getTestPreReqsReconcileBlackboxTargetsPhaseInProgress()...),
			},
			want: integreatlyv1alpha1.PhaseInProgress,
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
			got, err := r.reconcileBlackboxTargets(tt.args.ctx, tt.args.serverClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileBlackboxTargets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("reconcileBlackboxTargets() got = %v, want %v", got, tt.want)
			}
		})
	}
}
