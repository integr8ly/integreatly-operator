package codeready

import (
	"context"
	"fmt"

	"github.com/integr8ly/integreatly-operator/pkg/products/rhsso"

	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"

	"github.com/sirupsen/logrus"

	chev1 "github.com/eclipse/che-operator/pkg/apis/org/v1"

	crov1alpha1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	cro1types "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/resources"

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
	tier                         = "production"
)

type Reconciler struct {
	Config        *config.CodeReady
	ConfigManager config.ConfigReadWriter
	installation  *integreatlyv1alpha1.RHMI
	extraParams   map[string]string
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
	*resources.Reconciler
	recorder record.EventRecorder
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder) (*Reconciler, error) {
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

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
		installation:  installation,
		mpm:           mpm,
		logger:        logger,
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

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileFinalizer(ctx, serverClient, r.installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		phase, err := resources.RemoveNamespace(ctx, r.installation, serverClient, r.Config.GetNamespace())
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		phase, err = resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetOperatorNamespace())
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		return integreatlyv1alpha1.PhaseCompleted, nil
	})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetOperatorNamespace(), r.installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetOperatorNamespace()), err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetNamespace(), r.installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetNamespace()), err)
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, r.Config.GetNamespace(), serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, fmt.Sprintf("Failed to retrieve %s namespace", r.Config.GetNamespace()), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: constants.CodeReadySubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetOperatorNamespace(), ManifestPackage: manifestPackage}, []string{r.Config.GetNamespace()}, serverClient)
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

	phase, err = r.reconcileTemplates(ctx, serverClient)
	logrus.Infof("Phase: %s reconcileTemplates", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile templates", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	r.logger.Infof("%s has reconciled successfully", r.Config.GetProductName())
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileExternalDatasources(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	logrus.Infof("Reconciling external datastore")
	ns := r.installation.Namespace

	// setup postgres custom resource
	postgresName := fmt.Sprintf("%s%s", constants.CodeReadyPostgresPrefix, r.installation.Name)
	postgres, err := croUtil.ReconcilePostgres(ctx, serverClient, defaultInstallationNamespace, r.installation.Spec.Type, tier, postgresName, ns, postgresName, ns, func(cr metav1.Object) error {
		owner.AddIntegreatlyOwnerAnnotations(cr, r.installation)
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile postgres: %w", err)
	}

	if postgres.Status.Phase != cro1types.PhaseComplete {
		return integreatlyv1alpha1.PhaseAwaitingComponents, nil
	}

	// create the prometheus availability rule
	if _, err = resources.CreatePostgresAvailabilityAlert(ctx, serverClient, r.installation, postgres); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres prometheus alert for codeready: %w", err)
	}
	// create the prometheus connectivity rule
	if _, err = resources.CreatePostgresConnectivityAlert(ctx, serverClient, r.installation, postgres); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres connectivity prometheus alert for codeready: %s", err)
	}

	// get the secret created by the cloud resources operator
	croSec := &corev1.Secret{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: postgres.Status.SecretRef.Name, Namespace: postgres.Status.SecretRef.Namespace}, croSec)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get postgres credential secret: %w", err)
	}

	// create backup secret
	logrus.Info("Reconciling codeready backup secret")
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

	r.logger.Infof("creating required custom resources in namespace: %s", r.Config.GetNamespace())

	kcRealm := &keycloak.KeycloakRealm{}
	key := k8sclient.ObjectKey{Name: kcConfig.GetRealm(), Namespace: kcConfig.GetNamespace()}
	err = serverClient.Get(ctx, key, kcRealm)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not retrieve: %+v: %w", key, err)
	}

	cheCluster, err := r.createCheCluster(ctx, kcConfig, kcRealm, serverClient)

	// che cluster hasn't reconciled yet
	if cheCluster == nil {
		return integreatlyv1alpha1.PhaseAwaitingComponents, nil
	}

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileTemplates(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// Interate over template_list
	for _, template := range r.Config.GetTemplateList() {
		// create it
		_, err := r.createResource(ctx, template, serverClient)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update monitoring template %s: %w", template, err)
		}
		logrus.Infof("Reconciling the monitoring template %s was successful", template)
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
	if err := resources.ReconcileBackup(ctx, serverClient, backupConfig, r.ConfigManager); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create backups for codeready: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) handleProgressPhase(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.logger.Info("checking that checluster custom resource is marked as available")

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
	r.logger.Infof("checking keycloak client exists for che")
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

	r.logger.Infof("The operation result for keycloakclient %s was %s", kcClient.Name, or)
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
