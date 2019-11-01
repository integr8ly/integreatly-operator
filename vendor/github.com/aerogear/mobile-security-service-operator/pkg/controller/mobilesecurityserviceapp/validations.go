package mobilesecurityserviceapp

import (
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/go-logr/logr"
)

//Check if the mandatory specs are filled
func hasMandatorySpecs(mssApp *mobilesecurityservicev1alpha1.MobileSecurityServiceApp, reqLogger logr.Logger) bool {
	//Check if the appId was added in the CR
	if len(mssApp.Spec.AppId) < 1 {
		reqLogger.Info("AppID was not found. Check the App CR configuration.")
		return false
	}

	//Check if the appName was added in the CR
	if len(mssApp.Spec.AppName) < 1 {
		reqLogger.Info("AppName was not found. Check the App CR configuration.")
		return false
	}

	return true
}
