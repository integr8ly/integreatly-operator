package rhmiConfigs

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/integr8ly/integreatly-operator/version"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"

	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	defaultNamespace = "redhat-rhmi-operator"
)

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
}

func buildScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	olmv1alpha1.SchemeBuilder.AddToScheme(scheme)

	return scheme
}

func nowOffset(hours int) time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+hours, now.Minute(), now.Second(), 0, time.UTC)
}

type scheduleScenario struct {
	name             string
	config           *integreatlyv1alpha1.RHMIConfig
	expectedSchedule *integreatlyv1alpha1.UpgradeSchedule
}

func makeScheduleScenario(scenario *scheduleScenario) struct {
	Name        string
	Config      *integreatlyv1alpha1.RHMIConfig
	InstallPlan *olmv1alpha1.InstallPlan
	Validate    func(*testing.T, error, *integreatlyv1alpha1.RHMIConfig, *olmv1alpha1.InstallPlan)
} {
	scenario.config.Name = "test-config"
	scenario.config.Namespace = "redhat-rhmi-operator"

	return struct {
		Name        string
		Config      *integreatlyv1alpha1.RHMIConfig
		InstallPlan *olmv1alpha1.InstallPlan
		Validate    func(*testing.T, error, *integreatlyv1alpha1.RHMIConfig, *olmv1alpha1.InstallPlan)
	}{
		Name:   scenario.name,
		Config: scenario.config,
		InstallPlan: &olmv1alpha1.InstallPlan{
			ObjectMeta: metav1.ObjectMeta{
				CreationTimestamp: metav1.Time{Time: nowOffset(-2)},
			},
		},
		Validate: func(t *testing.T, err error, config *integreatlyv1alpha1.RHMIConfig, plan *olmv1alpha1.InstallPlan) {
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
	scenarios := []struct {
		Name        string
		Config      *integreatlyv1alpha1.RHMIConfig
		InstallPlan *olmv1alpha1.InstallPlan
		Validate    func(*testing.T, error, *integreatlyv1alpha1.RHMIConfig, *olmv1alpha1.InstallPlan)
	}{
		{
			Name: "status updated when pending installplan exists",
			Config: &integreatlyv1alpha1.RHMIConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "redhat-rhmi-operator",
				},
				Spec: integreatlyv1alpha1.RHMIConfigSpec{
					Maintenance: integreatlyv1alpha1.Maintenance{
						ApplyFrom: strings.ToLower(nowOffset(-1).Format("Mon 15:04")),
					},
					Upgrade: integreatlyv1alpha1.Upgrade{
						NotBeforeDays:      "8",
						WaitForMaintenance: "true",
					},
				},
			},
			InstallPlan: &olmv1alpha1.InstallPlan{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{Time: nowOffset(-2)},
				},
			},
			Validate: func(t *testing.T, err error, config *integreatlyv1alpha1.RHMIConfig, plan *olmv1alpha1.InstallPlan) {
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
				expectedUpgradeWindow := time.Now().Format("2 Jan 2006") + " - " + time.Now().Add((time.Hour*24)*14).Format("2 Jan 2006")
				if config.Status.Upgrade.Window != expectedUpgradeWindow {
					t.Errorf("Expected upgrade window '%s', got: '%s'", expectedUpgradeWindow, config.Status.Upgrade.Window)
				}
			},
		}, {
			Name: "status unchanged with no pending installplan",
			Config: &integreatlyv1alpha1.RHMIConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "redhat-rhmi-operator",
				},
				Spec: integreatlyv1alpha1.RHMIConfigSpec{
					Maintenance: integreatlyv1alpha1.Maintenance{
						ApplyFrom: strings.ToLower(nowOffset(-1).Format("Mon 15:04")),
					},
				},
				Status: integreatlyv1alpha1.RHMIConfigStatus{
					Upgrade: integreatlyv1alpha1.RHMIConfigStatusUpgrade{
						Scheduled: &integreatlyv1alpha1.UpgradeSchedule{},
					},
				},
			},
			InstallPlan: &olmv1alpha1.InstallPlan{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{Time: nowOffset(-2)},
				},
				Spec: olmv1alpha1.InstallPlanSpec{
					Approved: true,
				},
			},
			Validate: func(t *testing.T, err error, config *integreatlyv1alpha1.RHMIConfig, plan *olmv1alpha1.InstallPlan) {
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
				expectedUpgradeWindow := ""
				if config.Status.Upgrade.Window != expectedUpgradeWindow {
					t.Errorf("Expected upgrade window '%s', got: '%s'", expectedUpgradeWindow, config.Status.Upgrade.Window)
				}
			},
		},
		makeScheduleScenario(&scheduleScenario{
			name: "do not wait for maintenance 0 days",
			config: &integreatlyv1alpha1.RHMIConfig{
				Spec: integreatlyv1alpha1.RHMIConfigSpec{
					Upgrade: integreatlyv1alpha1.Upgrade{
						NotBeforeDays:      "0",
						WaitForMaintenance: "false",
					},
				},
			},
			expectedSchedule: &integreatlyv1alpha1.UpgradeSchedule{
				For: nowOffset(-2).Format(integreatlyv1alpha1.DateFormat),
			},
		}),
		makeScheduleScenario(&scheduleScenario{
			name: "wait for maintenance 0 days",
			config: &integreatlyv1alpha1.RHMIConfig{
				Spec: integreatlyv1alpha1.RHMIConfigSpec{
					Maintenance: integreatlyv1alpha1.Maintenance{
						ApplyFrom: "Sun 00:00",
					},
					Upgrade: integreatlyv1alpha1.Upgrade{
						NotBeforeDays:      "0",
						WaitForMaintenance: "true",
					},
				},
			},
			expectedSchedule: &integreatlyv1alpha1.UpgradeSchedule{
				For: time.
					Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC).
					AddDate(0, 0, 7-int(time.Now().Weekday())).
					Format(integreatlyv1alpha1.DateFormat),
			},
		}),
		makeScheduleScenario(&scheduleScenario{
			name: "wait for maintenance, notBefore: 3 days before next window",
			config: &integreatlyv1alpha1.RHMIConfig{
				Spec: integreatlyv1alpha1.RHMIConfigSpec{
					Maintenance: integreatlyv1alpha1.Maintenance{
						ApplyFrom: strings.ToLower(time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC).
							Add(6 * 24 * time.Hour).
							Format("Mon 15:04")),
					},
					Upgrade: integreatlyv1alpha1.Upgrade{
						WaitForMaintenance: "true",
						NotBeforeDays:      "3",
					},
				},
			},
			expectedSchedule: &integreatlyv1alpha1.UpgradeSchedule{
				For: time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC).Add(6 * 24 * time.Hour).
					Format(integreatlyv1alpha1.DateFormat),
			},
		}),
		makeScheduleScenario(&scheduleScenario{
			name: "wait for maintenance, notBefore: 3 days after next window",
			config: &integreatlyv1alpha1.RHMIConfig{
				Spec: integreatlyv1alpha1.RHMIConfigSpec{
					Maintenance: integreatlyv1alpha1.Maintenance{
						ApplyFrom: strings.ToLower(time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC).
							Add(3 * 24 * time.Hour).
							Format("Mon 15:04")),
					},
					Upgrade: integreatlyv1alpha1.Upgrade{
						WaitForMaintenance: "true",
						NotBeforeDays:      "6",
					},
				},
			},
			expectedSchedule: &integreatlyv1alpha1.UpgradeSchedule{
				For: time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC).Add(10 * 24 * time.Hour).
					Format(integreatlyv1alpha1.DateFormat),
			},
		}),
		makeScheduleScenario(&scheduleScenario{
			name: "do not wait for maintenance, notBefore > 0",
			config: &integreatlyv1alpha1.RHMIConfig{
				Spec: integreatlyv1alpha1.RHMIConfigSpec{
					Upgrade: integreatlyv1alpha1.Upgrade{
						NotBeforeDays:      "3",
						WaitForMaintenance: "false",
					},
				},
			},
			expectedSchedule: &integreatlyv1alpha1.UpgradeSchedule{
				For: nowOffset(-2).Add(3 * 24 * time.Hour).Format(integreatlyv1alpha1.DateFormat),
			},
		}),
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			client := fake.NewFakeClientWithScheme(buildScheme(), scenario.Config)
			err := UpdateStatus(context.TODO(), client, scenario.Config, scenario.InstallPlan)
			updatedConfig := &integreatlyv1alpha1.RHMIConfig{}
			client.Get(context.TODO(), k8sclient.ObjectKey{Name: "test-config", Namespace: "redhat-rhmi-operator"}, updatedConfig)
			scenario.Validate(t, err, updatedConfig, scenario.InstallPlan)
		})
	}
}

func TestCanUpgradeNow(t *testing.T) {
	scenarios := []struct {
		Name         string
		Config       *integreatlyv1alpha1.RHMIConfig
		Installation *integreatlyv1alpha1.RHMI
		Validate     func(*testing.T, bool, error)
	}{
		{
			Name: "always immediately returns true",
			Config: &integreatlyv1alpha1.RHMIConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rhmi-config",
					Namespace: "redhat-rhmi-operator",
				},
				Spec: integreatlyv1alpha1.RHMIConfigSpec{
					Upgrade: integreatlyv1alpha1.Upgrade{
						NotBeforeDays:      "0",
						WaitForMaintenance: "false",
					},
				},
				Status: integreatlyv1alpha1.RHMIConfigStatus{
					Upgrade: integreatlyv1alpha1.RHMIConfigStatusUpgrade{
						Scheduled: &integreatlyv1alpha1.UpgradeSchedule{
							For: nowOffset(-1).Format(integreatlyv1alpha1.DateFormat),
						},
					},
				},
			},
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rhmi",
					Namespace: "redhat-rhmi-operator",
				},
				Status: integreatlyv1alpha1.RHMIStatus{
					Stage: integreatlyv1alpha1.StageName(integreatlyv1alpha1.PhaseCompleted),
				},
			},
			Validate: func(t *testing.T, canUpgrade bool, err error) {
				if err != nil {
					t.Error("Expected no errors")
				}
				if !canUpgrade {
					t.Error("Expected canUpgrade true, got false")
				}

			},
		},
		{
			Name: "during maintenance now returns true",
			Config: &integreatlyv1alpha1.RHMIConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rhmi-config",
					Namespace: "redhat-rhmi-operator",
				},
				Spec: integreatlyv1alpha1.RHMIConfigSpec{
					Upgrade: integreatlyv1alpha1.Upgrade{
						WaitForMaintenance: "true",
						NotBeforeDays:      "0",
					},
				},
				Status: integreatlyv1alpha1.RHMIConfigStatus{
					Maintenance: integreatlyv1alpha1.RHMIConfigStatusMaintenance{
						ApplyFrom: nowOffset(-1).Format("2-1-2006 15:04"),
						Duration:  "6hrs",
					},
					Upgrade: integreatlyv1alpha1.RHMIConfigStatusUpgrade{
						Scheduled: &integreatlyv1alpha1.UpgradeSchedule{
							For: nowOffset(-1).Format(integreatlyv1alpha1.DateFormat),
						},
					},
				},
			},
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rhmi",
					Namespace: "redhat-rhmi-operator",
				},
				Status: integreatlyv1alpha1.RHMIStatus{
					Stage: integreatlyv1alpha1.StageName(integreatlyv1alpha1.PhaseCompleted),
				},
			},
			Validate: func(t *testing.T, canUpgrade bool, err error) {
				if err != nil {
					t.Error("Expected no errors, got: " + err.Error())
				}
				if !canUpgrade {
					t.Error("Expected canUpgrade true, got false")
				}

			},
		},
		{
			Name: "during maintenance expired returns false",
			Config: &integreatlyv1alpha1.RHMIConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rhmi-config",
					Namespace: "redhat-rhmi-operator",
				},
				Spec: integreatlyv1alpha1.RHMIConfigSpec{
					Upgrade: integreatlyv1alpha1.Upgrade{
						WaitForMaintenance: "true",
					},
				},
				Status: integreatlyv1alpha1.RHMIConfigStatus{
					Maintenance: integreatlyv1alpha1.RHMIConfigStatusMaintenance{
						ApplyFrom: nowOffset(-7).Format("2-1-2006 15:04"),
						Duration:  "6hrs",
					},
					Upgrade: integreatlyv1alpha1.RHMIConfigStatusUpgrade{
						Scheduled: &integreatlyv1alpha1.UpgradeSchedule{
							For: nowOffset(-7).Format(integreatlyv1alpha1.DateFormat),
						},
					},
				},
			},
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rhmi",
					Namespace: "redhat-rhmi-operator",
				},
				Status: integreatlyv1alpha1.RHMIStatus{
					Stage: integreatlyv1alpha1.StageName(integreatlyv1alpha1.PhaseCompleted),
				},
			},
			Validate: func(t *testing.T, canUpgrade bool, err error) {
				if err != nil {
					t.Error("Expected no errors, got: " + err.Error())
				}
				if canUpgrade {
					t.Error("Expected canUpgrade false, got true")
				}
			},
		},
		{
			Name: "during maintenance in future returns false",
			Config: &integreatlyv1alpha1.RHMIConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rhmi-config",
					Namespace: "redhat-rhmi-operator",
				},
				Spec: integreatlyv1alpha1.RHMIConfigSpec{
					Upgrade: integreatlyv1alpha1.Upgrade{
						WaitForMaintenance: "true",
					},
				},
				Status: integreatlyv1alpha1.RHMIConfigStatus{
					Maintenance: integreatlyv1alpha1.RHMIConfigStatusMaintenance{
						ApplyFrom: nowOffset(2).Format("2-1-2006 15:04"),
						Duration:  "6hrs",
					},
					Upgrade: integreatlyv1alpha1.RHMIConfigStatusUpgrade{
						Scheduled: &integreatlyv1alpha1.UpgradeSchedule{
							For: nowOffset(2).Format(integreatlyv1alpha1.DateFormat),
						},
					},
				},
			},
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rhmi",
					Namespace: "redhat-rhmi-operator",
				},
				Status: integreatlyv1alpha1.RHMIStatus{
					Stage: integreatlyv1alpha1.StageName(integreatlyv1alpha1.PhaseCompleted),
				},
			},
			Validate: func(t *testing.T, canUpgrade bool, err error) {
				if err != nil {
					t.Error("Expected no errors, got: " + err.Error())
				}
				if canUpgrade {
					t.Error("Expected canUpgrade false, got true")
				}
			},
		},
		{
			Name: "Do not upgrade when another upgrade is in progress",
			Config: &integreatlyv1alpha1.RHMIConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rhmi-config",
					Namespace: "redhat-rhmi-operator",
				},
				Spec: integreatlyv1alpha1.RHMIConfigSpec{
					Upgrade: integreatlyv1alpha1.Upgrade{
						NotBeforeDays:      "0",
						WaitForMaintenance: "false",
					},
				},
			},
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rhmi",
					Namespace: "redhat-rhmi-operator",
				},
				Status: integreatlyv1alpha1.RHMIStatus{
					Stage:     integreatlyv1alpha1.StageName(integreatlyv1alpha1.PhaseInProgress),
					ToVersion: "next-version",
				},
			},
			Validate: func(t *testing.T, canUpgrade bool, err error) {
				if err != nil {
					t.Error("Expected no errors")
				}
				if canUpgrade {
					t.Error("Expected canUpgrade false, got true")
				}

			},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			canUpgrade, err := CanUpgradeNow(scenario.Config, scenario.Installation)
			scenario.Validate(t, canUpgrade, err)
		})
	}
}

func TestIsUpgradeAvailable(t *testing.T) {
	scenarios := []struct {
		Name                           string
		RhmiSubscription               *olmv1alpha1.Subscription
		ExpectedUpgradeAvailableResult bool
	}{
		{
			Name:                           "Test no subscription found",
			RhmiSubscription:               nil,
			ExpectedUpgradeAvailableResult: false,
		},
		{
			Name: "Test same RHMI CSV version in subscription",
			RhmiSubscription: &olmv1alpha1.Subscription{
				Status: olmv1alpha1.SubscriptionStatus{
					CurrentCSV:   "1.0.0",
					InstalledCSV: "1.0.0",
				},
			},
			ExpectedUpgradeAvailableResult: false,
		},
		{
			Name: "Test new RHMI CSV version in subscription",
			RhmiSubscription: &olmv1alpha1.Subscription{
				Status: olmv1alpha1.SubscriptionStatus{
					CurrentCSV:   "1.0.1",
					InstalledCSV: "1.0.0",
				},
			},
			ExpectedUpgradeAvailableResult: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			upgradeAvailable := IsUpgradeAvailable(scenario.RhmiSubscription)
			if upgradeAvailable != scenario.ExpectedUpgradeAvailableResult {
				t.Fatalf("Expected upgradeAvailable to be %v but got %v", scenario.ExpectedUpgradeAvailableResult, upgradeAvailable)
			}
		})
	}
}

func TestIsUpgradeServiceAffecting(t *testing.T) {
	scenarios := []struct {
		Name                           string
		RhmiCSV                        *olmv1alpha1.ClusterServiceVersion
		ExpectedServiceAffectingResult bool
	}{
		{
			Name:                           "Test no CSV",
			RhmiCSV:                        nil,
			ExpectedServiceAffectingResult: true,
		},
		{
			Name: "Test CSV with no service_affecting annotation",
			RhmiCSV: &olmv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			ExpectedServiceAffectingResult: true,
		},
		{
			Name: "Test CSV with service_affecting annotation true",
			RhmiCSV: &olmv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"serviceAffecting": "true",
					},
				},
			},
			ExpectedServiceAffectingResult: true,
		},
		{
			Name: "Test CSV with service_affecting annotation false",
			RhmiCSV: &olmv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"serviceAffecting": "false",
					},
				},
			},
			ExpectedServiceAffectingResult: false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			isServiceAffecting := IsUpgradeServiceAffecting(scenario.RhmiCSV)
			if isServiceAffecting != scenario.ExpectedServiceAffectingResult {
				t.Fatalf("Expected isServiceAffecting to be %v but got %v", scenario.ExpectedServiceAffectingResult, isServiceAffecting)
			}
		})
	}
}

func TestApproveUpgrade(t *testing.T) {
	installPlanObjectMeta := metav1.ObjectMeta{
		Name:      "rhmi-ip",
		Namespace: defaultNamespace,
	}

	installPlanReadyForApproval := &olmv1alpha1.InstallPlan{
		ObjectMeta: installPlanObjectMeta,
		Spec: olmv1alpha1.InstallPlanSpec{
			Approved: false,
			ClusterServiceVersionNames: []string{
				"RHMI-v1.0.0",
			},
		},
		Status: olmv1alpha1.InstallPlanStatus{
			Plan: []*olmv1alpha1.Step{
				{
					Resource: olmv1alpha1.StepResource{
						Kind:     "ClusterServiceVersion",
						Manifest: fmt.Sprintf("{\"kind\":\"ClusterServiceVersion\",    \"spec\": {      \"version\": \"%s\"}}", version.Version),
					},
				},
			},
		},
	}

	rhmiMock := &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhmi",
			Namespace: "redhat-rhmi-operator",
		},
	}

	installPlanAlreadyUpgrading := &olmv1alpha1.InstallPlan{
		ObjectMeta: installPlanObjectMeta,
		Spec: olmv1alpha1.InstallPlanSpec{
			Approved: false,
		},
		Status: olmv1alpha1.InstallPlanStatus{
			Phase: olmv1alpha1.InstallPlanPhaseInstalling,
		},
	}

	scenarios := []struct {
		Name            string
		FakeClient      k8sclient.Client
		Context         context.Context
		EventRecorder   record.EventRecorder
		RhmiInstallPlan *olmv1alpha1.InstallPlan
		Config          *integreatlyv1alpha1.RHMIConfig
		RHMI            *integreatlyv1alpha1.RHMI
		Verify          func(rhmiInstallPlan *olmv1alpha1.InstallPlan, config *integreatlyv1alpha1.RHMIConfig, rhmi *integreatlyv1alpha1.RHMI, err error)
	}{
		{
			Name:            "Test install plan already upgrading",
			FakeClient:      fake.NewFakeClientWithScheme(buildScheme(), installPlanAlreadyUpgrading, rhmiMock),
			Context:         context.TODO(),
			EventRecorder:   setupRecorder(),
			RhmiInstallPlan: installPlanAlreadyUpgrading,
			Config: &integreatlyv1alpha1.RHMIConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "redhat-rhmi-operator",
				},
			},
			RHMI: rhmiMock,
			Verify: func(updatedRhmiInstallPlan *olmv1alpha1.InstallPlan, config *integreatlyv1alpha1.RHMIConfig, rhmi *integreatlyv1alpha1.RHMI, err error) {
				// Should not return an error
				if err != nil {
					t.Fatalf("Unexpected error %v", err)
				}

				if updatedRhmiInstallPlan.Spec.Approved != false {
					t.Fatalf("Expected installPlan to not be upgraded")
				}
			},
		},
		{
			Name:            "Test install plan ready to upgrade",
			FakeClient:      fake.NewFakeClientWithScheme(buildScheme(), installPlanReadyForApproval, rhmiMock),
			Context:         context.TODO(),
			EventRecorder:   setupRecorder(),
			RhmiInstallPlan: installPlanReadyForApproval,
			Config: &integreatlyv1alpha1.RHMIConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "redhat-rhmi-operator",
				},
				Status: integreatlyv1alpha1.RHMIConfigStatus{
					Upgrade: integreatlyv1alpha1.RHMIConfigStatusUpgrade{
						Scheduled: &integreatlyv1alpha1.UpgradeSchedule{
							For: "13 Jul 2020 00:00",
						},
					},
				},
			},
			RHMI: rhmiMock,
			Verify: func(updatedRhmiInstallPlan *olmv1alpha1.InstallPlan, config *integreatlyv1alpha1.RHMIConfig, rhmi *integreatlyv1alpha1.RHMI, err error) {
				// Should not return an error
				if err != nil {
					t.Fatalf("Unexpected error %v", err)
				}

				if updatedRhmiInstallPlan.Spec.Approved != true {
					t.Fatalf("Expected installplan.Spec.Approved to be true")
				}

				if config.Status.Upgrade.Scheduled != nil {
					t.Fatalf("Expected scheduled field to be empty")
				}
				if rhmi.Status.ToVersion != version.Version {
					t.Fatalf("Expected ToVersion to be version.version")
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			ApproveUpgrade(context.TODO(), scenario.FakeClient, scenario.RHMI, scenario.RhmiInstallPlan, scenario.EventRecorder)
			retrievedInstallPlan := &olmv1alpha1.InstallPlan{}
			err := scenario.FakeClient.Get(scenario.Context, k8sclient.ObjectKey{Name: scenario.RhmiInstallPlan.Name, Namespace: scenario.RhmiInstallPlan.Namespace}, retrievedInstallPlan)
			rhmi := &integreatlyv1alpha1.RHMI{}
			err = scenario.FakeClient.Get(scenario.Context, k8sclient.ObjectKey{Name: scenario.RHMI.Name, Namespace: scenario.RHMI.Namespace}, rhmi)
			updatedConfig := &integreatlyv1alpha1.RHMIConfig{}
			scenario.FakeClient.Get(context.TODO(), k8sclient.ObjectKey{Name: "test-config", Namespace: "redhat-rhmi-operator"}, updatedConfig)
			scenario.Verify(retrievedInstallPlan, updatedConfig, rhmi, err)
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
		t.Errorf("Expected result to be same day, got %s", r.Format(integreatlyv1alpha1.DateFormat))
	}

	// Test next day
	r, _, err = getWeeklyWindow(from, "Tue 00:00", time.Hour)
	if err != nil {
		t.Errorf("Error calculating weekly window for same day: %v", err)
	} else if r.Day() != from.Day()+1 || r.Month() != from.Month() || r.Year() != from.Year() {
		t.Errorf("Expected result to be next day, got %s", r.Format(integreatlyv1alpha1.DateFormat))
	}

	// Test day before
	r, _, err = getWeeklyWindow(from, "SuN 00:00", time.Hour)
	if err != nil {
		t.Errorf("Error calculating weekly window for same day: %v", err)
	} else if r.Day() != from.Day()+6 || r.Month() != from.Month() || r.Year() != from.Year() {
		t.Errorf("Expected result to be next Sunday, got %s", r.Format(integreatlyv1alpha1.DateFormat))
	}

	// Test 3 days after
	r, _, err = getWeeklyWindow(from, "Thu 02:00", time.Hour)
	if err != nil {
		t.Errorf("Error calculating weekly window for same day: %v", err)
	} else if r.Day() != from.Day()+3 || r.Month() != from.Month() || r.Year() != from.Year() {
		t.Errorf("Expected result to be Thursday, got %s", r.Format(integreatlyv1alpha1.DateFormat))
	}
}
