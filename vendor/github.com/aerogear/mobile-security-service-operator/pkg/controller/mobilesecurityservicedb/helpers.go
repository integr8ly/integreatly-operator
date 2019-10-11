package mobilesecurityservicedb

import (
	"context"
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func getDBLabels(name string) map[string]string {
	return map[string]string{"app": "mobilesecurityservicedb", "mobilesecurityservicedb_cr": name, "name": "mobilesecurityservicedb"}
}

func (r *ReconcileMobileSecurityServiceDB) getDatabaseNameEnvVar(m *mobilesecurityservicev1alpha1.MobileSecurityServiceDB) corev1.EnvVar {
	if r.hasAppConfigMap(m) {
		return corev1.EnvVar{
			Name: m.Spec.DatabaseNameParam,
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: getConfigMapName(m),
					},
					Key: "PGDATABASE",
				},
			},
		}
	}

	return corev1.EnvVar{
		Name:  m.Spec.DatabaseNameParam,
		Value: m.Spec.DatabaseName,
	}
}

//Check if has App Config Map created
func (r *ReconcileMobileSecurityServiceDB) hasAppConfigMap(m *mobilesecurityservicev1alpha1.MobileSecurityServiceDB) bool {
	configMap := &corev1.ConfigMap{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: getConfigMapName(m), Namespace: m.Namespace}, configMap)
	if err != nil {
		return false
	}
	return true
}

func (r *ReconcileMobileSecurityServiceDB) getDatabaseUserEnvVar(m *mobilesecurityservicev1alpha1.MobileSecurityServiceDB) corev1.EnvVar {
	if r.hasAppConfigMap(m) {
		return corev1.EnvVar{
			Name: m.Spec.DatabaseUserParam,
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: getConfigMapName(m),
					},
					Key: "PGUSER",
				},
			},
		}
	}

	return corev1.EnvVar{
		Name:  m.Spec.DatabaseUserParam,
		Value: m.Spec.DatabaseUser,
	}
}

func (r *ReconcileMobileSecurityServiceDB) getDatabasePasswordEnvVar(m *mobilesecurityservicev1alpha1.MobileSecurityServiceDB) corev1.EnvVar {
	if r.hasAppConfigMap(m) {
		return corev1.EnvVar{
			Name: m.Spec.DatabasePasswordParam,
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: getConfigMapName(m),
					},
					Key: "PGPASSWORD",
				},
			},
		}
	}

	return corev1.EnvVar{
		Name:  m.Spec.DatabasePasswordParam,
		Value: m.Spec.DatabasePassword,
	}
}

//getConfigMapName returns an string name with the name of the configMap
func getConfigMapName(m *mobilesecurityservicev1alpha1.MobileSecurityServiceDB) string{
	if len(m.Spec.ConfigMapName) > 0 {
		return m.Spec.ConfigMapName
	}
	return m.Name
}