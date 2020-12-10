package addon

import (
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
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
	log = l.NewLoggerWithContext(l.Fields{l.ComponentLogContext: "addon"})
)

// GetName resolves the add-on name given the installation type
func GetName(installationType integreatlyv1alpha1.InstallationType) string {
	addonName, ok := addonNames[installationType]
	if !ok {
		return RHMI
	}

	return addonName
}
