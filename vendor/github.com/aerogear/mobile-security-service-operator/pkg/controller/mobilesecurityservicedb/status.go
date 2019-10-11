package mobilesecurityservicedb

import (
	"context"
	"fmt"
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"reflect"
)

//updateAppStatus returns error when status regards the all required resources could not be updated
func (r *ReconcileMobileSecurityServiceDB) updateDBStatus(reqLogger logr.Logger, deploymentStatus *v1beta1.Deployment, serviceStatus *corev1.Service, pvcStatus *corev1.PersistentVolumeClaim, instance *mobilesecurityservicev1alpha1.MobileSecurityServiceDB) error {
	reqLogger.Info("Updating App Status for the MobileSecurityServiceDB")
	if len(deploymentStatus.Name) < 1 && len(serviceStatus.Name) < 1 && len(pvcStatus.Name) < 1 {
		err := fmt.Errorf("Failed to get OK Status for MobileSecurityService Database")
		reqLogger.Error(err, "One of the resources are not created", "MobileSecurityServiceDB.Namespace", instance.Namespace, "MobileSecurityServiceDB.Name", instance.Name)
		return err
	}
	status:= "OK"
	if !reflect.DeepEqual(status, instance.Status.DatabaseStatus) {
		instance.Status.DatabaseStatus = status
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Status for the MobileSecurityService Database")
			return err
		}
	}
	return nil
}

//updateDeploymentStatus returns error when status regards the Deployment resource could not be updated
func (r *ReconcileMobileSecurityServiceDB) updateDeploymentStatus(reqLogger logr.Logger, instance *mobilesecurityservicev1alpha1.MobileSecurityServiceDB) (*v1beta1.Deployment, error) {
	reqLogger.Info("Updating Deployment Status for the MobileSecurityServiceDB")
	deploymentStatus, err := r.fetchDBDeployment(reqLogger, instance)
	if err != nil {
		reqLogger.Error(err, "Failed to get Deployment for Status", "MobileSecurityServiceDB.Namespace", instance.Namespace, "MobileSecurityServiceDB.Name", instance.Name)
		return deploymentStatus, err
	}
	if !reflect.DeepEqual(deploymentStatus.Name, instance.Status.DeploymentName) {
		instance.Status.DeploymentName = deploymentStatus.Name
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Deployment Name Status for the MobileSecurityServiceDB")
			return deploymentStatus, err
		}
	}
	if !reflect.DeepEqual(deploymentStatus.Status, instance.Status.DeploymentStatus) {
		instance.Status.DeploymentStatus = deploymentStatus.Status
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Deployment Status for the MobileSecurityServiceDB")
			return deploymentStatus, err
		}
	}
	return deploymentStatus, nil
}

//updateServiceStatus returns error when status regards the Service resource could not be updated
func (r *ReconcileMobileSecurityServiceDB) updateServiceStatus(reqLogger logr.Logger, instance *mobilesecurityservicev1alpha1.MobileSecurityServiceDB) (*corev1.Service, error) {
	reqLogger.Info("Updating Service Status for the MobileSecurityServiceDB")
	serviceStatus, err := r.fetchDBService(reqLogger, instance)
	if err != nil {
		reqLogger.Error(err, "Failed to get Service for Status", "MobileSecurityServiceDB.Namespace", instance.Namespace, "MobileSecurityServiceDB.Name", instance.Name)
		return serviceStatus, err
	}
	if !reflect.DeepEqual(serviceStatus.Name, instance.Status.ServiceName) {
		instance.Status.ServiceName = serviceStatus.Name
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Service Name Status for the MobileSecurityServiceDB")
			return serviceStatus, err
		}
	}
	if !reflect.DeepEqual(serviceStatus.Status, instance.Status.ServiceStatus) {
		instance.Status.ServiceStatus = serviceStatus.Status
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Service Status for the MobileSecurityServiceDB")
			return serviceStatus, err
		}
	}
	return serviceStatus, nil
}

//updatePvcStatus returns error when status regards the PersistentVolumeClaim resource could not be updated
func (r *ReconcileMobileSecurityServiceDB) updatePvcStatus(reqLogger logr.Logger, instance *mobilesecurityservicev1alpha1.MobileSecurityServiceDB) (*corev1.PersistentVolumeClaim, error) {
	reqLogger.Info("Updating PersistentVolumeClaim Status for the MobileSecurityServiceDB")
	pvcStatus, err := r.fetchDBPersistentVolumeClaim(reqLogger, instance)
	if err != nil {
		reqLogger.Error(err, "Failed to get PersistentVolumeClaim for Status", "MobileSecurityServiceDB.Namespace", instance.Namespace, "MobileSecurityServiceDB.Name", instance.Name)
		return pvcStatus, err
	}
	if !reflect.DeepEqual(pvcStatus.Name, instance.Status.PersistentVolumeClaimName) {
		instance.Status.PersistentVolumeClaimName = pvcStatus.Name
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update PersistentVolumeClaim Status for the MobileSecurityServiceDB")
			return pvcStatus, err
		}
	}
	return pvcStatus, nil
}

