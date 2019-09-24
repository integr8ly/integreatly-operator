package util

import (
	"context"

	"github.com/go-logr/logr"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AddFinalizer will add a finalizer to the PushApplication CR so that
// we can delete from UPS appropriately
func AddFinalizer(client client.Client, reqLogger logr.Logger, o metav1.Object) error {
	// This is based on the example code at:
	// https://github.com/operator-framework/operator-sdk/blob/master/doc/user-guide.md#handle-cleanup-on-deletion

	if len(o.GetFinalizers()) > 0 || o.GetDeletionTimestamp() != nil {
		return nil
	}

	reqLogger.Info("Adding Finalizer to the PushApplication")
	o.SetFinalizers([]string{"finalizer.push.aerogear.org"})

	runtimeObject, ok := o.(runtime.Object)
	if !ok {
		reqLogger.Info("Can't determine the type of thing to add finalizer to")
		return nil
	}

	err := client.Update(context.TODO(), runtimeObject)
	if err != nil {
		reqLogger.Error(err, "Failed to update a CR with a finalizer")
		return err
	}

	return nil
}
