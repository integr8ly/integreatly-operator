package common

import (
	"context"
	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getKeycloakNamespaces(installType string) []Namespace {
	ns := []Namespace{}

	rhsso := Namespace{
		Name: NamespacePrefix + "rhsso",
		PodDisruptionBudgetNames: []string{
			"keycloak",
		},
	}

	rhssouser := Namespace{
		Name: NamespacePrefix + "user-sso",
		PodDisruptionBudgetNames: []string{
			"keycloak",
		},
	}

	if rhmiv1alpha1.IsRHOAMMultitenant(rhmiv1alpha1.InstallationType(installType)) {
		return []Namespace{rhsso}
	} else {
		return append(ns, rhsso, rhssouser)
	}
}

func TestIntegreatlyPodDisruptionBudgetsExist(t TestingTB, ctx *TestingContext) {

	rhmi, err := GetRHMI(ctx.Client, true)

	if err != nil {
		t.Fatalf("failed to get the RHMI: %s", err)
	}

	for _, namespace := range getKeycloakNamespaces(rhmi.Spec.Type) {
		for _, podDisruptionBudgetName := range namespace.PodDisruptionBudgetNames {
			_, err := ctx.KubeClient.PolicyV1().PodDisruptionBudgets(namespace.Name).Get(context.TODO(), podDisruptionBudgetName, v1.GetOptions{})
			if err != nil {
				t.Errorf("PodDisruptionBudget %s not found in namespace %s - Error: %s", podDisruptionBudgetName, namespace.Name, err)
			}
		}
	}
}
