package launcher

import (
	"context"
	"github.com/RHsyseng/operator-utils/pkg/olm"
	launcherv1alpha2 "github.com/fabric8-launcher/launcher-operator/pkg/apis/launcher/v1alpha2"
	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	appsv1 "github.com/openshift/api/apps/v1"
	routev1 "github.com/openshift/api/route/v1"
	appsv1Client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	"github.com/pkg/errors"
	pkgerr "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace         = "launcher"
	defaultSubscriptionName              = "integreatly-launcher"
	defaultLauncherDeployementConfigName = "launcher-application"
	defaultLauncherName                  = "launcher"
	defaultLauncherConfigMapName         = "launcher"
	launcherRouteName                    = "launcher"
	clientId                             = "launcher"
)

type Reconciler struct {
	*resources.Reconciler
	installation  *v1alpha1.Installation
	appsv1Client  appsv1Client.AppsV1Interface
	Config        *config.Launcher
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, appsv1Client appsv1Client.AppsV1Interface, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	config, err := configManager.ReadLauncher()
	if err != nil {
		return nil, err
	}

	if config.GetNamespace() == "" {
		config.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}

	return &Reconciler{
		appsv1Client:  appsv1Client,
		ConfigManager: configManager,
		Config:        config,
		mpm:           mpm,
		installation:  instance,
		Reconciler:    resources.NewReconciler(mpm),
	}, nil
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultLauncherDeployementConfigName,
			Namespace: ns,
		},
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, product *v1alpha1.InstallationProductStatus, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Infof("Reconciling %s", defaultLauncherName)

	phase, err := r.ReconcileFinalizer(ctx, serverClient, inst, string(r.Config.GetProductName()), func() (v1alpha1.StatusPhase, error) {
		phase, err := resources.RemoveNamespace(ctx, inst, serverClient, r.Config.GetNamespace())
		if err != nil || phase != v1alpha1.PhaseCompleted {
			return phase, err
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
	version, err := resources.NewVersion(v1alpha1.OperatorVersionLauncher)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "invalid version number for launcher")
	}
	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Namespace: r.Config.GetNamespace(), Channel: marketplace.IntegreatlyChannel, Pkg: defaultSubscriptionName}, serverClient, version)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileLauncher(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.handleProgressPhase(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileRHSSOIntegration(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()

	logrus.Infof("%s is successfully reconciled", defaultLauncherName)
	return phase, err
}

func (r *Reconciler) reconcileLauncher(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	// Reconcile Launcher custom resource
	rhssoConfig, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	cr := &launcherv1alpha2.Launcher{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultLauncherName,
			Namespace: r.Config.GetNamespace(),
		},
		Spec: launcherv1alpha2.LauncherSpec{
			OpenShift: launcherv1alpha2.OpenShiftConfig{
				ConsoleURL: r.installation.Spec.MasterURL,
				Clusters: []launcherv1alpha2.OpenShiftClusterConfig{
					{
						ID:         "openshift-v4",
						Name:       "Local Openshift Cluster",
						ApiURL:     "https://openshift.default.svc.cluster.local",
						ConsoleURL: r.installation.Spec.MasterURL,
						Type:       "local",
					},
				},
			},
			OAuth: launcherv1alpha2.OAuthConfig{
				Enabled:          true,
				KeycloakURL:      rhssoConfig.GetHost() + "/auth",
				KeycloakRealm:    rhssoConfig.GetRealm(),
				KeycloakClientID: clientId,
			},
			Catalog: launcherv1alpha2.CatalogConfig{
				RepositoryURL: "https://github.com/integr8ly/launcher-booster-catalog",
				RepositoryRef: "master",
				Filter:        "booster.version.id.indexOf('redhat')>=0",
			},
		},
	}

	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: cr.Name, Namespace: cr.Namespace}, cr)
	if err != nil && !k8serr.IsNotFound(err) {
		return v1alpha1.PhaseFailed, err
	}
	if err != nil {
		if err := serverClient.Create(ctx, cr); err != nil {
			return v1alpha1.PhaseFailed, pkgerr.Wrap(err, "failed to create launcher custom resource during reconcile")
		}
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) handleProgressPhase(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	// Check installation status
	// No status is available in the Launcher custom resource, will need to check deploymentconfigs to ensure installation is ready
	launcherDcs, err := r.appsv1Client.DeploymentConfigs(r.Config.GetNamespace()).List(metav1.ListOptions{LabelSelector: "app=fabric8-launcher"})
	if err != nil {
		return v1alpha1.PhaseFailed, pkgerr.Wrap(err, "failed to list deployment configs")
	}
	dcStatus := olm.GetDeploymentConfigStatus(launcherDcs.Items)
	if len(dcStatus.Starting) == 0 && len(dcStatus.Stopped) == 0 && len(dcStatus.Ready) > 0 {
		// Set Launcher route if not available
		if r.Config.GetHost() == "" {
			launcherRoute := &routev1.Route{}
			err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: launcherRouteName, Namespace: r.Config.GetNamespace()}, launcherRoute)
			if err != nil {
				return v1alpha1.PhaseFailed, pkgerr.Wrapf(err, "failed to get route for launcher")
			}

			r.Config.SetHost("https://" + launcherRoute.Spec.Host)
			err = r.ConfigManager.WriteConfig(r.Config)
			if err != nil {
				return v1alpha1.PhaseFailed, pkgerr.Wrap(err, "could not update launcher config")
			}
		}

		logrus.Infof("%s application is ready", defaultLauncherName)
		return v1alpha1.PhaseCompleted, nil
	}

	return v1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) reconcileRHSSOIntegration(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	// Create Keycloak client for launcher
	launcherUrl := r.Config.GetHost()

	rhssoConfig, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	rhssoNamespace := rhssoConfig.GetNamespace()
	rhssoRealm := rhssoConfig.GetRealm()
	if rhssoNamespace == "" || rhssoRealm == "" {
		logrus.Info("Cannot configure SSO integration without SSO namespace and SSO realm")
		return v1alpha1.PhaseInProgress, nil
	}

	kcr := &aerogearv1.KeycloakRealm{}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: rhssoRealm, Namespace: rhssoNamespace}, kcr)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	if !aerogearv1.ContainsClient(kcr.Spec.Clients, clientId) {
		logrus.Infof("Adding keycloak realm client")
		kcr.Spec.Clients = append(kcr.Spec.Clients, &aerogearv1.KeycloakClient{
			KeycloakApiClient: &aerogearv1.KeycloakApiClient{
				ID:                      clientId,
				ClientID:                clientId,
				ClientAuthenticatorType: "client-secret",
				Enabled:                 true,
				PublicClient:            true,
				RedirectUris: []string{
					launcherUrl,
					launcherUrl + "/*",
				},
				WebOrigins: []string{
					launcherUrl,
					launcherUrl + "/*",
				},
				StandardFlowEnabled: true,
				RootURL:             launcherUrl,
				FullScopeAllowed:    true,
				Access: map[string]bool{
					"view":      true,
					"configure": true,
					"manage":    true,
				},
				ProtocolMappers: []aerogearv1.KeycloakProtocolMapper{
					{
						Name:            "given name",
						Protocol:        "openid-connect",
						ProtocolMapper:  "oidc-usermodel-property-mapper",
						ConsentRequired: true,
						ConsentText:     "${givenName}",
						Config: map[string]string{
							"userinfo.token.claim": "true",
							"user.attribute":       "firstName",
							"id.token.claim":       "true",
							"access.token.claim":   "true",
							"claim.name":           "given_name",
							"jsonType.label":       "String",
						},
					},
					{
						Name:            "full name",
						Protocol:        "openid-connect",
						ProtocolMapper:  "oidc-full-name-mapper",
						ConsentRequired: true,
						ConsentText:     "${fullName}",
						Config: map[string]string{
							"id.token.claim":       "true",
							"access.token.claim":   "true",
							"userinfo.token.claim": "true",
						},
					},
					{
						Name:            "family name",
						Protocol:        "openid-connect",
						ProtocolMapper:  "oidc-usermodel-property-mapper",
						ConsentRequired: true,
						ConsentText:     "${familyName}",
						Config: map[string]string{
							"userinfo.token.claim": "true",
							"user.attribute":       "lastName",
							"id.token.claim":       "true",
							"access.token.claim":   "true",
							"claim.name":           "family_name",
							"jsonType.label":       "String",
						},
					},
					{
						Name:            "role list",
						Protocol:        "saml",
						ProtocolMapper:  "saml-role-list-mapper",
						ConsentRequired: false,
						ConsentText:     "${familyName}",
						Config: map[string]string{
							"single":               "false",
							"attribute.nameformat": "Basic",
							"attribute.name":       "Role",
						},
					},
					{
						Name:            "email",
						Protocol:        "openid-connect",
						ProtocolMapper:  "oidc-usermodel-property-mapper",
						ConsentRequired: true,
						ConsentText:     "${email}",
						Config: map[string]string{
							"userinfo.token.claim": "true",
							"user.attribute":       "email",
							"id.token.claim":       "true",
							"access.token.claim":   "true",
							"claim.name":           "email",
							"jsonType.label":       "String",
						},
					},
					{
						Name:            "username",
						Protocol:        "openid-connect",
						ProtocolMapper:  "oidc-usermodel-property-mapper",
						ConsentRequired: false,
						Config: map[string]string{
							"userinfo.token.claim": "true",
							"user.attribute":       "username",
							"id.token.claim":       "true",
							"access.token.claim":   "true",
							"claim.name":           "preferred_username",
							"jsonType.label":       "String",
						},
					},
				},
			},
		})

		err = serverClient.Update(ctx, kcr)
		if err != nil {
			return v1alpha1.PhaseFailed, err
		}
	}

	return v1alpha1.PhaseCompleted, nil
}
