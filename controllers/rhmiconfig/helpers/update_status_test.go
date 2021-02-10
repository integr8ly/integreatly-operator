package helpers

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	rhmiconfigv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"

	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type scheduleScenario struct {
	name             string
	config           *rhmiconfigv1alpha1.RHMIConfig
	expectedSchedule *rhmiconfigv1alpha1.UpgradeSchedule
}

func makeScheduleScenario(scenario *scheduleScenario) struct {
	Name     string
	Config   *rhmiconfigv1alpha1.RHMIConfig
	Validate func(*testing.T, error, *rhmiconfigv1alpha1.RHMIConfig)
} {
	scenario.config.Name = "test-config"
	scenario.config.Namespace = "testing-namespaces-operator"

	return struct {
		Name     string
		Config   *rhmiconfigv1alpha1.RHMIConfig
		Validate func(*testing.T, error, *rhmiconfigv1alpha1.RHMIConfig)
	}{
		Name:   scenario.name,
		Config: scenario.config,
		Validate: func(t *testing.T, err error, config *rhmiconfigv1alpha1.RHMIConfig) {
			if err != nil {
				t.Errorf("Unexpected error occurred: %v", err)
			}
			if !reflect.DeepEqual(config.Status.Upgrade.Scheduled, scenario.expectedSchedule) {
				t.Errorf("Upgrade schedule different than expected")
			}
		},
	}
}

func TestUpdateStatus(t *testing.T) {
	targetVersion := "integreatly-operator-v2.3.0"
	scenarios := []struct {
		Name     string
		Config   *rhmiconfigv1alpha1.RHMIConfig
		Validate func(*testing.T, error, *rhmiconfigv1alpha1.RHMIConfig)
	}{
		{
			Name: "status updated with pending installplan",
			Config: &rhmiconfigv1alpha1.RHMIConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "testing-namespaces-operator",
				},
				Spec: rhmiconfigv1alpha1.RHMIConfigSpec{
					Maintenance: rhmiconfigv1alpha1.Maintenance{
						ApplyFrom: strings.ToLower(nowOffset(-1).Format("Mon 15:04")),
					},
					Upgrade: rhmiconfigv1alpha1.Upgrade{
						NotBeforeDays:      intPtr(8),
						WaitForMaintenance: boolPtr(true),
						Schedule:           boolPtr(true),
					},
				},
				Status: rhmiconfigv1alpha1.RHMIConfigStatus{
					UpgradeAvailable: &rhmiconfigv1alpha1.UpgradeAvailable{
						TargetVersion: targetVersion,
					},
				},
			},
			Validate: func(t *testing.T, err error, config *rhmiconfigv1alpha1.RHMIConfig) {
				if err != nil {
					t.Error("Expected no error, but got: " + err.Error())
				}
				expectedMaintenanceFrom := nowOffset(-1).Format("2-1-2006 15:04")
				if config.Status.Maintenance.ApplyFrom != expectedMaintenanceFrom {
					t.Errorf("expected maintenance from '%s', but got '%s'", expectedMaintenanceFrom, config.Status.Maintenance.ApplyFrom)
				}
				if config.Status.Maintenance.Duration != "6hrs" {
					t.Errorf("expected maintenance duration '6hrs' but got '%s'", config.Status.Maintenance.Duration)
				}
			},
		}, {
			Name: "status unchanged with no pending installplan",
			Config: &rhmiconfigv1alpha1.RHMIConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "testing-namespaces-operator",
				},
				Spec: rhmiconfigv1alpha1.RHMIConfigSpec{
					Maintenance: rhmiconfigv1alpha1.Maintenance{
						ApplyFrom: strings.ToLower(nowOffset(-1).Format("Mon 15:04")),
					},
					Upgrade: rhmiconfigv1alpha1.Upgrade{
						Schedule: boolPtr(true),
					},
				},
				Status: rhmiconfigv1alpha1.RHMIConfigStatus{
					UpgradeAvailable: &rhmiconfigv1alpha1.UpgradeAvailable{
						TargetVersion: targetVersion,
					},
				},
			},
			Validate: func(t *testing.T, err error, config *rhmiconfigv1alpha1.RHMIConfig) {
				if err != nil {
					t.Error("Expected no error, but got: " + err.Error())
				}

				expectedMaintenanceFrom := nowOffset(-1).Format("2-1-2006 15:04")
				if config.Status.Maintenance.ApplyFrom != expectedMaintenanceFrom {
					t.Errorf("expected maintenance from '%s', but got '%s'", expectedMaintenanceFrom, config.Status.Maintenance.ApplyFrom)
				}
				if config.Status.Maintenance.Duration != "6hrs" {
					t.Errorf("expected maintenance duration '6hrs' but got '%s'", config.Status.Maintenance.Duration)
				}
			},
		},
		makeScheduleScenario(&scheduleScenario{
			name: "do not wait for maintenance 0 days",
			config: &rhmiconfigv1alpha1.RHMIConfig{
				Spec: rhmiconfigv1alpha1.RHMIConfigSpec{
					Upgrade: rhmiconfigv1alpha1.Upgrade{
						NotBeforeDays:      intPtr(0),
						WaitForMaintenance: boolPtr(false),
						Schedule:           boolPtr(true),
					},
				},
				Status: rhmiconfigv1alpha1.RHMIConfigStatus{
					UpgradeAvailable: &rhmiconfigv1alpha1.UpgradeAvailable{
						TargetVersion: targetVersion,
						AvailableAt:   kubeNow(0),
					},
				},
			},
			expectedSchedule: &rhmiconfigv1alpha1.UpgradeSchedule{
				For: kubeNow(0).Format(rhmiconfigv1alpha1.DateFormat),
			},
		}),
		makeScheduleScenario(&scheduleScenario{
			name: "wait for maintenance 0 days",
			config: &rhmiconfigv1alpha1.RHMIConfig{
				Spec: rhmiconfigv1alpha1.RHMIConfigSpec{
					Maintenance: rhmiconfigv1alpha1.Maintenance{
						ApplyFrom: "Sun 00:00",
					},
					Upgrade: rhmiconfigv1alpha1.Upgrade{
						NotBeforeDays:      intPtr(0),
						WaitForMaintenance: boolPtr(true),
						Schedule:           boolPtr(true),
					},
				},
				Status: rhmiconfigv1alpha1.RHMIConfigStatus{
					UpgradeAvailable: &rhmiconfigv1alpha1.UpgradeAvailable{
						TargetVersion: targetVersion,
						AvailableAt:   kubeNow(0),
					},
				},
			},
			expectedSchedule: &rhmiconfigv1alpha1.UpgradeSchedule{
				For: time.
					Date(now().Year(), now().Month(), now().Day(), 0, 0, 0, 0, time.UTC).
					AddDate(0, 0, (7-int(now().Weekday()))%7).
					Format(rhmiconfigv1alpha1.DateFormat),
			},
		}),
		makeScheduleScenario(&scheduleScenario{
			name: "wait for maintenance, notBefore: 3 days before next window",
			config: &rhmiconfigv1alpha1.RHMIConfig{
				Spec: rhmiconfigv1alpha1.RHMIConfigSpec{
					Maintenance: rhmiconfigv1alpha1.Maintenance{
						ApplyFrom: strings.ToLower(time.Date(now().Year(), now().Month(), now().Day(), 0, 0, 0, 0, time.UTC).
							Add(6 * 24 * time.Hour).
							Format("Mon 15:04")),
					},
					Upgrade: rhmiconfigv1alpha1.Upgrade{
						WaitForMaintenance: boolPtr(true),
						NotBeforeDays:      intPtr(3),
						Schedule:           boolPtr(true),
					},
				},
				Status: rhmiconfigv1alpha1.RHMIConfigStatus{
					UpgradeAvailable: &rhmiconfigv1alpha1.UpgradeAvailable{
						TargetVersion: targetVersion,
						AvailableAt:   kubeNow(0),
					},
				},
			},
			expectedSchedule: &rhmiconfigv1alpha1.UpgradeSchedule{
				For: time.Date(now().Year(), now().Month(), now().Day(), 0, 0, 0, 0, time.UTC).Add(6 * 24 * time.Hour).
					Format(rhmiconfigv1alpha1.DateFormat),
			},
		}),
		makeScheduleScenario(&scheduleScenario{
			name: "wait for maintenance, notBefore: 3 days after next window",
			config: &rhmiconfigv1alpha1.RHMIConfig{
				Spec: rhmiconfigv1alpha1.RHMIConfigSpec{
					Maintenance: rhmiconfigv1alpha1.Maintenance{
						ApplyFrom: strings.ToLower(time.Date(now().Year(), now().Month(), now().Day(), 0, 0, 0, 0, time.UTC).
							Add(3 * 24 * time.Hour).
							Format("Mon 15:04")),
					},
					Upgrade: rhmiconfigv1alpha1.Upgrade{
						WaitForMaintenance: boolPtr(true),
						NotBeforeDays:      intPtr(6),
						Schedule:           boolPtr(true),
					},
				},
				Status: rhmiconfigv1alpha1.RHMIConfigStatus{
					UpgradeAvailable: &rhmiconfigv1alpha1.UpgradeAvailable{
						TargetVersion: targetVersion,
						AvailableAt:   kubeNow(0),
					},
				},
			},
			expectedSchedule: &rhmiconfigv1alpha1.UpgradeSchedule{
				For: time.Date(now().Year(), now().Month(), now().Day(), 0, 0, 0, 0, time.UTC).Add(10 * 24 * time.Hour).
					Format(rhmiconfigv1alpha1.DateFormat),
			},
		}),
		makeScheduleScenario(&scheduleScenario{
			name: "do not wait for maintenance, notBefore > 0",
			config: &rhmiconfigv1alpha1.RHMIConfig{
				Spec: rhmiconfigv1alpha1.RHMIConfigSpec{
					Upgrade: rhmiconfigv1alpha1.Upgrade{
						NotBeforeDays:      intPtr(3),
						WaitForMaintenance: boolPtr(false),
						Schedule:           boolPtr(true),
					},
				},
				Status: rhmiconfigv1alpha1.RHMIConfigStatus{
					UpgradeAvailable: &rhmiconfigv1alpha1.UpgradeAvailable{
						TargetVersion: targetVersion,
						AvailableAt:   kubeNow(-2),
					},
				},
			},
			expectedSchedule: &rhmiconfigv1alpha1.UpgradeSchedule{
				For: nowOffset(-2).Add(3 * 24 * time.Hour).Format(rhmiconfigv1alpha1.DateFormat),
			},
		}),
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			client := fake.NewFakeClientWithScheme(buildScheme(), scenario.Config)
			err := UpdateStatus(context.TODO(), client, scenario.Config)
			updatedConfig := &rhmiconfigv1alpha1.RHMIConfig{}
			client.Get(context.TODO(), k8sclient.ObjectKey{Name: "test-config", Namespace: "testing-namespaces-operator"}, updatedConfig)
			scenario.Validate(t, err, updatedConfig)
		})
	}
}

func TestGetWeeklyWindow(t *testing.T) {
	// Monday
	from := time.Date(2020, time.June, 1, 0, 0, 0, 0, time.UTC)

	// Test same day
	r, _, err := getWeeklyWindow(from, "Mon 00:00", time.Hour)
	if err != nil {
		t.Errorf("Error calculating weekly window for same day: %v", err)
	} else if r.Day() != from.Day() || r.Month() != from.Month() || r.Year() != from.Year() {
		t.Errorf("Expected result to be same day, got %s", r.Format(rhmiconfigv1alpha1.DateFormat))
	}

	// Test next day
	r, _, err = getWeeklyWindow(from, "Tue 00:00", time.Hour)
	if err != nil {
		t.Errorf("Error calculating weekly window for same day: %v", err)
	} else if r.Day() != from.Day()+1 || r.Month() != from.Month() || r.Year() != from.Year() {
		t.Errorf("Expected result to be next day, got %s", r.Format(rhmiconfigv1alpha1.DateFormat))
	}

	// Test day before
	r, _, err = getWeeklyWindow(from, "SuN 00:00", time.Hour)
	if err != nil {
		t.Errorf("Error calculating weekly window for same day: %v", err)
	} else if r.Day() != from.Day()+6 || r.Month() != from.Month() || r.Year() != from.Year() {
		t.Errorf("Expected result to be next Sunday, got %s", r.Format(rhmiconfigv1alpha1.DateFormat))
	}

	// Test 3 days after
	r, _, err = getWeeklyWindow(from, "Thu 02:00", time.Hour)
	if err != nil {
		t.Errorf("Error calculating weekly window for same day: %v", err)
	} else if r.Day() != from.Day()+3 || r.Month() != from.Month() || r.Year() != from.Year() {
		t.Errorf("Expected result to be Thursday, got %s", r.Format(rhmiconfigv1alpha1.DateFormat))
	}
}

func buildScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	rhmiconfigv1alpha1.SchemeBuilder.AddToScheme(scheme)
	olmv1alpha1.SchemeBuilder.AddToScheme(scheme)

	return scheme
}

func nowOffset(hours int) time.Time {
	now := now()
	return time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+hours, now.Minute(), now.Second(), 0, time.UTC)
}

func kubeNow(addHours int) metav1.Time {
	t := metav1.Time{}
	t.Time = now().Add(time.Duration(addHours) * time.Hour)
	return t
}

func now() time.Time {
	return time.Now().UTC()
}

func intPtr(value int) *int {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
