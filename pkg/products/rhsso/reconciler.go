package rhsso

import (
	"context"
	"fmt"
	"strings"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"

	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	userHelper "github.com/integr8ly/integreatly-operator/pkg/resources/user"

	"k8s.io/apimachinery/pkg/util/intstr"

	oauthv1 "github.com/openshift/api/oauth/v1"
	corev1 "k8s.io/api/core/v1"

	keycloakCommon "github.com/integr8ly/keycloak-client/pkg/common"
	"github.com/sirupsen/logrus"

	"github.com/integr8ly/integreatly-operator/pkg/products/monitoring"

	monitoringv1alpha1 "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	"github.com/keycloak/keycloak-operator/pkg/common"

	appsv1 "github.com/openshift/api/apps/v1"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	defaultOperandNamespace   = "rhsso"
	keycloakName              = "rhsso"
	keycloakRealmName         = "openshift"
	idpAlias                  = "openshift-v4"
	githubIdpAlias            = "github"
	authFlowAlias             = "authdelay"
	manifestPackage           = "integreatly-rhsso"
	adminCredentialSecretName = "credential-" + keycloakName
	numberOfReplicas          = 2
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
	installation  *integreatlyv1alpha1.RHMI
	logger        *logrus.Entry
	oauthv1Client oauthClient.OauthV1Interface
	APIURL        string
	*resources.Reconciler
	recorder              record.EventRecorder
	keycloakClientFactory keycloakCommon.KeycloakClientFactory
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, oauthv1Client oauthClient.OauthV1Interface, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, APIURL string, keycloakClientFactory keycloakCommon.KeycloakClientFactory) (*Reconciler, error) {
	config, err := configManager.ReadRHSSO()
	if err != nil {
		return nil, err
	}
	if config.GetNamespace() == "" {
		config.SetNamespace(installation.Spec.NamespacePrefix + defaultOperandNamespace)
	}
	if config.GetOperatorNamespace() == "" {
		if installation.Spec.OperatorsInProductNamespace {
			config.SetOperatorNamespace(config.GetNamespace())
		} else {
			config.SetOperatorNamespace(config.GetNamespace() + "-operator")
		}
	}

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		Config:                config,
		ConfigManager:         configManager,
		mpm:                   mpm,
		installation:          installation,
		logger:                logger,
		oauthv1Client:         oauthv1Client,
		Reconciler:            resources.NewReconciler(mpm),
		recorder:              recorder,
		APIURL:                APIURL,
		keycloakClientFactory: keycloakClientFactory,
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
func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
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
		err = resources.RemoveOauthClient(r.oauthv1Client, r.getOAuthClientName())
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		//if both namespaces are deleted, return complete
		_, operatorNSErr := resources.GetNS(ctx, r.Config.GetOperatorNamespace(), serverClient)
		_, nsErr := resources.GetNS(ctx, r.Config.GetNamespace(), serverClient)
		if k8serr.IsNotFound(operatorNSErr) && k8serr.IsNotFound(nsErr) {
			return integreatlyv1alpha1.PhaseCompleted, nil
		}
		return integreatlyv1alpha1.PhaseInProgress, nil
	})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetNamespace(), installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetNamespace()), err)
		return phase, err
	}
	phase, err = r.ReconcileNamespace(ctx, r.Config.GetOperatorNamespace(), installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetOperatorNamespace()), err)
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, r.Config.GetNamespace(), serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, fmt.Sprintf("Failed to retrieve %s namespace", r.Config.GetNamespace()), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	phase, err = resources.ReconcileSecretToProductNamespace(ctx, serverClient, r.ConfigManager, adminCredentialSecretName, r.Config.GetNamespace())
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile admin credentials secret", err)
		return phase, err
	}

	preUpgradeBackupsExecutor := r.preUpgradeBackupsExecutor()
	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: constants.RHSSOSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetOperatorNamespace(), ManifestPackage: manifestPackage}, []string{r.Config.GetNamespace()}, preUpgradeBackupsExecutor, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.RHSSOSubscriptionName), err)
		return phase, err
	}

	phase, err = r.createKeycloakRoute(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to handle in progress phase", err)
		return phase, err
	}

	phase, err = r.reconcileCloudResources(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile cloud resources", err)
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile components", err)
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

	phase, err = resources.ReconcileSecretToRHMIOperatorNamespace(ctx, serverClient, r.ConfigManager, adminCredentialSecretName, r.Config.GetNamespace())
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile admin credential secret to RHMI operator namespace", err)
		return phase, err
	}

	phase, err = r.reconcileKubeStateMetricsEndpointAvailableAlerts(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile endpoint available alerts", err)
		return phase, err
	}

	phase, err = r.reconcileKubeStateMetricsOperatorEndpointAvailableAlerts(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile operator endpoint available alerts", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.AuthenticationStage, r.Config.GetProductName())
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) cleanupKeycloakResources(ctx context.Context, inst *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
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
func (r *Reconciler) createResource(ctx context.Context, resourceName string, serverClient k8sclient.Client) (runtime.Object, error) {
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

// workaround: the keycloak operator creates a route with TLS passthrough config
// this should use the same valid certs as the cluster itself but for some reason the
// signing operator gives out self signed certs
// to circumvent this we create another keycloak route with edge termination
func (r *Reconciler) createKeycloakRoute(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// We need a route with edge termination to serve the correct cluster certificate
	edgeRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak-edge",
			Namespace: r.Config.GetNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, edgeRoute, func() error {
		host := edgeRoute.Spec.Host
		edgeRoute.Spec = routev1.RouteSpec{
			Host: host,
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: "keycloak",
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
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating keycloak edge route: %w", err)
	}
	r.logger.Info(fmt.Sprintf("operation result of creating %v service was %v", edgeRoute.Name, or))

	if edgeRoute.Spec.Host == "" {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	// Override the keycloak host to the host of the edge route (instead of the
	// operator generated route)
	r.Config.SetHost(fmt.Sprintf("https://%v", edgeRoute.Spec.Host))
	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Error writing to config in rhssouser reconciler: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileCloudResources(ctx context.Context, installation *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.logger.Info("Reconciling Keycloak external database instance")

	postgresName := fmt.Sprintf("%s%s", constants.RHSSOPostgresPrefix, installation.Name)
	postgres, credentialSec, err := resources.ReconcileRHSSOPostgresCredentials(ctx, installation, serverClient, postgresName, r.Config.GetNamespace(), defaultOperandNamespace)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile database credentials secret while provisioning sso: %w", err)
	}

	// at this point it should be ok to create the failed alert.
	if postgres != nil {
		// create prometheus phase failed rule
		_, err = resources.CreatePostgresResourceStatusPhaseFailedAlert(ctx, serverClient, installation, postgres)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres failure alert: %w", err)
		}

		// create prometheus pending rule only when CR has completed for the first time.
		if postgres.Status.Phase == types.PhaseComplete {

			_, err = resources.CreatePostgresResourceStatusPhasePendingAlert(ctx, serverClient, installation, postgres)
			if err != nil {
				return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres pending alert: %w", err)
			}
		}
	}

	// postgres provisioning is still in progress
	if credentialSec == nil {
		return integreatlyv1alpha1.PhaseAwaitingCloudResources, nil
	}

	// create the prometheus availability rule
	if _, err = resources.CreatePostgresAvailabilityAlert(ctx, serverClient, installation, postgres); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres prometheus alert for rhsso: %w", err)
	}

	// create the prometheus connectivity rule
	if _, err = resources.CreatePostgresConnectivityAlert(ctx, serverClient, installation, postgres); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres prometheus connectivity alert for rhsso : %s", err)
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, installation *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.logger.Info("Reconciling Keycloak components")
	kc := &keycloak.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, kc, func() error {
		kc.Spec.Extensions = []string{
			"https://github.com/aerogear/keycloak-metrics-spi/releases/download/2.0.1/keycloak-metrics-spi-2.0.1.jar",
			"https://github.com/integr8ly/authentication-delay-plugin/releases/download/1.0.1/authdelay.jar",
		}
		kc.Labels = GetInstanceLabels()
		if kc.Spec.Instances < numberOfReplicas {
			kc.Spec.Instances = numberOfReplicas
		}
		kc.Spec.ExternalDatabase = keycloak.KeycloakExternalDatabase{Enabled: true}
		kc.Spec.ExternalAccess = keycloak.KeycloakExternalAccess{
			Enabled: true,
		}
		kc.Spec.Profile = RHSSOProfile
		kc.Spec.PodDisruptionBudget = keycloak.PodDisruptionBudgetConfig{Enabled: true}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update keycloak custom resource: %w", err)
	}
	host := r.Config.GetHost()
	if host == "" {
		r.logger.Infof("URL for Keycloak not yet available")
		return integreatlyv1alpha1.PhaseAwaitingComponents, fmt.Errorf("Host for Keycloak not yet available")
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
		err = r.setupOpenshiftIDP(ctx, installation, kcr, serverClient, host)
		if err != nil {
			return fmt.Errorf("failed to setup Openshift IDP: %w", err)
		}

		err = r.setupGithubIDP(ctx, kc, kcr, serverClient, installation)
		if err != nil {
			return fmt.Errorf("failed to setup Github IDP: %w", err)
		}
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update keycloak realm: %w", err)
	}
	r.logger.Infof("The operation result for keycloakrealm %s was %s", kcr.Name, or)

	// create keycloak authentication delay flow and adds to openshift idp
	authenticated, err := r.keycloakClientFactory.AuthenticatedClient(*kc)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to authenticate client in keycloak api %w", err)
	}

	err = createAuthDelayAuthenticationFlow(authenticated)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create and add keycloak authentication flow: %w", err)
	}
	r.logger.Infof("Authentication flow added to %s IDP", idpAlias)

	// Get all currently existing keycloak users
	keycloakUsers, err := GetKeycloakUsers(ctx, serverClient, r.Config.GetNamespace())
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to list the keycloak users: %w", err)
	}

	// Sync keycloak with openshift users
	users, err := syncronizeWithOpenshiftUsers(ctx, keycloakUsers, serverClient, r.Config.GetNamespace())
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to synchronize the users: %w", err)
	}

	// Create / update the synchronized users
	for _, user := range users {
		or, err = r.createOrUpdateKeycloakUser(ctx, user, serverClient)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update the customer admin user: %w", err)
		}
		r.logger.Infof("The operation result for keycloakuser %s was %s", user.UserName, or)

	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) handleProgressPhase(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	kc := &keycloak.Keycloak{}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: keycloakName, Namespace: r.Config.GetNamespace()}, kc)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	// The keycloak operator does not set the product version currently - should fetch from KeyCloak.Status.Version when fixed
	r.Config.SetProductVersion(string(integreatlyv1alpha1.VersionRHSSO))
	// The Keycloak Operator doesn't currently set the operator version
	r.Config.SetOperatorVersion(string(integreatlyv1alpha1.OperatorVersionRHSSO))
	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	r.logger.Info("checking ready status for rhsso")
	kcr := &keycloak.KeycloakRealm{}

	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: keycloakRealmName, Namespace: r.Config.GetNamespace()}, kcr)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get keycloak realm custom resource: %w", err)
	}

	if kcr.Status.Phase == keycloak.PhaseReconciling {
		err = r.exportConfig(ctx, serverClient)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to write rhsso config: %w", err)
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

	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return fmt.Errorf("could not update keycloak config: %w", err)
	}
	return nil
}

func (r *Reconciler) setupOpenshiftIDP(ctx context.Context, installation *integreatlyv1alpha1.RHMI, kcr *keycloak.KeycloakRealm, serverClient k8sclient.Client, host string) error {
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
		return fmt.Errorf("Could not find %s key in %s Secret: %w", string(r.Config.GetProductName()), oauthClientSecrets.Name, err)
	}
	clientSecret := string(clientSecretBytes)

	redirectUris := []string{
		host + "/auth/realms/openshift/broker/openshift-v4/endpoint",
	}

	oauthClient := &oauthv1.OAuthClient{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.getOAuthClientName(),
		},
		Secret:       clientSecret,
		RedirectURIs: redirectUris,
		GrantMethod:  oauthv1.GrantHandlerAuto,
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

func (r *Reconciler) reconcileBlackboxTargets(ctx context.Context, installation *integreatlyv1alpha1.RHMI, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	cfg, err := r.ConfigManager.ReadMonitoring()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error reading monitoring config: %w", err)
	}

	err = monitoring.CreateBlackboxTarget(ctx, "integreatly-rhsso", monitoringv1alpha1.BlackboxtargetData{
		Url:     r.Config.GetHost(),
		Service: "rhsso-ui",
	}, cfg, installation, client)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating rhsso blackbox target: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) setupGithubIDP(ctx context.Context, kc *keycloak.Keycloak, kcr *keycloak.KeycloakRealm, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI) error {
	githubCreds := &corev1.Secret{}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: r.ConfigManager.GetGHOauthClientsSecretName(), Namespace: r.ConfigManager.GetOperatorNamespace()}, githubCreds)
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

	githubClientID := string(githubCreds.Data["clientId"])
	githubClientSecret := string(githubCreds.Data["secret"])

	// check if GH credentials have been set up
	githubMockCred := "dummy"
	if githubClientID == githubMockCred || githubClientSecret == githubMockCred {
		return nil
	}

	logrus.Infof("Syncing github identity provider to the keycloak realm")

	// Get an authenticated keycloak api client for the instance
	keycloakFactory := common.LocalConfigKeycloakFactory{}
	authenticated, err := keycloakFactory.AuthenticatedClient(*kc)
	if err != nil {
		return fmt.Errorf("Unable to authenticate to the Keycloak API: %s", err)
	}

	identityProvider, err := authenticated.GetIdentityProvider(githubIdpAlias, kcr.Spec.Realm.DisplayName)
	if err != nil {
		return fmt.Errorf("Unable to get Identity Provider from Keycloak API: %s", err)
	}

	identityProvider.Config["clientId"] = githubClientID
	identityProvider.Config["clientSecret"] = githubClientSecret
	err = authenticated.UpdateIdentityProvider(identityProvider, kcr.Spec.Realm.DisplayName)
	if err != nil {
		return fmt.Errorf("Unable to update Identity Provider to Keycloak API: %s", err)
	}

	installation.Status.GitHubOAuthEnabled = true

	return nil
}

func (r *Reconciler) preUpgradeBackupsExecutor() backup.BackupExecutor {
	if r.installation.Spec.UseClusterStorage != "false" {
		return backup.NewNoopBackupExecutor()
	}

	return backup.NewAWSBackupExecutor(
		r.installation.Namespace,
		"rhsso-postgres-rhmi",
		backup.PostgresSnapshotType,
	)
}

func containsIdentityProvider(providers []*keycloak.KeycloakIdentityProvider, alias string) bool {
	for _, p := range providers {
		if p.Alias == alias {
			return true
		}
	}
	return false
}

func getUserDiff(keycloakUsers []keycloak.KeycloakAPIUser, openshiftUsers []usersv1.User, groups *usersv1.GroupList) (added []usersv1.User, deleted []keycloak.KeycloakAPIUser) {
	for _, osUser := range openshiftUsers {
		if !kcContainsOsUser(keycloakUsers, osUser) && !userHelper.UserInExclusionGroup(osUser, groups) {
			added = append(added, osUser)
		}
	}

	for _, kcUser := range keycloakUsers {
		if !OsUserInKc(openshiftUsers, kcUser) {
			deleted = append(deleted, kcUser)
		}
	}

	return added, deleted
}

func syncronizeWithOpenshiftUsers(ctx context.Context, keycloakUsers []keycloak.KeycloakAPIUser, serverClient k8sclient.Client, ns string) ([]keycloak.KeycloakAPIUser, error) {
	openshiftUsers := &usersv1.UserList{}
	err := serverClient.List(ctx, openshiftUsers)
	if err != nil {
		return nil, err
	}

	groups := &usersv1.GroupList{}
	err = serverClient.List(ctx, groups)
	if err != nil {
		return nil, err
	}

	added, deletedUsers := getUserDiff(keycloakUsers, openshiftUsers.Items, groups)

	keycloakUsers, err = deleteKeycloakUsers(keycloakUsers, deletedUsers, ns, ctx, serverClient)
	if err != nil {
		return nil, err
	}

	for _, osUser := range added {
		email, err := userHelper.GetUserEmailFromIdentity(ctx, serverClient, osUser)

		if err != nil {
			return nil, err
		}

		if email == "" {
			email = osUser.Name + "@rhmi.io"
		}

		newKeycloakUser := keycloak.KeycloakAPIUser{
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
		}
		userHelper.AppendUpdateProfileActionForUserWithoutEmail(&newKeycloakUser)

		keycloakUsers = append(keycloakUsers, newKeycloakUser)
	}

	if err != nil && !k8serr.IsNotFound(err) {
		return nil, err
	}
	for index := range keycloakUsers {
		keycloakUsers[index].ClientRoles = getKeycloakRoles()
	}

	return keycloakUsers, nil
}

func deleteKeycloakUsers(allKcUsers []keycloak.KeycloakAPIUser, deletedUsers []keycloak.KeycloakAPIUser, ns string, ctx context.Context, serverClient k8sclient.Client) ([]keycloak.KeycloakAPIUser, error) {

	for _, delUser := range deletedUsers {
		// Remove from all users list
		for i, user := range allKcUsers {
			// ID is not populated, have to use UserName. Should be unique on master Realm
			if delUser.UserName == user.UserName {
				allKcUsers[i] = allKcUsers[len(allKcUsers)-1]
				allKcUsers = allKcUsers[:len(allKcUsers)-1]
				break
			}
		}

		// Delete the CR
		kcUser := &keycloak.KeycloakUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      userHelper.GetValidGeneratedUserName(delUser),
				Namespace: ns,
			},
		}
		err := serverClient.Delete(ctx, kcUser)
		if err != nil {
			return nil, fmt.Errorf("failed to delete keycloak user: %w", err)
		}
	}

	return allKcUsers, nil
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

func (r *Reconciler) createOrUpdateKeycloakUser(ctx context.Context, user keycloak.KeycloakAPIUser, serverClient k8sclient.Client) (controllerutil.OperationResult, error) {
	kcUser := &keycloak.KeycloakUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userHelper.GetValidGeneratedUserName(user),
			Namespace: r.Config.GetNamespace(),
		},
	}

	return controllerutil.CreateOrUpdate(ctx, serverClient, kcUser, func() error {
		kcUser.Spec.RealmSelector = &metav1.LabelSelector{
			MatchLabels: GetInstanceLabels(),
		}
		kcUser.Labels = GetInstanceLabels()
		kcUser.Spec.User = user
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

// creates keycloak authentication flow to delay login until user is reconciled in 3scale or other products
func createAuthDelayAuthenticationFlow(authenticated keycloakCommon.KeycloakInterface) error {

	authFlow, err := authenticated.FindAuthenticationFlowByAlias(authFlowAlias, keycloakRealmName)
	if err != nil {
		return fmt.Errorf("failed to find authentication flow by alias via keycloak api %w", err)
	}
	if authFlow == nil {
		authFlow := keycloakCommon.AuthenticationFlow{
			Alias:      authFlowAlias,
			ProviderID: "basic-flow", // providerId is "client-flow" for client and "basic-flow" for generic in Top Level Flow Type
			TopLevel:   true,
			BuiltIn:    false,
		}
		_, err := authenticated.CreateAuthenticationFlow(authFlow, keycloakRealmName)
		if err != nil {
			return fmt.Errorf("failed to create authentication flow via keycloak api %w", err)
		}
	}

	executionProviderID := "delay-authentication"
	authExecution, err := authenticated.FindAuthenticationExecutionForFlow(authFlowAlias, keycloakRealmName, func(execution *keycloak.AuthenticationExecutionInfo) bool {
		return execution.ProviderID == executionProviderID
	})
	if err != nil {
		return fmt.Errorf("failed to find authentication execution flow via keycloak api %w", err)
	}
	if authExecution == nil {
		err = authenticated.AddExecutionToAuthenticatonFlow(authFlowAlias, keycloakRealmName, executionProviderID, keycloakCommon.Required)
		if err != nil {
			return fmt.Errorf("failed to add execution to authentication flow via keycloak api %w", err)
		}
	}

	idp, err := authenticated.GetIdentityProvider(idpAlias, keycloakRealmName)
	if err != nil {
		return fmt.Errorf("failed to get identity provider via keycloak api %w", err)
	}
	if idp.FirstBrokerLoginFlowAlias != authFlowAlias {
		idp.FirstBrokerLoginFlowAlias = authFlowAlias
		err = authenticated.UpdateIdentityProvider(idp, keycloakRealmName)
		if err != nil {
			return fmt.Errorf("failed to update identity provider via keycloak api %w", err)
		}
	}

	return nil
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
