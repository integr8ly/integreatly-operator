package mobilesecurityserviceapp

import (
	"context"

	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/aerogear/mobile-security-service-operator/pkg/utils"
	"k8s.io/apimachinery/pkg/types"
)

const FinalizerMetadata = "finalizer.mobile-security-service.aerogear.org"

// hasConditionsToBeDeleted will return true if the Service instance was not found and/or is marked to be deleted
// OR
// if the APP CR was marked to be deleted
func hasConditionsToBeDeleted(mssApp *mobilesecurityservicev1alpha1.MobileSecurityServiceApp, mss *mobilesecurityservicev1alpha1.MobileSecurityService) bool {
	//Check if the APP CR was marked to be deleted
	isAppMarkedToBeDeleted := mssApp.GetDeletionTimestamp() != nil
	hasFinalizer := len(mssApp.GetFinalizers()) > 0
	isMssInstanceDeleted := mss == nil
	isMssInstanceMarkedToBeDeleted := mss.GetDeletionTimestamp() != nil
	return (isAppMarkedToBeDeleted && hasFinalizer) || isMssInstanceDeleted || isMssInstanceMarkedToBeDeleted
}

// isMobileSecurityServiceDeleted return true if it is not found because was deleted and/or was marked to be deleted
func (r *ReconcileMobileSecurityServiceApp) isMobileSecurityServiceDeleted(operatorNamespace string, mss *mobilesecurityservicev1alpha1.MobileSecurityService) bool {
	if err := r.client.Get(context.TODO(), types.NamespacedName{Name: utils.MobileSecurityServiceCRName, Namespace: operatorNamespace}, mss); err != nil || mss.GetDeletionTimestamp() != nil {
		return true
	}
	return false
}
