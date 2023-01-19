package common

import (
	goctx "context"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func managedApiNamespaces() []string {
	return []string{
		ObservabilityOperatorNamespace,
		ObservabilityProductNamespace,
		CloudResourceOperatorNamespace,
		RHSSOUserProductNamespace,
		RHSSOUserOperatorNamespace,
		RHSSOProductNamespace,
		RHSSOOperatorNamespace,
		ThreeScaleProductNamespace,
		ThreeScaleOperatorNamespace,
		Marin3rOperatorNamespace,
		Marin3rProductNamespace,
		CustomerGrafanaNamespace,
	}
}

func mtManagedApiNamespaces() []string {
	return []string{
		ObservabilityOperatorNamespace,
		ObservabilityProductNamespace,
		CloudResourceOperatorNamespace,
		RHSSOProductNamespace,
		RHSSOOperatorNamespace,
		ThreeScaleProductNamespace,
		ThreeScaleOperatorNamespace,
		Marin3rOperatorNamespace,
		Marin3rProductNamespace,
		CustomerGrafanaNamespace,
	}
}

func TestNamespaceCreated(t TestingTB, ctx *TestingContext) {

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

func getNamespaces(t TestingTB, ctx *TestingContext) []string {

	//get RHMI
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Errorf("error getting RHMI CR: %v", err)
	}

	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(rhmi.Spec.Type)) {
		if GetPlatformType(ctx) == string(configv1.GCPPlatformType) {
			return append(mtManagedApiNamespaces(), McgOperatorNamespace)
		}
		return mtManagedApiNamespaces()
	} else {
		if GetPlatformType(ctx) == string(configv1.GCPPlatformType) {
			return append(managedApiNamespaces(), McgOperatorNamespace)
		}
		return managedApiNamespaces()
	}
}
