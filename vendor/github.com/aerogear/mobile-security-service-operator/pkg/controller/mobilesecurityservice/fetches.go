package mobilesecurityservice

import (
	"context"
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Request object not found, could have been deleted after reconcile request.
// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
func fetch(r *ReconcileMobileSecurityService, reqLogger logr.Logger, err error) (reconcile.Result, error) {
	if errors.IsNotFound(err) {
		// Return and don't create
		reqLogger.Info("Mobile Security Service App resource not found. Ignoring since object must be deleted")
		return reconcile.Result{}, nil
	}
	// Error reading the object - create the request.
	reqLogger.Error(err, "Failed to get Mobile Security Service App")
	return reconcile.Result{}, err
}

//fetchAppIngress returns the ingress resource created for this instance
func (r *ReconcileMobileSecurityService) fetchAppIngress(reqLogger logr.Logger, instance *mobilesecurityservicev1alpha1.MobileSecurityService) (*v1beta1.Ingress, error) {
	reqLogger.Info("Checking if the ingress already exists")
	ingress := &v1beta1.Ingress{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, ingress)
	return ingress, err
}

//fetchAppService returns the service resource created for this instance
func (r *ReconcileMobileSecurityService) fetchAppService(reqLogger logr.Logger, instance *mobilesecurityservicev1alpha1.MobileSecurityService) (*corev1.Service, error) {
	reqLogger.Info("Checking if the service already exists")
	service := &corev1.Service{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, service)
	return service, err
}

//fetchAppDeployment returns the deployment resource created for this instance
func (r *ReconcileMobileSecurityService) fetchAppDeployment(reqLogger logr.Logger, instance *mobilesecurityservicev1alpha1.MobileSecurityService) (*v1beta1.Deployment, error) {
	reqLogger.Info("Checking if the deployment already exists")
	deployment := &v1beta1.Deployment{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, deployment)
	return deployment, err
}

//fetchAppConfigMap returns the config map resource created for this instance
func (r *ReconcileMobileSecurityService) fetchAppConfigMap(reqLogger logr.Logger, instance *mobilesecurityservicev1alpha1.MobileSecurityService) (*corev1.ConfigMap, error) {
	reqLogger.Info("Checking if the ConfigMap already exists")
	configMap := &corev1.ConfigMap{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: getConfigMapName(instance), Namespace: instance.Namespace}, configMap)
	return configMap, err
}


