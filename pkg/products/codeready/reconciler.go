package codeready

import (
	"context"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	"github.com/integr8ly/integreatly-operator/pkg/resources/events"

	"github.com/integr8ly/integreatly-operator/pkg/products/rhsso"
	"github.com/integr8ly/integreatly-operator/version"

	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"

	chev1 "github.com/eclipse/che-operator/pkg/apis/org/v1"

	crov1alpha1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/client"

	monitoringv1alpha1 "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/products/monitoring"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "codeready-workspaces"
	defaultClientName            = "che-client"
	defaultCheClusterName        = "rhmi-cluster"
	manifestPackage              = "integreatly-codeready-workspaces"
)

type Reconciler struct {
	Config        *config.CodeReady
	ConfigManager config.ConfigReadWriter
	installation  *integreatlyv1alpha1.RHMI
	extraParams   map[string]string
	mpm           marketplace.MarketplaceInterface
	*resources.Reconciler
	recorder record.EventRecorder
	log      l.Logger
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, log l.Logger) (*Reconciler, error) {
	config, err := configManager.ReadCodeReady()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve che config: %w", err)
	}
	if config.GetNamespace() == "" {
		config.SetNamespace(installation.Spec.NamespacePrefix + defaultInstallationNamespace)
	}
	if config.GetOperatorNamespace() == "" {
		if installation.Spec.OperatorsInProductNamespace {
			config.SetOperatorNamespace(config.GetNamespace())
		} else {
			config.SetOperatorNamespace(config.GetNamespace() + "-operator")
		}
	}

	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
		installation:  installation,
		mpm:           mpm,
		log:           log,
		Reconciler:    resources.NewReconciler(mpm),
		recorder:      recorder,
	}, nil
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codeready",
			Namespace: ns,
		},
	}
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return version.VerifyProductAndOperatorVersion(
		installation.Status.Stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductCodeReadyWorkspaces],
		string(integreatlyv1alpha1.VersionCodeReadyWorkspaces),
		string(integreatlyv1alpha1.OperatorVersionCodeReadyWorkspaces),
	)
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	operatorNamespace := r.Config.GetOperatorNamespace()
	productNamespace := r.Config.GetNamespace()
	phase, err := r.ReconcileFinalizer(ctx, serverClient, r.installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		phase, err := resources.RemoveNamespace(ctx, r.installation, serverClient, productNamespace, r.log)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		phase, err = resources.RemoveNamespace(ctx, installation, serverClient, operatorNamespace, r.log)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		return integreatlyv1alpha1.PhaseCompleted, nil
	}, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, operatorNamespace, r.installation, serverClient, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", operatorNamespace), err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, productNamespace, r.installation, serverClient, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", productNamespace), err)
		return phase, err
	}

	phase, err = r.reconcileSubscription(ctx, serverClient, installation, productNamespace, operatorNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.CodeReadySubscriptionName), err)
		return phase, err
	}

	phase, err = r.reconcileExternalDatasources(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile external data sources", err)
		return phase, err
	}

	phase, err = r.reconcileCheCluster(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile che cluster", err)
		return phase, err
	}

	phase, err = r.reconcileKeycloakClient(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile keycloak client", err)
		return phase, err
	}

	phase, err = r.reconcileBlackboxTargets(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile blackbox targets", err)
		return phase, err
	}

	phase, err = r.reconcileBackups(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile backups", err)
		return phase, err
	}

	phase, err = r.newAlertsReconciler().ReconcileAlerts(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile alerts", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	r.log.Infof("Reconciled successfully", l.Fields{"productName": r.Config.GetProductName()})
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileExternalDatasources(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Reconciling external datastore")
	ns := r.installation.Namespace

	// setup postgres custom resource
	postgresName := fmt.Sprintf("%s%s", constants.CodeReadyPostgresPrefix, r.installation.Name)
	postgres, err := croUtil.ReconcilePostgres(ctx, serverClient, defaultInstallationNamespace, r.installation.Spec.Type, croUtil.TierProduction, postgresName, ns, postgresName, ns, func(cr metav1.Object) error {
		owner.AddIntegreatlyOwnerAnnotations(cr, r.installation)
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile postgres: %w", err)
	}

	// reconcile postgres alerts
	phase, err := resources.ReconcilePostgresAlerts(ctx, serverClient, r.installation, postgres, r.log)
	productName := postgres.Labels["productName"]
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile postgres alerts for %s: %w", productName, err)
	}
	if phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, nil
	}

	// get the secret created by the cloud resources operator
	croSec := &corev1.Secret{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: postgres.Status.SecretRef.Name, Namespace: postgres.Status.SecretRef.Namespace}, croSec)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get postgres credential secret: %w", err)
	}

	// create backup secret
	r.log.Info("Reconciling codeready backup secret")
	cheBackUpSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Config.GetPostgresBackupSecretName(),
			Namespace: r.Config.GetNamespace(),
		},
		Data: map[string][]byte{},
	}

	// create or update backup secret
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, cheBackUpSecret, func() error {
		cheBackUpSecret.Data["POSTGRES_HOST"] = croSec.Data["host"]
		cheBackUpSecret.Data["POSTGRES_USERNAME"] = croSec.Data["username"]
		cheBackUpSecret.Data["POSTGRES_PASSWORD"] = croSec.Data["password"]
		cheBackUpSecret.Data["POSTGRES_DATABASE"] = croSec.Data["database"]
		cheBackUpSecret.Data["POSTGRES_PORT"] = croSec.Data["port"]
		cheBackUpSecret.Data["POSTGRES_VERSION"] = []byte("10")
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create or update %s connection secret: %w", r.Config.GetPostgresBackupSecretName(), err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileCheCluster(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	kcConfig, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not retrieve keycloak config: %w", err)
	}
	if err = kcConfig.Validate(); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("keycloak config is not valid: %w", err)
	}

	r.log.Infof("Creating required custom resources", l.Fields{"namespace": r.Config.GetNamespace()})

	kcRealm := &keycloak.KeycloakRealm{}
	key := k8sclient.ObjectKey{Name: kcConfig.GetRealm(), Namespace: kcConfig.GetNamespace()}
	err = serverClient.Get(ctx, key, kcRealm)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not retrieve: %+v: %w", key, err)
	}

	cheCluster, err := r.createCheCluster(ctx, kcConfig, kcRealm, serverClient)

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// che cluster hasn't reconciled yet
	if cheCluster == nil {
		return integreatlyv1alpha1.PhaseAwaitingComponents, nil
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

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

func (r *Reconciler) reconcileBackups(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	backupConfig := resources.BackupConfig{
		Namespace:     r.Config.GetNamespace(),
		Name:          "codeready",
		BackendSecret: resources.BackupSecretLocation{Name: r.Config.GetBackupsSecretName(), Namespace: r.Config.GetNamespace()},
		Components: []resources.BackupComponent{
			{
				Name:     "codeready-pv-backup",
				Type:     "codeready_pv",
				Schedule: r.Config.GetBackupSchedule(),
			},
		},
	}
	if err := resources.ReconcileBackup(ctx, serverClient, backupConfig, r.ConfigManager, r.log); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create backups for codeready: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) handleProgressPhase(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("checking that checluster custom resource is marked as available")

	// retrive the checluster so we can use its URL for redirect and web origins in the keycloak client
	cheCluster := &chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultCheClusterName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: defaultCheClusterName, Namespace: r.Config.GetNamespace()}, cheCluster)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not retrieve checluster for keycloak client update: %w", err)
	}
	if cheCluster.Status.CheClusterRunning != "Available" {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileKeycloakClient(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("checking keycloak client exists for che")
	kcConfig, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not retrieve keycloak config: %w", err)
	}
	if err = kcConfig.Validate(); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("keycloak config is not valid: %w", err)
	}

	// retrive the checluster so we can use its URL for redirect and web origins in the keycloak client
	cheCluster := &chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultCheClusterName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: defaultCheClusterName, Namespace: r.Config.GetNamespace()}, cheCluster)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not retrieve checluster for keycloak client update: %w", err)
	}

	cheURL := cheCluster.Status.CheURL
	if cheURL == "" {
		//still waiting for the Che URL, so exit codeready reconciling now and try again
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	if r.Config.GetHost() != cheURL {
		r.Config.SetHost(cheURL)
		if err = r.ConfigManager.WriteConfig(r.Config); err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not write che configuration: %w", err)
		}
	}

	kcClient := &keycloak.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultClientName,
			Namespace: kcConfig.GetNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, kcClient, func() error {
		kcClient.Spec = getKeycloakClientSpec(cheURL)
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not create/update codeready keycloak client: %w", err)
	}

	r.log.Infof("Operation result for keycloakclient", l.Fields{"kcClientName": kcClient.Name, "result": string(or)})
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileBlackboxTargets(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	cfg, err := r.ConfigManager.ReadMonitoring()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error reading monitoring config: %w", err)
	}

	err = monitoring.CreateBlackboxTarget(ctx, "integreatly-codeready", monitoringv1alpha1.BlackboxtargetData{
		Url:     r.Config.GetHost(),
		Service: "codeready-ui",
	}, cfg, r.installation, client)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating codeready blackbox target: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) createCheCluster(ctx context.Context, kcCfg *config.RHSSO, kr *keycloak.KeycloakRealm, serverClient k8sclient.Client) (*chev1.CheCluster, error) {
	selfSignedCerts := r.installation.Spec.SelfSignedCerts

	// get postgres cloud resource cr
	pcr := &crov1alpha1.Postgres{}
	postgresName := fmt.Sprintf("codeready-postgres-%s", r.installation.Name)
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: postgresName, Namespace: r.installation.Namespace}, pcr)
	if err != nil {
		return nil, fmt.Errorf("failed to find postgres custom resource: %w", err)
	}

	// get the postgres cloud resources operator cr
	croSec := &corev1.Secret{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: pcr.Status.SecretRef.Name, Namespace: pcr.Status.SecretRef.Namespace}, croSec)
	if err != nil {
		return nil, fmt.Errorf("failed to get postgres credential secret: %w", err)
	}

	cheCluster := &chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultCheClusterName,
			Namespace: r.Config.GetNamespace(),
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, cheCluster, func() error {
		cheCluster.Name = defaultCheClusterName
		cheCluster.Namespace = r.Config.GetNamespace()
		cheCluster.APIVersion = fmt.Sprintf(
			"%s/%s",
			chev1.SchemeGroupVersion.Group,
			chev1.SchemeGroupVersion.Version,
		)
		cheCluster.Kind = "CheCluster"
		cheCluster.Spec.Server.CheFlavor = "codeready"
		cheCluster.Spec.Server.TlsSupport = true
		cheCluster.Spec.Server.SelfSignedCert = selfSignedCerts
		cheCluster.Spec.Database.ExternalDb = true
		cheCluster.Spec.Database.ChePostgresDb = string(croSec.Data["database"])
		cheCluster.Spec.Database.ChePostgresPassword = string(croSec.Data["password"])
		cheCluster.Spec.Database.ChePostgresPort = string(croSec.Data["port"])
		cheCluster.Spec.Database.ChePostgresUser = string(croSec.Data["username"])
		cheCluster.Spec.Database.ChePostgresHostName = string(croSec.Data["host"])
		cheCluster.Spec.Auth.OpenShiftoAuth = false
		cheCluster.Spec.Auth.ExternalIdentityProvider = true
		cheCluster.Spec.Auth.IdentityProviderURL = kcCfg.GetHost()
		cheCluster.Spec.Auth.IdentityProviderRealm = kr.Name
		cheCluster.Spec.Auth.IdentityProviderClientId = defaultClientName
		cheCluster.Spec.Storage.PvcStrategy = "per-workspace"
		cheCluster.Spec.Storage.PvcClaimSize = "1Gi"
		cheCluster.Spec.Storage.PreCreateSubPaths = true

		owner.AddIntegreatlyOwnerAnnotations(cheCluster, r.installation)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create che cluster resource: %w", err)
	}

	return cheCluster, nil
}

func (r *Reconciler) preUpgradeBackupExecutor() backup.BackupExecutor {
	pvBackup := backup.NewCronJobBackupExecutor(
		"codeready-pv-backup",
		r.Config.GetNamespace(),
		"codeready-preupgrade-pv-backup",
	)

	if r.installation.Spec.UseClusterStorage != "false" {
		return pvBackup
	}

	return backup.NewConcurrentBackupExecutor(
		pvBackup,
		backup.NewAWSBackupExecutor(
			r.installation.Namespace,
			"codeready-postgres-rhmi",
			backup.PostgresSnapshotType,
		),
	)
}

func getKeycloakClientSpec(cheURL string) keycloak.KeycloakClientSpec {
	return keycloak.KeycloakClientSpec{
		RealmSelector: &metav1.LabelSelector{
			MatchLabels: rhsso.GetInstanceLabels(),
		},
		Client: &keycloak.KeycloakAPIClient{
			ID:                        defaultClientName,
			ClientID:                  defaultClientName,
			ClientAuthenticatorType:   "client-secret",
			Enabled:                   true,
			PublicClient:              true,
			DirectAccessGrantsEnabled: true,
			RedirectUris:              []string{cheURL, fmt.Sprintf("%s/*", cheURL)},
			WebOrigins:                []string{cheURL, fmt.Sprintf("%s/*", cheURL)},
			StandardFlowEnabled:       true,
			RootURL:                   cheURL,
			FullScopeAllowed:          true,
			Access: map[string]bool{
				"view":      true,
				"configure": true,
				"manage":    true,
			},
			ProtocolMappers: []keycloak.KeycloakProtocolMapper{
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
					ConsentText:     "n.a.",
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
	}
}

func (r *Reconciler) reconcileSubscription(ctx context.Context, serverClient k8sclient.Client, inst *integreatlyv1alpha1.RHMI, productNamespace string, operatorNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	target := marketplace.Target{
		Pkg:       constants.CodeReadySubscriptionName,
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
		r.preUpgradeBackupExecutor(),
		serverClient,
		catalogSourceReconciler,
		r.log,
	)
}
