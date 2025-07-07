package metrics

import (
	"context"
	"reflect"
	"testing"

	"github.com/integr8ly/integreatly-operator/api/v1alpha1"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	"github.com/integr8ly/integreatly-operator/utils"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetContainerCPUMetric(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		Name           string
		FakeClient     k8sclient.Client
		ExpectedMetric string
		ExpectedError  bool
	}{
		{
			Name: "Test GetContainerCPUMetric for OpenShift < 4.9",
			FakeClient: utils.NewTestClient(scheme,
				&configv1.ClusterVersionList{
					Items: []configv1.ClusterVersion{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "version",
							},
							Status: configv1.ClusterVersionStatus{
								History: []configv1.UpdateHistory{
									{
										Version: "4.8",
									},
								},
							},
						},
					},
				},
			),
			ExpectedMetric: "node_namespace_pod_container:container_cpu_usage_seconds_total:sum_rate",
			ExpectedError:  false,
		},
		{
			Name: "Test GetContainerCPUMetric for OpenShift > 4.9",
			FakeClient: utils.NewTestClient(scheme,
				&configv1.ClusterVersionList{
					Items: []configv1.ClusterVersion{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "version",
							},
							Status: configv1.ClusterVersionStatus{
								History: []configv1.UpdateHistory{
									{
										Version: "4.9",
									},
								},
							},
						},
					},
				},
			),
			ExpectedMetric: "node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate",
			ExpectedError:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			metric, err := GetContainerCPUMetric(context.TODO(), tt.FakeClient, l.NewLogger())

			if err != nil {
				t.Fatalf("Failed test with: %v", err)
			}

			if metric != tt.ExpectedMetric {
				t.Fatalf("incorrect metric returned, expected %v, got %v", tt.ExpectedMetric, metric)
			}
		})
	}
}

func Test_GetStats(t *testing.T) {
	type args struct {
		cr *v1alpha1.RHMI
	}
	tests := []struct {
		name    string
		args    args
		want    RhoamState
		wantErr bool
	}{
		{
			name:    "nil passed into function",
			wantErr: true,
		},
		{
			name: "status is \"in progress\"",
			want: RhoamState{
				Status: v1alpha1.PhaseInProgress,
			},
			wantErr: false,
			args: args{cr: &v1alpha1.RHMI{
				Status: v1alpha1.RHMIStatus{
					Stage: "bootstrap",
				},
			}},
		},
		{
			name: "status is \"complete\"",
			want: RhoamState{
				Status:    v1alpha1.PhaseCompleted,
				Upgrading: false,
				Version:   "",
			},
			wantErr: false,
			args: args{cr: &v1alpha1.RHMI{
				Status: v1alpha1.RHMIStatus{
					Stage: "complete",
				},
			}},
		},
		{
			name: "status is empty", // Should never happen
			want: RhoamState{
				Status:    v1alpha1.PhaseInProgress,
				Upgrading: false,
				Version:   "",
			},
			wantErr: false,
			args: args{cr: &v1alpha1.RHMI{
				Status: v1alpha1.RHMIStatus{},
			}},
		},
		{
			name: "upgrading is true, has 'version' & 'toVersion'",
			want: RhoamState{
				Status:    v1alpha1.PhaseInProgress,
				Upgrading: true,
				Version:   "1.2.3",
			},
			wantErr: false,
			args: args{cr: &v1alpha1.RHMI{
				Status: v1alpha1.RHMIStatus{
					Version:   "1.2.3",
					ToVersion: "1.2.4",
				},
			}},
		},
		{
			name: "upgrading is false, has 'version'",
			want: RhoamState{
				Status:    v1alpha1.PhaseInProgress,
				Upgrading: false,
				Version:   "1.2.3",
			},
			wantErr: false,
			args: args{cr: &v1alpha1.RHMI{
				Status: v1alpha1.RHMIStatus{
					Version: "1.2.3",
				},
			}},
		},
		{
			name: "fresh installation",
			want: RhoamState{
				Status:    v1alpha1.PhaseInProgress,
				Upgrading: false,
				Version:   "1.2.3",
			},
			wantErr: false,
			args: args{cr: &v1alpha1.RHMI{
				Status: v1alpha1.RHMIStatus{
					ToVersion: "1.2.3",
				},
			}},
		},
		{
			name: "version is 1.2.3",
			want: RhoamState{
				Status:    v1alpha1.PhaseInProgress,
				Upgrading: false,
				Version:   "1.2.3",
			},
			wantErr: false,
			args: args{cr: &v1alpha1.RHMI{
				Status: v1alpha1.RHMIStatus{
					Version: "1.2.3",
				},
			}},
		},
		{
			name: "version is empty",
			want: RhoamState{
				Status:    v1alpha1.PhaseInProgress,
				Upgrading: false,
				Version:   "",
			},
			wantErr: false,
			args: args{cr: &v1alpha1.RHMI{
				Status: v1alpha1.RHMIStatus{
					Version: "",
				},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetRhoamState(tt.args.cr)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRhoamState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRhoamState() got = %v, want %v", got, tt.want)
			}
		})
	}
}
