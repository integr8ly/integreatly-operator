package sts

import (
	"context"
	"fmt"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/utils"
	cloudcredentialv1 "github.com/openshift/api/operator/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestIsClusterSTS(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx    context.Context
		client client.Client
		log    logger.Logger
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "failed to get cluster cloud credential",
			args: args{
				ctx: context.TODO(),
				client: &moqclient.SigsClientInterfaceMock{GetFunc: func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
					return fmt.Errorf("get error")
				}},
				log: logger.NewLogger(),
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "STS cluster",
			args: args{
				ctx: context.TODO(),
				client: utils.NewTestClient(scheme, &cloudcredentialv1.CloudCredential{
					ObjectMeta: metav1.ObjectMeta{
						Name: ClusterCloudCredentialName,
					},
					Spec: cloudcredentialv1.CloudCredentialSpec{
						CredentialsMode: cloudcredentialv1.CloudCredentialsModeManual,
					},
				}),
				log: logger.NewLogger(),
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Non STS cluster",
			args: args{
				ctx: context.TODO(),
				client: utils.NewTestClient(scheme, &cloudcredentialv1.CloudCredential{
					ObjectMeta: metav1.ObjectMeta{
						Name: ClusterCloudCredentialName,
					},
					Spec: cloudcredentialv1.CloudCredentialSpec{
						CredentialsMode: cloudcredentialv1.CloudCredentialsModeDefault,
					},
				}),
				log: logger.NewLogger(),
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsClusterSTS(tt.args.ctx, tt.args.client, tt.args.log)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsClusterSTS() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsClusterSTS() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ValidateAddOnStsRoleArnParameterPattern(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	const namespace = "test"

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
			name: "test: can't get secret",
			args: args{
				client: &moqclient.SigsClientInterfaceMock{
					ListFunc: func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
						return fmt.Errorf("listError")
					},
				},
				namespace: namespace,
			},
			wantErr: true,
			want:    false,
		},
		{
			name: "test: role arn not found",
			args: args{
				client:    utils.NewTestClient(scheme),
				namespace: namespace,
			},
			wantErr: true,
			want:    false,
		},
		{
			name: "test: role arn empty",
			args: args{
				client:    utils.NewTestClient(scheme, buildAddonSecret(namespace, map[string][]byte{RoleArnParameterName: []byte("")})),
				namespace: namespace,
			},
			wantErr: true,
			want:    false,
		},
		{
			name: "test: role arn regex not match",
			args: args{
				client:    utils.NewTestClient(scheme, buildAddonSecret(namespace, map[string][]byte{RoleArnParameterName: []byte("notAnARN")})),
				namespace: namespace,
			},
			wantErr: true,
			want:    false,
		},
		{
			name: "test: role arn regex match",
			args: args{
				client:    utils.NewTestClient(scheme, buildAddonSecret(namespace, map[string][]byte{RoleArnParameterName: []byte("arn:aws:iam::123456789012:role/12345")})),
				namespace: namespace,
			},
			wantErr: false,
			want:    true,
		},
		{
			name: "test: role arn regex match for AWS GovCloud (US) Regions",
			args: args{
				client:    utils.NewTestClient(scheme, buildAddonSecret(namespace, map[string][]byte{RoleArnParameterName: []byte("arn:aws-us-gov:iam::123456789012:role/12345")})),
				namespace: namespace,
			},
			wantErr: false,
			want:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateAddOnStsRoleArnParameterPattern(tt.args.client, tt.args.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAddOnStsRoleArnParameterPattern() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("validateAddOnStsRoleArnParameterPattern() got = %v, want %v", got, tt.want)
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

func Test_CreateSTSArnSecret(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx                   context.Context
		client                client.Client
		installationNamespace string
		operatorNamespace     string
	}
	tests := []struct {
		name    string
		args    args
		want    integreatlyv1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "test: phase failed on error getting role arn",
			args: args{
				client: moqclient.NewSigsClientMoqWithScheme(scheme),
			},
			want:    integreatlyv1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "test: phase complete on creating secret",
			args: args{
				client: moqclient.NewSigsClientMoqWithScheme(scheme, &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "addon-managed-api-service-parameters",
						Namespace: "redhat-rhoam-operator",
					},
					Data: map[string][]byte{
						RoleArnParameterName: []byte("arn:aws:iam::123456789012:role/12345"),
					},
				}),
				installationNamespace: "redhat-rhoam-operator",
				operatorNamespace:     "",
			},
			want: integreatlyv1alpha1.PhaseCompleted,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateSTSARNSecret(tt.args.ctx, tt.args.client, tt.args.installationNamespace, tt.args.operatorNamespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSTSARNSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("createSTSARNSecret() got = %v, want %v", got, tt.want)
			}
		})
	}
}
