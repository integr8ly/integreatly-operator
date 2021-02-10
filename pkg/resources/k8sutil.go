package resources

import (
	"fmt"
	"os"
)

const (
	serviceAccountDir = "/var/run/secrets/kubernetes.io/serviceaccount"
)

// GetWatchNamespace returns the Namespace the operator should be watching for changes
func GetWatchNamespace() (string, error) {
	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which specifies the Namespace to watch.
	// An empty value means the operator is running with cluster scope.
	var watchNamespaceEnvVar = "WATCH_NAMESPACE"

	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}
	return ns, nil
}

// IsRunLocally checks if the operator is run locally
func IsRunLocally() bool {
	return !IsRunInCluster()
}

// IsRunInCluster checks if the operator is run in cluster
func IsRunInCluster() bool {
	_, err := os.Stat(serviceAccountDir)
	if err == nil {
		return true
	}

	return !os.IsNotExist(err)
}
