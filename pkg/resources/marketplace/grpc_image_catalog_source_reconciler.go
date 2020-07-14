package marketplace

import (
	"context"
	"fmt"

	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type GRPCImageCatalogSourceReconciler struct {
	Image     string
	Client    k8sclient.Client
	Namespace string
	CSName    string
}

var _ CatalogSourceReconciler = &GRPCImageCatalogSourceReconciler{}

func NewGRPCImageCatalogSourceReconciler(image string, client client.Client, namespace string, catalogSourceName string) *GRPCImageCatalogSourceReconciler {
	return &GRPCImageCatalogSourceReconciler{
		Image:     image,
		Client:    client,
		Namespace: namespace,
		CSName:    catalogSourceName,
	}
}

func (r *GRPCImageCatalogSourceReconciler) Reconcile(ctx context.Context) (reconcile.Result, error) {
	logrus.Infof("Reconciling registry catalog source for namespace %s", r.Namespace)

	catalogSource := &coreosv1alpha1.CatalogSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.CatalogSourceName(),
			Namespace: r.Namespace,
		},
	}

	catalogSourceSpec := coreosv1alpha1.CatalogSourceSpec{
		SourceType:  coreosv1alpha1.SourceTypeGrpc,
		Image:       r.Image,
		DisplayName: r.CatalogSourceName(),
		Publisher:   Publisher,
	}

	or, err := controllerutil.CreateOrUpdate(ctx, r.Client, catalogSource, func() error {
		catalogSource.Spec = catalogSourceSpec
		return nil
	})
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create/update registry catalog source for namespace '%s': %w", r.Namespace, err)
	}

	switch or {
	case controllerutil.OperationResultCreated:
		logrus.Infof("Created registry catalog source for namespace %s", r.Namespace)
	case controllerutil.OperationResultUpdated:
		logrus.Infof("Updated registry catalog source for namespace %s", r.Namespace)
	case controllerutil.OperationResultNone:
		break
	default:
		return reconcile.Result{}, fmt.Errorf("Unknown controllerutil.OperationResult '%v'", or)
	}

	logrus.Infof("Successfully reconciled registry catalog source for namespace %s", r.Namespace)

	return reconcile.Result{}, nil
}

func (r *GRPCImageCatalogSourceReconciler) CatalogSourceName() string {
	return r.CSName
}
