package autoscaling

import (
	"context"
	"fmt"
	"strconv"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func ReconcileHPA(ctx context.Context, client k8sclient.Client, installation integreatlyv1alpha1.RHMI, hpaTargetName string, hpaTargetNamespace string, minReplicas *int32, maxReplicas int32) (integreatlyv1alpha1.StatusPhase, error) {
	hpa := v2beta1.HorizontalPodAutoscaler{
		ObjectMeta: v1.ObjectMeta{
			Name:      hpaTargetName,
			Namespace: hpaTargetNamespace,
		},
	}

	// if autoscaling is set to false, attempt to delete HPA object
	if !installation.Spec.AutoscalingEnabled {
		err := client.Delete(ctx, &hpa)
		if err != nil {
			if k8serr.IsNotFound(err) {
				return integreatlyv1alpha1.PhaseCompleted, nil
			}
			return integreatlyv1alpha1.PhaseFailed, err
		}

		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	// retrieve targetUtilization configmap
	targetUtilizationCM := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      "autoscaling-config",
			Namespace: installation.ObjectMeta.Namespace,
		},
	}
	err := client.Get(ctx, k8sclient.ObjectKey{Name: targetUtilizationCM.Name, Namespace: targetUtilizationCM.Namespace}, targetUtilizationCM)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to retrieve target utilization CM: %s", err)
	}

	targetUtilization, err := strconv.ParseInt(targetUtilizationCM.Data[hpaTargetName], 10, 32)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to convert target utilization CM value to int: %s", err)
	}
	convertedTargetUtilization := int32(targetUtilization)

	// setup metrics, get target util from cm
	metrics := []v2beta1.MetricSpec{
		{
			Type: v2beta1.ResourceMetricSourceType,
			Resource: &v2beta1.ResourceMetricSource{
				Name:                     "cpu",
				TargetAverageUtilization: &convertedTargetUtilization,
			},
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, client, &hpa, func() error {
		hpa.Spec.ScaleTargetRef.Name = hpaTargetName
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
