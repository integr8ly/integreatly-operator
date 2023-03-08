package marketplace

import (
	"context"
	"fmt"
	"testing"

	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/test/utils"
	v1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestManager_reconcileOperatorGroup(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx                     context.Context
		serverClient            client.Client
		t                       Target
		operatorGroupNamespaces []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test operator group is created",
			args: args{
				ctx:          context.TODO(),
				serverClient: utils.NewTestClient(scheme),
				t: Target{
					Namespace:        "ns",
					SubscriptionName: "subName",
				},
				operatorGroupNamespaces: []string{"ns"},
			},
		},
		{
			name: "test operator group is updated",
			args: args{
				ctx: context.TODO(),
				serverClient: utils.NewTestClient(scheme, &v1.OperatorGroup{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      OperatorGroupName,
						Namespace: "ns",
					},
					Spec: v1.OperatorGroupSpec{
						TargetNamespaces: []string{"ns"},
					},
				}),
				t: Target{
					Namespace:        "ns",
					SubscriptionName: "subName",
				},
				operatorGroupNamespaces: []string{"updated"},
			},
		},
		{
			name: "test operator group error",
			args: args{
				ctx: context.TODO(),
				serverClient: &moqclient.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key types.NamespacedName, obj client.Object) error {
						return fmt.Errorf("error")
					},
				},
				t: Target{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{}
			if err := m.reconcileOperatorGroup(tt.args.ctx, tt.args.serverClient, tt.args.t, tt.args.operatorGroupNamespaces); (err != nil) != tt.wantErr {
				t.Errorf("createOperatorGroup() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
