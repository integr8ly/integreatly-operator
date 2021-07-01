package cloudresources

import (
	"bytes"
	"context"
	"testing"

	threescalev1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	monitoringv1 "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	crov1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
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
	cloudcredentialv1 "github.com/openshift/api/operator/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	operatorNamespace = "openshift-operators"
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
	err = cloudcredentialv1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	return scheme, err
}

func TestReconciler_validateStsRoleArnPattern(t *testing.T) {

	tests := []struct {
		name          string
		awsArnPattern string
		want          bool
	}{
		{
			name:          "ARN string Pattern is matching for AWS GovCloud (US) Regions",
			awsArnPattern: "arn:aws-us-gov:iam::485026278258:role/12345",
			want:          true,
		},
		{
			name:          "ARN string Pattern is matching",
			awsArnPattern: "arn:aws:iam::485026278258:role/12345",
			want:          true,
		},
		{
			name:          "ARN string Pattern is not matching #1",
			awsArnPattern: "arn:aws:iam::485026278258:user/12345",
			want:          false,
		},
		{
			name:          "ARN string Pattern is not matching #2",
			awsArnPattern: "12345",
			want:          false,
		},
		{
			name:          "ARN string Pattern is not matching #3",
			awsArnPattern: "",
			want:          false,
		},
	}
	for _, tt := range tests {
		got, err := validateStsRoleArnPattern(tt.awsArnPattern)
		if err != nil {
			t.Errorf("failed to validate STS role ARN parameter: %w", err)
			return
		}
		if got != tt.want {
			t.Errorf("validateStsRoleArnPattern() got = %v, want %v", got, tt.want)
		}
	}

}

func TestReconciler_passArnIntoSecretInCroNamespace(t *testing.T) {
	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorNamespace,
			Namespace: operatorNamespace,
		},
		Data: map[string][]byte{
			"role_arn": {'t', 'e', 's', 't'},
		},
	}
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("Error obtaining scheme")
	}
	type args struct {
		ctx    context.Context
		client client.Client
	}
	tests := []struct {
		name    string
		ARN     string
		args    args
		wantErr bool
	}{
		{
			name: "ARN Secret passed to Cro Namespace",
			ARN:  "test", //Note that any ARN will be passed, there is no checking of ARN pattern here.
			args: args{
				ctx: context.TODO(),
				client: moqclient.NewSigsClientMoqWithScheme(scheme, sourceSecret, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: operatorNamespace,
						Name:      operatorNamespace,
					},
				}),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		err := passArnIntoSecretInCroNamespace(tt.args.ctx, tt.args.client, operatorNamespace, tt.ARN)
		if (err != nil) != tt.wantErr {
			t.Errorf("passArnIntoSecretInCroNamespace() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		destinationSecret := &corev1.Secret{}
		err = tt.args.client.Get(context.TODO(), k8sclient.ObjectKey{Name: croSecretName, Namespace: operatorNamespace}, destinationSecret)
		if err != nil {
			return
		}
		if !bytes.Equal(destinationSecret.Data["role_arn"], sourceSecret.Data["role_arn"]) {
			t.Fatalf("expected data %v, but got %v", sourceSecret.Data["role_arn"], destinationSecret.Data["role_arn"])
		}
	}
}

func TestReconciler_checkIfStsClusterByCredentialsMode(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("Error obtaining scheme")
	}
	tests := []struct {
		name       string
		ARN        string
		fakeClient k8sclient.Client
		want       bool
		wantErr    bool
	}{
		{
			name: "STS cluster",
			fakeClient: fake.NewFakeClientWithScheme(scheme, &cloudcredentialv1.CloudCredential{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: cloudcredentialv1.CloudCredentialSpec{
					CredentialsMode: cloudcredentialv1.CloudCredentialsModeManual,
				},
			}),
			want:    true,
			wantErr: false,
		},
		{
			name: "Non STS cluster",
			fakeClient: fake.NewFakeClientWithScheme(scheme, &cloudcredentialv1.CloudCredential{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: cloudcredentialv1.CloudCredentialSpec{
					CredentialsMode: cloudcredentialv1.CloudCredentialsModeDefault,
				},
			}),
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		got, err := checkIfStsClusterByCredentialsMode(context.TODO(), tt.fakeClient, operatorNamespace)
		if (err != nil) != tt.wantErr {
			t.Errorf("checkIfStsClusterByCredentialsMode() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
		if got != tt.want {
			t.Errorf("checkIfStsClusterByCredentialsMode() got = %v, want %v", got, tt.want)
		}
	}
}
