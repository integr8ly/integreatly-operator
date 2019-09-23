package products

import (
	"context"
	"crypto/tls"
	"errors"

	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/monitoring"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"net/http"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/amqstreams"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/codeready"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/fuse"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/fuseonopenshift"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/launcher"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/mobilesecurityservice"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/nexus"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/rhsso"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/rhssouser"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/solutionexplorer"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/ups"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"k8s.io/client-go/rest"

	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/amqonline"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/threescale"
	appsv1Client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate moq -out Reconciler_moq.go . Interface
type Interface interface {
	Reconcile(ctx context.Context, inst *v1alpha1.Installation, product *v1alpha1.InstallationProductStatus, serverClient client.Client) (newPhase v1alpha1.StatusPhase, err error)
	GetPreflightObject(ns string) runtime.Object
}

func NewReconciler(product v1alpha1.ProductName, rc *rest.Config, configManager config.ConfigReadWriter, instance *v1alpha1.Installation) (reconciler Interface, err error) {
	mpm := marketplace.NewManager()
	oauthResolver := resources.NewOauthResolver(http.DefaultClient)
	oauthResolver.Host = rc.Host

	switch product {
	case v1alpha1.ProductAMQStreams:
		reconciler, err = amqstreams.NewReconciler(configManager, instance, mpm)
	case v1alpha1.ProductRHSSO:
		oauthv1Client, err := oauthClient.NewForConfig(rc)
		if err != nil {
			return nil, err
		}

		reconciler, err = rhsso.NewReconciler(configManager, instance, oauthv1Client, mpm)
	case v1alpha1.ProductRHSSOUser:
		oauthv1Client, err := oauthClient.NewForConfig(rc)
		if err != nil {
			return nil, err
		}

		reconciler, err = rhssouser.NewReconciler(configManager, instance, oauthv1Client, mpm)
	case v1alpha1.ProductCodeReadyWorkspaces:
		reconciler, err = codeready.NewReconciler(configManager, instance, mpm)
	case v1alpha1.ProductFuse:
		reconciler, err = fuse.NewReconciler(configManager, instance, mpm)
	case v1alpha1.ProductFuseOnOpenshift:
		reconciler, err = fuseonopenshift.NewReconciler(configManager, instance, mpm)
	case v1alpha1.ProductAMQOnline:
		reconciler, err = amqonline.NewReconciler(configManager, instance, mpm)
	case v1alpha1.ProductSolutionExplorer:
		oauthv1Client, err := oauthClient.NewForConfig(rc)
		if err != nil {
			return nil, err
		}

		reconciler, err = solutionexplorer.NewReconciler(configManager, instance, oauthv1Client, mpm, oauthResolver)
	case v1alpha1.ProductLauncher:
		appsv1, err := appsv1Client.NewForConfig(rc)
		if err != nil {
			return nil, err
		}

		reconciler, err = launcher.NewReconciler(configManager, instance, appsv1, mpm)
	case v1alpha1.ProductMonitoring:
		reconciler, err = monitoring.NewReconciler(configManager, instance, mpm)
	case v1alpha1.ProductMobileSecurityService:
		reconciler, err = mobilesecurityservice.NewReconciler(configManager, instance, mpm)
	case v1alpha1.Product3Scale:
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
				TLSClientConfig: &tls.Config{InsecureSkipVerify: instance.Spec.SelfSignedCerts},
			},
		}

		tsClient := threescale.NewThreeScaleClient(httpc, instance.Spec.RoutingSubdomain)

		reconciler, err = threescale.NewReconciler(configManager, instance, appsv1, oauthv1Client, tsClient, mpm)
	case v1alpha1.ProductNexus:
		reconciler, err = nexus.NewReconciler(configManager, instance, mpm)
	case v1alpha1.ProductUps:
		reconciler, err = ups.NewReconciler(configManager, instance, mpm)
	default:
		err = errors.New("unknown products: " + string(product))
		reconciler = &NoOp{}
	}
	return reconciler, err
}

type NoOp struct {
}

func (n *NoOp) Reconcile(ctx context.Context, inst *v1alpha1.Installation, product *v1alpha1.InstallationProductStatus, serverClient client.Client) (v1alpha1.StatusPhase, error) {
	return v1alpha1.PhaseNone, nil
}

func (n *NoOp) GetPreflightObject(ns string) runtime.Object {
	return &v1.Deployment{}
}
