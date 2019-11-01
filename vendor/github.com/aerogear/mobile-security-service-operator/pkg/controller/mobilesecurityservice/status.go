package mobilesecurityservice

import (
	"context"
	"fmt"
	"reflect"

	"github.com/aerogear/mobile-security-service-operator/pkg/utils"
	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//updateStatus returns error when status regards the all required resources could not be updated
func (r *ReconcileMobileSecurityService) updateStatus(reqLogger logr.Logger, configMapStatus *corev1.ConfigMap, deploymentStatus *appsv1.Deployment, proxyServiceStatus *corev1.Service, applicationServiceStatus *corev1.Service, routeStatus *routev1.Route, request reconcile.Request) error {
	reqLogger.Info("Updating App Status for the MobileSecurityService")
	// Get the latest version of the CR
	mss, err := r.fetchMssInstance(reqLogger, request)
	if err != nil {
		return err
	}

	//Check if all required objects are created
	if len(configMapStatus.UID) < 1 && len(deploymentStatus.UID) < 1 && len(proxyServiceStatus.UID) < 1 && len(applicationServiceStatus.UID) < 1 && len(routeStatus.Name) < 1 {
		err := fmt.Errorf("Failed to get OK Status for MobileSecurityService")
		reqLogger.Error(err, "One of the resources are not created", "MobileSecurityService.Namespace", mss.Namespace, "MobileSecurityService.Name", mss.Name)
		return err
	}
	status := "OK"

	// Update CR with the AppStatus == OK
	if !reflect.DeepEqual(status, mss.Status.AppStatus) {
		// Set the data
		mss.Status.AppStatus = status

		// Update the CR
		err = r.client.Status().Update(context.TODO(), mss)
		if err != nil {
			reqLogger.Error(err, "Failed to update Project Status for the MobileSecurityService")
			return err
		}
	}
	return nil
}

// updateBindStatusWithInvalidNamespace returns error when status regards the all required resources could not be updated
func (r *ReconcileMobileSecurityService) updateStatusWithInvalidNamespace(reqLogger logr.Logger, request reconcile.Request) error {
	reqLogger.Info("Updating Bind App Status for the MobileSecurityServiceApp")

	// Get the latest version of CR
	mss, err := r.fetchMssInstance(reqLogger, request)
	if err != nil {
		return err
	}

	status := "Invalid Namespace"

	//Update Bind CR Status with OK
	if !reflect.DeepEqual(status, mss.Status.AppStatus) {
		// Get the latest version of the CR in order to try to avoid errors when try to update the CR
		instance, err := r.fetchMssInstance(reqLogger, request)
		if err != nil {
			return err
		}

		// Set the data
		instance.Status.AppStatus = status

		// Update the CR
		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Status for the MobileSecurityService Bind")
			return err
		}
	}
	return nil
}

//updateConfigMapStatus returns error when status regards the ConfigMap resource could not be updated
func (r *ReconcileMobileSecurityService) updateConfigMapStatus(reqLogger logr.Logger, request reconcile.Request) (*corev1.ConfigMap, error) {
	reqLogger.Info("Updating ConfigMap Status for the MobileSecurityService")
	// Get the latest version of the CR
	mss, err := r.fetchMssInstance(reqLogger, request)
	if err != nil {
		return nil, err
	}

	// Get the ConfigMap object
	configMapStatus, err := r.fetchConfigMap(reqLogger, mss)
	if err != nil {
		reqLogger.Error(err, "Failed to get ConfigMap Name for Status", "MobileSecurityService.Namespace", mss.Namespace, "MobileSecurityService.Name", mss.Name)
		return configMapStatus, err
	}

	// Update ConfigMap Name
	if configMapStatus.Name != mss.Status.ConfigMapName {
		// Get the latest version of the CR in order to try to avoid errors when try to update the CR
		instance, err := r.fetchMssInstance(reqLogger, request)
		if err != nil {
			return nil, err
		}

		// Set the data
		instance.Status.ConfigMapName = configMapStatus.Name

		// Update the CR
		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update ConfigMap Name and Status for the MobileSecurityService")
			return configMapStatus, err
		}
	}
	return configMapStatus, nil
}

//updateDeploymentStatus returns error when status regards the Deployment resource could not be updated
func (r *ReconcileMobileSecurityService) updateDeploymentStatus(reqLogger logr.Logger, request reconcile.Request) (*appsv1.Deployment, error) {
	reqLogger.Info("Updating Deployment Status for the MobileSecurityService")
	// Get the latest version of the CR
	mss, err := r.fetchMssInstance(reqLogger, request)
	if err != nil {
		return nil, err
	}

	// Get the deployment object
	deploymentStatus, err := r.fetchDeployment(reqLogger, mss)
	if err != nil {
		reqLogger.Error(err, "Failed to get Deployment for Status", "MobileSecurityService.Namespace", mss.Namespace, "MobileSecurityService.Name", mss.Name)
		return deploymentStatus, err
	}

	// Update the Deployment Name and Status
	if deploymentStatus.Name != mss.Status.DeploymentName || !reflect.DeepEqual(deploymentStatus.Status, mss.Status.DeploymentStatus) {
		// Get the latest version of the CR in order to try to avoid errors when try to update the CR
		instance, err := r.fetchMssInstance(reqLogger, request)
		if err != nil {
			return nil, err
		}

		// Set the data
		instance.Status.DeploymentName = deploymentStatus.Name
		instance.Status.DeploymentStatus = deploymentStatus.Status

		// Update the CR
		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Deployment Name and Status for the MobileSecurityService")
			return deploymentStatus, err
		}
	}

	return deploymentStatus, nil
}

//updateAppServiceStatus returns error when status regards the Service resource could not be updated
func (r *ReconcileMobileSecurityService) updateAppServiceStatus(reqLogger logr.Logger, request reconcile.Request) (*corev1.Service, error) {
	reqLogger.Info("Updating App Service Status for the MobileSecurityService")
	// Get the latest version of the CR
	mss, err := r.fetchMssInstance(reqLogger, request)
	if err != nil {
		return nil, err
	}
	// Get the Service Object
	serviceStatus, err := r.fetchService(reqLogger, mss, utils.ApplicationServiceInstanceName)
	if err != nil {
		reqLogger.Error(err, "Failed to get App Service for Status", "MobileSecurityService.Namespace", mss.Namespace, "MobileSecurityService.Name", mss.Name)
		return serviceStatus, err
	}

	// Update the Deployment Name and Status
	if serviceStatus.Name != mss.Status.ServiceName || !reflect.DeepEqual(serviceStatus.Status, mss.Status.ServiceStatus) {
		// Get the latest version of the CR in order to try to avoid errors when try to update the CR
		mss, err := r.fetchMssInstance(reqLogger, request)
		if err != nil {
			return nil, err
		}

		// Set the data
		mss.Status.ServiceName = serviceStatus.Name
		mss.Status.ServiceStatus = serviceStatus.Status

		// Update the CR
		err = r.client.Status().Update(context.TODO(), mss)
		if err != nil {
			reqLogger.Error(err, "Failed to update App Service Name and Status for the MobileSecurityService")
			return serviceStatus, err
		}
	}

	return serviceStatus, nil
}

//updateAppServiceStatus returns error when status regards the Service resource could not be updated
func (r *ReconcileMobileSecurityService) updateProxyServiceStatus(reqLogger logr.Logger, request reconcile.Request) (*corev1.Service, error) {
	reqLogger.Info("Updating Proxy Service Status for the MobileSecurityService")
	// Get the latest version of the CR
	mss, err := r.fetchMssInstance(reqLogger, request)
	if err != nil {
		return nil, err
	}
	// Get the Service Object
	serviceStatus, err := r.fetchService(reqLogger, mss, utils.ProxyServiceInstanceName)
	if err != nil {
		reqLogger.Error(err, "Failed to get Proxy Service for Status", "MobileSecurityService.Namespace", mss.Namespace, "MobileSecurityService.Name", mss.Name)
		return serviceStatus, err
	}

	// Update the Service Status and Name
	if serviceStatus.Name != mss.Status.ProxyServiceName || !reflect.DeepEqual(serviceStatus.Status, mss.Status.ProxyServiceStatus) {
		// Get the latest version of the CR in order to try to avoid errors when try to update the CR
		mss, err := r.fetchMssInstance(reqLogger, request)
		if err != nil {
			return nil, err
		}

		// Set the data
		mss.Status.ProxyServiceName = serviceStatus.Name
		mss.Status.ProxyServiceStatus = serviceStatus.Status

		// Update the CR
		err = r.client.Status().Update(context.TODO(), mss)
		if err != nil {
			reqLogger.Error(err, "Failed to update Proxy Service Name and Status for the MobileSecurityService")
			return serviceStatus, err
		}
	}

	return serviceStatus, nil
}

//updateRouteStatus returns error when status regards the route resource could not be updated
func (r *ReconcileMobileSecurityService) updateRouteStatus(reqLogger logr.Logger, request reconcile.Request) (*routev1.Route, error) {
	reqLogger.Info("Updating Route Status for the MobileSecurityService")
	// Get the latest version of the CR
	mss, err := r.fetchMssInstance(reqLogger, request)
	if err != nil {
		return nil, err
	}

	//Get the route Object
	route, err := r.fetchRoute(reqLogger, mss)
	if err != nil {
		reqLogger.Error(err, "Failed to get Route for Status", "MobileSecurityService.Namespace", mss.Namespace, "MobileSecurityService.Name", mss.Name)
		return route, err
	}

	// Update the Route Status and Name
	if mss.Spec.RouteName != mss.Status.RouteName || !reflect.DeepEqual(route.Status, mss.Status.RouteStatus) {
		// Get the latest version of the CR in order to try to avoid errors when try to update the CR
		mss, err := r.fetchMssInstance(reqLogger, request)
		if err != nil {
			return nil, err
		}

		// Set the data
		mss.Status.RouteName = mss.Spec.RouteName
		mss.Status.RouteStatus = route.Status

		// Update the CR
		err = r.client.Status().Update(context.TODO(), mss)
		if err != nil {
			reqLogger.Error(err, "Failed to update Route Name and Status for the MobileSecurityService")
			return route, err
		}
	}
	return route, nil
}
