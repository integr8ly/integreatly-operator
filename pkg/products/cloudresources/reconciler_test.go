package cloudresources

import (
	"context"
	"testing"

	croAWS "github.com/integr8ly/cloud-resource-operator/pkg/providers/aws"
	croGCP "github.com/integr8ly/cloud-resource-operator/pkg/providers/gcp"
	"github.com/integr8ly/integreatly-operator/pkg/resources/sts"
	"github.com/integr8ly/integreatly-operator/utils"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	configv1 "github.com/openshift/api/config/v1"
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
		want    integreatlyv1alpha1.StatusPhase
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
		want    integreatlyv1alpha1.StatusPhase
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
				client:            utils.NewTestClient(scheme, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: sts.CredsSecretName, Namespace: "cro-operator-test"}}),
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
				client:            utils.NewTestClient(scheme),
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

func TestReconciler_setPlatformStrategyName(t *testing.T) {
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
		client client.Client
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "successfully set strategy name for aws infrastructure",
			fields: fields{
				Config: config.NewCloudResources(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				ConfigManager: nil,
				installation:  nil,
				mpm:           nil,
				log:           logger.Logger{},
				Reconciler:    nil,
				recorder:      nil,
			},
			args: args{
				client: moqclient.NewSigsClientMoqWithScheme(scheme, &configv1.Infrastructure{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Status: configv1.InfrastructureStatus{
						PlatformStatus: &configv1.PlatformStatus{
							Type: configv1.AWSPlatformType,
						},
					},
				}),
			},
			want:    croAWS.DefaultConfigMapName,
			wantErr: false,
		},
		{
			name: "successfully set strategy name for gcp infrastructure",
			fields: fields{
				Config: config.NewCloudResources(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				ConfigManager: nil,
				installation:  nil,
				mpm:           nil,
				log:           logger.Logger{},
				Reconciler:    nil,
				recorder:      nil,
			},
			args: args{
				client: moqclient.NewSigsClientMoqWithScheme(scheme, &configv1.Infrastructure{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Status: configv1.InfrastructureStatus{
						PlatformStatus: &configv1.PlatformStatus{
							Type: configv1.GCPPlatformType,
						},
					},
				}),
			},
			want:    croGCP.DefaultConfigMapName,
			wantErr: false,
		},
		{
			name: "error determining platform type",
			fields: fields{
				Config: config.NewCloudResources(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				ConfigManager: nil,
				installation:  nil,
				mpm:           nil,
				log:           logger.Logger{},
				Reconciler:    nil,
				recorder:      nil,
			},
			args: args{
				client: moqclient.NewSigsClientMoqWithScheme(scheme),
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "error unsupported platform type",
			fields: fields{
				Config: config.NewCloudResources(config.ProductConfig{
					"NAMESPACE": "test",
				}),
				ConfigManager: nil,
				installation:  nil,
				mpm:           nil,
				log:           logger.Logger{},
				Reconciler:    nil,
				recorder:      nil,
			},
			args: args{
				client: moqclient.NewSigsClientMoqWithScheme(scheme, &configv1.Infrastructure{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Status: configv1.InfrastructureStatus{
						PlatformStatus: &configv1.PlatformStatus{
							Type: configv1.AzurePlatformType,
						},
					},
				}),
			},
			want:    "",
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
			err := r.setPlatformStrategyName(context.TODO(), tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("setPlatformStrategyName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if r.Config.GetStrategiesConfigMapName() != tt.want {
				t.Errorf("setPlatformStrategyName() got = %v, want %v", r.Config.GetStrategiesConfigMapName(), tt.want)
			}
		})
	}
}

func TestReconciler_reconcileCIDRValue(t *testing.T) {
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
		ctx    context.Context
		client client.Client
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "success reconciling cidr value - aws",
			fields: fields{
				Config: config.NewCloudResources(config.ProductConfig{
					"STRATEGIES_CONFIG_MAP_NAME": croAWS.DefaultConfigMapName,
				}),
				ConfigManager: nil,
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "managed-api",
						Namespace: "test",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: "managed-api",
					},
				},
				mpm:        nil,
				log:        logger.Logger{},
				Reconciler: nil,
				recorder:   nil,
			},
			args: args{
				ctx: context.TODO(),
				client: moqclient.NewSigsClientMoqWithScheme(scheme,
					clusterInfrastructure(configv1.AWSPlatformType),
					addonParamsSecret("test",
						map[string][]byte{
							cidrRangeKeyAws: []byte("10.1.0.0/26"),
						},
					),
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      croAWS.DefaultConfigMapName,
							Namespace: "test",
						},
						Data: map[string]string{
							"_network": `{"development": { "region": "", "_network": "", "createStrategy": {}, "deleteStrategy": {} }, "production": { "region": "", "_network": "","createStrategy": {}, "deleteStrategy": {} }}`,
						},
					},
				),
			},
			wantErr: false,
		},
		{
			name: "success reconciling cidr value - gcp",
			fields: fields{
				Config: config.NewCloudResources(config.ProductConfig{
					"STRATEGIES_CONFIG_MAP_NAME": croGCP.DefaultConfigMapName,
				}),
				ConfigManager: nil,
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "managed-api",
						Namespace: "test",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: "managed-api",
					},
				},
				mpm:        nil,
				log:        logger.Logger{},
				Reconciler: nil,
				recorder:   nil,
			},
			args: args{
				ctx: context.TODO(),
				client: moqclient.NewSigsClientMoqWithScheme(scheme,
					clusterInfrastructure(configv1.GCPPlatformType),
					addonParamsSecret("test",
						map[string][]byte{
							cidrRangeKeyGcp: []byte("10.1.0.0/22"),
						},
					),
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      croGCP.DefaultConfigMapName,
							Namespace: "test",
						},
						Data: map[string]string{
							"_network": `{"development": { "region": "", "projectID": "", "createStrategy": {}, "deleteStrategy": {} }, "production": { "region": "", "projectID": "", "createStrategy": {}, "deleteStrategy": {} }}`,
						},
					},
				),
			},
			wantErr: false,
		},
		{
			name: "error determining platform type",
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
				ctx:    context.TODO(),
				client: moqclient.NewSigsClientMoqWithScheme(scheme),
			},
			wantErr: true,
		},
		{
			name: "error unsupported platform type",
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
				ctx: context.TODO(),
				client: moqclient.NewSigsClientMoqWithScheme(scheme,
					clusterInfrastructure(configv1.AzurePlatformType),
				),
			},
			wantErr: true,
		},
		{
			name: "error retrieving cidr range value",
			fields: fields{
				Config:        nil,
				ConfigManager: nil,
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "managed-api",
						Namespace: "test",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: "managed-api",
					},
				},
				mpm:        nil,
				log:        logger.Logger{},
				Reconciler: nil,
				recorder:   nil,
			},
			args: args{
				ctx: context.TODO(),
				client: moqclient.NewSigsClientMoqWithScheme(scheme,
					clusterInfrastructure(configv1.AWSPlatformType),
				),
			},
			wantErr: true,
		},
		{
			name: "error retrieving strategy config map",
			fields: fields{
				Config: config.NewCloudResources(config.ProductConfig{
					"STRATEGIES_CONFIG_MAP_NAME": croAWS.DefaultConfigMapName,
				}),
				ConfigManager: nil,
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "managed-api",
						Namespace: "test",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: "managed-api",
					},
				},
				mpm:        nil,
				log:        logger.Logger{},
				Reconciler: nil,
				recorder:   nil,
			},
			args: args{
				ctx: context.TODO(),
				client: moqclient.NewSigsClientMoqWithScheme(scheme,
					clusterInfrastructure(configv1.AWSPlatformType),
					addonParamsSecret("test",
						map[string][]byte{
							cidrRangeKeyAws: []byte("10.1.0.0/26"),
						},
					),
				),
			},
			wantErr: true,
		},
		{
			name: "error decoding strategy config",
			fields: fields{
				Config: config.NewCloudResources(config.ProductConfig{
					"STRATEGIES_CONFIG_MAP_NAME": croAWS.DefaultConfigMapName,
				}),
				ConfigManager: nil,
				installation: &integreatlyv1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "managed-api",
						Namespace: "test",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: "managed-api",
					},
				},
				mpm:        nil,
				log:        logger.Logger{},
				Reconciler: nil,
				recorder:   nil,
			},
			args: args{
				ctx: context.TODO(),
				client: moqclient.NewSigsClientMoqWithScheme(scheme,
					clusterInfrastructure(configv1.AWSPlatformType),
					addonParamsSecret("test",
						map[string][]byte{
							cidrRangeKeyAws: []byte("10.1.0.0/26"),
						},
					),
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      croAWS.DefaultConfigMapName,
							Namespace: "test",
						},
						Data: map[string]string{
							"_network": "invalid json",
						},
					},
				),
			},
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
			if err := r.reconcileCIDRValue(tt.args.ctx, tt.args.client); (err != nil) != tt.wantErr {
				t.Errorf("reconcileCIDRValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReconciler_reconcileCloudResourceStrategies(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	const testNamespace = "test-namespace"

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
		client client.Client
		ctx    context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "success when params are entered in UI",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{
					GetOperatorNamespaceFunc: func() string {
						return testNamespace
					},
				},
				log: getLogger(),
				Reconciler: resources.NewReconciler(&marketplace.MarketplaceInterfaceMock{}).
					WithProductDeclaration(marketplace.ProductDeclaration{}),
				recorder: nil,
			},
			args: args{
				client: moqclient.NewSigsClientMoqWithScheme(scheme,
					clusterInfrastructure(configv1.GCPPlatformType),
					addonParamsSecret(testNamespace,
						map[string][]byte{
							MaintenanceDay:  []byte("2"),
							MaintenanceHour: []byte("5"),
						},
					),
				),
				ctx: context.TODO(),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "success when params are not entered in UI, use defaults",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{
					GetOperatorNamespaceFunc: func() string {
						return testNamespace
					},
				},
				log: getLogger(),
				Reconciler: resources.NewReconciler(&marketplace.MarketplaceInterfaceMock{}).
					WithProductDeclaration(marketplace.ProductDeclaration{}),
			},
			args: args{
				client: moqclient.NewSigsClientMoqWithScheme(scheme,
					clusterInfrastructure(configv1.GCPPlatformType),
					addonParamsSecret(testNamespace,
						map[string][]byte{
							MaintenanceDay:  []byte(""),
							MaintenanceHour: []byte(""),
						},
					),
				),
				ctx: context.TODO(),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "success when params are not in addon params secret (aws), use defaults",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{
					GetOperatorNamespaceFunc: func() string {
						return testNamespace
					},
				},
				log: getLogger(),
				Reconciler: resources.NewReconciler(&marketplace.MarketplaceInterfaceMock{}).
					WithProductDeclaration(marketplace.ProductDeclaration{}),
			},
			args: args{
				client: moqclient.NewSigsClientMoqWithScheme(scheme,
					clusterInfrastructure(configv1.AWSPlatformType),
					addonParamsSecret(testNamespace,
						map[string][]byte{},
					),
				),
				ctx: context.TODO(),
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "error when incorrect values entered for maintenanceDay",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{
					GetOperatorNamespaceFunc: func() string {
						return testNamespace
					},
				},
				log: getLogger(),
				Reconciler: resources.NewReconciler(&marketplace.MarketplaceInterfaceMock{}).
					WithProductDeclaration(marketplace.ProductDeclaration{}),
			},
			args: args{
				client: moqclient.NewSigsClientMoqWithScheme(scheme,
					clusterInfrastructure(configv1.GCPPlatformType),
					addonParamsSecret(testNamespace,
						map[string][]byte{
							MaintenanceDay:  []byte("Monday"),
							MaintenanceHour: []byte("2"),
						},
					),
				),
				ctx: context.TODO(),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "error when incorrect values entered for maintenanceHour",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{
					GetOperatorNamespaceFunc: func() string {
						return testNamespace
					},
				},
				log: getLogger(),
				Reconciler: resources.NewReconciler(&marketplace.MarketplaceInterfaceMock{}).
					WithProductDeclaration(marketplace.ProductDeclaration{}),
			},
			args: args{
				client: moqclient.NewSigsClientMoqWithScheme(scheme,
					clusterInfrastructure(configv1.GCPPlatformType),
					addonParamsSecret(testNamespace,
						map[string][]byte{
							MaintenanceDay:  []byte("4"),
							MaintenanceHour: []byte("Two"),
						},
					),
				),
				ctx: context.TODO(),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "failure reconciling strategy map",
			fields: fields{
				ConfigManager: &config.ConfigReadWriterMock{
					GetOperatorNamespaceFunc: func() string {
						return testNamespace
					},
				},
				log: getLogger(),
				Reconciler: resources.NewReconciler(&marketplace.MarketplaceInterfaceMock{}).
					WithProductDeclaration(marketplace.ProductDeclaration{}),
				recorder: nil,
			},
			args: args{
				client: moqclient.NewSigsClientMoqWithScheme(scheme, &configv1.Infrastructure{}),
				ctx:    context.TODO(),
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
			got, err := r.reconcileCloudResourceStrategies(tt.args.ctx, tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileCloudResourceStrategies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("reconcileCloudResourceStrategies() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func clusterInfrastructure(platformType configv1.PlatformType) *configv1.Infrastructure {
	return &configv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Status: configv1.InfrastructureStatus{
			PlatformStatus: &configv1.PlatformStatus{
				Type: platformType,
			},
		},
	}
}

func addonParamsSecret(namespace string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "addon-managed-api-service-parameters",
			Namespace: namespace,
		},
		Data: data,
	}
}
