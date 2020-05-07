package common

import (
	goctx "context"
	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

var (
	namespaces_to_verify = []string{
		"redhat-rhmi-cloud-resources-operator",
		"redhat-rhmi-middleware-monitoring-federate",
		"redhat-rhmi-operator",
	}
)

func TestNamespaceNamingConventions(t *testing.T, ctx *TestingContext) {

	for _, namespaceName := range namespaces_to_verify {
		namespace := &corev1.Namespace{}
		err := ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: namespaceName}, namespace)
		if err != nil {
			t.Fatalf("Error getting namespace: %v from cluster: %w", namespaceName, err)
		}
	}
}
