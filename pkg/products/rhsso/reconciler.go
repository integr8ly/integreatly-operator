package rhsso

import (
	"context"
	"fmt"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strings"

	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	oauthv1 "github.com/openshift/api/oauth/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/sirupsen/logrus"

	"github.com/integr8ly/integreatly-operator/pkg/products/monitoring"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	monitoringv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"

	appsv1 "github.com/openshift/api/apps/v1"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	defaultRhssoNamespace               = "rhsso"
	keycloakName                        = "rhsso"
	keycloakRealmName                   = "openshift"
	defaultSubscriptionName             = "integreatly-rhsso"
	idpAlias                            = "openshift-v4"
	githubIdpAlias                      = "github"
	githubOauthAppCredentialsSecretName = "github-oauth-secret"
	manifestPackage                     = "integreatly-rhsso"
)

const (
	SSOLabelKey   = "sso"
	SSOLabelValue = "integreatly"
	RHSSOProfile  = "RHSSO"
)

type Reconciler struct {
	Config        *config.RHSSO
	ConfigManager config.ConfigReadWriter
	extraParams   map[string]string
	mpm           marketplace.MarketplaceInterface
	installation  *integreatlyv1alpha1.Installation
	logger        *logrus.Entry
	oauthv1Client oauthClient.OauthV1Interface
	KeycloakHost  string
	ApiUrl        string
	*resources.Reconciler
	recorder record.EventRecorder
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.Installation, oauthv1Client oauthClient.OauthV1Interface, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, apiUrl string) (*Reconciler, error) {
	rhssoConfig, err := configManager.ReadRHSSO()
	if err != nil {
		return nil, err
	}
	if rhssoConfig.GetNamespace() == "" {
		rhssoConfig.SetNamespace(installation.Spec.NamespacePrefix + defaultRhssoNamespace)
	}

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		Config:        rhssoConfig,
		ConfigManager: configManager,
		mpm:           mpm,
		installation:  installation,
		logger:        logger,
		oauthv1Client: oauthv1Client,
		Reconciler:    resources.NewReconciler(mpm),
		recorder:      recorder,
		ApiUrl:        apiUrl,
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
	ns := r.Config.GetNamespace()

	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		// Check if namespace is still present before trying to delete it resources
		_, err := resources.GetNS(ctx, ns, serverClient)
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

	phase, err = r.ReconcileNamespace(ctx, ns, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", ns), err)
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, ns, serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, fmt.Sprintf("Failed to retrieve %s namespace", ns), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: defaultSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetNamespace(), ManifestPackage: manifestPackage}, r.Config.GetNamespace(), serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", defaultSubscriptionName), err)
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile components", err)
		return phase, err
	}

	phase, err = r.createKeycloakRoute(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to handle in progress phase", err)
		return phase, err
	}

	phase, err = r.reconcileTemplates(ctx, installation, serverClient)
	logrus.Infof("Phase: %s reconcileTemplates", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile templates", err)
		return phase, err
	}

	phase, err = r.handleProgressPhase(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileBlackboxTargets(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile blackbox targets", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.AuthenticationStage, r.Config.GetProductName())
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

// CreateResource Creates a generic kubernetes resource from a template
func (r *Reconciler) createResource(ctx context.Context, installation *integreatlyv1alpha1.Installation, resourceName string, serverClient k8sclient.Client) (runtime.Object, error) {
	if r.extraParams == nil {
		r.extraParams = map[string]string{}
	}
	r.extraParams = map[string]string{}
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

// workaround: the keycloak operator creates a route with TLS passthrough config
// this should use the same valid certs as the cluster itself but for some reason the
// signing operator gives out self signed certs
// to circumvent this we create another keycloak route with edge termination
func (r *Reconciler) createKeycloakRoute(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// We need to create a workaround service to allow accessing keycloak on
	// the http port
	httpService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak-http",
			Namespace: r.Config.GetNamespace(),
		},
	}

	// We need a route with edge termination to serve the correct cluster certificate
	edgeRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak-edge",
			Namespace: r.Config.GetNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, httpService, func() error {
		clusterIp := httpService.Spec.ClusterIP
		httpService.Annotations = map[string]string{
			"service.alpha.openshift.io/serving-cert-secret-name": "sso-x509-https-secret",
		}
		httpService.Spec = corev1.ServiceSpec{
			ClusterIP: clusterIp,
			Ports: []corev1.ServicePort{
				{
					Name:       "keycloak",
					Protocol:   corev1.ProtocolTCP,
					Port:       8443,
					TargetPort: intstr.FromInt(8443),
				},
			},
			Selector: map[string]string{
				"app":       "keycloak",
				"component": "keycloak",
			},
			Type: corev1.ServiceTypeClusterIP,
		}
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "error creating keycloak http service")
	}
	r.logger.Info(fmt.Sprintf("operation result of creating %v service was %v", httpService.Name, or))

	or, err = controllerutil.CreateOrUpdate(ctx, serverClient, edgeRoute, func() error {
		host := edgeRoute.Spec.Host
		edgeRoute.Spec = routev1.RouteSpec{
			Host: host,
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: "keycloak-http",
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("keycloak"),
			},
			TLS: &routev1.TLSConfig{
				Termination: routev1.TLSTerminationReencrypt,
			},
			WildcardPolicy: routev1.WildcardPolicyNone,
		}
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "error creating keycloak http service")
	}
	r.logger.Info(fmt.Sprintf("operation result of creating %v service was %v", edgeRoute.Name, or))

	if edgeRoute.Spec.Host == "" {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	// Override the keycloak host to the host of the edge route (instead of the
	// operator generated route)
	r.KeycloakHost = fmt.Sprintf("https://%v", edgeRoute.Spec.Host)

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
		ownerutil.EnsureOwner(kc, installation)
		kc.Spec.Extensions = []string{
			"https://github.com/aerogear/keycloak-metrics-spi/releases/download/1.0.4/keycloak-metrics-spi-1.0.4.jar",
		}
		kc.Labels = GetInstanceLabels()
		kc.Spec.Instances = 1
		kc.Spec.ExternalAccess = keycloak.KeycloakExternalAccess{
			Enabled: true,
		}
		kc.Spec.Profile = RHSSOProfile
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "failed to create/update keycloak custom resource")
	}
	r.logger.Infof("The operation result for keycloak %s was %s", kc.Name, or)
	kcr := &keycloak.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakRealmName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	or, err = controllerutil.CreateOrUpdate(ctx, serverClient, kcr, func() error {
		ownerutil.EnsureOwner(kcr, installation)
		kcr.Spec.RealmOverrides = []*keycloak.RedirectorIdentityProviderOverride{
			{
				IdentityProvider: idpAlias,
				ForFlow:          "browser",
			},
		}

		kcr.Spec.InstanceSelector = &metav1.LabelSelector{
			MatchLabels: GetInstanceLabels(),
		}

		// The labels are needed so that created users can identify their realm
		// with a selector
		kcr.Labels = GetInstanceLabels()

		kcr.Spec.Realm = &keycloak.KeycloakAPIRealm{
			ID:          keycloakRealmName,
			Realm:       keycloakRealmName,
			Enabled:     true,
			DisplayName: keycloakRealmName,
			EventsListeners: []string{
				"metrics-listener",
			},
		}

		// The identity providers need to be set up before the realm CR gets
		// created because the Keycloak operator does not allow updates to
		// the realms
		err = r.setupOpenshiftIDP(ctx, installation, kcr, serverClient)
		if err != nil {
			return errors.Wrap(err, "failed to setup Openshift IDP")
		}

		err = r.setupGithubIDP(ctx, kcr, serverClient)
		if err != nil {
			return errors.Wrap(err, "failed to setup Github IDP")
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update keycloak realm: %w", err)
	}
	r.logger.Infof("The operation result for keycloakrealm %s was %s", kcr.Name, or)

	// Get all currently existing keycloak users
	keycloakUsers, err := GetKeycloakUsers(ctx, serverClient, r.Config.GetNamespace())
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "failed to list the keycloak users")
	}

	// Sync keycloak with openshift users
	users, err := syncronizeWithOpenshiftUsers(keycloakUsers, ctx, serverClient)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "failed to synchronize the users")
	}

	// Create / update the synchronized users
	for _, user := range users {
		or, err = r.createOrUpdateKeycloakUser(user, installation, ctx, serverClient)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "failed to create/update the customer admin user")
		} else {
			r.logger.Infof("The operation result for keycloakuser %s was %s", user.UserName, or)
		}
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) handleProgressPhase(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	kc := &keycloak.Keycloak{}
	// if this errors, it can be ignored
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: keycloakName, Namespace: r.Config.GetNamespace()}, kc)
	if err == nil && string(r.Config.GetProductVersion()) != kc.Status.Version {
		r.Config.SetProductVersion(kc.Status.Version)
		r.ConfigManager.WriteConfig(r.Config)
	}
	// The Keycloak Operator doesn't currently set the operator version
	r.Config.SetOperatorVersion(string(integreatlyv1alpha1.OperatorVersionRHSSO))
	r.ConfigManager.WriteConfig(r.Config)

	r.logger.Info("checking ready status for rhsso")
	kcr := &keycloak.KeycloakRealm{}

	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: keycloakRealmName, Namespace: r.Config.GetNamespace()}, kcr)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get keycloak realm custom resource: %w", err)
	}

	if kcr.Status.Phase == keycloak.PhaseReconciling {
		err = r.exportConfig(ctx, serverClient)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "failed to write rhsso config")
		}

		logrus.Infof("Keycloak has successfully processed the keycloakRealm")
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	r.logger.Infof("KeycloakRealm status phase is: %s", kcr.Status.Phase)
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
		return fmt.Errorf("could not retrieve keycloak custom resource for keycloak config: %w", err)
	}

	r.Config.SetRealm(keycloakRealmName)

	// TODO: once the keycloak operator generates a route with a valid certificate, that
	// should be reverted back to using the InternalURL
	// r.Config.SetHost(kc.Status.InternalURL)
	r.Config.SetHost(r.KeycloakHost)

	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return errors.Wrap(err, "could not update keycloak config")
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
		return errors.Wrapf(err, "Could not find %s Secret", oauthClientSecrets.Name)
	}

	clientSecretBytes, ok := oauthClientSecrets.Data[string(r.Config.GetProductName())]
	if !ok {
		return errors.Wrapf(err, "Could not find %s key in %s Secret", string(r.Config.GetProductName()), oauthClientSecrets.Name)
	}
	clientSecret := string(clientSecretBytes)

	redirectUris := []string{
		r.Config.GetHost() + "/auth/realms/openshift/broker/openshift-v4/endpoint",
	}

	oauthClient := &oauthv1.OAuthClient{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.getOAuthClientName(),
		},
		Secret:       clientSecret,
		RedirectURIs: redirectUris,
		GrantMethod:  oauthv1.GrantHandlerPrompt,
	}

	_, err = r.ReconcileOauthClient(ctx, installation, oauthClient, serverClient)
	if err != nil {
		return err
	}

	if !containsIdentityProvider(kcr.Spec.Realm.IdentityProviders, idpAlias) {
		logrus.Infof("Adding keycloak realm client")
		if kcr.Spec.Realm.IdentityProviders == nil {
			kcr.Spec.Realm.IdentityProviders = []*keycloak.KeycloakIdentityProvider{}
		}
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
				"baseUrl":         r.ApiUrl,
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

func (r *Reconciler) reconcileBlackboxTargets(ctx context.Context, installation *integreatlyv1alpha1.Installation, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	cfg, err := r.ConfigManager.ReadMonitoring()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error reading monitoring config: %w", err)
	}

	err = monitoring.CreateBlackboxTarget("integreatly-rhsso", monitoringv1alpha1.BlackboxtargetData{
		Url:     r.Config.GetHost(),
		Service: "rhsso-ui",
	}, ctx, cfg, installation, client)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating rhsso blackbox target: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) setupGithubIDP(ctx context.Context, kcr *keycloak.KeycloakRealm, serverClient k8sclient.Client) error {
	githubCreds := &corev1.Secret{}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: githubOauthAppCredentialsSecretName, Namespace: r.ConfigManager.GetOperatorNamespace()}, githubCreds)
	if err != nil {
		logrus.Errorf("Unable to find Github oauth credentials secret in namespace %s", r.ConfigManager.GetOperatorNamespace())
		return err
	}

	if !containsIdentityProvider(kcr.Spec.Realm.IdentityProviders, githubIdpAlias) {
		logrus.Infof("Adding github identity provider to the keycloak realm")
		if kcr.Spec.Realm.IdentityProviders == nil {
			kcr.Spec.Realm.IdentityProviders = []*keycloak.KeycloakIdentityProvider{}
		}
		kcr.Spec.Realm.IdentityProviders = append(kcr.Spec.Realm.IdentityProviders, &keycloak.KeycloakIdentityProvider{
			Alias:                     githubIdpAlias,
			ProviderID:                githubIdpAlias,
			Enabled:                   true,
			TrustEmail:                false,
			StoreToken:                true,
			AddReadTokenRoleOnCreate:  true,
			FirstBrokerLoginFlowAlias: "first broker login",
			LinkOnly:                  true,
			Config: map[string]string{
				"hideOnLoginPage": "true",
				"clientId":        fmt.Sprintf("%s", githubCreds.Data["clientId"]),
				"disableUserInfo": "",
				"clientSecret":    fmt.Sprintf("%s", githubCreds.Data["secret"]),
				"defaultScope":    "repo,user,write:public_key,admin:repo_hook,read:org,public_repo,user:email",
				"useJwksUrl":      "true",
			},
		})
	}
	// We need to revisit how the github idp gets created/updated
	// client ID and secret can get outdated we need to ensure they are synced with the value secret in the github-oauth-secret
	return nil
}

func containsIdentityProvider(providers []*keycloak.KeycloakIdentityProvider, alias string) bool {
	for _, p := range providers {
		if p.Alias == alias {
			return true
		}
	}
	return false
}
func getUserDiff(keycloakUsers []keycloak.KeycloakAPIUser, openshiftUsers []usersv1.User) ([]usersv1.User, []int) {
	var added []usersv1.User
	for _, osUser := range openshiftUsers {
		if !kcContainsOsUser(keycloakUsers, osUser) {
			added = append(added, osUser)
		}
	}

	var deleted []int
	for i, kcUser := range keycloakUsers {
		if !OsUserInKc(openshiftUsers, kcUser) {
			deleted = append(deleted, i)
		}
	}

	return added, deleted
}

func syncronizeWithOpenshiftUsers(keycloakUsers []keycloak.KeycloakAPIUser, ctx context.Context, serverClient k8sclient.Client) ([]keycloak.KeycloakAPIUser, error) {
	openshiftUsers := &usersv1.UserList{}
	err := serverClient.List(ctx, openshiftUsers)
	if err != nil {
		return nil, err
	}
	added, deletedIndexes := getUserDiff(keycloakUsers, openshiftUsers.Items)

	for _, index := range deletedIndexes {
		keycloakUsers = remove(index, keycloakUsers)
	}

	for _, osUser := range added {
		email := osUser.Name
		if !strings.Contains(email, "@") {
			email = email + "@example.com"
		}
		keycloakUsers = append(keycloakUsers, keycloak.KeycloakAPIUser{
			Enabled:       true,
			UserName:      osUser.Name,
			EmailVerified: true,
			Email:         email,
			FederatedIdentities: []keycloak.FederatedIdentity{
				{
					IdentityProvider: idpAlias,
					UserID:           string(osUser.UID),
					UserName:         osUser.Name,
				},
			},
		})
	}

	openshiftAdminGroup := &usersv1.Group{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: "dedicated-admins"}, openshiftAdminGroup)
	if err != nil && !k8serr.IsNotFound(err) {
		return nil, err
	}
	for _, kcUser := range keycloakUsers {
		kcUser.ClientRoles = getKeycloakRoles()
	}

	return keycloakUsers, nil
}

func remove(index int, kcUsers []keycloak.KeycloakAPIUser) []keycloak.KeycloakAPIUser {
	kcUsers[index] = kcUsers[len(kcUsers)-1]
	return kcUsers[:len(kcUsers)-1]
}

func kcContainsOsUser(kcUsers []keycloak.KeycloakAPIUser, osUser usersv1.User) bool {
	for _, kcu := range kcUsers {
		if kcu.UserName == osUser.Name {
			return true
		}
	}

	return false
}

func OsUserInKc(osUsers []usersv1.User, kcUser keycloak.KeycloakAPIUser) bool {
	for _, osu := range osUsers {
		if osu.Name == kcUser.UserName {
			return true
		}
	}

	return false
}

func (r *Reconciler) createOrUpdateKeycloakUser(user keycloak.KeycloakAPIUser, inst *integreatlyv1alpha1.Installation, ctx context.Context, serverClient k8sclient.Client) (controllerutil.OperationResult, error) {
	selector := &keycloak.KeycloakUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("generated-%v", user.UserName),
			Namespace: r.Config.GetNamespace(),
		},
	}

	return controllerutil.CreateOrUpdate(ctx, serverClient, selector, func() error {
		ownerutil.EnsureOwner(selector, inst)
		selector.Spec.RealmSelector = &metav1.LabelSelector{
			MatchLabels: GetInstanceLabels(),
		}
		selector.Labels = GetInstanceLabels()
		selector.Spec.User = user
		return nil
	})
}

func GetKeycloakUsers(ctx context.Context, serverClient k8sclient.Client, ns string) ([]keycloak.KeycloakAPIUser, error) {
	var users keycloak.KeycloakUserList

	listOptions := []k8sclient.ListOption{
		k8sclient.MatchingLabels(GetInstanceLabels()),
		k8sclient.InNamespace(ns),
	}
	err := serverClient.List(ctx, &users, listOptions...)
	if err != nil {
		return nil, err
	}

	mappedUsers := make([]keycloak.KeycloakAPIUser, len(users.Items))
	for i, user := range users.Items {
		mappedUsers[i] = user.Spec.User
	}

	return mappedUsers, nil
}

func getKeycloakRoles() map[string][]string {
	roles := map[string][]string{
		"account": {
			"manage-account",
			"view-profile",
		},
		"broker": {
			"read-token",
		},
	}
	return roles
}

func GetInstanceLabels() map[string]string {
	return map[string]string{
		SSOLabelKey: SSOLabelValue,
	}
}
