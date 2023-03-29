package resources

import (
	croType "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	LabelClusterIDKey   = "clusterID"
	LabelResourceIDKey  = "resourceID"
	LabelNamespaceKey   = "namespace"
	LabelInstanceIDKey  = "instanceID"
	LabelProductNameKey = "productName"
	LabelStrategyKey    = "strategy"
	LabelStatusKey      = "status"
	LabelStatusPhaseKey = "statusPhase"
)

// BuildGenericMetricLabels returns generic labels to be added to every metric
func BuildGenericMetricLabels(objectMeta v1.ObjectMeta, clusterID, instanceID, providerName string) map[string]string {
	return map[string]string{
		LabelClusterIDKey:   clusterID,
		LabelResourceIDKey:  objectMeta.Name,
		LabelNamespaceKey:   objectMeta.Namespace,
		LabelInstanceIDKey:  instanceID,
		LabelProductNameKey: objectMeta.Labels["productName"],
		LabelStrategyKey:    providerName,
	}
}

// BuildInfoMetricLabels adds extra information to labels around resource
func BuildInfoMetricLabels(r v1.ObjectMeta, status string, clusterID, cacheName, providerName string) map[string]string {
	labels := BuildGenericMetricLabels(r, clusterID, cacheName, providerName)
	if status != "" {
		labels[LabelStatusKey] = status
		return labels
	}
	labels[LabelStatusKey] = "nil"
	return labels
}

func BuildStatusMetricsLabels(r v1.ObjectMeta, clusterID, cacheName, providerName string, phase croType.StatusPhase) map[string]string {
	labels := BuildGenericMetricLabels(r, clusterID, cacheName, providerName)
	labels[LabelStatusPhaseKey] = string(phase)
	return labels
}
