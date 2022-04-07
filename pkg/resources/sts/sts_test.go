package sts

import (
	"context"
	"fmt"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	cloudcredentialv1 "github.com/openshift/api/operator/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func getBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := cloudcredentialv1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = corev1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	return scheme, err
}

func TestIsClusterSTS(t *testing.T) {
	scheme, err := getBuildScheme()
	if err != nil {
		t.Fatalf("Error obtaining scheme")
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
				client: &moqclient.SigsClientInterfaceMock{GetFunc: func(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
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
				client: fakeclient.NewFakeClientWithScheme(scheme, &cloudcredentialv1.CloudCredential{
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
				client: fakeclient.NewFakeClientWithScheme(scheme, &cloudcredentialv1.CloudCredential{
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
