package mobilesecurityservicedb

import (
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//Returns the Deployment object for the Mobile Security Service Database
func (r *ReconcileMobileSecurityServiceDB) buildPVCForDB(m *mobilesecurityservicev1alpha1.MobileSecurityServiceDB) *corev1.PersistentVolumeClaim {
	ls := getDBLabels(m.Name)
	pv := &corev1.PersistentVolumeClaim{
		TypeMeta: v1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
			Labels:    ls,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(m.Spec.DatabaseStorageRequest),
				},
			},
		},
	}
	// Set MobileSecurityServiceDB instance as the owner and controller
	controllerutil.SetControllerReference(m, pv, r.scheme)
	return pv
}