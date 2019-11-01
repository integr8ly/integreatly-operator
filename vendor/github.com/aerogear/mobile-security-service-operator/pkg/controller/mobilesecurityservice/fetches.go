package mobilesecurityservice

import (
	"context"

	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Request object not found, could have been deleted after reconcile request.
// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
func (r *ReconcileMobileSecurityService) fetchMssInstance(reqLogger logr.Logger, request reconcile.Request) (*mobilesecurityservicev1alpha1.MobileSecurityService, error) {
	reqLogger.Info("Checking if the ReconcileMobileSecurityService already exists")
	mss := &mobilesecurityservicev1alpha1.MobileSecurityService{}
	//Fetch the MobileSecurityService mss
	err := r.client.Get(context.TODO(), request.NamespacedName, mss)
	return mss, err
}

//fetchRoute returns the Route resource created for this instance
func (r *ReconcileMobileSecurityService) fetchRoute(reqLogger logr.Logger, mss *mobilesecurityservicev1alpha1.MobileSecurityService) (*routev1.Route, error) {
	reqLogger.Info("Checking if the route already exists")
	route := &routev1.Route{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: mss.Spec.RouteName, Namespace: mss.Namespace}, route)
	return route, err
}

//fetchServiceAccount returns the ServiceAccount resource created for this instance
func (r *ReconcileMobileSecurityService) fetchServiceAccount(reqLogger logr.Logger, mss *mobilesecurityservicev1alpha1.MobileSecurityService) (*corev1.ServiceAccount, error) {
	reqLogger.Info("Checking if the serviceaccount already exists")
	serviceAccount := &corev1.ServiceAccount{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: mss.Name, Namespace: mss.Namespace}, serviceAccount)
	return serviceAccount, err
}

//fetchService returns the service resource created for this instance
func (r *ReconcileMobileSecurityService) fetchService(reqLogger logr.Logger, mss *mobilesecurityservicev1alpha1.MobileSecurityService, serviceName string) (*corev1.Service, error) {
	reqLogger.Info("Checking if the service already exists")
	service := &corev1.Service{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: serviceName, Namespace: mss.Namespace}, service)
	return service, err
}

//fetchDeployment returns the deployment resource created for this instance
func (r *ReconcileMobileSecurityService) fetchDeployment(reqLogger logr.Logger, mss *mobilesecurityservicev1alpha1.MobileSecurityService) (*appsv1.Deployment, error) {
	reqLogger.Info("Checking if the deployment already exists")
	deployment := &appsv1.Deployment{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: mss.Name, Namespace: mss.Namespace}, deployment)
	return deployment, err
}

//fetchConfigMap returns the config map resource created for this instance
func (r *ReconcileMobileSecurityService) fetchConfigMap(reqLogger logr.Logger, mss *mobilesecurityservicev1alpha1.MobileSecurityService) (*corev1.ConfigMap, error) {
	reqLogger.Info("Checking if the ConfigMap already exists")
	configMap := &corev1.ConfigMap{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: mss.Spec.ConfigMapName, Namespace: mss.Namespace}, configMap)
	return configMap, err
}
