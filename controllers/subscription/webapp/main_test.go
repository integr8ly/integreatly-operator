package webapp

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	solutionExplorerv1alpha1 "github.com/integr8ly/integreatly-operator/apis-products/tutorial-web-app-operator/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/products/solutionexplorer"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func buildScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	solutionExplorerv1alpha1.SchemeBuilder.AddToScheme(scheme)

	return scheme
}

func TestNotifyUpgrade(t *testing.T) {
	type notifyUpgradeScenario struct {
		name               string
		config             *integreatlyv1alpha1.RHMIConfig
		version            string
		isServiceAffecting bool
		webapp             *solutionExplorerv1alpha1.WebApp
		assertion          func(integreatlyv1alpha1.StatusPhase, error, *solutionExplorerv1alpha1.WebApp) error
	}

	scenarios := []*notifyUpgradeScenario{
		{
			name: "Non existent webapp",
			config: &integreatlyv1alpha1.RHMIConfig{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "test-namespaces-webapp",
				},
			},
			isServiceAffecting: true,
			version:            "2.3.0",
			webapp:             nil,
			// Assert that there's no error and the returned phase is "in progress"
			assertion: func(phase integreatlyv1alpha1.StatusPhase, err error, _ *solutionExplorerv1alpha1.WebApp) error {
				if err != nil {
					return err
				}
				if phase != integreatlyv1alpha1.PhaseInProgress {
					return fmt.Errorf("Expected phase to be %s, got %s",
						integreatlyv1alpha1.PhaseInProgress,
						phase,
					)
				}

				return nil
			},
		},
		{
			name: "Upgrade data added",
			config: &integreatlyv1alpha1.RHMIConfig{
				ObjectMeta: v1.ObjectMeta{
					Namespace: "test-namespaces-webapp",
				},
				Status: integreatlyv1alpha1.RHMIConfigStatus{
					Upgrade: integreatlyv1alpha1.RHMIConfigStatusUpgrade{
						Scheduled: &integreatlyv1alpha1.UpgradeSchedule{
							For: "13 Jul 2020 00:00",
						},
					},
				},
			},
			isServiceAffecting: true,
			version:            "2.3.0",
			webapp: &solutionExplorerv1alpha1.WebApp{
				ObjectMeta: v1.ObjectMeta{
					Name:      solutionexplorer.DefaultName,
					Namespace: "test-namespaces-solution-explorer",
				},
				Spec: solutionExplorerv1alpha1.WebAppSpec{
					Template: solutionExplorerv1alpha1.WebAppTemplate{
						Parameters: map[string]string{
							"EXISTING_PARAM": "foo",
						},
					},
				},
			},
			assertion: func(phase integreatlyv1alpha1.StatusPhase, err error, webapp *solutionExplorerv1alpha1.WebApp) error {
				// Assert there's no error
				if err != nil {
					return err
				}

				// Assert the phase completed
				if phase != integreatlyv1alpha1.PhaseCompleted {
					return fmt.Errorf("Expected phase to be %s, got %s",
						integreatlyv1alpha1.PhaseCompleted,
						phase,
					)
				}

				// Assert no existing parameter was overriden
				if existingParam, ok := webapp.Spec.Template.Parameters["EXISTING_PARAM"]; !ok || existingParam != "foo" {
					return fmt.Errorf("Error asserting existing parameter on EXISTING_PARAMETER on WebApp was not modified. Expected foo, got %s", existingParam)
				}

				// Assert the new parameter was added
				upgradeDataString, ok := webapp.Spec.Template.Parameters[solutionexplorer.ParamUpgradeData]
				if !ok {
					return fmt.Errorf("UPGRADE_DATA parameter not found on WebApp")
				}

				// Unmarshal the parameter and assert the the value is correct
				upgradeDataValue := &upgradeData{}
				if err := json.Unmarshal([]byte(upgradeDataString), upgradeDataValue); err != nil {
					return fmt.Errorf("Failed to unmarshall upgrade data parameter value: %v", err)
				}

				expectedUpgradeData := &upgradeData{
					ScheduledFor:       "13 Jul 2020 00:00",
					Version:            "2.3.0",
					IsServiceAffecting: true,
				}

				if !reflect.DeepEqual(upgradeDataValue, expectedUpgradeData) {
					return fmt.Errorf("Unexpected value for upgrade data. Expected %v, got %v", expectedUpgradeData, upgradeDataValue)
				}

				return nil
			},
		},
	}

	scheme := buildScheme()

	for _, scenario := range scenarios {
		objects := make([]runtime.Object, 0, 2)
		if scenario.config != nil {
			objects = append(objects, scenario.config)
		}
		if scenario.webapp != nil {
			objects = append(objects, scenario.webapp)
		}

		client := fake.NewFakeClientWithScheme(scheme, objects...)
		notifier := NewUpgradeNotifierWithClient(context.TODO(), client)

		phase, err := notifier.NotifyUpgrade(scenario.config, scenario.version, scenario.isServiceAffecting)

		var webapp *solutionExplorerv1alpha1.WebApp
		if scenario.webapp != nil {
			webapp = scenario.webapp.DeepCopy()
			client.Get(context.TODO(), k8sclient.ObjectKey{
				Name:      webapp.Name,
				Namespace: webapp.Namespace,
			}, webapp)
		}

		if err := scenario.assertion(phase, err, webapp); err != nil {
			t.Errorf("Unexpected result for scenario \"%s\": %v",
				scenario.name, err)
		}
	}
}

func TestClearNotification(t *testing.T) {
	type clearNotificationScenario struct {
		name      string
		webapp    *solutionExplorerv1alpha1.WebApp
		assertion func(error, *solutionExplorerv1alpha1.WebApp) error
		nsPrefix  string
	}

	scenarios := []*clearNotificationScenario{
		{
			name:   "Unexisting WebApp doesn't return an error",
			webapp: nil,
			assertion: func(err error, _ *solutionExplorerv1alpha1.WebApp) error {
				if err != nil {
					return err
				}

				return nil
			},
			nsPrefix: "testing-namespaces-",
		},
		{
			name: "Upgrade parameter is set to null",
			webapp: &solutionExplorerv1alpha1.WebApp{
				ObjectMeta: v1.ObjectMeta{
					Name:      solutionexplorer.DefaultName,
					Namespace: "testing-namespaces-solution-explorer",
				},
				Spec: solutionExplorerv1alpha1.WebAppSpec{
					Template: solutionExplorerv1alpha1.WebAppTemplate{
						Parameters: map[string]string{
							"EXISTING_PARAMETER":              "foo",
							solutionexplorer.ParamUpgradeData: "{...}",
						},
					},
				},
			},
			assertion: func(err error, webapp *solutionExplorerv1alpha1.WebApp) error {
				if err != nil {
					return err
				}

				if existingParameter, ok := webapp.Spec.Template.Parameters["EXISTING_PARAMETER"]; !ok || existingParameter != "foo" {
					return fmt.Errorf("Existing parameter was altered. Expected foo, got %s", existingParameter)
				}

				upgradeData, ok := webapp.Spec.Template.Parameters[solutionexplorer.ParamUpgradeData]
				if !ok {
					return fmt.Errorf("Expected upgrade data parameter to be set to null, but was removed")
				}
				if upgradeData != "null" {
					return fmt.Errorf("Expected upgrade data parameter to be set to null, but was set to %s instead", upgradeData)
				}

				return nil
			},
			nsPrefix: "testing-namespaces-",
		},
	}

	scheme := buildScheme()

	for _, scenario := range scenarios {
		var objects []runtime.Object
		if scenario.webapp != nil {
			objects = []runtime.Object{scenario.webapp}
		} else {
			objects = []runtime.Object{}
		}

		client := fake.NewFakeClientWithScheme(scheme, objects...)
		notifier := NewUpgradeNotifierWithClient(context.TODO(), client)

		err := notifier.ClearNotification(scenario.nsPrefix)

		var webapp *solutionExplorerv1alpha1.WebApp
		if scenario.webapp != nil {
			webapp = scenario.webapp.DeepCopy()
			client.Get(context.TODO(), k8sclient.ObjectKey{
				Name:      webapp.Name,
				Namespace: webapp.Namespace,
			}, webapp)
		}

		if err := scenario.assertion(err, webapp); err != nil {
			t.Errorf("Unexpected result for scenario \"%s\": %v",
				scenario.name,
				err,
			)
		}
	}
}
