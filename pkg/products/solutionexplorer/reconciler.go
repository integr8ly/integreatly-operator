package solutionexplorer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"

	"github.com/sirupsen/logrus"

	"github.com/integr8ly/integreatly-operator/pkg/products/monitoring"

	webapp "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/tutorial-web-app-operator/pkg/apis/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	monitoringv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/monitoring/v1alpha1"
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
	manifestPackage         = "integreatly-solution-explorer"
)

type Reconciler struct {
	*resources.Reconciler
	coreClient    kubernetes.Interface
	oauthv1Client oauthClient.OauthV1Interface
	Config        *config.SolutionExplorer
	ConfigManager config.ConfigReadWriter
	extraParams   map[string]string
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
	OauthResolver OauthResolver
	installation  *integreatlyv1alpha1.Installation
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

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.Installation, oauthv1Client oauthClient.OauthV1Interface, mpm marketplace.MarketplaceInterface, resolver OauthResolver, recorder record.EventRecorder) (*Reconciler, error) {
	seConfig, err := configManager.ReadSolutionExplorer()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve solution explorer config: %w", err)
	}

	if seConfig.GetNamespace() == "" {
		seConfig.SetNamespace(installation.Spec.NamespacePrefix + defaultName)
	}
	if err = seConfig.Validate(); err != nil {
		return nil, fmt.Errorf("solution explorer config is not valid: %w", err)
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

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.Installation, product *integreatlyv1alpha1.InstallationProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	logrus.Info("Reconciling solution explorer")

	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		phase, err := resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetNamespace())
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}
		phase, err = resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetOperatorNamespace())
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		err = resources.RemoveOauthClient(ctx, installation, serverClient, r.oauthv1Client, r.getOAuthClientName())
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
		return integreatlyv1alpha1.PhaseCompleted, nil
	})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetOperatorNamespace(), installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetOperatorNamespace()), err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetNamespace(), installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetNamespace()), err)
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, r.Config.GetNamespace(), serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, fmt.Sprintf("Failed to retrieve %s namespace", r.Config.GetNamespace()), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: defaultSubNameAndPkg, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetOperatorNamespace(), ManifestPackage: manifestPackage}, r.Config.GetNamespace(), serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", defaultSubNameAndPkg), err)
		return phase, err
	}

	phase, err = r.ReconcileCustomResource(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile custom resource", err)
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

	phase, err = r.reconcileTemplates(ctx, installation, serverClient)
	logrus.Infof("Phase: %s reconcileTemplates", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile templates", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.SolutionExplorerStage, r.Config.GetProductName())
	return integreatlyv1alpha1.PhaseCompleted, nil
}

// CreateResource Creates a generic kubernetes resource from a template
func (r *Reconciler) createResource(ctx context.Context, installation *integreatlyv1alpha1.Installation, resourceName string, serverClient k8sclient.Client) (runtime.Object, error) {
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

func (r *Reconciler) reconcileTemplates(ctx context.Context, installation *integreatlyv1alpha1.Installation, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// Interate over template_list
	for _, template := range r.Config.GetTemplateList() {
		// create it
		_, err := r.createResource(ctx, installation, template, serverClient)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update monitoring template %s: %w", template, err)
		}
		logrus.Infof("Reconciling the monitoring template %s was successful", template)
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileBlackboxTarget(ctx context.Context, installation *integreatlyv1alpha1.Installation, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
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

func (r *Reconciler) ReconcileCustomResource(ctx context.Context, installation *integreatlyv1alpha1.Installation, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	//todo shouldn't need to do this with each reconcile
	oauthConfig, err := r.OauthResolver.GetOauthEndPoint()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get oauth details: %w", err)
	}
	ssoConfig, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	seCR := &webapp.WebApp{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      defaultName,
		},
	}
	oauthURL := strings.Replace(strings.Replace(oauthConfig.AuthorizationEndpoint, "https://", "", 1), "/oauth/authorize", "", 1)
	logrus.Info("ReconcileCustomResource setting url for openshift host ", oauthURL)

	installedServices, err := r.getInstalledProducts(installation)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to retrieve installed products information from %s CR: %w", installation.Name, err)
	}
	_, err = controllerutil.CreateOrUpdate(ctx, client, seCR, func() error {
		owner.AddIntegreatlyOwnerAnnotations(seCR, installation)
		seCR.Spec.AppLabel = "tutorial-web-app"
		seCR.Spec.Template.Path = defaultTemplateLoc
		seCR.Spec.Template.Parameters = map[string]string{
			paramOauthClient:        r.getOAuthClientName(),
			paramSSORoute:           ssoConfig.GetHost(),
			paramOpenShiftHost:      installation.Spec.MasterURL,
			paramOpenShiftOauthHost: oauthURL,
			paramOpenShiftVersion:   "4",
			paramInstalledServices:  installedServices,
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

func (r *Reconciler) getInstalledProducts(installation *integreatlyv1alpha1.Installation) (string, error) {
	installedProducts := installation.Status.Stages["products"].Products

	// Ensure that amq online console is not added to the installed products, a per user amq online is used instead which is provisioned by the webapp
	// Ensure that ups is not added to the installed products
	products := make(map[integreatlyv1alpha1.ProductName]productInfo)
	for name, info := range installedProducts {
		if name != integreatlyv1alpha1.ProductAMQOnline && name != integreatlyv1alpha1.ProductUps && info.Host != "" {
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
