package observability

import (
	"context"
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	projectv1 "github.com/openshift/api/project/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	observability "github.com/redhat-developer/observability-operator/v3/api/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var localProductDeclaration = marketplace.LocalProductDeclaration("observability-operator")

func TestNewReconciler(t *testing.T) {
	type args struct {
		configManager      config.ConfigReadWriter
		installation       *integreatlyv1alpha1.RHMI
		mpm                marketplace.MarketplaceInterface
		recorder           record.EventRecorder
		logger             logger.Logger
		productDeclaration *marketplace.ProductDeclaration
	}
	tests := []struct {
		name    string
		args    args
		want    *Reconciler
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewReconciler(tt.args.configManager, tt.args.installation, tt.args.mpm, tt.args.recorder, tt.args.logger, tt.args.productDeclaration)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewReconciler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewReconciler() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_GetPreflightObject(t *testing.T) {
	type fields struct {
		Reconciler    *resources.Reconciler
		ConfigManager config.ConfigReadWriter
		Config        *config.Observability
		installation  *integreatlyv1alpha1.RHMI
		mpm           marketplace.MarketplaceInterface
		log           logger.Logger
		extraParams   map[string]string
		recorder      record.EventRecorder
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
		installation  *integreatlyv1alpha1.RHMI
		mpm           *marketplace.MarketplaceInterfaceMock
		log           logger.Logger
		extraParams   map[string]string
		recorder      record.EventRecorder
	}
	type args struct {
		ctx          context.Context
		installation *integreatlyv1alpha1.RHMI
		product      *integreatlyv1alpha1.RHMIProductStatus
		client       client.Client
		in4          quota.ProductConfig
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
			got, err := r.Reconcile(tt.args.ctx, tt.args.installation, tt.args.product, tt.args.client, tt.args.in4)
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
		installation  *integreatlyv1alpha1.RHMI
		mpm           marketplace.MarketplaceInterface
		log           logger.Logger
		extraParams   map[string]string
		recorder      record.EventRecorder
	}
	type args struct {
		installation *integreatlyv1alpha1.RHMI
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

func TestReconciler_preUpgradeBackupExecutor(t *testing.T) {
	type fields struct {
		Reconciler    *resources.Reconciler
		ConfigManager config.ConfigReadWriter
		Config        *config.Observability
		installation  *integreatlyv1alpha1.RHMI
		mpm           marketplace.MarketplaceInterface
		log           logger.Logger
		extraParams   map[string]string
		recorder      record.EventRecorder
	}
	tests := []struct {
		name   string
		fields fields
		want   backup.BackupExecutor
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
			if got := r.preUpgradeBackupExecutor(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("preUpgradeBackupExecutor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_reconcileComponents(t *testing.T) {
	type fields struct {
		Reconciler    *resources.Reconciler
		ConfigManager config.ConfigReadWriter
		Config        *config.Observability
		installation  *integreatlyv1alpha1.RHMI
		mpm           marketplace.MarketplaceInterface
		log           logger.Logger
		extraParams   map[string]string
		recorder      record.EventRecorder
	}
	type args struct {
		ctx          context.Context
		serverClient client.Client
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
			got, err := r.reconcileComponents(tt.args.ctx, tt.args.serverClient, defaultInstallationNamespace)
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

func TestReconciler_reconcileSubscription(t *testing.T) {
	type fields struct {
		Reconciler    *resources.Reconciler
		ConfigManager config.ConfigReadWriter
		Config        *config.Observability
		installation  *integreatlyv1alpha1.RHMI
		mpm           marketplace.MarketplaceInterface
		log           logger.Logger
		extraParams   map[string]string
		recorder      record.EventRecorder
	}
	type args struct {
		ctx               context.Context
		serverClient      client.Client
		in2               *integreatlyv1alpha1.RHMI
		productNamespace  string
		operatorNamespace string
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
			got, err := r.reconcileSubscription(tt.args.ctx, tt.args.serverClient, tt.args.productNamespace, tt.args.operatorNamespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileSubscription() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("reconcileSubscription() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_fullReconcile(t *testing.T) {
	scheme, err := getBuildScheme()
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
		ExpectedStatus integreatlyv1alpha1.StatusPhase
		ExpectedError  string
		FakeConfig     *config.ConfigReadWriterMock
		FakeClient     k8sclient.Client
		FakeMPM        *marketplace.MarketplaceInterfaceMock
		Installation   *integreatlyv1alpha1.RHMI
		Product        *integreatlyv1alpha1.RHMIProductStatus
		Recorder       record.EventRecorder
	}{
		{
			Name:           "test successful reconcile",
			ExpectedStatus: integreatlyv1alpha1.PhaseInProgress,
			FakeClient:     moqclient.NewSigsClientMoqWithScheme(scheme, ns, operatorNS, installation),
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
			Product:      &integreatlyv1alpha1.RHMIProductStatus{},
			Recorder:     setupRecorder(),
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reconciler, err := NewReconciler(tc.FakeConfig, tc.Installation, tc.FakeMPM, tc.Recorder, getLogger(), localProductDeclaration)
			if err != nil && err.Error() != tc.ExpectedError {
				t.Fatalf("unexpected error : '%v', expected: '%v'", err, tc.ExpectedError)
			}
			status, err := reconciler.Reconcile(context.TODO(), tc.Installation, tc.Product, tc.FakeClient, &quota.ProductConfigMock{})
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

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := observability.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := v1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := projectv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := rbac.AddToScheme(scheme); err != nil {
		return nil, err
	}

	return scheme, nil
}

func basicInstallation() *integreatlyv1alpha1.RHMI {
	return &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "installation",
			Namespace: defaultInstallationNamespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "RHMI",
			APIVersion: integreatlyv1alpha1.GroupVersion.String(),
		},
		Spec: integreatlyv1alpha1.RHMISpec{
			//SMTPSecret:           mockSMTPSecretName,
			//PagerDutySecret:      mockPagerdutySecretName,
			//DeadMansSnitchSecret: mockDMSSecretName,
			Type: string(integreatlyv1alpha1.InstallationTypeManaged),
		},
	}
}

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.ProductApicurioRegistry})
}
