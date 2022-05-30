package marin3r

import (
	"context"
	"fmt"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	marin3rconfig "github.com/integr8ly/integreatly-operator/pkg/products/marin3r/config"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	corev1 "k8s.io/api/core/v1"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRateLimitService(t *testing.T) {
	scheme := newScheme()
	v2beta1.AddToScheme(scheme)
	minReplicasValue := int32(2)
	targetUtil := int32(60)

	scenarios := []struct {
		Name          string
		Reconciler    *RateLimitServiceReconciler
		ProductConfig quota.ProductConfig
		InitObjs      []runtime.Object
		Assert        func(k8sclient.Client, integreatlyv1alpha1.StatusPhase, error) error
	}{
		{
			Name: "Service deployed without metrics",
			Reconciler: NewRateLimitServiceReconciler(marin3rconfig.RateLimitConfig{
				Unit:            "minute",
				RequestsPerUnit: 1,
			},
				&integreatlyv1alpha1.RHMI{}, "redhat-test-marin3r", "ratelimit-redis"),
			ProductConfig: &quota.ProductConfigMock{
				ConfigureFunc: func(obj metav1.Object) error {
					return nil
				},
			},
			InitObjs: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ratelimit-redis",
						Namespace: "redhat-test-marin3r",
					},
					Data: map[string][]byte{
						"URL": []byte("test-url"),
					},
				},
			},
			Assert: allOf(
				assertNoError,
				assertPhase(integreatlyv1alpha1.PhaseCompleted),
				func(client k8sclient.Client, phase integreatlyv1alpha1.StatusPhase, reconcileError error) error {
					configMap := &corev1.ConfigMap{}
					if err := client.Get(context.TODO(), k8sclient.ObjectKey{
						Name:      RateLimitingConfigMapName,
						Namespace: "redhat-test-marin3r",
					}, configMap); err != nil {
						return fmt.Errorf("failed to obtain expected ConfigMap: %v", err)
					}

					return nil
				},
				assertDeployment(assertEnvs(map[string]func(string) error{
					"REDIS_URL": func(url string) error {
						if url == "" {
							return errors.Errorf("REDIS_URL not found in environment variables")
						}

						if url != "redis://test-url" {
							return fmt.Errorf("unexpected value for REDIS_URL: %s", url)
						}

						return nil
					},
				})),
				func(client k8sclient.Client, phase integreatlyv1alpha1.StatusPhase, reconcileError error) error {
					service := &corev1.Service{}
					if err := client.Get(context.TODO(), k8sclient.ObjectKey{
						Name:      "ratelimit",
						Namespace: "redhat-test-marin3r",
					}, service); err != nil {
						return fmt.Errorf("failed to obtain expected service: %v", err)
					}

					return nil
				},
			),
		},

		{
			Name: "Service deployed with metrics",
			InitObjs: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ratelimit-redis",
						Namespace: "redhat-test-marin3r",
					},
					Data: map[string][]byte{
						"URL": []byte("test-url"),
					},
				},
			},
			Reconciler: NewRateLimitServiceReconciler(
				marin3rconfig.RateLimitConfig{
					Unit:            "minute",
					RequestsPerUnit: 1,
				},
				&integreatlyv1alpha1.RHMI{},
				"redhat-test-marin3r",
				"ratelimit-redis",
			),
			ProductConfig: &quota.ProductConfigMock{
				ConfigureFunc: func(obj metav1.Object) error {
					return nil
				},
			},
			Assert: allOf(
				assertNoError,
				assertPhase(integreatlyv1alpha1.PhaseCompleted),
				assertDeployment(assertEnvs(map[string]func(string) error{
					"REDIS_URL": func(url string) error {
						if url == "" {
							return errors.Errorf("REDIS_URL not found in environment variables")
						}

						if url != "redis://test-url" {
							return fmt.Errorf("unexpected value for REDIS_URL: %s", url)
						}

						return nil
					},
					"RUST_LOG": func(level string) error {
						if level == "" {
							return errors.Errorf("RUST_LOG not found in environment variables")
						}

						if level != "info" {
							return fmt.Errorf("unexpected value for RUST_LOG: %s", level)
						}

						return nil
					},
					"LIMITS_FILE": func(path string) error {
						if path == "" {
							return errors.Errorf("LIMITS_FILE not found in environment variables")
						}

						return nil
					},
				})),
			),
		},

		{
			Name:     "Wait for redis",
			InitObjs: []runtime.Object{},
			Reconciler: NewRateLimitServiceReconciler(marin3rconfig.RateLimitConfig{
				Unit:            "minute",
				RequestsPerUnit: 1,
			},
				&integreatlyv1alpha1.RHMI{}, "redhat-test-marin3r", "ratelimit-redis"),
			ProductConfig: &quota.ProductConfigMock{
				ConfigureFunc: func(obj metav1.Object) error {
					return nil
				},
			},
			Assert: allOf(
				assertNoError,
				assertPhase(integreatlyv1alpha1.PhaseAwaitingComponents),
				assertDeployment(func(_ *appsv1.Deployment, e error) error {
					if !k8serrors.IsNotFound(e) {
						return fmt.Errorf("expected deployment not found error, got: %v", e)
					}
					return nil
				}),
			),
		},

		{
			Name: "Pod priority set",
			InitObjs: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ratelimit-redis",
						Namespace: "redhat-test-marin3r",
					},
					Data: map[string][]byte{
						"URL": []byte("test-url"),
					},
				},
			},
			Reconciler: NewRateLimitServiceReconciler(
				marin3rconfig.RateLimitConfig{
					Unit:            "minute",
					RequestsPerUnit: 1,
				},
				&integreatlyv1alpha1.RHMI{
					Spec: integreatlyv1alpha1.RHMISpec{
						PriorityClassName: "rhoam-pod-priority",
					},
				},
				"redhat-test-marin3r",
				"ratelimit-redis",
			),
			ProductConfig: &quota.ProductConfigMock{
				ConfigureFunc: func(obj metav1.Object) error {
					return nil
				},
			},
			Assert: allOf(
				assertNoError,
				assertPhase(integreatlyv1alpha1.PhaseCompleted),
				assertDeployment(func(deployment *appsv1.Deployment, e error) error {
					if deployment.Spec.Template.Spec.PriorityClassName != "rhoam-pod-priority" {
						return fmt.Errorf("expected pod priority not set, got: %v", e)
					}
					return nil
				}),
			),
		},

		{
			Name: "confirm that HPA was created for rate limiting",
			Reconciler: NewRateLimitServiceReconciler(marin3rconfig.RateLimitConfig{
				Unit:            "minute",
				RequestsPerUnit: 1,
			},
				&integreatlyv1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rhoam",
						Namespace: "redhat-rhoam-operator",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						AutoscalingEnabled: true,
					},
				}, "redhat-test-marin3r", "ratelimit-redis"),
			ProductConfig: &quota.ProductConfigMock{
				ConfigureFunc: func(obj metav1.Object) error {
					return nil
				},
			},
			InitObjs: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ratelimit-redis",
						Namespace: "redhat-test-marin3r",
					},
					Data: map[string][]byte{
						"URL": []byte("test-url"),
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "autoscaling-config",
						Namespace: "redhat-rhoam-operator",
					},
					Data: map[string]string{
						"ratelimit": "60",
					},
				},
			},
			Assert: allOf(
				assertNoError,
				assertPhase(integreatlyv1alpha1.PhaseCompleted),
				func(client k8sclient.Client, phase integreatlyv1alpha1.StatusPhase, reconcileError error) error {

					hpaList := v2beta1.HorizontalPodAutoscalerList{}
					err := client.List(context.TODO(), &hpaList)
					if err != nil {
						return fmt.Errorf("failed to obtain expected hpa: %v", err)
					}
					for _, hpa := range hpaList.Items {
						if hpa.Name != "ratelimit" {
							return fmt.Errorf("required ratelimit hpa not found")
						}
						if hpa.Spec.MaxReplicas != int32(3) {
							return fmt.Errorf("ratelimit hpa max replicas values incorrect got: %v, want: %v", hpa.Spec.MaxReplicas, 3)
						}
						if *hpa.Spec.MinReplicas != int32(minReplicasValue) {
							return fmt.Errorf("ratelimit hpa min replicas values incorrect got: %v, want: %v", *hpa.Spec.MinReplicas, minReplicasValue)
						}
						if *hpa.Spec.Metrics[0].Resource.TargetAverageUtilization != targetUtil {
							return fmt.Errorf("ratelimit targetUtils values incorrect got: %v, want: %v", *hpa.Spec.Metrics[0].Resource.TargetAverageUtilization, targetUtil)
						}
					}
					return nil
				},
			),
		},
		{
			Name: "confirm that HPA was deleted for rate limiting",
			Reconciler: NewRateLimitServiceReconciler(marin3rconfig.RateLimitConfig{
				Unit:            "minute",
				RequestsPerUnit: 1,
			},
				&integreatlyv1alpha1.RHMI{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rhoam",
						Namespace: "redhat-rhoam-operator",
					},
					Spec: integreatlyv1alpha1.RHMISpec{
						AutoscalingEnabled: false,
					},
				}, "redhat-test-marin3r", "ratelimit-redis"),
			ProductConfig: &quota.ProductConfigMock{
				ConfigureFunc: func(obj metav1.Object) error {
					return nil
				},
			},
			InitObjs: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ratelimit-redis",
						Namespace: "redhat-test-marin3r",
					},
					Data: map[string][]byte{
						"URL": []byte("test-url"),
					},
				},
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "autoscaling-config",
						Namespace: "redhat-rhoam-operator",
					},
					Data: map[string]string{
						"ratelimit": "60",
					},
				},
				&v2beta1.HorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ratelimit",
						Namespace: "redhat-rhoam-marin3r",
					},
					Spec: v2beta1.HorizontalPodAutoscalerSpec{
						MinReplicas: &minReplicasValue,
						MaxReplicas: int32(3),
						Metrics: []v2beta1.MetricSpec{
							{
								Resource: &v2beta1.ResourceMetricSource{
									TargetAverageUtilization: &targetUtil,
									Name:                     "cpu",
								},
							},
						},
					},
				},
			},
			Assert: allOf(
				assertNoError,
				assertPhase(integreatlyv1alpha1.PhaseCompleted),
				func(client k8sclient.Client, phase integreatlyv1alpha1.StatusPhase, reconcileError error) error {
					hpaList := v2beta1.HorizontalPodAutoscalerList{}
					err := client.List(context.TODO(), &hpaList)
					if err != nil {
						return fmt.Errorf("failed to obtain expected hpa: %v", err)
					}
					if len(hpaList.Items) != 0 {
						return fmt.Errorf("failed to delete hpa, expecting: 0 hpas, found: %v", len(hpaList.Items))
					}
					return nil
				},
			),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			client := fake.NewFakeClientWithScheme(scheme, scenario.InitObjs...)
			phase, err := scenario.Reconciler.ReconcileRateLimitService(context.TODO(), client, scenario.ProductConfig)

			if err := scenario.Assert(client, phase, err); err != nil {
				t.Error(err)
			}
		})
	}
}

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	appsv1.AddToScheme(scheme)

	return scheme
}

func assertPhase(expectedPhase integreatlyv1alpha1.StatusPhase) func(k8sclient.Client, integreatlyv1alpha1.StatusPhase, error) error {
	return func(_ k8sclient.Client, phase integreatlyv1alpha1.StatusPhase, _ error) error {
		if phase != expectedPhase {
			return fmt.Errorf("unexpected phase. Expected %s, got %s", expectedPhase, phase)
		}

		return nil
	}
}

func assertNoError(_ k8sclient.Client, _ integreatlyv1alpha1.StatusPhase, err error) error {
	if err != nil {
		return fmt.Errorf("unexpected error: %v", err)
	}

	return nil
}

func allOf(assertions ...func(k8sclient.Client, integreatlyv1alpha1.StatusPhase, error) error) func(k8sclient.Client, integreatlyv1alpha1.StatusPhase, error) error {
	return func(client k8sclient.Client, phase integreatlyv1alpha1.StatusPhase, err error) error {
		for _, assertion := range assertions {
			if assertionErr := assertion(client, phase, err); assertionErr != nil {
				return assertionErr
			}
		}

		return nil
	}
}

func assertDeployment(assertion func(*appsv1.Deployment, error) error) func(k8sclient.Client, integreatlyv1alpha1.StatusPhase, error) error {
	return func(client k8sclient.Client, phase integreatlyv1alpha1.StatusPhase, err error) error {
		deployment := &appsv1.Deployment{}
		clientErr := client.Get(context.TODO(), k8sclient.ObjectKey{
			Name:      "ratelimit",
			Namespace: "redhat-test-marin3r",
		}, deployment)

		return assertion(deployment, clientErr)
	}
}

func assertEnvs(assertions map[string]func(string) error) func(*appsv1.Deployment, error) error {
	return func(deployment *appsv1.Deployment, err error) error {
		if err != nil {
			return fmt.Errorf("failed to obtain deployment: %v", err)
		}

		for env, assertion := range assertions {
			value := ""

			for _, e := range deployment.Spec.Template.Spec.Containers[0].Env {
				if e.Name == env {
					value = e.Value
					break
				}
			}

			if err := assertion(value); err != nil {
				return err
			}
		}

		return nil
	}
}
