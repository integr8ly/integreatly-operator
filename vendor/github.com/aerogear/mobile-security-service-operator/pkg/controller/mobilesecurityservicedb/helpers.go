package mobilesecurityservicedb

import (
	"context"
	"github.com/aerogear/mobile-security-service-operator/pkg/utils"
	"time"

	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func getDBLabels(name string) map[string]string {
	return map[string]string{"app": "mobilesecurityservice", "mobilesecurityservicedb_cr": name, "name": "mobilesecurityservicedb"}
}

func (r *ReconcileMobileSecurityServiceDB) getDatabaseNameEnvVar(db *mobilesecurityservicev1alpha1.MobileSecurityServiceDB, serviceConfigMapName string) corev1.EnvVar {
	if len(serviceConfigMapName) > 0 {
		return corev1.EnvVar{
			Name: db.Spec.DatabaseNameParam,
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: serviceConfigMapName,
					},
					Key: "PGDATABASE",
				},
			},
		}
	}

	return corev1.EnvVar{
		Name:  db.Spec.DatabaseNameParam,
		Value: db.Spec.DatabaseName,
	}
}

// getMssConfigMapName will return the name of the configMap created by MSS with the env var values which should be shared by Service and Database
func (r *ReconcileMobileSecurityServiceDB) getMssConfigMapName(db *mobilesecurityservicev1alpha1.MobileSecurityServiceDB) string {

	serviceConfigMapName := r.fetchMssConfigMap(db)
	if len(serviceConfigMapName) < 1 {
		// Wait for 30 seconds to check if will be created
		time.Sleep(30 * time.Second)
		// Try again
		serviceConfigMapName = r.fetchMssConfigMap(db)
	}
	return serviceConfigMapName
}

// fetchMssConfigMap returns the resource created/managed by MSS instance with the values which will be used by th env vars.
func (r *ReconcileMobileSecurityServiceDB) fetchMssConfigMap(db *mobilesecurityservicev1alpha1.MobileSecurityServiceDB) string {
	// It will fetch the service
	// db for the DB type be able to get the configMap config created by it, however,
	// if the Instance cannot be found and/or its configMap was not created than the default values specified in its CR will be used
	mss := &mobilesecurityservicev1alpha1.MobileSecurityService{}
	r.client.Get(context.TODO(), types.NamespacedName{Name: utils.MobileSecurityServiceCRName, Namespace: db.Namespace}, mss)

	//if has not service db return false
	if len(mss.Spec.ConfigMapName) > 1 {
		//Looking for the configMap created by the service db
		configMap := &corev1.ConfigMap{}
		err := r.client.Get(context.TODO(), types.NamespacedName{Name: mss.Spec.ConfigMapName, Namespace: db.Namespace}, configMap)
		if err == nil {
			return configMap.Name
		}

	}
	return ""
}

func (r *ReconcileMobileSecurityServiceDB) getDatabaseUserEnvVar(db *mobilesecurityservicev1alpha1.MobileSecurityServiceDB, serviceConfigMapName string) corev1.EnvVar {
	if len(serviceConfigMapName) > 0 {
		return corev1.EnvVar{
			Name: db.Spec.DatabaseUserParam,
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: serviceConfigMapName,
					},
					Key: "PGUSER",
				},
			},
		}
	}

	return corev1.EnvVar{
		Name:  db.Spec.DatabaseUserParam,
		Value: db.Spec.DatabaseUser,
	}
}

func (r *ReconcileMobileSecurityServiceDB) getDatabasePasswordEnvVar(db *mobilesecurityservicev1alpha1.MobileSecurityServiceDB, serviceConfigMapName string) corev1.EnvVar {
	if len(serviceConfigMapName) > 0 {
		return corev1.EnvVar{
			Name: db.Spec.DatabasePasswordParam,
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: serviceConfigMapName,
					},
					Key: "PGPASSWORD",
				},
			},
		}
	}

	return corev1.EnvVar{
		Name:  db.Spec.DatabasePasswordParam,
		Value: db.Spec.DatabasePassword,
	}
}
