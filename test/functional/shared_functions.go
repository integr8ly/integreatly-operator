package functional

import (
	"context"
	"errors"
	"fmt"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// return resource identifier annotation from cr
func GetCROAnnotation(instance metav1.Object) (string, error) {
	annotations := instance.GetAnnotations()
	if annotations == nil {
		return "", errors.New(fmt.Sprintf("annotations for %s can not be nil", instance.GetName()))
	}

	for k, v := range annotations {
		if "resourceIdentifier" == k {
			return v, nil
		}
	}
	return "", errors.New(fmt.Sprintf("no resource identifier found for resource %s", instance.GetName()))
}

// GetClusterID retrieves cluster id from cluster infrastructure
func GetClusterID(ctx context.Context, client dynclient.Client) (string, error) {
	infra := &configv1.Infrastructure{}
	if err := client.Get(ctx, types.NamespacedName{Name: "cluster"}, infra); err != nil {
		return "", fmt.Errorf("failed to retreive cluster infrastructure : %w", err)
	}
	return infra.Status.InfrastructureName, nil
}
