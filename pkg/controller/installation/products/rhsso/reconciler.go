package rhsso

import (
	"context"
	"errors"
	"fmt"
	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	pkgerr "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	DefaultRhssoNamespace = "rhsso"
	CustomerAdminName     = "customer-admin"
	keycloakName          = "rhsso"
	KeycloakRealmName     = "openshift"
	rhssoId               = "openshift-client"
	clientSecret          = rhssoId + "-secret"
)

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	config, err := configManager.ReadRHSSO()
	if err != nil {
		return nil, err
	}
	if config.GetNamespace() == "" {
		config.SetNamespace(instance.Spec.NamespacePrefix + DefaultRhssoNamespace)
	}
	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
		mpm:           mpm,
		installation:  instance,
	}, nil
}

type Reconciler struct {
	Config        *config.RHSSO
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	installation  *v1alpha1.Installation
}

func (r *Reconciler) Reconcile(inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	phase := inst.Status.ProductStatus[r.Config.GetProductName()]
	switch v1alpha1.StatusPhase(phase) {
	case v1alpha1.PhaseNone:
		return r.handleNoPhase(serverClient)
	case v1alpha1.PhaseAwaitingNS:
		return r.handleAwaitingNSPhase(serverClient)
	case v1alpha1.PhaseCreatingSubscription:
		return r.handleCreatingSubscription(serverClient)
	case v1alpha1.PhaseAwaitingOperator:
		return r.handleAwaitingOperator()
	case v1alpha1.PhaseCreatingComponents:
		return r.handleCreatingComponents(serverClient)
	case v1alpha1.PhaseInProgress:
		return r.handleProgressPhase(serverClient)
	case v1alpha1.PhaseCompleted:
		return v1alpha1.PhaseCompleted, nil
	case v1alpha1.PhaseFailed:
		//potentially do a dump and recover in the future
		return v1alpha1.PhaseFailed, errors.New("installation of RHSSO failed")
	default:
		return v1alpha1.PhaseFailed, errors.New("Unknown phase for RHSSO: " + string(phase))
	}
}

func (r *Reconciler) handleNoPhase(serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      r.Config.GetNamespace(),
		},
	}
	err := serverClient.Create(context.TODO(), ns)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, err
	}
	return v1alpha1.PhaseAwaitingNS, nil
}

func (r *Reconciler) handleAwaitingNSPhase(serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Config.GetNamespace(),
		},
	}
	err := serverClient.Get(context.TODO(), pkgclient.ObjectKey{Name: r.Config.GetNamespace()}, ns)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	if ns.Status.Phase == v1.NamespaceActive {
		logrus.Infof("Creating subscription")
		return v1alpha1.PhaseCreatingSubscription, nil
	}

	return v1alpha1.PhaseAwaitingNS, nil
}

func (r *Reconciler) handleCreatingSubscription(serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	err := r.mpm.CreateSubscription(
		serverClient,
		marketplace.GetOperatorSources().Integreatly,
		r.Config.GetNamespace(),
		"rhsso",
		"integreatly",
		[]string{r.Config.GetNamespace()},
		coreosv1alpha1.ApprovalAutomatic)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, err
	}

	return v1alpha1.PhaseAwaitingOperator, nil
}

func (r *Reconciler) handleAwaitingOperator() (v1alpha1.StatusPhase, error) {
	ip, err := r.mpm.GetSubscriptionInstallPlan("rhsso", r.Config.GetNamespace())
	if err != nil {
		if k8serr.IsNotFound(err) {
			logrus.Infof("No installplan created yet")
			return v1alpha1.PhaseAwaitingOperator, nil
		}

		logrus.Infof("Error getting rhsso subscription installplan")
		return v1alpha1.PhaseFailed, err
	}

	if ip.Status.Phase != coreosv1alpha1.InstallPlanPhaseComplete {
		logrus.Infof("rhsso installplan phase is %s", ip.Status.Phase)
		return v1alpha1.PhaseAwaitingOperator, nil
	}

	logrus.Infof("rhsso installplan phase is %s", coreosv1alpha1.InstallPlanPhaseComplete)

	return v1alpha1.PhaseCreatingComponents, nil
}

func (r *Reconciler) handleCreatingComponents(serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Infof("Creating Components")

	logrus.Infof("Creating Keycloak")

	kc := &aerogearv1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: r.Config.GetNamespace(),
		},
		Spec: aerogearv1.KeycloakSpec{
			AdminCredentials: "",
			Plugins: []string{
				"keycloak-metrics-spi",
			},
			Backups:   []aerogearv1.KeycloakBackup{},
			Provision: true,
		},
	}

	err := serverClient.Create(context.TODO(), kc)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	logrus.Infof("Creating Keycloakrealm")
	password := "Password1"
	kcr := &aerogearv1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KeycloakRealmName,
			Namespace: r.Config.GetNamespace(),
		},
		Spec: aerogearv1.KeycloakRealmSpec{
			CreateOnly: true,
			KeycloakApiRealm: &aerogearv1.KeycloakApiRealm{
				ID:          KeycloakRealmName,
				Realm:       KeycloakRealmName,
				DisplayName: KeycloakRealmName,
				Enabled:     true,
				EventsListeners: []string{
					"metrics-listener",
				},
				Users: []*aerogearv1.KeycloakUser{
					{
						KeycloakApiUser: &aerogearv1.KeycloakApiUser{
							Enabled:       true,
							Attributes:    aerogearv1.KeycloakAttributes{},
							UserName:      "customer-admin",
							EmailVerified: true,
							Email:         "customer-admin@example.com",
							RealmRoles: []string{
								"offline_access",
								"uma_authorization",
							},
							ClientRoles: map[string][]string{
								"account": {
									"manage-account",
									"view-profile",
								},
								"realm-management": {
									"manage-users",
									"manage-identity-providers",
									"view-realm",
								},
							},
						},
						Password:     &password,
						OutputSecret: "customer-admin-user-credentials",
					},
				},
				Clients: []*aerogearv1.KeycloakClient{
					{
						KeycloakApiClient: &aerogearv1.KeycloakApiClient{
							ID:                      rhssoId,
							ClientID:                rhssoId,
							Enabled:                 true,
							Secret:                  clientSecret,
							ClientAuthenticatorType: "client-secret",
							RedirectUris: []string{
								fmt.Sprintf("https://tutorial-web-app-webapp.%s", r.installation.Spec.RoutingSubdomain),
								fmt.Sprintf("%v/*", r.installation.Spec.MasterURL),
								"http://localhost:3006*",
							},
							StandardFlowEnabled:       true,
							DirectAccessGrantsEnabled: true,
						},
						OutputSecret: rhssoId + "-client",
					},
				},
			},
		},
	}
	err = serverClient.Create(context.TODO(), kcr)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	return v1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) handleProgressPhase(serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Infof("checking ready status for rhsso")
	kcr := &aerogearv1.KeycloakRealm{}

	err := serverClient.Get(context.TODO(), pkgclient.ObjectKey{Name: KeycloakRealmName, Namespace: r.Config.GetNamespace()}, kcr)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	if kcr.Status.Phase == aerogearv1.PhaseReconcile {
		err = r.exportConfig(serverClient)
		if err != nil {
			logrus.Errorf("Failed to write RH-SSO config %v", err)
			return v1alpha1.PhaseFailed, err
		}

		logrus.Infof("Keycloak has successfully processed the keycloakRealm")
		return v1alpha1.PhaseCompleted, nil
	}

	logrus.Infof("KeycloakRealm status phase is: %s", kcr.Status.Phase)
	return v1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) exportConfig(serverClient pkgclient.Client) error {
	kc := &aerogearv1.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err := serverClient.Get(context.TODO(), pkgclient.ObjectKey{Name: keycloakName, Namespace: r.Config.GetNamespace()}, kc)
	if err != nil {
		return pkgerr.Wrap(err, "could not retrieve keycloak custom resource for keycloak config")
	}
	kcAdminCredSecretName := kc.Spec.AdminCredentials

	kcAdminCredSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kcAdminCredSecretName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err = serverClient.Get(context.TODO(), pkgclient.ObjectKey{Name: kcAdminCredSecretName, Namespace: r.Config.GetNamespace()}, kcAdminCredSecret)
	if err != nil {
		return pkgerr.Wrap(err, "could not retrieve keycloak admin credential secret for keycloak config")
	}
	kcURLBytes := kcAdminCredSecret.Data["SSO_ADMIN_URL"]
	r.Config.SetRealm(KeycloakRealmName)
	r.Config.SetURL(string(kcURLBytes))
	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return pkgerr.Wrap(err, "could not update keycloak config")
	}
	return nil
}
