package resources

import (
	"context"
	"errors"
	"fmt"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PodTemplateSelector is a function that obtains the PodTemplateSpec from a resource
// that contains it
type PodTemplateSelector func(v1.Object) *corev1.PodTemplateSpec

// ReconcileZoneTopologySpreadConstraints populates obj with the resource found by
// the objKey. If it's not found it returns PhaseAwaitingComponents. If it is
// found, it updates the CR, setting the TopologySpreadConstraints for AZ spreading
func ReconcileZoneTopologySpreadConstraints(ctx context.Context, client client.Client, objKey client.ObjectKey, templateSelector PodTemplateSelector, labelMatch string, obj runtime.Object) (integreatlyv1alpha1.StatusPhase, error) {
	err := client.Get(ctx, objKey, obj)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseAwaitingComponents, nil
		}

		return integreatlyv1alpha1.PhaseFailed, err
	}

	if err := UpdateZoneTopologySpreadConstraints(ctx, client, templateSelector, labelMatch, obj); err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

// UpdateZoneTopologySpreadConstraints updates the topologySpreadConstraints
// on obj to spread the pods among multiple zones
func UpdateZoneTopologySpreadConstraints(ctx context.Context, client client.Client, templateSelector PodTemplateSelector, labelMatch string, obj runtime.Object) error {
	if err := SetZoneTopologySpreadConstraints(templateSelector, labelMatch, obj.(v1.Object)); err != nil {
		return err
	}

	return client.Update(ctx, obj)
}

func SetZoneTopologySpreadConstraints(templateSelector PodTemplateSelector, labelMatch string, obj v1.Object) error {
	podTemplate := templateSelector(obj)

	labels := obj.GetLabels()
	if labels == nil {
		return errors.New("object has no labels")
	}

	labelValue, ok := labels[labelMatch]
	if !ok {
		return fmt.Errorf("label %s not found in object", labelMatch)
	}

	podTemplate.Spec.TopologySpreadConstraints = []corev1.TopologySpreadConstraint{
		{
			MaxSkew:           1,
			TopologyKey:       "topology.kubernetes.io/zone",
			WhenUnsatisfiable: corev1.ScheduleAnyway,
			LabelSelector: &v1.LabelSelector{
				MatchLabels: map[string]string{
					labelMatch: labelValue,
				},
			},
		},
	}

	return nil
}

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
