package resources

import (
	"fmt"
	"time"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	upv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
)

func IsInProw(inst *integreatlyv1alpha1.RHMI) bool {
	annotationMap := inst.GetObjectMeta().GetAnnotations()
	isInProw, ok := annotationMap["in_prow"]
	if ok && isInProw == "true" {
		return true
	}
	return false
}

func IsSkipFinalDBSnapshots(inst *integreatlyv1alpha1.RHMI) bool {
	annotationMap := inst.GetObjectMeta().GetAnnotations()
	skipFinalDBSnapshots, ok := annotationMap["skip_final_db_snapshots"]
	if ok && skipFinalDBSnapshots == "true" {
		return true
	}
	return false
}

// DurationPtr creates a pointer to an OBO (monv1) Duration from a string like "5m".
// Validates the string with time.ParseDuration and panics if invalid.
func DurationPtr(durationStr string) *monv1.Duration {
	if _, err := time.ParseDuration(durationStr); err != nil {
		panic(fmt.Sprintf("Invalid duration string provided to DurationPtr: %s. Error: %v", durationStr, err))
	}
	d := monv1.Duration(durationStr)
	return &d
}

// UpDurationPtr creates a pointer to an upstream prom-operator Duration (monitoring/v1).
// Validates the string with time.ParseDuration and panics if invalid.
func UpDurationPtr(durationStr string) *upv1.Duration {
	if _, err := time.ParseDuration(durationStr); err != nil {
		panic(fmt.Sprintf("Invalid duration string provided to UpDurationPtr: %s. Error: %v", durationStr, err))
	}
	d := upv1.Duration(durationStr)
	return &d
}
