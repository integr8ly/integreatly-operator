package common

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
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
			Name:      constants.AMQOnlineSubscriptionName,
			Namespace: fmt.Sprintf("%samq-online", namespacePrefix),
		},
		{
			Name:      constants.ApicuritoSubscriptionName,
			Namespace: fmt.Sprintf("%sapicurito-operator", namespacePrefix),
		},
		{
			Name:      constants.CloudResourceSubscriptionName,
			Namespace: fmt.Sprintf("%scloud-resources-operator", namespacePrefix),
		},
		{
			Name:      constants.CodeReadySubscriptionName,
			Namespace: fmt.Sprintf("%scodeready-workspaces-operator", namespacePrefix),
		},
		{
			Name:      constants.FuseSubscriptionName,
			Namespace: fmt.Sprintf("%sfuse-operator", namespacePrefix),
		},
		{
			Name:      constants.MonitoringSubscriptionName,
			Namespace: fmt.Sprintf("%smiddleware-monitoring-operator", namespacePrefix),
		},
		{
			Name:      constants.RHSSOUserSubscriptionName,
			Namespace: fmt.Sprintf("%suser-sso-operator", namespacePrefix),
		},
		{
			Name:      constants.RHSSOSubscriptionName,
			Namespace: fmt.Sprintf("%srhsso-operator", namespacePrefix),
		},
		{
			Name:      constants.SolutionExplorerSubscriptionName,
			Namespace: fmt.Sprintf("%ssolution-explorer-operator", namespacePrefix),
		},
		{
			Name:      constants.ThreeScaleSubscriptionName,
			Namespace: fmt.Sprintf("%s3scale-operator", namespacePrefix),
		},
		{
			Name:      constants.UPSSubscriptionName,
			Namespace: fmt.Sprintf("%sups-operator", namespacePrefix),
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
