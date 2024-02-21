package marin3r

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"github.com/integr8ly/integreatly-operator/pkg/resources/ratelimit"
	"github.com/integr8ly/integreatly-operator/utils"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	marin3rconfig "github.com/integr8ly/integreatly-operator/pkg/products/marin3r/config"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestRateLimitService(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	rateLimitPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ratelimit",
			Namespace: "redhat-test-marin3r",
			Labels:    map[string]string{"app": quota.RateLimitName},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	podExecutorMock := &resources.PodExecutorInterfaceMock{
		ExecuteRemoteCommandFunc: func(ns string, podName string, command []string) (string, string, error) {
			return "[{\"namespace\":\"apicast-ratelimit\",\"max_value\":1,\"seconds\":60,\"name\":null,\"conditions\":[\"generic_key == slowpath\"],\"variables\":[\"generic_key\"]}]", "", nil
		},
	}

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
				&integreatlyv1alpha1.RHMI{}, "redhat-test-marin3r", "ratelimit-redis",
				podExecutorMock, &config.ConfigReadWriterMock{},
			),
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
				rateLimitPod,
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
				rateLimitPod,
			},
			Reconciler: NewRateLimitServiceReconciler(
				marin3rconfig.RateLimitConfig{
					Unit:            "minute",
					RequestsPerUnit: 1,
				},
				&integreatlyv1alpha1.RHMI{},
				"redhat-test-marin3r",
				"ratelimit-redis",
				podExecutorMock,
				&config.ConfigReadWriterMock{},
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
				&integreatlyv1alpha1.RHMI{}, "redhat-test-marin3r", "ratelimit-redis", resources.PodExecutor{}, &config.ConfigReadWriterMock{},
			),
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
				rateLimitPod,
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
				podExecutorMock,
				&config.ConfigReadWriterMock{},
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
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			client := utils.NewTestClient(scheme, scenario.InitObjs...)
			phase, err := scenario.Reconciler.ReconcileRateLimitService(context.TODO(), client, scenario.ProductConfig)

			if err := scenario.Assert(client, phase, err); err != nil {
				t.Error(err)
			}
		})
	}
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

func TestRateLimitServiceReconciler_differentLimitSettings(t *testing.T) {
	type fields struct {
		Namespace       string
		RedisSecretName string
		Installation    *integreatlyv1alpha1.RHMI
		RateLimitConfig marin3rconfig.RateLimitConfig
		Config          *config.Marin3r
	}
	type args struct {
		redisLimits   []limitadorLimit
		currentLimits []limitadorLimit
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "test true when list size is different",
			args: args{
				redisLimits: []limitadorLimit{},
				currentLimits: []limitadorLimit{
					{
						Namespace: "test",
						MaxValue:  1,
					},
				},
			},
			want: true,
		},
		{
			name: "test true when list is different",
			args: args{
				redisLimits: []limitadorLimit{
					{
						Namespace: "test1",
						MaxValue:  1,
					},
				},
				currentLimits: []limitadorLimit{
					{
						Namespace: "test",
						MaxValue:  1,
					},
				},
			},
			want: true,
		},
		{
			name: "test slices are sorted by Namespace first for comparison",
			args: args{
				redisLimits: []limitadorLimit{
					{
						Namespace: "test",
						MaxValue:  1,
					},
					{
						Namespace: "test2",
						MaxValue:  12,
					},
				},
				currentLimits: []limitadorLimit{
					{
						Namespace: "test2",
						MaxValue:  12,
					},
					{
						Namespace: "test",
						MaxValue:  1,
					},
				},
			},
			want: false,
		},
		{
			name: "test slices are sorted by MaxValue if matching Namespace",
			args: args{
				redisLimits: []limitadorLimit{
					{
						Namespace: "test",
						MaxValue:  12,
					},
					{
						Namespace: "test",
						MaxValue:  1,
					},
				},
				currentLimits: []limitadorLimit{
					{
						Namespace: "test",
						MaxValue:  1,
					},
					{
						Namespace: "test",
						MaxValue:  12,
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RateLimitServiceReconciler{
				Namespace:       tt.fields.Namespace,
				RedisSecretName: tt.fields.RedisSecretName,
				Installation:    tt.fields.Installation,
				RateLimitConfig: tt.fields.RateLimitConfig,
			}
			if got := r.differentLimitSettings(tt.args.redisLimits, tt.args.currentLimits); got != tt.want {
				t.Errorf("differentLimitSettings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRateLimitServiceReconciler_getLimitadorSetting(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Namespace       string
		RedisSecretName string
		Installation    *integreatlyv1alpha1.RHMI
		RateLimitConfig marin3rconfig.RateLimitConfig
		PodExecutor     resources.PodExecutorInterface
	}
	type args struct {
		ctx    context.Context
		client k8sclient.Client
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []limitadorLimit
		wantErr bool
	}{
		{
			name: "test get rhoam limitator config",
			fields: fields{
				Installation: &integreatlyv1alpha1.RHMI{
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: string(integreatlyv1alpha1.InstallationTypeManagedApi),
					},
				},
				RateLimitConfig: marin3rconfig.RateLimitConfig{Unit: "second", RequestsPerUnit: 1},
			},
			want: []limitadorLimit{
				{
					Namespace: ratelimit.RateLimitDomain,
					MaxValue:  1,
					Seconds:   1,
					Conditions: []string{
						fmt.Sprintf("%s == %s", genericKey, ratelimit.RateLimitDescriptorValue),
					},
					Variables: []string{
						genericKey,
					},
				},
			},
		},
		{
			name: "test error get rhoam limitator config",
			fields: fields{
				Installation: &integreatlyv1alpha1.RHMI{
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: string(integreatlyv1alpha1.InstallationTypeManagedApi),
					},
				},
				RateLimitConfig: marin3rconfig.RateLimitConfig{Unit: "notUnit", RequestsPerUnit: 1},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "test get rhoam multitenant limitator config",
			args: args{
				ctx: context.TODO(),
				client: utils.NewTestClient(scheme, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      multitenantLimitConfigMap,
						Namespace: "test",
					},
					Data: map[string]string{
						multitenantRateLimit: "10",
					},
				}),
			},
			fields: fields{
				Namespace: "test",
				Installation: &integreatlyv1alpha1.RHMI{
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: string(integreatlyv1alpha1.InstallationTypeMultitenantManagedApi),
					},
				},
				RateLimitConfig: marin3rconfig.RateLimitConfig{Unit: "second", RequestsPerUnit: 1},
			},
			want: []limitadorLimit{
				{
					Namespace: ratelimit.RateLimitDomain,
					MaxValue:  1,
					Seconds:   1,
					Conditions: []string{
						fmt.Sprintf("%s == %s", genericKey, ratelimit.RateLimitDescriptorValue),
					},
					Variables: []string{
						genericKey,
					},
				},
				{
					Namespace: ratelimit.RateLimitDomain,
					MaxValue:  10,
					Seconds:   1,
					Conditions: []string{
						fmt.Sprintf("%s == %s", headerMatch, multitenantDescriptorValue),
					},
					Variables: []string{
						headerKey,
					},
				},
			},
		},
		{
			name: "test error get rhoam multitenant limitator config",
			args: args{
				ctx:    context.TODO(),
				client: utils.NewTestClient(scheme),
			},
			fields: fields{
				Namespace: "test",
				Installation: &integreatlyv1alpha1.RHMI{
					Spec: integreatlyv1alpha1.RHMISpec{
						Type: string(integreatlyv1alpha1.InstallationTypeMultitenantManagedApi),
					},
				},
				RateLimitConfig: marin3rconfig.RateLimitConfig{Unit: "notUnit", RequestsPerUnit: 1},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RateLimitServiceReconciler{
				Namespace:       tt.fields.Namespace,
				RedisSecretName: tt.fields.RedisSecretName,
				Installation:    tt.fields.Installation,
				RateLimitConfig: tt.fields.RateLimitConfig,
				PodExecutor:     tt.fields.PodExecutor,
			}
			got, err := r.getLimitadorSetting(tt.args.ctx, tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("getLimitadorSetting() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getLimitadorSetting() got = %v, want %v", got, tt.want)
			}
		})
	}
}
