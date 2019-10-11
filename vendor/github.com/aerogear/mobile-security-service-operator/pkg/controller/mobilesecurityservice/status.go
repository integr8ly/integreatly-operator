package mobilesecurityservice

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
func (r *ReconcileMobileSecurityService) updateAppStatus(reqLogger logr.Logger, configMapStatus *corev1.ConfigMap, deploymentStatus *v1beta1.Deployment, serviceStatus *corev1.Service, ingressStatus *v1beta1.Ingress, instance *mobilesecurityservicev1alpha1.MobileSecurityService) error {
	reqLogger.Info("Updating App Status for the MobileSecurityService")
	if len(configMapStatus.UID) < 1 && len(deploymentStatus.UID) < 1 && len(serviceStatus.UID) < 1 && len(ingressStatus.Name) < 1 {
		err := fmt.Errorf("Failed to get OK Status for MobileSecurityService App")
		reqLogger.Error(err, "One of the resources are not created", "MobileSecurityService.Namespace", instance.Namespace, "MobileSecurityService.Name", instance.Name)
		return err
	}
	status:= "OK"
	if !reflect.DeepEqual(status, instance.Status.AppStatus) {
		instance.Status.AppStatus = status
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Status for the MobileSecurityService App")
			return err
		}
	}
	return nil
}

//updateConfigMapStatus returns error when status regards the ConfigMap resource could not be updated
func (r *ReconcileMobileSecurityService) updateConfigMapStatus(reqLogger logr.Logger, instance *mobilesecurityservicev1alpha1.MobileSecurityService) (*corev1.ConfigMap, error) {
	reqLogger.Info("Updating ConfigMap Status for the MobileSecurityService")
	configMapStatus, err := r.fetchAppConfigMap(reqLogger, instance)
	if err != nil {
		reqLogger.Error(err, "Failed to get ConfigMap Name for Status", "MobileSecurityService.Namespace", instance.Namespace, "MobileSecurityService.Name", instance.Name)
		return configMapStatus, err
	}
	if !reflect.DeepEqual(configMapStatus.Name, instance.Status.ConfigMapName) {
		instance.Status.ConfigMapName = configMapStatus.Name
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update ConfigMap Name Status for the MobileSecurityService")
			return configMapStatus, err
		}
	}
	return configMapStatus, nil
}

//updateDeploymentStatus returns error when status regards the Deployment resource could not be updated
func (r *ReconcileMobileSecurityService) updateDeploymentStatus(reqLogger logr.Logger, instance *mobilesecurityservicev1alpha1.MobileSecurityService) (*v1beta1.Deployment, error) {
	reqLogger.Info("Updating Deployment Status for the MobileSecurityService")
	deploymentStatus, err := r.fetchAppDeployment(reqLogger, instance)
	if err != nil {
		reqLogger.Error(err, "Failed to get Deployment for Status", "MobileSecurityService.Namespace", instance.Namespace, "MobileSecurityService.Name", instance.Name)
		return deploymentStatus, err
	}
	if !reflect.DeepEqual(deploymentStatus.Name, instance.Status.DeploymentName) {
		instance.Status.DeploymentName = deploymentStatus.Name
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Deployment Name Status for the MobileSecurityService")
			return deploymentStatus, err
		}
	}
	if !reflect.DeepEqual(deploymentStatus.Status, instance.Status.DeploymentStatus) {
		instance.Status.DeploymentStatus = deploymentStatus.Status
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Deployment Status for the MobileSecurityService")
			return deploymentStatus, err
		}
	}
	return deploymentStatus, nil
}

//updateServiceStatus returns error when status regards the Service resource could not be updated
func (r *ReconcileMobileSecurityService) updateServiceStatus(reqLogger logr.Logger, instance *mobilesecurityservicev1alpha1.MobileSecurityService) (*corev1.Service, error) {
	reqLogger.Info("Updating Service Status for the MobileSecurityService")
	serviceStatus, err := r.fetchAppService(reqLogger, instance)
	if err != nil {
		reqLogger.Error(err, "Failed to get Service for Status", "MobileSecurityService.Namespace", instance.Namespace, "MobileSecurityService.Name", instance.Name)
		return serviceStatus, err
	}
	if !reflect.DeepEqual(serviceStatus.Name, instance.Status.ServiceName) {
		instance.Status.ServiceName = serviceStatus.Name
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Service Name Status for the MobileSecurityService")
			return serviceStatus, err
		}
	}
	if !reflect.DeepEqual(serviceStatus.Status, instance.Status.ServiceStatus) {
		instance.Status.ServiceStatus = serviceStatus.Status
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Service Status for the MobileSecurityService")
			return serviceStatus, err
		}
	}
	return serviceStatus, nil
}

//updateIngressStatus returns error when status regards the ingress resource could not be updated
func (r *ReconcileMobileSecurityService) updateIngressStatus(reqLogger logr.Logger, instance *mobilesecurityservicev1alpha1.MobileSecurityService) (*v1beta1.Ingress, error) {
	reqLogger.Info("Updating Ingress Status for the MobileSecurityService")
	ingressStatus, err := r.fetchAppIngress(reqLogger, instance)
	if err != nil {
		reqLogger.Error(err, "Failed to get Ingress for Status", "MobileSecurityService.Namespace", instance.Namespace, "MobileSecurityService.Name", instance.Name)
		return ingressStatus, err
	}
	if !reflect.DeepEqual(ingressStatus.Name, instance.Status.IngressName) {
		instance.Status.IngressName = ingressStatus.Name
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Ingress Name Status for the MobileSecurityService")
			return ingressStatus, err
		}
	}
	if !reflect.DeepEqual(ingressStatus.Status, instance.Status.IngressStatus) {
		instance.Status.IngressStatus = ingressStatus.Status
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Ingress Status for the MobileSecurityService")
			return ingressStatus, err
		}
	}
	return ingressStatus, nil
}


