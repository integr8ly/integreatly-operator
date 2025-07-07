package rhmiConfigs

import (
	"context"
	"fmt"

	"k8s.io/client-go/tools/record"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func IsUpgradeAvailable(subscription *operatorsv1alpha1.Subscription) bool {
	if subscription == nil {
		return false
	}
	// How to tell an upgrade is available - https://operator-framework.github.io/olm-book/docs/subscriptions.html#how-do-i-know-when-an-update-is-available-for-an-operator
	return subscription.Status.CurrentCSV != subscription.Status.InstalledCSV
}

func GetLatestInstallPlan(ctx context.Context, subscription *operatorsv1alpha1.Subscription, client k8sclient.Client) (*operatorsv1alpha1.InstallPlan, error) {
	latestInstallPlan := &operatorsv1alpha1.InstallPlan{}
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

func IsUpgradeServiceAffecting(csv *operatorsv1alpha1.ClusterServiceVersion) bool {
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

func ApproveUpgrade(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI, installPlan *operatorsv1alpha1.InstallPlan, eventRecorder record.EventRecorder) error {

	if installPlan.Status.Phase == operatorsv1alpha1.InstallPlanPhaseInstalling {
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
