package resources

import (
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
)

func IsRHOAM(installType integreatlyv1alpha1.InstallationType) bool {
	return installType == "managed-api" || installType == "multitenant-managed-api"
}

func IsRHMI(installType integreatlyv1alpha1.InstallationType) bool {
	return installType == "managed"
}

func IsRHOAMMultitenant(installType integreatlyv1alpha1.InstallationType) bool {
	return IsRHOAM(installType) && isMultitenant(installType)
}

func isMultitenant(installType integreatlyv1alpha1.InstallationType) bool {
	return installType == "multitenant-managed-api"
}
