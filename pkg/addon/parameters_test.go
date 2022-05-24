package addon

import (
	"context"
	clientMock "github.com/integr8ly/integreatly-operator/pkg/client"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetParameter(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)

	scenarios := []struct {
		Name               string
		ExistingParameters map[string][]byte
		Parameter          string
		ExpectedFound      bool
		ExpectedValue      []byte
		Client             client.Client
		WantErr            bool
	}{
		{
			Name: "Parameter found",
			ExistingParameters: map[string][]byte{
				"test": []byte("foo"),
			},
			ExpectedFound: true,
			ExpectedValue: []byte("foo"),
			Parameter:     "test",
			Client: fake.NewFakeClientWithScheme(scheme,
				&integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Name:      "managed-api",
						Namespace: "redhat-test-operator",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: string(integreatlyv1alpha1.InstallationTypeManagedApi),
					},
				},
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      "addon-managed-api-service-parameters",
						Namespace: "redhat-test-operator",
					},
					Data: map[string][]byte{
						"test": []byte("foo"),
					},
				}),
		},
		{
			Name: "Parameter not found: not in secret",
			ExistingParameters: map[string][]byte{
				"test": []byte("foo"),
			},
			ExpectedFound: false,
			Parameter:     "bar",
			Client: fake.NewFakeClientWithScheme(scheme,
				&integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Name:      "managed-api",
						Namespace: "redhat-test-operator",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: string(integreatlyv1alpha1.InstallationTypeManagedApi),
					},
				},
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      "addon-managed-api-service-parameters",
						Namespace: "redhat-test-operator",
					},
					Data: map[string][]byte{
						"test": []byte("foo"),
					},
				}),
		},
		{
			Name:               "Parameter not found: secret not defined",
			ExistingParameters: nil,
			ExpectedFound:      false,
			Parameter:          "test",
			Client: fake.NewFakeClientWithScheme(scheme, &integreatlyv1alpha1.RHMI{
				ObjectMeta: v1.ObjectMeta{
					Name:      "managed-api",
					Namespace: "redhat-test-operator",
				},
				Spec: integreatlyv1alpha1.RHMISpec{
					Type: string(integreatlyv1alpha1.InstallationTypeManagedApi),
				},
			}),
		},
		{
			Name:               "Error retrieving RHMI CR",
			ExistingParameters: nil,
			ExpectedFound:      false,
			Parameter:          "test",
			Client: &clientMock.SigsClientInterfaceMock{
				ListFunc: func(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
					return genericError
				},
			},
			WantErr: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			result, ok, err := GetParameter(context.TODO(), scenario.Client, "redhat-test-operator", scenario.Parameter)
			if (err != nil) != scenario.WantErr {
				t.Fatalf("GetParameter() error = %v, wantErr %v", err, scenario.WantErr)
			}
			if ok != scenario.ExpectedFound {
				t.Fatalf("GetParameter() ok = %v, ExpectedFound %v", ok, scenario.ExpectedFound)
			}
			if string(result) != string(scenario.ExpectedValue) {
				t.Fatalf("GetParameter() result = %v, ExpectedValue %v", result, scenario.ExpectedValue)
			}
		})
	}
}

func TestGetStringParameterByInstallType(t *testing.T) {
	scheme := runtime.NewScheme()
	err := corev1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		ctx              context.Context
		client           client.Client
		installationType integreatlyv1alpha1.InstallationType
		namespace        string
		parameter        string
	}
	tests := []struct {
		name      string
		args      args
		wantValue string
		wantOk    bool
		wantErr   bool
	}{
		{
			name: "retrieve the string value for an addon parameter given the installation type",
			args: args{
				ctx: context.TODO(),
				client: fake.NewFakeClientWithScheme(scheme, &corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      "addon-managed-api-service-parameters",
						Namespace: "ns",
					},
					Data: map[string][]byte{
						"parameter": []byte("param value"),
					},
				}),
				installationType: integreatlyv1alpha1.InstallationTypeManagedApi,
				namespace:        "ns",
				parameter:        "parameter",
			},
			wantValue: "param value",
			wantOk:    true,
			wantErr:   false,
		},
		{
			name: "failed to retrieve secret",
			args: args{
				ctx: context.TODO(),
				client: &clientMock.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
						return genericError
					},
				},
			},
			wantValue: "",
			wantOk:    false,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, ok, err := GetStringParameterByInstallType(tt.args.ctx, tt.args.client, tt.args.installationType, tt.args.namespace, tt.args.parameter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStringParameterByInstallType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if value != tt.wantValue {
				t.Errorf("GetStringParameterByInstallType() value = %v, wantValue %v", value, tt.wantValue)
			}
			if ok != tt.wantOk {
				t.Errorf("GetStringParameterByInstallType() ok = %v, wantOk %v", ok, tt.wantOk)
			}
		})
	}
}

func TestGetStringParameter(t *testing.T) {
	scheme := runtime.NewScheme()
	err := integreatlyv1alpha1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}
	err = corev1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		ctx       context.Context
		client    client.Client
		namespace string
		parameter string
	}
	tests := []struct {
		name      string
		args      args
		wantValue string
		wantOk    bool
		wantErr   bool
	}{
		{
			name: "retrieve the string value for an addon parameter",
			args: args{
				ctx: context.TODO(),
				client: fake.NewFakeClientWithScheme(scheme,
					&integreatlyv1alpha1.RHMI{
						ObjectMeta: v1.ObjectMeta{
							Name:      "managed-api",
							Namespace: "ns",
						},
						Spec: integreatlyv1alpha1.RHMISpec{
							Type: string(integreatlyv1alpha1.InstallationTypeManagedApi),
						},
					},
					&corev1.Secret{
						ObjectMeta: v1.ObjectMeta{
							Name:      "addon-managed-api-service-parameters",
							Namespace: "ns",
						},
						Data: map[string][]byte{
							"parameter": []byte("param value"),
						},
					}),
				namespace: "ns",
				parameter: "parameter",
			},
			wantValue: "param value",
			wantOk:    true,
			wantErr:   false,
		},
		{
			name: "failed to retrieve RHMI CR",
			args: args{
				ctx: context.TODO(),
				client: &clientMock.SigsClientInterfaceMock{
					ListFunc: func(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
						return genericError
					},
				},
			},
			wantValue: "",
			wantOk:    false,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, ok, err := GetStringParameter(tt.args.ctx, tt.args.client, tt.args.namespace, tt.args.parameter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStringParameter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if value != tt.wantValue {
				t.Errorf("GetStringParameter() value = %v, wantValue %v", value, tt.wantValue)
			}
			if ok != tt.wantOk {
				t.Errorf("GetStringParameter() ok = %v, wantOk %v", ok, tt.wantOk)
			}
		})
	}
}

func TestGetIntParameter(t *testing.T) {
	scheme := runtime.NewScheme()
	err := integreatlyv1alpha1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}
	err = corev1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		ctx       context.Context
		client    client.Client
		namespace string
		parameter string
	}
	tests := []struct {
		name      string
		args      args
		wantValue int
		wantOk    bool
		wantErr   bool
	}{
		{
			name: "retrieve the integer value for an addon parameter",
			args: args{
				ctx: context.TODO(),
				client: fake.NewFakeClientWithScheme(scheme,
					&integreatlyv1alpha1.RHMI{
						ObjectMeta: v1.ObjectMeta{
							Name:      "managed-api",
							Namespace: "ns",
						},
						Spec: integreatlyv1alpha1.RHMISpec{
							Type: string(integreatlyv1alpha1.InstallationTypeManagedApi),
						},
					},
					&corev1.Secret{
						ObjectMeta: v1.ObjectMeta{
							Name:      "addon-managed-api-service-parameters",
							Namespace: "ns",
						},
						Data: map[string][]byte{
							"parameter": []byte("666"),
						},
					}),
				namespace: "ns",
				parameter: "parameter",
			},
			wantValue: 666,
			wantOk:    true,
			wantErr:   false,
		},
		{
			name: "failed to retrieve RHMI CR",
			args: args{
				ctx: context.TODO(),
				client: &clientMock.SigsClientInterfaceMock{
					ListFunc: func(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
						return genericError
					},
				},
			},
			wantValue: 0,
			wantOk:    false,
			wantErr:   true,
		},
		{
			name: "failed to parse string to integer",
			args: args{
				ctx: context.TODO(),
				client: fake.NewFakeClientWithScheme(scheme,
					&integreatlyv1alpha1.RHMI{
						ObjectMeta: v1.ObjectMeta{
							Name:      "managed-api",
							Namespace: "ns",
						},
						Spec: integreatlyv1alpha1.RHMISpec{
							Type: string(integreatlyv1alpha1.InstallationTypeManagedApi),
						},
					},
					&corev1.Secret{
						ObjectMeta: v1.ObjectMeta{
							Name:      "addon-managed-api-service-parameters",
							Namespace: "ns",
						},
						Data: map[string][]byte{
							"parameter": []byte("param value"),
						},
					}),
				namespace: "ns",
				parameter: "parameter",
			},
			wantValue: 0,
			wantOk:    true,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, ok, err := GetIntParameter(tt.args.ctx, tt.args.client, tt.args.namespace, tt.args.parameter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetIntParameter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if value != tt.wantValue {
				t.Errorf("GetIntParameter() value = %v, wantValue %v", value, tt.wantValue)
			}
			if ok != tt.wantOk {
				t.Errorf("GetIntParameter() ok = %v, wantOk %v", ok, tt.wantOk)
			}
		})
	}
}

func TestGetBoolParameter(t *testing.T) {
	scheme := runtime.NewScheme()
	err := integreatlyv1alpha1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}
	err = corev1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		ctx       context.Context
		client    client.Client
		namespace string
		parameter string
	}
	tests := []struct {
		name      string
		args      args
		wantValue bool
		wantOk    bool
		wantErr   bool
	}{
		{
			name: "retrieve the boolean value for an addon parameter",
			args: args{
				ctx: context.TODO(),
				client: fake.NewFakeClientWithScheme(scheme,
					&integreatlyv1alpha1.RHMI{
						ObjectMeta: v1.ObjectMeta{
							Name:      "managed-api",
							Namespace: "ns",
						},
						Spec: integreatlyv1alpha1.RHMISpec{
							Type: string(integreatlyv1alpha1.InstallationTypeManagedApi),
						},
					},
					&corev1.Secret{
						ObjectMeta: v1.ObjectMeta{
							Name:      "addon-managed-api-service-parameters",
							Namespace: "ns",
						},
						Data: map[string][]byte{
							"parameter": []byte("true"),
						},
					}),
				namespace: "ns",
				parameter: "parameter",
			},
			wantValue: true,
			wantOk:    true,
			wantErr:   false,
		},
		{
			name: "failed to retrieve RHMI CR",
			args: args{
				ctx: context.TODO(),
				client: &clientMock.SigsClientInterfaceMock{
					ListFunc: func(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
						return genericError
					},
				},
			},
			wantValue: false,
			wantOk:    false,
			wantErr:   true,
		},
		{
			name: "failed to parse string to boolean",
			args: args{
				ctx: context.TODO(),
				client: fake.NewFakeClientWithScheme(scheme,
					&integreatlyv1alpha1.RHMI{
						ObjectMeta: v1.ObjectMeta{
							Name:      "managed-api",
							Namespace: "ns",
						},
						Spec: integreatlyv1alpha1.RHMISpec{
							Type: string(integreatlyv1alpha1.InstallationTypeManagedApi),
						},
					},
					&corev1.Secret{
						ObjectMeta: v1.ObjectMeta{
							Name:      "addon-managed-api-service-parameters",
							Namespace: "ns",
						},
						Data: map[string][]byte{
							"parameter": []byte("param value"),
						},
					}),
				namespace: "ns",
				parameter: "parameter",
			},
			wantValue: false,
			wantOk:    true,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, ok, err := GetBoolParameter(tt.args.ctx, tt.args.client, tt.args.namespace, tt.args.parameter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBoolParameter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if value != tt.wantValue {
				t.Errorf("GetBoolParameter() value = %v, wantValue %v", value, tt.wantValue)
			}
			if ok != tt.wantOk {
				t.Errorf("GetBoolParameter() ok = %v, wantOk %v", ok, tt.wantOk)
			}
		})
	}
}

func TestExistsParameterByInstallation(t *testing.T) {
	scheme := runtime.NewScheme()
	err := integreatlyv1alpha1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}
	err = corev1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		ctx       context.Context
		client    client.Client
		install   *integreatlyv1alpha1.RHMI
		parameter string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "parameter exists",
			args: args{
				ctx: context.TODO(),
				client: fake.NewFakeClientWithScheme(scheme,
					&integreatlyv1alpha1.RHMI{
						ObjectMeta: v1.ObjectMeta{
							Name:      "managed-api",
							Namespace: "ns",
						},
						Spec: integreatlyv1alpha1.RHMISpec{
							Type: string(integreatlyv1alpha1.InstallationTypeManagedApi),
						},
					},
					&corev1.Secret{
						ObjectMeta: v1.ObjectMeta{
							Name:      "addon-managed-api-service-parameters",
							Namespace: "ns",
						},
						Data: map[string][]byte{
							"parameter": []byte("param value"),
						},
					}),
				install: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Name:      "managed-api",
						Namespace: "ns",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: string(integreatlyv1alpha1.InstallationTypeManagedApi),
					},
				},
				parameter: "parameter",
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := ExistsParameterByInstallation(tt.args.ctx, tt.args.client, tt.args.install, tt.args.parameter)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExistsParameterByInstallation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if found != tt.want {
				t.Errorf("ExistsParameterByInstallation() found = %v, want %v", found, tt.want)
			}
		})
	}
}
