package controllers

import (
	"context"
	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"reflect"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"testing"
)

func TestRHMIReconciler_getAlertingNamespace(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.SchemeBuilder.AddToScheme(scheme)

	type fields struct {
		Client                     client.Client
		Scheme                     *runtime.Scheme
		mgr                        controllerruntime.Manager
		controller                 controller.Controller
		restConfig                 *rest.Config
		customInformers            map[string]map[string]*cache.Informer
		productsInstallationLoader marketplace.ProductsInstallationLoader
	}
	type args struct {
		installation  *rhmiv1alpha1.RHMI
		configManager *config.Manager
	}

	resourceName := "test"

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string]string
		wantErr bool
	}{
		{
			name: "Test - RHOAM - openshift-monitoring and Observability is returned",
			args: args{
				installation:  &rhmiv1alpha1.RHMI{Spec: rhmiv1alpha1.RHMISpec{Type: string(rhmiv1alpha1.InstallationTypeManagedApi)}},
				configManager: &config.Manager{},
			},
			fields: fields{Client: fakeclient.NewFakeClientWithScheme(scheme, &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: resourceName},
				Data: map[string]string{
					"observability": "NAMESPACE: redhat-rhoam-observability",
				},
			})},
			want: map[string]string{
				"openshift-monitoring":       "alertmanager-main",
				"redhat-rhoam-observability": "alertmanager",
			},
		},
		{
			name: "Test - RHMI / Other install types - openshift-monitoring and middleware monitoring is returned",
			args: args{
				installation:  &rhmiv1alpha1.RHMI{Spec: rhmiv1alpha1.RHMISpec{Type: string(rhmiv1alpha1.InstallationTypeManaged)}},
				configManager: &config.Manager{},
			},
			fields: fields{Client: fakeclient.NewFakeClientWithScheme(scheme, &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: resourceName},
				Data: map[string]string{
					"middleware-monitoring": "OPERATOR_NAMESPACE: redhat-rhmi-middleware-monitoring-operator",
				},
			})},
			want: map[string]string{
				"openshift-monitoring":                       "alertmanager-main",
				"redhat-rhmi-middleware-monitoring-operator": "alertmanager-route",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RHMIReconciler{
				Client:                     tt.fields.Client,
				Scheme:                     tt.fields.Scheme,
				mgr:                        tt.fields.mgr,
				controller:                 tt.fields.controller,
				restConfig:                 tt.fields.restConfig,
				customInformers:            tt.fields.customInformers,
				productsInstallationLoader: tt.fields.productsInstallationLoader,
			}

			configManager, _ := config.NewManager(context.TODO(), tt.fields.Client, resourceName, resourceName, tt.args.installation)

			got, err := r.getAlertingNamespace(tt.args.installation, configManager)
			if (err != nil) != tt.wantErr {
				t.Errorf("getAlertingNamespace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAlertingNamespace() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatAlerts(t *testing.T) {
	input := []v1.Alert{
		{
			Labels: model.LabelSet{"alertname": "dummy", "severity": "High"},
			State:  "Firing",
		},
		{
			Labels: model.LabelSet{"alertname": "dummy", "severity": "High"},
			State:  "Firing",
		},
		{
			Labels: model.LabelSet{"alertname": "dummy", "severity": "Low"},
			State:  "Firing",
		},
		{
			Labels: model.LabelSet{"alertname": "dummy", "severity": "High"},
			State:  "Pending",
		},
		{
			Labels: model.LabelSet{"alertname": "dummy", "severity": "High"},
			State:  "Pending",
		},

		{
			Labels: model.LabelSet{"alertname": "dummy two", "severity": "High"},
			State:  "Firing",
		},
		{
			Labels: model.LabelSet{"alertname": "dummy two", "severity": "High"},
			State:  "Firing",
		},
		{
			Labels: model.LabelSet{"alertname": "DeadMansSwitch", "severity": "High"},
			State:  "Firing",
		},
	}
	expected := resources.AlertMetrics{
		{
			Name:     "dummy",
			Severity: "High",
			State:    "Firing",
		}: 2,
		{
			Name:     "dummy",
			Severity: "Low",
			State:    "Firing",
		}: 1,
		{
			Name:     "dummy",
			Severity: "High",
			State:    "Pending",
		}: 2,
		{
			Name:     "dummy two",
			Severity: "High",
			State:    "Firing",
		}: 2,
	}

	actual := formatAlerts(input)

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("alert metrics not equal; Actual: %v, Expected: %v", actual, expected)
	}

}

func TestGetCrName(t *testing.T) {
	tests := []struct {
		name        string
		installType string
		want        string
	}{
		{
			name:        "get RHOAM cr name",
			installType: string(rhmiv1alpha1.InstallationTypeManagedApi),
			want:        ManagedApiInstallationName,
		},
		{
			name:        "get multitenant RHOAM cr name",
			installType: string(rhmiv1alpha1.InstallationTypeMultitenantManagedApi),
			want:        ManagedApiInstallationName,
		},
		{
			name:        "get RHMI cr name",
			installType: string(rhmiv1alpha1.InstallationTypeManaged),
			want:        DefaultInstallationName,
		},
		{
			name:        "get default cr name",
			installType: "Not a real install type",
			want:        DefaultInstallationName,
		},
	}
	for _, tt := range tests {
		got := getCrName(tt.installType)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("getCrName() got = %v, want %v", got, tt.want)
		}
	}
}
