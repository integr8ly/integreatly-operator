package mobilesecurityserviceapp

import (
	"context"
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/aerogear/mobile-security-service-operator/pkg/service"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"github.com/aerogear/mobile-security-service-operator/pkg/models"
)

// Request object not found, could have been deleted after reconcile request.
// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
func fetch(r *ReconcileMobileSecurityServiceApp, reqLogger logr.Logger, err error) (reconcile.Result, error) {
	if errors.IsNotFound(err) {
		// Return and don't create
		reqLogger.Info("Mobile Security Service App resource not found. Ignoring since object must be deleted")
		return reconcile.Result{}, nil
	}
	// Error reading the object - create the request.
	reqLogger.Error(err, "Failed to get Mobile Security Service App")
	return reconcile.Result{}, err
}

//fetchSDKConfigMap returns the config map resource created for this instance
func (r *ReconcileMobileSecurityServiceApp) fetchSDKConfigMap(reqLogger logr.Logger, instance *mobilesecurityservicev1alpha1.MobileSecurityServiceApp) (*corev1.ConfigMap, error) {
	reqLogger.Info("Checking if the ConfigMap already exists")
	configMap := &corev1.ConfigMap{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: getConfigMapName(instance), Namespace: instance.Namespace}, configMap)
	return configMap, err
}

//fetchBindAppRestServiceByAppID return app struct from Mobile Security Service Project/REST API or error
func fetchBindAppRestServiceByAppID(instance *mobilesecurityservicev1alpha1.MobileSecurityServiceApp, reqLogger logr.Logger) (models.App, error){
	return service.GetAppFromServiceByRestApi(instance.Spec.Protocol, instance.Spec.ClusterHost, instance.Spec.HostSufix, instance.Spec.AppId, reqLogger)
}