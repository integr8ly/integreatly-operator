package resources

import (
	"context"

	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	"testing"

	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetSMTPFromAddress(t *testing.T) {
	type args struct {
		ctx          context.Context
		serverClient k8sclient.Client
		log          logger.Logger
		installation *v1alpha1.RHMI
	}
	tests := []struct {
		name        string
		args        args
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name:        "Function pasted nil pointers",
			want:        "",
			wantErr:     true,
			errContains: "nil pointer passed",
		},
		{
			name:    "Is RHOAM Multi Tenant install",
			want:    "test@rhmw.io",
			wantErr: false,
			args: args{
				installation: &v1alpha1.RHMI{
					Spec: v1alpha1.RHMISpec{
						Type: string(v1alpha1.InstallationTypeMultitenantManagedApi),
					},
				},
			},
		},
		{
			name:    "Has custom STMP configured",
			want:    "custom@smtp.com",
			wantErr: false,
			args: args{
				ctx: context.TODO(),
				serverClient: fakeclient.NewClientBuilder().WithRuntimeObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "custom-smtp",
							Namespace: "testing",
						},
						Data: map[string][]byte{
							"from_address": []byte("custom@smtp.com"),
						},
					}).Build(),
				log: logger.NewLogger(),
				installation: &v1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "testing",
					},
					Status: v1alpha1.RHMIStatus{
						CustomSmtp: &v1alpha1.CustomSmtpStatus{
							Enabled: true,
						},
					}},
			},
		},
		{
			name:        "Custom STMP configured and returns errors",
			want:        "",
			wantErr:     true,
			errContains: "secrets \"custom-smtp\" not found",
			args: args{
				ctx:          context.TODO(),
				serverClient: fakeclient.NewClientBuilder().Build(),
				log:          logger.NewLogger(),
				installation: &v1alpha1.RHMI{Status: v1alpha1.RHMIStatus{
					CustomSmtp: &v1alpha1.CustomSmtpStatus{
						Enabled: true,
					},
				}},
			},
		},
		{
			name:    "From address taken from installation spec",
			want:    "good@smtp.com",
			wantErr: false,
			args: args{
				ctx: context.TODO(),
				serverClient: fakeclient.NewClientBuilder().WithRuntimeObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      alertManagerConfigSecretName,
						Namespace: "testing",
					},
					Data: map[string][]byte{
						"alertmanager.yaml": []byte("global:\n  smtp_from: good@smtp.com"),
					},
				}).Build(),
				log:          logger.NewLogger(),
				installation: &v1alpha1.RHMI{Spec: v1alpha1.RHMISpec{AlertFromAddress: "good@smtp.com"}},
			},
		},
		{
			name:    "envar used for From Address",
			want:    "envar@smtp.com",
			wantErr: false,
			args: args{
				ctx:          context.TODO(),
				serverClient: fakeclient.NewClientBuilder().Build(),
				log:          logger.NewLogger(),
				installation: &v1alpha1.RHMI{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ALERT_SMTP_FROM", "envar@smtp.com")
			got, err := GetSMTPFromAddress(tt.args.ctx, tt.args.serverClient, tt.args.log, tt.args.installation)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSMTPFromAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetSMTPFromAddress() got = %v, want %v", got, tt.want)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("GetSMTPFromAddress()\nerror message = %v\nshould contain = %v", err, tt.errContains)
			}
		})
	}
}
