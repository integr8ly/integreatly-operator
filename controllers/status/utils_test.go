package status

import (
	"context"
	"fmt"
	"testing"

	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/utils"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestIsAddonOperatorInstalled(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		client client.Client
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "test true if AddonInstance CRD is installed",
			args: args{
				client: utils.NewTestClient(scheme, &apiextensionv1.CustomResourceDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: addonInstanceCRDName,
					},
				}),
			},
			want: true,
		},
		{
			name: "test false if AddonInstance CRD is not installed",
			args: args{
				client: utils.NewTestClient(scheme),
			},
			want: false,
		},
		{
			name: "test false if error getting AddonInstance CRD is not installed",
			args: args{
				client: &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
						return fmt.Errorf("error")
					},
				},
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsAddonOperatorInstalled(tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsAddonOperatorInstalled() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsAddonOperatorInstalled() got = %v, want %v", got, tt.want)
			}
		})
	}
}
