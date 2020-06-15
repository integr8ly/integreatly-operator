package rhmiConfigs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/integr8ly/integreatly-operator/pkg/metrics"
	"github.com/sirupsen/logrus"

	"k8s.io/client-go/tools/record"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	WINDOW        = 6
	WINDOW_MARGIN = 1
)

func IsUpgradeAvailable(subscription *olmv1alpha1.Subscription) bool {
	if subscription == nil {
		return false
	}
	// How to tell an upgrade is available - https://operator-framework.github.io/olm-book/docs/subscriptions.html#how-do-i-know-when-an-update-is-available-for-an-operator
	return subscription.Status.CurrentCSV != subscription.Status.InstalledCSV
}

func GetLatestInstallPlan(ctx context.Context, subscription *olmv1alpha1.Subscription, client k8sclient.Client) (*olmv1alpha1.InstallPlan, error) {
	latestInstallPlan := &olmv1alpha1.InstallPlan{}
	// Get the latest installPlan associated with the currentCSV (newest known to OLM)
	installPlanName := subscription.Status.InstallPlanRef.Name
	installPlanNamespace := subscription.Status.InstallPlanRef.Namespace
	err := client.Get(ctx, k8sclient.ObjectKey{Name: installPlanName, Namespace: installPlanNamespace}, latestInstallPlan)
	if err != nil {
		return nil, err
	}

	return latestInstallPlan, nil
}

func GetCSV(installPlan *olmv1alpha1.InstallPlan) (*olmv1alpha1.ClusterServiceVersion, error) {
	csv := &olmv1alpha1.ClusterServiceVersion{}

	// The latest CSV is only represented in the new install plan while the upgrade is pending approval
	for _, installPlanResources := range installPlan.Status.Plan {
		if installPlanResources.Resource.Kind == olmv1alpha1.ClusterServiceVersionKind {
			err := json.Unmarshal([]byte(installPlanResources.Resource.Manifest), &csv)
			if err != nil {
				return csv, fmt.Errorf("failed to unmarshal json: %w", err)
			}
		}
	}

	return csv, nil
}

func UpdateStatus(ctx context.Context, client k8sclient.Client, config *integreatlyv1alpha1.RHMIConfig, installplan *olmv1alpha1.InstallPlan) error {
	// Calculate the next maintenance window based on the maintenance schedule
	if config.Spec.Maintenance.ApplyFrom != "" {
		mtStart, _, err := getWeeklyWindowFromNow(config.Spec.Maintenance.ApplyFrom, time.Hour*WINDOW)
		if err != nil {
			return err
		}

		config.Status.Maintenance.ApplyFrom = mtStart.Format("2-1-2006 15:04")
		config.Status.Maintenance.Duration = strconv.Itoa(WINDOW) + "hrs"
	}

	// If the install plan was approved, simply clear the status
	if installplan.Spec.Approved {
		config.Status.Upgrade.Scheduled.For = ""

		return client.Status().Update(ctx, config)
	}

	// Calculate the upgrade schedule based on the spec:
	// We can assume there's no error parsing the value as it was validated
	notBeforeDays := *config.Spec.Upgrade.NotBeforeDays
	waitForMaintenance := *config.Spec.Upgrade.WaitForMaintenance

	upgradeSchedule := installplan.ObjectMeta.CreationTimestamp.Time.
		Add(daysDuration(notBeforeDays))

	if waitForMaintenance {
		var err error
		upgradeSchedule, _, err = getWeeklyWindow(upgradeSchedule, config.Spec.Maintenance.ApplyFrom, time.Hour*WINDOW)
		if err != nil {
			return err
		}
	}

	// Update the upgrade status
	config.Status.Upgrade = integreatlyv1alpha1.RHMIConfigStatusUpgrade{
		Scheduled: &integreatlyv1alpha1.UpgradeSchedule{
			For: upgradeSchedule.Format(integreatlyv1alpha1.DateFormat),
		},
	}

	return client.Status().Update(ctx, config)
}

func CanUpgradeNow(config *integreatlyv1alpha1.RHMIConfig, installation *integreatlyv1alpha1.RHMI) (bool, error) {
	//Another upgrade in progress - don't proceed with upgrade
	if (string(installation.Status.Stage) != string(integreatlyv1alpha1.PhaseCompleted)) && installation.Status.ToVersion != "" {
		return false, nil
	}

	var duration int
	// Upgrade window taken either from the maintenance window or, by default
	// from the WINDOW constant
	waitForMaintenance := *config.Spec.Upgrade.WaitForMaintenance
	if waitForMaintenance {
		var err error
		duration, err = strconv.Atoi(strings.Replace(config.Status.Maintenance.Duration, "hrs", "", -1))
		if err != nil {
			return false, err
		}
	} else {
		duration = WINDOW
	}

	//don't approve upgrades in the last hour of the window
	window := time.Hour * time.Duration(duration-WINDOW_MARGIN)
	upgradeTime, err := time.Parse(integreatlyv1alpha1.DateFormat, config.Status.Upgrade.Scheduled.For)
	if err != nil {
		return false, err
	}

	return inWindow(upgradeTime, upgradeTime.Add(window)), nil
}

func inWindow(windowStart time.Time, windowEnd time.Time) bool {
	now := time.Now().UTC()
	return windowStart.Before(now) && windowEnd.After(now)
}

func getWeeklyWindow(from time.Time, windowStartStr string, duration time.Duration) (time.Time, time.Time, error) {
	var shortDays = map[string]int{
		"sun": 0,
		"mon": 1,
		"tue": 2,
		"wed": 3,
		"thu": 4,
		"fri": 5,
		"sat": 6,
	}

	windowSegments := strings.Split(windowStartStr, " ")
	windowDay := windowSegments[0]

	windowTimeSegments := strings.Split(windowSegments[1], ":")
	windowHour, err := strconv.Atoi(windowTimeSegments[0])
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	windowMin, err := strconv.Atoi(windowTimeSegments[1])
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	//calculate how far away from maintenance day today is, within the current week
	dayDiff := shortDays[strings.ToLower(windowDay)] - int(from.Weekday())
	if dayDiff < 0 {
		dayDiff = 7 + dayDiff
	}

	//negative days roll back the month and year, tested here: https://play.golang.org/p/gBBHw49nH1b
	windowStart := time.Date(from.Year(), from.Month(), from.Day(), windowHour, windowMin, 0, 0, time.UTC)
	windowStart = windowStart.Add(daysDuration(dayDiff))
	return windowStart, windowStart.Add(duration), nil
}

//windowStartStr must be in format: sun 23:00
func getWeeklyWindowFromNow(windowStartStr string, duration time.Duration) (time.Time, time.Time, error) {
	return getWeeklyWindow(time.Now().UTC(), windowStartStr, duration)
}

func IsUpgradeServiceAffecting(csv *olmv1alpha1.ClusterServiceVersion) bool {
	// Always default to the release being service affecting and requiring manual upgrade approval
	serviceAffectingUpgrade := true
	if csv == nil {
		return serviceAffectingUpgrade
	}

	if val, ok := csv.ObjectMeta.Annotations["serviceAffecting"]; ok && val == "false" {
		serviceAffectingUpgrade = false
	}
	return serviceAffectingUpgrade
}

func ApproveUpgrade(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI, installPlan *olmv1alpha1.InstallPlan, eventRecorder record.EventRecorder) error {

	if installPlan.Status.Phase == olmv1alpha1.InstallPlanPhaseInstalling {
		return nil
	}

	eventRecorder.Eventf(installPlan, "Normal", integreatlyv1alpha1.EventUpgradeApproved,
		"Approving %s install plan: %s", installPlan.Name, installPlan.Spec.ClusterServiceVersionNames[0])

	installPlan.Spec.Approved = true
	err := client.Update(ctx, installPlan)
	if err != nil {
		return err
	}

	csv, err := GetCSV(installPlan)
	if err != nil {
		return err
	}

	version := csv.Spec.Version.String()
	logrus.Infof("Update approved, setting rhmi version to install %s", version)
	installation.Status.ToVersion = version
	err = client.Status().Update(ctx, installation)
	if err != nil {
		return err
	}

	metrics.SetRhmiVersions(string(installation.Status.Stage), installation.Status.Version, installation.Status.ToVersion, installation.CreationTimestamp.Unix())

	return nil
}

func daysDuration(numberOfDays int) time.Duration {
	return time.Duration(numberOfDays) * 24 * time.Hour
}
