package products

import (
	"context"
	"crypto/tls"
	"errors"

	"github.com/integr8ly/integreatly-operator/pkg/products/monitoring"

	"net/http"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/products/amqstreams"
	"github.com/integr8ly/integreatly-operator/pkg/products/cloudresources"
	"github.com/integr8ly/integreatly-operator/pkg/products/codeready"
	"github.com/integr8ly/integreatly-operator/pkg/products/fuse"
	"github.com/integr8ly/integreatly-operator/pkg/products/fuseonopenshift"
	"github.com/integr8ly/integreatly-operator/pkg/products/rhsso"
	"github.com/integr8ly/integreatly-operator/pkg/products/rhssouser"
	"github.com/integr8ly/integreatly-operator/pkg/products/solutionexplorer"
	"github.com/integr8ly/integreatly-operator/pkg/products/ups"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	"github.com/integr8ly/integreatly-operator/pkg/products/amqonline"
	"github.com/integr8ly/integreatly-operator/pkg/products/threescale"
	appsv1Client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

//go:generate moq -out Reconciler_moq.go . Interface
type Interface interface {
	Reconcile(ctx context.Context, installation *integreatlyv1alpha1.Installation, product *integreatlyv1alpha1.InstallationProductStatus, serverClient k8sclient.Client) (newPhase integreatlyv1alpha1.StatusPhase, err error)
	GetPreflightObject(ns string) runtime.Object
}

func NewReconciler(product integreatlyv1alpha1.ProductName, rc *rest.Config, configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.Installation, mgr manager.Manager) (reconciler Interface, err error) {
	mpm := marketplace.NewManager()
	oauthResolver := resources.NewOauthResolver(http.DefaultClient)
	oauthResolver.Host = rc.Host
	recorder := mgr.GetEventRecorderFor(string(product))

	switch product {
	case integreatlyv1alpha1.ProductAMQStreams:
		reconciler, err = amqstreams.NewReconciler(configManager, installation, mpm, recorder)
	case integreatlyv1alpha1.ProductRHSSO:
		oauthv1Client, err := oauthClient.NewForConfig(rc)
		if err != nil {
			return nil, err
		}
		reconciler, err = rhsso.NewReconciler(configManager, installation, oauthv1Client, mpm, recorder, rc.Host)
	case integreatlyv1alpha1.ProductRHSSOUser:
		oauthv1Client, err := oauthClient.NewForConfig(rc)
		if err != nil {
			return nil, err
		}
		reconciler, err = rhssouser.NewReconciler(configManager, installation, oauthv1Client, mpm, recorder, rc.Host)
	case integreatlyv1alpha1.ProductCodeReadyWorkspaces:
		reconciler, err = codeready.NewReconciler(configManager, installation, mpm, recorder)
	case integreatlyv1alpha1.ProductFuse:
		reconciler, err = fuse.NewReconciler(configManager, installation, mpm, recorder)
	case integreatlyv1alpha1.ProductFuseOnOpenshift:
		reconciler, err = fuseonopenshift.NewReconciler(configManager, installation, mpm, recorder)
	case integreatlyv1alpha1.ProductAMQOnline:
		reconciler, err = amqonline.NewReconciler(configManager, installation, mpm, recorder)
	case integreatlyv1alpha1.ProductSolutionExplorer:
		oauthv1Client, err := oauthClient.NewForConfig(rc)
		if err != nil {
			return nil, err
		}

		reconciler, err = solutionexplorer.NewReconciler(configManager, installation, oauthv1Client, mpm, oauthResolver, recorder)
	case integreatlyv1alpha1.ProductMonitoring:
		reconciler, err = monitoring.NewReconciler(configManager, installation, mpm, recorder)
	case integreatlyv1alpha1.Product3Scale:
		appsv1, err := appsv1Client.NewForConfig(rc)
		if err != nil {
			return nil, err
		}

		oauthv1Client, err := oauthClient.NewForConfig(rc)
		if err != nil {
			return nil, err
		}

		httpc := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: installation.Spec.SelfSignedCerts},
			},
		}

		tsClient := threescale.NewThreeScaleClient(httpc, installation.Spec.RoutingSubdomain)

		reconciler, err = threescale.NewReconciler(configManager, installation, appsv1, oauthv1Client, tsClient, mpm, recorder)
	case integreatlyv1alpha1.ProductUps:
		reconciler, err = ups.NewReconciler(configManager, installation, mpm, recorder)
	case integreatlyv1alpha1.ProductCloudResources:
		reconciler, err = cloudresources.NewReconciler(configManager, installation, mpm, recorder)
	default:
		err = errors.New("unknown products: " + string(product))
		reconciler = &NoOp{}
	}
	return reconciler, err
}

type NoOp struct {
}

func (n *NoOp) Reconcile(_ context.Context, _ *integreatlyv1alpha1.Installation, _ *integreatlyv1alpha1.InstallationProductStatus, _ k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	return integreatlyv1alpha1.PhaseNone, nil
}

func (n *NoOp) GetPreflightObject(ns string) runtime.Object {
	return &appsv1.Deployment{}
}
