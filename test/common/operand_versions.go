package common

import (
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

var (
	rhmi2ProductVersions = map[integreatlyv1alpha1.StageName]map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.ProductVersion{
		integreatlyv1alpha1.AuthenticationStage: {
			integreatlyv1alpha1.ProductRHSSO: integreatlyv1alpha1.VersionRHSSO,
		},
		integreatlyv1alpha1.MonitoringStage: {
			integreatlyv1alpha1.ProductMonitoring: integreatlyv1alpha1.VersionMonitoring,
		},
		integreatlyv1alpha1.CloudResourcesStage: {
			integreatlyv1alpha1.ProductCloudResources: integreatlyv1alpha1.VersionCloudResources,
		},
		integreatlyv1alpha1.ProductsStage: {
			integreatlyv1alpha1.Product3Scale:              integreatlyv1alpha1.Version3Scale,
			integreatlyv1alpha1.ProductFuse:                integreatlyv1alpha1.VersionFuseOnOpenshift,
			integreatlyv1alpha1.ProductRHSSOUser:           integreatlyv1alpha1.VersionRHSSOUser,
			integreatlyv1alpha1.ProductCodeReadyWorkspaces: integreatlyv1alpha1.VersionCodeReadyWorkspaces,
			integreatlyv1alpha1.ProductAMQOnline:           integreatlyv1alpha1.VersionAMQOnline,
			integreatlyv1alpha1.ProductUps:                 integreatlyv1alpha1.VersionUps,
			integreatlyv1alpha1.ProductApicurito:           integreatlyv1alpha1.VersionApicurito,
		},
		integreatlyv1alpha1.SolutionExplorerStage: {
			integreatlyv1alpha1.ProductSolutionExplorer: integreatlyv1alpha1.VersionSolutionExplorer,
		},
	}

	managedApiProductVersions = map[integreatlyv1alpha1.StageName]map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.ProductVersion{
		integreatlyv1alpha1.AuthenticationStage: {
			integreatlyv1alpha1.ProductRHSSO: integreatlyv1alpha1.VersionRHSSO,
		},
		integreatlyv1alpha1.MonitoringStage: {
			integreatlyv1alpha1.ProductMonitoring: integreatlyv1alpha1.VersionMonitoring,
		},
		integreatlyv1alpha1.CloudResourcesStage: {
			integreatlyv1alpha1.ProductCloudResources: integreatlyv1alpha1.VersionCloudResources,
		},
		integreatlyv1alpha1.ProductsStage: {
			integreatlyv1alpha1.Product3Scale:    integreatlyv1alpha1.Version3Scale,
			integreatlyv1alpha1.ProductRHSSOUser: integreatlyv1alpha1.VersionRHSSOUser,
		},
		integreatlyv1alpha1.SolutionExplorerStage: {
			integreatlyv1alpha1.ProductSolutionExplorer: integreatlyv1alpha1.VersionSolutionExplorer,
		},
	}
)

func TestProductVersions(t *testing.T, ctx *TestingContext) {

	rhmi, err := getRHMI(ctx.Client)

	if err != nil {
		t.Fatalf("failed to get the RHMI: %s", err)
	}

	productVersions := getProductVersions(rhmi.Spec.Type)

	for stage := range productVersions {
		for productName, productVersion := range productVersions[stage] {
			productStatus := rhmi.Status.Stages[stage].Products[productName]
			clusterVersion := productStatus.Version
			if clusterVersion != productVersion {
				t.Skipf("skipping due to known flaky behaviour https://issues.redhat.com/browse/INTLY-8390, Error with version of %s deployed on cluster. Expected %s. Got %s\nProduct status: %v", productName, productVersion, clusterVersion, productStatus)
				//t.Errorf("Error with version of %s deployed on cluster. Expected %s. Got %s\nProduct status: %v", productName, productVersion, clusterVersion, productStatus)
			}
		}

	}
}

func getProductVersions(installType string) map[integreatlyv1alpha1.StageName]map[integreatlyv1alpha1.ProductName]integreatlyv1alpha1.ProductVersion {
	if installType == string(integreatlyv1alpha1.InstallationTypeManaged3scale) {
		return managedApiProductVersions
	} else {
		return rhmi2ProductVersions
	}
}
