package rhssocommon

import (
	"context"
	"fmt"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Note: The code in this file was developed in collaboration with Cursor AI

// RHOAM PrometheusRules expect kube_endpoint_address{endpoint='rhsso-operator-metrics'} in the
// RH SSO operator namespace. Upstream RH SSO operator CSV may not ship this Service; reconcile it here.
const (
	RhssoOperatorMetricsServiceName = "rhsso-operator-metrics"
	// Operator SDK / RHOAM operators commonly bind Prometheus metrics on 8383 (see cloud-resource-operator CSV).
	rhssoOperatorMetricsPort int32 = 8383
)

// ReconcileOperatorMetricsService ensures a ClusterIP Service selects the rhsso-operator deployment and
// exposes a port named rhsso-operator-metrics (required for kube-state-metrics endpoint alerts).
func (r *Reconciler) ReconcileOperatorMetricsService(ctx context.Context, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI, operatorNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	if operatorNamespace == "" {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("operator namespace is empty")
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RhssoOperatorMetricsServiceName,
			Namespace: operatorNamespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, svc, func() error {
		owner.AddIntegreatlyOwnerAnnotations(svc, installation)
		if svc.Labels == nil {
			svc.Labels = map[string]string{}
		}
		svc.Labels["name"] = "rhsso-operator"

		svc.Spec.Selector = map[string]string{
			"name": "rhsso-operator",
		}
		svc.Spec.Type = corev1.ServiceTypeClusterIP
		svc.Spec.Ports = []corev1.ServicePort{
			{
				Name:       RhssoOperatorMetricsServiceName,
				Protocol:   corev1.ProtocolTCP,
				Port:       rhssoOperatorMetricsPort,
				TargetPort: intstr.FromInt(int(rhssoOperatorMetricsPort)),
			},
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile %s/%s Service: %w", operatorNamespace, RhssoOperatorMetricsServiceName, err)
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}
