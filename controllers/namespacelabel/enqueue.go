package controllers

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

// EnqueueNamespaceFromObject is an EventHandler that enqueues requests with
// the name taken from the source namespace. Example:
// An event cause by an object called Foo in the namespace Bar will enqueue
// the following request:
// { Name: "Bar", Namespace: "" }
type EnqueueNamespaceFromObject struct{}

var _ handler.EventHandler = &EnqueueNamespaceFromObject{}

// Create implements EventHandler
func (e *EnqueueNamespaceFromObject) Create(ctx context.Context, evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		return
	}

	addNamespaceRequest(evt.Object, q)
}

// Update implements EventHandler
func (e *EnqueueNamespaceFromObject) Update(ctx context.Context, evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if evt.ObjectOld != nil {
		addNamespaceRequest(evt.ObjectOld, q)
	}

	if evt.ObjectNew != nil {
		addNamespaceRequest(evt.ObjectNew, q)
	}
}

// Delete implements EventHandler
func (e *EnqueueNamespaceFromObject) Delete(ctx context.Context, evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		return
	}

	addNamespaceRequest(evt.Object, q)
}

// Generic implements EventHandler
func (e *EnqueueNamespaceFromObject) Generic(ctx context.Context, evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		return
	}

	addNamespaceRequest(evt.Object, q)
}

func addNamespaceRequest(meta metav1.Object, q workqueue.RateLimitingInterface) {
	q.Add(ctrl.Request{NamespacedName: types.NamespacedName{
		Name: meta.GetNamespace(),
	}})
}
