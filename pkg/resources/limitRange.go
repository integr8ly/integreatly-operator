package resources

import (
	"context"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultLimitRangeName = "limit-range"
)

type LimitRangeParams struct {
	CpuRequest    string
	CpuLimit      string
	MemoryRequest string
	MemoryLimit   string
	ResourceType  corev1.LimitType
}

var DefaultLimitRangeParams = LimitRangeParams{
	CpuRequest:    "5m",
	MemoryRequest: "10Mi",
	ResourceType:  corev1.LimitTypeContainer,
}

func ReconcileLimitRange(ctx context.Context, client k8sclient.Client, namespace string, params LimitRangeParams) (integreatlyv1alpha1.StatusPhase, error) {
	limitRange := &corev1.LimitRange{
		TypeMeta: metav1.TypeMeta{
			Kind:       "LimitRange",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultLimitRangeName,
			Namespace: namespace,
		},
		Spec: corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{
				{
					Type:           params.ResourceType,
					Default:        map[corev1.ResourceName]resource.Quantity{},
					DefaultRequest: map[corev1.ResourceName]resource.Quantity{},
				},
			},
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, client, limitRange, func() error {
		if params.CpuLimit != "" {
			limitRange.Spec.Limits[0].Default[corev1.ResourceCPU] = resource.MustParse(params.CpuLimit)
		}
		if params.CpuRequest != "" {
			limitRange.Spec.Limits[0].DefaultRequest[corev1.ResourceCPU] = resource.MustParse(params.CpuRequest)
		}
		if params.MemoryLimit != "" {
			limitRange.Spec.Limits[0].Default[corev1.ResourceMemory] = resource.MustParse(params.MemoryLimit)
		}
		if params.MemoryRequest != "" {
			limitRange.Spec.Limits[0].DefaultRequest[corev1.ResourceMemory] = resource.MustParse(params.MemoryRequest)
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
