package autoscaling

import (
	"context"

	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
)

func ReconcileHPA(ctx context.Context, client k8sclient.Client, deploymentName string, deploymentNS string, maxReplicas int32, minReplicas *int32) (integreatlyv1alpha1.StatusPhase, error) {
	hpa := v2beta1.HorizontalPodAutoscaler{
		ObjectMeta: v1.ObjectMeta{
			Name: deploymentName,
			Namespace: deploymentNS,
		},
	}

	// get targetUtil from CM
	var testInt int32
	testInt = 2
	
	// setup metrics, get target util from cm
	metrics := []v2beta1.MetricSpec {
		{
			Type: v2beta1.ResourceMetricSourceType,
			Resource: &v2beta1.ResourceMetricSource{
				Name: "cpu",
				TargetAverageUtilization: &testInt,
			},
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, &hpa, func() error {
		hpa.Spec.ScaleTargetRef.Name = deploymentName
		hpa.Spec.MaxReplicas = maxReplicas
		hpa.Spec.MinReplicas = minReplicas
		hpa.Spec.Metrics = metrics

		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
