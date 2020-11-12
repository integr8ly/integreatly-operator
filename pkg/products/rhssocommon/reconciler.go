package rhssocommon

import (
	"context"
	"fmt"
	"strings"

	monitoringv1alpha1 "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/products/monitoring"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	userHelper "github.com/integr8ly/integreatly-operator/pkg/resources/user"
	keycloakCommon "github.com/integr8ly/keycloak-client/pkg/common"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	appsv1 "github.com/openshift/api/apps/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"
	"github.com/sirupsen/logrus"
	k8sappsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	idpAlias        = "openshift-v4"
	manifestPackage = "integreatly-rhsso"
)

type Reconciler struct {
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	Installation  *integreatlyv1alpha1.RHMI
	Logger        *logrus.Entry
	Oauthv1Client oauthClient.OauthV1Interface
	APIURL        string
	*resources.Reconciler
	Recorder              record.EventRecorder
	KeycloakClientFactory keycloakCommon.KeycloakClientFactory
}

func NewReconciler(configManager config.ConfigReadWriter, mpm marketplace.MarketplaceInterface, installation *integreatlyv1alpha1.RHMI, logger *logrus.Entry, oauthv1Client oauthClient.OauthV1Interface, recorder record.EventRecorder, APIURL string, keycloakClientFactory keycloakCommon.KeycloakClientFactory) *Reconciler {
	return &Reconciler{
		ConfigManager:         configManager,
		mpm:                   mpm,
		Installation:          installation,
		Logger:                logger,
		Oauthv1Client:         oauthv1Client,
		APIURL:                APIURL,
		Reconciler:            resources.NewReconciler(mpm),
		Recorder:              recorder,
		KeycloakClientFactory: keycloakClientFactory,
	}
}

func SetNameSpaces(installation *integreatlyv1alpha1.RHMI, config *config.RHSSOCommon, namespace string) {
	if config.GetNamespace() == "" {
		config.SetNamespace(installation.Spec.NamespacePrefix + namespace)
	}

	if config.GetOperatorNamespace() == "" {
		if installation.Spec.OperatorsInProductNamespace {
			config.SetOperatorNamespace(config.GetNamespace())
		} else {
			config.SetOperatorNamespace(config.GetNamespace() + "-operator")
		}
	}
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sso",
			Namespace: ns,
		},
	}
}

func (r *Reconciler) CleanupKeycloakResources(ctx context.Context, inst *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client, ns string) (integreatlyv1alpha1.StatusPhase, error) {
	if inst.DeletionTimestamp == nil {
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	opts := &k8sclient.ListOptions{
		Namespace: ns,
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

	// Check users and clients have been removed before realms are removed
	// Refresh the user list
	err = serverClient.List(ctx, users, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	} else if len(users.Items) > 0 {
		logrus.Println("rhsso deletion of users in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	// Refresh the clients list
	err = serverClient.List(ctx, clients, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	} else if len(clients.Items) > 0 {
		logrus.Println("rhsso deletion of clients in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
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
		realm.SetFinalizers([]string{})
		err := serverClient.Update(ctx, &realm)
		if !k8serr.IsNotFound(err) && err != nil {
			logrus.Info("Error removing finalizer from Realm", err)
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	// Refresh the realm list
	err = serverClient.List(ctx, realms, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	} else if len(realms.Items) > 0 {
		logrus.Println("rhsso deletion of realms in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

// workaround: the keycloak operator creates a route with TLS passthrough config
// this should use the same valid certs as the cluster itself but for some reason the
// signing operator gives out self signed certs
// to circumvent this we create another keycloak route with edge termination
func (r *Reconciler) CreateKeycloakRoute(ctx context.Context, serverClient k8sclient.Client, config config.ConfigReadable, ssoCommon *config.RHSSOCommon) (integreatlyv1alpha1.StatusPhase, error) {
	// We need a route with edge termination to serve the correct cluster certificate
	edgeRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak-edge",
			Namespace: ssoCommon.GetNamespace(),
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
	r.Logger.Info(fmt.Sprintf("operation result of creating %v service was %v", edgeRoute.Name, or))

	if edgeRoute.Spec.Host == "" {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	// Override the keycloak host to the host of the edge route (instead of the
	// operator generated route)
	ssoCommon.SetHost(fmt.Sprintf("https://%v", edgeRoute.Spec.Host))
	err = r.ConfigManager.WriteConfig(config)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Error writing to config in rhsso reconciler: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) SetupOpenshiftIDP(ctx context.Context, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI, sso config.RHSSOInterface, kcr *keycloak.KeycloakRealm, redirectUris []string) error {

	clientSecret, err := r.getClientSecret(ctx, serverClient, sso)
	if err != nil {
		return err
	}

	oauthClient := &oauthv1.OAuthClient{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.GetOAuthClientName(sso),
		},
		Secret:       clientSecret,
		RedirectURIs: redirectUris,
		GrantMethod:  oauthv1.GrantHandlerAuto,
	}

	_, err = r.ReconcileOauthClient(ctx, installation, oauthClient, serverClient)
	if err != nil {
		return err
	}

	if !ContainsIdentityProvider(kcr.Spec.Realm.IdentityProviders, idpAlias) {
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
				"baseUrl":         "https://" + strings.Replace(r.Installation.Spec.RoutingSubdomain, "apps", "api", 1) + ":6443",
				"clientId":        r.GetOAuthClientName(sso),
				"disableUserInfo": "",
				"clientSecret":    clientSecret,
				"defaultScope":    "user:full",
				"useJwksUrl":      "true",
			},
		})
	}

	return nil
}

// Sync the secret to cover the scenario where the secret is changed through the Keycloak GUI.
func (r *Reconciler) SyncOpenshiftIDPClientSecret(ctx context.Context, serverClient k8sclient.Client, authenticated keycloakCommon.KeycloakInterface, sso config.RHSSOInterface, keycloakRealmName string) error {

	clientSecret, err := r.getClientSecret(ctx, serverClient, sso)
	if err != nil {
		return err
	}

	idp, err := authenticated.GetIdentityProvider(idpAlias, keycloakRealmName)
	if err != nil {
		r.Logger.Errorf("failed to get identity provider via keycloak api %v", err)
		return fmt.Errorf("failed to get identity provider via keycloak api %w", err)
	}

	if idp.Config == nil {
		idp.Config = map[string]string{}
	}

	idp.Config["clientSecret"] = clientSecret
	err = authenticated.UpdateIdentityProvider(idp, keycloakRealmName)
	if err != nil {
		r.Logger.Errorf("Unable to update Identity Provider %v", err)
		return fmt.Errorf("Unable to update Identity Provider %w", err)
	}

	r.Logger.Infof("Updated Identity Provider, %s, with client Secret: ", idpAlias)

	return nil
}

func (r *Reconciler) getClientSecret(ctx context.Context, serverClient k8sclient.Client, sso config.RHSSOInterface) (string, error) {
	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.ConfigManager.GetOauthClientsSecretName(),
		},
	}

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: oauthClientSecrets.Name, Namespace: r.ConfigManager.GetOperatorNamespace()}, oauthClientSecrets)
	if err != nil {
		r.Logger.Errorf("Could not find %s Secret: %v", oauthClientSecrets.Name, err)
		return "", fmt.Errorf("Could not find %s Secret: %w", oauthClientSecrets.Name, err)
	}

	clientSecretBytes, ok := oauthClientSecrets.Data[string(sso.GetProductName())]
	if !ok {
		r.Logger.Errorf("Could not find %s key in %s Secret: %v", string(sso.GetProductName()), oauthClientSecrets.Name, err)
		return "", fmt.Errorf("Could not find %s key in %s Secret: %w", string(sso.GetProductName()), oauthClientSecrets.Name, err)
	}
	return string(clientSecretBytes), nil
}

func (r *Reconciler) GetOAuthClientName(sso config.RHSSOInterface) string {
	return r.Installation.Spec.NamespacePrefix + string(sso.GetProductName())
}

func ContainsIdentityProvider(providers []*keycloak.KeycloakIdentityProvider, alias string) bool {
	for _, p := range providers {
		if p.Alias == alias {
			return true
		}
	}
	return false
}

func (r *Reconciler) ReconcileCloudResources(dbPRefix string, defaultNamespace string, ssoType string, config *config.RHSSOCommon, ctx context.Context, installation *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.Logger.Info("Reconciling Keycloak external database instance")
	postgresName := fmt.Sprintf("%s%s", dbPRefix, installation.Name)
	postgres, err := resources.ReconcileRHSSOPostgresCredentials(ctx, installation, serverClient, postgresName, config.GetNamespace(), defaultNamespace)

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile database credentials secret while provisioning %s: %w", ssoType, err)
	}

	// at this point it should be ok to create the failed alert.
	if postgres != nil {
		// reconcile postgres alerts
		phase, err := resources.ReconcilePostgresAlerts(ctx, serverClient, installation, postgres)
		productName := postgres.Labels["productName"]
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile postgres alerts for %s: %w", productName, err)
		}
		if phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, nil
		}
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	// postgres provisioning is still in progress
	return integreatlyv1alpha1.PhaseAwaitingCloudResources, nil
}

func (r *Reconciler) PreUpgradeBackupsExecutor(resourceName string) backup.BackupExecutor {
	if r.Installation.Spec.UseClusterStorage != "false" {
		return backup.NewNoopBackupExecutor()
	}

	return backup.NewAWSBackupExecutor(
		r.Installation.Namespace,
		resourceName,
		backup.PostgresSnapshotType,
	)
}

func (r *Reconciler) ReconcileSubscription(ctx context.Context, serverClient k8sclient.Client, inst *integreatlyv1alpha1.RHMI, productNamespace string, operatorNamespace string, resourceName string) (integreatlyv1alpha1.StatusPhase, error) {
	target := marketplace.Target{
		Pkg:       constants.RHSSOSubscriptionName,
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
		r.PreUpgradeBackupsExecutor(resourceName),
		serverClient,
		catalogSourceReconciler,
	)
}

func (r *Reconciler) ReconcileStatefulSet(ctx context.Context, serverClient k8sclient.Client, config *config.RHSSOCommon) (integreatlyv1alpha1.StatusPhase, error) {
	statefulSet := &k8sappsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak",
			Namespace: config.GetNamespace(),
		},
	}

	// Include the PodPriority mutation only if the install type is Managed API
	mutatePodPriority := resources.NoopMutate
	if r.Installation.Spec.Type == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		mutatePodPriority = resources.MutatePodPriority(r.Installation.Spec.PriorityClassName)
	}

	return resources.UpdatePodTemplateIfExists(
		ctx,
		serverClient,
		resources.SelectFromStatefulSet,
		resources.AllMutationsOf(
			resources.MutateMultiAZAntiAffinity(ctx, serverClient, "app"),
			resources.MutateZoneTopologySpreadConstraints("app"),
			mutatePodPriority,
		),
		statefulSet,
	)
}

func DeleteKeycloakUsers(allKcUsers []keycloak.KeycloakAPIUser, deletedUsers []keycloak.KeycloakAPIUser, ns string, ctx context.Context, serverClient k8sclient.Client) ([]keycloak.KeycloakAPIUser, error) {

	for _, delUser := range deletedUsers {

		if delUser.UserName == "" {
			continue
		}

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

func OsUserInKc(osUsers []usersv1.User, kcUser keycloak.KeycloakAPIUser) bool {
	for _, osu := range osUsers {
		if osu.Name == kcUser.UserName {
			return true
		}
	}

	return false
}

func (r *Reconciler) HandleProgressPhase(ctx context.Context, serverClient k8sclient.Client, keycloakName string, keycloakRealmName string, config config.ConfigReadable, ssoCommon *config.RHSSOCommon, rhssoVersion string, operatorVersion string) (integreatlyv1alpha1.StatusPhase, error) {
	kc := &keycloak.Keycloak{}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: keycloakName, Namespace: config.GetNamespace()}, kc)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	// The keycloak operator does not set the product version currently - should fetch from KeyCloak.Status.Version when fixed
	ssoCommon.SetProductVersion(rhssoVersion)
	// The Keycloak Operator doesn't currently set the operator version
	ssoCommon.SetOperatorVersion(operatorVersion)
	err = r.ConfigManager.WriteConfig(config)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	r.Logger.Info("checking ready status for rhsso")
	kcr := &keycloak.KeycloakRealm{}

	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: keycloakRealmName, Namespace: config.GetNamespace()}, kcr)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get keycloak realm custom resource: %w", err)
	}

	if kcr.Status.Phase == keycloak.PhaseReconciling {
		err = r.exportConfig(ctx, serverClient, keycloakName, keycloakRealmName, config, ssoCommon)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to write rhsso config: %w", err)
		}

		logrus.Infof("Keycloak has successfully processed the keycloakRealm")
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	r.Logger.Infof("KeycloakRealm status phase is: %s", kcr.Status.Phase)
	return integreatlyv1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) exportConfig(ctx context.Context, serverClient k8sclient.Client, keycloakName string, keycloakRealmName string, config config.ConfigReadable, ssoCommon *config.RHSSOCommon) error {
	kc := &keycloak.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: config.GetNamespace(),
		},
	}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: keycloakName, Namespace: config.GetNamespace()}, kc)
	if err != nil {
		return fmt.Errorf("Could not retrieve keycloak custom resource for keycloak config: %w", err)
	}

	ssoCommon.SetRealm(keycloakRealmName)

	return nil
}

func (r *Reconciler) ReconcileBlackboxTargets(ctx context.Context, installation *integreatlyv1alpha1.RHMI, client k8sclient.Client, targetName string, url string, service string) (integreatlyv1alpha1.StatusPhase, error) {
	cfg, err := r.ConfigManager.ReadMonitoring()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("errror reading monitoring config: %w", err)
	}

	err = monitoring.CreateBlackboxTarget(ctx, targetName, monitoringv1alpha1.BlackboxtargetData{
		Url:     url,
		Service: service,
	}, cfg, installation, client)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create rhsso blackbox target: %w", err)
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}
