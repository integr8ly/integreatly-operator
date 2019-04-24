package products

import (
	"errors"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/amqstreams"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Interface interface {
	Reconcile(phase v1alpha1.StatusPhase) (newPhase v1alpha1.StatusPhase, err error)
}

func NewReconciler(product v1alpha1.ProductName, client client.Client, configManager config.ConfigReadWriter, namespacePrefix string, clusterHasOLM bool) (reconciler Interface, err error) {
	switch product {
	case v1alpha1.ProductAMQStreams:
		reconciler, err = amqstreams.NewReconciler(client, configManager, namespacePrefix, clusterHasOLM)
	default:
		err = errors.New("unknown products: " + string(product))
		reconciler = &NoOp{}
	}
	return reconciler, err
}

type NoOp struct {
}

func (n *NoOp) Reconcile(phase v1alpha1.StatusPhase) (v1alpha1.StatusPhase, error) {
	return phase, nil
}
