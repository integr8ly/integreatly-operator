package resources

import (
	croType "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildGenericMetricLabels returns generic labels to be added to every metric
func BuildGenericMetricLabels(r v1.ObjectMeta, clusterID, cacheName, providerName string) map[string]string {
	return map[string]string{
		"clusterID":   clusterID,
		"resourceID":  r.Name,
		"namespace":   r.Namespace,
		"instanceID":  cacheName,
		"productName": r.Labels["productName"],
		"strategy":    providerName,
	}
}

// BuildInfoMetricLabels adds extra information to labels around resource
func BuildInfoMetricLabels(r v1.ObjectMeta, status string, clusterID, cacheName, providerName string) map[string]string {
	labels := BuildGenericMetricLabels(r, clusterID, cacheName, providerName)
	if status != "" {
		labels["status"] = status
		return labels
	}
	labels["status"] = "nil"
	return labels
}

func BuildStatusMetricsLabels(r v1.ObjectMeta, clusterID, cacheName, providerName string, phase croType.StatusPhase) map[string]string {
	labels := BuildGenericMetricLabels(r, clusterID, cacheName, providerName)
	labels["statusPhase"] = string(phase)
	return labels
}
