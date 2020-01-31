package resources

import (
	"context"
	"reflect"

	croType "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"

	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"

	errorUtil "github.com/pkg/errors"
)

//UpdatePhase Updates the custom resource with the current phase
func UpdatePhase(ctx context.Context, client client.Client, inst runtime.Object, phase croType.StatusPhase, msg croType.StatusMessage) error {
	if msg == croType.StatusEmpty {
		return nil
	}
	rts := &croType.ResourceTypeStatus{}
	if err := runtime.Field(reflect.ValueOf(inst).Elem(), "Status", rts); err != nil {
		return errorUtil.Wrap(err, "failed to retrieve status block from object")
	}
	rts.Message = msg
	rts.Phase = phase
	if err := runtime.SetField(*rts, reflect.ValueOf(inst).Elem(), "Status"); err != nil {
		return errorUtil.Wrap(err, "failed to set status block of object")
	}
	if err := client.Status().Update(ctx, inst); err != nil {
		return errorUtil.Wrap(err, "failed to update resource status phase and message")
	}
	return nil
}

//UpdateSnapshotPhase Updates the snapshot custom resource with the current phase
func UpdateSnapshotPhase(ctx context.Context, client client.Client, inst runtime.Object, phase croType.StatusPhase, msg croType.StatusMessage) error {
	if msg == croType.StatusEmpty {
		return nil
	}
	rts := &croType.ResourceTypeSnapshotStatus{}
	if err := runtime.Field(reflect.ValueOf(inst).Elem(), "Status", rts); err != nil {
		return errorUtil.Wrap(err, "failed to retrieve status block from object")
	}
	rts.Message = msg
	rts.Phase = phase
	if err := runtime.SetField(*rts, reflect.ValueOf(inst).Elem(), "Status"); err != nil {
		return errorUtil.Wrap(err, "failed to set status block of object")
	}
	if err := client.Status().Update(ctx, inst); err != nil {
		return errorUtil.Wrap(err, "failed to update resource status phase and message")
	}
	return nil
}
