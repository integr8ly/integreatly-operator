package resources

import (
	"context"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultProbeModule = "http_2xx"
	labelKey           = "monitoring-key"
	labelValue         = "middleware"
)

func CreatePrometheusProbe(ctx context.Context, client k8sclient.Client, inst *integreatlyv1alpha1.RHMI, name string, module string, targets monv1.ProbeTargetStaticConfig) (integreatlyv1alpha1.StatusPhase, error) {
	if len(targets.Targets) == 0 {
		// Retry later if the URL(s) is not yet known
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	// The default policy is to require a 2xx http return code
	if module == "" {
		module = defaultProbeModule

	}

	// Prepare the probe
	probe := &monv1.Probe{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: inst.Namespace,
		},
	}
	owner.AddIntegreatlyOwnerAnnotations(probe, inst)
	_, err := controllerutil.CreateOrUpdate(ctx, client, probe, func() error {
		probe.Labels = map[string]string{
			labelKey: labelValue,
		}
		probe.Spec = monv1.ProbeSpec{
			JobName: "blackbox",
			ProberSpec: monv1.ProberSpec{
				URL:    "127.0.0.1:9115",
				Scheme: "http",
				Path:   "/probe",
			},
			Module: module,
			Targets: monv1.ProbeTargets{
				StaticConfig: &targets,
			},
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
