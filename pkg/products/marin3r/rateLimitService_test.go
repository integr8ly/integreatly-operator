package marin3r

import (
	"context"
	"fmt"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
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
			Name:       "Service deployed",
			Reconciler: NewRateLimitServiceReconciler("redhat-test-marin3r", "ratelimit-redis"),
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
			Assert: func(client k8sclient.Client, phase integreatlyv1alpha1.StatusPhase, reconcileError error) error {
				if reconcileError != nil {
					return fmt.Errorf("unexpected error: %v", reconcileError)
				}

				if phase != integreatlyv1alpha1.PhaseCompleted {
					return fmt.Errorf("unexpected phase. Expected PhaseCompleted, got %s", phase)
				}

				configMap := &corev1.ConfigMap{}
				if err := client.Get(context.TODO(), k8sclient.ObjectKey{
					Name:      "ratelimit-config",
					Namespace: "redhat-test-marin3r",
				}, configMap); err != nil {
					return fmt.Errorf("failed to obtain expected ConfigMap: %v", err)
				}

				deployment := &appsv1.Deployment{}
				if err := client.Get(context.TODO(), k8sclient.ObjectKey{
					Name:      "ratelimit",
					Namespace: "redhat-test-marin3r",
				}, deployment); err != nil {
					return fmt.Errorf("failed to obtain expected deployment: %v", err)
				}

				envs := deployment.Spec.Template.Spec.Containers[0].Env
				url, err := func() (string, error) {
					for _, env := range envs {
						if env.Name == "REDIS_URL" {
							return env.Value, nil
						}
					}

					return "", errors.Errorf("REDIS_URL not found in environment variables")
				}()

				if err != nil {
					return err
				}
				if url != "test-url" {
					return fmt.Errorf("unexpected value for REDIS_URL: %s", url)
				}

				service := &corev1.Service{}
				if err := client.Get(context.TODO(), k8sclient.ObjectKey{
					Name:      "ratelimit",
					Namespace: "redhat-test-marin3r",
				}, service); err != nil {
					return fmt.Errorf("failed to obtain expected service: %v", err)
				}

				return nil
			},
		},

		{
			Name:       "Wait for redis",
			InitObjs:   []runtime.Object{},
			Reconciler: NewRateLimitServiceReconciler("redhat-test-marin3r", "ratelimit-redis"),
			Assert: func(client k8sclient.Client, phase integreatlyv1alpha1.StatusPhase, reconcileError error) error {
				if reconcileError != nil {
					return fmt.Errorf("unexpected error: %v", reconcileError)
				}

				if phase != integreatlyv1alpha1.PhaseAwaitingComponents {
					return fmt.Errorf("unexpected phase. Expected %s, got %s",
						integreatlyv1alpha1.PhaseAwaitingComponents,
						phase,
					)
				}

				deployment := &appsv1.Deployment{}
				if err := client.Get(context.TODO(), k8sclient.ObjectKey{
					Name:      "ratelimit",
					Namespace: "redhat-test-marin3r",
				}, deployment); !k8serrors.IsNotFound(err) {
					return fmt.Errorf("expected deployment not found error, got: %v", err)
				}

				return nil
			},
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
