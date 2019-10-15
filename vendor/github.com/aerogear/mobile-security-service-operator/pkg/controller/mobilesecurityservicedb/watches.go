package mobilesecurityservicedb

import (
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

//Watch for changes to secondary resources and create the owner MobileSecurityServiceDB

func watchPersistenceVolumeClaim(c controller.Controller) error {
	err := c.Watch(&source.Kind{Type: &corev1.PersistentVolumeClaim{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &mobilesecurityservicev1alpha1.MobileSecurityServiceDB{},
	})
	return err
}

//Watch Service objects created in the project/namespace
func watchService(c controller.Controller) error {
	err := c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &mobilesecurityservicev1alpha1.MobileSecurityServiceDB{},
	})
	return err
}

//Watch for changes to secondary resources and create the owner MobileSecurityServiceDB
//Watch Deployment objects created in the project/namespace
func watchDeployment(c controller.Controller) error {
	err := c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &mobilesecurityservicev1alpha1.MobileSecurityServiceDB{},
	})
	return err
}
