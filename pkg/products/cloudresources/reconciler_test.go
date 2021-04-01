package cloudresources

import (
	"context"
	threescalev1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	monitoringv1 "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	kafkav1alpha1 "github.com/integr8ly/integreatly-operator/apis-products/kafka.strimzi.io/v1alpha1"
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

func TestReconciler_cleanupResources(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("Error obtaining sheme")
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
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("Error obtaining sheme")
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
	err = apiextensionv1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	return scheme, err
}
