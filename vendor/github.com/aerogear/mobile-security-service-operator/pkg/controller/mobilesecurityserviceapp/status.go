package mobilesecurityserviceapp

import (
	"context"
	"fmt"
	"reflect"

	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//updateAppStatus returns error when status regards the all required resources could not be updated
func (r *ReconcileMobileSecurityServiceApp) updateBindStatus(serviceURL string, reqLogger logr.Logger, mssApp *mobilesecurityservicev1alpha1.MobileSecurityServiceApp, request reconcile.Request) error {
	reqLogger.Info("Updating Bind App Status for the MobileSecurityServiceApp")

	// Get the latest version of CR
	mssApp, err := r.fetchMssAppInstance(reqLogger, request)
	if err != nil {
		return err
	}

	// Get App created in the Rest Service
	app, err := fetchBindAppRestServiceByAppID(serviceURL, mssApp, reqLogger)
	if err != nil {
		reqLogger.Error(err, "Failed to get App for Status", "MobileSecurityServiceApp.Namespace", mssApp.Namespace, "MobileSecurityServiceApp.Name", mssApp.Name)
		return err
	}

	// Check if the App is created in the Rest Service
	if len(mssApp.UID) < 1 && app.ID == "" {
		err := fmt.Errorf("failed to get OK Status for MobileSecurityService Bind")
		reqLogger.Error(err, "One of the resources are not created", "MobileSecurityServiceApp.Namespace", mssApp.Namespace, "MobileSecurityServiceApp.Name", mssApp.Name)
		return err
	}
	status := "OK"

	//Update Bind CR Status with OK
	if !reflect.DeepEqual(status, mssApp.Status.BindStatus) {
		// Get the latest version of the CR in order to try to avoid errors when try to update the CR
		instance, err := r.fetchMssAppInstance(reqLogger, request)
		if err != nil {
			return err
		}

		// Set the data
		instance.Status.BindStatus = status

		// Update the CR
		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Status for the MobileSecurityService Bind")
			return err
		}
	}
	return nil
}

// updateBindStatusWithInvalidNamespace returns error when status regards the all required resources could not be updated
func (r *ReconcileMobileSecurityServiceApp) updateBindStatusWithInvalidNamespace(reqLogger logr.Logger, request reconcile.Request) error {
	reqLogger.Info("Updating Bind App Status for the MobileSecurityServiceApp")

	// Get the latest version of CR
	mssApp, err := r.fetchMssAppInstance(reqLogger, request)
	if err != nil {
		return err
	}

	status := "Invalid Namespace"

	//Update Bind CR Status with OK
	if !reflect.DeepEqual(status, mssApp.Status.BindStatus) {
		// Get the latest version of the CR in order to try to avoid errors when try to update the CR
		instance, err := r.fetchMssAppInstance(reqLogger, request)
		if err != nil {
			return err
		}

		// Set the data
		instance.Status.BindStatus = status

		// Update the CR
		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Status for the MobileSecurityService Bind")
			return err
		}
	}
	return nil
}
