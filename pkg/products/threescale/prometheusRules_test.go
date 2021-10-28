package threescale

import "testing"

func TestAlertThreeScaleContainerHighMemorySeverity(t *testing.T) {
	alert := alertThreeScaleContainerHighMemory("dummy", "dummy-namespace")
	severity := alert.Labels["severity"]

	if severity != "info" {
		t.Fatalf("severity level; Expected: info, Got: %v", severity)
	}
}
