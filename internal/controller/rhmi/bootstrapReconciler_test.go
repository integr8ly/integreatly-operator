package controllers

import (
	"context"
	"errors"
	"fmt"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	userHelper "github.com/integr8ly/integreatly-operator/pkg/resources/user"
	"github.com/integr8ly/integreatly-operator/utils"
	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	rhoamOperatorNs      = "redhat-rhoam-operator"
	threescaleNs         = "redhat-rhoam-3scale"
	threescaleOperatorNs = "redhat-rhoam-3scale-operator"
	croNs                = "redhat-rhoam-cloud-resources"
	customerMonitoringNs = "redhat-rhoam-customer-monitoring"
	marin3rNs            = "redhat-rhoam-marin3r"
	marin3rOperatorNs    = "redhat-rhoam-marin3r-operator"
	oboNs                = "redhat-rhoam-operator-observability"
	rhssoNs              = "redhat-rhoam-rhsso"
	rhssoOperatorNs      = "redhat-rhoam-rhsso-operator"
	userSsoNs            = "redhat-rhoam-user-sso"
	userSsoOperatorNs    = "redhat-rhoam-user-sso-operator"
	someRandomNs         = "some-random-nspace"
)

func getNamespaces() *corev1.NamespaceList {
	return &corev1.NamespaceList{
		TypeMeta: v1.TypeMeta{},
		ListMeta: v1.ListMeta{},
		Items: []corev1.Namespace{
			{
				ObjectMeta: v1.ObjectMeta{
					Name: rhoamOperatorNs,
				},
			},
			{
				ObjectMeta: v1.ObjectMeta{
					Name: threescaleNs,
					Labels: map[string]string{
						"integreatly": "true",
					},
				},
			},
			{
				ObjectMeta: v1.ObjectMeta{
					Name: threescaleOperatorNs,
					Labels: map[string]string{
						"integreatly": "true",
					},
				},
			},
			{
				ObjectMeta: v1.ObjectMeta{
					Name: croNs,
					Labels: map[string]string{
						"integreatly": "true",
					},
				},
			},
			{
				ObjectMeta: v1.ObjectMeta{
					Name: customerMonitoringNs,
					Labels: map[string]string{
						"integreatly": "true",
					},
				},
			},
			{
				ObjectMeta: v1.ObjectMeta{
					Name: marin3rNs,
					Labels: map[string]string{
						"integreatly": "true",
					},
				},
			},
			{
				ObjectMeta: v1.ObjectMeta{
					Name: marin3rOperatorNs,
					Labels: map[string]string{
						"integreatly": "true",
					},
				},
			},
			{
				ObjectMeta: v1.ObjectMeta{
					Name: oboNs,
					Labels: map[string]string{
						"integreatly": "true",
					},
				},
			},
			{
				ObjectMeta: v1.ObjectMeta{
					Name: rhssoNs,
					Labels: map[string]string{
						"integreatly": "true",
					},
				},
			},
			{
				ObjectMeta: v1.ObjectMeta{
					Name: rhssoOperatorNs,
					Labels: map[string]string{
						"integreatly": "true",
					},
				},
			},
			{
				ObjectMeta: v1.ObjectMeta{
					Name: userSsoNs,
					Labels: map[string]string{
						"integreatly": "true",
					},
				},
			},
			{
				ObjectMeta: v1.ObjectMeta{
					Name: userSsoOperatorNs,
					Labels: map[string]string{
						"integreatly": "true",
					},
				},
			},
			{
				ObjectMeta: v1.ObjectMeta{
					Name: someRandomNs,
				},
			},
		},
	}
}

func Test_tenantExists(t *testing.T) {
	type args struct {
		user    string
		tenants []userHelper.MultiTenantUser
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Empty list of tenants given",
			args: args{
				user:    "username",
				tenants: []userHelper.MultiTenantUser{},
			},
			want: false,
		},
		{
			name: "Tenant list is nil",
			args: args{
				user:    "username",
				tenants: nil,
			},
			want: false,
		},
		{
			name: "Name not in tenant list given",
			args: args{
				user: "tenantName",
				tenants: []userHelper.MultiTenantUser{
					{
						TenantName: "tenantName01",
					},
					{
						TenantName: "tenantName02",
					},
				},
			},
			want: false,
		},
		{
			name: "Name in list of tenants, list length 1",
			args: args{
				user: "tenantName",
				tenants: []userHelper.MultiTenantUser{
					{
						TenantName: "tenantName",
					},
				},
			},
			want: true,
		},
		{
			name: "Name in list of tenants, list length 2",
			args: args{
				user: "tenantName",
				tenants: []userHelper.MultiTenantUser{
					{
						TenantName: "tenantName01",
					},
					{
						TenantName: "tenantName",
					},
				},
			},
			want: true,
		},
		{
			name: "Tenant name is empty string",
			args: args{
				user: "",
				tenants: []userHelper.MultiTenantUser{
					{
						TenantName: "tenantName01",
					},
					{
						TenantName: "tenantName02",
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if got := tenantExists(tt.args.user, tt.args.tenants); got != tt.want {
				t.Errorf("tenantExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_reconcileAddonManagedApiServiceParameters(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		FakeConfigManager config.ConfigReadWriter
		Config            *config.ThreeScale
		FakeMpm           marketplace.MarketplaceInterface
		installation      *integreatlyv1alpha1.RHMI
		Reconciler        *resources.Reconciler
		recorder          record.EventRecorder
		log               l.Logger
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
			name: "test addon-managed-api-service-parameters secret found",
			fields: fields{
				FakeConfigManager: &config.ConfigReadWriterMock{
					GetOperatorNamespaceFunc: func() string {
						return "redhat-rhoam-operator"
					},
				},
				Config:  &config.ThreeScale{},
				FakeMpm: &marketplace.MarketplaceInterfaceMock{},
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Namespace: rhoamOperatorNs,
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						NamespacePrefix: "redhat-rhoam-",
					},
				},
				Reconciler: &resources.Reconciler{},
				recorder:   record.NewFakeRecorder(50),
				log:        l.Logger{},
			},
			args: args{
				ctx: context.TODO(),
				serverClient: utils.NewTestClient(scheme,
					&corev1.Secret{
						ObjectMeta: v1.ObjectMeta{
							Name:      "addon-managed-api-service-parameters",
							Namespace: rhoamOperatorNs,
						},
					},
					getNamespaces(),
				),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.FakeConfigManager,
				Config:        tt.fields.Config,
				mpm:           tt.fields.FakeMpm,
				installation:  tt.fields.installation,
				Reconciler:    tt.fields.Reconciler,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			got, err := r.reconcileAddonManagedApiServiceParameters(tt.args.ctx, tt.args.serverClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileAddonManagedApiServiceParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("reconcileAddonManagedApiServiceParameters() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_retrieveConsoleURLAndSubdomain(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		Reconciler    *resources.Reconciler
		recorder      record.EventRecorder
		log           l.Logger
	}
	type args struct {
		ctx          context.Context
		serverClient func() k8sclient.Client
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "successfully retrieve console url and subdomain",
			fields: fields{
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Name:      "managed-api",
						Namespace: "test",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: "managed-api",
					},
				},
			},
			args: args{
				ctx: context.TODO(),
				serverClient: func() k8sclient.Client {
					return moqclient.NewSigsClientMoqWithScheme(scheme,
						&routev1.Route{
							ObjectMeta: v1.ObjectMeta{
								Name:      "console",
								Namespace: "openshift-console",
							},
							Status: routev1.RouteStatus{
								Ingress: []routev1.RouteIngress{
									{
										Host: "host",
									},
								},
							},
						},
						&corev1.Secret{
							ObjectMeta: v1.ObjectMeta{
								Name:      "addon-managed-api-service-parameters",
								Namespace: "test",
							},
							Data: map[string][]byte{
								"custom-domain_domain": []byte("apps.example.com"),
							},
						})
				},
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "invalid domain",
			fields: fields{
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Name:      "managed-api",
						Namespace: "test",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: "managed-api",
					},
				},
			},
			args: args{
				ctx: context.TODO(),
				serverClient: func() k8sclient.Client {
					return moqclient.NewSigsClientMoqWithScheme(scheme,
						&routev1.Route{
							ObjectMeta: v1.ObjectMeta{
								Name:      "console",
								Namespace: "openshift-console",
							},
							Status: routev1.RouteStatus{
								Ingress: []routev1.RouteIngress{
									{
										Host: "host",
									},
								},
							},
						},
						&corev1.Secret{
							ObjectMeta: v1.ObjectMeta{
								Name:      "addon-managed-api-service-parameters",
								Namespace: "test",
							},
							Data: map[string][]byte{
								"custom-domain_domain": []byte("{ } < > | ` ^"),
							},
						})
				},
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "failed to retrieve console route cr",
			fields: fields{
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Name:      "managed-api",
						Namespace: "test",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: "managed-api",
					},
				},
			},
			args: args{
				ctx: context.TODO(),
				serverClient: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme)
					mockClient.GetFunc = func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
						return errors.New("generic error")
					}
					return mockClient
				},
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "cannot find console route cr",
			fields: fields{
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Name:      "managed-api",
						Namespace: "test",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: "managed-api",
					},
				},
			},
			args: args{
				ctx: context.TODO(),
				serverClient: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme)
					mockClient.GetFunc = func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
						return k8serr.NewNotFound(schema.GroupResource{}, "generic")
					}
					return mockClient
				},
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "fallback to RouterCanonicalHostname as routing subdomain when custom domain addon param not present",
			fields: fields{
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Name:      "managed-api",
						Namespace: "test",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: "managed-api",
					},
				},
			},
			args: args{
				ctx: context.TODO(),
				serverClient: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme,
						&routev1.Route{
							ObjectMeta: v1.ObjectMeta{
								Name:      "console",
								Namespace: "openshift-console",
							},
							Status: routev1.RouteStatus{
								Ingress: []routev1.RouteIngress{
									{
										Host: "host",
									},
								},
							},
						},
						&corev1.Secret{
							ObjectMeta: v1.ObjectMeta{
								Name:      "addon-managed-api-service-parameters",
								Namespace: "test",
							},
						},
					)
					return mockClient
				},
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "fallback to RouterCanonicalHostname as routing subdomain when empty custom domain addon param",
			fields: fields{
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Name:      "managed-api",
						Namespace: "test",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: "managed-api",
					},
				},
			},
			args: args{
				ctx: context.TODO(),
				serverClient: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme,
						&routev1.Route{
							ObjectMeta: v1.ObjectMeta{
								Name:      "console",
								Namespace: "openshift-console",
							},
							Status: routev1.RouteStatus{
								Ingress: []routev1.RouteIngress{
									{
										Host: "host",
									},
								},
							},
						},
						&corev1.Secret{
							ObjectMeta: v1.ObjectMeta{
								Name:      "addon-managed-api-service-parameters",
								Namespace: "test",
							},
							Data: map[string][]byte{
								"custom-domain_domain": []byte(""),
							},
						},
					)
					return mockClient
				},
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "routing subdomain already set, exit early",
			fields: fields{
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Name:      "managed-api",
						Namespace: "test",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type:             "managed-api",
						RoutingSubdomain: "apps.example.com",
					},
				},
			},
			args: args{
				ctx: context.TODO(),
				serverClient: func() k8sclient.Client {
					return moqclient.NewSigsClientMoqWithScheme(scheme,
						&routev1.Route{
							ObjectMeta: v1.ObjectMeta{
								Name:      "console",
								Namespace: "openshift-console",
							},
							Status: routev1.RouteStatus{
								Ingress: []routev1.RouteIngress{
									{
										Host: "host",
									},
								},
							},
						},
						&corev1.Secret{
							ObjectMeta: v1.ObjectMeta{
								Name:      "addon-managed-api-service-parameters",
								Namespace: "test",
							},
							Data: map[string][]byte{
								"custom-domain_domain": []byte("apps.example.com"),
							},
						})
				},
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
				Reconciler:    tt.fields.Reconciler,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			got, err := r.retrieveConsoleURLAndSubdomain(tt.args.ctx, tt.args.serverClient())
			if (err != nil) != tt.wantErr {
				t.Errorf("retrieveConsoleURLAndSubdomain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("retrieveConsoleURLAndSubdomain() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_retrieveAPIServerURL(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		Reconciler    *resources.Reconciler
		recorder      record.EventRecorder
		log           l.Logger
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
			name:    "No Infrastructure CR found",
			args:    args{ctx: context.TODO(), serverClient: utils.NewTestClient(scheme)},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "No URL found starting with APi",
			args: args{ctx: context.TODO(), serverClient: utils.NewTestClient(scheme, &configv1.Infrastructure{
				ObjectMeta: v1.ObjectMeta{
					Name: "cluster",
				},
				Status: configv1.InfrastructureStatus{APIServerURL: ""},
			})},
			fields:  fields{installation: &integreatlyv1alpha1.RHMI{Spec: integreatlyv1alpha1.RHMISpec{}}},
			wantErr: true,
			want:    integreatlyv1alpha1.PhaseFailed,
		},
		{
			name: "Found API URL from list of names",
			args: args{ctx: context.TODO(), serverClient: utils.NewTestClient(scheme, &configv1.Infrastructure{
				ObjectMeta: v1.ObjectMeta{
					Name: "cluster",
				},
				Status: configv1.InfrastructureStatus{APIServerURL: "https://api.example.com"},
			})},
			fields:  fields{installation: &integreatlyv1alpha1.RHMI{Spec: integreatlyv1alpha1.RHMISpec{}}},
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
				Reconciler:    tt.fields.Reconciler,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			got, err := r.retrieveAPIServerURL(tt.args.ctx, tt.args.serverClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("retrieveAPIServerURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("retrieveAPIServerURL() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_generateSecret(t *testing.T) {
	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		Reconciler    *resources.Reconciler
		recorder      record.EventRecorder
		log           l.Logger
	}
	type args struct {
		length int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "Test you get a string of correct length",
			args: args{
				length: 32,
			},
			want: "V8bHJmLKToT1La4GHkKTVt1NlCECK7W8",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				Reconciler:    tt.fields.Reconciler,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			if got := r.generateSecret(tt.args.length); len(got) != len(tt.want) {
				t.Errorf("generateSecret() = %v, want %v", len(got), len(tt.want))
			}
		})
	}
}

func TestReconciler_checkCloudResourcesConfig(t *testing.T) {
	type fields struct {
		ConfigManager config.ConfigReadWriter
		Config        *config.ThreeScale
		mpm           marketplace.MarketplaceInterface
		installation  *integreatlyv1alpha1.RHMI
		Reconciler    *resources.Reconciler
		recorder      record.EventRecorder
		log           l.Logger
	}
	type args struct {
		serverClient k8sclient.Client
	}
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "successfully check aws cloud resources config",
			fields: fields{
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Name:      "rhoam",
						Namespace: rhoamOperatorNs,
					},
				},
			},
			args: args{
				serverClient: utils.NewTestClient(scheme,
					&corev1.ConfigMap{
						ObjectMeta: v1.ObjectMeta{
							Name:       DefaultCloudResourceConfigName,
							Namespace:  rhoamOperatorNs,
							Finalizers: []string{previousDeletionFinalizer},
						},
					},
					&configv1.Infrastructure{
						ObjectMeta: v1.ObjectMeta{
							Name: "cluster",
						},
						Status: configv1.InfrastructureStatus{
							PlatformStatus: &configv1.PlatformStatus{
								Type: configv1.AWSPlatformType,
							},
						},
					},
				),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "successfully check cloud resources config with enabled useClusterStorage",
			fields: fields{
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Name:      "rhoam",
						Namespace: rhoamOperatorNs,
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						UseClusterStorage: "true",
					},
				},
			},
			args: args{
				serverClient: utils.NewTestClient(scheme,
					&corev1.ConfigMap{
						ObjectMeta: v1.ObjectMeta{
							Name:       DefaultCloudResourceConfigName,
							Namespace:  rhoamOperatorNs,
							Finalizers: []string{previousDeletionFinalizer},
						},
					},
					&configv1.Infrastructure{
						ObjectMeta: v1.ObjectMeta{
							Name: "cluster",
						},
						Status: configv1.InfrastructureStatus{
							PlatformStatus: &configv1.PlatformStatus{
								Type: configv1.AWSPlatformType,
							},
						},
					},
				),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "fail to check cloud resources config",
			fields: fields{
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Name:      "rhoam",
						Namespace: rhoamOperatorNs,
					},
				},
			},
			args: args{
				serverClient: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme)
					mockClient.GetFunc = func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
						return fmt.Errorf("generic error")
					}
					return mockClient
				}(),
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: true,
		},
		{
			name: "fail to determine platform type",
			fields: fields{
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Name:      "rhoam",
						Namespace: rhoamOperatorNs,
					},
				},
			},
			args: args{
				serverClient: moqclient.NewSigsClientMoqWithScheme(scheme,
					&corev1.ConfigMap{
						ObjectMeta: v1.ObjectMeta{
							Name:       DefaultCloudResourceConfigName,
							Namespace:  rhoamOperatorNs,
							Finalizers: []string{previousDeletionFinalizer},
						},
					}),
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				mpm:           tt.fields.mpm,
				installation:  tt.fields.installation,
				Reconciler:    tt.fields.Reconciler,
				recorder:      tt.fields.recorder,
				log:           tt.fields.log,
			}
			got, err := r.checkCloudResourcesConfig(context.TODO(), tt.args.serverClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkCloudResourcesConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkCloudResourcesConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}
