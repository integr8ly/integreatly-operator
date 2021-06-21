package resources

import (
	"context"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/k8s"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Exists checks if obj exists on the cluster accessible by serverClient
//
// Deprecated: use pkg/resources/k8s.Exists instead
func Exists(ctx context.Context, serverClient k8sclient.Client, obj runtime.Object) (bool, error) {
	return k8s.Exists(ctx, serverClient, obj)
}

// UpdateIfExists uses serverClient to retrieve obj by it's object key. If
// obj is not found, it returns InProgress. If obj is found, it applies fn
// and updates the object
//
// Deprecated: use pkg/resources/k8s.UpdateIfExists instead
func UpdateIfExists(ctx context.Context, serverClient k8sclient.Client, fn controllerutil.MutateFn, obj runtime.Object) (integreatlyv1alpha1.StatusPhase, error) {
	return k8s.UpdateIfExists(ctx, serverClient, fn, obj)
}
