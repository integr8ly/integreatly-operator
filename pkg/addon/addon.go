package addon

import (
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

const (
	ManagedAPIService = "managed-api-service"
	RHMI              = "rhmi"
)

var (
	addonNames = map[integreatlyv1alpha1.InstallationType]string{
		integreatlyv1alpha1.InstallationTypeManagedApi: ManagedAPIService,
		integreatlyv1alpha1.InstallationTypeManaged:    RHMI,
	}
)

// GetName resolves the add-on name given the installation type
func GetName(installationType integreatlyv1alpha1.InstallationType) string {
	addonName, ok := addonNames[installationType]
	if !ok {
		return RHMI
	}

	return addonName
}
