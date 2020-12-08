package version

import (
	"os"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/sirupsen/logrus"
)

const (
	installTypeEnvName = "INSTALLATION_TYPE"
)

var (
	version           = "2.7.0"
	managedAPIVersion = "1.0.1"
)

func VerifyProductAndOperatorVersion(product integreatlyv1alpha1.RHMIProductStatus, expectedProductVersion string, expectedOpVersion string) bool {
	installedOpVersion := string(product.OperatorVersion)
	installedProductVersion := string(product.Version)

	if expectedOpVersion != installedOpVersion {
		logrus.Debugf("%s Operator Version is not as expected. Expected %s, Actual %s", product.Name, expectedOpVersion, installedOpVersion)
		return false
	}
	if expectedProductVersion != installedProductVersion {
		logrus.Debugf("%s Version is not as expected. Expected %s, Actual %s", product.Name, expectedProductVersion, installedProductVersion)
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
