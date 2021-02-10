package controllers

import (
	"context"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"

	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// installationManager implements the Mapper interface so that it can be passed to
// handler.EnqueueRequestsFromMapFunc{}.
//
// The purpose of it is to be able to enqueue reconcile requests for ALL Installation CRs in a
// namespace, when a watch picks up a relevant event.
type installationMapper struct {
	context context.Context
	client  k8sclient.Client
}

func (m installationMapper) Map(mo handler.MapObject) []reconcile.Request {
	installationList := &integreatlyv1alpha1.RHMIList{}
	err := m.client.List(m.context, installationList)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(installationList.Items))
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
// the object doesn't fullfil the predicate in the Predicate field
type objectPredicate struct {
	Predicate func(mo handler.MapObject) bool
}

func newObjectPredicate(predicate func(mo handler.MapObject) bool) *objectPredicate {
	return &objectPredicate{
		Predicate: predicate,
	}
}

var _ predicate.Predicate = &objectPredicate{}

func (p *objectPredicate) Create(e event.CreateEvent) bool {
	return p.Predicate(handler.MapObject{Meta: e.Meta, Object: e.Object})
}

func (p *objectPredicate) Delete(e event.DeleteEvent) bool {
	return p.Predicate(handler.MapObject{Meta: e.Meta, Object: e.Object})
}

func (p *objectPredicate) Update(e event.UpdateEvent) bool {
	return p.Predicate(handler.MapObject{Meta: e.MetaNew, Object: e.ObjectNew})
}

func (p *objectPredicate) Generic(e event.GenericEvent) bool {
	return p.Predicate(handler.MapObject{Meta: e.Meta, Object: e.Object})
}

// isName creates a predicate that returns true if the object name matches name
func isName(name string) func(handler.MapObject) bool {
	return func(mo handler.MapObject) bool {
		return mo.Meta.GetName() == name
	}
}
