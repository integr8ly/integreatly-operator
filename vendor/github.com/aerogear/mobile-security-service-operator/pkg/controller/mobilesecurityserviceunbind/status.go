package mobilesecurityserviceunbind

import (
	"context"
	"fmt"
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/go-logr/logr"
	"reflect"
)

//updateAppStatus returns error when status regards the all required resources could not be updated
func (r *ReconcileMobileSecurityServiceUnbind) updateUnbindStatus(reqLogger logr.Logger, instance *mobilesecurityservicev1alpha1.MobileSecurityServiceUnbind) error {
	reqLogger.Info("Updating Unbind App Status for the MobileSecurityServiceUnbind")
	if app, err := fetchBindAppRestServiceByAppID(instance, reqLogger);  err != nil || hasApp(app){
		if hasApp(app) {
			err := fmt.Errorf("App was found in the REST Service API")
			reqLogger.Error(err, "Failed to update Unbind App status", "App.appId", instance.Spec.AppId)
			return err
		}
		return err
	}
	status:= "OK"
	if !reflect.DeepEqual(status, instance.Status.UnbindStatus) {
		instance.Status.UnbindStatus = status
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Status for the MobileSecurityService Bind")
			return err
		}
	}

	return nil
}
