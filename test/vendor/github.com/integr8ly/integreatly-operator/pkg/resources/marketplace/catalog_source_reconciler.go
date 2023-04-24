package marketplace

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type CatalogSourceReconciler interface {
	Reconcile(ctx context.Context, subName string) (reconcile.Result, error)
	CatalogSourceName() string
	CatalogSourceNamespace() string
}
