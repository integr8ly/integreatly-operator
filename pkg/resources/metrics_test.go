package resources

import (
	"github.com/prometheus/alertmanager/api/v2/models"
	"testing"
)

func TestInstallationState(t *testing.T) {
	type testScenario struct {
		Name  string
		Input struct {
			Version   string
			ToVersion string
		}
		Expected string
	}

	scenarios := []testScenario{
		{
			Name: "No version information is set",
			Input: struct {
				Version   string
				ToVersion string
			}{Version: "", ToVersion: ""},
			Expected: "Unknown State",
		},
		{
			Name: "Initial installation",
			Input: struct {
				Version   string
				ToVersion string
			}{Version: "", ToVersion: "1.1.0"},
			Expected: "Installation",
		},
		{
			Name: "Upgrade installation",
			Input: struct {
				Version   string
				ToVersion string
			}{Version: "1.1.0", ToVersion: "1.2.0"},
			Expected: "Upgrade",
		},
		{
			Name: "Installed state",
			Input: struct {
				Version   string
				ToVersion string
			}{Version: "1.1.0", ToVersion: ""},
			Expected: "Installed",
		},
	}

	for _, scenario := range scenarios {
		actual := InstallationState(scenario.Input.Version, scenario.Input.ToVersion)

		if actual != scenario.Expected {
			t.Fatalf("Test: %s; Status not equal to expected result, Expected: %s, Actual: %s", scenario.Name, scenario.Expected, actual)
		}
	}
}

func TestAlertMetric_ContainsName(t *testing.T) {
	type testScenario struct {
		Name     string
		Metric   AlertMetric
		Input    string
		Expected bool
	}
	scenarios := []testScenario{
		{
			Name:     "Has name",
			Metric:   AlertMetric{Name: "Contains Name"},
			Input:    "Contains Name",
			Expected: true,
		},
		{
			Name:     "Not got name",
			Metric:   AlertMetric{Name: "Contains Name"},
			Input:    "Does Not Contain Name",
			Expected: false,
		},
		{
			Name:     "No name is set",
			Metric:   AlertMetric{State: "Firing"},
			Input:    "Contains Name",
			Expected: false,
		},
	}

	for _, scenario := range scenarios {
		actual := scenario.Metric.ContainsName(scenario.Input)

		if actual != scenario.Expected {
			t.Fatalf("Test: %s; Alert Metric did not match the name; Expected: %t, Actual: %t", scenario.Name, scenario.Expected, actual)
		}

	}
}

func TestAlertMetric_ContainsSeverity(t *testing.T) {
	type testScenario struct {
		Name     string
		Metric   AlertMetric
		Input    string
		Expected bool
	}
	scenarios := []testScenario{
		{
			Name:     "Has severity",
			Metric:   AlertMetric{Severity: "High"},
			Input:    "High",
			Expected: true,
		},
		{
			Name:     "Not got severity",
			Metric:   AlertMetric{Severity: "High"},
			Input:    "Low",
			Expected: false,
		},
		{
			Name:     "No severity is set",
			Metric:   AlertMetric{State: "Firing"},
			Input:    "High",
			Expected: false,
		},
	}

	for _, scenario := range scenarios {
		actual := scenario.Metric.ContainsSeverity(scenario.Input)

		if actual != scenario.Expected {
			t.Fatalf("Test: %s; Alert Metric did not match the severity; Expected: %t, Actual: %t", scenario.Name, scenario.Expected, actual)
		}

	}
}

func TestAlertMetric_ContainsState(t *testing.T) {
	type testScenario struct {
		Name     string
		Metric   AlertMetric
		Input    string
		Expected bool
	}
	scenarios := []testScenario{
		{
			Name:     "Has State",
			Metric:   AlertMetric{State: "Firing"},
			Input:    "Firing",
			Expected: true,
		},
		{
			Name:     "Not got state",
			Metric:   AlertMetric{State: "Firing"},
			Input:    "Pending",
			Expected: false,
		},
		{
			Name:     "No state is set",
			Metric:   AlertMetric{Name: "No State"},
			Input:    "Firing",
			Expected: false,
		},
	}

	for _, scenario := range scenarios {
		actual := scenario.Metric.ContainsState(scenario.Input)

		if actual != scenario.Expected {
			t.Fatalf("Test: %s; Alert Metric did not match the state; Expected: %t, Actual: %t", scenario.Name, scenario.Expected, actual)
		}

	}
}

func TestAlertMetric_Contains(t *testing.T) {
	type input struct {
		Labels models.LabelSet `json:"labels"`
		State  string          `json:"state"`
	}

	type testScenario struct {
		Name     string
		Metric   AlertMetric
		Input    input
		Expected bool
	}

	scenarios := []testScenario{
		{
			Name: "has metric",
			Metric: AlertMetric{
				Name:     "Metric Exists",
				Severity: "High",
				State:    "Firing",
				Value:    0,
			},
			Input: input{
				Labels: models.LabelSet{
					"alertname": "Metric Exists",
					"severity":  "High",
				},
				State: "Firing",
			},
			Expected: true,
		},
		{
			Name: "Does not contain metric, missing name",
			Metric: AlertMetric{
				Name:     "Metric Exists",
				Severity: "High",
				State:    "Firing",
				Value:    0,
			},
			Input: input{
				Labels: models.LabelSet{
					"alertname": "Metric Missing",
					"severity":  "High",
				},
				State: "Firing",
			},
			Expected: false,
		},
		{
			Name: "Does not contain metric, missing severity",
			Metric: AlertMetric{
				Name:     "Metric Exists",
				Severity: "High",
				State:    "Firing",
				Value:    0,
			},
			Input: input{
				Labels: models.LabelSet{
					"alertname": "Metric Exists",
					"severity":  "Missing",
				},
				State: "Firing",
			},
			Expected: false,
		},
		{
			Name: "Does not contain metric, missing state",
			Metric: AlertMetric{
				Name:     "Metric Exists",
				Severity: "High",
				State:    "Firing",
				Value:    0,
			},
			Input: input{
				Labels: models.LabelSet{
					"alertname": "Metric Exists",
					"severity":  "High",
				},
				State: "Missing",
			},
			Expected: false,
		},
	}

	for _, scenario := range scenarios {
		actual := scenario.Metric.Contains(scenario.Input)

		if actual != scenario.Expected {
			t.Fatalf("Test: %s; Alert Metric did not match Metric; Expected: %t, Actual: %t", scenario.Name, scenario.Expected, actual)
		}

	}
}

func TestAlertMetrics_Contains(t *testing.T) {
	type input struct {
		Labels models.LabelSet `json:"labels"`
		State  string          `json:"state"`
	}

	type testScenario struct {
		Name     string
		Metrics  AlertMetrics
		Input    input
		Expected bool
	}

	scenarios := []testScenario{
		{
			Name: "Metric list has input",
			Metrics: AlertMetrics{Alerts: []AlertMetric{
				{
					Name:     "Metric Exists",
					Severity: "High",
					State:    "Firing",
					Value:    0,
				},
			}},
			Input: input{
				Labels: models.LabelSet{
					"alertname": "Metric Exists",
					"severity":  "High",
				},
				State: "Firing",
			},
			Expected: true,
		},
		{
			Name: "Metric list not got input",
			Metrics: AlertMetrics{Alerts: []AlertMetric{
				{
					Name:     "Metric Exists",
					Severity: "High",
					State:    "Firing",
					Value:    0,
				},
			}},
			Input: input{
				Labels: models.LabelSet{
					"alertname": "Metric Missing",
					"severity":  "High",
				},
				State: "Firing",
			},
			Expected: false,
		},
		{
			Name:    "Metric list is empty",
			Metrics: AlertMetrics{Alerts: []AlertMetric{}},
			Input: input{
				Labels: models.LabelSet{
					"alertname": "Metric Missing",
					"severity":  "High",
				},
				State: "Firing",
			},
			Expected: false,
		},
	}

	for _, scenario := range scenarios {
		actual := scenario.Metrics.Contains(scenario.Input)

		if actual != scenario.Expected {
			t.Fatalf("Test: %s; Alert Metrics do not contain Metric Input; Expected: %t, Actual: %t", scenario.Name, scenario.Expected, actual)
		}

	}
}
