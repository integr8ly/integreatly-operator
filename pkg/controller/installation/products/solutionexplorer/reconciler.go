package solutionexplorer

import (
	"context"
	"fmt"
	webapp "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/tutorial-web-app-operator/pkg/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	v1 "github.com/openshift/api/oauth/v1"
	v13 "github.com/openshift/api/route/v1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	defaultName          = "solution-explorer"
	defaultSubNameAndPkg = "solution-explorer"
	defaultTemplateLoc   = "/home/tutorial-web-app-operator/deploy/template/tutorial-web-app.yml"
	param_openshift_host = "OPENSHIFT_HOST"
	param_oauth_client   = "OPENSHIFT_OAUTHCLIENT_ID"
	param_sso_route      = "SSO_ROUTE"
	defaultRouteName     = "tutorial-web-app"
)

type Reconciler struct {
	*resources.Reconciler
	coreClient    kubernetes.Interface
	Config        *config.SolutionExplorer
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
	OauthResolver OauthResolver
}

//go:generate moq -out OauthResolver_moq.go . OauthResolver
type OauthResolver interface {
	GetOauthEndPoint() (*resources.OauthServerConfig, error)
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface, resolver OauthResolver) (*Reconciler, error) {
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
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileNamespace(ctx, r.Config.GetNamespace(), inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}
	phase, err = r.ReconcileSubscription(ctx, inst, defaultSubNameAndPkg, r.Config.GetNamespace(), serverClient)
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
	phase, err = r.ReconcileOauthClient(ctx, inst, &v1.OAuthClient{
		RedirectURIs: []string{route},
		Secret:       "test",
		GrantMethod:  v1.GrantHandlerAuto,
		ObjectMeta: v12.ObjectMeta{
			Name: defaultSubNameAndPkg,
		},
	}, serverClient)
	return phase, err
}

func (r *Reconciler) ensureAppUrl(ctx context.Context, client pkgclient.Client) (string, error) {
	route := &v13.Route{
		ObjectMeta: v12.ObjectMeta{
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
		ObjectMeta: v12.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      defaultName,
		},
	}
	ownerutil.AddOwner(seCR, inst, true, true)
	oauthURL := strings.Replace(strings.Replace(oauthConfig.AuthorizationEndpoint, "https://", "", 1), "/oauth/authorize", "", 1)
	logrus.Info("ReconcileCustomResource setting url for openshift host ", oauthURL)
	seCR.Spec = webapp.WebAppSpec{
		AppLabel: "tutorial-web-app",
		Template: webapp.WebAppTemplate{
			Path: defaultTemplateLoc,
			Parameters: map[string]string{
				param_oauth_client:   defaultSubNameAndPkg,
				param_sso_route:      ssoConfig.GetURL(),
				param_openshift_host: oauthURL,
			},
		},
	}
	if err := client.Create(ctx, seCR); err != nil && !errors2.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, errors.Wrap(err, " failed to create webapp")
	}
	// do a get to ensure we have an upto date copy
	if err := client.Get(ctx, pkgclient.ObjectKey{Namespace: seCR.Namespace, Name: seCR.Name}, seCR); err != nil {
		// any error here is bad as it should exist now
		return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("failed to get the webapp resource namespace %s name %s", seCR.Namespace, seCR.Name))
	}
	if seCR.Status.Message == "OK" {
		return v1alpha1.PhaseCompleted, nil
	}
	return v1alpha1.PhaseInProgress, nil

}
