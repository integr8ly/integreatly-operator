package common

import (
	"context"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

const (
	expectedApprovalStrategy = coreosv1alpha1.ApprovalManual
)

var (
	subscriptionsToCheck = []SubscriptionCheck{
		{
			Name:      constants.AMQOnlineSubscriptionName,
			Namespace: AMQOnlineOperatorNamespace,
		},
		{
			Name:      constants.ApicuritoSubscriptionName,
			Namespace: ApicuritoOperatorNamespace,
		},
		{
			Name:      constants.CloudResourceSubscriptionName,
			Namespace: CloudResourceOperatorNamespace,
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
			Name:      constants.MonitoringSubscriptionName,
			Namespace: MonitoringOperatorNamespace,
		},
		{
			Name:      constants.RHSSOUserSubscriptionName,
			Namespace: RHSSOUserOperatorNamespace,
		},
		{
			Name:      constants.RHSSOSubscriptionName,
			Namespace: RHSSOOperatorNamespace,
		},
		{
			Name:      constants.SolutionExplorerSubscriptionName,
			Namespace: SolutionExplorerOperatorNamespace,
		},
		{
			Name:      constants.ThreeScaleSubscriptionName,
			Namespace: ThreeScaleOperatorNamespace,
		},
		{
			Name:      constants.UPSSubscriptionName,
			Namespace: UPSOperatorNamespace,
		},
	}
)

func TestSubscriptionInstallPlanType(t *testing.T, ctx *TestingContext) {
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
