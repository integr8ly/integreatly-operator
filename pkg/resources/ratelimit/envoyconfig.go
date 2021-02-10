package ratelimit

import (
	"context"
	"fmt"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func DeleteEnvoyConfigsInNamespaces(ctx context.Context, client k8sclient.Client, namespaces ...string) (integreatlyv1alpha1.StatusPhase, error) {
	phase := integreatlyv1alpha1.PhaseCompleted

	for _, namespace := range namespaces {
		nsPhase, err := DeleteEnvoyConfigsInNamespace(ctx, client, namespace)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		// Only change the status phase if it was Completed, to ensure that
		// as long as one of the namespaces returns InProgress, the phase is
		// set to InProgress
		if phase == integreatlyv1alpha1.PhaseCompleted {
			phase = nsPhase
		}
	}

	return phase, nil
}

func DeleteEnvoyConfigsInNamespace(ctx context.Context, client k8sclient.Client, namespace string) (integreatlyv1alpha1.StatusPhase, error) {
	envoyConfigs := &marin3rv1alpha1.EnvoyConfigList{}
	if err := client.List(ctx, envoyConfigs, k8sclient.InNamespace(namespace)); err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	if len(envoyConfigs.Items) == 0 {
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	for _, envoyConfig := range envoyConfigs.Items {
		if err := k8sclient.IgnoreNotFound(
			client.Delete(ctx, &envoyConfig),
		); err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to delete envoyconfig for namespace %s: %v",
				namespace, err)
		}
	}

	return integreatlyv1alpha1.PhaseInProgress, nil
}
