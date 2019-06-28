package products

import (
	"errors"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/amqstreams"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/rhsso"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/threescale"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type Interface interface {
	Reconcile(instance *v1alpha1.Installation) (newPhase v1alpha1.StatusPhase, err error)
}

func NewReconciler(product v1alpha1.ProductName, client client.Client, rc *rest.Config, configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mgr manager.Manager) (reconciler Interface, err error) {
	switch product {
	case v1alpha1.ProductAMQStreams:
		reconciler, err = amqstreams.NewReconciler(client, rc, configManager, instance, mgr)
	case v1alpha1.ProductRHSSO:
		reconciler, err = rhsso.NewReconciler(client, rc, configManager, instance, mgr)
	case v1alpha1.Product3Scale:
		reconciler, err = threescale.NewReconciler(client, rc, configManager, instance, mgr)
	default:
		err = errors.New("unknown products: " + string(product))
		reconciler = &NoOp{}
	}
	return reconciler, err
}

type NoOp struct {
}

func (n *NoOp) Reconcile(instance *v1alpha1.Installation) (v1alpha1.StatusPhase, error) {
	return v1alpha1.PhaseNone, nil
}
