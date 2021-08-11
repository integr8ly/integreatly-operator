package v1alpha1

func IsRHOAM(installType InstallationType) bool {
	return installType == InstallationTypeManagedApi || installType == InstallationTypeMultitenantManagedApi
}

func IsRHMI(installType InstallationType) bool {
	return installType == InstallationTypeManaged
}

func IsRHOAMMultitenant(installType InstallationType) bool {
	return IsRHOAM(installType) && isMultitenant(installType)
}

func isMultitenant(installType InstallationType) bool {
	return installType == InstallationTypeMultitenantManagedApi
}

func IsManaged(installType InstallationType) bool {
	return installType == InstallationTypeManaged || installType == InstallationTypeManagedApi || isMultitenant(installType)
}
