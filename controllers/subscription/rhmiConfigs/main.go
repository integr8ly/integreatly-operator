package rhmiConfigs

import (
	"context"
	"fmt"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"k8s.io/client-go/tools/record"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"

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
	if subscription.Status.InstallPlanRef == nil {
		return nil, fmt.Errorf("installplan not found in the subscription status reference")
	}
	installPlanName := subscription.Status.InstallPlanRef.Name
	installPlanNamespace := subscription.Status.InstallPlanRef.Namespace
	err := client.Get(ctx, k8sclient.ObjectKey{Name: installPlanName, Namespace: installPlanNamespace}, latestInstallPlan)
	if err != nil {
		return nil, err
	}

	return latestInstallPlan, nil
}

func DeleteInstallPlan(ctx context.Context, installPlan *olmv1alpha1.InstallPlan, client k8sclient.Client) error {
	// remove cloud resource config map
	err := client.Delete(ctx, installPlan)
	if err != nil {
		return fmt.Errorf("error occurred trying to delete installplan, %w", err)
	}
	return nil
}

func CreateInstallPlan(ctx context.Context, rhmiSubscription *olmv1alpha1.Subscription, client k8sclient.Client) error {
	// workaround to trigger the creation of another installplan by OLM
	rhmiSubscription.Status.State = operatorsv1alpha1.SubscriptionStateAtLatest
	rhmiSubscription.Status.InstallPlanRef = nil
	rhmiSubscription.Status.Install = nil
	rhmiSubscription.Status.CurrentCSV = rhmiSubscription.Status.InstalledCSV

	err := client.Status().Update(ctx, rhmiSubscription)
	if err != nil {
		return fmt.Errorf("error updating the subscripion status block %w", err)
	}
	return nil
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

	return nil
}
