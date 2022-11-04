package marketplace

import (
	"context"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"

	"github.com/integr8ly/integreatly-operator/pkg/addon"
	k8sresources "github.com/integr8ly/integreatly-operator/pkg/resources/k8s"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	rhmiresources "github.com/integr8ly/integreatly-operator/pkg/resources/rhmi"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	coreosv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
)

// ImplicitCatalogSourceReconciler is a CatalogSourceReconciler implementation
// that does not create a new CatalogSource, but expects the operator to be
// included in the CatalogSource installed by the add-on
type ImplicitCatalogSourceReconciler struct {
	Client            k8sclient.Client
	Log               l.Logger
	selfCatalogSource *coreosv1alpha1.CatalogSource
}

var _ CatalogSourceReconciler = &ImplicitCatalogSourceReconciler{}

func NewImplicitCatalogSourceReconciler(log l.Logger, client k8sclient.Client) (*ImplicitCatalogSourceReconciler, error) {
	reconciler := &ImplicitCatalogSourceReconciler{
		Log:    log,
		Client: client,
	}

	return reconciler, nil
}

// Reconcile finds the CatalogSource that provides the current installation,
// returning an error if it fails to find it. Caches the found CatalogSource
// to be used by r.CatalogSourceName() and r.CatalogSourceNamespace()
func (r *ImplicitCatalogSourceReconciler) Reconcile(ctx context.Context, subName string) (reconcile.Result, error) {
	// Get the CatalogSource that installed the operator
	catalogSource, err := r.getSelfCatalogSource(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}
	// If the CatalogSource was not found, return an error
	if catalogSource == nil {
		return reconcile.Result{}, errors.New("catalog source not found for implicit product installation type")
	}

	r.Log.Infof("Found: ", l.Fields{"catalogSource": catalogSource.Name, "ns": catalogSource.Namespace})

	watchNS, err := k8sresources.GetWatchNamespace()
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "could not get watch namespace in ImplicitCatalogSourceReconciler")
	}
	namespaceSegments := strings.Split(watchNS, "-")
	namespacePrefix := strings.Join(namespaceSegments[0:2], "-") + "-"
	ns := namespacePrefix + "operator"

	// If the Catalog Source is in the rhoam operator ns but the subscription is for 3Scale then
	// we need to copy the CS to the 3Scale operator namespace in order for it to be discovered.
	if catalogSource.Namespace == ns && strings.Contains(subName, "3scale") {
		r.Log.Info("Copying 3scale Catalog source")

		cs := &coreosv1alpha1.CatalogSource{
			ObjectMeta: v1.ObjectMeta{
				Name:      catalogSource.Name,
				Namespace: namespacePrefix + "3scale-operator",
			},
		}

		_, err = controllerutil.CreateOrUpdate(ctx, r.Client, cs, func() error {
			cs.Spec = catalogSource.DeepCopy().Spec
			return nil
		})
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "could not createOrUpdate catalogSource in ImplicitCatalogSourceReconciler")
		}

		r.Log.Infof("Created 3Scale catalog source ", l.Fields{"image": cs.Spec.Image})
		r.selfCatalogSource = cs
	}

	return reconcile.Result{}, nil
}

func (r *ImplicitCatalogSourceReconciler) CatalogSourceName() string {
	return r.selfCatalogSource.Name
}

func (r *ImplicitCatalogSourceReconciler) CatalogSourceNamespace() string {
	return r.selfCatalogSource.Namespace
}

func (r *ImplicitCatalogSourceReconciler) getSelfCatalogSource(ctx context.Context) (*coreosv1alpha1.CatalogSource, error) {
	if r.selfCatalogSource == nil {
		ns, err := k8sresources.GetWatchNamespace()
		if err != nil {
			return nil, err
		}

		installation, err := rhmiresources.GetRhmiCr(r.Client, ctx, ns, r.Log)
		if err != nil {
			return nil, err
		}

		catalogSource, err := addon.GetCatalogSource(ctx, r.Client, installation)
		if err != nil {
			return nil, err
		}

		r.selfCatalogSource = catalogSource
	}

	return r.selfCatalogSource, nil
}
