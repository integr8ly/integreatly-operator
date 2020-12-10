package marin3r

import (
	"context"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"testing"

	"github.com/integr8ly/integreatly-operator/test/common"

	prometheusmonitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	marin3rconfig "github.com/integr8ly/integreatly-operator/pkg/products/marin3r/config"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	coreosv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func getRateLimitConfigMap() *corev1.ConfigMap {
	rateLimtCMString := `
domain: kuard
descriptors:
  - key: generic_key
    value: slowpath
    ratelimit:
      unit: minute
      requestsperunit: 1`

	return &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      "ratelimit-config",
			Namespace: "marin3r",
			Labels: map[string]string{
				"app":     "ratelimit",
				"part-of": "3scale-saas",
			},
		},
		Data: map[string]string{
			"kuard.yaml": rateLimtCMString,
		},
	}
}

func getBasicReconciler() *Reconciler {
	return &Reconciler{
		installation: getBasicInstallation(),
		log:          getLogger(),
		Config: &config.Marin3r{
			Config: config.ProductConfig{
				"NAMESPACE": defaultInstallationNamespace,
			},
		},
		AlertsConfig: map[string]*marin3rconfig.AlertConfig{
			"api-usage-alert-level1": {
				Type:     marin3rconfig.AlertTypeThreshold,
				RuleName: "RHOAMApiUsageLevel1ThresholdExceeded",
				Level:    "warning",
				Threshold: &marin3rconfig.AlertThresholdConfig{
					MinRate: "80%",
					MaxRate: strPtr("90%"),
				},
				Period: "4h",
			},
			"api-usage-alert-level2": {
				Type:     marin3rconfig.AlertTypeThreshold,
				RuleName: "RHOAMApiUsageLevel2ThresholdExceeded",
				Level:    "warning",
				Threshold: &marin3rconfig.AlertThresholdConfig{
					MinRate: "90%",
					MaxRate: strPtr("95%"),
				},
				Period: "2h",
			},
			"api-usage-alert-level3": {
				Type:     marin3rconfig.AlertTypeThreshold,
				RuleName: "RHOAMApiUsageLevel3ThresholdExceeded",
				Level:    "warning",
				Threshold: &marin3rconfig.AlertThresholdConfig{
					MinRate: "95%",
				},
				Period: "30m",
			},
			"rate-limit-spike": {
				Type:     marin3rconfig.AlertTypeSpike,
				RuleName: "RHOAMApiUsageOverLimit",
				Level:    "warning",
				Period:   "30m",
			},
		},
		RateLimitConfig: &marin3rconfig.RateLimitConfig{
			Unit:            "minute",
			RequestsPerUnit: 1,
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
			Kind:       integreatlyv1alpha1.SchemaGroupVersionKind.Kind,
			APIVersion: integreatlyv1alpha1.SchemeGroupVersion.String(),
		},
		Spec: integreatlyv1alpha1.RHMISpec{
			NamespacePrefix: common.NamespacePrefix,
		},
		Status: integreatlyv1alpha1.RHMIStatus{
			Stages: map[integreatlyv1alpha1.StageName]integreatlyv1alpha1.RHMIStageStatus{
				integreatlyv1alpha1.ProductsStage: {
					Name:  integreatlyv1alpha1.ProductsStage,
					Phase: integreatlyv1alpha1.PhaseInProgress,
					Products: map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.RHMIProductStatus{
						integreatlyv1alpha1.ProductGrafana: {
							Name:   integreatlyv1alpha1.ProductGrafana,
							Status: integreatlyv1alpha1.PhaseInProgress,
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
			Namespace: common.NamespacePrefix + "customer-monitoring",
		},
		Spec: routev1.RouteSpec{
			Host: "sampleHost",
		},
	}
}

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := corev1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := coreosv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := prometheusmonitoringv1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := projectv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := routev1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	return scheme, nil
}

func TestAlertCreation(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		serverClient func() k8sclient.Client
		reconciler   func() *Reconciler
		installation *integreatlyv1alpha1.RHMI
		want         integreatlyv1alpha1.StatusPhase
		wantErr      string
		wantFn       func(c k8sclient.Client) error
	}{
		{
			name: "returns expected alerts",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(scheme, getRateLimitConfigMap(), getGrafanaRoute())
			},
			reconciler: func() *Reconciler {
				return getBasicReconciler()
			},
			want: integreatlyv1alpha1.PhaseCompleted,
		},
		{
			name: "returns PhaseInProgress when grafana not installed",
			serverClient: func() k8sclient.Client {
				return fakeclient.NewFakeClientWithScheme(scheme, getRateLimitConfigMap())
			},
			reconciler: func() *Reconciler {
				return getBasicReconciler()
			},
			want: integreatlyv1alpha1.PhaseInProgress,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reconciler := tt.reconciler()
			serverClient := tt.serverClient()

			got, err := reconciler.reconcileAlerts(context.TODO(), serverClient, reconciler.installation)
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

func strPtr(str string) *string {
	return &str
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{l.ProductLogContext: integreatlyv1alpha1.ProductApicurioRegistry})
}
