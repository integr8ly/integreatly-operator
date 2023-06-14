package observability

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/addon"
	clientMock "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"github.com/integr8ly/integreatly-operator/utils"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	observability "github.com/redhat-developer/observability-operator/v4/api/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	localProductDeclaration = marketplace.LocalProductDeclaration("observability-operator")
	genericError            = errors.New("generic error")
)

func TestNewReconciler(t *testing.T) {
	type args struct {
		configManager      config.ConfigReadWriter
		installation       *v1alpha1.RHMI
		mpm                marketplace.MarketplaceInterface
		recorder           record.EventRecorder
		logger             logger.Logger
		productDeclaration *marketplace.ProductDeclaration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "could not retrieve observability config",
			args: args{
				configManager: &config.ConfigReadWriterMock{
					ReadObservabilityFunc: func() (*config.Observability, error) {
						return nil, genericError
					},
				},
				installation: &v1alpha1.RHMI{
					Spec: v1alpha1.RHMISpec{},
				},
				logger: l.Logger{},
			},
			wantErr: true,
		},
		{
			name: "error writing config",
			args: args{
				configManager: &config.ConfigReadWriterMock{
					ReadObservabilityFunc: func() (*config.Observability, error) {
						return &config.Observability{
							Config: config.ProductConfig{},
						}, nil
					},
					WriteConfigFunc: func(config config.ConfigReadable) error {
						return genericError
					},
				},
				installation: &v1alpha1.RHMI{
					Spec: v1alpha1.RHMISpec{},
				},
				logger: l.Logger{},
			},
			wantErr: true,
		},
		{
			name: "success",
			args: args{
				configManager: &config.ConfigReadWriterMock{
					ReadObservabilityFunc: func() (*config.Observability, error) {
						return &config.Observability{
							Config: config.ProductConfig{},
						}, nil
					},
					WriteConfigFunc: func(config config.ConfigReadable) error {
						return nil
					},
				},
				installation: &v1alpha1.RHMI{
					Spec: v1alpha1.RHMISpec{
						OperatorsInProductNamespace: true,
					},
				},
				logger:             l.Logger{},
				productDeclaration: localProductDeclaration,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewReconciler(tt.args.configManager, tt.args.installation, tt.args.mpm, tt.args.recorder, tt.args.logger, tt.args.productDeclaration)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewReconciler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Fatalf("NewReconciler() got = %v, want non nil", got)
			}
		})
	}
}

func TestReconciler_GetPreflightObject(t *testing.T) {
	type fields struct {
		Reconciler    *resources.Reconciler
		ConfigManager config.ConfigReadWriter
		Config        *config.Observability
	}
	type args struct {
		ns string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   runtime.Object
	}{
		{
			name:   "retrieve preflight object",
			fields: fields{},
			args:   args{},
			want:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{}
			if got := r.GetPreflightObject(tt.args.ns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetPreflightObject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_Reconcile(t *testing.T) {
	type fields struct {
		Reconciler    *resources.Reconciler
		ConfigManager *config.ConfigReadWriterMock
		Config        *config.Observability
		installation  *v1alpha1.RHMI
		mpm           *marketplace.MarketplaceInterfaceMock
		log           logger.Logger
		extraParams   map[string]string
		recorder      record.EventRecorder
	}
	type args struct {
		ctx          context.Context
		installation *v1alpha1.RHMI
		product      *v1alpha1.RHMIProductStatus
		client       client.Client
		in4          quota.ProductConfig
		uninstall    bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    v1alpha1.StatusPhase
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				Reconciler:    tt.fields.Reconciler,
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				installation:  tt.fields.installation,
				mpm:           tt.fields.mpm,
				log:           tt.fields.log,
				extraParams:   tt.fields.extraParams,
				recorder:      tt.fields.recorder,
			}
			got, err := r.Reconcile(tt.args.ctx, tt.args.installation, tt.args.product, tt.args.client, tt.args.in4, tt.args.uninstall)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Reconcile() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_VerifyVersion(t *testing.T) {
	type fields struct {
		Reconciler    *resources.Reconciler
		ConfigManager config.ConfigReadWriter
		Config        *config.Observability
		installation  *v1alpha1.RHMI
		mpm           marketplace.MarketplaceInterface
		log           logger.Logger
		extraParams   map[string]string
		recorder      record.EventRecorder
	}
	type args struct {
		installation *v1alpha1.RHMI
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "test TestReconciler_VerifyVersion - negative",
			args: args{
				installation: basicInstallation(),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				Reconciler:    tt.fields.Reconciler,
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				installation:  tt.fields.installation,
				mpm:           tt.fields.mpm,
				log:           tt.fields.log,
				extraParams:   tt.fields.extraParams,
				recorder:      tt.fields.recorder,
			}
			if got := r.VerifyVersion(tt.args.installation); got != tt.want {
				t.Errorf("VerifyVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_fullReconcile(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultInstallationNamespace,
			Labels: map[string]string{
				resources.OwnerLabelKey: string(basicInstallation().GetUID()),
			},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}
	operatorNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultInstallationNamespace + "-operator",
			Labels: map[string]string{
				resources.OwnerLabelKey: string(basicInstallation().GetUID()),
			},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}
	installation := basicInstallation()
	cases := []struct {
		Name           string
		ExpectError    bool
		ExpectedStatus v1alpha1.StatusPhase
		ExpectedError  string
		FakeConfig     *config.ConfigReadWriterMock
		FakeClient     k8sclient.Client
		FakeMPM        *marketplace.MarketplaceInterfaceMock
		Installation   *v1alpha1.RHMI
		Product        *v1alpha1.RHMIProductStatus
		Recorder       record.EventRecorder
		Uninstall      bool
	}{
		{
			Name:           "test successful reconcile",
			ExpectedStatus: v1alpha1.PhaseCompleted,
			FakeClient:     clientMock.NewSigsClientMoqWithScheme(scheme, ns, operatorNS, installation),
			FakeConfig: &config.ConfigReadWriterMock{
				WriteConfigFunc: func(config config.ConfigReadable) error {
					return nil
				},
				ReadObservabilityFunc: func() (ready *config.Observability, e error) {
					return config.NewObservability(config.ProductConfig{}), nil
				},
			},
			FakeMPM: &marketplace.MarketplaceInterfaceMock{
				InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {
					return nil
				},
				GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plan *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
					return &operatorsv1alpha1.InstallPlan{
							ObjectMeta: metav1.ObjectMeta{
								Name: "install-plan",
							},
							Status: operatorsv1alpha1.InstallPlanStatus{
								Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
							},
						}, &operatorsv1alpha1.Subscription{
							Status: operatorsv1alpha1.SubscriptionStatus{
								Install: &operatorsv1alpha1.InstallPlanReference{
									Name: "install-plan",
								},
							},
						}, nil
				},
			},
			Installation: basicInstallation(),
			Product:      &v1alpha1.RHMIProductStatus{},
			Recorder:     setupRecorder(),
			Uninstall:    false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM, tc.Recorder, getLogger(), localProductDeclaration)
			if err != nil && err.Error() != tc.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
			}
			status, err := reconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.FakeClient, &quota.ProductConfigMock{}, tc.Uninstall)
			if err != nil && !tc.ExpectError {
				t.Fatalf("expected no error but got one: %v", err)
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

func basicInstallation() *v1alpha1.RHMI {
	return &v1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "installation",
			Namespace:       defaultInstallationNamespace,
			ResourceVersion: "1",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "RHMI",
			APIVersion: v1alpha1.GroupVersion.String(),
		},
		Spec: v1alpha1.RHMISpec{
			Type: string(v1alpha1.InstallationTypeManagedApi),
		},
	}
}

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: "Dummy Product"})
}

func TestReconciler_deleteObservabilityCR(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}
	type fields struct {
		Reconciler    *resources.Reconciler
		ConfigManager config.ConfigReadWriter
		Config        *config.Observability
		installation  *v1alpha1.RHMI
		mpm           marketplace.MarketplaceInterface
		log           l.Logger
		extraParams   map[string]string
		recorder      record.EventRecorder
	}
	type args struct {
		ctx             context.Context
		serverClient    k8sclient.Client
		inst            *v1alpha1.RHMI
		targetNamespace string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    v1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "deletion timestamp is nil, do not proceed",
			args: args{
				inst: &v1alpha1.RHMI{},
			},
			want:    v1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name:   "failed to determine if operator is hive managed",
			fields: fields{},
			args: args{
				ctx: context.TODO(),
				serverClient: func() k8sclient.Client {
					fakeClient := clientMock.NewSigsClientMoqWithScheme(scheme)
					fakeClient.GetFunc = func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
						return fmt.Errorf("generic error")
					}
					return fakeClient
				}(),
				inst: &v1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						DeletionTimestamp: &metav1.Time{
							Time: time.Now(),
						},
					},
				},
			},
			want:    v1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name:   "unexpected error retrieving dms secret",
			fields: fields{},
			args: args{
				ctx: context.TODO(),
				serverClient: func() k8sclient.Client {
					fakeClient := clientMock.NewSigsClientMoqWithScheme(scheme)
					fakeClient.GetFunc = func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
						switch obj := obj.(type) {
						case *corev1.Namespace:
							obj.Labels = map[string]string{
								addon.RhoamAddonInstallManagedLabel: "true",
							}
							return nil
						case *corev1.Secret:
							return fmt.Errorf("generic error")
						default:
							return nil
						}
					}
					return fakeClient
				}(),
				inst: &v1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-ns",
						DeletionTimestamp: &metav1.Time{
							Time: time.Now(),
						},
					},
				},
			},
			want:    v1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name:   "dms secret is still present, requeue",
			fields: fields{},
			args: args{
				ctx: context.TODO(),
				serverClient: func() k8sclient.Client {
					fakeClient := clientMock.NewSigsClientMoqWithScheme(scheme)
					fakeClient.GetFunc = func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
						switch obj := obj.(type) {
						case *corev1.Namespace:
							obj.Labels = map[string]string{addon.RhoamAddonInstallManagedLabel: "true"}
							return nil
						case *corev1.Secret:
							obj.Data = map[string][]byte{"SNITCH_URL": []byte("www.example.com")}
							return nil
						default:
							return nil
						}
					}
					return fakeClient
				}(),
				inst: &v1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-ns",
						DeletionTimestamp: &metav1.Time{
							Time: time.Now(),
						},
					},
					Spec: v1alpha1.RHMISpec{
						DeadMansSnitchSecret: "dms-secret",
					},
				},
			},
			want:    v1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name:   "unexpected error retrieving observability cr",
			fields: fields{},
			args: args{
				ctx: context.TODO(),
				serverClient: func() k8sclient.Client {
					fakeClient := clientMock.NewSigsClientMoqWithScheme(scheme)
					fakeClient.GetFunc = func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
						switch obj.(type) {
						case *corev1.Namespace:
							return nil
						case *corev1.Secret:
							return nil
						case *observability.Observability:
							return fmt.Errorf("generic error")
						default:
							return nil
						}
					}
					return fakeClient
				}(),
				inst: &v1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-ns",
						DeletionTimestamp: &metav1.Time{
							Time: time.Now(),
						},
					},
				},
			},
			want:    v1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name:   "successfully deleted observability cr",
			fields: fields{},
			args: args{
				ctx: context.TODO(),
				serverClient: func() k8sclient.Client {
					fakeClient := clientMock.NewSigsClientMoqWithScheme(scheme)
					fakeClient.GetFunc = func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
						switch obj.(type) {
						case *corev1.Namespace:
							return nil
						case *corev1.Secret:
							return nil
						case *observability.Observability:
							return k8serr.NewNotFound(schema.GroupResource{}, "generic")
						default:
							return nil
						}
					}
					return fakeClient
				}(),
				inst: &v1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-ns",
						DeletionTimestamp: &metav1.Time{
							Time: time.Now(),
						},
					},
				},
			},
			want:    v1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name:   "failed to mark the observability cr for deletion",
			fields: fields{},
			args: args{
				ctx: context.TODO(),
				serverClient: func() k8sclient.Client {
					fakeClient := clientMock.NewSigsClientMoqWithScheme(scheme)
					fakeClient.GetFunc = func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
						switch obj.(type) {
						case *corev1.Namespace:
							return nil
						case *corev1.Secret:
							return nil
						case *observability.Observability:
							return nil
						default:
							return nil
						}
					}
					fakeClient.DeleteFunc = func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.DeleteOption) error {
						return fmt.Errorf("generic error")
					}
					return fakeClient
				}(),
				inst: &v1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-ns",
						DeletionTimestamp: &metav1.Time{
							Time: time.Now(),
						},
					},
				},
			},
			want:    v1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name:   "successfully marked the observability cr for deletion",
			fields: fields{},
			args: args{
				ctx: context.TODO(),
				serverClient: func() k8sclient.Client {
					fakeClient := clientMock.NewSigsClientMoqWithScheme(scheme)
					fakeClient.GetFunc = func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
						switch obj.(type) {
						case *corev1.Namespace:
							return nil
						case *corev1.Secret:
							return nil
						case *observability.Observability:
							return nil
						default:
							return nil
						}
					}
					fakeClient.DeleteFunc = func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.DeleteOption) error {
						return nil
					}
					return fakeClient
				}(),
				inst: &v1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test-ns",
						DeletionTimestamp: &metav1.Time{
							Time: time.Now(),
						},
					},
				},
			},
			want:    v1alpha1.PhaseInProgress,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				Reconciler:    tt.fields.Reconciler,
				ConfigManager: tt.fields.ConfigManager,
				Config:        tt.fields.Config,
				installation:  tt.fields.installation,
				mpm:           tt.fields.mpm,
				log:           tt.fields.log,
				extraParams:   tt.fields.extraParams,
				recorder:      tt.fields.recorder,
			}
			got, err := r.deleteObservabilityCR(tt.args.ctx, tt.args.serverClient, tt.args.inst, tt.args.targetNamespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("deleteObservabilityCR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("deleteObservabilityCR() got = %v, want %v", got, tt.want)
			}
		})
	}
}
