package marin3r

import (
	"context"
	"fmt"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	marin3rconfig "github.com/integr8ly/integreatly-operator/pkg/products/marin3r/config"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRateLimitService(t *testing.T) {
	scheme := newScheme()

	scenarios := []struct {
		Name       string
		Reconciler *RateLimitServiceReconciler
		InitObjs   []runtime.Object
		Assert     func(k8sclient.Client, integreatlyv1alpha1.StatusPhase, error) error
	}{
		{
			Name: "Service deployed without metrics",
			Reconciler: NewRateLimitServiceReconciler(&marin3rconfig.RateLimitConfig{
				Unit:            "minute",
				RequestsPerUnit: 1,
			},
				&integreatlyv1alpha1.RHMI{}, "redhat-test-marin3r", "ratelimit-redis"),
			InitObjs: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
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
						Name:      "ratelimit-config",
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

						if url != "test-url" {
							return fmt.Errorf("unexpected value for REDIS_URL: %s", url)
						}

						return nil
					},
					"USE_STATSD": func(s string) error {
						if s != "false" {
							return fmt.Errorf("unexpected value for USE_STATSD variable. Expected true, got %s", s)
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
					ObjectMeta: v1.ObjectMeta{
						Name:      "ratelimit-redis",
						Namespace: "redhat-test-marin3r",
					},
					Data: map[string][]byte{
						"URL": []byte("test-url"),
					},
				},
			},
			Reconciler: NewRateLimitServiceReconciler(
				&marin3rconfig.RateLimitConfig{
					Unit:            "minute",
					RequestsPerUnit: 1,
				},
				&integreatlyv1alpha1.RHMI{},
				"redhat-test-marin3r",
				"ratelimit-redis",
			).
				WithStatsdConfig(StatsdConfig{
					Host: "test-host",
					Port: "9092",
				}),
			Assert: allOf(
				assertNoError,
				assertPhase(integreatlyv1alpha1.PhaseCompleted),
				assertDeployment(assertEnvs(map[string]func(string) error{
					"STATSD_PORT": func(s string) error {
						if s != "9092" {
							return fmt.Errorf("unexpected value for STATSD_PORT variable. Expected 9092, got %s", s)
						}

						return nil
					},
					"STATSD_HOST": func(s string) error {
						if s != "test-host" {
							return fmt.Errorf("unexpected value for STATSD_HOST variable. Expected test-host, got %s", s)
						}

						return nil
					},
					"USE_STATSD": func(s string) error {
						if s != "true" {
							return fmt.Errorf("unexpected value for USE_STATSD variable. Expected true, got %s", s)
						}

						return nil
					},
				})),
			),
		},

		{
			Name:     "Wait for redis",
			InitObjs: []runtime.Object{},
			Reconciler: NewRateLimitServiceReconciler(&marin3rconfig.RateLimitConfig{
				Unit:            "minute",
				RequestsPerUnit: 1,
			},
				&integreatlyv1alpha1.RHMI{}, "redhat-test-marin3r", "ratelimit-redis"),
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
					ObjectMeta: v1.ObjectMeta{
						Name:      "ratelimit-redis",
						Namespace: "redhat-test-marin3r",
					},
					Data: map[string][]byte{
						"URL": []byte("test-url"),
					},
				},
			},
			Reconciler: NewRateLimitServiceReconciler(
				&marin3rconfig.RateLimitConfig{
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
			).
				WithStatsdConfig(StatsdConfig{
					Host: "test-host",
					Port: "9092",
				}),
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
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			client := fake.NewFakeClientWithScheme(scheme, scenario.InitObjs...)
			phase, err := scenario.Reconciler.ReconcileRateLimitService(context.TODO(), client)

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
