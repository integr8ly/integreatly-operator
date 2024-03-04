package version

import (
	"os"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
)

const (
	installTypeEnvName = "INSTALLATION_TYPE"
)

var (
	managedAPIVersion = "1.41.0"
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
	if integreatlyv1alpha1.IsRHOAM(integreatlyv1alpha1.InstallationType(installType)) {
		return managedAPIVersion
	} else {
		return managedAPIVersion
	}
}
