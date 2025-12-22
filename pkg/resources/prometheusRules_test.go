package resources

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/utils"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/types"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestReconcileAlerts(t *testing.T) {
	type testScenario struct {
		Name           string
		Installation   *integreatlyv1alpha1.RHMI
		ExistingRules  []*monv1.PrometheusRule
		Alerts         []AlertConfiguration
		AlertsToRemove []AlertConfiguration
		Assertion      func(k8sclient.Client, integreatlyv1alpha1.StatusPhase, error) error
		Client         client.SigsClientInterface
	}

	var genericError = fmt.Errorf("some error")

	scenarios := []testScenario{
		// Verify that the reconciler creates the alerts when they don't exist
		{
			Name:          "Create alerts",
			ExistingRules: []*monv1.PrometheusRule{},
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: v1.ObjectMeta{
					Name:      "rhoam",
					Namespace: "testing-namespaces-test",
				},
			},
			Alerts: alerts,
			Assertion: func(client k8sclient.Client, phase integreatlyv1alpha1.StatusPhase, err error) error {
				if err != nil {
					return fmt.Errorf("unexpected error: %v", err)
				}

				if phase != integreatlyv1alpha1.PhaseCompleted {
					return fmt.Errorf("expected phase to be %s, got %s",
						integreatlyv1alpha1.PhaseCompleted, phase)
				}

				rule := &monv1.PrometheusRule{}
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
			ExistingRules: []*monv1.PrometheusRule{
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test-alert",
						Namespace: "testing-namespaces-test",
					},
					Spec: monv1.PrometheusRuleSpec{
						Groups: []monv1.RuleGroup{
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
					Name:      "rhoam",
					Namespace: "testing-namespaces-test",
				},
			},
			Alerts: alerts,
			Assertion: func(client k8sclient.Client, phase integreatlyv1alpha1.StatusPhase, err error) error {
				if err := assertPhaseCompleteAndAlertIsNotFound(client, phase, err); err != nil {
					return err
				}

				// Assert that the existing rule is unmodified
				existingRule := &monv1.PrometheusRule{}
				objectKey := k8sclient.ObjectKeyFromObject(existingRules)
				if err := client.Get(context.TODO(), objectKey, existingRule); err != nil {
					return fmt.Errorf("unexpected error retrieving existing rule: %v", err)
				}

				if !reflect.DeepEqual(existingRule.Spec.Groups[0].Rules, existingRules.Spec.Groups[0].Rules) {
					return fmt.Errorf("existing rule has differing values")
				}

				return nil
			},
		},
		{
			Name:          "Alerts marked for removal are deleted as part of reconcile",
			ExistingRules: []*monv1.PrometheusRule{},
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: v1.ObjectMeta{
					Name:      "rhoam",
					Namespace: "testing-namespaces-test",
				},
			},
			Alerts:         alerts,
			AlertsToRemove: alerts,
			Assertion:      assertPhaseCompleteAndAlertIsNotFound,
		},
		{
			Name:           "Reconcile continues if alerts marked for deletion are not present",
			ExistingRules:  []*monv1.PrometheusRule{},
			Installation:   &integreatlyv1alpha1.RHMI{},
			Alerts:         []AlertConfiguration{},
			AlertsToRemove: alerts,
			Assertion:      assertPhaseCompleteAndAlertIsNotFound,
		},
		{
			Name:          "Phase failed deleting alerts during reconcile",
			ExistingRules: []*monv1.PrometheusRule{},
			Installation:  &integreatlyv1alpha1.RHMI{},
			Alerts:        []AlertConfiguration{},
			Client: &client.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
					return genericError
				},
			},
			AlertsToRemove: alerts,
			Assertion:      assertErrorAndPhaseFailed,
		},
		{
			Name:          "Phase failed deleting alerts due to error from getting alerts",
			ExistingRules: []*monv1.PrometheusRule{},
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: v1.ObjectMeta{},
			},
			Alerts: []AlertConfiguration{},
			Client: &client.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
					return genericError
				},
			},
			AlertsToRemove: alerts,
			Assertion:      assertErrorAndPhaseFailed,
		},
		{
			Name:          "Phase failed deleting alerts due to error on deletion",
			ExistingRules: []*monv1.PrometheusRule{},
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: v1.ObjectMeta{},
			},
			Alerts: []AlertConfiguration{},
			Client: &client.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
					return nil
				},
				DeleteFunc: func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.DeleteOption) error {
					return genericError
				},
			},
			AlertsToRemove: alerts,
			Assertion:      assertErrorAndPhaseFailed,
		},
		{
			Name:          "Phase failed creating alerts during reconcile",
			ExistingRules: []*monv1.PrometheusRule{},
			Installation:  &integreatlyv1alpha1.RHMI{},
			Client: &client.SigsClientInterfaceMock{
				GetFunc: func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
					return nil
				},
				UpdateFunc: func(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.UpdateOption) error {
					return genericError
				},
			},
			Alerts:    alerts,
			Assertion: assertErrorAndPhaseFailed,
		},
	}

	for _, scenario := range scenarios {
		scheme, err := utils.NewTestScheme()
		if err != nil {
			t.Errorf("error building scheme: %v", err)
			continue
		}

		serverClient := utils.NewTestClient(scheme, scenario.Installation)
		for _, rule := range scenario.ExistingRules {
			if err := serverClient.Create(context.TODO(), rule); err != nil {
				t.Errorf("Failed to create alert for test: %s", scenario.Name)
			}
		}

		alertReconciler := &AlertReconcilerImpl{
			ProductName:   "Test",
			Alerts:        scenario.Alerts,
			Installation:  scenario.Installation,
			Log:           getLogger(),
			RemovedAlerts: scenario.AlertsToRemove,
		}

		// Allow overriding client if defined in the scenario
		if scenario.Client != nil {
			serverClient = scenario.Client
		}

		phase, err := alertReconciler.ReconcileAlerts(context.TODO(), serverClient)

		if assertionError := scenario.Assertion(serverClient, phase, err); assertionError != nil {
			t.Errorf("%s failed: %v", scenario.Name, assertionError)
		}
	}
}

func assertErrorAndPhaseFailed(_ k8sclient.Client, phase integreatlyv1alpha1.StatusPhase, err error) error {
	if err == nil {
		return fmt.Errorf("expected error but got none")
	}

	if phase != integreatlyv1alpha1.PhaseFailed {
		return fmt.Errorf("expected phase to be %s, got %s",
			integreatlyv1alpha1.PhaseFailed, phase)
	}

	return nil
}

func assertPhaseCompleteAndAlertIsNotFound(client k8sclient.Client, phase integreatlyv1alpha1.StatusPhase, err error) error {
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

	return nil
}

var (
	alerts = []AlertConfiguration{
		{
			AlertName: "test-alert",
			GroupName: "test-group",
			Namespace: "testing-namespaces-test",
			Rules:     rules,
		},
	}
	rules = []monv1.Rule{
		{
			Alert: "TestRule",
			Annotations: map[string]string{
				"test": "true",
			},
			Expr:   intstr.FromString("test expr"),
			For:    DurationPtr("5m"),
			Labels: map[string]string{"severity": "test"},
		},
	}

	existingRules = &monv1.PrometheusRule{
		ObjectMeta: v1.ObjectMeta{
			Name:      "existing-rules",
			Namespace: "testing-namespaces-other",
		},
		Spec: monv1.PrometheusRuleSpec{
			Groups: []monv1.RuleGroup{
				{
					Name: "existing-group",
					Rules: []monv1.Rule{
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

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{})
}
