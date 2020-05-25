package annotations

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Add makes sure that the provided key/value are set as an annotation
func Add(instance metav1.Object, key, value string) {
	annotations := instance.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[key] = value
	instance.SetAnnotations(annotations)
}

// Has returns true if the annotation is already set
func Has(instance metav1.Object, key string) bool {
	annotations := instance.GetAnnotations()
	if annotations == nil {
		return false
	}

	for k := range annotations {
		if key == k {
			return true
		}
	}

	return false
}
