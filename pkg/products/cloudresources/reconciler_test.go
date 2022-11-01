package cloudresources

import (
	"context"
	"github.com/integr8ly/integreatly-operator/pkg/resources/sts"
	"github.com/integr8ly/integreatly-operator/test/utils"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"

	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestReconciler_cleanupResources(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Config        *config.CloudResources
		ConfigManager config.ConfigReadWriter
		installation  *integreatlyv1alpha1.RHMI
		mpm           marketplace.MarketplaceInterface
		log           logger.Logger
		Reconciler    *resources.Reconciler
		recorder      record.EventRecorder
	}
	type args struct {
		ctx          context.Context
		installation *integreatlyv1alpha1.RHMI
		client       client.Client
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    v1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "Test uninstallation: missing Postgres Instances CRD returns phaseCompleted",
			fields: fields{
				log: getLogger(),
			},
			args: args{
				ctx:    context.TODO(),
				client: moqclient.NewSigsClientMoqWithScheme(scheme),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "Test uninstallation: missing v1 API returns phaseFailed",
			fields: fields{
				log: getLogger(),
			},
			args: args{
				ctx:    context.TODO(),
				client: moqclient.NewSigsClientMoqWithScheme(runtime.NewScheme()),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				Config:        tt.fields.Config,
				ConfigManager: tt.fields.ConfigManager,
				installation:  tt.fields.installation,
				mpm:           tt.fields.mpm,
				log:           tt.fields.log,
				Reconciler:    tt.fields.Reconciler,
				recorder:      tt.fields.recorder,
			}
			got, err := r.cleanupResources(tt.args.ctx, tt.args.installation, tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("cleanupResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("cleanupResources() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconciler_removeSnapshots(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Config        *config.CloudResources
		ConfigManager config.ConfigReadWriter
		installation  *integreatlyv1alpha1.RHMI
		mpm           marketplace.MarketplaceInterface
		log           logger.Logger
		Reconciler    *resources.Reconciler
		recorder      record.EventRecorder
	}
	type args struct {
		ctx          context.Context
		installation *integreatlyv1alpha1.RHMI
		client       client.Client
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    v1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "Test uninstallation: missing Postgres Instances CRD returns phaseCompleted",
			fields: fields{
				log: getLogger(),
			},
			args: args{
				ctx:    context.TODO(),
				client: moqclient.NewSigsClientMoqWithScheme(scheme),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "Test uninstallation: missing v1 API returns phaseFailed",
			fields: fields{
				log: getLogger(),
			},
			args: args{
				ctx:    context.TODO(),
				client: moqclient.NewSigsClientMoqWithScheme(runtime.NewScheme()),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				Config:        tt.fields.Config,
				ConfigManager: tt.fields.ConfigManager,
				installation:  tt.fields.installation,
				mpm:           tt.fields.mpm,
				log:           tt.fields.log,
				Reconciler:    tt.fields.Reconciler,
				recorder:      tt.fields.recorder,
			}
			got, err := r.removeSnapshots(tt.args.ctx, tt.args.installation, tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("removeSnapshots() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("removeSnapshots() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func getLogger() logger.Logger {
	return logger.NewLoggerWithContext(logger.Fields{logger.ProductLogContext: integreatlyv1alpha1.ProductCloudResources})
}

func TestReconciler_checkStsCredentialsPresent(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Config        *config.CloudResources
		ConfigManager config.ConfigReadWriter
		installation  *integreatlyv1alpha1.RHMI
		mpm           marketplace.MarketplaceInterface
		log           logger.Logger
		Reconciler    *resources.Reconciler
		recorder      record.EventRecorder
	}
	type args struct {
		client            client.Client
		operatorNamespace string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "search sts-credentials secret completed successfully",
			fields: fields{
				Config:        nil,
				ConfigManager: nil,
				installation:  nil,
				mpm:           nil,
				log:           logger.Logger{},
				Reconciler:    nil,
				recorder:      nil,
			},
			args: args{
				client: fakeclient.NewFakeClientWithScheme(
					scheme,
					&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: sts.CredsSecretName, Namespace: "cro-operator-test"}},
				),
				operatorNamespace: "cro-operator-test",
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "search sts-credentials secret completed successfully",
			fields: fields{
				Config:        nil,
				ConfigManager: nil,
				installation:  nil,
				mpm:           nil,
				log:           logger.Logger{},
				Reconciler:    nil,
				recorder:      nil,
			},
			args: args{
				client: fakeclient.NewFakeClientWithScheme(
					scheme,
				),
				operatorNamespace: "cro-operator-test",
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				Config:        tt.fields.Config,
				ConfigManager: tt.fields.ConfigManager,
				installation:  tt.fields.installation,
				mpm:           tt.fields.mpm,
				log:           tt.fields.log,
				Reconciler:    tt.fields.Reconciler,
				recorder:      tt.fields.recorder,
			}
			got, err := r.checkStsCredentialsPresent(tt.args.client, tt.args.operatorNamespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkStsCredentialsPresent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkStsCredentialsPresent() got = %v, want %v", got, tt.want)
			}
		})
	}
}
