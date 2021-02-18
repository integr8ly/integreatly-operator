package common

import (
	"context"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	expectedApprovalStrategy = coreosv1alpha1.ApprovalManual
)

func commonSubscriptionsToCheck() []SubscriptionCheck {
	return []SubscriptionCheck{
		{
			Name:      constants.MonitoringSubscriptionName,
			Namespace: MonitoringOperatorNamespace,
		},
		{
			Name:      constants.RHSSOSubscriptionName,
			Namespace: RHSSOUserOperatorNamespace,
		},
		{
			Name:      constants.RHSSOSubscriptionName,
			Namespace: RHSSOOperatorNamespace,
		},
		{
			Name:      constants.ThreeScaleSubscriptionName,
			Namespace: ThreeScaleOperatorNamespace,
		},
		{
			Name:      constants.CloudResourceSubscriptionName,
			Namespace: CloudResourceOperatorNamespace,
		},
	}
}

// Applicable to rhmi 2 install types
func rhmi2SubscriptionsToCheck() []SubscriptionCheck {
	return []SubscriptionCheck{
		{
			Name:      constants.AMQOnlineSubscriptionName,
			Namespace: AMQOnlineOperatorNamespace,
		},
		{
			Name:      constants.ApicuritoSubscriptionName,
			Namespace: ApicuritoOperatorNamespace,
		},
		{
			Name:      constants.CodeReadySubscriptionName,
			Namespace: CodeReadyOperatorNamespace,
		},
		{
			Name:      constants.FuseSubscriptionName,
			Namespace: FuseOperatorNamespace,
		},
		{
			Name:      constants.UPSSubscriptionName,
			Namespace: UPSOperatorNamespace,
		},
		{
			Name:      constants.SolutionExplorerSubscriptionName,
			Namespace: SolutionExplorerOperatorNamespace,
		},
	}
}
func managedApiSubscriptionsToCheck() []SubscriptionCheck {
	return []SubscriptionCheck{
		{
			Name:      constants.Marin3rSubscriptionName,
			Namespace: Marin3rOperatorNamespace,
		},
		{
			Name:      constants.GrafanaSubscriptionName,
			Namespace: CustomerGrafanaNamespace,
		},
	}
}

func TestSubscriptionInstallPlanType(t TestingTB, ctx *TestingContext) {

	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("failed to get the RHMI: %s", err)
	}
	subscriptionsToCheck := getSubscriptionsToCheck(rhmi.Spec.Type)

	for _, subscription := range subscriptionsToCheck {
		// Check subscription install plan approval strategy
		sub := &coreosv1alpha1.Subscription{}
		err := ctx.Client.Get(context.TODO(), k8sclient.ObjectKey{Name: subscription.Name, Namespace: subscription.Namespace}, sub)
		if err != nil {
			t.Errorf("Error getting subscription %s in ns %s: %s", subscription.Name, subscription.Namespace, err)
			continue
		}

		if sub.Spec.InstallPlanApproval != expectedApprovalStrategy {
			t.Errorf("Expected %s approval for %s subscription but got %s", expectedApprovalStrategy, subscription, sub.Spec.InstallPlanApproval)
			continue
		}

		// Check all install plan approvals in namespace
		installPlans := &coreosv1alpha1.InstallPlanList{}
		err = ctx.Client.List(context.TODO(), installPlans, &k8sclient.ListOptions{
			Namespace: subscription.Namespace,
		})

		if err != nil {
			t.Errorf("Error getting install plans for %s namespace: %s", subscription.Namespace, err)
			continue
		}

		if len(installPlans.Items) == 0 {
			t.Errorf("Expected at least 1 install plan in %s namespace but got 0", subscription.Namespace)
			continue
		}

		for _, installPlan := range installPlans.Items {
			if installPlan.Spec.Approval != expectedApprovalStrategy {
				t.Errorf("Expected %s approval for install plan in %s namespace but got %s", expectedApprovalStrategy, subscription.Namespace, installPlan.Spec.Approval)
			}
		}
	}
}

func getSubscriptionsToCheck(installType string) []SubscriptionCheck {
	if installType == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		return append(commonSubscriptionsToCheck(), managedApiSubscriptionsToCheck()...)
	} else {
		return append(commonSubscriptionsToCheck(), rhmi2SubscriptionsToCheck()...)
	}
}
