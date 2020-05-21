package common

import (
	goctx "context"
	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

var (
	namespacesCreated = []string{
		MonitoringOperatorNamespace,
		MonitoringFederateNamespace,
		AMQOnlineOperatorNamespace,
		ApicuritoProductNamespace,
		ApicuritoOperatorNamespace,
		CloudResourceOperatorNamespace,
		CodeReadyProductNamespace,
		CodeReadyOperatorNamespace,
		FuseProductNamespace,
		FuseOperatorNamespace,
		RHSSOUserProductOperatorNamespace,
		RHSSOUserOperatorNamespace,
		RHSSOProductNamespace,
		RHSSOOperatorNamespace,
		SolutionExplorerProductNamespace,
		SolutionExplorerOperatorNamespace,
		ThreeScaleProductNamespace,
		ThreeScaleOperatorNamespace,
		UPSProductNamespace,
		UPSOperatorNamespace,
	}
)

func TestNamespaceCreated(t *testing.T, ctx *TestingContext) {
	for _, namespace := range namespacesCreated {
		ns := &corev1.Namespace{}
		err := ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: namespace}, ns)

		if err != nil {
			t.Errorf("Expected %s namespace to be created but wasn't: %s", namespace, err)
			continue
		}
	}
}
