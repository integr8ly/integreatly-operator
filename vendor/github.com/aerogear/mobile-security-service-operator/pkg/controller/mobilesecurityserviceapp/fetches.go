package mobilesecurityserviceapp

import (
	"context"

	"github.com/aerogear/mobile-security-service-operator/pkg/utils"

	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/aerogear/mobile-security-service-operator/pkg/service"
	"github.com/aerogear/mobile-security-service/pkg/models"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Request object not found, could have been deleted after reconcile request.
// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
func (r *ReconcileMobileSecurityServiceApp) fetchMssAppInstance(reqLogger logr.Logger, request reconcile.Request) (*mobilesecurityservicev1alpha1.MobileSecurityServiceApp, error) {
	reqLogger.Info("Checking if the ReconcileMobileSecurityServiceApp already exists")
	mssApp := &mobilesecurityservicev1alpha1.MobileSecurityServiceApp{}
	//Fetch the MobileSecurityServiceApp mssApp
	err := r.client.Get(context.TODO(), request.NamespacedName, mssApp)
	return mssApp, err
}

// fetchBindAppRestServiceByAppID return app struct from Mobile Security Service Project/REST API or error
var fetchBindAppRestServiceByAppID = func(serviceURL string, mssApp *mobilesecurityservicev1alpha1.MobileSecurityServiceApp, reqLogger logr.Logger) (*models.App, error) {
	return service.GetAppFromServiceByRestApi(serviceURL, mssApp.Spec.AppId, reqLogger)
}

// fetchMssInstance return mss instance applied in the operator namespace
func (r *ReconcileMobileSecurityServiceApp) fetchMssInstance(reqLogger logr.Logger, operatorNamespace string, request reconcile.Request) *mobilesecurityservicev1alpha1.MobileSecurityService {
	mss := &mobilesecurityservicev1alpha1.MobileSecurityService{}
	r.client.Get(context.TODO(), types.NamespacedName{Name: utils.MobileSecurityServiceCRName, Namespace: operatorNamespace}, mss)
	return mss
}
