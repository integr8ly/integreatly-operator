package solutionexplorer

import (
	"context"
	"encoding/json"
	"fmt"
	v1alpha12 "github.com/integr8ly/integreatly-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/monitoring"
	"strings"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	webapp "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/tutorial-web-app-operator/pkg/apis/v1alpha1"
	appsv1 "github.com/openshift/api/apps/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	routev1 "github.com/openshift/api/route/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultName             = "solution-explorer"
	defaultSubNameAndPkg    = "integreatly-solution-explorer"
	defaultTemplateLoc      = "/home/tutorial-web-app-operator/deploy/template/tutorial-web-app.yml"
	paramOpenShiftHost      = "OPENSHIFT_HOST"
	paramOpenShiftOauthHost = "OPENSHIFT_OAUTH_HOST"
	paramOauthClient        = "OPENSHIFT_OAUTHCLIENT_ID"
	paramOpenShiftVersion   = "OPENSHIFT_VERSION"
	paramInstalledServices  = "INSTALLED_SERVICES"
	paramSSORoute           = "SSO_ROUTE"
	defaultRouteName        = "tutorial-web-app"
)

type Reconciler struct {
	*resources.Reconciler
	coreClient    kubernetes.Interface
	oauthv1Client oauthClient.OauthV1Interface
	Config        *config.SolutionExplorer
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
	OauthResolver OauthResolver
	installation  *v1alpha1.Installation
}

//go:generate moq -out OauthResolver_moq.go . OauthResolver
type OauthResolver interface {
	GetOauthEndPoint() (*resources.OauthServerConfig, error)
}

type productInfo struct {
	Host    string
	Version v1alpha1.ProductVersion
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, oauthv1Client oauthClient.OauthV1Interface, mpm marketplace.MarketplaceInterface, resolver OauthResolver) (*Reconciler, error) {
	seConfig, err := configManager.ReadSolutionExplorer()
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve solution explorer config")
	}

	if seConfig.GetNamespace() == "" {
		seConfig.SetNamespace(instance.Spec.NamespacePrefix + defaultName)
	}
	if err = seConfig.Validate(); err != nil {
		return nil, errors.Wrap(err, "solution explorer config is not valid")
	}

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		ConfigManager: configManager,
		Config:        seConfig,
		mpm:           mpm,
		logger:        logger,
		Reconciler:    resources.NewReconciler(mpm),
		OauthResolver: resolver,
		oauthv1Client: oauthv1Client,
		installation:  instance,
	}, nil
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tutorial-web-app",
			Namespace: ns,
		},
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, product *v1alpha1.InstallationProductStatus, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Info("Reconciling solution explorer")

	phase, err := r.ReconcileFinalizer(ctx, serverClient, inst, string(r.Config.GetProductName()), func() (v1alpha1.StatusPhase, error) {
		phase, err := resources.RemoveNamespace(ctx, inst, serverClient, r.Config.GetNamespace())
		if err != nil || phase != v1alpha1.PhaseCompleted {
			return phase, err
		}

		err = resources.RemoveOauthClient(ctx, inst, serverClient, r.oauthv1Client, r.getOAuthClientName())
		if err != nil {
			return v1alpha1.PhaseFailed, err
		}
		return v1alpha1.PhaseCompleted, nil
	})
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetNamespace(), inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, r.Config.GetNamespace(), serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	version, err := resources.NewVersion(v1alpha1.OperatorVersionSolutionExplorer)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "invalid version number for solution explorer")
	}
	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: defaultSubNameAndPkg, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetNamespace()}, r.Config.GetNamespace(), serverClient, version)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.ReconcileCustomResource(ctx, inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	route, err := r.ensureAppUrl(ctx, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	if r.Config.GetHost() != route {
		r.Config.SetHost(route)
		r.ConfigManager.WriteConfig(r.Config)
	}

	phase, err = r.ReconcileOauthClient(ctx, inst, &oauthv1.OAuthClient{
		RedirectURIs: []string{route},
		GrantMethod:  oauthv1.GrantHandlerAuto,
		ObjectMeta: metav1.ObjectMeta{
			Name: r.getOAuthClientName(),
		},
	}, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileBlackboxTarget(ctx, inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileBlackboxTarget(ctx context.Context, inst *v1alpha1.Installation, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	cfg, err := r.ConfigManager.ReadMonitoring()
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "error reading monitoring config")
	}

	target := v1alpha12.BlackboxtargetData{
		Url:     r.Config.GetHost(),
		Service: "webapp-ui",
	}

	err = monitoring.CreateBlackboxTarget("integreatly-webapp", target, ctx, cfg, inst, client)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "error creating solution explorer blackbox target")
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) ensureAppUrl(ctx context.Context, client pkgclient.Client) (string, error) {
	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      defaultRouteName,
		},
	}
	if err := client.Get(ctx, pkgclient.ObjectKey{Name: route.Name, Namespace: route.Namespace}, route); err != nil {
		return "", errors.Wrap(err, "failed to get route for solution explorer")
	}
	protocol := "https"
	if route.Spec.TLS == nil {
		protocol = "http"
	}

	return fmt.Sprintf("%s://%s", protocol, route.Spec.Host), nil
}

func (r *Reconciler) ReconcileCustomResource(ctx context.Context, inst *v1alpha1.Installation, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	//todo shouldn't need to do this with each reconcile
	oauthConfig, err := r.OauthResolver.GetOauthEndPoint()
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to get oauth details ")
	}
	ssoConfig, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	seCR := &webapp.WebApp{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      defaultName,
		},
	}
	oauthURL := strings.Replace(strings.Replace(oauthConfig.AuthorizationEndpoint, "https://", "", 1), "/oauth/authorize", "", 1)
	logrus.Info("ReconcileCustomResource setting url for openshift host ", oauthURL)

	installedServices, err := r.getInstalledProducts(inst)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrapf(err, "failed to retrieve installed products information from %s CR", inst.Name)
	}
	_, err = controllerutil.CreateOrUpdate(ctx, client, seCR, func(existing runtime.Object) error {
		cr := existing.(*webapp.WebApp)
		cr.Spec.AppLabel = "tutorial-web-app"
		cr.Spec.Template.Path = defaultTemplateLoc
		cr.Spec.Template.Parameters = map[string]string{
			paramOauthClient:        r.getOAuthClientName(),
			paramSSORoute:           ssoConfig.GetHost(),
			paramOpenShiftHost:      inst.Spec.MasterURL,
			paramOpenShiftOauthHost: oauthURL,
			paramOpenShiftVersion:   "4",
			paramInstalledServices:  installedServices,
		}
		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to reconcile webapp resource")
	}
	// do a get to ensure we have an upto date copy
	if err := client.Get(ctx, pkgclient.ObjectKey{Namespace: seCR.Namespace, Name: seCR.Name}, seCR); err != nil {
		// any error here is bad as it should exist now
		return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("failed to get the webapp resource namespace %s name %s", seCR.Namespace, seCR.Name))
	}
	if seCR.Status.Message == "OK" {
		if r.Config.GetProductVersion() != v1alpha1.ProductVersion(seCR.Status.Version) {
			r.Config.SetProductVersion(seCR.Status.Version)
			r.ConfigManager.WriteConfig(r.Config)
		}
		return v1alpha1.PhaseCompleted, nil
	}
	return v1alpha1.PhaseInProgress, nil

}

func (r *Reconciler) getInstalledProducts(inst *v1alpha1.Installation) (string, error) {
	installedProducts := inst.Status.Stages["products"].Products

	// Ensure that amq online console is not added to the installed products, a per user amq online is used instead which is provisioned by the webapp
	// Ensure that ups is not added to the installed products
	products := make(map[v1alpha1.ProductName]productInfo)
	for name, info := range installedProducts {
		if name != v1alpha1.ProductAMQOnline && name != v1alpha1.ProductUps && info.Host != "" {
			id := r.getProductId(name)
			products[id] = productInfo{
				Host:    info.Host,
				Version: info.Version,
			}
		}
	}

	productsData, err := json.MarshalIndent(products, "", "  ")
	if err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal json data %s", products)
	}

	return string(productsData), nil
}

// Gets the product's id used by the webapp
// https://github.com/integr8ly/tutorial-web-app/blob/master/src/product-info.js
func (r *Reconciler) getProductId(name v1alpha1.ProductName) v1alpha1.ProductName {
	id := name

	if name == v1alpha1.ProductFuse {
		id = "fuse-managed"
	}

	if name == v1alpha1.ProductCodeReadyWorkspaces {
		id = "codeready"
	}

	if name == v1alpha1.ProductRHSSOUser {
		id = "user-rhsso"
	}

	return id
}

func (r *Reconciler) getOAuthClientName() string {
	return r.installation.Spec.NamespacePrefix + string(r.Config.GetProductName())
}
