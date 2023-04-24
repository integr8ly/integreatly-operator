package resources

import (
	"context"
	"github.com/integr8ly/integreatly-operator/pkg/resources/k8s"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// PodTemplateMutation represents a mutating function over a PodTemplate.
// The function receives the object that the PodTemplate belongs to, and
// the PodTemplate itself
type PodTemplateMutation func(metav1.Object, *corev1.PodTemplateSpec) error

// PodTemplateSelector is a function that obtains the PodTemplateSpec from a resource
// that contains it
type PodTemplateSelector func(v1.Object) *corev1.PodTemplateSpec

// SelectFromStatefulSet returns obj.Spec.Template
func SelectFromStatefulSet(obj v1.Object) *corev1.PodTemplateSpec {
	ss := obj.(*appsv1.StatefulSet)
	return &ss.Spec.Template
}

// SelectFromDeploymentConfig returns obj.Spec.Template
func SelectFromDeploymentConfig(obj v1.Object) *corev1.PodTemplateSpec {
	dc := obj.(*openshiftappsv1.DeploymentConfig)
	return dc.Spec.Template
}

// SelectFromDeployment returns obj.Spec.Template
func SelectFromDeployment(obj v1.Object) *corev1.PodTemplateSpec {
	d := obj.(*appsv1.Deployment)
	return &d.Spec.Template
}

// UpdatePodTemplateIfExists updates the template retrieved by templateSelector
// on obj by applying the given mutation. If the object obj is not found, it
// returns InProgress and no error
func UpdatePodTemplateIfExists(ctx context.Context, client k8sclient.Client, templateSelector PodTemplateSelector, mutation PodTemplateMutation, obj metav1.Object) (integreatlyv1alpha1.StatusPhase, error) {
	mutateFn := func() error {
		podTemplate := templateSelector(obj)
		return mutation(obj, podTemplate)
	}

	return k8s.UpdateIfExists(ctx, client, mutateFn, obj.(k8sclient.Object))
}

// SetPodTemplate updates the template retrieved by templateSelector
// on obj by applying the given mutation.
func SetPodTemplate(templateSelector PodTemplateSelector, mutation PodTemplateMutation, obj metav1.Object) error {
	podTemplate := templateSelector(obj)
	return mutation(obj, podTemplate)
}

// AllMutationsOf composes a list of PodTemplateMutations by applying them
// sequentially
func AllMutationsOf(mutations ...PodTemplateMutation) PodTemplateMutation {
	return func(obj metav1.Object, podTemplate *corev1.PodTemplateSpec) error {
		for _, mutation := range mutations {
			if err := mutation(obj, podTemplate); err != nil {
				return err
			}
		}

		return nil
	}
}

// NoopMutate is a PodTemplateMutation that doesn't perform any changes
func NoopMutate(_ metav1.Object, _ *corev1.PodTemplateSpec) error {
	return nil
}
