package resources

import (
	"context"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

func ReconcilePodPriority(ctx context.Context, client client.Client, objKey client.ObjectKey, templateSelector PodTemplateSelector, obj runtime.Object, priorityClassName string) (integreatlyv1alpha1.StatusPhase, error) {
	err := client.Get(ctx, objKey, obj)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseAwaitingComponents, nil
		}

		return integreatlyv1alpha1.PhaseFailed, err
	}

	if err := UpdatePodPriority(ctx, client, templateSelector, obj, priorityClassName); err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func UpdatePodPriority(ctx context.Context, client client.Client, templateSelector PodTemplateSelector, obj runtime.Object, priorityClassName string) error {
	if err := SetPodPriority(templateSelector, obj.(v1.Object), priorityClassName); err != nil {
		return err
	}

	return client.Update(ctx, obj)
}

func SetPodPriority(templateSelector PodTemplateSelector, obj v1.Object, priorityClassName string) error {
	podTemplate := templateSelector(obj)

	podTemplate.Spec.PriorityClassName = priorityClassName

	return nil
}
