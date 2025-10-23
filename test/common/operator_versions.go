package common

import (
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
)

var (
	managedApiProductOperatorVersions = map[integreatlyv1alpha1.StageName]map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.OperatorVersion{
		integreatlyv1alpha1.InstallStage: {
			integreatlyv1alpha1.ProductRHSSO:          integreatlyv1alpha1.OperatorVersionRHSSO,
			integreatlyv1alpha1.ProductCloudResources: integreatlyv1alpha1.OperatorVersionCloudResources,
			integreatlyv1alpha1.Product3Scale:         integreatlyv1alpha1.OperatorVersion3Scale,
			integreatlyv1alpha1.ProductRHSSOUser:      integreatlyv1alpha1.OperatorVersionRHSSOUser,
		},
	}
	mtManagedApiProductOperatorVersions = map[integreatlyv1alpha1.StageName]map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.OperatorVersion{
		integreatlyv1alpha1.InstallStage: {
			integreatlyv1alpha1.ProductRHSSO:          integreatlyv1alpha1.OperatorVersionRHSSO,
			integreatlyv1alpha1.ProductCloudResources: integreatlyv1alpha1.OperatorVersionCloudResources,
			integreatlyv1alpha1.Product3Scale:         integreatlyv1alpha1.OperatorVersion3Scale,
		},
	}
)

func TestProductOperatorVersions(t TestingTB, ctx *TestingContext) {

	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("failed to get the RHMI: %s", err)
	}

	operatorVersions := getOperatorVersions(rhmi.Spec.Type)
	var messages []string

	for stage := range operatorVersions {
		for productName, operatorVersion := range operatorVersions[stage] {
			clusterVersion := rhmi.Status.Stages[stage].Products[productName].OperatorVersion
			if clusterVersion != operatorVersion {
				messages = append(messages, fmt.Sprintf("Error with version of %s operator deployed on cluster. Expected %s. Got %s", productName, operatorVersion, clusterVersion))
			}
		}
	}

	if messages != nil {
		for _, message := range messages {
			t.Log(message)
		}
		t.Fail()
	}
}

func getOperatorVersions(installType string) map[integreatlyv1alpha1.StageName]map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.OperatorVersion {
	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installType)) {
		return mtManagedApiProductOperatorVersions
	} else {
		return managedApiProductOperatorVersions
	}
}
