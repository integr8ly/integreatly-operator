package resources

import (
	"context"
	"testing"

	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcileLimitRange(t *testing.T) {
	namespaceName := "test-namespace"
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}
	client := utils.NewTestClient(scheme, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
		},
	})

	type args struct {
		namespace string
		params    LimitRangeParams
	}
	tests := []struct {
		name    string
		args    args
		want    v1alpha1.StatusPhase
		wantErr bool
	}{
		{
			name: "success when requests and limits are set",
			args: args{
				namespace: namespaceName,
				params: LimitRangeParams{
					CpuRequest:    "5m",
					CpuLimit:      "10m",
					MemoryRequest: "5Mi",
					MemoryLimit:   "10Mi",
					ResourceType:  corev1.LimitTypeContainer,
				},
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "success when just CPU values are set",
			args: args{
				namespace: namespaceName,
				params: LimitRangeParams{
					CpuRequest:   "5m",
					CpuLimit:     "10m",
					ResourceType: corev1.LimitTypeContainer,
				},
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "success when just memory values are set",
			args: args{
				namespace: namespaceName,
				params: LimitRangeParams{
					MemoryRequest: "5Mi",
					MemoryLimit:   "10Mi",
					ResourceType:  corev1.LimitTypeContainer,
				},
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "success when just requests are set",
			args: args{
				namespace: namespaceName,
				params: LimitRangeParams{
					CpuRequest:    "5m",
					MemoryRequest: "5Mi",
					ResourceType:  corev1.LimitTypeContainer,
				},
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "success when just limits are set",
			args: args{
				namespace: namespaceName,
				params: LimitRangeParams{
					CpuLimit:     "5m",
					MemoryLimit:  "5Mi",
					ResourceType: corev1.LimitTypeContainer,
				},
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
		{
			name: "success when using DefaultLimitRangeParams",
			args: args{
				namespace: namespaceName,
				params:    DefaultLimitRangeParams,
			},
			want:    integreatlyv1alpha1.PhaseCompleted,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReconcileLimitRange(context.TODO(), client, tt.args.namespace, tt.args.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileLimitRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReconcileLimitRange() got = %v, want %v", got, tt.want)
			}
		})
	}
}
