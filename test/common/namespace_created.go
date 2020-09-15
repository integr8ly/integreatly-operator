package common

import (
	goctx "context"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

var (
	rhmi2Namespaces = []string{
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
	managedApiNamespaces = []string{
		MonitoringOperatorNamespace,
		MonitoringFederateNamespace,
		CloudResourceOperatorNamespace,
		RHSSOUserProductOperatorNamespace,
		RHSSOUserOperatorNamespace,
		RHSSOProductNamespace,
		RHSSOOperatorNamespace,
		ThreeScaleProductNamespace,
		ThreeScaleOperatorNamespace,
	}
)

func TestNamespaceCreated(t *testing.T, ctx *TestingContext) {

	namespacesCreated := getNamespaces(t, ctx)

	for _, namespace := range namespacesCreated {
		ns := &corev1.Namespace{}
		err := ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: namespace}, ns)

		if err != nil {
			t.Errorf("Expected %s namespace to be created but wasn't: %s", namespace, err)
			continue
		}
	}
}

func getNamespaces(t *testing.T, ctx *TestingContext) []string {

	//get RHMI
	rhmi, err := getRHMI(ctx.Client)
	if err != nil {
		t.Errorf("error getting RHMI CR: %v", err)
	}

	if rhmi.Spec.Type == string(integreatlyv1alpha1.InstallationTypeManaged3scale) {
		return managedApiNamespaces
	} else {
		return rhmi2Namespaces
	}
}
