package products

import (
	"context"
	"errors"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/amqstreams"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/codeready"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/fuse"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/rhsso"
	"k8s.io/client-go/rest"

	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/amqonline"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/threescale"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	appsv1Client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate moq -out Reconciler_moq.go . Interface
type Interface interface {
	Reconcile(ctx context.Context, inst *v1alpha1.Installation, serverClient client.Client) (newPhase v1alpha1.StatusPhase, err error)
}

func NewReconciler(product v1alpha1.ProductName, client client.Client, rc *rest.Config, configManager config.ConfigReadWriter, instance *v1alpha1.Installation) (reconciler Interface, err error) {
	mpm := marketplace.NewManager()
	nsr := resources.NewNamespaceReconciler(client)
	switch product {
	case v1alpha1.ProductAMQStreams:
		reconciler, err = amqstreams.NewReconciler(configManager, instance, mpm)
	case v1alpha1.ProductRHSSO:
		reconciler, err = rhsso.NewReconciler(configManager, instance, mpm)
	case v1alpha1.ProductCodeReadyWorkspaces:
		reconciler, err = codeready.NewReconciler(configManager, instance, mpm)
	case v1alpha1.ProductFuse:
		reconciler, err = fuse.NewReconciler(configManager, instance, mpm)
	case v1alpha1.ProductAMQOnline:
		reconciler, err = amqonline.NewReconciler(configManager, instance, mpm, nsr)
	case v1alpha1.Product3Scale:
		appsv1, err := appsv1Client.NewForConfig(rc)
		if err != nil {
			return nil, err
		}

		oauthv1Client, err := oauthClient.NewForConfig(rc)
		if err != nil {
			return nil, err
		}

		httpc := &http.Client{}
		tsClient := threescale.NewThreeScaleClient(httpc, instance.Spec.RoutingSubdomain)

		reconciler, err = threescale.NewReconciler(configManager, instance, appsv1, oauthv1Client, tsClient, mpm)
	default:
		err = errors.New("unknown products: " + string(product))
		reconciler = &NoOp{}
	}
	return reconciler, err
}

type NoOp struct {
}

func (n *NoOp) Reconcile(ctx context.Context, inst *v1alpha1.Installation, serverClient client.Client) (v1alpha1.StatusPhase, error) {
	return v1alpha1.PhaseNone, nil
}
