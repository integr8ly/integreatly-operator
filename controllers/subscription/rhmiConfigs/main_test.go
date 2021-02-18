package rhmiConfigs

import (
	"context"
	"fmt"
	"testing"
	"time"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/version"
	"k8s.io/apimachinery/pkg/runtime"

	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	defaultNamespace = "testing-namespaces-operator"
)

func setupRecorder() record.EventRecorder {
	return record.NewFakeRecorder(50)
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
					Namespace: defaultNamespace,
				},
				Spec: integreatlyv1alpha1.RHMIConfigSpec{
					Upgrade: integreatlyv1alpha1.Upgrade{
						NotBeforeDays:      intPtr(0),
						WaitForMaintenance: boolPtr(false),
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
					Namespace: defaultNamespace,
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
					Namespace: defaultNamespace,
				},
				Spec: integreatlyv1alpha1.RHMIConfigSpec{
					Upgrade: integreatlyv1alpha1.Upgrade{
						WaitForMaintenance: boolPtr(true),
						NotBeforeDays:      intPtr(0),
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
					Namespace: defaultNamespace,
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
					Namespace: defaultNamespace,
				},
				Spec: integreatlyv1alpha1.RHMIConfigSpec{
					Upgrade: integreatlyv1alpha1.Upgrade{
						WaitForMaintenance: boolPtr(true),
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
					Namespace: defaultNamespace,
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
					Namespace: defaultNamespace,
				},
				Spec: integreatlyv1alpha1.RHMIConfigSpec{
					Upgrade: integreatlyv1alpha1.Upgrade{
						WaitForMaintenance: boolPtr(true),
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
					Namespace: defaultNamespace,
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
					Namespace: defaultNamespace,
				},
				Spec: integreatlyv1alpha1.RHMIConfigSpec{
					Upgrade: integreatlyv1alpha1.Upgrade{
						NotBeforeDays:      intPtr(0),
						WaitForMaintenance: boolPtr(false),
					},
				},
			},
			Installation: &integreatlyv1alpha1.RHMI{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rhmi",
					Namespace: defaultNamespace,
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
						Manifest: fmt.Sprintf("{\"kind\":\"ClusterServiceVersion\",    \"spec\": {      \"version\": \"%s\"}}", version.GetVersion()),
					},
				},
			},
		},
	}

	rhmiMock := &integreatlyv1alpha1.RHMI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhmi",
			Namespace: defaultNamespace,
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
					Namespace: defaultNamespace,
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
					Namespace: defaultNamespace,
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
			scenario.FakeClient.Get(context.TODO(), k8sclient.ObjectKey{Name: "test-config", Namespace: defaultNamespace}, updatedConfig)
			scenario.Verify(retrievedInstallPlan, updatedConfig, rhmi, err)
		})
	}
}

func buildScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	integreatlyv1alpha1.SchemeBuilder.AddToScheme(scheme)
	olmv1alpha1.SchemeBuilder.AddToScheme(scheme)

	return scheme
}

func nowOffset(hours int) time.Time {
	now := now()
	return time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+hours, now.Minute(), now.Second(), 0, time.UTC)
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
