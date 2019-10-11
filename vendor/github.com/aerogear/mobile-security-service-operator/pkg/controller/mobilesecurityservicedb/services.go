package mobilesecurityservicedb

import (
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"k8s.io/apimachinery/pkg/util/intstr"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//Returns the Service object for the Mobile Security Service Database
func (r *ReconcileMobileSecurityServiceDB) buildDBService(m *mobilesecurityservicev1alpha1.MobileSecurityServiceDB) *corev1.Service {
	ls := getDBLabels(m.Name)
	ser := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
			Labels:    ls,
		},
		Spec: corev1.ServiceSpec{
			Selector: ls,
			Type:     corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name: m.Name,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: m.Spec.DatabasePort,
					},
					Port:     m.Spec.DatabasePort,
					Protocol: "TCP",
				},
			},
		},
	}
	// Set MobileSecurityServiceDB instance as the owner and controller
	controllerutil.SetControllerReference(m, ser, r.scheme)
	return ser
}
