package mobilesecurityserviceapp

import (
	"context"

	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// addFinalizer will add the Finalizer metadata in the Mobile Security Service App CR
func (r *ReconcileMobileSecurityServiceApp) addFinalizer(reqLogger logr.Logger, mssApp *mobilesecurityservicev1alpha1.MobileSecurityServiceApp, request reconcile.Request) error {
	if len(mssApp.GetFinalizers()) < 1 && mssApp.GetDeletionTimestamp() == nil {
		reqLogger.Info("Adding Finalizer for the MobileSecurityServiceApp")

		// Get the latest version of CR
		mssApp, err := r.fetchMssAppInstance(reqLogger, request)
		if err != nil {
			return err
		}

		//Set finalizer string/metadata
		mssApp.SetFinalizers([]string{FinalizerMetadata})

		//Update CR
		err = r.client.Update(context.TODO(), mssApp)
		if err != nil {
			reqLogger.Error(err, "Failed to update MobileSecurityService App CR with  finalizer")
			return err
		}
	}
	return nil
}

// handleFinalizer returns error when the app still not deleted in the REST Service
func (r *ReconcileMobileSecurityServiceApp) handleFinalizer(serviceAPI string, reqLogger logr.Logger, request reconcile.Request) error {

	// Get the latest version of CR
	mssApp, err := r.fetchMssAppInstance(reqLogger, request)
	if err != nil {
		return err
	}

	if len(mssApp.GetFinalizers()) > 0 && mssApp.GetDeletionTimestamp() != nil {
		reqLogger.Info("Removing Finalizer for the MobileSecurityServiceApp")
		if app, err := fetchBindAppRestServiceByAppID(serviceAPI, mssApp, reqLogger); err != nil || app.ID != "" {
			reqLogger.Error(err, "Unable to delete app from the service sid", "mssApp.appId", mssApp.Spec.AppId, "app.ID", app.ID)
			return err
		}

		if err := r.removeFinalizerFromCR(mssApp); err != nil {
			reqLogger.Error(err, "Failed to update MobileSecurityService App CR with finalizer")
			return err
		}
	}
	return nil
}

// removeFinalizerFromCR return an error when is not possible remove the finalizer metadata from the app instance
func (r *ReconcileMobileSecurityServiceApp) removeFinalizerFromCR(mssApp *mobilesecurityservicev1alpha1.MobileSecurityServiceApp) error {
	//Remove finalizer
	mssApp.SetFinalizers(nil)

	//Update CR
	err := r.client.Update(context.TODO(), mssApp)
	if err != nil {
		return err
	}
	return nil
}
