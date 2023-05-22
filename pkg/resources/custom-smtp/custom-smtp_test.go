package custom_smtp

import (
	"context"
	"reflect"
	"testing"

	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"

	"github.com/integr8ly/integreatly-operator/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetCustomAddonValues(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		serverClient k8sclient.Client
		namespace    string
	}

	namespace := "test"
	tests := []struct {
		name    string
		args    args
		want    *CustomSmtp
		wantErr bool
	}{
		{
			name: "Happy path all values are returned",
			args: args{
				serverClient: utils.NewTestClient(scheme,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "addon-managed-api-service-parameters",
							Namespace: namespace,
						},
						Data: map[string][]byte{
							"custom-smtp-from_address": []byte("from_address"),
							"custom-smtp-address":      []byte("host_url"),
							"custom-smtp-password":     []byte("token"),
							"custom-smtp-port":         []byte("port"),
							"custom-smtp-username":     []byte("test_user_01"),
						},
					}),
				namespace: namespace,
			},
			want: &CustomSmtp{
				FromAddress: "from_address",
				Address:     "host_url",
				Password:    "token",
				Port:        "port",
				Username:    "test_user_01",
			},
			wantErr: false,
		},
		{
			name: "some values are returned",
			args: args{
				serverClient: utils.NewTestClient(scheme,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "addon-managed-api-service-parameters",
							Namespace: namespace,
						},
						Data: map[string][]byte{
							"custom-smtp-from_address": []byte("has value"),
							"custom-smtp-address":      []byte("has value"),
							"custom-smtp-password":     []byte("has value"),
						},
					}),
				namespace: namespace,
			},
			want: &CustomSmtp{
				FromAddress: "has value",
				Address:     "has value",
				Password:    "has value",
			},
			wantErr: false,
		},
		{
			name: "no values are returned",
			args: args{
				serverClient: utils.NewTestClient(scheme,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "addon-managed-api-service-parameters",
							Namespace: namespace,
						},
						Data: nil}),
				namespace: namespace,
			},
			want:    &CustomSmtp{},
			wantErr: false,
		},
		{
			name: "no secret found",
			args: args{
				serverClient: utils.NewTestClient(scheme,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-name",
							Namespace: namespace,
						},
						Data: nil}),
				namespace: namespace,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCustomAddonValues(tt.args.serverClient, tt.args.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCustomAddonValues() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCustomAddonValues() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParameterValidation(t *testing.T) {
	type args struct {
		creeds *CustomSmtp
	}
	tests := []struct {
		name string
		args args
		want ValidationResponse
	}{
		{
			name: "Confirms valid parameters",
			args: args{
				creeds: &CustomSmtp{
					FromAddress: "FromAddress address",
					Address:     "Address Name",
					Password:    "Password1",
					Port:        "567",
					Username:    "test_user",
				},
			},
			want: Valid,
		},
		{
			name: "Partial valid parameters are included",
			args: args{
				creeds: &CustomSmtp{
					FromAddress: "FromAddress address",
					Address:     "Address Name",
					Password:    "Password1",
				},
			},
			want: Partial,
		},
		{
			name: "No parameters",
			args: args{
				creeds: &CustomSmtp{},
			},
			want: Blank,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParameterValidation(tt.args.creeds); got != tt.want {
				t.Errorf("ParameterValidation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateOrUpdateCustomSMTPSecret(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx          context.Context
		serverClient k8sclient.Client
		smtp         *CustomSmtp
		namespace    string
	}

	namespace := "test"

	tests := []struct {
		name    string
		args    args
		want    v1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "Custom smtp secret is created",
			args: args{
				ctx: context.TODO(),
				serverClient: utils.NewTestClient(scheme, &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "custom-smtp",
						Namespace: namespace,
					},
					Data: map[string][]byte{
						"from":     []byte("text@example.com"),
						"host":     []byte("example.com"),
						"password": []byte("password"),
						"port":     []byte("port"),
						"username": []byte("dummy"),
					},
				}),
				smtp: &CustomSmtp{
					FromAddress: "text@example.com",
					Address:     "example.com",
					Password:    "password",
					Port:        "567",
					Username:    "dummy",
				},
				namespace: namespace,
			},
			want:    v1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "Custom smtp secret is created",
			args: args{
				ctx: context.TODO(),
				serverClient: utils.NewTestClient(scheme, &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "custom-smtp",
						Namespace: namespace,
					},
					Data: map[string][]byte{
						"from":     []byte("text@example.com"),
						"host":     []byte("example.com"),
						"password": []byte("password"),
						"port":     []byte("port"),
						"username": []byte("dummy"),
					},
				}),
				smtp:      nil,
				namespace: namespace,
			},
			want:    v1alpha1.PhaseFailed,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateOrUpdateCustomSMTPSecret(tt.args.ctx, tt.args.serverClient, tt.args.smtp, tt.args.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateOrUpdateCustomSMTPSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CreateOrUpdateCustomSMTPSecret() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeleteCustomSMTP(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx          context.Context
		serverClient k8sclient.Client
		namespace    string
	}

	namespace := "test"

	tests := []struct {
		name    string
		args    args
		want    v1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "Secret is deleted correctly",
			args: args{
				ctx: context.TODO(),
				serverClient: utils.NewTestClient(scheme, &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "custom-smtp",
						Namespace: namespace,
					}}),
				namespace: namespace,
			},
			want:    v1alpha1.PhaseCompleted,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeleteCustomSMTP(tt.args.ctx, tt.args.serverClient, tt.args.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteCustomSMTP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DeleteCustomSMTP() got = %v, want %v", got, tt.want)
			}
		})
	}
}
