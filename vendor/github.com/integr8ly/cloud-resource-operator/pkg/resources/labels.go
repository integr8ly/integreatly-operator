package resources

import (
	croType "github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
func BuildGenericMetricLabels(objectMeta metav1.ObjectMeta, clusterID, instanceID, providerName string) map[string]string {
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
func BuildInfoMetricLabels(r metav1.ObjectMeta, status string, clusterID, cacheName, providerName string) map[string]string {
	labels := BuildGenericMetricLabels(r, clusterID, cacheName, providerName)
	if status != "" {
		labels[LabelStatusKey] = status
		return labels
	}
	labels[LabelStatusKey] = "nil"
	return labels
}

func BuildStatusMetricsLabels(r metav1.ObjectMeta, clusterID, cacheName, providerName string, phase croType.StatusPhase) map[string]string {
	labels := BuildGenericMetricLabels(r, clusterID, cacheName, providerName)
	labels[LabelStatusPhaseKey] = string(phase)
	return labels
}

// HasLabel returns true if the label is already set
func HasLabel(object metav1.Object, key string) bool {
	labels := object.GetLabels()
	_, ok := labels[key]
	return ok
}

// HasLabelWithValue returns true if the label with corresponding value matches
func HasLabelWithValue(object metav1.Object, key, value string) bool {
	labels := object.GetLabels()
	return labels[key] == value
}

// GetLabel retrieves a label value
func GetLabel(object metav1.Object, key string) string {
	labels := object.GetLabels()
	return labels[key]
}

// AddLabel makes sure that the provided key/value are set as a label
func AddLabel(object metav1.Object, key, value string) {
	labels := object.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[key] = value
	object.SetLabels(labels)
}

// RemoveLabel makes sure that the provided label is removed
func RemoveLabel(object metav1.Object, key string) {
	labels := object.GetLabels()
	delete(labels, key)
	object.SetLabels(labels)
}
