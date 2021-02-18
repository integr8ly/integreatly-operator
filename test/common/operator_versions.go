package common

import (
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
)

var (
	rhmiProductOperatorVersions = map[integreatlyv1alpha1.StageName]map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.OperatorVersion{
		integreatlyv1alpha1.AuthenticationStage: {
			integreatlyv1alpha1.ProductRHSSO: integreatlyv1alpha1.OperatorVersionRHSSO,
		},
		integreatlyv1alpha1.MonitoringStage: {
			integreatlyv1alpha1.ProductMonitoring: integreatlyv1alpha1.OperatorVersionMonitoring,
		},
		integreatlyv1alpha1.CloudResourcesStage: {
			integreatlyv1alpha1.ProductCloudResources: integreatlyv1alpha1.OperatorVersionCloudResources,
		},
		integreatlyv1alpha1.ProductsStage: {
			integreatlyv1alpha1.Product3Scale:              integreatlyv1alpha1.OperatorVersion3Scale,
			integreatlyv1alpha1.ProductFuse:                integreatlyv1alpha1.OperatorVersionFuse,
			integreatlyv1alpha1.ProductRHSSOUser:           integreatlyv1alpha1.OperatorVersionRHSSOUser,
			integreatlyv1alpha1.ProductCodeReadyWorkspaces: integreatlyv1alpha1.OperatorVersionCodeReadyWorkspaces,
			integreatlyv1alpha1.ProductAMQOnline:           integreatlyv1alpha1.OperatorVersionAMQOnline,
			integreatlyv1alpha1.ProductUps:                 integreatlyv1alpha1.OperatorVersionUPS,
			integreatlyv1alpha1.ProductApicurito:           integreatlyv1alpha1.OperatorVersionApicurito,
		},
		integreatlyv1alpha1.SolutionExplorerStage: {
			integreatlyv1alpha1.ProductSolutionExplorer: integreatlyv1alpha1.OperatorVersionSolutionExplorer,
		},
	}
	managedApiProductOperatorVersions = map[integreatlyv1alpha1.StageName]map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.OperatorVersion{
		integreatlyv1alpha1.AuthenticationStage: {
			integreatlyv1alpha1.ProductRHSSO: integreatlyv1alpha1.OperatorVersionRHSSO,
		},
		integreatlyv1alpha1.MonitoringStage: {
			integreatlyv1alpha1.ProductMonitoring: integreatlyv1alpha1.OperatorVersionMonitoring,
		},
		integreatlyv1alpha1.CloudResourcesStage: {
			integreatlyv1alpha1.ProductCloudResources: integreatlyv1alpha1.OperatorVersionCloudResources,
		},
		integreatlyv1alpha1.ProductsStage: {
			integreatlyv1alpha1.Product3Scale:    integreatlyv1alpha1.OperatorVersion3Scale,
			integreatlyv1alpha1.ProductRHSSOUser: integreatlyv1alpha1.OperatorVersionRHSSOUser,
		},
	}
)

func TestProductOperatorVersions(t TestingTB, ctx *TestingContext) {

	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("failed to get the RHMI: %s", err)
	}

	operatorVersions := getOperatorVersions(rhmi.Spec.Type)

	for stage := range operatorVersions {
		for productName, operatorVersion := range operatorVersions[stage] {
			clusterVersion := rhmi.Status.Stages[stage].Products[productName].OperatorVersion
			if clusterVersion != operatorVersion {
				t.Errorf("Error with version of %s operator deployed on cluster. Expected %s. Got %s", productName, operatorVersion, clusterVersion)
			}
		}

	}
}

func getOperatorVersions(installType string) map[integreatlyv1alpha1.StageName]map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.OperatorVersion {

	if installType == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		return managedApiProductOperatorVersions
	} else {
		return rhmiProductOperatorVersions
	}
}
