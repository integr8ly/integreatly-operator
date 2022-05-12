package resources

import (
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
			Expected: "Installing",
		},
		{
			Name: "Upgrade installation",
			Input: struct {
				Version   string
				ToVersion string
			}{Version: "1.1.0", ToVersion: "1.2.0"},
			Expected: "Upgrading",
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
