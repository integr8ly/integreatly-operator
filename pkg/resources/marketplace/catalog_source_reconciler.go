package marketplace

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type CatalogSourceReconciler interface {
	Reconcile(ctx context.Context) (reconcile.Result, error)
	CatalogSourceName() string
	CatalogSourceNamespace() string
}
