package installation

import (
	"context"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
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
	installationList := &integreatlyv1alpha1.InstallationList{}
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
