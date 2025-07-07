package controllers

import (
	"context"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// installationMapper implements the Mapper interface so that it can be passed to
// handler.EnqueueRequestsFromMapFunc{}.
//
// The purpose of it is to be able to enqueue reconcile requests for ALL Installation CRs in a
// namespace, when a watch picks up a relevant event.
type installationMapper struct {
	context context.Context
	client  client.Client
}

// MapFunc defines the function signature expected by EnqueueRequestsFromMapFunc.
type MapFunc func(context.Context, client.Object) []reconcile.Request

func (m installationMapper) Map(ctx context.Context, obj client.Object) []reconcile.Request {
	installationList := &integreatlyv1alpha1.RHMIList{}
	err := m.client.List(ctx, installationList)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, 0, len(installationList.Items))
	for _, installation := range installationList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      installation.Name,
				Namespace: installation.Namespace,
			},
		})
	}

	return requests
}

// objectPredicate implements the predicate.Predicate interface, which is used
// to filter watch events. This implementation will filter out events where
// the object doesn't fulfill the predicate in the Predicate field
type objectPredicate struct {
	Predicate func(object client.Object) bool
}

func newObjectPredicate(predicate func(object client.Object) bool) *objectPredicate {
	return &objectPredicate{
		Predicate: predicate,
	}
}

var _ predicate.Predicate = &objectPredicate{}

func (p *objectPredicate) Create(e event.CreateEvent) bool {
	return p.Predicate(e.Object)
}

func (p *objectPredicate) Delete(e event.DeleteEvent) bool {
	return p.Predicate(e.Object)
}

func (p *objectPredicate) Update(e event.UpdateEvent) bool {
	return p.Predicate(e.ObjectNew)
}

func (p *objectPredicate) Generic(e event.GenericEvent) bool {
	return p.Predicate(e.Object)
}

func isName(name string) func(client.Object) bool {
	return func(object client.Object) bool {
		return object.GetName() == name
	}
}
