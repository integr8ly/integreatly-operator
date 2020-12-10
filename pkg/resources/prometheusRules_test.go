package resources

import (
	"context"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"reflect"
	"testing"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcileAlerts(t *testing.T) {
	type testScenario struct {
		Name          string
		Installation  *integreatlyv1alpha1.RHMI
		ExistingRules []*monitoringv1.PrometheusRule
		Alerts        []AlertConfiguration
		Assertion     func(k8sclient.Client, integreatlyv1alpha1.StatusPhase, error) error
	}

	scenarios := []testScenario{
		// Verify that the reconciler creates the alerts when they don't exist
		{
			Name:          "Create alerts",
			ExistingRules: []*monitoringv1.PrometheusRule{},
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: v1.ObjectMeta{
					Name:      "rhmi",
					Namespace: "testing-namespaces-test",
				},
			},
			Alerts: []AlertConfiguration{
				{
					AlertName: "test-alert",
					GroupName: "test-group",
					Namespace: "testing-namespaces-test",
					Rules:     rules,
				},
			},
			Assertion: func(client k8sclient.Client, phase integreatlyv1alpha1.StatusPhase, err error) error {
				if err != nil {
					return fmt.Errorf("unexpected error: %v", err)
				}

				if phase != integreatlyv1alpha1.PhaseCompleted {
					return fmt.Errorf("expected phase to be %s, got %s",
						integreatlyv1alpha1.PhaseCompleted, phase)
				}

				rule := &monitoringv1.PrometheusRule{}
				if err := client.Get(context.TODO(), k8sclient.ObjectKey{
					Name:      "test-alert",
					Namespace: "testing-namespaces-test",
				}, rule); err != nil {
					return fmt.Errorf("error retrieving rule: %v", err)
				}

				if rule.Spec.Groups[0].Name != "test-group" {
					return fmt.Errorf("expected group name to be test-group, got %s",
						rule.Spec.Groups[0].Name)
				}

				if !reflect.DeepEqual(rule.Spec.Groups[0].Rules, rules) {
					return fmt.Errorf("rules for test-alert differ")
				}

				return nil
			},
		},

		// Verify that the reconciler deletes the alerts when the installation
		// is marked for deletion
		{
			Name: "Delete alerts",
			ExistingRules: []*monitoringv1.PrometheusRule{
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test-alert",
						Namespace: "testing-namespaces-test",
					},
					Spec: monitoringv1.PrometheusRuleSpec{
						Groups: []monitoringv1.RuleGroup{
							{
								Name:  "test-group",
								Rules: rules,
							},
						},
					},
				},
				existingRules,
			},
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: v1.ObjectMeta{
					Name:              "rhmi",
					Namespace:         "testing-namespaces-test",
					DeletionTimestamp: now(),
				},
			},
			Alerts: []AlertConfiguration{
				{
					AlertName: "test-alert",
					GroupName: "test-group",
					Namespace: "testing-namespaces-test",
					Rules:     rules,
				},
			},
			Assertion: func(client k8sclient.Client, phase integreatlyv1alpha1.StatusPhase, err error) error {
				// Assert there was no error
				if err != nil {
					return fmt.Errorf("unexpected error: %v", err)
				}

				if phase != integreatlyv1alpha1.PhaseCompleted {
					return fmt.Errorf("expected phase to be %s, got %s",
						integreatlyv1alpha1.PhaseCompleted, phase)
				}

				// Assert that the rule was deleted
				deletedRule := &monitoringv1.PrometheusRule{}
				err = client.Get(context.TODO(), k8sclient.ObjectKey{
					Name:      "test-alert",
					Namespace: "testing-namespaces-test",
				}, deletedRule)
				if err == nil {
					return fmt.Errorf("expected an error retrieving deleted rule")
				}

				if !errors.IsNotFound(err) {
					return fmt.Errorf("expected error to be not found, got %v", err)
				}

				// Assert that the existing rule is unmodified
				existingRule := &monitoringv1.PrometheusRule{}
				objectKey, _ := k8sclient.ObjectKeyFromObject(existingRules)
				if err := client.Get(context.TODO(), objectKey, existingRule); err != nil {
					return fmt.Errorf("unexpected error retrieving existing rule: %v", err)
				}

				if !reflect.DeepEqual(existingRule.Spec.Groups[0].Rules, existingRules.Spec.Groups[0].Rules) {
					return fmt.Errorf("existing rule has differing values")
				}

				return nil
			},
		},
	}

	for _, scenario := range scenarios {
		scheme, err := buildSchemePrometheusRules()
		if err != nil {
			t.Errorf("error building scheme: %v", err)
			continue
		}

		client := fake.NewFakeClientWithScheme(scheme, scenario.Installation)
		for _, rule := range scenario.ExistingRules {
			client.Create(context.TODO(), rule)
		}

		alertReconciler := &AlertReconcilerImpl{
			ProductName:  "Test",
			Alerts:       scenario.Alerts,
			Installation: scenario.Installation,
			Log:          getLogger(),
		}

		phase, err := alertReconciler.ReconcileAlerts(context.TODO(), client)

		if assertionError := scenario.Assertion(client, phase, err); assertionError != nil {
			t.Errorf("%s failed: %v", scenario.Name, assertionError)
		}
	}
}

func buildSchemePrometheusRules() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()

	schemeBuilder := runtime.NewSchemeBuilder(
		monitoringv1.AddToScheme,
		integreatlyv1alpha1.SchemeBuilder.AddToScheme,
	)

	err := schemeBuilder.AddToScheme(scheme)
	return scheme, err
}

var (
	rules = []monitoringv1.Rule{
		{
			Alert: "TestRule",
			Annotations: map[string]string{
				"test": "true",
			},
			Expr:   intstr.FromString("test expr"),
			For:    "5m",
			Labels: map[string]string{"severity": "test"},
		},
	}

	existingRules = &monitoringv1.PrometheusRule{
		ObjectMeta: v1.ObjectMeta{
			Name:      "existing-rules",
			Namespace: "testing-namespaces-other",
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name: "existing-group",
					Rules: []monitoringv1.Rule{
						{
							Alert: "ExistingRule",
							Expr:  intstr.FromString("test existing expr"),
						},
					},
				},
			},
		},
	}
)

func now() *v1.Time {
	now := v1.Now()
	return &now
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{})
}
