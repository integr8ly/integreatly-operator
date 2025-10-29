package common

import (
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
)

var (
	managedApiProductVersions = map[integreatlyv1alpha1.StageName]map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.ProductVersion{
		integreatlyv1alpha1.InstallStage: {
			integreatlyv1alpha1.ProductRHSSO:          integreatlyv1alpha1.VersionRHSSO,
			integreatlyv1alpha1.ProductCloudResources: integreatlyv1alpha1.VersionCloudResources,
			integreatlyv1alpha1.Product3Scale:         integreatlyv1alpha1.Version3Scale,
			integreatlyv1alpha1.ProductRHSSOUser:      integreatlyv1alpha1.VersionRHSSOUser,
		},
	}

	mtManagedApiProductVersions = map[integreatlyv1alpha1.StageName]map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.ProductVersion{
		integreatlyv1alpha1.InstallStage: {
			integreatlyv1alpha1.ProductRHSSO:          integreatlyv1alpha1.VersionRHSSO,
			integreatlyv1alpha1.ProductCloudResources: integreatlyv1alpha1.VersionCloudResources,
			integreatlyv1alpha1.Product3Scale:         integreatlyv1alpha1.Version3Scale,
		},
	}
)

func TestProductVersions(t TestingTB, ctx *TestingContext) {

	rhmi, err := GetRHMI(ctx.Client, true)

	if err != nil {
		t.Fatalf("failed to get the RHMI: %s", err)
	}

	productVersions := getProductVersions(rhmi.Spec.Type)
	var messages []string

	for stage := range productVersions {
		for productName, productVersion := range productVersions[stage] {
			productStatus := rhmi.Status.Stages[stage].Products[productName]
			clusterVersion := productStatus.Version
			if clusterVersion != productVersion {
				messages = append(messages, fmt.Sprintf("Error with version of %s deployed on cluster. Expected %s. Got %s\nProduct status: %v", productName, productVersion, clusterVersion, productStatus))
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

func getProductVersions(installType string) map[integreatlyv1alpha1.StageName]map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.ProductVersion {
	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installType)) {
		return mtManagedApiProductVersions
	} else {
		return managedApiProductVersions
	}
}
