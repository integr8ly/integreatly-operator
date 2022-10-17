package rhsso

import (
	"context"
	"fmt"

	grafanav1alpha1 "github.com/grafana-operator/grafana-operator/v4/api/integreatly/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/products/rhssocommon"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	dr "github.com/integr8ly/integreatly-operator/pkg/resources/dynamic-resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	userHelper "github.com/integr8ly/integreatly-operator/pkg/resources/user"
	"github.com/integr8ly/integreatly-operator/version"
	keycloakCommon "github.com/integr8ly/keycloak-client/pkg/common"
	keycloakTypes "github.com/integr8ly/keycloak-client/pkg/types"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	usersv1 "github.com/openshift/api/user/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	adminCredentialSecretName = "credential-" + keycloakName
	ssoType                   = "rhsso"
	postgresResourceName      = "rhsso-postgres-rhmi"
	routeName                 = "keycloak-edge"
)

const (
	SSOLabelKey   = "sso"
	SSOLabelValue = "integreatly"
	RHSSOProfile  = "RHSSO"
)

type Reconciler struct {
	Config *config.RHSSO
	Log    l.Logger
	*rhssocommon.Reconciler
	isUpgrade             bool
	KeycloakClientFactory keycloakCommon.KeycloakClientFactory
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, oauthv1Client oauthClient.OauthV1Interface, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, APIURL string, keycloakClientFactory keycloakCommon.KeycloakClientFactory, logger l.Logger, productDeclaration *marketplace.ProductDeclaration) (*Reconciler, error) {
	if productDeclaration == nil {
		return nil, fmt.Errorf("no product declaration found for RHSSO")
	}

	config, err := configManager.ReadRHSSO()
	if err != nil {
		return nil, err
	}

	rhssocommon.SetNameSpaces(installation, config.RHSSOCommon, defaultOperandNamespace)

	return &Reconciler{
		Config:     config,
		Log:        logger,
		Reconciler: rhssocommon.NewReconciler(configManager, mpm, installation, logger, oauthv1Client, recorder, APIURL, keycloakClientFactory, *productDeclaration),
		isUpgrade:  rhssocommon.IsUpgrade(config.RHSSOCommon, integreatlyv1alpha1.VersionRHSSO),
	}, nil
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return version.VerifyProductAndOperatorVersion(
		installation.Status.Stages[integreatlyv1alpha1.InstallStage].Products[integreatlyv1alpha1.ProductRHSSO],
		string(integreatlyv1alpha1.VersionRHSSO),
		string(integreatlyv1alpha1.OperatorVersionRHSSO),
	)
}

// Reconcile reads that state of the cluster for rhsso and makes changes based on the state read
// and what is required
func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, productStatus *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client, _ quota.ProductConfig, uninstall bool) (integreatlyv1alpha1.StatusPhase, error) {
	operatorNamespace := r.Config.GetOperatorNamespace()
	productNamespace := r.Config.GetNamespace()
	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), uninstall, func() (integreatlyv1alpha1.StatusPhase, error) {
		// Check if namespace is still present before trying to delete it resources
		_, err := resources.GetNS(ctx, productNamespace, serverClient)
		if !k8serr.IsNotFound(err) {
			phase, err := r.CleanupKeycloakResources(ctx, installation, serverClient, productNamespace)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}

			phase, err = resources.RemoveNamespace(ctx, installation, serverClient, productNamespace, r.Log)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
		}
		_, err = resources.GetNS(ctx, operatorNamespace, serverClient)
		if !k8serr.IsNotFound(err) {
			phase, err := resources.RemoveNamespace(ctx, installation, serverClient, operatorNamespace, r.Log)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
		}
		err = resources.RemoveOauthClient(r.Oauthv1Client, r.GetOAuthClientName(r.Config), r.Log)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		//if both namespaces are deleted, return complete
		_, operatorNSErr := resources.GetNS(ctx, operatorNamespace, serverClient)
		_, nsErr := resources.GetNS(ctx, productNamespace, serverClient)
		if k8serr.IsNotFound(operatorNSErr) && k8serr.IsNotFound(nsErr) {
			return integreatlyv1alpha1.PhaseCompleted, nil
		}
		return integreatlyv1alpha1.PhaseInProgress, nil
	}, r.Log)
	if err != nil || phase == integreatlyv1alpha1.PhaseFailed {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	if uninstall {
		return phase, nil
	}
	phase, err = r.ReconcileNamespace(ctx, productNamespace, installation, serverClient, r.Log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", productNamespace), err)
		return phase, err
	}
	phase, err = r.ReconcileNamespace(ctx, operatorNamespace, installation, serverClient, r.Log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", operatorNamespace), err)
		return phase, err
	}

	phase, err = resources.ReconcileSecretToProductNamespace(ctx, serverClient, r.ConfigManager, adminCredentialSecretName, productNamespace, r.Log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile admin credentials secret", err)
		return phase, err
	}

	phase, err = r.SetRollingStrategyForUpgrade(r.isUpgrade, ctx, serverClient, r.Config.RHSSOCommon, integreatlyv1alpha1.VersionRHSSO, keycloakName)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to set rolling strategy for upgrade", err)
		return phase, err
	}

	phase, err = r.CheckGrafanaDashboardCRD(ctx, r.Oauthv1Client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to retrieve grafana dashboard crd", err)
		return phase, err
	}

	phase, err = r.ReconcileSubscription(ctx, serverClient, installation, productNamespace, operatorNamespace, postgresResourceName)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.RHSSOSubscriptionName), err)
		return phase, err
	}

	phase, err = r.CreateKeycloakRoute(ctx, serverClient, r.Config, r.Config.RHSSOCommon, routeName)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to handle in progress phase", err)
		return phase, err
	}

	phase, err = r.ReconcileCloudResources(constants.RHSSOPostgresPrefix, defaultOperandNamespace, ssoType, r.Config.RHSSOCommon, ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile cloud resources", err)
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile components", err)
		return phase, err
	}

	phase, err = r.ReconcileStatefulSet(ctx, serverClient, r.Config.RHSSOCommon)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconsile RHSSO pod priority", err)
		return phase, err
	}

	phase, err = r.HandleProgressPhase(ctx, serverClient, keycloakName, keycloakRealmName, r.Config, r.Config.RHSSOCommon, string(integreatlyv1alpha1.VersionRHSSO), string(integreatlyv1alpha1.OperatorVersionRHSSO))
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, err
	}

	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error writing to config in rhsso cluster reconciler: %w", err)
	}

	phase, err = r.ReconcilePrometheusProbes(ctx, serverClient, "integreatly-rhsso", r.Config.GetHost(), "rhsso-ui")
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile prometheus probes", err)
		return phase, err
	}

	phase, err = r.RemovePodMonitors(ctx, serverClient, r.Config)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to remove pod monitor", err)
		return phase, err
	}

	phase, err = resources.ReconcileSecretToRHMIOperatorNamespace(ctx, serverClient, r.ConfigManager, adminCredentialSecretName, productNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile admin credential secret to RHMI operator namespace", err)
		return phase, err
	}

	phase, err = r.newAlertsReconciler(r.Log, r.Installation.Spec.Type).ReconcileAlerts(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile alerts", err)
		return phase, err
	}

	phase, err = r.exportDashboard(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to export dashboard to the observability namespace", err)
		return phase, err
	}

	phase, err = r.ExportAlerts(ctx, serverClient, string(r.Config.GetProductName()), r.Config.GetNamespace())
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to export alerts to the observability namespace", err)
		return phase, err
	}

	productStatus.Host = r.Config.GetHost()
	productStatus.Version = r.Config.GetProductVersion()
	productStatus.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.Recorder, installation, integreatlyv1alpha1.InstallStage, r.Config.GetProductName())
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, installation *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// Create empty unstructured and attempt getting it from cluster
	kcUnstructured := dr.CreateUnstructuredWithGVK(keycloakTypes.KeycloakGroup, keycloakTypes.KeycloakKind, keycloakTypes.KeycloakVersion, keycloakName, r.Config.GetNamespace())

	err := serverClient.Get(context.TODO(), types.NamespacedName{Namespace: r.Config.GetNamespace(), Name: keycloakName}, kcUnstructured)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	// Convert keycloak from cluster to typed object
	kcOriginal, err := dr.ConvertKeycloakUnstructuredToTyped(*kcUnstructured)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Retrieve keycloak typed object desired state
	kcTypedDesired, err := r.createKeycloakTypedObject(installation)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Update keycloak from cluster typed spec to the desired typed spec
	kcOriginal.Spec = kcTypedDesired.Spec
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Convert updated original to unstructed - this is done to createOrUpdate the original pre-update
	kcUnstructuredOriginalUpdated, _ := dr.ConvertKeycloakTypedToUnstructured(*kcOriginal)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// CreateOrUpdate Keycloak CR
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, kcUnstructured, func() error {
		kcUnstructured.Object = kcUnstructuredOriginalUpdated.Object
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update keycloak custom resource: %w", err)
	}
	// Patching the OwnerReference on the admin credentials secret
	err = resources.AddOwnerRefToSSOSecret(ctx, serverClient, adminCredentialSecretName, r.Config.GetNamespace(), *kcOriginal, r.Log)
	if err != nil {
		events.HandleError(r.Recorder, installation, integreatlyv1alpha1.PhaseFailed, "Failed to add ownerReference admin credentials secret", err)
		return integreatlyv1alpha1.PhaseFailed, err
	}
	host := r.Config.GetHost()
	if host == "" {
		r.Log.Info("URL for Keycloak not yet available")
		return integreatlyv1alpha1.PhaseAwaitingComponents, nil
	}

	// Create empty unstructured and attempt getting it from cluster
	kcRealmUnstructured := dr.CreateUnstructuredWithGVK(keycloakTypes.KeycloakRealmGroup, keycloakTypes.KeycloakRealmKind, keycloakTypes.KeycloakRealmVersion, keycloakRealmName, r.Config.GetNamespace())

	err = serverClient.Get(context.TODO(), types.NamespacedName{Namespace: r.Config.GetNamespace(), Name: keycloakRealmName}, kcRealmUnstructured)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	kcRealmTyped, err := dr.ConvertKeycloakRealmUnstructuredToTyped(*kcRealmUnstructured)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Retrieve keycloak typed object desired state
	kcRealmTypedDesired, err := r.createDesiredKeycloakRealmTypedObject(ctx, serverClient, installation, *kcOriginal)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	kcRealmTyped.Labels = kcRealmTypedDesired.Labels
	kcRealmTyped.Spec = kcRealmTypedDesired.Spec

	// Convert updated original to unstructed - this is done to createOrUpdate the original pre-update
	kcRealmUnstructuredOriginalUpdated, _ := dr.ConvertKeycloakRealmTypedToUnstructured(*kcRealmTyped)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, kcRealmUnstructured, func() error {
		kcRealmUnstructured = kcRealmUnstructuredOriginalUpdated
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update keycloak realm: %w", err)
	}
	r.Log.Infof("Operation result", l.Fields{"keycloakrealm": kcRealmTyped.Name, "result": or})

	// create keycloak authentication delay flow and adds to openshift idp
	authenticated, err := r.KeycloakClientFactory.AuthenticatedClient(*kcOriginal)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to authenticate client in keycloak api %w", err)
	}

	err = createAuthDelayAuthenticationFlow(authenticated)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create and add keycloak authentication flow: %w", err)
	}
	r.Log.Infof("Authentication flow added to IDP", l.Fields{"idpAlias": idpAlias})

	err = r.SyncOpenshiftIDPClientSecret(ctx, serverClient, authenticated, r.Config, keycloakRealmName)
	if err != nil {
		r.Log.Error("Failed to sync openshift idp client secret", err)
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to sync openshift idp client secret: %w", err)
	}

	// Get all currently existing keycloak users
	keycloakUsers, err := GetKeycloakUsers(ctx, serverClient, r.Config.GetNamespace())
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to list the keycloak users: %w", err)
	}

	// Sync keycloak with openshift users
	users, err := syncronizeWithOpenshiftUsers(ctx, keycloakUsers, serverClient, r.Config.GetNamespace(), r.Installation, r.Log)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to synchronize the users: %w", err)
	}

	// Create / update the synchronized users
	for _, user := range users {
		err = r.createOrUpdateKeycloakUser(ctx, user, serverClient)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update the customer admin user: %w", err)
		}
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) createDesiredKeycloakRealmTypedObject(ctx context.Context, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI, kcOriginal keycloakTypes.Keycloak) (keycloakTypes.KeycloakRealm, error) {
	kcr := keycloakTypes.KeycloakRealm{}
	kcr.Spec.RealmOverrides = []*keycloakTypes.RedirectorIdentityProviderOverride{
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

	kcr.Spec.Realm = &keycloakTypes.KeycloakAPIRealm{
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
	redirectUris := []string{r.Config.GetHost() + "/auth/realms/openshift/broker/openshift-v4/endpoint"}
	err := r.SetupOpenshiftIDP(ctx, serverClient, installation, r.Config, &kcr, redirectUris, "")
	if err != nil {
		return kcr, fmt.Errorf("failed to setup Openshift IDP: %w", err)
	}

	if !integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installation.Spec.Type)) {
		err = r.setupGithubIDP(ctx, &kcOriginal, &kcr, serverClient, installation)
		if err != nil {
			return kcr, fmt.Errorf("failed to setup Github IDP: %w", err)
		}
	}

	return kcr, nil
}

func (r *Reconciler) createKeycloakTypedObject(installation *integreatlyv1alpha1.RHMI) (keycloakTypes.Keycloak, error) {
	kcTyped := keycloakTypes.Keycloak{}
	kcTyped.APIVersion = keycloakTypes.KeycloakGroup + "/" + keycloakTypes.KeycloakVersion
	kcTyped.Kind = keycloakTypes.KeycloakKind
	kcTyped.ObjectMeta.Namespace = r.Config.GetNamespace()
	kcTyped.ObjectMeta.Name = keycloakName
	kcTyped.Spec.Extensions = []string{
		"https://github.com/aerogear/keycloak-metrics-spi/releases/download/2.0.1/keycloak-metrics-spi-2.0.1.jar",
		"https://github.com/integr8ly/authentication-delay-plugin/releases/download/1.0.2/authdelay.jar",
	}
	kcTyped.Spec.ExternalDatabase.Enabled = true
	kcTyped.Labels = GetInstanceLabels()
	kcTyped.Spec.ExternalAccess.Enabled = true
	kcTyped.Spec.Profile = RHSSOProfile
	kcTyped.Spec.PodDisruptionBudget.Enabled = true
	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installation.Spec.Type)) {
		kcTyped.Spec.KeycloakDeploymentSpec.Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{corev1.ResourceCPU: k8sresource.MustParse("2000m"), corev1.ResourceMemory: k8sresource.MustParse("2G")},
			Limits:   corev1.ResourceList{corev1.ResourceCPU: k8sresource.MustParse("2000m"), corev1.ResourceMemory: k8sresource.MustParse("2G")},
		}
	} else {
		kcTyped.Spec.KeycloakDeploymentSpec.Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{corev1.ResourceCPU: k8sresource.MustParse("650m"), corev1.ResourceMemory: k8sresource.MustParse("2G")},
			Limits:   corev1.ResourceList{corev1.ResourceCPU: k8sresource.MustParse("650m"), corev1.ResourceMemory: k8sresource.MustParse("2G")},
		}
	}

	// On an upgrade, migration could have changed to recreate strategy for major and minor version bumps
	// Keep the current migration strategy until operator upgrades are complete. Once complete use rolling strategy.
	// On patch upgrades, the rolling strategy will be kept and used throughout the upgrade
	if !r.isUpgrade && r.IsOperatorInstallComplete(kcTyped, integreatlyv1alpha1.OperatorVersionRHSSO) {
		//Set keycloak Update Strategy to Rolling as default
		r.Log.Info("Setting keycloak migration strategy to rolling")
		kcTyped.Spec.Migration.MigrationStrategy = keycloakTypes.StrategyRolling
	}

	// OSD has more resources than PROW, so adding an exception
	numberOfReplicas := r.Config.GetReplicasConfig(r.Installation)

	if kcTyped.Spec.Instances < numberOfReplicas {
		kcTyped.Spec.Instances = numberOfReplicas
	}

	return kcTyped, nil
}

func (r *Reconciler) setupGithubIDP(ctx context.Context, kc *keycloakTypes.Keycloak, kcr *keycloakTypes.KeycloakRealm, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI) error {
	githubCreds := &corev1.Secret{}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: r.ConfigManager.GetGHOauthClientsSecretName(), Namespace: r.ConfigManager.GetOperatorNamespace()}, githubCreds)
	if err != nil {
		r.Log.Errorf("Unable to find Github oauth credentials secret", l.Fields{"ns": r.ConfigManager.GetOperatorNamespace()}, err)
		return err
	}

	if !rhssocommon.ContainsIdentityProvider(kcr.Spec.Realm.IdentityProviders, githubIdpAlias) {
		r.Log.Info("Adding github identity provider to the keycloak realm")
		if kcr.Spec.Realm.IdentityProviders == nil {
			kcr.Spec.Realm.IdentityProviders = []*keycloakTypes.KeycloakIdentityProvider{}
		}
		kcr.Spec.Realm.IdentityProviders = append(kcr.Spec.Realm.IdentityProviders, &keycloakTypes.KeycloakIdentityProvider{
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

	r.Log.Info("Syncing github identity provider to the keycloak realm")

	// Get an authenticated keycloak api client for the instance
	// keycloakFactory := r.LocalConfigKeycloakFactory{}
	// INVESTIGATE WHY KC CLIENTFACTORY WAS USED?
	authenticated, err := r.KeycloakClientFactory.AuthenticatedClient(*kc)
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

func getUserDiff(keycloakUsers []keycloakTypes.KeycloakAPIUser, openshiftUsers []usersv1.User, groups *usersv1.GroupList) (added []usersv1.User, deleted []keycloakTypes.KeycloakAPIUser) {
	for _, osUser := range openshiftUsers {
		if !kcContainsOsUser(keycloakUsers, osUser) && !userHelper.UserInExclusionGroup(osUser, groups) {
			added = append(added, osUser)
		}
	}

	for _, kcUser := range keycloakUsers {
		if !rhssocommon.OsUserInKc(openshiftUsers, kcUser) {
			deleted = append(deleted, kcUser)
		}
	}

	return added, deleted
}

func syncronizeWithOpenshiftUsers(ctx context.Context, keycloakUsers []keycloakTypes.KeycloakAPIUser, serverClient k8sclient.Client, ns string, installation *integreatlyv1alpha1.RHMI, logger l.Logger) ([]keycloakTypes.KeycloakAPIUser, error) {
	var openshiftUsers *usersv1.UserList
	var err error

	openshiftUsers, err = userHelper.GetUsersInActiveIDPs(ctx, serverClient, logger)
	if err != nil {
		return nil, errors.Wrap(err, "could not get users in active IDPs")
	}

	groups := &usersv1.GroupList{}
	err = serverClient.List(ctx, groups)
	if err != nil {
		return nil, err
	}

	added, deletedUsers := getUserDiff(keycloakUsers, openshiftUsers.Items, groups)

	keycloakUsers, err = rhssocommon.DeleteKeycloakUsers(keycloakUsers, deletedUsers, ns, ctx, serverClient)
	if err != nil {
		return nil, err
	}

	identitiesList := usersv1.IdentityList{}
	err = serverClient.List(ctx, &identitiesList)
	if err != nil {
		return nil, err
	}

	for _, osUser := range added {
		email := userHelper.GetUserEmailFromIdentity(ctx, serverClient, osUser, identitiesList)

		if email == "" {
			email = userHelper.SetUserNameAsEmail(osUser.Name)
		}

		newKeycloakUser := keycloakTypes.KeycloakAPIUser{
			Enabled:       true,
			UserName:      osUser.Name,
			EmailVerified: true,
			Email:         email,
			FederatedIdentities: []keycloakTypes.FederatedIdentity{
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
		keycloakUsers[index].ClientRoles = getKeycloakRoles(integreatlyv1alpha1.InstallationType(installation.Spec.Type))
	}

	return keycloakUsers, nil
}

func kcContainsOsUser(kcUsers []keycloakTypes.KeycloakAPIUser, osUser usersv1.User) bool {
	for _, kcu := range kcUsers {
		if kcu.UserName == osUser.Name {
			return true
		}
	}

	return false
}

func (r *Reconciler) createOrUpdateKeycloakUser(ctx context.Context, user keycloakTypes.KeycloakAPIUser, serverClient k8sclient.Client) error {
	// Create empty unstructured and attempt getting it from cluster
	kcUserUnstructured := dr.CreateUnstructuredWithGVK(keycloakTypes.KeycloakUserGroup, keycloakTypes.KeycloakUserKind, keycloakTypes.KeycloakUserVersion, "", r.Config.GetNamespace())

	err := serverClient.Get(context.TODO(), types.NamespacedName{Namespace: r.Config.GetNamespace(), Name: userHelper.GetValidGeneratedUserName(user)}, kcUserUnstructured)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			return err
		}
	}

	kcUserTyped, err := dr.ConvertKeycloakUserUnstructuredToTyped(*kcUserUnstructured)
	if err != nil {
		return err
	}

	// Retrieve keycloak user typed object desired state
	kcUserTypedDesired := createDesiredKeycloakUserTypedObject(user)
	if err != nil {
		return err
	}

	kcUserTyped.Spec = kcUserTypedDesired.Spec
	kcUserTyped.Labels = kcUserTypedDesired.Labels

	// Convert updated original to unstructed - this is done to createOrUpdate the original pre-update
	kcUserUnstructuredOriginalUpdated, err := dr.ConvertKeycloakUserTypedToUnstructured(*kcUserTyped)
	if err != nil {
		return err
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, kcUserUnstructured, func() error {
		kcUserUnstructured = kcUserUnstructuredOriginalUpdated
		return nil
	})
	if err != nil {
		return err
	}

	r.Log.Infof("Operation result", l.Fields{"keycloakuser": user.UserName, "result": or})

	return nil
}

func createDesiredKeycloakUserTypedObject(user keycloakTypes.KeycloakAPIUser) keycloakTypes.KeycloakUser {
	kcUser := keycloakTypes.KeycloakUser{}
	kcUser.Spec.RealmSelector = &metav1.LabelSelector{
		MatchLabels: GetInstanceLabels(),
	}
	kcUser.Labels = GetInstanceLabels()
	kcUser.Spec.User = user

	return kcUser
}

func (r *Reconciler) exportDashboard(ctx context.Context, apiClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	dashboard := "keycloak"

	ssoDB := &grafanav1alpha1.GrafanaDashboard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dashboard,
			Namespace: r.Config.GetNamespace(),
		},
	}

	err := apiClient.Get(ctx, k8sclient.ObjectKey{Name: ssoDB.Name, Namespace: ssoDB.Namespace}, ssoDB)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	observabilityConfig, err := r.ConfigManager.ReadObservability()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	observabilityDB := &grafanav1alpha1.GrafanaDashboard{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dashboard,
			Namespace: observabilityConfig.GetNamespace(),
		},
	}

	opRes, err := controllerutil.CreateOrUpdate(ctx, apiClient, observabilityDB, func() error {
		observabilityDB.Labels = ssoDB.Labels
		observabilityDB.Spec = ssoDB.Spec
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if opRes != controllerutil.OperationResultNone {
		r.Log.Infof("Operation result grafana ssoDB", l.Fields{"grafanaDashboard": observabilityDB.Name, "result": opRes})
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func GetKeycloakUsers(ctx context.Context, serverClient k8sclient.Client, ns string) ([]keycloakTypes.KeycloakAPIUser, error) {
	// Create empty unstructured and attempt getting it from cluster
	kcUsersListUnstructured := dr.CreateUnstructuredListWithGVK(keycloakTypes.KeycloakUserGroup, keycloakTypes.KeycloakUserKind, keycloakTypes.KeycloakUserListKind, keycloakTypes.KeycloakUserVersion, "", ns)

	listOptions := []k8sclient.ListOption{
		k8sclient.MatchingLabels(GetInstanceLabels()),
		k8sclient.InNamespace(ns),
	}

	err := serverClient.List(ctx, kcUsersListUnstructured, listOptions...)
	if err != nil {
		return nil, err
	}

	// map unstructured users to keycloakTypes.Users
	users, err := dr.ConvertKeycloakUsersUnstructuredToTyped(*kcUsersListUnstructured)
	if err != nil {
		return nil, err
	}

	mappedUsers := make([]keycloakTypes.KeycloakAPIUser, len(users.Items))
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
	authExecution, err := authenticated.FindAuthenticationExecutionForFlow(authFlowAlias, keycloakRealmName, func(execution *keycloakTypes.AuthenticationExecutionInfo) bool {
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

func getKeycloakRoles(installationType integreatlyv1alpha1.InstallationType) map[string][]string {
	var roles map[string][]string
	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installationType)) {
		roles = map[string][]string{
			"account": {
				"view-profile",
			},
		}
	} else {
		roles = map[string][]string{
			"account": {
				"manage-account",
				"view-profile",
			},
			"broker": {
				"read-token",
			},
		}
	}
	return roles
}

func GetInstanceLabels() map[string]string {
	return map[string]string{
		SSOLabelKey: SSOLabelValue,
	}
}
