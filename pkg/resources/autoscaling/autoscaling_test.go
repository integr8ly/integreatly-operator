package autoscaling

import (
	"context"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	autoscaling "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/runtime"

	corev1 "k8s.io/api/core/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	fake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	testNamespace = "test-ns"
)

var minReplicas = int32(1)

func TestReconciler_ReconcileHPA(t *testing.T) {
	scheme := runtime.NewScheme()
	autoscaling.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	lowTargetUtil := int32(20)
	highTargetUtil := int32(60)

	fakeConfigMap := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      "autoscaling-config",
			Namespace: testNamespace,
		},
		Data: map[string]string{
			"ratelimit":          "60",
			"apicast-production": "60",
			"backend-worker":     "20",
			"backend-listener":   "20",
			"usersso":            "20",
		},
	}

	tests := []struct {
		Name               string
		Installation       integreatlyv1alpha1.RHMI
		HpaTargetKind      string
		HpaTargetName      string
		HpaTargetNs        string
		MinReplicas        *int32
		MaxReplicas        int32
		ExpectedTargetUtil *int32
		ExpectedStatus     integreatlyv1alpha1.StatusPhase
		FakeClient         k8sclient.Client
		Assertion          func(k8sclient.Client, string, *int32) bool
	}{
		{
			Name: "Test that HPA is created for rate limit DC",
			Installation: integreatlyv1alpha1.RHMI{
				ObjectMeta: v1.ObjectMeta{
					Namespace: testNamespace,
				},
				Spec: integreatlyv1alpha1.RHMISpec{
					AutoscalingEnabled: true,
				},
			},
			HpaTargetKind:      "Deployment",
			HpaTargetName:      "ratelimit",
			HpaTargetNs:        testNamespace,
			MinReplicas:        &minReplicas,
			MaxReplicas:        3,
			ExpectedTargetUtil: &highTargetUtil,
			ExpectedStatus:     integreatlyv1alpha1.PhaseCompleted,
			FakeClient: fake.NewFakeClientWithScheme(
				scheme,
				fakeConfigMap,
			),
			Assertion: verifyHpaExistsWithCorrectValues,
		},
		{
			Name: "Test that HPA is deleted for rate limit DC",
			Installation: integreatlyv1alpha1.RHMI{
				ObjectMeta: v1.ObjectMeta{
					Namespace: testNamespace,
				},
				Spec: integreatlyv1alpha1.RHMISpec{
					AutoscalingEnabled: false,
				},
			},
			HpaTargetKind:      "Deployment",
			HpaTargetName:      "ratelimit",
			HpaTargetNs:        testNamespace,
			MinReplicas:        &minReplicas,
			MaxReplicas:        3,
			ExpectedTargetUtil: &highTargetUtil,
			ExpectedStatus:     integreatlyv1alpha1.PhaseCompleted,
			FakeClient: fake.NewFakeClientWithScheme(
				scheme,
				fakeConfigMap,
				retrieveExistsingHPA("ratelimit", &minReplicas, 3, &highTargetUtil),
			),
			Assertion: verifyHpaIsRemoved,
		},
		{
			Name: "Test that HPA is created for backend-listener DC",
			Installation: integreatlyv1alpha1.RHMI{
				ObjectMeta: v1.ObjectMeta{
					Namespace: testNamespace,
				},
				Spec: integreatlyv1alpha1.RHMISpec{
					AutoscalingEnabled: true,
				},
			},
			HpaTargetKind:      "Deployment",
			HpaTargetName:      "backend-listener",
			HpaTargetNs:        testNamespace,
			MinReplicas:        &minReplicas,
			MaxReplicas:        3,
			ExpectedTargetUtil: &lowTargetUtil,
			ExpectedStatus:     integreatlyv1alpha1.PhaseCompleted,
			FakeClient: fake.NewFakeClientWithScheme(
				scheme,
				fakeConfigMap,
			),
			Assertion: verifyHpaExistsWithCorrectValues,
		},
		{
			Name: "Test that HPA is deleted for backend-listener DC",
			Installation: integreatlyv1alpha1.RHMI{
				ObjectMeta: v1.ObjectMeta{
					Namespace: testNamespace,
				},
				Spec: integreatlyv1alpha1.RHMISpec{
					AutoscalingEnabled: false,
				},
			},
			HpaTargetName:      "backend-listener",
			HpaTargetNs:        testNamespace,
			MinReplicas:        &minReplicas,
			MaxReplicas:        3,
			ExpectedTargetUtil: &lowTargetUtil,
			ExpectedStatus:     integreatlyv1alpha1.PhaseCompleted,
			FakeClient: fake.NewFakeClientWithScheme(
				scheme,
				fakeConfigMap,
				retrieveExistsingHPA("backend-listener", &minReplicas, 3, &lowTargetUtil),
			),
			Assertion: verifyHpaIsRemoved,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			phase, err := ReconcileHPA(context.TODO(), tt.FakeClient, tt.Installation, tt.HpaTargetKind, tt.HpaTargetName, tt.HpaTargetNs, tt.MinReplicas, tt.MaxReplicas)
			if err != nil {
				t.Fatalf("Expected %s phase but got %s", tt.ExpectedStatus, phase)
			}

			if phase != tt.ExpectedStatus {
				t.Fatalf("Expected %s phase but got %s", tt.ExpectedStatus, phase)
			}

			hpaExists := tt.Assertion(tt.FakeClient, tt.HpaTargetName, tt.ExpectedTargetUtil)
			if !hpaExists {
				t.Fatalf("Assertion for HPA: %s failed", tt.HpaTargetName)
			}
		})
	}
}

func verifyHpaExistsWithCorrectValues(client k8sclient.Client, targetName string, expectedTargetUtil *int32) bool {
	hpa := autoscaling.HorizontalPodAutoscaler{
		ObjectMeta: v1.ObjectMeta{
			Name:      targetName,
			Namespace: testNamespace,
		},
	}

	err := client.Get(context.TODO(), k8sclient.ObjectKey{Name: targetName, Namespace: testNamespace}, &hpa)
	if err != nil {
		return false
	}

	currentTargetUtil := hpa.Spec.TargetCPUUtilizationPercentage
	if *currentTargetUtil != *expectedTargetUtil {
		return false
	}

	if hpa.Spec.MinReplicas != hpa.Spec.MinReplicas {
		return false
	}

	if hpa.Spec.MaxReplicas != 3 {
		return false
	}

	return true
}

func retrieveExistsingHPA(targetName string, minReplicas *int32, maxReplicas int32, targetUtil *int32) *autoscaling.HorizontalPodAutoscaler {
	return &autoscaling.HorizontalPodAutoscaler{
		ObjectMeta: v1.ObjectMeta{
			Name:      targetName,
			Namespace: testNamespace,
		},
		Spec: autoscaling.HorizontalPodAutoscalerSpec{
			MinReplicas: minReplicas,
			MaxReplicas: maxReplicas,
			ScaleTargetRef: autoscaling.CrossVersionObjectReference{
				Name: targetName,
			},
			TargetCPUUtilizationPercentage: targetUtil,
		},
	}
}

func verifyHpaIsRemoved(client k8sclient.Client, targetName string, expectedTargetUtil *int32) bool {
	hpa := autoscaling.HorizontalPodAutoscaler{
		ObjectMeta: v1.ObjectMeta{
			Name:      targetName,
			Namespace: testNamespace,
		},
	}

	err := client.Get(context.TODO(), k8sclient.ObjectKey{Name: targetName, Namespace: testNamespace}, &hpa)
	if err != nil {
		if k8sError.IsNotFound(err) {
			return true
		}

		return false
	}

	return true
}
