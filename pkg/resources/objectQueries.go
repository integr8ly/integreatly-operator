package resources

import (
	"context"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func Exists(ctx context.Context, serverClient k8sclient.Client, obj runtime.Object) (bool, error) {
	metaobj, err := meta.Accessor(obj)
	if err != nil {
		return false, err
	}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: metaobj.GetName(), Namespace: metaobj.GetNamespace()}, obj)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// UpdateIfExists uses serverClient to retrieve obj by it's object key. If
// obj is not found, it returns InProgress. If obj is found, it applies fn
// and updates the object
func UpdateIfExists(ctx context.Context, serverClient k8sclient.Client, fn controllerutil.MutateFn, obj runtime.Object) (integreatlyv1alpha1.StatusPhase, error) {
	objKey, err := k8sclient.ObjectKeyFromObject(obj)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	if err := serverClient.Get(ctx, objKey, obj); err != nil {
		if k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseInProgress, nil
		}

		return integreatlyv1alpha1.PhaseFailed, err
	}

	if err := fn(); err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	if err := serverClient.Update(ctx, obj); err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
