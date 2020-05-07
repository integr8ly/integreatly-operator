package rhmiConfigs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	if config.Spec.Maintenance.ApplyFrom != "" {
		mtStart, _, err := getWeeklyWindow(config.Spec.Maintenance.ApplyFrom, time.Hour*WINDOW)
		if err != nil {
			return err
		}

		config.Status.Maintenance.ApplyFrom = mtStart.Format("2-1-2006 15:04")
		config.Status.Maintenance.Duration = strconv.Itoa(WINDOW) + "hrs"
	}

	if installplan.Spec.Approved {
		config.Status.Upgrade.Window = ""
	} else {
		upStart := installplan.ObjectMeta.CreationTimestamp.Time.Format("2 Jan 2006")
		upEnd := installplan.ObjectMeta.CreationTimestamp.Time.Add((time.Hour * 24) * 14).Format("2 Jan 2006")
		config.Status.Upgrade.Window = upStart + " - " + upEnd
	}
	return client.Status().Update(ctx, config)
}

func CanUpgradeNow(config *integreatlyv1alpha1.RHMIConfig) (bool, error) {
	if config.Spec.Upgrade.AlwaysImmediately {
		return true, nil
	}

	//Check if we are in the maintenance window
	if config.Spec.Upgrade.DuringNextMaintenance {
		duration, err := strconv.Atoi(strings.Replace(config.Status.Maintenance.Duration, "hrs", "", -1))
		if err != nil {
			return false, err
		}

		//don't approve upgrades in the last hour of maintenance
		window := time.Hour * time.Duration(duration-WINDOW_MARGIN)

		mtStart, err := time.Parse("2-1-2006 15:04", config.Status.Maintenance.ApplyFrom)
		if err != nil {
			return false, err
		}

		return inWindow(mtStart, mtStart.Add(window)), nil
	}

	if config.Spec.Upgrade.ApplyOn != "" {
		upTime, err := time.Parse("2 Jan 2006 15:04", config.Spec.Upgrade.ApplyOn)
		if err != nil {
			return false, err
		}
		return inWindow(upTime, upTime.Add(time.Hour*(WINDOW-WINDOW_MARGIN))), nil
	}
	return false, nil
}

func inWindow(windowStart time.Time, windowEnd time.Time) bool {
	now := time.Now().UTC()
	return windowStart.Before(now) && windowEnd.After(now)
}

//windowStartStr must be in format: sun 23:00
func getWeeklyWindow(windowStartStr string, duration time.Duration) (time.Time, time.Time, error) {
	var shortDays = map[string]int{
		"sun": 0,
		"mon": 1,
		"tue": 2,
		"wed": 3,
		"thu": 4,
		"fri": 5,
		"sat": 6,
	}
	now := time.Now().UTC()
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
	dayDiff := shortDays[windowDay] - int(now.Weekday())

	//negative days roll back the month and year, tested here: https://play.golang.org/p/gBBHw49nH1b
	windowStart := time.Date(now.Year(), now.Month(), now.Day()+dayDiff, windowHour, windowMin, 0, 0, time.UTC)
	return windowStart, windowStart.Add(duration), nil
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

func ApproveUpgrade(ctx context.Context, client k8sclient.Client, installPlan *olmv1alpha1.InstallPlan, eventRecorder record.EventRecorder) error {
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

	return nil
}
