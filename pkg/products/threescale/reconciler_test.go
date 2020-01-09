package threescale

import (
	"context"
	"testing"

	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"

	threescalev1 "github.com/integr8ly/integreatly-operator/pkg/apis/3scale/v1alpha1"
	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	oauthv1 "github.com/openshift/api/oauth/v1"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	appsv1Client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	fakeoauthClient "github.com/openshift/client-go/oauth/clientset/versioned/fake"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	marketplacev1 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	marketplacev2 "github.com/operator-framework/operator-marketplace/pkg/apis/operators/v2"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	integreatlyOperatorNamespace = "integreatly-operator-namespace"
)

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := threescalev1.SchemeBuilder.AddToScheme(scheme)
	err = aerogearv1.SchemeBuilder.AddToScheme(scheme)
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
	return scheme, err
}

type ThreeScaleTestScenario struct {
	Name                 string
	Installation         *integreatlyv1alpha1.Installation
	FakeSigsClient       k8sclient.Client
	FakeAppsV1Client     appsv1Client.AppsV1Interface
	FakeOauthClient      oauthClient.OauthV1Interface
	FakeThreeScaleClient *ThreeScaleInterfaceMock
	ExpectedStatus       integreatlyv1alpha1.StatusPhase
	Assert               AssertFunc
	MPM                  marketplace.MarketplaceInterface
	Product              *integreatlyv1alpha1.InstallationProductStatus
}

func getTestInstallation() *integreatlyv1alpha1.Installation {
	return &integreatlyv1alpha1.Installation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: integreatlyv1alpha1.InstallationSpec{
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
			Name:      "threescale-blobstorage-test",
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
			Installation: &integreatlyv1alpha1.Installation{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-installation",
					Namespace:  "integreatly-operator-namespace",
					Finalizers: []string{"finalizer.3scale.integreatly.org"},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       integreatlyv1alpha1.SchemaGroupVersionKind.Kind,
					APIVersion: integreatlyv1alpha1.SchemeGroupVersion.String(),
				},
				Spec: integreatlyv1alpha1.InstallationSpec{
					MasterURL:        "https://console.apps.example.com",
					RoutingSubdomain: "apps.example.com",
				},
			},
			MPM:            marketplace.NewManager(),
			ExpectedStatus: integreatlyv1alpha1.PhaseCompleted,
			Product:        &integreatlyv1alpha1.InstallationProductStatus{},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			ctx := context.TODO()
			configManager, err := config.NewManager(ctx, scenario.FakeSigsClient, configManagerConfigMap.Namespace, configManagerConfigMap.Name, scenario.Installation)
			if err != nil {
				t.Fatalf("Error creating config manager")
			}

			tsReconciler, err := NewReconciler(configManager, scenario.Installation, scenario.FakeAppsV1Client, scenario.FakeOauthClient, scenario.FakeThreeScaleClient, scenario.MPM)
			if err != nil {
				t.Fatalf("Error creating new reconciler %s: %v", packageName, err)
			}
			status, err := tsReconciler.Reconcile(ctx, scenario.Installation, scenario.Product, scenario.FakeSigsClient)
			if err != nil {
				t.Fatalf("Error reconciling %s: %v", packageName, err)
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
		installation  *integreatlyv1alpha1.Installation
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
		installation  *integreatlyv1alpha1.Installation
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
