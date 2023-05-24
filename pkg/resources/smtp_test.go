package resources

import (
	"context"

	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/utils"

	"testing"

	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetExistingSMTPFromAddress(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []struct {
		Name       string
		FakeClient k8sclient.Client
		WantRes    string
		WantErr    bool
	}{
		{
			Name: "successfully retrieve existing smtp from address",
			FakeClient: utils.NewTestClient(scheme, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      alertManagerConfigSecretName,
					Namespace: "test",
				},
				Data: map[string][]byte{
					"alertmanager.yaml": []byte("global:\n  smtp_from: noreply-alert@devshift.org"),
				},
			}),
			WantRes: "noreply-alert@devshift.org",
			WantErr: false,
		},
		{
			Name:       "failed to retrieve alert manager config secret",
			FakeClient: utils.NewTestClient(scheme),
			WantRes:    "",
			WantErr:    true,
		},
		{
			Name: "failed to find alertmanager.yaml in alertmanager-application-monitoring secret data",
			FakeClient: utils.NewTestClient(scheme, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      alertManagerConfigSecretName,
					Namespace: "test",
				},
				Data: map[string][]byte{
					"fake": []byte("fake:\n test: yes"),
				},
			}),
			WantRes: "",
			WantErr: true,
		},
		{
			Name: "failed to find smtp_from in alert manager config map",
			FakeClient: utils.NewTestClient(scheme, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      alertManagerConfigSecretName,
					Namespace: "test",
				},
				Data: map[string][]byte{
					"alertmanager.yaml": []byte("global:"),
				},
			}),
			WantRes: "",
			WantErr: true,
		},
		{
			Name: "failed to unmarshal yaml from secret data",
			FakeClient: utils.NewTestClient(scheme, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      alertManagerConfigSecretName,
					Namespace: "test",
				},
				Data: map[string][]byte{
					"alertmanager.yaml": []byte("invalid yaml"),
				},
			}),
			WantRes: "",
			WantErr: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			smtpFrom, err := GetExistingSMTPFromAddress(context.TODO(), scenario.FakeClient, "test")
			if !scenario.WantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if scenario.WantRes != smtpFrom {
				t.Fatalf("unexpected result from GetExistingSMTPFromAddress(): got %s, want %s", smtpFrom, scenario.WantRes)
			}
		})
	}
}

func TestGetSMTPFromAddress(t *testing.T) {
	type args struct {
		ctx          context.Context
		serverClient k8sclient.Client
		log          logger.Logger
		installation *v1alpha1.RHMI
		namespace    string
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
				namespace: "testing",
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
				namespace: "testing",
			},
		},
		{
			name:    "From address taken from alertmanager.yaml",
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
				log: logger.NewLogger(),
				installation: &v1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "testing",
					},
				},
				namespace: "testing",
			},
		},
		{
			name:        "Failure to find the alertmanager configuration",
			want:        "",
			wantErr:     true,
			errContains: "cannot unmarshal !!str",
			args: args{
				ctx: context.TODO(),
				serverClient: fakeclient.NewClientBuilder().WithRuntimeObjects(&corev1.Secret{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      alertManagerConfigSecretName,
						Namespace: "testing",
					},
					Data: map[string][]byte{
						"alertmanager.yaml": []byte("|\nglobal: foo"),
					},
				}).Build(),
				log: logger.NewLogger(),
				installation: &v1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "testing",
					},
				},
				namespace: "testing",
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
				installation: &v1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "testing",
					},
				},
				namespace: "",
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
