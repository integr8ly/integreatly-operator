package resources

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/integr8ly/integreatly-operator/utils"
	"k8s.io/apimachinery/pkg/runtime"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestReconcileTopologySpreadConstraints(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []struct {
		Name             string
		InitObjs         []runtime.Object
		TemplateSelector PodTemplateSelector
		LabelMatch       string
		TargetObj        v1.Object
		Assert           func(k8sclient.Client, integreatlyv1alpha1.StatusPhase, error) error
	}{
		{
			Name:       "Object not found",
			InitObjs:   []runtime.Object{},
			LabelMatch: "app",
			TargetObj: &openshiftappsv1.DeploymentConfig{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-statefulset",
					Namespace: "redhat-test-operator",
				},
			},
			TemplateSelector: SelectFromDeploymentConfig,
			Assert: func(_ k8sclient.Client, phase integreatlyv1alpha1.StatusPhase, err error) error {
				if err != nil {
					return fmt.Errorf("unexpected error: %v", err)
				}

				if phase != integreatlyv1alpha1.PhaseInProgress {
					return fmt.Errorf("unexpected phase. Expected %s, got %s", integreatlyv1alpha1.PhaseInProgress, phase)
				}

				return nil
			},
		},
		{
			Name: "StatefulSet",
			InitObjs: []runtime.Object{
				&appsv1.StatefulSet{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test-statefulset",
						Namespace: "redhat-test-operator",
						Labels: map[string]string{
							"app": "test-statefulset",
						},
					},
					Spec: appsv1.StatefulSetSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Image:   "test",
										Command: []string{"test"},
										Name:    "test-container",
									},
								},
							},
						},
					},
				},
			},
			LabelMatch: "app",
			TargetObj: &appsv1.StatefulSet{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-statefulset",
					Namespace: "redhat-test-operator",
				},
			},
			TemplateSelector: SelectFromStatefulSet,
			Assert: func(client k8sclient.Client, phase integreatlyv1alpha1.StatusPhase, err error) error {
				if err != nil {
					return fmt.Errorf("unexpected error: %v", err)
				}

				ss := &appsv1.StatefulSet{}
				if err := client.Get(context.TODO(), k8sclient.ObjectKey{
					Name:      "test-statefulset",
					Namespace: "redhat-test-operator",
				}, ss); err != nil {
					return fmt.Errorf("failed to obtain stateful set: %v", err)
				}

				if ss.Spec.Template.Spec.Containers[0].Name != "test-container" {
					return fmt.Errorf("unexpexted value for stateful set container. Expected test-container, got %s", ss.Spec.Template.Spec.Containers[0].Name)
				}

				expectedTopology := corev1.TopologySpreadConstraint{
					MaxSkew:           1,
					TopologyKey:       "topology.kubernetes.io/zone",
					WhenUnsatisfiable: corev1.ScheduleAnyway,
					LabelSelector: &v1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "test-statefulset",
						},
					},
				}

				if !reflect.DeepEqual(ss.Spec.Template.Spec.TopologySpreadConstraints[0], expectedTopology) {
					return fmt.Errorf("invalid value for TopologySpreadConstraint: %v", ss.Spec.Template.Spec.TopologySpreadConstraints[0])
				}

				return nil
			},
		},
		{
			Name: "DeploymentConfig",
			InitObjs: []runtime.Object{
				&openshiftappsv1.DeploymentConfig{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test-dc",
						Namespace: "redhat-test-operator",
						Labels: map[string]string{
							"app": "test-dc",
						},
					},
					Spec: openshiftappsv1.DeploymentConfigSpec{
						Template: &corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Image:   "test",
										Command: []string{"test"},
										Name:    "test-container",
									},
								},
							},
						},
					},
				},
			},
			LabelMatch: "app",
			TargetObj: &openshiftappsv1.DeploymentConfig{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-dc",
					Namespace: "redhat-test-operator",
				},
			},
			TemplateSelector: SelectFromDeploymentConfig,
			Assert: func(client k8sclient.Client, phase integreatlyv1alpha1.StatusPhase, err error) error {
				if err != nil {
					return fmt.Errorf("unexpected error: %v", err)
				}

				ss := &openshiftappsv1.DeploymentConfig{}
				if err := client.Get(context.TODO(), k8sclient.ObjectKey{
					Name:      "test-dc",
					Namespace: "redhat-test-operator",
				}, ss); err != nil {
					return fmt.Errorf("failed to obtain deployment config: %v", err)
				}

				if ss.Spec.Template.Spec.Containers[0].Name != "test-container" {
					return fmt.Errorf("unexpexted value for stateful set container. Expected test-container, got %s", ss.Spec.Template.Spec.Containers[0].Name)
				}

				expectedTopology := corev1.TopologySpreadConstraint{
					MaxSkew:           1,
					TopologyKey:       "topology.kubernetes.io/zone",
					WhenUnsatisfiable: corev1.ScheduleAnyway,
					LabelSelector: &v1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "test-dc",
						},
					},
				}

				if !reflect.DeepEqual(ss.Spec.Template.Spec.TopologySpreadConstraints[0], expectedTopology) {
					return fmt.Errorf("invalid value for TopologySpreadConstraint: %v", ss.Spec.Template.Spec.TopologySpreadConstraints[0])
				}

				return nil
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			client := utils.NewTestClient(scheme, scenario.InitObjs...)
			phase, reconcileErr := UpdatePodTemplateIfExists(
				context.TODO(),
				client,
				scenario.TemplateSelector,
				MutateZoneTopologySpreadConstraints(scenario.LabelMatch),
				scenario.TargetObj,
			)

			if err := scenario.Assert(client, phase, reconcileErr); err != nil {
				t.Error(err)
			}
		})
	}
}
