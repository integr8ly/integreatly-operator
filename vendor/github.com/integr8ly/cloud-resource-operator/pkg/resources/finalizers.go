package resources

import (
	"context"
	"reflect"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	errorUtil "github.com/pkg/errors"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func HasFinalizer(om *controllerruntime.ObjectMeta, finalizer string) bool {
	return Contains(om.GetFinalizers(), finalizer)
}

func addFinalizer(om *controllerruntime.ObjectMeta, finalizer string) {
	if !HasFinalizer(om, finalizer) {
		om.SetFinalizers([]string{finalizer})
	}
}

func RemoveFinalizer(om *controllerruntime.ObjectMeta, finalizer string) {
	om.SetFinalizers(remove(om.GetFinalizers(), finalizer))
}

//Contains checks if a string exists in a slice of strings
func Contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func CreateFinalizer(ctx context.Context, c client.Client, inst runtime.Object, df string) error {
	dt := &v1.ObjectMeta{}
	if err := runtime.Field(reflect.ValueOf(inst).Elem(), "ObjectMeta", dt); err != nil {
		return errorUtil.Wrap(err, "failed to retrieve timestamp")
	}
	if dt.DeletionTimestamp == nil {
		addFinalizer(dt, df)
		if err := runtime.SetField(*dt, reflect.ValueOf(inst).Elem(), "ObjectMeta"); err != nil {
			return errorUtil.Wrap(err, "failed to set object meta back to instance")
		}
		if err := c.Update(ctx, inst); err != nil {
			return errorUtil.Wrapf(err, "failed to add finalizer to instance")
		}
	}
	return nil
}
