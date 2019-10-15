package mobilesecurityserviceapp

import (
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Watch for changes to secondary resources and create the owner MobileSecurityService
// Watch ConfigMap objects created in the project/namespace
func watchConfigMap(c controller.Controller) error {
	err := c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &mobilesecurityservicev1alpha1.MobileSecurityServiceApp{},
	})
	return err
}
