package controllers

import (
	"context"
	"reflect"
	"testing"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	routev1 "github.com/openshift/api/route/v1"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

const (
	FakeName      = "fake-name"
	FakeNamespace = "fake-namespace"
	FakeHost      = "fake-route.org"
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

func buildAddonSecret(namespace string, secretData map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "addon-managed-api-service-parameters",
			Namespace: namespace,
		},
		Data: secretData,
	}
}

func TestFormatAlerts(t *testing.T) {
	input := []prometheusv1.Alert{
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

func TestHandleCROConfigDeletion(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name         string
		installation rhmiv1alpha1.RHMI
		wantErr      bool
	}{
		{
			name: "handle CRO config map deletion when config map exists",
			installation: rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FakeName,
					Namespace: FakeNamespace,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		r := &RHMIReconciler{
			Client: fakeclient.NewFakeClientWithScheme(scheme, getCROConfigMap()),
		}
		err := r.handleCROConfigDeletion(tt.installation)
		if (err != nil) != tt.wantErr {
			t.Errorf("handleCROConfigDeletion() error = %v, wantErr %v", err, tt.wantErr)
			return
		}
	}
}

func TestFirstInstallFirstReconcile(t *testing.T) {
	tests := []struct {
		name         string
		installation *rhmiv1alpha1.RHMI
		want         bool
	}{
		{
			name: "test CR for first install, first reconcile",
			installation: &rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FakeName,
					Namespace: FakeNamespace,
				},
				Status: rhmiv1alpha1.RHMIStatus{
					Version:   "",
					ToVersion: "",
				},
			},
			want: true,
		},
		{
			name: "test CR for first install, installation complete",
			installation: &rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FakeName,
					Namespace: FakeNamespace,
				},
				Status: rhmiv1alpha1.RHMIStatus{
					Version:   "x.y.z",
					ToVersion: "",
				},
			},
			want: false,
		},
		{
			name: "test CR for first install, installation in progress",
			installation: &rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FakeName,
					Namespace: FakeNamespace,
				},
				Status: rhmiv1alpha1.RHMIStatus{
					Version:   "",
					ToVersion: "x.y.z",
				},
			},
			want: false,
		},
		{
			name: "test CR for installation complete, upgrade in progress",
			installation: &rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FakeName,
					Namespace: FakeNamespace,
				},
				Status: rhmiv1alpha1.RHMIStatus{
					Version:   "x.y.z",
					ToVersion: "x.y.z",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		got := firstInstallFirstReconcile(tt.installation)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("firstInstallFirstReconcile() got = %v, want %v", got, tt.want)
		}
	}
}

func TestUpgradeFirstReconcile(t *testing.T) {
	tests := []struct {
		name         string
		installation *rhmiv1alpha1.RHMI
		want         bool
	}{
		{
			name: "test CR for first install, first reconcile",
			installation: &rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FakeName,
					Namespace: FakeNamespace,
				},
				Spec: rhmiv1alpha1.RHMISpec{
					Type: string(rhmiv1alpha1.InstallationTypeManagedApi),
				},
				Status: rhmiv1alpha1.RHMIStatus{
					Version:   "",
					ToVersion: "",
				},
			},
			want: false,
		},
		{
			name: "test CR for first install, installation complete",
			installation: &rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FakeName,
					Namespace: FakeNamespace,
				},
				Spec: rhmiv1alpha1.RHMISpec{
					Type: string(rhmiv1alpha1.InstallationTypeManagedApi),
				},
				Status: rhmiv1alpha1.RHMIStatus{
					Version:   "x.y.z",
					ToVersion: "",
				},
			},
			want: true,
		},
		{
			name: "test CR for first install, installation in progress",
			installation: &rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FakeName,
					Namespace: FakeNamespace,
				},
				Spec: rhmiv1alpha1.RHMISpec{
					Type: string(rhmiv1alpha1.InstallationTypeManagedApi),
				},
				Status: rhmiv1alpha1.RHMIStatus{
					Version:   "",
					ToVersion: "x.y.z",
				},
			},
			want: false,
		},
		{
			name: "test CR for installation complete, upgrade in progress",
			installation: &rhmiv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FakeName,
					Namespace: FakeNamespace,
				},
				Spec: rhmiv1alpha1.RHMISpec{
					Type: string(rhmiv1alpha1.InstallationTypeManagedApi),
				},
				Status: rhmiv1alpha1.RHMIStatus{
					Version:   "x.y.z",
					ToVersion: "x.y.z",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		got := upgradeFirstReconcile(tt.installation)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("upgradeFirstReconcile() got = %v, want %v", got, tt.want)
		}
	}
}

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := corev1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := routev1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	return scheme, nil
}

func getCROConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultCloudResourceConfigName,
			Namespace: FakeNamespace,
			Finalizers: []string{
				deletionFinalizer,
			},
		},
	}
}

func Test_checkStsCredentialsPresent(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.SchemeBuilder.AddToScheme(scheme)

	type args struct {
		client    client.Client
		namespace string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "Test sts-credentials exist",
			args: args{
				client:    fakeclient.NewFakeClientWithScheme(scheme, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "sts-credentials", Namespace: "test"}}),
				namespace: "test",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Test sts-credentials do not exist",
			args: args{
				client:    fakeclient.NewFakeClientWithScheme(scheme, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "sts-credentials", Namespace: "other"}}),
				namespace: "test",
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkStsCredentialsPresent(tt.args.client, tt.args.namespace)
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
