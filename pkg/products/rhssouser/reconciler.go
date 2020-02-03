package rhssouser

import (
	"context"
	"fmt"
	"strings"

	"github.com/integr8ly/integreatly-operator/pkg/products/rhsso"
	"github.com/pkg/errors"

	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"

	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/sirupsen/logrus"

	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	appsv1 "github.com/openshift/api/apps/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	defaultRhssoNamespace   = "user-sso"
	keycloakName            = "rhssouser"
	keycloakRealmName       = "user-sso"
	defaultSubscriptionName = "integreatly-rhsso"
	idpAlias                = "openshift-v4"
	manifestPackage         = "integreatly-rhsso"
)

const (
	userSsoLabelKey   = "sso"
	userSsoLabelValue = "user"
)

type Reconciler struct {
	Config        *config.RHSSOUser
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	installation  *integreatlyv1alpha1.Installation
	logger        *logrus.Entry
	oauthv1Client oauthClient.OauthV1Interface
	ApiUrl        string
	*resources.Reconciler
	recorder record.EventRecorder
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.Installation, oauthv1Client oauthClient.OauthV1Interface, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, apiURL string) (*Reconciler, error) {
	rhssoUserConfig, err := configManager.ReadRHSSOUser()
	if err != nil {
		return nil, err
	}
	if rhssoUserConfig.GetNamespace() == "" {
		rhssoUserConfig.SetNamespace(installation.Spec.NamespacePrefix + defaultRhssoNamespace)
	}

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		Config:        rhssoUserConfig,
		ConfigManager: configManager,
		mpm:           mpm,
		installation:  installation,
		logger:        logger,
		oauthv1Client: oauthv1Client,
		Reconciler:    resources.NewReconciler(mpm),
		recorder:      recorder,
		ApiUrl:        apiURL,
	}, nil
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sso",
			Namespace: ns,
		},
	}
}

// Reconcile reads that state of the cluster for rhsso and makes changes based on the state read
// and what is required
func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.Installation, product *integreatlyv1alpha1.InstallationProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		// Check if namespace is still present before trying to delete it resources
		_, err := resources.GetNS(ctx, r.Config.GetNamespace(), serverClient)
		if !k8serr.IsNotFound(err) {
			phase, err := r.cleanupKeycloakResources(ctx, installation, serverClient)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}

			phase, err = r.isKeycloakResourcesDeleted(ctx, serverClient)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}

			phase, err = resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetNamespace())
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
		}

		_, err = resources.GetNS(ctx, r.Config.GetOperatorNamespace(), serverClient)
		if !k8serr.IsNotFound(err) {
			phase, err := resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetOperatorNamespace())
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
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

	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: defaultSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetOperatorNamespace(), ManifestPackage: manifestPackage}, r.Config.GetNamespace(), serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", defaultSubscriptionName), err)
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile components", err)
		return phase, err
	}

	phase, err = r.handleProgressPhase(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to handle in progress phase", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	r.logger.Infof("%s has reconciled successfully", r.Config.GetProductName())
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, installation *integreatlyv1alpha1.Installation, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.logger.Info("Reconciling Keycloak components")
	kc := &keycloak.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, kc, func() error {
		owner.AddIntegreatlyOwnerAnnotations(kc, installation)
		kc.Spec.Extensions = []string{
			"https://github.com/aerogear/keycloak-metrics-spi/releases/download/1.0.4/keycloak-metrics-spi-1.0.4.jar",
		}
		kc.Labels = getInstanceLabels()
		kc.Spec.Instances = 3
		kc.Spec.ExternalAccess = keycloak.KeycloakExternalAccess{Enabled: true}
		kc.Spec.Profile = rhsso.RHSSOProfile
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update keycloak custom resource: %w", err)
	}
	r.logger.Infof("The operation result for keycloak %s was %s", kc.Name, or)

	kcr := &keycloak.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakRealmName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	or, err = controllerutil.CreateOrUpdate(ctx, serverClient, kcr, func() error {
		kcr.Spec.RealmOverrides = []*keycloak.RedirectorIdentityProviderOverride{
			{
				IdentityProvider: idpAlias,
				ForFlow:          "browser",
			},
		}

		kcr.Spec.InstanceSelector = &metav1.LabelSelector{
			MatchLabels: getInstanceLabels(),
		}
		kcr.Labels = getInstanceLabels()
		kcr.Spec.Realm = &keycloak.KeycloakAPIRealm{
			ID:              keycloakRealmName,
			Realm:           keycloakRealmName,
			Enabled:         true,
			DisplayName:     keycloakRealmName,
			EventsListeners: []string{"metrics-listener"},
		}

		// The identity providers need to be set up before the realm CR gets
		// created because the Keycloak operator does not allow updates to
		// the realms
		err = r.setupOpenshiftIDP(ctx, installation, kcr, serverClient)
		if err != nil {
			return errors.Wrap(err, "failed to setup Openshift IDP for user-sso")
		}

		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update keycloak realm: %w", err)
	}
	r.logger.Infof("The operation result for keycloakrealm %s was %s", kcr.Name, or)

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) cleanupKeycloakResources(ctx context.Context, inst *integreatlyv1alpha1.Installation, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	if inst.DeletionTimestamp == nil {
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	opts := &k8sclient.ListOptions{
		Namespace: r.Config.GetNamespace(),
	}

	// Delete all users
	users := &keycloak.KeycloakUserList{}
	err := serverClient.List(ctx, users, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	for _, user := range users.Items {
		err = serverClient.Delete(ctx, &user)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	// Delete all clients
	clients := &keycloak.KeycloakClientList{}
	err = serverClient.List(ctx, clients, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	for _, client := range clients.Items {
		err = serverClient.Delete(ctx, &client)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	// Delete all realms
	realms := &keycloak.KeycloakRealmList{}
	err = serverClient.List(ctx, realms, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, nil
	}
	for _, realm := range realms.Items {
		err = serverClient.Delete(ctx, &realm)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) isKeycloakResourcesDeleted(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	opts := &k8sclient.ListOptions{
		Namespace: r.Config.GetNamespace(),
	}

	// Check if users are all gone
	users := &keycloak.KeycloakUserList{}
	err := serverClient.List(ctx, users, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if len(users.Items) > 0 {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	// Check if clients are all gone
	clients := &keycloak.KeycloakClientList{}
	err = serverClient.List(ctx, clients, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if len(clients.Items) > 0 {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	// Check if realms are all gone
	realms := &keycloak.KeycloakRealmList{}
	err = serverClient.List(ctx, realms, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, nil
	}
	if len(realms.Items) > 0 {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) handleProgressPhase(ctx context.Context, installation *integreatlyv1alpha1.Installation, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	kc := &keycloak.Keycloak{}
	// if this errors, it can be ignored
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: keycloakName, Namespace: r.Config.GetNamespace()}, kc)
	if err == nil && string(r.Config.GetProductVersion()) != kc.Status.Version {
		r.Config.SetProductVersion(kc.Status.Version)
		r.ConfigManager.WriteConfig(r.Config)
	}

	r.logger.Info("checking ready status for user-sso")
	kcr := &keycloak.KeycloakRealm{}

	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: keycloakRealmName, Namespace: r.Config.GetNamespace()}, kcr)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get keycloak realm custom resource: %w", err)
	}

	if kcr.Status.Phase == keycloak.PhaseReconciling {
		err = r.exportConfig(ctx, serverClient)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to write user-sso config: %w", err)
		}

		logrus.Infof("Keycloak has successfully processed the keycloakRealm for user-sso")
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	r.logger.Infof("user-sso KeycloakRealm status phase is: %s", kcr.Status.Phase)
	return integreatlyv1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) exportConfig(ctx context.Context, serverClient k8sclient.Client) error {
	kc := &keycloak.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: keycloakName, Namespace: r.Config.GetNamespace()}, kc)
	if err != nil {
		return fmt.Errorf("could not retrieve keycloak custom resource for keycloak config for user-sso: %w", err)
	}
	r.Config.SetRealm(keycloakRealmName)
	r.Config.SetHost(kc.Status.InternalURL)
	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return fmt.Errorf("could not update keycloak config for user-sso: %w", err)
	}
	return nil
}

func (r *Reconciler) setupOpenshiftIDP(ctx context.Context, installation *integreatlyv1alpha1.Installation, kcr *keycloak.KeycloakRealm, serverClient k8sclient.Client) error {
	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.ConfigManager.GetOauthClientsSecretName(),
		},
	}

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: oauthClientSecrets.Name, Namespace: r.ConfigManager.GetOperatorNamespace()}, oauthClientSecrets)
	if err != nil {
		return fmt.Errorf("Could not find %s Secret: %w", oauthClientSecrets.Name, err)
	}

	clientSecretBytes, ok := oauthClientSecrets.Data[string(r.Config.GetProductName())]
	if !ok {
		return fmt.Errorf("Could not find %s key in %s Secret", string(r.Config.GetProductName()), oauthClientSecrets.Name)
	}
	clientSecret := string(clientSecretBytes)

	oauthc := &oauthv1.OAuthClient{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.getOAuthClientName(),
		},
		Secret: clientSecret,
		RedirectURIs: []string{
			r.Config.GetHost() + "/auth/realms/user-sso/broker/openshift-v4/endpoint",
		},
		GrantMethod: oauthv1.GrantHandlerPrompt,
	}
	_, err = r.ReconcileOauthClient(ctx, installation, oauthc, serverClient)
	if err != nil {
		return fmt.Errorf("Could not create OauthClient object for OpenShift IDP: %w", err)
	}

	if !containsIdentityProvider(kcr.Spec.Realm.IdentityProviders, idpAlias) {
		logrus.Infof("Adding keycloak realm client")

		kcr.Spec.Realm.IdentityProviders = append(kcr.Spec.Realm.IdentityProviders, &keycloak.KeycloakIdentityProvider{
			Alias:                     idpAlias,
			ProviderID:                "openshift-v4",
			Enabled:                   true,
			TrustEmail:                true,
			StoreToken:                true,
			AddReadTokenRoleOnCreate:  true,
			FirstBrokerLoginFlowAlias: "first broker login",
			Config: map[string]string{
				"hideOnLoginPage": "",
				"baseUrl":         "https://" + strings.Replace(r.installation.Spec.RoutingSubdomain, "apps", "api", 1) + ":6443",
				"clientId":        r.getOAuthClientName(),
				"disableUserInfo": "",
				"clientSecret":    clientSecret,
				"defaultScope":    "user:full",
				"useJwksUrl":      "true",
			},
		})
	}
	return nil
}

func (r *Reconciler) getOAuthClientName() string {
	return r.installation.Spec.NamespacePrefix + string(r.Config.GetProductName())
}

func containsIdentityProvider(providers []*keycloak.KeycloakIdentityProvider, alias string) bool {
	for _, p := range providers {
		if p.Alias == alias {
			return true
		}
	}
	return false
}

func getInstanceLabels() map[string]string {
	return map[string]string{
		userSsoLabelKey: userSsoLabelValue,
	}
}
