package controllers

import (
	"context"
	"errors"
	"fmt"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	IntegreatlyOwnerNamespace = "integreatly-namespace"
	IntegreatlyOwnerName      = "integreatly-name"
)

type EnqueueIntegreatlyOwner struct {
	log l.Logger
}

func (e *EnqueueIntegreatlyOwner) Create(ctx context.Context, evt event.CreateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	if req, err := e.getIntegreatlyOwner(evt.Object); err == nil {
		q.Add(req)
	}
}

func (e *EnqueueIntegreatlyOwner) Update(ctx context.Context, evt event.UpdateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	if req, err := e.getIntegreatlyOwner(evt.ObjectNew); err == nil {
		q.Add(req)
	}
}

func (e *EnqueueIntegreatlyOwner) Delete(ctx context.Context, evt event.DeleteEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	if req, err := e.getIntegreatlyOwner(evt.Object); err == nil {
		q.Add(req)
	}
}

func (e *EnqueueIntegreatlyOwner) Generic(ctx context.Context, evt event.GenericEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	if req, err := e.getIntegreatlyOwner(evt.Object); err == nil {
		q.Add(req)
	}
}

func (e *EnqueueIntegreatlyOwner) getIntegreatlyOwner(object client.Object) (reconcile.Request, error) {
	typeObj, err := meta.TypeAccessor(object)
	if err != nil {
		return reconcile.Request{}, err
	}

	ant := object.GetAnnotations()
	if ns, ok := ant[IntegreatlyOwnerNamespace]; ok {
		if name, ok := ant[IntegreatlyOwnerName]; ok {
			log.Info(fmt.Sprintf("%s %s/%s > got integreatly owner %s/%s", typeObj.GetKind(), object.GetNamespace(), object.GetName(), ns, name))
			return reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: ns,
					Name:      name,
				},
			}, nil
		}
	}
	return reconcile.Request{}, errors.New("object does not have an integreatly owner")
}
