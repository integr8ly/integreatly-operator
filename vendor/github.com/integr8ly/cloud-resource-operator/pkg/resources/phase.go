package resources

import (
	"context"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	errorUtil "github.com/pkg/errors"
)

func UpdatePhase(ctx context.Context, client client.Client, inst runtime.Object, phase v1alpha1.StatusPhase, msg v1alpha1.StatusMessage) error {
	if msg == v1alpha1.StatusEmpty {
		return nil
	}
	rts := &v1alpha1.ResourceTypeStatus{}
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
