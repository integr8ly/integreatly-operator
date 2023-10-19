package products

import (
	"context"
	"crypto/tls"
	"errors"
	"time"

	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"

	"github.com/integr8ly/integreatly-operator/pkg/products/marin3r"

	keycloakCommon "github.com/integr8ly/keycloak-client/pkg/common"

	"net/http"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/products/cloudresources"
	"github.com/integr8ly/integreatly-operator/pkg/products/grafana"
	"github.com/integr8ly/integreatly-operator/pkg/products/rhsso"
	"github.com/integr8ly/integreatly-operator/pkg/products/rhssouser"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	"github.com/integr8ly/integreatly-operator/pkg/products/threescale"
	appsv1Client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/rest"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

//go:generate moq -out Reconciler_moq.go . Interface
type Interface interface {
	//Reconcile is the primary entry point of your reconciler, on each reconcile loop of the integreatly-operator,
	//all the logic in here should be written with the assumption that resources may or may not exist, and can
	//be created or updated based on their current state.
	//
	//## Parameters
	//There are several parameters passed into this function:
	//
	//### ctx
	// This must be used in all network requests performed by the reconciler, as the integreatly-operator maintains this context
	// and may kill it if an uninstall is detected.
	//
	//### installation
	//This is the CR we are basing the install from, it has values that are occasionally required by reconcilers, for example
	//the namespace prefix.
	//
	//### product
	//This is a pointer to this reconciler's product in the status block of the CR, it can be used to set values
	//such as version, host and operator version.
	//### serverClient
	//This is the client to the cluster, and is used for getting, creating, updating and deleting resources in the cluster.
	//
	//## Return Values
	//The return values from this method are `state` and `err`:
	//
	//### State
	//This is communicated back to the user via the status block of the RHMI CR, this is usually either in progress
	//or complete. It can go to `fail` if something has broken, but this will not prevent the installation_controller
	//from calling the Reconcile function in the future, which may allow the reconciler to fix whatever issue had
	//occurred (i.e. the service had not come up yet, so there were network errors accessing it's API).
	//
	//### Err
	//This is how we can communicate to the user via the status block of the RHMI CR what is causing a product to
	//enter a failed state, and is written into the `status.lastError` of the CR along with any other errors from
	//other reconcilers.
	Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client, productConfig quota.ProductConfig, uninstall bool) (newPhase integreatlyv1alpha1.StatusPhase, err error)

	//GetPreflightObject informs the operator of what object it should look for, to check if the product is already installed. The
	//namespace argument is the namespace currently being scanned for existing installations.
	//
	//For example, codeready looks for a deployment in the scanned namespace with the name "codeready", if found this
	//installation will stall until that product is removed.
	GetPreflightObject(ns string) k8sclient.Object

	//VerifyVersion checks if the version of the product installed is the same as the one defined in the operator
	//
	VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool
}

func NewReconciler(product integreatlyv1alpha1.ProductName, rc *rest.Config, configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mgr manager.Manager, log l.Logger, productsInstallationLoader marketplace.ProductsInstallationLoader) (reconciler Interface, err error) {
	mpm := marketplace.NewManager()

	if installation.Spec.SelfSignedCerts {
		log.Warning("TLS insecure skip verify is enabled")
	}

	recorder := mgr.GetEventRecorderFor(string(product))

	productsInstallation, err := productsInstallationLoader.GetProductsInstallation()
	if err != nil {
		return nil, err
	}

	var productDeclaration *marketplace.ProductDeclaration
	pd, ok := productsInstallation.Products[string(product)]
	if ok {
		productDeclaration = &pd
	}

	switch product {
	case integreatlyv1alpha1.ProductRHSSO:
		oauthv1Client, err := oauthClient.NewForConfig(rc)
		oauthv1Client.RESTClient().(*rest.RESTClient).Client.Timeout = 10 * time.Second
		if err != nil {
			return nil, err
		}
		reconciler, err = rhsso.NewReconciler(configManager, installation, oauthv1Client, mpm, recorder, rc.Host, &keycloakCommon.LocalConfigKeycloakFactory{}, log, productDeclaration)
		if err != nil {
			return nil, err
		}
	case integreatlyv1alpha1.ProductRHSSOUser:
		oauthv1Client, err := oauthClient.NewForConfig(rc)
		oauthv1Client.RESTClient().(*rest.RESTClient).Client.Timeout = 10 * time.Second
		if err != nil {
			return nil, err
		}
		reconciler, err = rhssouser.NewReconciler(configManager, installation, oauthv1Client, mpm, recorder, rc.Host, &keycloakCommon.LocalConfigKeycloakFactory{}, log, productDeclaration)
		if err != nil {
			return nil, err
		}
	case integreatlyv1alpha1.Product3Scale:
		client, err := appsv1Client.NewForConfig(rc)
		client.RESTClient().(*rest.RESTClient).Client.Timeout = 10 * time.Second
		if err != nil {
			return nil, err
		}

		oauthv1Client, err := oauthClient.NewForConfig(rc)
		oauthv1Client.RESTClient().(*rest.RESTClient).Client.Timeout = 10 * time.Second
		if err != nil {
			return nil, err
		}

		/* #nosec */
		httpc := &http.Client{
			Timeout: time.Second * 10,
			Transport: &http.Transport{
				DisableKeepAlives: true,
				IdleConnTimeout:   time.Second * 10,
				TLSClientConfig:   &tls.Config{InsecureSkipVerify: installation.Spec.SelfSignedCerts}, // gosec G402, value is read from CR config
			},
		}

		if installation.Spec.SelfSignedCerts {
			log.Warning("TLS insecure skip verify is enabled")
		}

		tsClient := threescale.NewThreeScaleClient(httpc, installation.Spec.RoutingSubdomain)
		reconciler, err = threescale.NewReconciler(configManager, installation, client, oauthv1Client, tsClient, mpm, recorder, log, productDeclaration)
		if err != nil {
			return nil, err
		}
	case integreatlyv1alpha1.ProductCloudResources:
		reconciler, err = cloudresources.NewReconciler(configManager, installation, mpm, recorder, log, productDeclaration)
	case integreatlyv1alpha1.ProductMarin3r:
		reconciler, err = marin3r.NewReconciler(configManager, installation, mpm, recorder, log, productDeclaration)
	case integreatlyv1alpha1.ProductGrafana:
		reconciler, err = grafana.NewReconciler(configManager, installation, mpm, recorder, log)
	default:
		err = errors.New("unknown products: " + string(product))
		reconciler = &NoOp{}
	}

	return reconciler, err
}

type NoOp struct {
}

func (n *NoOp) Reconcile(_ context.Context, _ *integreatlyv1alpha1.RHMI, _ *integreatlyv1alpha1.RHMIProductStatus, _ k8sclient.Client, _ quota.ProductConfig, _ bool) (integreatlyv1alpha1.StatusPhase, error) {
	return integreatlyv1alpha1.PhaseNone, nil
}

func (n *NoOp) GetPreflightObject(_ string) k8sclient.Object {
	return &appsv1.Deployment{}
}

func (n *NoOp) VerifyVersion(_ *integreatlyv1alpha1.RHMI) bool {
	return true
}
