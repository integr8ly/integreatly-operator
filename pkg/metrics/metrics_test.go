package metrics

import (
	"context"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	configv1 "github.com/openshift/api/config/v1"
	userv1 "github.com/openshift/api/user/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestGetContainerCPUMetric(t *testing.T) {
	scheme := runtime.NewScheme()
	err := userv1.AddToScheme(scheme)
	err = configv1.AddToScheme(scheme)

	if err != nil {
		t.Fatalf("Error creating build scheme")
	}

	tests := []struct {
		Name           string
		FakeClient     k8sclient.Client
		ExpectedMetric string
		ExpectedError  bool
	}{
		{
			Name: "Test GetContainerCPUMetric for OpenShift < 4.9",
			FakeClient: fake.NewFakeClientWithScheme(scheme,
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
