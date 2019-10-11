package utils

import (
	corev1 "k8s.io/api/core/v1"
)

const APP_URL =  "mobile-security-service-app"

// getPodNames returns the pod names of the array of pods passed in
func GetPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

func GetAppIngressURL(protocol, host, hostSufix string) string {
	return protocol +"://" + APP_URL + "." + host + hostSufix
}

func GetAppIngress(host, hostSufix string) string {
	return APP_URL + "." + host + hostSufix
}

func GetRestAPIForApps(protocol, host, hostSufix string) string {
	return GetAppIngressURL(protocol, host, hostSufix) + "/api/apps"
}