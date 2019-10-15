package mobilesecurityservicedb

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//updateDBStatus returns error when status regards the all required resources could not be updated
func (r *ReconcileMobileSecurityServiceDB) updateDBStatus(reqLogger logr.Logger, deploymentStatus *appsv1.Deployment, serviceStatus *corev1.Service, pvcStatus *corev1.PersistentVolumeClaim, request reconcile.Request) error {
	reqLogger.Info("Updating App Status for the MobileSecurityServiceDB")

	//Get the latest version of the CR
	instance, err := r.fetchDBInstance(reqLogger, request)
	if err != nil {
		return err
	}

	// Check if ALL required objects are created
	if len(deploymentStatus.Name) < 1 && len(serviceStatus.Name) < 1 && len(pvcStatus.Name) < 1 {
		err := fmt.Errorf("Failed to get OK Status for MobileSecurityService Database")
		reqLogger.Error(err, "One of the resources are not created", "MobileSecurityServiceDB.Namespace", instance.Namespace, "MobileSecurityServiceDB.Name", instance.Name)
		return err
	}
	status := "OK"

	// Update Database Status == OK
	if !reflect.DeepEqual(status, instance.Status.DatabaseStatus) {
		// Get the latest version of the CR in order to try to avoid errors when try to update the CR
		instance, err := r.fetchDBInstance(reqLogger, request)
		if err != nil {
			return err
		}

		// Set the data
		instance.Status.DatabaseStatus = status

		// Update the CR
		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Deployment Status for the MobileSecurityService Database")
			return err
		}
	}
	return nil
}

//updateDeploymentStatus returns error when status regards the Deployment resource could not be updated
func (r *ReconcileMobileSecurityServiceDB) updateDeploymentStatus(reqLogger logr.Logger, request reconcile.Request) (*appsv1.Deployment, error) {
	reqLogger.Info("Updating Deployment Status for the MobileSecurityServiceDB")
	// Get the latest version of the instance CR
	instance, err := r.fetchDBInstance(reqLogger, request)
	if err != nil {
		return nil, err
	}
	// Get the Deployment Object
	deploymentStatus, err := r.fetchDBDeployment(reqLogger, instance)
	if err != nil {
		reqLogger.Error(err, "Failed to get Deployment for Status", "MobileSecurityServiceDB.Namespace", instance.Namespace, "MobileSecurityServiceDB.Name", instance.Name)
		return deploymentStatus, err
	}
	// Update the Deployment Name and Status
	if deploymentStatus.Name != instance.Status.DeploymentName || !reflect.DeepEqual(deploymentStatus.Status, instance.Status.DeploymentStatus) {
		// Get the latest version of the CR in order to try to avoid errors when try to update the CR
		instance, err := r.fetchDBInstance(reqLogger, request)
		if err != nil {
			return nil, err
		}

		// Set the data
		instance.Status.DeploymentName = deploymentStatus.Name
		instance.Status.DeploymentStatus = deploymentStatus.Status

		// Update the CR
		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Deployment Name and Status for the MobileSecurityServiceDB")
			return deploymentStatus, err
		}
	}

	return deploymentStatus, nil
}

//updateServiceStatus returns error when status regards the Service resource could not be updated
func (r *ReconcileMobileSecurityServiceDB) updateServiceStatus(reqLogger logr.Logger, request reconcile.Request) (*corev1.Service, error) {
	reqLogger.Info("Updating Service Status for the MobileSecurityServiceDB")
	// Get the latest version of the instance CR
	instance, err := r.fetchDBInstance(reqLogger, request)
	if err != nil {
		return nil, err
	}
	// Get the Service Object
	serviceStatus, err := r.fetchDBService(reqLogger, instance)
	if err != nil {
		reqLogger.Error(err, "Failed to get Service for Status", "MobileSecurityServiceDB.Namespace", instance.Namespace, "MobileSecurityServiceDB.Name", instance.Name)
		return serviceStatus, err
	}

	if serviceStatus.Name != instance.Status.ServiceName || !reflect.DeepEqual(serviceStatus.Status, instance.Status.ServiceStatus) {
		// Get the latest version of the CR in order to try to avoid errors when try to update the CR
		instance, err := r.fetchDBInstance(reqLogger, request)
		if err != nil {
			return nil, err
		}

		// Set the data
		instance.Status.ServiceName = serviceStatus.Name
		instance.Status.ServiceStatus = serviceStatus.Status

		// Update the CR
		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Service Name and Status for the MobileSecurityServiceDB")
			return serviceStatus, err
		}
	}

	return serviceStatus, nil
}

//updatePvcStatus returns error when status regards the PersistentVolumeClaim resource could not be updated
func (r *ReconcileMobileSecurityServiceDB) updatePvcStatus(reqLogger logr.Logger, request reconcile.Request) (*corev1.PersistentVolumeClaim, error) {
	reqLogger.Info("Updating PersistentVolumeClaim Status for the MobileSecurityServiceDB")
	// Get the latest version of the CR
	instance, err := r.fetchDBInstance(reqLogger, request)
	if err != nil {
		return nil, err
	}

	// Get PVC Object
	pvcStatus, err := r.fetchDBPersistentVolumeClaim(reqLogger, instance)
	if err != nil {
		reqLogger.Error(err, "Failed to get PersistentVolumeClaim for Status", "MobileSecurityServiceDB.Namespace", instance.Namespace, "MobileSecurityServiceDB.Name", instance.Name)
		return pvcStatus, err
	}

	// Update CR with PVC name
	if pvcStatus.Name != instance.Status.PersistentVolumeClaimName {
		// Get the latest version of the CR in order to try to avoid errors when try to update the CR
		instance, err := r.fetchDBInstance(reqLogger, request)
		if err != nil {
			return nil, err
		}

		// Set the data
		instance.Status.PersistentVolumeClaimName = pvcStatus.Name

		// Update the CR
		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update PersistentVolumeClaim Status for the MobileSecurityServiceDB")
			return pvcStatus, err
		}
	}
	return pvcStatus, nil
}

// updateStatusWithInvalidNamespace returns error when status regards the all required resources could not be updated
func (r *ReconcileMobileSecurityServiceDB) updateStatusWithInvalidNamespace(reqLogger logr.Logger, request reconcile.Request) error {
	reqLogger.Info("Updating App Status for the MobileSecurityServiceDB")

	// Get the latest version of CR
	instance, err := r.fetchDBInstance(reqLogger, request)
	if err != nil {
		return err
	}

	status := "Invalid Namespace"

	//Update Bind CR Status with OK
	if !reflect.DeepEqual(status, instance.Status.DatabaseStatus) {
		// Get the latest version of the CR in order to try to avoid errors when try to update the CR
		instance, err := r.fetchDBInstance(reqLogger, request)
		if err != nil {
			return err
		}

		// Set the data
		instance.Status.DatabaseStatus = status

		// Update the CR
		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Status for the MobileSecurityServiceDB")
			return err
		}
	}
	return nil
}
