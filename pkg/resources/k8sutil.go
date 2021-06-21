package resources

import (
	"github.com/integr8ly/integreatly-operator/pkg/resources/k8s"
)

// GetWatchNamespace returns the Namespace the operator should be watching for changes
//
// Deprecated: Use pkg/resources/k8s.GetWatchNamespace() instead
func GetWatchNamespace() (string, error) {
	return k8s.GetWatchNamespace()
}

// IsRunLocally checks if the operator is run locally
//
// Deprecated: Use pkg/resources/k8s.IsRunLocally() instead
func IsRunLocally() bool {
	return k8s.IsRunLocally()
}

// IsRunInCluster checks if the operator is run in cluster
//
// Deprecated: Use pkg/resources/k8s.IsRunInCluster() instead
func IsRunInCluster() bool {
	return k8s.IsRunInCluster()
}
