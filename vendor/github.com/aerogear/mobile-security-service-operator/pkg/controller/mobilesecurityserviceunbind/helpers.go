package mobilesecurityserviceunbind

import (
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/aerogear/mobile-security-service-operator/pkg/models"
	"github.com/go-logr/logr"
)

//hasApp return true when APP has ID which is just created by the REST Service API
func hasApp(app models.App) bool {
	return len(app.ID) > 0
}

//Check if the mandatory specs are filled
func hasSpecs(instance *mobilesecurityservicev1alpha1.MobileSecurityServiceUnbind, reqLogger logr.Logger) bool {
	//Check if the cluster host was added in the CR
	if len(instance.Spec.ClusterHost) < 1 || instance.Spec.ClusterHost == "{{clusterHost}}" {
		reqLogger.Info( "Cluster Host IP was not found. Check the Unbind CR configuration or ignore if the object was deleted")
		return false
	}

	if len(instance.Spec.AppId) < 1 {
		reqLogger.Info("AppID was not found. Check the Unbind CR configuration or ignore if the object was deleted")
		return false
	}
	return true
}