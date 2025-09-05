package resources

import (
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestIsInProw(t *testing.T) {
	type args struct {
		inst *integreatlyv1alpha1.RHMI
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "in_prow annotation is true",
			want: true,
			args: args{
				inst: &integreatlyv1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"in_prow": "true",
						},
					},
				},
			},
		},
		{
			name: "in_prow annotation is false",
			want: false,
			args: args{
				inst: &integreatlyv1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"in_prow": "false",
						},
					},
				},
			},
		},
		{
			name: "in_prow annotation doesn't exist",
			want: false,
			args: args{
				inst: &integreatlyv1alpha1.RHMI{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInProw(tt.args.inst); got != tt.want {
				t.Errorf("IsInProw() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsSkipFinalDBSnapshots(t *testing.T) {
	type args struct {
		inst *integreatlyv1alpha1.RHMI
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "in_prow skip_final_db_snapshots is true",
			want: true,
			args: args{
				inst: &integreatlyv1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"skip_final_db_snapshots": "true",
						},
					},
				},
			},
		},
		{
			name: "skip_final_db_snapshots annotation is false",
			want: false,
			args: args{
				inst: &integreatlyv1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"skip_final_db_snapshots": "false",
						},
					},
				},
			},
		},
		{
			name: "skip_final_db_snapshots annotation doesn't exist",
			want: false,
			args: args{
				inst: &integreatlyv1alpha1.RHMI{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSkipFinalDBSnapshots(tt.args.inst); got != tt.want {
				t.Errorf("IsSkipFinalDBSnapshots() = %v, want %v", got, tt.want)
			}
		})
	}
}
