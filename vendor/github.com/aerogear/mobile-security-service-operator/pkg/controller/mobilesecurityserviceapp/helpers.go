package mobilesecurityserviceapp

import (
	"encoding/json"
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/aerogear/mobile-security-service-operator/pkg/models"
	"github.com/go-logr/logr"
)

const SDK  = "-sdk"

// Returns an string map with the labels which wil be associated to the kubernetes/openshift objects
// which will be created and managed by this operator
func getAppLabels(name string) map[string]string {
	return map[string]string{"app": "mobilesecurityservice", "mobilesecurityserviceapp_cr": name}
}

//To transform the object into a string with its json
func getSdkConfigStringJsonFormat(sdk *models.SDKConfig) string{
	jsonSdk, _ := json.MarshalIndent(sdk, "", "\t")
	return string(jsonSdk)
}

// return properties for the response SDK
func getConfigMapSDKForMobileSecurityService(m *mobilesecurityservicev1alpha1.MobileSecurityServiceApp) map[string]string {
	sdk := models.NewSDKConfig(m)
	return map[string]string{
		"SDKConfig": getSdkConfigStringJsonFormat(sdk),
	}
}

// return properties for the response SDK
func getConfigMapName(m *mobilesecurityservicev1alpha1.MobileSecurityServiceApp) string {
	return m.Spec.AppName + SDK
}

//hasApp return true when APP has ID which is just created by the REST Service API
func hasApp(app models.App) bool {
	return len(app.ID) > 0
}

//Check if the mandatory specs are filled
func hasSpecs(instance *mobilesecurityservicev1alpha1.MobileSecurityServiceApp, reqLogger logr.Logger) bool {
	//Check if the cluster host was added in the CR
	if len(instance.Spec.ClusterHost) < 1 || instance.Spec.ClusterHost == "{{clusterHost}}" {
		reqLogger.Info( "Cluster Host IP was not found. Check the App CR configuration or ignore if the object was deleted")
		return false
	}

	if len(instance.Spec.AppId) < 1 {
		reqLogger.Info("AppID was not found. Check the App CR configuration or ignore if the object was deleted")
		return false
	}
	return true
}