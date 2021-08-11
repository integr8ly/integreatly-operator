package v1alpha1

import (
	"testing"
)

func TestRHMIInstallType(t *testing.T) {
	tests := []struct {
		name            string
		installType     InstallationType
		expectedOutcome bool
	}{
		{
			name:            "test that isRHMI returns true",
			installType:     InstallationTypeManaged,
			expectedOutcome: true,
		},
		{
			name:            "test that isRHMI returns false",
			installType:     InstallationTypeManagedApi,
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
		installType     InstallationType
		expectedOutcome bool
	}{
		{
			name:            "test that isRHOAM returns true",
			installType:     InstallationTypeManagedApi,
			expectedOutcome: true,
		},
		{
			name:            "test that isRHOAM returns false",
			installType:     InstallationTypeManaged,
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
		installType     InstallationType
		expectedOutcome bool
	}{
		{
			name:            "test that isMultitenant returns true",
			installType:     InstallationTypeManagedApi,
			expectedOutcome: true,
		},
		{
			name:            "test that isMultitenant returns false",
			installType:     InstallationTypeManaged,
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
		installType     InstallationType
		expectedOutcome bool
	}{
		{
			name:            "test that isMultitenant returns true",
			installType:     InstallationTypeMultitenantManagedApi,
			expectedOutcome: true,
		},
		{
			name:            "test that isMultitenant returns false",
			installType:     InstallationTypeManaged,
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

func TestManagedInstallType(t *testing.T) {
	tests := []struct {
		name            string
		installType     InstallationType
		expectedOutcome bool
	}{
		{
			name:            "test that installation is managed for managed-api",
			installType:     InstallationTypeManagedApi,
			expectedOutcome: true,
		},
		{
			name:            "test that installation is managed for multitenant managed-api",
			installType:     InstallationTypeMultitenantManagedApi,
			expectedOutcome: true,
		},
		{
			name:            "test that installation is managed",
			installType:     InstallationTypeManaged,
			expectedOutcome: true,
		},
		{
			name:            "test that installation is not managed",
			installType:     InstallationTypeSelfManaged,
			expectedOutcome: false,
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			v := IsManaged(c.installType)
			if v != c.expectedOutcome {
				t.Errorf("Outcome does not match expected value - got %v; expecting %v", v, c.expectedOutcome)
			}
		})
	}
}
