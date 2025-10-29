package resources

import (
	"context"
	"testing"

	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/utils"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	rhoamNS = "redhat-rhoam-operator"
)

func TestInstallationState(t *testing.T) {
	type testScenario struct {
		Name  string
		Input struct {
			Version   string
			ToVersion string
		}
		Expected string
	}

	scenarios := []testScenario{
		{
			Name: "No version information is set",
			Input: struct {
				Version   string
				ToVersion string
			}{Version: "", ToVersion: ""},
			Expected: "Unknown State",
		},
		{
			Name: "Initial installation",
			Input: struct {
				Version   string
				ToVersion string
			}{Version: "", ToVersion: "1.1.0"},
			Expected: "Installing",
		},
		{
			Name: "Upgrade installation",
			Input: struct {
				Version   string
				ToVersion string
			}{Version: "1.1.0", ToVersion: "1.2.0"},
			Expected: "Upgrading",
		},
		{
			Name: "Installed state",
			Input: struct {
				Version   string
				ToVersion string
			}{Version: "1.1.0", ToVersion: ""},
			Expected: "Installed",
		},
	}

	for _, scenario := range scenarios {
		actual := InstallationState(scenario.Input.Version, scenario.Input.ToVersion)

		if actual != scenario.Expected {
			t.Fatalf("Test: %s; Status not equal to expected result, Expected: %s, Actual: %s", scenario.Name, scenario.Expected, actual)
		}
	}
}

func TestCreateAddonManagedApiServiceParametersExists(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx    context.Context
		client client.Client
		cr     *v1alpha1.RHMI
	}
	tests := []struct {
		name    string
		args    args
		want    v1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "test create AddonManagedApiServiceParametersExists alert successful",
			args: args{
				ctx: context.TODO(),
				client: utils.NewTestClient(scheme, &monitoringv1.PrometheusRule{
					ObjectMeta: metav1.ObjectMeta{
						Name: "AddonManagedApiServiceParameters",
					},
				}, &corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      "addon-test-parameters",
						Namespace: defaultOperatorNamespace,
					},
				}, &operatorsv1alpha1.SubscriptionList{
					Items: []operatorsv1alpha1.Subscription{
						{
							ObjectMeta: v1.ObjectMeta{
								Name:      "test",
								Namespace: defaultOperatorNamespace,
							},
						},
					},
				}, getNamespaces()),
				cr: getRHMIcr(),
			},
			want:    v1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "test create AddonManagedApiServiceParametersExists alert failure",
			args: args{
				ctx: context.TODO(),
				client: utils.NewTestClient(scheme, &corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      "addon-testfailed-parameters",
						Namespace: defaultOperatorNamespace,
					},
				}, &operatorsv1alpha1.SubscriptionList{
					Items: []operatorsv1alpha1.Subscription{
						{
							ObjectMeta: v1.ObjectMeta{
								Name:      "test",
								Namespace: defaultOperatorNamespace,
							},
						},
					},
				}, getNamespaces()),
				cr: getRHMIcr(),
			},
			want:    v1alpha1.PhaseFailed,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateAddonManagedApiServiceParametersExists(tt.args.ctx, tt.args.client, tt.args.cr)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateAddonManagedApiServiceParametersExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CreateAddonManagedApiServiceParametersExists() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateDeadMansSnitchSecretExists(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx    context.Context
		client client.Client
		cr     *v1alpha1.RHMI
	}
	tests := []struct {
		name    string
		args    args
		want    v1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "test create CreateDeadMansSnitchSecretExists alert successful",
			args: args{
				ctx: context.TODO(),
				client: utils.NewTestClient(scheme, &monitoringv1.PrometheusRule{
					ObjectMeta: metav1.ObjectMeta{
						Name: "DeadMansSnitchSecretExists",
					},
				}, getNamespaces()),
				cr: getRHMIcr(),
			},
			want:    v1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "test create CreateDeadMansSnitchSecretExists alert failure",
			args: args{
				ctx:    context.TODO(),
				client: utils.NewTestClient(runtime.NewScheme()),
				cr:     getRHMIcr(),
			},
			want:    v1alpha1.PhaseFailed,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateDeadMansSnitchSecretExists(tt.args.ctx, tt.args.client, tt.args.cr)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateDeadMansSnitchSecretExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CreateDeadMansSnitchSecretExists() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateSmtpSecretExists(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx    context.Context
		client client.Client
		cr     *v1alpha1.RHMI
	}
	tests := []struct {
		name    string
		args    args
		want    v1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "test create CreateSmtpSecretExists alert successful",
			args: args{
				ctx: context.TODO(),
				client: utils.NewTestClient(scheme, &monitoringv1.PrometheusRule{
					ObjectMeta: metav1.ObjectMeta{
						Name: "CreateSmtpSecretExists",
					},
				}, getNamespaces()),
				cr: getRHMIcr(),
			},
			want:    v1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "test create CreateSmtpSecretExists alert failure",
			args: args{
				ctx:    context.TODO(),
				client: utils.NewTestClient(runtime.NewScheme()),
				cr:     getRHMIcr(),
			},
			want:    v1alpha1.PhaseFailed,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateSmtpSecretExists(tt.args.ctx, tt.args.client, tt.args.cr)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSmtpSecretExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CreateSmtpSecretExists() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func getNamespaces() *corev1.NamespaceList {
	return &corev1.NamespaceList{
		TypeMeta: v1.TypeMeta{},
		ListMeta: v1.ListMeta{},
		Items: []corev1.Namespace{
			{
				ObjectMeta: v1.ObjectMeta{
					Name: rhoamNS,
					Labels: map[string]string{
						"integreatly": "true",
					},
				},
			},
		},
	}
}

func getRHMIcr() *v1alpha1.RHMI {
	return &v1alpha1.RHMI{
		ObjectMeta: controllerruntime.ObjectMeta{
			Name:      "test",
			Namespace: defaultOperatorNamespace,
		},
	}
}
