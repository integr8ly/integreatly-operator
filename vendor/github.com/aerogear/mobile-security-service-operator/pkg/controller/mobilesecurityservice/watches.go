package mobilesecurityservice

import (
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

//Watch for changes to secondary resources and create the owner MobileSecurityService
//Watch ConfigMap resources created in the project/namespace
func watchConfigMap(c controller.Controller) error {
	err := c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &mobilesecurityservicev1alpha1.MobileSecurityService{},
	})
	return err
}

//Watch for changes to secondary resources and create the owner MobileSecurityService
//Watch Service resources created in the project/namespace
func watchService(c controller.Controller) error {
	err := c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &mobilesecurityservicev1alpha1.MobileSecurityService{},
	})
	return err
}

//Watch for changes to secondary resources and create the owner MobileSecurityService
//Watch Deployment resources created in the project/namespace
func watchDeployment(c controller.Controller) error {
	err := c.Watch(&source.Kind{Type: &v1beta1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &mobilesecurityservicev1alpha1.MobileSecurityService{},
	})
	return err
}

//Watch for changes to secondary resources and requeue the owner MobileSecurityService
//Watch Ingress resources created in the project/namespace
func watchIngress(c controller.Controller) error {
	err := c.Watch(&source.Kind{Type: &v1beta1.Ingress{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &mobilesecurityservicev1alpha1.MobileSecurityService{},
	})
	return err
}