package codeready

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	chev1 "github.com/eclipse/che-operator/pkg/apis/org/v1"

	v1alpha13 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	cro1types "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/resources"

	keycloakv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	v1alpha12 "github.com/integr8ly/integreatly-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/monitoring"
	"github.com/integr8ly/integreatly-operator/pkg/resources"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "codeready-workspaces"
	defaultClientName            = "che-client"
	defaultCheClusterName        = "integreatly-cluster"
	defaultSubscriptionName      = "integreatly-codeready-workspaces"
	manifestPackage              = "integreatly-codeready-workspaces"
	tier                         = "production"
)

type Reconciler struct {
	Config        *config.CodeReady
	ConfigManager config.ConfigReadWriter
	installation  *v1alpha1.Installation
	extraParams   map[string]string
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
	*resources.Reconciler
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	config, err := configManager.ReadCodeReady()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve che config: %w", err)
	}
	if config.GetNamespace() == "" {
		config.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
		installation:  instance,
		mpm:           mpm,
		logger:        logger,
		Reconciler:    resources.NewReconciler(mpm),
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

func (r *Reconciler) Reconcile(ctx context.Context, instance *v1alpha1.Installation, product *v1alpha1.InstallationProductStatus, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileFinalizer(ctx, serverClient, r.installation, string(r.Config.GetProductName()), func() (v1alpha1.StatusPhase, error) {
		phase, err := resources.RemoveNamespace(ctx, r.installation, serverClient, r.Config.GetNamespace())
		if err != nil || phase != v1alpha1.PhaseCompleted {
			return phase, err
		}

		return v1alpha1.PhaseCompleted, nil
	})
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetNamespace(), r.installation, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, r.Config.GetNamespace(), serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: defaultSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetNamespace(), ManifestPackage: manifestPackage}, r.Config.GetNamespace(), serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileExternalDatasources(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileCheCluster(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileKeycloakClient(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileBlackboxTargets(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileBackups(ctx, serverClient, namespace)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileTemplates(ctx, serverClient)
	logrus.Infof("Phase: %s reconcileTemplates", phase)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		logrus.Infof("Error: %s", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	r.logger.Infof("%s has reconciled successfully", r.Config.GetProductName())
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileExternalDatasources(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Infof("Reconciling external datastore")
	ns := r.installation.Namespace

	// setup postgres custom resource
	postgresName := fmt.Sprintf("codeready-postgres-%s", r.installation.Name)
	postgres, err := croUtil.ReconcilePostgres(ctx, serverClient, r.installation.Spec.Type, tier, postgresName, ns, postgresName, ns, func(cr metav1.Object) error {
		ownerutil.EnsureOwner(cr, r.installation)
		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile postgres: %w", err)
	}

	if postgres.Status.Phase != cro1types.PhaseComplete {
		return v1alpha1.PhaseAwaitingComponents, nil
	}

	// get the secret created by the cloud resources operator
	croSec := &corev1.Secret{}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: postgres.Status.SecretRef.Name, Namespace: postgres.Status.SecretRef.Namespace}, croSec)
	if err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to get postgres credential secret: %w", err)
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
		cheBackUpSecret.Data["POSTGRESQL_USER"] = croSec.Data["username"]
		cheBackUpSecret.Data["POSTGRESQL_PASSWORD"] = croSec.Data["password"]
		cheBackUpSecret.Data["POSTGRESQL_DATABASE"] = croSec.Data["database"]
		cheBackUpSecret.Data["POSTGRESQL_PORT"] = croSec.Data["port"]
		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to create or update %s connection secret: %w", r.Config.GetPostgresBackupSecretName(), err)
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileCheCluster(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	kcConfig, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("could not retrieve keycloak config: %w", err)
	}
	if err = kcConfig.Validate(); err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("keycloak config is not valid: %w", err)
	}

	r.logger.Infof("creating required custom resources in namespace: %s", r.Config.GetNamespace())

	kcRealm := &keycloakv1.KeycloakRealm{}
	key := pkgclient.ObjectKey{Name: kcConfig.GetRealm(), Namespace: kcConfig.GetNamespace()}
	err = serverClient.Get(ctx, key, kcRealm)
	if err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("could not retrieve: %+v: %w", key, err)
	}

	cheCluster := &chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultCheClusterName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: defaultCheClusterName, Namespace: r.Config.GetNamespace()}, cheCluster)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			return v1alpha1.PhaseFailed, fmt.Errorf("could not retrieve checluster custom resource in namespace: %s: %w", r.Config.GetNamespace(), err)
		}
		cheCluster, err := r.createCheCluster(ctx, kcConfig, kcRealm, serverClient)
		if err != nil {
			return v1alpha1.PhaseFailed, fmt.Errorf("could not create checluster custom resource in namespace: %s: %w", r.Config.GetNamespace(), err)
		}
		// che cluster hasn't reconciled yet
		if cheCluster == nil {
			return v1alpha1.PhaseAwaitingComponents, nil
		}
		return v1alpha1.PhaseInProgress, err
	}

	// check cr values
	if cheCluster.Spec.Auth.ExternalKeycloak &&
		!cheCluster.Spec.Auth.OpenShiftOauth &&
		cheCluster.Spec.Auth.KeycloakURL == kcConfig.GetHost() &&
		cheCluster.Spec.Auth.KeycloakRealm == kcConfig.GetRealm() &&
		cheCluster.Spec.Auth.KeycloakClientId == defaultClientName {
		logrus.Debug("skipping checluster custom resource update as all values are correct")
		return v1alpha1.PhaseCompleted, nil
	}

	// update cr values
	cheCluster.Spec.Auth.ExternalKeycloak = true
	cheCluster.Spec.Auth.OpenShiftOauth = false
	cheCluster.Spec.Auth.KeycloakURL = kcConfig.GetHost()
	cheCluster.Spec.Auth.KeycloakRealm = kcRealm.Name
	cheCluster.Spec.Auth.KeycloakClientId = defaultClientName
	if err = serverClient.Update(ctx, cheCluster); err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("could not update checluster custom resource in namespace: %s: %w", r.Config.GetNamespace(), err)
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileTemplates(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	// Interate over template_list
	for _, template := range r.Config.GetTemplateList() {
		// create it
		_, err := r.createResource(ctx, r.installation, template, serverClient)
		if err != nil {
			return v1alpha1.PhaseFailed, fmt.Errorf("failed to create/update monitoring template %s: %w", template, err)
		}
		logrus.Infof("Reconciling the monitoring template %s was successful", template)
	}
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) createResource(ctx context.Context, inst *v1alpha1.Installation, resourceName string, serverClient pkgclient.Client) (runtime.Object, error) {
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

func (r *Reconciler) reconcileBackups(ctx context.Context, serverClient pkgclient.Client, owner ownerutil.Owner) (v1alpha1.StatusPhase, error) {
	backupConfig := resources.BackupConfig{
		Namespace:     r.Config.GetNamespace(),
		Name:          "codeready",
		BackendSecret: resources.BackupSecretLocation{Name: r.Config.GetBackendSecretName(), Namespace: r.ConfigManager.GetOperatorNamespace()},
		Components: []resources.BackupComponent{
			{
				Name:     "codeready-postgres-backup",
				Type:     "postgres",
				Secret:   resources.BackupSecretLocation{Name: r.Config.GetPostgresBackupSecretName(), Namespace: r.Config.GetNamespace()},
				Schedule: r.Config.GetBackupSchedule(),
			},
			{
				Name:     "codeready-pv-backup",
				Type:     "codeready_pv",
				Schedule: r.Config.GetBackupSchedule(),
			},
		},
	}
	if err := resources.ReconcileBackup(ctx, serverClient, backupConfig, owner); err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to create backups for codeready: %w", err)
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) handleProgressPhase(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Info("checking that checluster custom resource is marked as available")

	// retrive the checluster so we can use its URL for redirect and web origins in the keycloak client
	cheCluster := &chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultCheClusterName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: defaultCheClusterName, Namespace: r.Config.GetNamespace()}, cheCluster)
	if err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("could not retrieve checluster for keycloak client update: %w", err)
	}
	if cheCluster.Status.CheClusterRunning != "Available" {
		return v1alpha1.PhaseInProgress, nil
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileKeycloakClient(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Infof("checking keycloak client exists for che")
	kcConfig, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("could not retrieve keycloak config: %w", err)
	}
	if err = kcConfig.Validate(); err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("keycloak config is not valid: %w", err)
	}

	// retrive the checluster so we can use its URL for redirect and web origins in the keycloak client
	cheCluster := &chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultCheClusterName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: defaultCheClusterName, Namespace: r.Config.GetNamespace()}, cheCluster)
	if err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("could not retrieve checluster for keycloak client update: %w", err)
	}

	cheURL := cheCluster.Status.CheURL
	if cheURL == "" {
		//still waiting for the Che URL, so exit codeready reconciling now and try again
		return v1alpha1.PhaseInProgress, nil
	}

	if r.Config.GetHost() != cheURL {
		r.Config.SetHost(cheURL)
		if err = r.ConfigManager.WriteConfig(r.Config); err != nil {
			return v1alpha1.PhaseFailed, fmt.Errorf("could not write che configuration: %w", err)
		}
	}

	// retrieve the sso config so we can find the keycloakrealm custom resource to update
	kcRealm := &keycloakv1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kcConfig.GetRealm(),
			Namespace: kcConfig.GetNamespace(),
		},
	}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: kcConfig.GetRealm(), Namespace: kcConfig.GetNamespace()}, kcRealm)
	if err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("could not retrieve keycloakrealm for keycloak client update: %w", err)
	}

	// Create a che client that can be used in keycloak for che to login with
	if !keycloakv1.ContainsClient(kcRealm.Spec.Clients, defaultClientName) {
		r.logger.Infof("creating che client, %s, in keycloak", defaultClientName)
		kcRealm.Spec.Clients = append(kcRealm.Spec.Clients, &keycloakv1.KeycloakClient{
			KeycloakApiClient: &keycloakv1.KeycloakApiClient{
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
				ProtocolMappers: []keycloakv1.KeycloakProtocolMapper{
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
		if err = serverClient.Update(ctx, kcRealm); err != nil {
			return v1alpha1.PhaseFailed, fmt.Errorf("could not update keycloakrealm custom resource with codeready client: %w", err)
		}
	}
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileBlackboxTargets(ctx context.Context, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	cfg, err := r.ConfigManager.ReadMonitoring()
	if err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("error reading monitoring config: %w", err)
	}

	err = monitoring.CreateBlackboxTarget("integreatly-codeready", v1alpha12.BlackboxtargetData{
		Url:     r.Config.GetHost(),
		Service: "codeready-ui",
	}, ctx, cfg, r.installation, client)
	if err != nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("error creating codeready blackbox target: %w", err)
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) createCheCluster(ctx context.Context, kcCfg *config.RHSSO, kr *keycloakv1.KeycloakRealm, serverClient pkgclient.Client) (*chev1.CheCluster, error) {
	selfSignedCerts := r.installation.Spec.SelfSignedCerts

	// get postgres cloud resource cr
	pcr := &v1alpha13.Postgres{}
	postgresName := fmt.Sprintf("codeready-postgres-%s", r.installation.Name)
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: postgresName, Namespace: r.installation.Namespace}, pcr)
	if err != nil {
		return nil, fmt.Errorf("failed to find postgres custom resource: %w", err)
	}

	// get the postgres cloud resources operator cr
	croSec := &corev1.Secret{}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: pcr.Status.SecretRef.Name, Namespace: pcr.Status.SecretRef.Namespace}, croSec)
	if err != nil {
		return nil, fmt.Errorf("failed to get postgres credential secret: %w", err)
	}

	cheCluster := &chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultCheClusterName,
			Namespace: r.Config.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: fmt.Sprintf(
				"%s/%s",
				chev1.SchemeGroupVersion.Group,
				chev1.SchemeGroupVersion.Version,
			),
			Kind: "CheCluster",
		},
		Spec: chev1.CheClusterSpec{
			Server: chev1.CheClusterSpecServer{
				CheFlavor:      "codeready",
				TlsSupport:     true,
				SelfSignedCert: selfSignedCerts,
			},
			Database: chev1.CheClusterSpecDB{
				ExternalDB:            true,
				ChePostgresDb:         string(croSec.Data["database"]),
				ChePostgresPassword:   string(croSec.Data["password"]),
				ChePostgresPort:       string(croSec.Data["port"]),
				ChePostgresUser:       string(croSec.Data["username"]),
				ChePostgresDBHostname: string(croSec.Data["host"]),
			},
			Auth: chev1.CheClusterSpecAuth{
				OpenShiftOauth:   false,
				ExternalKeycloak: true,
				KeycloakURL:      kcCfg.GetHost(),
				KeycloakRealm:    kr.Name,
				KeycloakClientId: defaultClientName,
			},
			Storage: chev1.CheClusterSpecStorage{
				PvcStrategy:       "per-workspace",
				PvcClaimSize:      "1Gi",
				PreCreateSubPaths: true,
			},
		},
	}

	ownerutil.EnsureOwner(cheCluster, r.installation)
	if err := serverClient.Create(ctx, cheCluster); err != nil {
		return nil, fmt.Errorf("failed to create che cluster resource: %w", err)
	}
	return cheCluster, nil
}
