package config

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetRateLimitConfig(t *testing.T) {
	scheme := testScheme()

	scenarios := []struct {
		Name        string
		InitialObjs []runtime.Object
		Namespace   string
		Assert      func(client.Client, *RateLimitConfig, error) error
	}{
		{
			Name:      "Success",
			Namespace: "redhat-test-operator",
			InitialObjs: []runtime.Object{
				&corev1.ConfigMap{
					ObjectMeta: v1.ObjectMeta{
						Name:      "sku-limits-managed-api-service",
						Namespace: "redhat-test-operator",
					},
					Data: map[string]string{
						"rate_limit": `
						{
							"RHOAM SERVICE SKU": {
								"unit": "minute",
								"requests_per_unit": 42
							}
						}
						`,
					},
				},
			},
			Assert: func(c client.Client, config *RateLimitConfig, err error) error {
				if err != nil {
					return fmt.Errorf("Unexpected error: %v", err)
				}

				expectedConfig := &RateLimitConfig{
					Unit:            "minute",
					RequestsPerUnit: 42,
				}

				if !reflect.DeepEqual(config, expectedConfig) {
					return fmt.Errorf("Obtained invalid config. Expected %v, but got %v", expectedConfig, config)
				}

				return nil
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			client := fake.NewFakeClientWithScheme(scheme, scenario.InitialObjs...)
			config, err := GetRateLimitConfig(context.TODO(), client, scenario.Namespace)

			if err := scenario.Assert(client, config, err); err != nil {
				t.Error(err)
			}
		})
	}
}

func testScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	return scheme
}
