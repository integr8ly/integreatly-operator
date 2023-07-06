package status

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/utils"
	addonv1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	addoninstance "github.com/openshift/addon-operator/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestStatusReconciler_BuildAddonInstanceConditions(t *testing.T) {
	var installation v1alpha1.RHMI

	type args struct {
		installation *v1alpha1.RHMI
	}
	tests := []struct {
		name string
		args args
		want []metav1.Condition
	}{
		{
			name: "test uninstalled condition if installation is nil",
			want: []metav1.Condition{installation.UninstalledCondition()},
		},
		{
			name: "test installed, core components unhealthy and degraded condition",
			args: args{
				installation: &v1alpha1.RHMI{Status: v1alpha1.RHMIStatus{Version: "0.0.0", Stage: v1alpha1.ProductsStage}},
			},
			want: []metav1.Condition{installation.InstalledCondition(), installation.UnHealthyCondition(), installation.DegradedCondition()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &StatusReconciler{
				Log: logger.NewLogger(),
			}
			if got := r.buildAddonInstanceConditions(tt.args.installation); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildAddonInstanceConditions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatusReconciler_AppendInstalledConditions(t *testing.T) {
	var installation v1alpha1.RHMI

	type args struct {
		installation *v1alpha1.RHMI
	}
	tests := []struct {
		name string
		args args
		want []metav1.Condition
	}{
		{
			name: "test installed condition returned if installed",
			args: args{installation: &v1alpha1.RHMI{Status: v1alpha1.RHMIStatus{Version: "0.0.0"}}},
			want: []metav1.Condition{installation.InstalledCondition()},
		},
		{
			name: "test installed blocked condition returned if installed over 2 hours and version not set",
			args: args{installation: &v1alpha1.RHMI{ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: time.Now().Add(-2 * time.Hour)}}, Status: v1alpha1.RHMIStatus{Version: ""}}},
			want: []metav1.Condition{installation.InstallBlockedCondition()},
		},
		{
			name: "test uninstalled blocked condition returned if uninstalled over 2 hours",
			args: args{installation: &v1alpha1.RHMI{ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &metav1.Time{Time: time.Now().Add(-2 * time.Hour)}}}},
			want: []metav1.Condition{installation.UninstallBlockedCondition()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &StatusReconciler{}
			if got := r.appendInstalledConditions(tt.args.installation); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AppendInstalledConditions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatusReconciler_AppendHealthConditions(t *testing.T) {
	var installation v1alpha1.RHMI
	statusFactory := func(phase1, phase2, phase3 v1alpha1.StatusPhase) v1alpha1.RHMIStatus {
		return v1alpha1.RHMIStatus{Stages: map[v1alpha1.StageName]v1alpha1.RHMIStageStatus{
			v1alpha1.InstallStage: {
				Name: v1alpha1.InstallStage,
				Products: map[v1alpha1.ProductName]v1alpha1.RHMIProductStatus{
					v1alpha1.Product3Scale: {
						Phase: phase1,
					},
					v1alpha1.ProductRHSSOUser: {
						Phase: phase2,
					},
					v1alpha1.ProductCloudResources: {
						Phase: phase3,
					},
				},
			},
		}}
	}

	type args struct {
		installation *v1alpha1.RHMI
	}
	tests := []struct {
		name string
		args args
		want []metav1.Condition
	}{
		{
			name: "test health condition when core components is healthy",
			args: args{installation: &v1alpha1.RHMI{Status: statusFactory(v1alpha1.PhaseCompleted, v1alpha1.PhaseCompleted, v1alpha1.PhaseCompleted)}},
			want: []metav1.Condition{installation.HealthyCondition()},
		},
		{
			name: "test unhealthy condition when core components is unhealthy",
			args: args{installation: &v1alpha1.RHMI{Status: statusFactory(v1alpha1.PhaseFailed, v1alpha1.PhaseCompleted, v1alpha1.PhaseCompleted)}},
			want: []metav1.Condition{installation.UnHealthyCondition()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &StatusReconciler{}
			if got := r.appendHealthConditions(tt.args.installation); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AppendHealthConditions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatusReconciler_AppendDegradedConditions(t *testing.T) {
	var installation v1alpha1.RHMI

	type args struct {
		installation *v1alpha1.RHMI
	}
	tests := []struct {
		name string
		args args
		want []metav1.Condition
	}{
		{
			name: "test degraded condition if stage not stage complete",
			args: args{installation: &v1alpha1.RHMI{Status: v1alpha1.RHMIStatus{Stage: v1alpha1.ProductsStage}}},
			want: []metav1.Condition{installation.DegradedCondition()},
		},
		{
			name: "test not degraded condition if stage is stage complete",
			args: args{installation: &v1alpha1.RHMI{Status: v1alpha1.RHMIStatus{Stage: v1alpha1.CompleteStage}}},
			want: []metav1.Condition{installation.NonDegradedCondition()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &StatusReconciler{
				Log: logger.NewLogger(),
			}
			if got := r.appendDegradedConditions(tt.args.installation); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AppendDegradedConditions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatusReconciler_UpdateAddonInstanceWithConditions(t *testing.T) {
	testDetail := "test"
	ctx := context.Background()
	requestFactory := func(name, namespace string) controllerruntime.Request {
		return controllerruntime.Request{NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}}
	}
	cfg := ControllerOptions{AddonInstanceName: testDetail, AddonInstanceNamespace: testDetail}
	cfg.Default()

	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}
	addonInstance := &addonv1alpha1.AddonInstance{ObjectMeta: metav1.ObjectMeta{Name: testDetail, Namespace: testDetail}}
	basicClient := utils.NewTestClient(scheme, addonInstance)
	errStatusClient := moqclient.NewSigsClientMoqWithScheme(scheme, addonInstance)
	errStatusClient.StatusFunc = func() client.SubResourceWriter {
		return utils.NewSubResourceWriterMock(true)
	}

	type fields struct {
		Client client.Client
	}
	type args struct {
		ctx        context.Context
		req        controllerruntime.Request
		conditions []metav1.Condition
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test successfully updating addon instance",
			args: args{
				ctx:        ctx,
				req:        requestFactory(testDetail, testDetail),
				conditions: []metav1.Condition{},
			},
			fields: fields{
				Client: basicClient,
			},
		},
		{
			name: "test error getting addon instance",
			args: args{
				ctx:        ctx,
				req:        requestFactory(testDetail, "non-existent"),
				conditions: []metav1.Condition{},
			},
			fields: fields{
				Client: basicClient,
			},
			wantErr: true,
		},
		{
			name: "test error updating addon instance",
			args: args{
				ctx:        ctx,
				req:        requestFactory(testDetail, testDetail),
				conditions: []metav1.Condition{},
			},
			fields: fields{
				Client: errStatusClient,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &StatusReconciler{
				Client:              tt.fields.Client,
				cfg:                 cfg,
				addonInstanceClient: addoninstance.NewAddonInstanceClient(tt.fields.Client),
			}
			if err := r.updateAddonInstanceWithConditions(tt.args.ctx, addonInstance, tt.args.conditions); (err != nil) != tt.wantErr {
				t.Errorf("UpdateAddonInstanceWithConditions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStatusReconciler_Reconcile(t *testing.T) {
	testDetail := "test"
	ctx := context.Background()
	requestFactory := func(name, namespace string) controllerruntime.Request {
		return controllerruntime.Request{NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}}
	}
	cfg := ControllerOptions{AddonInstanceName: testDetail, AddonInstanceNamespace: testDetail}
	cfg.Default()

	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Client client.Client
		Log    logger.Logger
	}
	type args struct {
		ctx context.Context
		req controllerruntime.Request
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    controllerruntime.Result
		wantErr bool
	}{
		{
			name: "test successful reconcile",
			args: args{
				ctx: ctx,
				req: requestFactory(testDetail, testDetail),
			},
			fields: fields{
				Client: utils.NewTestClient(scheme, &addonv1alpha1.AddonInstance{ObjectMeta: metav1.ObjectMeta{Name: testDetail, Namespace: testDetail}}, &v1alpha1.RHMI{ObjectMeta: metav1.ObjectMeta{Namespace: testDetail}}),
			},
			want:    controllerruntime.Result{RequeueAfter: defaultRequeueTime},
			wantErr: false,
		},
		{
			name: "test error patching AddonInstance CR",
			args: args{
				ctx: ctx,
				req: requestFactory(testDetail, testDetail),
			},
			fields: fields{
				Client: &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
						return nil
					},
					PatchFunc: func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
						return fmt.Errorf("error")
					},
				},
			},
			want:    controllerruntime.Result{},
			wantErr: true,
		},
		{
			name: "test error getting RHMI CR",
			args: args{
				ctx: ctx,
				req: requestFactory(testDetail, testDetail),
			},
			fields: fields{
				Client: &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
						return nil
					},
					PatchFunc: func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
						return nil
					},
					ListFunc: func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
						return fmt.Errorf("error")
					},
				},
			},
			want:    controllerruntime.Result{},
			wantErr: true,
		},
		{
			name: "test error updating addon instance",
			args: args{
				ctx: ctx,
				req: requestFactory(testDetail, "non-existent"),
			},
			fields: fields{
				Client: utils.NewTestClient(scheme, &addonv1alpha1.AddonInstance{ObjectMeta: metav1.ObjectMeta{Name: testDetail, Namespace: testDetail}}, &v1alpha1.RHMI{ObjectMeta: metav1.ObjectMeta{Namespace: testDetail}}),
			},
			want:    controllerruntime.Result{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &StatusReconciler{
				Client:              tt.fields.Client,
				Log:                 logger.NewLogger(),
				cfg:                 cfg,
				addonInstanceClient: addoninstance.NewAddonInstanceClient(tt.fields.Client),
			}
			got, err := r.Reconcile(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reconcile() got = %v, want %v", got, tt.want)
			}
		})
	}
}
