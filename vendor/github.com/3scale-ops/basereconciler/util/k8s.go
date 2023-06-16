package util

import (
	"reflect"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ObjectKey(o client.Object) types.NamespacedName {
	return types.NamespacedName{Name: o.GetName(), Namespace: o.GetNamespace()}
}

// this is an ugly function to retrieve the list of Items from a
// client.ObjectList because the interface doesn't have a GetItems
// method
func GetItems(list client.ObjectList) []client.Object {
	items := []client.Object{}
	values := reflect.ValueOf(list).Elem().FieldByName("Items")
	for i := 0; i < values.Len(); i++ {
		item := values.Index(i)
		if item.Kind() == reflect.Pointer {
			items = append(items, item.Interface().(client.Object))
		} else {
			items = append(items, item.Addr().Interface().(client.Object))
		}
	}

	return items
}

func IsBeingDeleted(o client.Object) bool {
	return !o.GetDeletionTimestamp().IsZero()
}
