package products

import (
	"context"
	"crypto/tls"
	"errors"
	"github.com/integr8ly/integreatly-operator/pkg/products/marin3r"
	"time"

	"github.com/integr8ly/integreatly-operator/pkg/products/monitoringspec"

	keycloakCommon "github.com/integr8ly/keycloak-client/pkg/common"

	"net/http"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/products/amqstreams"
	"github.com/integr8ly/integreatly-operator/pkg/products/apicurioregistry"
	"github.com/integr8ly/integreatly-operator/pkg/products/apicurito"
	"github.com/integr8ly/integreatly-operator/pkg/products/cloudresources"
	"github.com/integr8ly/integreatly-operator/pkg/products/codeready"
	"github.com/integr8ly/integreatly-operator/pkg/products/datasync"
	"github.com/integr8ly/integreatly-operator/pkg/products/fuse"
	"github.com/integr8ly/integreatly-operator/pkg/products/fuseonopenshift"
	"github.com/integr8ly/integreatly-operator/pkg/products/grafana"
	"github.com/integr8ly/integreatly-operator/pkg/products/monitoring"
	"github.com/integr8ly/integreatly-operator/pkg/products/rhsso"
	"github.com/integr8ly/integreatly-operator/pkg/products/rhssouser"
	"github.com/integr8ly/integreatly-operator/pkg/products/solutionexplorer"
	"github.com/integr8ly/integreatly-operator/pkg/products/ups"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
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
	//This is a pointer to the this reconciler's product in the status block of the CR, it can be used to set values
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
	Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (newPhase integreatlyv1alpha1.StatusPhase, err error)

	//GetPreflightObjects informs the operator of what object it should look for, to check if the product is already installed. The
	//namespace argument is the namespace currently being scanned for existing installations.
	//
	//For example, codeready looks for a deployment in the scanned namespace with the name "codeready", if found this
	//installation will stall until that product is removed.
	GetPreflightObject(ns string) runtime.Object

	//VerifyVersion checks if the version of the product installed is the same as the one defined in the operator
	//
	VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool
}

func NewReconciler(product integreatlyv1alpha1.ProductName, rc *rest.Config, configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mgr manager.Manager, log l.Logger) (reconciler Interface, err error) {
	mpm := marketplace.NewManager()
	oauthHttpClient := &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			IdleConnTimeout:   time.Second * 10,
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: installation.Spec.SelfSignedCerts},
		},
	}
	oauthResolver := resources.NewOauthResolver(oauthHttpClient, log)
	oauthResolver.Host = rc.Host
	recorder := mgr.GetEventRecorderFor(string(product))

	switch product {
	case integreatlyv1alpha1.ProductAMQStreams:
		reconciler, err = amqstreams.NewReconciler(configManager, installation, mpm, recorder, log)
	case integreatlyv1alpha1.ProductRHSSO:
		oauthv1Client, err := oauthClient.NewForConfig(rc)
		oauthv1Client.RESTClient().(*rest.RESTClient).Client.Timeout = 10 * time.Second
		if err != nil {
			return nil, err
		}
		reconciler, err = rhsso.NewReconciler(configManager, installation, oauthv1Client, mpm, recorder, rc.Host, &keycloakCommon.LocalConfigKeycloakFactory{}, log)
		if err != nil {
			return nil, err
		}
	case integreatlyv1alpha1.ProductRHSSOUser:
		oauthv1Client, err := oauthClient.NewForConfig(rc)
		oauthv1Client.RESTClient().(*rest.RESTClient).Client.Timeout = 10 * time.Second
		if err != nil {
			return nil, err
		}
		reconciler, err = rhssouser.NewReconciler(configManager, installation, oauthv1Client, mpm, recorder, rc.Host, &keycloakCommon.LocalConfigKeycloakFactory{}, log)
		if err != nil {
			return nil, err
		}
	case integreatlyv1alpha1.ProductCodeReadyWorkspaces:
		reconciler, err = codeready.NewReconciler(configManager, installation, mpm, recorder, log)
	case integreatlyv1alpha1.ProductFuse:
		reconciler, err = fuse.NewReconciler(configManager, installation, mpm, recorder, log)
	case integreatlyv1alpha1.ProductFuseOnOpenshift:
		reconciler, err = fuseonopenshift.NewReconciler(configManager, installation, mpm, recorder, &http.Client{}, "", log)
	case integreatlyv1alpha1.ProductAMQOnline:
		reconciler, err = amqonline.NewReconciler(configManager, installation, mpm, recorder, log)
	case integreatlyv1alpha1.ProductSolutionExplorer:
		oauthv1Client, err := oauthClient.NewForConfig(rc)
		oauthv1Client.RESTClient().(*rest.RESTClient).Client.Timeout = 10 * time.Second
		if err != nil {
			return nil, err
		}
		reconciler, err = solutionexplorer.NewReconciler(configManager, installation, oauthv1Client, mpm, oauthResolver, recorder, log)
		if err != nil {
			return nil, err
		}
	case integreatlyv1alpha1.ProductMonitoring:
		reconciler, err = monitoring.NewReconciler(configManager, installation, mpm, recorder, log)
	case integreatlyv1alpha1.ProductMonitoringSpec:
		reconciler, err = monitoringspec.NewReconciler(configManager, installation, mpm, recorder, log)
	case integreatlyv1alpha1.ProductApicurioRegistry:
		reconciler, err = apicurioregistry.NewReconciler(configManager, installation, mpm, recorder, log)
	case integreatlyv1alpha1.ProductApicurito:
		reconciler, err = apicurito.NewReconciler(configManager, installation, mpm, recorder, log)
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

		httpc := &http.Client{
			Timeout: time.Second * 10,
			Transport: &http.Transport{
				DisableKeepAlives: true,
				IdleConnTimeout:   time.Second * 10,
				TLSClientConfig:   &tls.Config{InsecureSkipVerify: installation.Spec.SelfSignedCerts},
			},
		}

		tsClient := threescale.NewThreeScaleClient(httpc, installation.Spec.RoutingSubdomain)
		reconciler, err = threescale.NewReconciler(configManager, installation, client, oauthv1Client, tsClient, mpm, recorder, log)
		if err != nil {
			return nil, err
		}
	case integreatlyv1alpha1.ProductUps:
		reconciler, err = ups.NewReconciler(configManager, installation, mpm, recorder, log)
	case integreatlyv1alpha1.ProductCloudResources:
		reconciler, err = cloudresources.NewReconciler(configManager, installation, mpm, recorder, log)
	case integreatlyv1alpha1.ProductDataSync:
		reconciler, err = datasync.NewReconciler(configManager, installation, mpm, recorder, log)
	case integreatlyv1alpha1.ProductMarin3r:
		reconciler, err = marin3r.NewReconciler(configManager, installation, mpm, recorder, log)
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

func (n *NoOp) Reconcile(_ context.Context, _ *integreatlyv1alpha1.RHMI, _ *integreatlyv1alpha1.RHMIProductStatus, _ k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	return integreatlyv1alpha1.PhaseNone, nil
}

func (n *NoOp) GetPreflightObject(ns string) runtime.Object {
	return &appsv1.Deployment{}
}

func (n *NoOp) VerifyVersion(_ *integreatlyv1alpha1.RHMI) bool {
	return true
}
