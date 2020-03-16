package common

import (
	"context"
	"fmt"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

const (
	expectedApprovalStrategy = coreosv1alpha1.ApprovalManual
)

type SubscriptionCheck struct {
	Name      string
	Namespace string
}

var (
	subscriptionsToCheck = []SubscriptionCheck{
		{
			Name:      "rhmi-3scale",
			Namespace: "redhat-rhmi-3scale-operator",
		},
		{
			Name:      "rhmi-amq-online",
			Namespace: "redhat-rhmi-amq-online",
		},
		{
			Name:      "rhmi-apicurito",
			Namespace: "redhat-rhmi-apicurito-operator",
		},
		{
			Name:      "rhmi-cloud-resources",
			Namespace: "redhat-rhmi-cloud-resources-operator",
		},
		{
			Name:      "rhmi-codeready-workspaces",
			Namespace: "redhat-rhmi-codeready-workspaces-operator",
		},
		{
			Name:      "rhmi-monitoring",
			Namespace: "redhat-rhmi-middleware-monitoring-operator",
		},
		{
			Name:      "rhmi-rhsso",
			Namespace: "redhat-rhmi-user-sso-operator",
		},
		{
			Name:      "rhmi-rhsso",
			Namespace: "redhat-rhmi-rhsso-operator",
		},
		{
			Name:      "rhmi-solution-explorer",
			Namespace: "redhat-rhmi-solution-explorer-operator",
		},
		{
			Name:      "rhmi-syndesis",
			Namespace: "redhat-rhmi-fuse-operator",
		},
		{
			Name:      "rhmi-unifiedpush",
			Namespace: "redhat-rhmi-ups-operator",
		},
	}
)

func TestSubscriptionInstallPlanType(t *testing.T, ctx *TestingContext) {
	var testErrors []string

	for _, subscription := range subscriptionsToCheck {
		sub := &coreosv1alpha1.Subscription{}
		err := ctx.Client.Get(context.TODO(), k8sclient.ObjectKey{Name: subscription.Name, Namespace: subscription.Namespace}, sub)
		if err != nil {
			testErrors = append(testErrors, fmt.Sprintf("\nError getting subscription %s in ns: %s", subscription.Name, subscription.Namespace))
		}

		if err == nil && sub.Spec.InstallPlanApproval != expectedApprovalStrategy {
			testErrors = append(testErrors, fmt.Sprintf("\nExpected %s approval but got %s", sub.Spec.InstallPlanApproval, expectedApprovalStrategy))
		}
	}

	if len(testErrors) != 0 {
		t.Fatalf("Test subscription install plan type failed with the following errors: %s", testErrors)
	}
}
