package mcg

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"github.com/integr8ly/integreatly-operator/utils"
	obv1 "github.com/kube-object-storage/lib-bucket-provisioner/pkg/apis/objectbucket.io/v1alpha1"
	noobaav1 "github.com/noobaa/noobaa-operator/v5/pkg/apis/noobaa/v1alpha1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	localProductDeclaration  = marketplace.LocalProductDeclaration("integreatly-mcg")
	defaultOperatorNamespace = DefaultInstallationNamespace + "-operator"
)

func getLogger() logger.Logger {
	return logger.NewLoggerWithContext(logger.Fields{logger.ProductLogContext: integreatlyv1alpha1.ProductMCG})
}

func basicConfigMock() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		GetOperatorNamespaceFunc: func() string {
			return DefaultInstallationNamespace
		},
		ReadMCGFunc: func() (*config.MCG, error) {
			return config.NewMCG(config.ProductConfig{
				"OPERATOR_NAMEPSACE": defaultOperatorNamespace,
			}), nil
		},
		WriteConfigFunc: func(config config.ConfigReadable) error {
			return nil
		},
		ReadObservabilityFunc: func() (*config.Observability, error) {
			return config.NewObservability(config.ProductConfig{
				"NAMESPACE": "namespace",
			}), nil
		},
	}
}

func basicInstallation(delete bool) *integreatlyv1alpha1.RHMI {
	rhmi := &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "rhoam",
			Generation:      1,
			ResourceVersion: "1",
		},
		Spec: integreatlyv1alpha1.RHMISpec{},
	}
	if delete {
		rhmi.ObjectMeta.DeletionTimestamp = &metav1.Time{}
	}
	return rhmi
}

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
}

func TestReconciler_ReconcileMCG(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	objects := getSuccessfulTestPreReqs()

	type fields struct {
		Config     *config.MCG
		FakeConfig *config.ConfigReadWriterMock
		mpm        marketplace.MarketplaceInterface
		Reconciler *resources.Reconciler
		recorder   record.EventRecorder
	}
	type args struct {
		installation  *integreatlyv1alpha1.RHMI
		productStatus *integreatlyv1alpha1.RHMIProductStatus
		client        k8sclient.Client
		productConfig quota.ProductConfig
		uninstall     bool
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
				FakeConfig: basicConfigMock(),
				recorder:   setupRecorder(),
				mpm: &marketplace.MarketplaceInterfaceMock{
					InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {
						return nil
					},
					GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
						return &operatorsv1alpha1.InstallPlan{
								ObjectMeta: metav1.ObjectMeta{
									Name: "mcg-install-plan",
								},
								Status: operatorsv1alpha1.InstallPlanStatus{
									Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
								},
							}, &operatorsv1alpha1.Subscription{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "rhmi-mcg",
									Namespace: "mcg",
								},
								Status: operatorsv1alpha1.SubscriptionStatus{
									Install: &operatorsv1alpha1.InstallPlanReference{
										Name: "mcg-install-plan",
									},
								},
							}, nil
					},
				},
			},
			args: args{
				installation:  basicInstallation(false),
				productStatus: &integreatlyv1alpha1.RHMIProductStatus{},
				productConfig: &quota.ProductConfigMock{
					ConfigureFunc: func(obj metav1.Object) error {
						return nil
					},
				},
				client:    moqclient.NewSigsClientMoqWithScheme(scheme, objects...),
				uninstall: false,
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "test errors reconciling alerts",
			fields: fields{
				FakeConfig: basicConfigMock(),
				recorder:   setupRecorder(),
				mpm: &marketplace.MarketplaceInterfaceMock{
					InstallOperatorFunc: func(ctx context.Context, serverClient k8sclient.Client, t marketplace.Target, operatorGroupNamespaces []string, approvalStrategy operatorsv1alpha1.Approval, catalogSourceReconciler marketplace.CatalogSourceReconciler) error {
						return nil
					},
					GetSubscriptionInstallPlanFunc: func(ctx context.Context, serverClient k8sclient.Client, subName string, ns string) (plans *operatorsv1alpha1.InstallPlan, subscription *operatorsv1alpha1.Subscription, e error) {
						return &operatorsv1alpha1.InstallPlan{
								ObjectMeta: metav1.ObjectMeta{
									Name: "mcg-install-plan",
								},
								Status: operatorsv1alpha1.InstallPlanStatus{
									Phase: operatorsv1alpha1.InstallPlanPhaseComplete,
								},
							}, &operatorsv1alpha1.Subscription{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "rhmi-mcg",
									Namespace: "mcg",
								},
								Status: operatorsv1alpha1.SubscriptionStatus{
									Install: &operatorsv1alpha1.InstallPlanReference{
										Name: "mcg-install-plan",
									},
								},
							}, nil
					},
				},
			},
			args: args{
				installation:  basicInstallation(false),
				productStatus: &integreatlyv1alpha1.RHMIProductStatus{},
				productConfig: &quota.ProductConfigMock{
					ConfigureFunc: func(obj metav1.Object) error {
						return nil
					},
				},
				client: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, objects...)
					mockClient.CreateFunc = func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.CreateOption) error {
						switch obj.(type) {
						case *monv1.PrometheusRule:
							return errors.New("test error")
						default:
							return nil
						}
					}
					return mockClient
				}(),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewReconciler(
				tt.fields.FakeConfig,
				tt.args.installation,
				tt.fields.mpm,
				tt.fields.recorder,
				getLogger(),
				localProductDeclaration,
			)
			if err != nil {
				t.Fatalf("NewReconciler() error = '%v'", err)
			}
			got, err := r.Reconcile(context.TODO(), tt.args.installation, tt.args.productStatus, tt.args.client, tt.args.productConfig, tt.args.uninstall)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("Reconcile() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_ReconcileNoobaa(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				defaultStorageClassAnnotation: "true",
			},
		},
	}

	noobaa := &noobaav1.NooBaa{
		ObjectMeta: metav1.ObjectMeta{
			Name:      noobaaName,
			Namespace: defaultOperatorNamespace,
		},
		Status: noobaav1.NooBaaStatus{
			Phase: noobaav1.SystemPhaseReady,
		},
	}

	type fields struct {
		Config       *config.MCG
		FakeConfig   *config.ConfigReadWriterMock
		mpm          marketplace.MarketplaceInterface
		Reconciler   *resources.Reconciler
		recorder     record.EventRecorder
		installation *integreatlyv1alpha1.RHMI
	}
	type args struct {
		client k8sclient.Client
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "test successfully reconcile noobaa",
			fields: fields{
				FakeConfig:   basicConfigMock(),
				installation: basicInstallation(false),
			},
			args: args{
				client: moqclient.NewSigsClientMoqWithScheme(scheme, storageClass, noobaa),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "test reconcile noobaa in progress",
			fields: fields{
				FakeConfig:   basicConfigMock(),
				installation: basicInstallation(false),
			},
			args: args{
				client: moqclient.NewSigsClientMoqWithScheme(scheme, storageClass),
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: false,
		},
		{
			name: "test error retrieving noobaa",
			fields: fields{
				FakeConfig:   basicConfigMock(),
				installation: basicInstallation(false),
			},
			args: args{
				client: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, storageClass, noobaa)
					mockClient.GetFunc = func(ctx context.Context, key k8sTypes.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
						return errors.New("test error")
					}
					return mockClient
				}(),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "test error creating noobaa",
			fields: fields{
				FakeConfig:   basicConfigMock(),
				installation: basicInstallation(false),
			},
			args: args{
				client: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, storageClass)
					mockClient.CreateFunc = func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.CreateOption) error {
						return errors.New("test error")
					}
					return mockClient
				}(),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "test error updating noobaa",
			fields: fields{
				FakeConfig:   basicConfigMock(),
				installation: basicInstallation(false),
			},
			args: args{
				client: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, storageClass, noobaa)
					mockClient.UpdateFunc = func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.UpdateOption) error {
						return errors.New("test error")
					}
					return mockClient
				}(),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewReconciler(
				tt.fields.FakeConfig,
				tt.fields.installation,
				tt.fields.mpm,
				tt.fields.recorder,
				getLogger(),
				localProductDeclaration,
			)
			if err != nil {
				t.Fatalf("NewReconciler() error = '%v'", err)
			}
			got, err := r.ReconcileNoobaa(context.TODO(), tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ReconcileNoobaa() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("ReconcileNoobaa() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_ReconcileObjectBucketClaim(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	obc := &noobaav1.ObjectBucketClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ThreescaleBucketClaim,
			Namespace: defaultOperatorNamespace,
		},
		Status: obv1.ObjectBucketClaimStatus{
			Phase: obv1.ObjectBucketClaimStatusPhaseBound,
		},
	}

	type fields struct {
		Config       *config.MCG
		FakeConfig   *config.ConfigReadWriterMock
		mpm          marketplace.MarketplaceInterface
		Reconciler   *resources.Reconciler
		recorder     record.EventRecorder
		installation *integreatlyv1alpha1.RHMI
	}
	type args struct {
		client k8sclient.Client
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "test successfully reconcile object bucket claim",
			fields: fields{
				FakeConfig:   basicConfigMock(),
				installation: basicInstallation(false),
			},
			args: args{
				client: moqclient.NewSigsClientMoqWithScheme(scheme, obc),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "test reconcile object bucket claim in progress",
			fields: fields{
				FakeConfig:   basicConfigMock(),
				installation: basicInstallation(false),
			},
			args: args{
				client: moqclient.NewSigsClientMoqWithScheme(scheme),
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: false,
		},
		{
			name: "test error retrieving object bucket claim",
			fields: fields{
				FakeConfig:   basicConfigMock(),
				installation: basicInstallation(false),
			},
			args: args{
				client: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, obc)
					mockClient.GetFunc = func(ctx context.Context, key k8sTypes.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
						return errors.New("test error")
					}
					return mockClient
				}(),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "test error creating object bucket claim",
			fields: fields{
				FakeConfig:   basicConfigMock(),
				installation: basicInstallation(false),
			},
			args: args{
				client: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme)
					mockClient.CreateFunc = func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.CreateOption) error {
						return errors.New("test error")
					}
					return mockClient
				}(),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "test error updating object bucket claim",
			fields: fields{
				FakeConfig:   basicConfigMock(),
				installation: basicInstallation(false),
			},
			args: args{
				client: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, obc)
					mockClient.UpdateFunc = func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.UpdateOption) error {
						return errors.New("test error")
					}
					return mockClient
				}(),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewReconciler(
				tt.fields.FakeConfig,
				tt.fields.installation,
				tt.fields.mpm,
				tt.fields.recorder,
				getLogger(),
				localProductDeclaration,
			)
			if err != nil {
				t.Fatalf("NewReconciler() error = '%v'", err)
			}
			got, err := r.ReconcileObjectBucketClaim(context.TODO(), tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ReconcileObjectBucketClaim() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("ReconcileObjectBucketClaim() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_cleanupResources(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	noobaaCRD := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "noobaas.noobaa.io",
		},
	}

	noobaa := &noobaav1.NooBaa{
		ObjectMeta: metav1.ObjectMeta{
			Name:      noobaaName,
			Namespace: defaultOperatorNamespace,
		},
	}

	obc := &noobaav1.ObjectBucketClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ThreescaleBucketClaim,
			Namespace: defaultOperatorNamespace,
		},
	}

	ob := &noobaav1.ObjectBucket{
		ObjectMeta: metav1.ObjectMeta{
			Name:      threescaleBucket,
			Namespace: defaultOperatorNamespace,
		},
	}

	type fields struct {
		Config     *config.MCG
		FakeConfig *config.ConfigReadWriterMock
		mpm        marketplace.MarketplaceInterface
		Reconciler *resources.Reconciler
		recorder   record.EventRecorder
	}
	type args struct {
		client       k8sclient.Client
		installation *integreatlyv1alpha1.RHMI
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "test missing deletion timestamp returns phase completed",
			fields: fields{
				FakeConfig: basicConfigMock(),
			},
			args: args{
				client:       moqclient.NewSigsClientMoqWithScheme(scheme),
				installation: &integreatlyv1alpha1.RHMI{},
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "test missing NooBaa CRD returns phase completed",
			fields: fields{
				FakeConfig: basicConfigMock(),
			},
			args: args{
				client:       moqclient.NewSigsClientMoqWithScheme(scheme),
				installation: basicInstallation(true),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "test error retrieving noobaa CRD",
			fields: fields{
				FakeConfig: basicConfigMock(),
			},
			args: args{
				client: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme)
					mockClient.GetFunc = func(ctx context.Context, key k8sTypes.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
						return fmt.Errorf("test error")
					}
					return mockClient
				}(),
				installation: basicInstallation(true),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "test error listing object bucket claims",
			fields: fields{
				FakeConfig: basicConfigMock(),
			},
			args: args{
				client: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, noobaaCRD)
					mockClient.ListFunc = func(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
						return fmt.Errorf("test error")
					}
					return mockClient
				}(),
				installation: basicInstallation(true),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "test error deleting object bucket claims",
			fields: fields{
				FakeConfig: basicConfigMock(),
			},
			args: args{
				client: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, noobaaCRD, obc)
					mockClient.DeleteFunc = func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.DeleteOption) error {
						return fmt.Errorf("test error")
					}
					return mockClient
				}(),
				installation: basicInstallation(true),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "test error listing object buckets",
			fields: fields{
				FakeConfig: basicConfigMock(),
			},
			args: args{
				client: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, noobaaCRD)
					mockClient.ListFunc = func(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
						if _, ok := list.(*noobaav1.ObjectBucketList); ok {
							return fmt.Errorf("test error")
						}
						return nil
					}
					return mockClient
				}(),
				installation: basicInstallation(true),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "test object bucket present returns in progress",
			fields: fields{
				FakeConfig: basicConfigMock(),
			},
			args: args{
				client:       moqclient.NewSigsClientMoqWithScheme(scheme, noobaaCRD, obc, ob, noobaa),
				installation: basicInstallation(true),
			},
			want:    integreatlyv1alpha1.PhaseInProgress,
			wantErr: false,
		},
		{
			name: "test error listing noobaas",
			fields: fields{
				FakeConfig: basicConfigMock(),
			},
			args: args{
				client: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, noobaaCRD)
					mockClient.ListFunc = func(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
						if _, ok := list.(*noobaav1.NooBaaList); ok {
							return fmt.Errorf("test error")
						}
						return nil
					}
					return mockClient
				}(),
				installation: basicInstallation(true),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "test error deleting noobaa",
			fields: fields{
				FakeConfig: basicConfigMock(),
			},
			args: args{
				client: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme, noobaaCRD, noobaa)
					mockClient.DeleteFunc = func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.DeleteOption) error {
						if _, ok := obj.(*noobaav1.NooBaa); ok {
							return fmt.Errorf("test error")
						}
						return nil
					}
					return mockClient
				}(),
				installation: basicInstallation(true),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "test successfully cleanup resources",
			fields: fields{
				FakeConfig: basicConfigMock(),
			},
			args: args{
				client:       moqclient.NewSigsClientMoqWithScheme(scheme, noobaaCRD, obc, noobaa),
				installation: basicInstallation(true),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "test all resources have been removed",
			fields: fields{
				FakeConfig: basicConfigMock(),
			},
			args: args{
				client:       moqclient.NewSigsClientMoqWithScheme(scheme, noobaaCRD),
				installation: basicInstallation(true),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewReconciler(
				tt.fields.FakeConfig,
				tt.args.installation,
				tt.fields.mpm,
				tt.fields.recorder,
				getLogger(),
				localProductDeclaration,
			)
			if err != nil {
				t.Fatalf("NewReconciler() error = '%v'", err)
			}
			got, err := r.cleanupResources(context.TODO(), tt.args.client, tt.args.installation)
			if (err != nil) != tt.wantErr {
				t.Fatalf("cleanupResources() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("cleanupResources() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_retrieveDefaultStorageClass(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				defaultStorageClassAnnotation: "true",
			},
		},
	}

	type args struct {
		client k8sclient.Client
	}
	tests := []struct {
		name    string
		args    args
		want    *storagev1.StorageClass
		wantErr bool
	}{
		{
			name: "test no annotated storage class returns error",
			args: args{
				client: moqclient.NewSigsClientMoqWithScheme(scheme),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "test error retrieving storage classes",
			args: args{
				client: func() k8sclient.Client {
					mockClient := moqclient.NewSigsClientMoqWithScheme(scheme)
					mockClient.ListFunc = func(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
						return errors.New("test error")
					}
					return mockClient
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "test successfully retrieve default storage class",
			args: args{
				client: moqclient.NewSigsClientMoqWithScheme(scheme, storageClass),
			},
			want:    storageClass,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := retrieveDefaultStorageClass(context.TODO(), tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Fatalf("retrieveDefaultStorageClass() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("retrieveDefaultStorageClass() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func getSuccessfulTestPreReqs() []runtime.Object {
	return append([]runtime.Object{
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "s3",
				Namespace: defaultOperatorNamespace,
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: DefaultInstallationNamespace,
			},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceActive,
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: defaultOperatorNamespace,
			},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceActive,
			},
		},
		&storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default-storage",
				Annotations: map[string]string{
					defaultStorageClassAnnotation: "true",
				},
			},
		},
		&noobaav1.NooBaa{
			ObjectMeta: metav1.ObjectMeta{
				Name:      noobaaName,
				Namespace: defaultOperatorNamespace,
			},
			Status: noobaav1.NooBaaStatus{
				Phase: noobaav1.SystemPhaseReady,
			},
		},
		&noobaav1.ObjectBucketClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ThreescaleBucketClaim,
				Namespace: defaultOperatorNamespace,
			},
			Status: obv1.ObjectBucketClaimStatus{
				Phase: obv1.ObjectBucketClaimStatusPhaseBound,
			},
		},
	}, basicInstallation(false))
}
