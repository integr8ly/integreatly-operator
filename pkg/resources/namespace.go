package resources

import (
	"context"
	"fmt"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate moq -out NamespaceReconciler_moq.go . NamespaceReconciler
type NamespaceReconciler interface {
	Reconcile(ctx context.Context, ns *v1.Namespace, owner ownerutil.Owner) (*v1.Namespace, error)
}

type SimpleNamespaceReconciler struct {
	client pkgclient.Client
}

func NewNamespaceReconciler(client pkgclient.Client) NamespaceReconciler {
	return &SimpleNamespaceReconciler{client: client}
}

func (nr *SimpleNamespaceReconciler) Reconcile(ctx context.Context, ns *v1.Namespace, owner ownerutil.Owner) (*v1.Namespace, error) {
	if ns.Name == "" {
		return ns, errors.New("cannot reconcile namespace, it has no name")
	}
	err := nr.client.Get(ctx, pkgclient.ObjectKey{Name: ns.Name}, ns)
	if err != nil {
		if !errors2.IsNotFound(err) {
			return nil, errors.Wrap(err, fmt.Sprintf("could not retrieve namespace: %s", ns.Name))
		}
		ownerutil.EnsureOwner(ns, owner)
		if ns.Labels == nil {
			ns.Labels = map[string]string{}
		}
		ns.Labels["integreatly"] = "true"
		if err = nr.client.Create(ctx, ns); err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("could not create namespace: %s", ns.Name))
		}
		return ns, nil
	}
	// ns exists so check it is our namespace
	if !ownerutil.IsOwnedBy(ns, owner) && ns.Status.Phase != v1.NamespaceTerminating {
		return nil, errors.New("existing namespace found with name " + ns.Name + " but it is not owned by the integreatly installation")
	}

	ownerutil.EnsureOwner(ns, owner)
	if err := nr.client.Update(ctx, ns); err != nil {
		return ns, errors.Wrap(err, "failed to update the ns definition ")
	}
	return ns, nil
}
