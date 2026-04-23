package rhssocommon

import (
	"context"
	"strings"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	"github.com/integr8ly/integreatly-operator/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Note: The code in this file was developed in collaboration with Cursor AI

func TestReconciler_ReconcileOperatorMetricsService(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	const operatorNS = "redhat-rhoam-rhsso-operator"

	installation := &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhoam",
			Namespace: "redhat-rhoam-operator",
		},
	}

	r := &Reconciler{}

	t.Run("empty operator namespace returns failed phase", func(t *testing.T) {
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		phase, err := r.ReconcileOperatorMetricsService(context.Background(), client, installation, "")
		if phase != integreatlyv1alpha1.PhaseFailed {
			t.Fatalf("phase = %v, want PhaseFailed", phase)
		}
		if err == nil || !strings.Contains(err.Error(), "operator namespace is empty") {
			t.Fatalf("err = %v, want message containing operator namespace is empty", err)
		}
	})

	t.Run("creates Service with expected spec and owner annotations", func(t *testing.T) {
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		phase, err := r.ReconcileOperatorMetricsService(context.Background(), client, installation, operatorNS)
		if err != nil {
			t.Fatalf("ReconcileOperatorMetricsService: %v", err)
		}
		if phase != integreatlyv1alpha1.PhaseCompleted {
			t.Fatalf("phase = %v, want PhaseCompleted", phase)
		}

		got := &corev1.Service{}
		key := types.NamespacedName{Namespace: operatorNS, Name: RhssoOperatorMetricsServiceName}
		if err := client.Get(context.Background(), key, got); err != nil {
			t.Fatalf("Get Service: %v", err)
		}

		if got.Labels["name"] != "rhsso-operator" {
			t.Errorf("labels[name] = %q, want rhsso-operator", got.Labels["name"])
		}
		if got.Annotations[owner.IntegreatlyOwnerName] != installation.Name {
			t.Errorf("owner name annotation = %q, want %q", got.Annotations[owner.IntegreatlyOwnerName], installation.Name)
		}
		if got.Annotations[owner.IntegreatlyOwnerNamespace] != installation.Namespace {
			t.Errorf("owner namespace annotation = %q, want %q", got.Annotations[owner.IntegreatlyOwnerNamespace], installation.Namespace)
		}
		if got.Spec.Type != corev1.ServiceTypeClusterIP {
			t.Errorf("Spec.Type = %v, want ClusterIP", got.Spec.Type)
		}
		if len(got.Spec.Ports) != 1 {
			t.Fatalf("len(Spec.Ports) = %d, want 1", len(got.Spec.Ports))
		}
		p := got.Spec.Ports[0]
		if p.Name != RhssoOperatorMetricsServiceName || p.Port != rhssoOperatorMetricsPort || p.Protocol != corev1.ProtocolTCP {
			t.Errorf("port = {Name:%q Port:%d Protocol:%v}, want name=%q port=%d TCP",
				p.Name, p.Port, p.Protocol, RhssoOperatorMetricsServiceName, rhssoOperatorMetricsPort)
		}
		if p.TargetPort != intstr.FromInt(int(rhssoOperatorMetricsPort)) {
			t.Errorf("TargetPort = %#v, want FromInt(%d)", p.TargetPort, rhssoOperatorMetricsPort)
		}
		wantSel := map[string]string{"name": "rhsso-operator"}
		if len(got.Spec.Selector) != len(wantSel) || got.Spec.Selector["name"] != wantSel["name"] {
			t.Errorf("Spec.Selector = %v, want %v", got.Spec.Selector, wantSel)
		}
	})

	t.Run("updates existing Service to expected spec", func(t *testing.T) {
		existing := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      RhssoOperatorMetricsServiceName,
				Namespace: operatorNS,
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Selector: map[string]string{
					"name": "wrong-selector",
				},
				Ports: []corev1.ServicePort{
					{Name: "metrics", Port: 9999, Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromInt(9999)},
				},
			},
		}
		client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(existing).Build()

		phase, err := r.ReconcileOperatorMetricsService(context.Background(), client, installation, operatorNS)
		if err != nil {
			t.Fatalf("ReconcileOperatorMetricsService: %v", err)
		}
		if phase != integreatlyv1alpha1.PhaseCompleted {
			t.Fatalf("phase = %v, want PhaseCompleted", phase)
		}

		got := &corev1.Service{}
		key := types.NamespacedName{Namespace: operatorNS, Name: RhssoOperatorMetricsServiceName}
		if err := client.Get(context.Background(), key, got); err != nil {
			t.Fatalf("Get Service: %v", err)
		}
		if got.Spec.Selector["name"] != "rhsso-operator" {
			t.Errorf("Spec.Selector[name] = %q after update", got.Spec.Selector["name"])
		}
		if len(got.Spec.Ports) != 1 || got.Spec.Ports[0].Port != rhssoOperatorMetricsPort {
			t.Errorf("ports after update = %#v", got.Spec.Ports)
		}
	})
}
