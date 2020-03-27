package common

import (
	"context"
	"github.com/integr8ly/integreatly-operator/pkg/products/amqonline"
	"github.com/integr8ly/integreatly-operator/pkg/products/apicurito"
	"github.com/integr8ly/integreatly-operator/pkg/products/cloudresources"
	"github.com/integr8ly/integreatly-operator/pkg/products/codeready"
	"github.com/integr8ly/integreatly-operator/pkg/products/fuse"
	"github.com/integr8ly/integreatly-operator/pkg/products/monitoring"
	"github.com/integr8ly/integreatly-operator/pkg/products/rhsso"
	"github.com/integr8ly/integreatly-operator/pkg/products/rhssouser"
	"github.com/integr8ly/integreatly-operator/pkg/products/solutionexplorer"
	"github.com/integr8ly/integreatly-operator/pkg/products/threescale"
	"github.com/integr8ly/integreatly-operator/pkg/products/ups"
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
			Name:      amqonline.DefaultSubscriptionName,
			Namespace: AMQOnlineOperatorNamespace,
		},
		{
			Name:      apicurito.DefaultSubscriptionName,
			Namespace: ApicuritoOperatorNamespace,
		},
		{
			Name:      cloudresources.DefaultSubscriptionName,
			Namespace: CloudResourceOperatorNamespace,
		},
		{
			Name:      codeready.DefaultSubscriptionName,
			Namespace: CodeReadyOperatorNamespace,
		},
		{
			Name:      fuse.DefaultSubscriptionName,
			Namespace: FuseOperatorNamespace,
		},
		{
			Name:      monitoring.DefaultSubscriptionName,
			Namespace: MonitoringOperatorNamespace,
		},
		{
			Name:      rhssouser.DefaultSubscriptionName,
			Namespace: RHSSOUserOperatorNamespace,
		},
		{
			Name:      rhsso.DefaultSubscriptionName,
			Namespace: RHSSOOperatorNamespace,
		},
		{
			Name:      solutionexplorer.DefaultSubscriptionName,
			Namespace: SolutionExplorerOperatorNamespace,
		},
		{
			Name:      threescale.DefaultSubscriptionName,
			Namespace: ThreeScaleOperatorNamespace,
		},
		{
			Name:      ups.DefaultSubscriptionName,
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
