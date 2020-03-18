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
		sub := &coreosv1alpha1.Subscription{}
		err := ctx.Client.Get(context.TODO(), k8sclient.ObjectKey{Name: subscription.Name, Namespace: subscription.Namespace}, sub)
		if err != nil {
			t.Errorf("Error getting subscription %s in ns %s: %s", subscription.Name, subscription.Namespace, err)
		}

		if err == nil && sub.Spec.InstallPlanApproval != expectedApprovalStrategy {
			t.Errorf("Expected %s approval but got %s", sub.Spec.InstallPlanApproval, expectedApprovalStrategy)
		}
	}
}
