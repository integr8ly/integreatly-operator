package mobilesecurityservicedb

import (
	"github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/aerogear/mobile-security-service-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Centralized mock objects for use in tests
var (
	dbInstance = v1alpha1.MobileSecurityServiceDB{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mobile-security-service-db",
			Namespace: utils.OperatorNamespaceForLocalEnv,
		},
		Spec: v1alpha1.MobileSecurityServiceDBSpec{
			Image:                  "centos/postgresql-96-centos7",
			Size:                   1,
			ContainerName:          "database",
			DatabaseNameParam:      "POSTGRESQL_DATABASE",
			DatabasePasswordParam:  "POSTGRESQL_PASSWORD",
			DatabaseUserParam:      "POSTGRESQL_USER",
			DatabasePort:           5432,
			DatabaseMemoryLimit:    "512Mi",
			DatabaseMemoryRequest:  "512Mi",
			DatabaseStorageRequest: "1Gi",
			DatabaseName:           "mobile_security_service",
			DatabasePassword:       "postgres",
			DatabaseUser:           "postgresql",
		},
	}

	dbInstanceNonDefaultNamespace = v1alpha1.MobileSecurityServiceDB{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mobile-security-service-db",
			Namespace: "mobile-security-service-namespace",
		},
		Spec: v1alpha1.MobileSecurityServiceDBSpec{
			Image:                  "centos/postgresql-96-centos7",
			Size:                   1,
			ContainerName:          "database",
			DatabaseNameParam:      "POSTGRESQL_DATABASE",
			DatabasePasswordParam:  "POSTGRESQL_PASSWORD",
			DatabaseUserParam:      "POSTGRESQL_USER",
			DatabasePort:           5432,
			DatabaseMemoryLimit:    "512Mi",
			DatabaseMemoryRequest:  "512Mi",
			DatabaseStorageRequest: "1Gi",
			DatabaseName:           "mobile_security_service",
			DatabasePassword:       "postgres",
			DatabaseUser:           "postgresql",
		},
	}

	dbInstanceWithoutSpec = v1alpha1.MobileSecurityServiceDB{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mobile-security-service-db",
			Namespace: utils.OperatorNamespaceForLocalEnv,
		},
	}

	serviceInstance = v1alpha1.MobileSecurityService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.MobileSecurityServiceCRName,
			Namespace: utils.OperatorNamespaceForLocalEnv,
		},
		Spec: mobilesecurityservicev1alpha1.MobileSecurityServiceSpec{
			Size:            1,
			MemoryLimit:     "512Mi",
			MemoryRequest:   "512Mi",
			ClusterProtocol: "http",
			ConfigMapName:   "mss-config",
			RouteName:       "mss-route",
		},
	}

	configMap = corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceInstance.Spec.ConfigMapName,
			Namespace: serviceInstance.Namespace,
			Labels:    map[string]string{"app": "mobilesecurityservice", "mobilesecurityservice_cr": serviceInstance.Name},
		},
		Data: map[string]string{
			"PGHOST":                           serviceInstance.Spec.DatabaseHost,
			"LOG_LEVEL":                        serviceInstance.Spec.LogLevel,
			"LOG_FORMAT":                       serviceInstance.Spec.LogFormat,
			"ACCESS_CONTROL_ALLOW_ORIGIN":      serviceInstance.Spec.AccessControlAllowOrigin,
			"ACCESS_CONTROL_ALLOW_CREDENTIALS": serviceInstance.Spec.AccessControlAllowCredentials,
			"PGDATABASE":                       serviceInstance.Spec.DatabaseName,
			"PGPASSWORD":                       serviceInstance.Spec.DatabasePassword,
			"PGUSER":                           serviceInstance.Spec.DatabaseUser,
		},
	}
)
