package marin3r

import (
	"context"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/utils"
	"k8s.io/client-go/tools/record"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	marin3rconfig "github.com/integr8ly/integreatly-operator/pkg/products/marin3r/config"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNSPrefix = "redhat-rhoam-"
)

var (
	localProductDeclaration = marketplace.LocalProductDeclaration("integreatly-marin3r")
	RateLimitConfig         = marin3rconfig.RateLimitConfig{
		Unit:            "minute",
		RequestsPerUnit: 1,
	}
)

func getRateLimitConfigMap() *corev1.ConfigMap {
	rateLimtCMString := `
- namespace: kuard
  max_value: 1
  seconds: 60
  conditions:
    - descriptors[0]['generic_key'] == "slowpath"
  variables: []`

	return &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      RateLimitingConfigMapName,
			Namespace: "marin3r",
			Labels: map[string]string{
				"app":     quota.RateLimitName,
				"part-of": "3scale-saas",
			},
		},
		Data: map[string]string{
			"kuard.yaml": rateLimtCMString,
		},
	}
}

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
}

func getBasicConfig() *config.ConfigReadWriterMock {
	return &config.ConfigReadWriterMock{
		GetOperatorNamespaceFunc: func() string {
			return defaultInstallationNamespace
		},
		ReadMarin3rFunc: func() (*config.Marin3r, error) {
			return &config.Marin3r{
				Config: config.ProductConfig{
					"NAMESPACE": defaultInstallationNamespace,
				},
			}, nil
		},
		ReadThreeScaleFunc: func() (*config.ThreeScale, error) {
			return config.NewThreeScale(config.ProductConfig{
				"NAMESPACE": "3scale",
				"HOST":      "threescale.openshift-cluster.com",
			}), nil
		},
		WriteConfigFunc: func(config config.ConfigReadable) error {
			return nil
		},
	}
}

func getBasicInstallation() *integreatlyv1alpha1.RHMI {
	return &integreatlyv1alpha1.RHMI{
		ObjectMeta: v1.ObjectMeta{
			Name:      "installation",
			Namespace: defaultInstallationNamespace,
			UID:       types.UID("xyz"),
		},
		TypeMeta: v1.TypeMeta{
			Kind:       "RHMI",
			APIVersion: integreatlyv1alpha1.GroupVersion.String(),
		},
		Spec: integreatlyv1alpha1.RHMISpec{
			NamespacePrefix: testNSPrefix,
		},
		Status: integreatlyv1alpha1.RHMIStatus{
			Stages: map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus{
				integreatlyv1alpha1.InstallStage: {
					Name:  integreatlyv1alpha1.InstallStage,
					Phase: integreatlyv1alpha1.PhaseInProgress,
					Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
						integreatlyv1alpha1.ProductGrafana: {
							Name:  integreatlyv1alpha1.ProductGrafana,
							Phase: integreatlyv1alpha1.PhaseInProgress,
						},
					},
				},
			},
		},
	}
}

func getGrafanaRoute() *routev1.Route {
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grafana-route",
			Namespace: testNSPrefix + "customer-monitoring",
		},
		Spec: routev1.RouteSpec{
			Host: "sampleHost",
		},
	}
}

func TestAlertCreation(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		serverClient func() k8sclient.Client
		FakeConfig   *config.ConfigReadWriterMock
		FakeMPM      *marketplace.MarketplaceInterfaceMock
		Recorder     record.EventRecorder
		installation *integreatlyv1alpha1.RHMI
		want         integreatlyv1alpha1.StatusPhase
		wantErr      string
		wantFn       func(c k8sclient.Client) error
	}{
		{
			name: "returns expected alerts",
			serverClient: func() k8sclient.Client {
				return utils.NewTestClient(scheme, getRateLimitConfigMap(), getGrafanaRoute())
			},

			installation: getBasicInstallation(),
			FakeConfig:   getBasicConfig(),
			Recorder:     setupRecorder(),
			want:         integreatlyv1alpha1.PhaseCompleted,
		},
		{
			name: "returns PhaseInProgress when grafana not installed",
			serverClient: func() k8sclient.Client {
				return utils.NewTestClient(scheme, getRateLimitConfigMap())
			},
			installation: getBasicInstallation(),
			FakeConfig:   getBasicConfig(),
			Recorder:     setupRecorder(),
			want:         integreatlyv1alpha1.PhaseInProgress,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			reconciler, err := NewReconciler(getBasicConfig(), tt.installation, tt.FakeMPM, tt.Recorder, getLogger(), localProductDeclaration)
			reconciler.RateLimitConfig = RateLimitConfig
			if err != nil {
				t.Fatalf("Could not create new reconiler")
			}

			serverClient := tt.serverClient()

			got, err := reconciler.reconcileAlerts(context.TODO(), serverClient, reconciler.installation, getBasicInstallation().Name)
			if tt.wantErr != "" && err.Error() != tt.wantErr {
				t.Errorf("reconcileAlerts() error = %v, wantErr %v", err.Error(), tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("reconcileAlerts() got = %v, want %v", got, tt.want)
			}
			if tt.wantFn != nil {
				if err := tt.wantFn(serverClient); err != nil {
					t.Errorf("reconcileAlerts() error = %v", err)
				}
			}
		})
	}
}

func TestReconcileServiceMonitor(t *testing.T) {
	namespaceName := "test-namespace"
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}
	client := utils.NewTestClient(scheme)

	type args struct {
		namespace string
	}
	tests := []struct {
		name    string
		args    args
		FakeMPM *marketplace.MarketplaceInterfaceMock
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name:    "success on valid namespace",
			args:    args{namespace: namespaceName},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewReconciler(getBasicConfig(), getBasicInstallation(), tt.FakeMPM, setupRecorder(), getLogger(), localProductDeclaration)
			r.RateLimitConfig = RateLimitConfig
			if err != nil {
				t.Fatalf("Could not create new reconiler")
			}

			got, err := r.reconcileServiceMonitor(context.TODO(), client, tt.args.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileServiceMonitor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("reconcileServiceMonitor() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconcileDiscoveryService(t *testing.T) {
	namespaceName := "test-namespace"
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}
	client := utils.NewTestClient(scheme)

	type args struct {
		namespace string
	}
	tests := []struct {
		name    string
		args    args
		FakeMPM *marketplace.MarketplaceInterfaceMock
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name:    "success on valid namespace",
			args:    args{namespace: namespaceName},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewReconciler(getBasicConfig(), getBasicInstallation(), tt.FakeMPM, setupRecorder(), getLogger(), localProductDeclaration)
			r.RateLimitConfig = RateLimitConfig
			if err != nil {
				t.Fatalf("Could not create new reconiler")
			}

			got, err := r.reconcileDiscoveryService(context.TODO(), client, tt.args.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileDiscoveryService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("reconcileDiscoveryService() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.ProductMarin3r})
}
