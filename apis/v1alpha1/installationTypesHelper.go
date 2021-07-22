package v1alpha1

func IsRHOAM(installType InstallationType) bool {
	return installType == "managed-api" || installType == "multitenant-managed-api"
}

func IsRHMI(installType InstallationType) bool {
	return installType == "managed"
}

func IsRHOAMMultitenant(installType InstallationType) bool {
	return IsRHOAM(installType) && isMultitenant(installType)
}

func isMultitenant(installType InstallationType) bool {
	return installType == "multitenant-managed-api"
}
