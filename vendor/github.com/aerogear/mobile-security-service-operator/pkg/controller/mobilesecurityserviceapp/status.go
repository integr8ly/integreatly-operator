package mobilesecurityserviceapp

import (
	"context"
	"fmt"
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"reflect"
)

//updateSDKConfigMapStatus returns error when status regards the ConfigMap resource could not be updated
func (r *ReconcileMobileSecurityServiceApp) updateSDKConfigMapStatus(reqLogger logr.Logger, instance *mobilesecurityservicev1alpha1.MobileSecurityServiceApp) (*corev1.ConfigMap, error) {
	reqLogger.Info("Updating SDKConfigMap Status for the MobileSecurityServiceApp")
	SDKConfigMapStatus, err := r.fetchSDKConfigMap(reqLogger, instance)
	if err != nil {
		reqLogger.Error(err, "Failed to get SDKConfigMap for Status", "MobileSecurityServiceApp.Namespace", instance.Namespace, "MobileSecurityServiceApp.Name", instance.Name)
		return SDKConfigMapStatus, err
	}
	if !reflect.DeepEqual(SDKConfigMapStatus.Name, instance.Status.SDKConfigMapName) {
		instance.Status.SDKConfigMapName = SDKConfigMapStatus.Name
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update SDKConfigMap Status for the MobileSecurityServiceApp")
			return SDKConfigMapStatus, err
		}
	}
	return SDKConfigMapStatus, nil
}

//updateAppStatus returns error when status regards the all required resources could not be updated
func (r *ReconcileMobileSecurityServiceApp) updateBindStatus(reqLogger logr.Logger, SDKConfigMapStatus *corev1.ConfigMap, instance *mobilesecurityservicev1alpha1.MobileSecurityServiceApp) error {
	reqLogger.Info("Updating Bind App Status for the MobileSecurityServiceApp")
	app, err := fetchBindAppRestServiceByAppID(instance, reqLogger)
	if err != nil {
		reqLogger.Error(err, "Failed to get App for Status", "MobileSecurityServiceApp.Namespace", instance.Namespace, "MobileSecurityServiceApp.Name", instance.Name)
		return err
	}
	if len(SDKConfigMapStatus.UID) < 1 && !hasApp(app) {
		err := fmt.Errorf("Failed to get OK Status for MobileSecurityService Bind.")
		reqLogger.Error(err, "One of the resources are not created", "MobileSecurityServiceApp.Namespace", instance.Namespace, "MobileSecurityServiceApp.Name", instance.Name)
		return err
	}
	status:= "OK"
	if !reflect.DeepEqual(status, instance.Status.BindStatus) {
		instance.Status.BindStatus = status
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Status for the MobileSecurityService Bind")
			return err
		}
	}
	return nil
}