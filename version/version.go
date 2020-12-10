package version

import (
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"os"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

const (
	installTypeEnvName = "INSTALLATION_TYPE"
)

var (
	version           = "2.7.0"
	managedAPIVersion = "1.1.0"
	log               = l.NewLoggerWithContext(l.Fields{l.ComponentLogContext: "version"})
)

func VerifyProductAndOperatorVersion(product integreatlyv1alpha1.RHMIProductStatus, expectedProductVersion string, expectedOpVersion string) bool {
	installedOpVersion := string(product.OperatorVersion)
	installedProductVersion := string(product.Version)

	if expectedOpVersion != installedOpVersion {
		log.Debugf("Operator Version is not as expected", l.Fields{"product": product.Name, "expected": expectedOpVersion, "actual": installedOpVersion})
		return false
	}
	if expectedProductVersion != installedProductVersion {
		log.Debugf("Product Version is not as expected.", l.Fields{"product": product.Name, "expected": expectedProductVersion, "actual": installedProductVersion})
		return false
	}
	return true
}

func GetVersion() string {
	installTypeEnv, _ := os.LookupEnv(installTypeEnvName)

	return GetVersionByType(installTypeEnv)
}

func GetVersionByType(installType string) string {
	if installType == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		return managedAPIVersion
	} else {
		return version
	}
}
