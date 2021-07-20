package resources

import (
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
)

func TestRHMIInstallType(t *testing.T) {
	tests := []struct {
		name            string
		installType     integreatlyv1alpha1.InstallationType
		expectedOutcome bool
	}{
		{
			name:            "test that isRHMI returns true",
			installType:     "managed",
			expectedOutcome: true,
		},
		{
			name:            "test that isRHMI returns false",
			installType:     "managed-api",
			expectedOutcome: false,
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			v := IsRHMI(c.installType)
			if v != c.expectedOutcome {
				t.Errorf("Outcome does not match expected value - got %v; expecting %v", v, c.expectedOutcome)
			}
		})
	}
}

func TestRHOAMInstallType(t *testing.T) {
	tests := []struct {
		name            string
		installType     integreatlyv1alpha1.InstallationType
		expectedOutcome bool
	}{
		{
			name:            "test that isRHOAM returns true",
			installType:     "managed-api",
			expectedOutcome: true,
		},
		{
			name:            "test that isRHOAM returns false",
			installType:     "managed",
			expectedOutcome: false,
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			v := IsRHOAM(c.installType)
			if v != c.expectedOutcome {
				t.Errorf("Outcome does not match expected value - got %v; expecting %v", v, c.expectedOutcome)
			}
		})
	}
}

func TestIsMultitenant(t *testing.T) {
	tests := []struct {
		name            string
		installType     integreatlyv1alpha1.InstallationType
		expectedOutcome bool
	}{
		{
			name:            "test that isMultitenant returns true",
			installType:     "multitenant-managed-api",
			expectedOutcome: true,
		},
		{
			name:            "test that isMultitenant returns false",
			installType:     "managed",
			expectedOutcome: false,
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			v := isMultitenant(c.installType)
			if v != c.expectedOutcome {
				t.Errorf("Outcome does not match expected value - got %v; expecting %v", v, c.expectedOutcome)
			}
		})
	}
}

func TestRHOAMMultitenant(t *testing.T) {
	tests := []struct {
		name            string
		installType     integreatlyv1alpha1.InstallationType
		expectedOutcome bool
	}{
		{
			name:            "test that isMultitenant returns true",
			installType:     "multitenant-managed-api",
			expectedOutcome: true,
		},
		{
			name:            "test that isMultitenant returns false",
			installType:     "managed",
			expectedOutcome: false,
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			v := IsRHOAMMultitenant(c.installType)
			if v != c.expectedOutcome {
				t.Errorf("Outcome does not match expected value - got %v; expecting %v", v, c.expectedOutcome)
			}
		})
	}
}
