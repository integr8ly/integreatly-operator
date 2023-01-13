package config

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/integr8ly/integreatly-operator/test/utils"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetAlertConfig(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []struct {
		Name        string
		InitialObjs []runtime.Object
		Namespace   string
		Assert      func(client.Client, map[string]*AlertConfig, error) error
	}{
		{
			Name:      "Success",
			Namespace: "redhat-test-operator",
			InitialObjs: []runtime.Object{
				&corev1.ConfigMap{
					ObjectMeta: v1.ObjectMeta{
						Name:      "rate-limit-alerts",
						Namespace: "redhat-test-operator",
					},
					Data: map[string]string{
						"alerts": `
							{
								"alert-1": {
									"type": "Threshold",
									"ruleName": "Rule1",
									"level": "warning",
									"period": "2h",
									"threshold": {
										"minRate": "80%",
										"maxRate": "90%"
									}
								}
							}
						`,
					},
				},
			},
			Assert: func(c client.Client, config map[string]*AlertConfig, err error) error {
				if err != nil {
					return fmt.Errorf("unexpected error: %v", err)
				}

				alertConfig, ok := config["alert-1"]
				if !ok {
					return fmt.Errorf("expected key alert-1 not found in resulting config")
				}

				maxRate := "90%"

				expectedConfig := &AlertConfig{
					RuleName: "Rule1",
					Level:    "warning",
					Threshold: &AlertThresholdConfig{
						MaxRate: &maxRate,
						MinRate: "80%",
					},
					Period: "2h",
					Type:   AlertTypeThreshold,
				}

				if !reflect.DeepEqual(alertConfig, expectedConfig) {
					return fmt.Errorf("obtained invalid config. Expected %v, but got %v", expectedConfig, alertConfig)
				}

				return nil
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			k8sClient := utils.NewTestClient(scheme, scenario.InitialObjs...)
			config, err := GetAlertConfig(context.TODO(), k8sClient, scenario.Namespace)

			if err := scenario.Assert(k8sClient, config, err); err != nil {
				t.Error(err)
			}
		})
	}
}
