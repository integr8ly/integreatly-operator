package marketplace

import (
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"

	"github.com/integr8ly/integreatly-operator/pkg/addon"
	k8sresources "github.com/integr8ly/integreatly-operator/pkg/resources/k8s"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	rhmiresources "github.com/integr8ly/integreatly-operator/pkg/resources/rhmi"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
)

// ImplicitCatalogSourceReconciler is a CatalogSourceReconciler implementation
// that does not create a new CatalogSource, but expects the operator to be
// included in the CatalogSource installed by the add-on
type ImplicitCatalogSourceReconciler struct {
	Client            k8sclient.Client
	Log               logger.Logger
	selfCatalogSource *coreosv1alpha1.CatalogSource
}

var _ CatalogSourceReconciler = &ImplicitCatalogSourceReconciler{}

func NewImplicitCatalogSourceReconciler(log logger.Logger, client k8sclient.Client) (*ImplicitCatalogSourceReconciler, error) {
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

	logrus.Info("Found catalog source %s in %s", catalogSource.Name, catalogSource.Namespace)

	if (catalogSource.Namespace == "redhat-rhoam-operator" && strings.Contains(subName, "3scale")) {
		// Create same one in "redhat-rhoam-3scale-operator"
		// then when the subscription is being created

		logrus.Info("Copying 3scale Catalog source")

		cs := &coreosv1alpha1.CatalogSource{
			ObjectMeta: v1.ObjectMeta{
				Name:      catalogSource.Name,
				Namespace: "redhat-rhoam-3scale-operator",
			},
		}

		controllerutil.CreateOrUpdate(ctx, r.Client, cs, func() error {
			cs.Spec = catalogSource.DeepCopy().Spec
			return nil
		})


		logrus.Info("Created 3SCale CS, image: " + cs.Spec.Image)

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
