package mobilesecurityservice

import (
	"fmt"
	"math/rand"

	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

const (
	letterBytes    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	oauthProxyPort = 4180
)

//getAppLabels returns an string map with the labels which wil be associated to the kubernetes/ocp resource which will be created and managed by this operator
func getAppLabels(name string) map[string]string {
	return map[string]string{"app": "mobilesecurityservice", "mobilesecurityservice_cr": name, "name": "mobilesecurityservice"}
}

//buildAppEnvVars is a helper to build the env vars which will be configured in the deployment of the Mobile Security Service Project
func buildAppEnvVars(mss *mobilesecurityservicev1alpha1.MobileSecurityService) *[]corev1.EnvVar {
	res := []corev1.EnvVar{}
	for key := range getAppEnvVarsMap(mss) {
		env := corev1.EnvVar{
			Name: key,
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: mss.Spec.ConfigMapName,
					},
					Key: key,
				},
			},
		}
		res = append(res, env)
	}
	return &res
}

//getAppEnvVarsMap is a helper to get a map[string]string with the key and values required/used to setup the Mobile Security Service Project
func getAppEnvVarsMap(mss *mobilesecurityservicev1alpha1.MobileSecurityService) map[string]string {
	return map[string]string{
		"PGHOST":                           mss.Spec.DatabaseHost,
		"LOG_LEVEL":                        mss.Spec.LogLevel,
		"LOG_FORMAT":                       mss.Spec.LogFormat,
		"ACCESS_CONTROL_ALLOW_ORIGIN":      mss.Spec.AccessControlAllowOrigin,
		"ACCESS_CONTROL_ALLOW_CREDENTIALS": mss.Spec.AccessControlAllowCredentials,
		"PGDATABASE":                       mss.Spec.DatabaseName,
		"PGPASSWORD":                       mss.Spec.DatabasePassword,
		"PGUSER":                           mss.Spec.DatabaseUser,
		"PORT":                             string(mss.Spec.Port),
	}
}

// getOAuthArgsMap is a helper to get the []string with values required/used to set OAuth for the Mobile Security Service Project
func getOAuthArgsMap(mss *mobilesecurityservicev1alpha1.MobileSecurityService) []string {
	proxyPort := fmt.Sprint(oauthProxyPort)
	port := fmt.Sprint(mss.Spec.Port)
	return []string{
		fmt.Sprintf("--http-address=0.0.0.0:%s", proxyPort),
		"--https-address=",
		"--provider=openshift",
		fmt.Sprintf("--openshift-service-account=%s", mss.Name),
		fmt.Sprintf("--upstream=http://localhost:%s", port),
		"--cookie-secure=true",
		fmt.Sprintf("--cookie-secret=%s", RandStringBytes(16)),
		"--cookie-httponly=false",
		"--bypass-auth-for=/api/init",
		"--bypass-auth-for=/api/healthz",
		"--bypass-auth-for=/api/ping",
		"--pass-user-headers=true",
	}
}

// RandStringBytes will return a string of n random bytes
func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
