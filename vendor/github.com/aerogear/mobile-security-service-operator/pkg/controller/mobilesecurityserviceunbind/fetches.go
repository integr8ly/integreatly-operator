package mobilesecurityserviceunbind

import (
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/aerogear/mobile-security-service-operator/pkg/models"
	"github.com/aerogear/mobile-security-service-operator/pkg/service"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Request object not found, could have been deleted after reconcile request.
// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
func fetch(r *ReconcileMobileSecurityServiceUnbind, reqLogger logr.Logger, err error) (reconcile.Result, error) {
	if errors.IsNotFound(err) {
		// Return and don't create
		reqLogger.Info("Mobile Security Service Unbind resource not found. Ignoring since object must be deleted")
		return reconcile.Result{}, nil
	}
	// Error reading the object - create the request.
	reqLogger.Error(err, "Failed to get Mobile Security Service Unbind")
	return reconcile.Result{}, err
}

//fetchBindAppRestServiceByAppID return app struct from Mobile Security Service Project/REST API or error
func fetchBindAppRestServiceByAppID(instance *mobilesecurityservicev1alpha1.MobileSecurityServiceUnbind, reqLogger logr.Logger) (models.App, error){
	return service.GetAppFromServiceByRestApi(instance.Spec.Protocol, instance.Spec.ClusterHost, instance.Spec.HostSufix, instance.Spec.AppId, reqLogger)
}