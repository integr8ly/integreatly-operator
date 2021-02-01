package solutionexplorer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"

	"github.com/integr8ly/integreatly-operator/version"

	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"

	consolev1 "github.com/openshift/api/console/v1"

	"github.com/integr8ly/integreatly-operator/pkg/products/monitoring"

	monitoringv1alpha1 "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	solutionExplorerv1alpha1 "github.com/integr8ly/integreatly-operator/apis-products/tutorial-web-app-operator/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	appsv1 "github.com/openshift/api/apps/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	routev1 "github.com/openshift/api/route/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	DefaultName               = "solution-explorer"
	defaultTemplateLoc        = "/home/tutorial-web-app-operator/deploy/template/tutorial-web-app.yml"
	defaultWalkthroughsLoc    = "https://github.com/integr8ly/solution-patterns.git#v1.0.12"
	paramOpenShiftHost        = "OPENSHIFT_HOST"
	paramOpenShiftOauthHost   = "OPENSHIFT_OAUTH_HOST"
	paramOauthClient          = "OPENSHIFT_OAUTHCLIENT_ID"
	paramOpenShiftVersion     = "OPENSHIFT_VERSION"
	paramInstalledServices    = "INSTALLED_SERVICES"
	paramSSORoute             = "SSO_ROUTE"
	paramIntegreatlyVersion   = "INTEGREATLY_VERSION"
	paramClusterType          = "CLUSTER_TYPE"
	paramWalkthroughLocations = "WALKTHROUGH_LOCATIONS"
	defaultRouteName          = "solution-explorer"
	manifestPackage           = "integreatly-solution-explorer"
	paramRoutingSubdomain     = "ROUTING_SUBDOMAIN"
	paramInstallationType     = "INSTALLATION_TYPE"
	ParamUpgradeData          = "UPGRADE_DATA"
)

type Reconciler struct {
	*resources.Reconciler
	coreClient    kubernetes.Interface
	oauthv1Client oauthClient.OauthV1Interface
	Config        *config.SolutionExplorer
	ConfigManager config.ConfigReadWriter
	extraParams   map[string]string
	mpm           marketplace.MarketplaceInterface
	log           l.Logger
	OauthResolver OauthResolver
	installation  *integreatlyv1alpha1.RHMI
	recorder      record.EventRecorder
}

//go:generate moq -out OauthResolver_moq.go . OauthResolver
type OauthResolver interface {
	GetOauthEndPoint() (*resources.OauthServerConfig, error)
}

type productInfo struct {
	Host    string
	Version integreatlyv1alpha1.ProductVersion
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, oauthv1Client oauthClient.OauthV1Interface, mpm marketplace.MarketplaceInterface, resolver OauthResolver, recorder record.EventRecorder, logger l.Logger) (*Reconciler, error) {
	config, err := configManager.ReadSolutionExplorer()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve solution explorer config: %w", err)
	}

	if config.GetNamespace() == "" {
		config.SetNamespace(installation.Spec.NamespacePrefix + DefaultName)
	}
	if config.GetOperatorNamespace() == "" {
		if installation.Spec.OperatorsInProductNamespace {
			config.SetOperatorNamespace(config.GetNamespace())
		} else {
			config.SetOperatorNamespace(config.GetNamespace() + "-operator")
		}
	}
	if err = config.Validate(); err != nil {
		return nil, fmt.Errorf("solution explorer config is not valid: %w", err)
	}

	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
		mpm:           mpm,
		log:           logger,
		Reconciler:    resources.NewReconciler(mpm),
		OauthResolver: resolver,
		oauthv1Client: oauthv1Client,
		installation:  installation,
		recorder:      recorder,
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

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return version.VerifyProductAndOperatorVersion(
		installation.Status.Stages[integreatlyv1alpha1.SolutionExplorerStage].Products[integreatlyv1alpha1.ProductSolutionExplorer],
		string(integreatlyv1alpha1.VersionSolutionExplorer),
		string(integreatlyv1alpha1.OperatorVersionSolutionExplorer),
	)
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Reconciling solution explorer")

	operatorNamespace := r.Config.GetOperatorNamespace()
	productNamespace := r.Config.GetNamespace()

	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		phase, err := resources.RemoveNamespace(ctx, installation, serverClient, productNamespace, r.log)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}
		phase, err = resources.RemoveNamespace(ctx, installation, serverClient, operatorNamespace, r.log)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		cl := &consolev1.ConsoleLink{
			ObjectMeta: metav1.ObjectMeta{
				Name: "rhmi-solution-explorer",
			},
		}

		err = serverClient.Delete(ctx, cl)
		if err != nil && !k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		err = resources.RemoveOauthClient(r.oauthv1Client, r.getOAuthClientName(), r.log)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
		return integreatlyv1alpha1.PhaseCompleted, nil
	}, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, operatorNamespace, installation, serverClient, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", operatorNamespace), err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, productNamespace, installation, serverClient, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", productNamespace), err)
		return phase, err
	}

	phase, err = r.reconcileSubscription(ctx, serverClient, installation, productNamespace, operatorNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.SolutionExplorerSubscriptionName), err)
		return phase, err
	}

	phase, err = r.ReconcileCustomResource(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile custom resource", err)
		return phase, err
	}

	phase, err = r.addHSTSAnnotationToRoute(ctx, serverClient, installation)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to set HSTS header on Route", err)
		return phase, err
	}

	route, err := r.ensureAppURL(ctx, serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, "Route for solution explorer is not available", err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	if r.Config.GetHost() != route {
		r.Config.SetHost(route)
		r.ConfigManager.WriteConfig(r.Config)
	}

	phase, err = r.reconcileConsoleLink(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile console link", err)
		return phase, err
	}

	phase, err = r.ReconcileOauthClient(ctx, installation, &oauthv1.OAuthClient{
		RedirectURIs: []string{route},
		GrantMethod:  oauthv1.GrantHandlerAuto,
		ObjectMeta: metav1.ObjectMeta{
			Name: r.getOAuthClientName(),
		},
	}, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile oauth client", err)
		return phase, err
	}

	phase, err = r.reconcileBlackboxTarget(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile blackbox targets", err)
		return phase, err
	}

	phase, err = r.newAlertReconciler(r.log, r.installation.Spec.Type).ReconcileAlerts(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile solution explorer alerts", err)
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.SolutionExplorerStage, r.Config.GetProductName())
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileConsoleLink(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	cl := &consolev1.ConsoleLink{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rhmi-solution-explorer",
		},
		Spec: consolev1.ConsoleLinkSpec{
			ApplicationMenu: &consolev1.ApplicationMenuSpec{},
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, cl, func() error {
		cl.Spec.ApplicationMenu.ImageURL = "https://github.com/integr8ly/integreatly-operator/raw/master/assets/icons/Product_Icon-Red_Hat-Managed_Integration_Solution_Explorer-RGB.png"
		cl.Spec.ApplicationMenu.Section = "Red Hat Applications"
		cl.Spec.Href = r.Config.GetHost()
		cl.Spec.Location = consolev1.ApplicationMenu
		cl.Spec.Text = "Solution Explorer"
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating or updating solution explorer console link, %s", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

// CreateResource Creates a generic kubernetes resource from a template
func (r *Reconciler) createResource(ctx context.Context, resourceName string, serverClient k8sclient.Client) (runtime.Object, error) {
	if r.extraParams == nil {
		r.extraParams = map[string]string{}
	}
	r.extraParams["MonitoringKey"] = r.Config.GetLabelSelector()
	r.extraParams["Namespace"] = r.Config.GetNamespace()

	templateHelper := monitoring.NewTemplateHelper(r.extraParams)
	resource, err := templateHelper.CreateResource(resourceName)

	if err != nil {
		return nil, fmt.Errorf("createResource failed: %w", err)
	}

	err = serverClient.Create(ctx, resource)
	if err != nil {
		if !k8serr.IsAlreadyExists(err) {
			return nil, fmt.Errorf("error creating resource: %w", err)
		}
	}

	return resource, nil
}

func (r *Reconciler) reconcileBlackboxTarget(ctx context.Context, installation *integreatlyv1alpha1.RHMI, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	cfg, err := r.ConfigManager.ReadMonitoring()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error reading monitoring config: %w", err)
	}

	target := monitoringv1alpha1.BlackboxtargetData{
		Url:     r.Config.GetHost(),
		Service: "webapp-ui",
	}

	err = monitoring.CreateBlackboxTarget(ctx, "integreatly-webapp", target, cfg, installation, client)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating solution explorer blackbox target: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) addHSTSAnnotationToRoute(ctx context.Context, serverClient k8sclient.Client, inst *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {
	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultRouteName,
			Namespace: r.Config.GetNamespace(),
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, route, func() error {
		annotations := route.ObjectMeta.GetAnnotations()
		if annotations == nil {
			annotations = map[string]string{}
		}
		annotations["haproxy.router.openshift.io/hsts_header"] = "max-age=31536000;includeSubDomains;preload"
		route.ObjectMeta.SetAnnotations(annotations)

		if route.Spec.TLS != nil {
			route.Spec.TLS.InsecureEdgeTerminationPolicy = routev1.InsecureEdgeTerminationPolicyRedirect
		}

		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) ensureAppURL(ctx context.Context, client k8sclient.Client) (string, error) {
	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      defaultRouteName,
		},
	}
	if err := client.Get(ctx, k8sclient.ObjectKey{Name: route.Name, Namespace: route.Namespace}, route); err != nil {
		return "", fmt.Errorf("failed to get route for solution explorer: %w", err)
	}
	protocol := "https"
	if route.Spec.TLS == nil {
		protocol = "http"
	}

	return fmt.Sprintf("%s://%s", protocol, route.Spec.Host), nil
}

func (r *Reconciler) ReconcileCustomResource(ctx context.Context, installation *integreatlyv1alpha1.RHMI, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	//todo shouldn't need to do this with each reconcile
	oauthConfig, err := r.OauthResolver.GetOauthEndPoint()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get oauth details: %w", err)
	}
	ssoConfig, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	seCR := &solutionExplorerv1alpha1.WebApp{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      DefaultName,
		},
	}
	oauthURL := strings.Replace(strings.Replace(oauthConfig.AuthorizationEndpoint, "https://", "", 1), "/oauth/authorize", "", 1)
	r.log.Infof("ReconcileCustomResource setting url for openshift host ", l.Fields{"oauthURL": oauthURL})

	installedServices, err := r.getInstalledProducts(installation)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to retrieve installed products information from %s CR: %w", installation.Name, err)
	}
	_, err = controllerutil.CreateOrUpdate(ctx, client, seCR, func() error {
		owner.AddIntegreatlyOwnerAnnotations(seCR, installation)
		seCR.Spec.AppLabel = "tutorial-web-app"
		seCR.Spec.Template.Path = defaultTemplateLoc
		parameters := map[string]string{
			paramOauthClient:          r.getOAuthClientName(),
			paramSSORoute:             ssoConfig.GetHost(),
			paramOpenShiftHost:        installation.Spec.MasterURL,
			paramOpenShiftOauthHost:   oauthURL,
			paramOpenShiftVersion:     "4",
			paramClusterType:          "osd",
			paramInstalledServices:    installedServices,
			paramIntegreatlyVersion:   version.GetVersion(),
			paramWalkthroughLocations: defaultWalkthroughsLoc,
			paramRoutingSubdomain:     installation.Spec.RoutingSubdomain,
			paramInstallationType:     installation.Spec.Type,
		}
		if seCR.Spec.Template.Parameters == nil {
			seCR.Spec.Template.Parameters = map[string]string{}
		}
		for k, v := range parameters {
			seCR.Spec.Template.Parameters[k] = v
		}
		// If the upgrade data parameter is not found, set it to `null`
		if _, ok := seCR.Spec.Template.Parameters[ParamUpgradeData]; !ok {
			seCR.Spec.Template.Parameters[ParamUpgradeData] = "null"
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile webapp resource: %w", err)
	}
	// do a get to ensure we have an upto date copy
	if err := client.Get(ctx, k8sclient.ObjectKey{Namespace: seCR.Namespace, Name: seCR.Name}, seCR); err != nil {
		// any error here is bad as it should exist now
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get the webapp resource namespace %s name %s: %w", seCR.Namespace, seCR.Name, err)
	}
	if seCR.Status.Message == "OK" {
		if r.Config.GetProductVersion() != integreatlyv1alpha1.ProductVersion(seCR.Status.Version) {
			r.Config.SetProductVersion(seCR.Status.Version)
			r.ConfigManager.WriteConfig(r.Config)
		}
		return integreatlyv1alpha1.PhaseCompleted, nil
	}
	return integreatlyv1alpha1.PhaseInProgress, nil

}

func (r *Reconciler) getInstalledProducts(installation *integreatlyv1alpha1.RHMI) (string, error) {
	installedProducts := installation.Status.Stages["products"].Products

	// Ensure that amq online console is not added to the installed products, a per user amq online is used instead which is provisioned by the webapp
	// Ensure that ups is not added to the installed products
	products := make(map[integreatlyv1alpha1.ProductName]productInfo)
	for name, info := range installedProducts {
		if info.Host != "" {
			id := r.getProductID(name)
			products[id] = productInfo{
				Host:    info.Host,
				Version: info.Version,
			}
		}
	}

	productsData, err := json.MarshalIndent(products, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal json data %s: %w", products, err)
	}

	return string(productsData), nil
}

// Gets the product's id used by the webapp
// https://github.com/integr8ly/tutorial-web-app/blob/master/src/product-info.js
func (r *Reconciler) getProductID(name integreatlyv1alpha1.ProductName) integreatlyv1alpha1.ProductName {
	id := name

	if name == integreatlyv1alpha1.ProductFuse {
		id = "fuse-managed"
	}

	if name == integreatlyv1alpha1.ProductCodeReadyWorkspaces {
		id = "codeready"
	}

	if name == integreatlyv1alpha1.ProductRHSSOUser {
		id = "user-rhsso"
	}

	return id
}

func (r *Reconciler) getOAuthClientName() string {
	return r.installation.Spec.NamespacePrefix + string(r.Config.GetProductName())
}

func (r *Reconciler) reconcileSubscription(ctx context.Context, serverClient k8sclient.Client, inst *integreatlyv1alpha1.RHMI, productNamespace string, operatorNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	target := marketplace.Target{
		Pkg:       constants.SolutionExplorerSubscriptionName,
		Namespace: operatorNamespace,
		Channel:   marketplace.IntegreatlyChannel,
	}
	catalogSourceReconciler := marketplace.NewConfigMapCatalogSourceReconciler(
		manifestPackage,
		serverClient,
		operatorNamespace,
		marketplace.CatalogSourceName,
	)
	return r.Reconciler.ReconcileSubscription(
		ctx,
		target,
		[]string{productNamespace},
		backup.NewNoopBackupExecutor(),
		serverClient,
		catalogSourceReconciler,
		r.log,
	)
}
