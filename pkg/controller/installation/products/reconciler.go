package products

import (
	"errors"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/amqstreams"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/codeready"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/fuse"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/rhsso"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate moq -out Reconciler_moq.go . Interface
type Interface interface {
	Reconcile(inst *v1alpha1.Installation, serverClient client.Client) (newPhase v1alpha1.StatusPhase, err error)
}

func NewReconciler(product v1alpha1.ProductName, client client.Client, rc *rest.Config, coreClient *kubernetes.Clientset, configManager config.ConfigReadWriter, instance *v1alpha1.Installation) (reconciler Interface, err error) {
	mpm := marketplace.NewManager(client, rc)
	switch product {
	case v1alpha1.ProductAMQStreams:
		reconciler, err = amqstreams.NewReconciler(coreClient, configManager, instance, mpm)
	case v1alpha1.ProductRHSSO:
		reconciler, err = rhsso.NewReconciler(coreClient, configManager, instance, mpm)
	case v1alpha1.ProductCodeReadyWorkspaces:
		reconciler, err = codeready.NewReconciler(configManager, instance, mpm)
	case v1alpha1.ProductFuse:
		reconciler, err = fuse.NewReconciler(coreClient, configManager, instance, mpm)
	default:
		err = errors.New("unknown products: " + string(product))
		reconciler = &NoOp{}
	}
	return reconciler, err
}

type NoOp struct {
}

func (n *NoOp) Reconcile(inst *v1alpha1.Installation, serverClient client.Client) (v1alpha1.StatusPhase, error) {
	return v1alpha1.PhaseNone, nil
}
