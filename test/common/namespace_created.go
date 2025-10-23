package common

import (
	goctx "context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func managedApiNamespaces() []string {
	return []string{
		CloudResourceOperatorNamespace,
		RHSSOUserProductNamespace,
		RHSSOUserOperatorNamespace,
		RHSSOProductNamespace,
		RHSSOOperatorNamespace,
		ThreeScaleProductNamespace,
		ThreeScaleOperatorNamespace,
		Marin3rOperatorNamespace,
		Marin3rProductNamespace,
	}
}

func mtManagedApiNamespaces() []string {
	return []string{
		CloudResourceOperatorNamespace,
		RHSSOProductNamespace,
		RHSSOOperatorNamespace,
		ThreeScaleProductNamespace,
		ThreeScaleOperatorNamespace,
		Marin3rOperatorNamespace,
		Marin3rProductNamespace,
	}
}

func TestNamespaceCreated(t TestingTB, ctx *TestingContext) {

	namespacesCreated := getNamespaces(t, ctx)

	var messages []string

	for _, namespace := range namespacesCreated {
		ns := &corev1.Namespace{}
		err := ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: namespace}, ns)

		if err != nil {
			messages = append(messages, fmt.Sprintf("Expected %s namespace to be created but wasn't: %s", namespace, err))
		}
	}

	if messages != nil {
		for _, message := range messages {
			t.Log(message)
		}
		t.Fail()
	}

}

func getNamespaces(t TestingTB, ctx *TestingContext) []string {

	//get RHMI
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Errorf("error getting RHMI CR: %v", err)
	}

	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(rhmi.Spec.Type)) {
		if !resources.IsInProw(rhmi) {
			return append(mtManagedApiNamespaces(), CustomerGrafanaNamespace)
		}
		return mtManagedApiNamespaces()
	} else {
		if !resources.IsInProw(rhmi) {
			return append(managedApiNamespaces(), CustomerGrafanaNamespace)

		}
		return managedApiNamespaces()
	}
}
