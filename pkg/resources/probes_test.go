package resources

import (
	"context"
	"errors"
	"testing"

	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	clientMock "github.com/integr8ly/integreatly-operator/pkg/client"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	errFoo = errors.New("generic error")
)

func TestCreatePrometheusProbe(t *testing.T) {
	type args struct {
		ctx     context.Context
		client  k8sclient.Client
		inst    *v1alpha1.RHMI
		name    string
		module  string
		targets monv1.ProbeTargetStaticConfig
	}
	tests := []struct {
		name    string
		args    args
		want    v1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "retry if namespace field is empty",
			args: args{
				ctx: context.TODO(),
			},
			want:    v1alpha1.PhaseInProgress,
			wantErr: false,
		},
		{
			name: "retry if the URL(s) is not yet known",
			args: args{
				ctx:     context.TODO(),
				targets: monv1.ProbeTargetStaticConfig{},
			},
			want:    v1alpha1.PhaseInProgress,
			wantErr: false,
		},
		{
			name: "failed to create probe",
			args: args{
				ctx: context.TODO(),
				targets: monv1.ProbeTargetStaticConfig{
					Targets: []string{"testUrl"},
				},
				inst: &v1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rhoam",
						Namespace: "ns",
					},
				},
				client: &clientMock.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object) error {
						return nil
					},
					UpdateFunc: func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.UpdateOption) error {
						return errFoo
					},
				},
			},
			want:    v1alpha1.PhaseFailed,
			wantErr: true,
		},
		{
			name: "success creating probe",
			args: args{
				ctx: context.TODO(),
				targets: monv1.ProbeTargetStaticConfig{
					Targets: []string{"testUrl"},
				},
				inst: &v1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rhoam",
						Namespace: "ns",
					},
				},
				client: &clientMock.SigsClientInterfaceMock{
					GetFunc: func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object) error {
						return nil
					},
					UpdateFunc: func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.UpdateOption) error {
						return nil
					},
				},
			},
			want:    v1alpha1.PhaseCompleted,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreatePrometheusProbe(tt.args.ctx, tt.args.client, tt.args.inst, tt.args.name, tt.args.module, tt.args.targets)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreatePrometheusProbe() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CreatePrometheusProbe() got = %v, want %v", got, tt.want)
			}
		})
	}
}
