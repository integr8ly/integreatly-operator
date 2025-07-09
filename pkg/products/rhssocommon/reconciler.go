package rhssocommon

import (
	"context"
	"fmt"

	croType "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/pkg/resources/k8s"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	userHelper "github.com/integr8ly/integreatly-operator/pkg/resources/user"
	keycloak "github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"
	keycloakCommon "github.com/integr8ly/keycloak-client/pkg/common"
	appsv1 "github.com/openshift/api/apps/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	k8sappsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	idpAlias        = "openshift-v4"
	podMonitorName  = "keycloak-pod-monitor"
	keycloakPDBName = "keycloak"
)

const (
	KeycloakMetricsExtension = "https://github.com/integr8ly/keycloak-metrics-spi/releases/download/2.5.3/keycloak-metrics-spi.jar"
)

type Reconciler struct {
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	Installation  *integreatlyv1alpha1.RHMI
	Log           l.Logger
	Oauthv1Client oauthClient.OauthV1Interface
	APIURL        string
	*resources.Reconciler
	Recorder              record.EventRecorder
	KeycloakClientFactory keycloakCommon.KeycloakClientFactory
}

func NewReconciler(configManager config.ConfigReadWriter, mpm marketplace.MarketplaceInterface, installation *integreatlyv1alpha1.RHMI, logger l.Logger, oauthv1Client oauthClient.OauthV1Interface, recorder record.EventRecorder, APIURL string, keycloakClientFactory keycloakCommon.KeycloakClientFactory, productDeclaration marketplace.ProductDeclaration) *Reconciler {
	return &Reconciler{
		ConfigManager:         configManager,
		mpm:                   mpm,
		Installation:          installation,
		Log:                   logger,
		Oauthv1Client:         oauthv1Client,
		APIURL:                APIURL,
		Reconciler:            resources.NewReconciler(mpm).WithProductDeclaration(productDeclaration),
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

func (r *Reconciler) GetPreflightObject(ns string) k8sclient.Object {
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

	keycloakCRD := &apiextensionv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "keycloaks.keycloak.org",
		},
	}
	crdExists, err := k8s.Exists(ctx, serverClient, keycloakCRD)
	if err != nil {
		r.Log.Error("Error checking Keycloak CRD existence: ", nil, err)
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if !crdExists {
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	// Delete all users
	users := &keycloak.KeycloakUserList{}
	err = serverClient.List(ctx, users, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	for i := range users.Items {
		user := users.Items[i]
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
	for i := range clients.Items {
		client := clients.Items[i]
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
	}
	if len(users.Items) > 0 {
		r.Log.Info("rhsso deletion of users in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	// Refresh the clients list
	err = serverClient.List(ctx, clients, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if len(clients.Items) > 0 {
		r.Log.Info("rhsso deletion of clients in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	// Delete all realms
	realms := &keycloak.KeycloakRealmList{}
	err = serverClient.List(ctx, realms, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, nil
	}
	for i := range realms.Items {
		realm := realms.Items[i]
		err = serverClient.Delete(ctx, &realm)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	// Refresh the realm list
	err = serverClient.List(ctx, realms, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	// Delete all realm finalizers
	for i := range realms.Items {
		realm := realms.Items[i]
		realm.SetFinalizers([]string{})
		err = serverClient.Update(ctx, &realm)
		if err != nil && !k8serr.IsNotFound(err) {
			r.Log.Error("Error removing finalizer from Realm", nil, err)
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	// Refresh the realm list
	err = serverClient.List(ctx, realms, opts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if len(realms.Items) > 0 {
		r.Log.Info("rhsso deletion of realms in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

// CreateKeycloakRoute
// workaround: the keycloak operator creates a route with TLS passthrough config
// this should use the same valid certs as the cluster itself but for some reason the
// signing operator gives out self signed certs
// to circumvent this we create another keycloak route with edge termination
func (r *Reconciler) CreateKeycloakRoute(ctx context.Context, serverClient k8sclient.Client, config config.ConfigReadable, ssoCommon *config.RHSSOCommon, routeName string) (integreatlyv1alpha1.StatusPhase, error) {
	keycloakRoute := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeName,
			Namespace: ssoCommon.GetNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, keycloakRoute, func() error {
		host := keycloakRoute.Spec.Host
		keycloakRoute.Spec = routev1.RouteSpec{
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
		r.Log.Error("Error creating keycloak edge route", nil, err)
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating keycloak edge route: %w", err)
	}
	r.Log.Infof("Operation Result creating route", l.Fields{"service": keycloakRoute.Name, "result": or})

	if keycloakRoute.Spec.Host == "" {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	// Override the keycloak host to the host of the edge route (instead of the
	// operator generated route)
	ssoCommon.SetHost(fmt.Sprintf("https://%v", keycloakRoute.Spec.Host))
	err = r.ConfigManager.WriteConfig(config)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error writing to config in rhsso reconciler: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) SetupOpenshiftIDP(ctx context.Context, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI, sso config.RHSSOInterface, kcr *keycloak.KeycloakRealm, redirectUris []string, tenant string) error {
	var (
		clientSecret string
		clientId     string
		err          error
	)

	if tenant != "" {
		clientSecret, err = r.getTenantClientSecret(ctx, serverClient, tenant)
		if err != nil {
			return err
		}
		clientId = tenant
	} else {
		clientSecret, err = r.getClientSecret(ctx, serverClient, sso)
		if err != nil {
			return err
		}
		clientId = r.GetOAuthClientName(sso)
	}

	oauthClient := &oauthv1.OAuthClient{
		ObjectMeta: metav1.ObjectMeta{
			Name: clientId,
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
		r.Log.Info("Adding keycloak realm client")
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
				"baseUrl":         r.Installation.Spec.APIServer,
				"clientId":        clientId,
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
		r.Log.Error("Failed to get identity provider via keycloak api", nil, err)
		return fmt.Errorf("failed to get identity provider via keycloak api %w", err)
	}

	if idp.Config == nil {
		idp.Config = map[string]string{}
	}

	idp.Config["clientSecret"] = clientSecret
	err = authenticated.UpdateIdentityProvider(idp, keycloakRealmName)
	if err != nil {
		r.Log.Error("Unable to update Identity Provider", nil, err)
		return fmt.Errorf("Unable to update Identity Provider %w", err)
	}

	r.Log.Infof("Updated Identity Provider with client Secret: ", l.Fields{"idpAlias": idpAlias})

	return nil
}

func (r *Reconciler) getTenantClientSecret(ctx context.Context, serverClient k8sclient.Client, tenant string) (string, error) {
	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tenant-oauth-client-secrets",
		},
	}

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: oauthClientSecrets.Name, Namespace: r.ConfigManager.GetOperatorNamespace()}, oauthClientSecrets)
	if err != nil {
		r.Log.Error("Could not find secret", l.Fields{"secret": oauthClientSecrets.Name}, err)
		return "", fmt.Errorf("Could not find %s Secret: %w", oauthClientSecrets.Name, err)
	}

	clientSecretBytes, ok := oauthClientSecrets.Data[string(tenant)]
	if !ok {
		r.Log.Error("Could not find tenant key in secret", l.Fields{"tenant": string(tenant), "secret": oauthClientSecrets.Name}, err)
		return "", fmt.Errorf("Could not find %s key in %s Secret: %w", string(tenant), oauthClientSecrets.Name, err)
	}
	return string(clientSecretBytes), nil
}

func (r *Reconciler) getClientSecret(ctx context.Context, serverClient k8sclient.Client, sso config.RHSSOInterface) (string, error) {
	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.ConfigManager.GetOauthClientsSecretName(),
		},
	}

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: oauthClientSecrets.Name, Namespace: r.ConfigManager.GetOperatorNamespace()}, oauthClientSecrets)
	if err != nil {
		r.Log.Error("Could not find secret", l.Fields{"secret": oauthClientSecrets.Name}, err)
		return "", fmt.Errorf("Could not find %s Secret: %w", oauthClientSecrets.Name, err)
	}

	clientSecretBytes, ok := oauthClientSecrets.Data[string(sso.GetProductName())]
	if !ok {
		r.Log.Error("Could not find product key in secret", l.Fields{"product": string(sso.GetProductName()), "secret": oauthClientSecrets.Name}, err)
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
	r.Log.Info("Reconciling Keycloak external database instance")
	postgresName := fmt.Sprintf("%s%s", dbPRefix, installation.Name)
	var snapshotFrequency, snapshotRetention croType.Duration
	postgres, err := resources.ReconcileRHSSOPostgresCredentials(ctx, installation, serverClient, postgresName, config.GetNamespace(), defaultNamespace, snapshotFrequency, snapshotRetention)

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile database credentials secret while provisioning %s: %w", ssoType, err)
	}

	// at this point it should be ok to create the failed alert.
	if postgres != nil {
		// reconcile postgres alerts
		phase, err := resources.ReconcilePostgresAlerts(ctx, serverClient, installation, postgres, r.Log)
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
		SubscriptionName: constants.RHSSOSubscriptionName,
		Namespace:        operatorNamespace,
	}

	catalogSourceReconciler, err := r.GetProductDeclaration().PrepareTarget(
		r.Log,
		serverClient,
		marketplace.CatalogSourceName,
		&target,
	)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return r.Reconciler.ReconcileSubscription(
		ctx,
		target,
		[]string{productNamespace},
		r.PreUpgradeBackupsExecutor(resourceName),
		serverClient,
		catalogSourceReconciler,
		r.Log,
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
	mutatePodPriority := resources.MutatePodPriority(r.Installation.Spec.PriorityClassName)

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
		if kcUser.Name == "" {
			return nil, fmt.Errorf("failed to get valid generated username")
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

	r.Log.Info("checking ready status for rhsso")
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

		r.Log.Info("Keycloak has successfully processed the keycloakRealm")
		return integreatlyv1alpha1.PhaseCompleted, nil
	}
	r.Log.Infof("KeycloakRealm status %s", l.Fields{"phaseStatus": kcr.Status.Phase})
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

func (r *Reconciler) RemovePodMonitors(ctx context.Context, client k8sclient.Client, config config.ConfigReadable) (integreatlyv1alpha1.StatusPhase, error) {

	podMonitor := &monitoringv1.PodMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podMonitorName,
			Namespace: config.GetNamespace(),
		},
	}

	err := client.Get(ctx, k8sclient.ObjectKey{Name: podMonitorName, Namespace: config.GetNamespace()}, podMonitor)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseCompleted, nil
		}
		return integreatlyv1alpha1.PhaseFailed, err
	}

	err = client.Delete(ctx, podMonitor)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) ExportAlerts(ctx context.Context, apiClient k8sclient.Client, productName string, productNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	ssoAlert := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak",
			Namespace: productNamespace,
		},
	}

	err := apiClient.Get(ctx, k8sclient.ObjectKey{Name: ssoAlert.Name, Namespace: ssoAlert.Namespace}, ssoAlert)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	if ssoAlert.Spec.Groups == nil {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	for groupIdx, alertGroup := range ssoAlert.Spec.Groups {
		if alertGroup.Name == "general.rules" {
			for idx, alertRule := range alertGroup.Rules {
				if alertRule.Alert == "KeycloakInstanceNotAvailable" {
					ssoAlert.Spec.Groups[groupIdx].Rules = removeRule(ssoAlert.Spec.Groups[0].Rules, idx)
				}
			}
		}
	}

	alertToMove := &monv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      productName,
			Namespace: config.GetOboNamespace(r.Installation.Namespace),
		},
	}
	opRes, err := controllerutil.CreateOrUpdate(ctx, apiClient, alertToMove, func() error {
		var destinationRouleGroupArray []monv1.RuleGroup
		for _, ruleGroup := range ssoAlert.Spec.Groups {
			var destinationRules []monv1.Rule
			for _, rule := range ruleGroup.Rules {
				var destinationRule monv1.Rule
				destinationRule.Alert = rule.Alert
				destinationRule.Annotations = rule.Annotations
				destinationRule.Expr = rule.Expr
				destinationRule.For = monv1.Duration(rule.For)
				destinationRule.Record = rule.Record
				destinationRule.Labels = rule.Labels
				destinationRules = append(destinationRules, destinationRule)
			}
			var destinationRouleGroup monv1.RuleGroup
			destinationRouleGroup.Name = ruleGroup.Name
			destinationRouleGroup.PartialResponseStrategy = ruleGroup.PartialResponseStrategy
			destinationRouleGroup.Rules = destinationRules
			destinationRouleGroup.Interval = monv1.Duration(ruleGroup.Interval)
			destinationRouleGroupArray = append(destinationRouleGroupArray, destinationRouleGroup)
		}

		alertToMove.Labels = ssoAlert.Labels
		alertToMove.Spec.Groups = destinationRouleGroupArray
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if opRes != controllerutil.OperationResultNone {
		r.Log.Infof("Operation result export PrometheusRule", l.Fields{"PrometheusRule": alertToMove.Name, "result": opRes})
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

// ReconcileCSVEnvVars will take a keycloak-operator CSV and a map of env vars to update or create
func (r *Reconciler) ReconcileCSVEnvVars(csv *operatorsv1alpha1.ClusterServiceVersion, envVars map[string]string) (*operatorsv1alpha1.ClusterServiceVersion, bool, error) {
	updated := false
	for deploymentIndex, deployment := range csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs {
		if deployment.Name != "rhsso-operator" {
			continue
		}
		deploymentEnvVars := deployment.Spec.Template.Spec.Containers[0].Env
		for envVarIndex, envVar := range deploymentEnvVars {
			if newValue, ok := envVars[envVar.Name]; ok {
				delete(envVars, envVar.Name)
				if deploymentEnvVars[envVarIndex].Value != newValue {
					updated = true
					deploymentEnvVars[envVarIndex].Value = newValue
				}
			}
		}

		//any left are new entries
		for name, value := range envVars {
			updated = true
			deploymentEnvVars = append(deploymentEnvVars, corev1.EnvVar{Name: name, Value: value})
		}

		csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs[deploymentIndex].Spec.Template.Spec.Containers[0].Env = deploymentEnvVars
		break // no need to iterate any further
	}
	return csv, updated, nil
}

func removeRule(slice []monitoringv1.Rule, s int) []monitoringv1.Rule {
	return append(slice[:s], slice[s+1:]...)
}

func (r *Reconciler) ReconcilePodDisruptionBudget(ctx context.Context, apiClient k8sclient.Client, productNamespace string) (integreatlyv1alpha1.StatusPhase, error) {

	pdb := &policyv1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodDisruptionBudget",
			APIVersion: "policy/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakPDBName,
			Namespace: productNamespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, apiClient, pdb, func() error {
		pdb.Spec = policyv1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"component": keycloakPDBName},
			},
			MaxUnavailable: &intstr.IntOrString{IntVal: 1},
		}

		pdb.ObjectMeta.Labels = map[string]string{
			"app": keycloakPDBName,
		}

		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
