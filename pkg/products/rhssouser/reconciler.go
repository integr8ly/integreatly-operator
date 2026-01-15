package rhssouser

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/integr8ly/integreatly-operator/pkg/products/rhssocommon"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"

	"github.com/integr8ly/integreatly-operator/version"

	userHelper "github.com/integr8ly/integreatly-operator/pkg/resources/user"

	"github.com/integr8ly/integreatly-operator/pkg/products/rhsso"
	keycloakCommon "github.com/integr8ly/keycloak-client/pkg/common"
	consolev1 "github.com/openshift/api/console/v1"
	usersv1 "github.com/openshift/api/user/v1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"

	keycloak "github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	defaultNamespace          = "user-sso"
	keycloakName              = "rhssouser"
	idpAlias                  = "openshift-v4"
	masterRealmName           = "master"
	adminCredentialSecretName = "credential-" + keycloakName
	ssoType                   = "user sso"
	postgresResourceName      = "rhssouser-postgres-rhmi"
	routeName                 = "keycloak"
)

const (
	masterRealmLabelKey         = "sso"
	masterRealmLabelValue       = "master"
	developersGroupName         = "rhmi-developers"
	dedicatedAdminsGroupName    = "dedicated-admins"
	realmManagersGroupName      = "realm-managers"
	fullRealmManagersGroupPath  = dedicatedAdminsGroupName + "/" + realmManagersGroupName
	viewRealmRoleName           = "view-realm"
	createRealmRoleName         = "create-realm"
	manageUsersRoleName         = "manage-users"
	masterRealmClientName       = "master-realm"
	firstBrokerLoginFlowAlias   = "first broker login"
	reviewProfileExecutionAlias = "review profile config"
	userSsoConsoleLink          = "rhoam-user-sso-console-link"

	userSSOIcon = "data:image/svg+xml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0idXRmLTgiPz4KPCEtLSBHZW5lcmF0b3I6IEFkb2JlIElsbHVzdHJhdG9yIDI1LjIuMCwgU1ZHIEV4cG9ydCBQbHVnLUluIC4gU1ZHIFZlcnNpb246IDYuMDAgQnVpbGQgMCkgIC0tPgo8c3ZnIHZlcnNpb249IjEuMSIgaWQ9IkxheWVyXzEiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgeG1sbnM6eGxpbms9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkveGxpbmsiIHg9IjBweCIgeT0iMHB4IgoJIHZpZXdCb3g9IjAgMCAzNyAzNyIgc3R5bGU9ImVuYWJsZS1iYWNrZ3JvdW5kOm5ldyAwIDAgMzcgMzc7IiB4bWw6c3BhY2U9InByZXNlcnZlIj4KPHN0eWxlIHR5cGU9InRleHQvY3NzIj4KCS5zdDB7ZmlsbDojRUUwMDAwO30KCS5zdDF7ZmlsbDojRkZGRkZGO30KPC9zdHlsZT4KPGc+Cgk8cGF0aCBkPSJNMjcuNSwwLjVoLTE4Yy00Ljk3LDAtOSw0LjAzLTksOXYxOGMwLDQuOTcsNC4wMyw5LDksOWgxOGM0Ljk3LDAsOS00LjAzLDktOXYtMThDMzYuNSw0LjUzLDMyLjQ3LDAuNSwyNy41LDAuNUwyNy41LDAuNXoiCgkJLz4KCTxnPgoJCTxwYXRoIGNsYXNzPSJzdDAiIGQ9Ik0yNSwyMi4zN2MtMC45NSwwLTEuNzUsMC42My0yLjAyLDEuNWgtMS44NVYyMS41YzAtMC4zNS0wLjI4LTAuNjItMC42Mi0wLjYycy0wLjYyLDAuMjgtMC42MiwwLjYydjMKCQkJYzAsMC4zNSwwLjI4LDAuNjIsMC42MiwwLjYyaDIuNDhjMC4yNywwLjg3LDEuMDcsMS41LDIuMDIsMS41YzEuMTcsMCwyLjEyLTAuOTUsMi4xMi0yLjEyUzI2LjE3LDIyLjM3LDI1LDIyLjM3eiBNMjUsMjUuMzcKCQkJYy0wLjQ4LDAtMC44OC0wLjM5LTAuODgtMC44OHMwLjM5LTAuODgsMC44OC0wLjg4czAuODgsMC4zOSwwLjg4LDAuODhTMjUuNDgsMjUuMzcsMjUsMjUuMzd6Ii8+CgkJPHBhdGggY2xhc3M9InN0MCIgZD0iTTIwLjUsMTYuMTJjMC4zNCwwLDAuNjItMC4yOCwwLjYyLTAuNjJ2LTIuMzhoMS45MWMwLjMyLDAuNzcsMS4wOCwxLjMxLDEuOTYsMS4zMQoJCQljMS4xNywwLDIuMTItMC45NSwyLjEyLTIuMTJzLTAuOTUtMi4xMi0yLjEyLTIuMTJjLTEuMDIsMC0xLjg4LDAuNzMtMi4wOCwxLjY5SDIwLjVjLTAuMzQsMC0wLjYyLDAuMjgtMC42MiwwLjYydjMKCQkJQzE5Ljg3LDE1Ljg1LDIwLjE2LDE2LjEyLDIwLjUsMTYuMTJ6IE0yNSwxMS40M2MwLjQ4LDAsMC44OCwwLjM5LDAuODgsMC44OHMtMC4zOSwwLjg4LTAuODgsMC44OHMtMC44OC0wLjM5LTAuODgtMC44OAoJCQlTMjQuNTIsMTEuNDMsMjUsMTEuNDN6Ii8+CgkJPHBhdGggY2xhc3M9InN0MCIgZD0iTTEyLjEyLDE5Ljk2di0wLjg0aDIuMzhjMC4zNCwwLDAuNjItMC4yOCwwLjYyLTAuNjJzLTAuMjgtMC42Mi0wLjYyLTAuNjJoLTIuMzh2LTAuOTEKCQkJYzAtMC4zNS0wLjI4LTAuNjItMC42Mi0wLjYyaC0zYy0wLjM0LDAtMC42MiwwLjI4LTAuNjIsMC42MnYzYzAsMC4zNSwwLjI4LDAuNjIsMC42MiwwLjYyaDNDMTEuODQsMjAuNTksMTIuMTIsMjAuMzEsMTIuMTIsMTkuOTYKCQkJeiBNMTAuODcsMTkuMzRIOS4xMnYtMS43NWgxLjc1VjE5LjM0eiIvPgoJCTxwYXRoIGNsYXNzPSJzdDAiIGQ9Ik0yOC41LDE2LjM0aC0zYy0wLjM0LDAtMC42MiwwLjI4LTAuNjIsMC42MnYwLjkxSDIyLjVjLTAuMzQsMC0wLjYyLDAuMjgtMC42MiwwLjYyczAuMjgsMC42MiwwLjYyLDAuNjJoMi4zOAoJCQl2MC44NGMwLDAuMzUsMC4yOCwwLjYyLDAuNjIsMC42MmgzYzAuMzQsMCwwLjYyLTAuMjgsMC42Mi0wLjYydi0zQzI5LjEyLDE2LjYyLDI4Ljg0LDE2LjM0LDI4LjUsMTYuMzR6IE0yNy44NywxOS4zNGgtMS43NXYtMS43NQoJCQloMS43NVYxOS4zNHoiLz4KCQk8cGF0aCBjbGFzcz0ic3QwIiBkPSJNMTYuNSwyMC44N2MtMC4zNCwwLTAuNjMsMC4yOC0wLjYzLDAuNjJ2Mi4zOGgtMS44NWMtMC4yNy0wLjg3LTEuMDctMS41LTIuMDItMS41CgkJCWMtMS4xNywwLTIuMTIsMC45NS0yLjEyLDIuMTJzMC45NSwyLjEyLDIuMTIsMi4xMmMwLjk1LDAsMS43NS0wLjYzLDIuMDItMS41aDIuNDhjMC4zNCwwLDAuNjItMC4yOCwwLjYyLTAuNjJ2LTMKCQkJQzE3LjEyLDIxLjE1LDE2Ljg0LDIwLjg3LDE2LjUsMjAuODd6IE0xMiwyNS4zN2MtMC40OCwwLTAuODgtMC4zOS0wLjg4LTAuODhzMC4zOS0wLjg4LDAuODgtMC44OHMwLjg4LDAuMzksMC44OCwwLjg4CgkJCVMxMi40OCwyNS4zNywxMiwyNS4zN3oiLz4KCQk8cGF0aCBjbGFzcz0ic3QwIiBkPSJNMTYuNSwxMS44N2gtMi40MmMtMC4yLTAuOTctMS4wNi0xLjY5LTIuMDgtMS42OWMtMS4xNywwLTIuMTIsMC45NS0yLjEyLDIuMTJzMC45NSwyLjEyLDIuMTIsMi4xMgoJCQljMC44OCwwLDEuNjQtMC41NCwxLjk2LTEuMzFoMS45MXYyLjM4YzAsMC4zNSwwLjI4LDAuNjIsMC42MywwLjYyczAuNjItMC4yOCwwLjYyLTAuNjJ2LTNDMTcuMTIsMTIuMTUsMTYuODQsMTEuODcsMTYuNSwxMS44N3oKCQkJIE0xMiwxMy4xOGMtMC40OCwwLTAuODgtMC4zOS0wLjg4LTAuODhzMC4zOS0wLjg4LDAuODgtMC44OHMwLjg4LDAuMzksMC44OCwwLjg4UzEyLjQ4LDEzLjE4LDEyLDEzLjE4eiIvPgoJPC9nPgoJPHBhdGggY2xhc3M9InN0MSIgZD0iTTE4LjUsMjIuNjJjLTIuMjcsMC00LjEzLTEuODUtNC4xMy00LjEyczEuODUtNC4xMiw0LjEzLTQuMTJzNC4xMiwxLjg1LDQuMTIsNC4xMlMyMC43NywyMi42MiwxOC41LDIyLjYyegoJCSBNMTguNSwxNS42MmMtMS41OCwwLTIuODgsMS4yOS0yLjg4LDIuODhzMS4yOSwyLjg4LDIuODgsMi44OHMyLjg4LTEuMjksMi44OC0yLjg4UzIwLjA4LDE1LjYyLDE4LjUsMTUuNjJ6Ii8+CjwvZz4KPC9zdmc+Cg=="
)

var realmManagersClientRoles = []string{
	"create-client",
	"manage-authorization",
	"manage-clients",
	"manage-events",
	"manage-identity-providers",
	"manage-realm",
	"manage-users",
	"query-clients",
	"query-groups",
	"query-realms",
	"query-users",
	"view-authorization",
	"view-clients",
	"view-events",
	"view-identity-providers",
	"view-realm",
	"view-users",
}

type Reconciler struct {
	Config *config.RHSSOUser
	Log    l.Logger
	*rhssocommon.Reconciler
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, oauthv1Client oauthClient.OauthV1Interface, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, apiUrl string, keycloakClientFactory keycloakCommon.KeycloakClientFactory, logger l.Logger, productDeclaration *marketplace.ProductDeclaration) (*Reconciler, error) {
	if productDeclaration == nil {
		return nil, fmt.Errorf("no product declaration found for user SSO")
	}

	productConfig, err := configManager.ReadRHSSOUser()
	if err != nil {
		return nil, err
	}

	rhssocommon.SetNameSpaces(installation, productConfig.RHSSOCommon, defaultNamespace)

	return &Reconciler{
		Config:     productConfig,
		Log:        logger,
		Reconciler: rhssocommon.NewReconciler(configManager, mpm, installation, logger, oauthv1Client, recorder, apiUrl, keycloakClientFactory, *productDeclaration),
	}, nil
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return version.VerifyProductAndOperatorVersion(
		installation.Status.Stages[integreatlyv1alpha1.InstallStage].Products[integreatlyv1alpha1.ProductRHSSOUser],
		string(integreatlyv1alpha1.VersionRHSSOUser),
		string(integreatlyv1alpha1.OperatorVersionRHSSOUser),
	)
}

// Reconcile reads that state of the cluster for rhsso and makes changes based on the state read
// and what is required
func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, productStatus *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client, productConfig quota.ProductConfig, uninstall bool) (integreatlyv1alpha1.StatusPhase, error) {
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

		if err := r.deleteConsoleLink(ctx, serverClient, userSsoConsoleLink); err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		return integreatlyv1alpha1.PhaseCompleted, nil
	}, r.Log)
	if err != nil || phase == integreatlyv1alpha1.PhaseFailed {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	if uninstall {
		return phase, nil
	}

	phase, err = r.ReconcileNamespace(ctx, operatorNamespace, installation, serverClient, r.Log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", operatorNamespace), err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, productNamespace, installation, serverClient, r.Log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", productNamespace), err)
		return phase, err
	}

	phase, err = resources.ReconcileLimitRange(ctx, serverClient, operatorNamespace, resources.DefaultLimitRangeParams)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, fmt.Sprintf("Failed to reconcile LimitRange for Namespace %s", operatorNamespace), err)
		return phase, err
	}

	phase, err = resources.ReconcileLimitRange(ctx, serverClient, productNamespace, resources.DefaultLimitRangeParams)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, fmt.Sprintf("Failed to reconcile LimitRange for Namespace %s", productNamespace), err)
		return phase, err
	}

	phase, err = resources.ReconcileSecretToProductNamespace(ctx, serverClient, r.ConfigManager, adminCredentialSecretName, productNamespace, r.Log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile admin credentials secret", err)
		return phase, err
	}

	phase, err = r.ReconcileSubscription(ctx, serverClient, installation, productNamespace, operatorNamespace, postgresResourceName)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.RHSSOSubscriptionName), err)
		return phase, err
	}

	phase, err = r.ReconcileCsvDeploymentsPriority(
		ctx,
		serverClient,
		fmt.Sprintf("rhsso-operator.%s", "7.6.12-opr-005"),
		r.Config.GetOperatorNamespace(),
		installation.Spec.PriorityClassName,
	)
	if err != nil || phase == integreatlyv1alpha1.PhaseFailed {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile user rhsso-operator csv deployments priority class name", err)
		return phase, err
	}

	phase, err = r.CreateKeycloakRoute(ctx, serverClient, r.Config, r.Config.RHSSOCommon, routeName)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to handle in progress phase", err)
		return phase, err
	}

	// Wait for RHSSO postgres to be completed
	phase, err = resources.WaitForRHSSOPostgresToBeComplete(serverClient, installation.Name, r.ConfigManager.GetOperatorNamespace())
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Waiting for RHSSO postgres to be completed", err)
		return phase, err
	}

	phase, err = r.ReconcileCloudResources(constants.RHSSOUserProstgresPrefix, defaultNamespace, ssoType, r.Config.RHSSOCommon, ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile cloud resources", err)
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, installation, serverClient, productConfig)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile components", err)
		return phase, err
	}

	phase, err = r.ReconcileStatefulSet(ctx, serverClient, r.Config.RHSSOCommon)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconsile RHSSO pod priority", err)
		return phase, err
	}

	phase, err = r.HandleProgressPhase(ctx, serverClient, keycloakName, masterRealmName, r.Config, r.Config.RHSSOCommon, string(integreatlyv1alpha1.VersionRHSSOUser), string(integreatlyv1alpha1.OperatorVersionRHSSOUser))
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to handle in progress phase", err)
		return phase, err
	}

	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error writing to config in rhssouser reconciler: %w", err)
	}

	phase, err = resources.ReconcileSecretToRHMIOperatorNamespace(ctx, serverClient, r.ConfigManager, adminCredentialSecretName, productNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile admin credential secret to RHMI operator namespace", err)
		return phase, err
	}

	phase, err = r.newAlertsReconciler(r.Log, r.Installation.Spec.Type, config.GetOboNamespace(r.Installation.Namespace)).ReconcileAlerts(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile alerts", err)
		return phase, err
	}

	phase, err = r.RemovePodMonitors(ctx, serverClient, r.Config)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to remove pod monitor", err)
		return phase, err
	}

	if err := r.reconcileConsoleLink(ctx, serverClient); err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
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
	r.Log.Infof("Reconcile successful", l.Fields{"productStatus": r.Config.GetProductName()})
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, installation *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client, productConfig quota.ProductConfig) (integreatlyv1alpha1.StatusPhase, error) {
	r.Log.Info("Reconciling Keycloak components")
	kc := &keycloak.Keycloak{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakName,
			Namespace: r.Config.GetNamespace(),
		},
		Spec: keycloak.KeycloakSpec{
			KeycloakDeploymentSpec: keycloak.KeycloakDeploymentSpec{
				Experimental: keycloak.ExperimentalSpec{
					Env: []corev1.EnvVar{
						{
							Name:  "DISABLE_EXTERNAL_ACCESS",
							Value: "TRUE",
						},
					},
				},
			},
		},
	}

	key := k8sclient.ObjectKeyFromObject(kc)
	err := serverClient.Get(ctx, key, kc)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, kc, func() error {
		owner.AddIntegreatlyOwnerAnnotations(kc, installation)
		kc.Spec.Extensions = []string{
			rhssocommon.KeycloakMetricsExtension,
		}
		kc.Spec.ExternalDatabase = keycloak.KeycloakExternalDatabase{Enabled: true}
		kc.Labels = getMasterLabels()

		// Disabling the KCO route for managed-api
		kc.Spec.ExternalAccess = keycloak.KeycloakExternalAccess{Enabled: false}

		kc.Spec.Profile = rhsso.RHSSOProfile
		kc.Spec.PodDisruptionBudget = keycloak.PodDisruptionBudgetConfig{Enabled: true}

		// Always use rolling strategy
		// Recreate strategy might need to be used for minor or major version bumps
		kc.Spec.Migration.MigrationStrategy = keycloak.StrategyRolling

		err = productConfig.Configure(kc)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update keycloak custom resource: %w", err)
	}
	r.Log.Infof("Operation result", l.Fields{"keycloak": kc.Name, "res": or})

	// Patching the OwnerReference on the admin credentials secret
	err = resources.AddOwnerRefToSSOSecret(ctx, serverClient, adminCredentialSecretName, r.Config.GetNamespace(), *kc, r.Log)
	if err != nil {
		events.HandleError(r.Recorder, installation, integreatlyv1alpha1.PhaseFailed, "Failed to add ownerReferance admin credentials secret", err)
		return integreatlyv1alpha1.PhaseFailed, err
	}
	// We want to update the master realm by adding an openshift-v4 idp. We can not add the idp until we know the host
	if r.Config.GetHost() == "" {
		r.Log.Warning("Can not update keycloak master realm on user sso as host is not available yet")
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update keycloak master realm, host not available")
	}

	// Create the master realm. The master real already exists in Keycloak but we need to get a reference to it
	// in order to create the IDP and admin users on it
	masterKcr, err := r.updateMasterRealm(ctx, serverClient, installation)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	kcClient, err := r.KeycloakClientFactory.AuthenticatedClient(*kc)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Ensure the IDP exists before trying to create via rhsso client.
	// We have to create via rhsso client as keycloak will not accept changes to the master realm, via cr changes,
	// after its initial creation
	if masterKcr.Spec.Realm.IdentityProviders == nil && masterKcr.Spec.Realm.IdentityProviders[0] == nil {
		r.Log.Warning("Identity Provider not present on Realm - user sso")
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to update keycloak master realm with required IDP: %w", err)
	}

	exists, err := identityProviderExists(kcClient)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Error attempting to get existing idp on user sso, master realm: %w", err)
	} else if !exists {
		_, err = kcClient.CreateIdentityProvider(masterKcr.Spec.Realm.IdentityProviders[0], masterKcr.Spec.Realm.Realm)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Error creating idp on master realm, user sso: %w", err)
		}
	}

	phase, err := r.reconcileBrowserAuthFlow(ctx, kc, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.Recorder, installation, phase, "Failed to reconcile browser authentication flow", err)
		return phase, err
	}

	_, err = r.reconcileFirstLoginAuthFlow(kc)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Failed to reconcile first broker login authentication flow: %w", err)
	}

	// Get all currently existing keycloak users
	keycloakUsers, err := GetKeycloakUsers(ctx, serverClient, r.Config.GetNamespace())
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to list the keycloak users: %w", err)
	}

	_, err = r.reconcileGroups(ctx, serverClient, kc)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	_, err = r.reconcileAdminUsers(ctx, serverClient, kcClient, keycloakUsers)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	_, err = r.ReconcilePodDisruptionBudget(ctx, serverClient, r.Config.GetNamespace())
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileGroups(ctx context.Context, serverClient k8sclient.Client, kc *keycloak.Keycloak) (integreatlyv1alpha1.StatusPhase, error) {

	rolesConfigured, err := r.Config.GetDevelopersGroupConfigured()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if !rolesConfigured {
		_, err = r.reconcileDevelopersGroup(kc)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile rhmi-developers group: %w", err)
		}

		r.Config.SetDevelopersGroupConfigured(true)
		err = r.ConfigManager.WriteConfig(r.Config)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not update keycloak config for user-sso: %w", err)
		}
	}

	_, err = r.reconcileDedicatedAdminsGroup(kc)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile dedicated-admins group: %v", err)
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileAdminUsers(ctx context.Context, serverClient k8sclient.Client, kcClient keycloakCommon.KeycloakInterface, keycloakUsers []keycloak.KeycloakAPIUser) (integreatlyv1alpha1.StatusPhase, error) {

	// Sync keycloak with openshift users
	users, err := syncAdminUsersInMasterRealm(keycloakUsers, ctx, serverClient, r.Config.GetNamespace())
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to synchronize the users: %w", err)
	}

	// Create / update the synchronized users
	for _, user := range users {
		if user.UserName == "" {
			continue
		}

		// If the ID is not set, check if the user is already on Keycloak,
		// and set the ID on the CR to avoid the Keycloak operator from trying
		// to create the user, causing a conflict
		if user.ID == "" {
			kcUser, err := kcClient.FindUserByUsername(user.UserName, masterRealmName)
			if err != nil && err.Error() != "not found" {
				return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error attempting to retrieve user: %w", err)
			} else if err == nil {
				user.ID = kcUser.ID
			}
		}

		or, err := r.createOrUpdateKeycloakAdmin(user, ctx, serverClient)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update the customer admin user: %w", err)
		} else {
			r.Log.Infof("Operation result", l.Fields{"keycloakuser": user.UserName, "result": or})
		}
	}

	if err := r.reconcileGroupMembership(users, kcClient, masterRealmName); err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func getUsers(ctx context.Context, serverClient k8sclient.Client, ns string) ([]keycloak.KeycloakAPIUser, error) {
	var users keycloak.KeycloakUserList

	listOptions := []k8sclient.ListOption{
		k8sclient.MatchingLabels(getMasterLabels()),
		k8sclient.InNamespace(ns),
	}
	err := serverClient.List(ctx, &users, listOptions...)
	if err != nil {
		return nil, err
	}

	var mappedUsers []keycloak.KeycloakAPIUser
	for _, user := range users.Items {
		if strings.HasPrefix(user.ObjectMeta.Name, userHelper.GeneratedNamePrefix) {
			mappedUsers = append(mappedUsers, user.Spec.User)
		}
	}

	return mappedUsers, nil
}

func identityProviderExists(kcClient keycloakCommon.KeycloakInterface) (bool, error) {
	provider, err := kcClient.GetIdentityProvider(idpAlias, "master")
	if err != nil {
		return false, err
	}
	if provider != nil {
		return true, nil
	}
	return false, nil
}

// The master realm will be created as part of the Keycloak install. Here we update it to add the openshift idp
func (r *Reconciler) updateMasterRealm(ctx context.Context, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (*keycloak.KeycloakRealm, error) {

	kcr := &keycloak.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      masterRealmName,
			Namespace: r.Config.GetNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, kcr, func() error {
		kcr.Spec.RealmOverrides = []*keycloak.RedirectorIdentityProviderOverride{
			{
				IdentityProvider: idpAlias,
				ForFlow:          "browser",
			},
		}

		kcr.Spec.InstanceSelector = &metav1.LabelSelector{
			MatchLabels: getMasterLabels(),
		}

		kcr.Labels = getMasterLabels()

		kcr.Spec.Realm = &keycloak.KeycloakAPIRealm{
			ID:          masterRealmName,
			Realm:       masterRealmName,
			Enabled:     true,
			DisplayName: masterRealmName,
		}

		// The identity providers need to be set up before the realm CR gets
		// created because the Keycloak operator does not allow updates to
		// the realms
		redirectURIs := []string{r.Config.GetHost() + "/auth/realms/" + masterRealmName + "/broker/openshift-v4/endpoint"}
		err := r.SetupOpenshiftIDP(ctx, serverClient, installation, r.Config, kcr, redirectURIs, "")
		if err != nil {
			return fmt.Errorf("failed to setup Openshift IDP for user-sso: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create/update keycloak realm: %w", err)
	}
	r.Log.Infof("Operation result", l.Fields{"keycloakrealm": kcr.Name, "res": or})

	return kcr, nil
}

func (r *Reconciler) createOrUpdateKeycloakAdmin(user keycloak.KeycloakAPIUser, ctx context.Context, serverClient k8sclient.Client) (controllerutil.OperationResult, error) {
	kcUser := &keycloak.KeycloakUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userHelper.GetValidGeneratedUserName(user),
			Namespace: r.Config.GetNamespace(),
		},
	}
	if kcUser.Name == "" {
		return "", fmt.Errorf("failed to get valid generated username")
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, kcUser, func() error {
		kcUser.Spec.RealmSelector = &metav1.LabelSelector{
			MatchLabels: getMasterLabels(),
		}
		kcUser.Labels = getMasterLabels()
		kcUser.Spec.User = user

		return nil
	})
	if err != nil {
		return or, err
	}

	switch kcUser.Status.Phase {
	case keycloak.UserPhaseReconciled:
		return or, err
	case keycloak.UserPhaseFailing:
		return or, fmt.Errorf("keycloack operator failed to reconcile user: %s", kcUser.Status.Message)
	}

	return or, err
}

func GetKeycloakUsers(ctx context.Context, serverClient k8sclient.Client, ns string) ([]keycloak.KeycloakAPIUser, error) {
	return getUsers(ctx, serverClient, ns)
}

func getMasterLabels() map[string]string {
	return map[string]string{
		masterRealmLabelKey: masterRealmLabelValue,
	}
}

func syncAdminUsersInMasterRealm(keycloakUsers []keycloak.KeycloakAPIUser, ctx context.Context, serverClient k8sclient.Client, ns string) ([]keycloak.KeycloakAPIUser, error) {

	openshiftUsers := &usersv1.UserList{}
	err := serverClient.List(ctx, openshiftUsers)
	if err != nil {
		return nil, err
	}
	openshiftGroups := &usersv1.GroupList{}
	err = serverClient.List(ctx, openshiftGroups)
	if err != nil {
		return nil, err
	}

	dedicatedAdminUsers := getDedicatedAdmins(*openshiftUsers, *openshiftGroups)

	// added => Newly added to dedicated-admins group and OS
	// deleted => No longer exists in OS, remove from SSO
	// promoted => existing KC user, added to dedicated-admins group, promote KC privileges
	// demoted => existing KC user, removed from dedicated-admins group, demote KC privileges
	added, deleted, promoted, demoted := getUserDiff(keycloakUsers, openshiftUsers.Items, dedicatedAdminUsers)

	keycloakUsers, err = rhssocommon.DeleteKeycloakUsers(keycloakUsers, deleted, ns, ctx, serverClient)
	if err != nil {
		return nil, err
	}

	keycloakUsers = addKeycloakUsers(keycloakUsers, added)
	keycloakUsers = promoteKeycloakUsers(keycloakUsers, promoted)
	keycloakUsers = demoteKeycloakUsers(keycloakUsers, demoted)

	return keycloakUsers, nil
}

func addKeycloakUsers(keycloakUsers []keycloak.KeycloakAPIUser, added []usersv1.User) []keycloak.KeycloakAPIUser {

	for _, osUser := range added {

		keycloakUsers = append(keycloakUsers, keycloak.KeycloakAPIUser{
			Enabled:       true,
			UserName:      osUser.Name,
			EmailVerified: true,
			FederatedIdentities: []keycloak.FederatedIdentity{
				{
					IdentityProvider: idpAlias,
					UserID:           string(osUser.UID),
					UserName:         osUser.Name,
				},
			},
			RealmRoles: []string{"offline_access", "uma_authorization", "create-realm"},
			ClientRoles: map[string][]string{
				"account": {
					"manage-account",
					"view-profile",
				},
				"master-realm": {
					"view-clients",
					"view-realm",
					"manage-users",
				},
			},
			Groups: []string{dedicatedAdminsGroupName, fullRealmManagersGroupPath},
		})
	}
	return keycloakUsers
}

func promoteKeycloakUsers(allUsers []keycloak.KeycloakAPIUser, promoted []keycloak.KeycloakAPIUser) []keycloak.KeycloakAPIUser {

	for _, promotedUser := range promoted {
		for i, user := range allUsers {
			// ID is not populated, have to use UserName. Should be unique on master Realm
			if promotedUser.UserName == user.UserName {
				allUsers[i].ClientRoles = map[string][]string{
					"account": {
						"manage-account",
						"view-profile",
					},
					"master-realm": {
						"view-clients",
						"view-realm",
						"manage-users",
					}}
				allUsers[i].RealmRoles = []string{"offline_access", "uma_authorization", "create-realm"}

				// Add the "dedicated-admins" group if it's not there
				hasDedicatedAdminGroup := false
				hasRealmManagerGroup := false
				for _, group := range allUsers[i].Groups {
					if group == dedicatedAdminsGroupName {
						hasDedicatedAdminGroup = true
					}
					if group == fullRealmManagersGroupPath {
						hasRealmManagerGroup = true
					}
				}
				if !hasDedicatedAdminGroup {
					allUsers[i].Groups = append(allUsers[i].Groups, dedicatedAdminsGroupName)
				}
				if !hasRealmManagerGroup {
					allUsers[i].Groups = append(allUsers[i].Groups, fullRealmManagersGroupPath)
				}

				break
			}
		}
	}

	return allUsers
}

func demoteKeycloakUsers(allUsers []keycloak.KeycloakAPIUser, demoted []keycloak.KeycloakAPIUser) []keycloak.KeycloakAPIUser {

	for _, demotedUser := range demoted {
		for i, user := range allUsers {
			// ID is not populated, have to use UserName. Should be unique on master Realm
			if demotedUser.UserName == user.UserName { // ID is not set but UserName is
				allUsers[i].ClientRoles = map[string][]string{
					"account": {
						"manage-account",
						"manage-account-links",
						"view-profile",
					}}
				allUsers[i].RealmRoles = []string{"offline_access", "uma_authorization"}
				// Remove the dedicated-admins group from the user groups list
				groups := []string{}
				for _, group := range allUsers[i].Groups {
					if (group != dedicatedAdminsGroupName) && (group != fullRealmManagersGroupPath) {
						groups = append(groups, group)
					}
				}
				allUsers[i].Groups = groups
				break
			}
		}
	}

	return allUsers
}

// NOTE: The users type has a Groups field on it but it does not seem to get populated
// hence the need to check by name which is not ideal. However, this is the only field
// available on the Group type
func getDedicatedAdmins(osUsers usersv1.UserList, groups usersv1.GroupList) (dedicatedAdmins []usersv1.User) {

	var osUsersInGroup = getOsUsersInAdminsGroup(groups)

	for _, osUser := range osUsers.Items {
		if contains(osUsersInGroup, osUser.Name) {
			dedicatedAdmins = append(dedicatedAdmins, osUser)
		}
	}
	return dedicatedAdmins
}

func getOsUsersInAdminsGroup(groups usersv1.GroupList) (users []string) {

	for _, group := range groups.Items {
		if group.Name == "dedicated-admins" {
			if group.Users != nil {
				users = group.Users
			}
			break
		}
	}

	return users
}

// There are 3 conceptual user types
// 1. OpenShift User. 2. Keycloak User created by CR 3. Keycloak User created by customer
// The distinction is important as we want to try avoid managing users created by the customer apart from certain
// scenarios such as removing a user if they do not exist in OpenShift. This needs further consideration

// This function should return
// 1. Users in dedicated-admins group but not keycloak master realm => Added
// return osUser list
// 2. Users in dedicated-admins group, in keycloak master realm, but not have privledges => Promoted
// return keylcoak user list
// 3. Users in OS, Users in keycloak master realm, represented by a Keycloak CR, but not dedicated-admins group => Demote
// return keylcoak user list
// 4. Users not in OpenShift but in Keycloak Master Realm, represented by a Keycloak CR 			=> Delete
// return keylcoak user list
func getUserDiff(keycloakUsers []keycloak.KeycloakAPIUser, openshiftUsers []usersv1.User, dedicatedAdmins []usersv1.User) ([]usersv1.User, []keycloak.KeycloakAPIUser, []keycloak.KeycloakAPIUser, []keycloak.KeycloakAPIUser) {
	var added []usersv1.User
	var deleted []keycloak.KeycloakAPIUser
	var promoted []keycloak.KeycloakAPIUser
	var demoted []keycloak.KeycloakAPIUser

	for _, admin := range dedicatedAdmins {
		keycloakUser := getKeyCloakUser(admin, keycloakUsers)
		if keycloakUser == nil {
			// User in dedicated-admins group but not keycloak master realm
			added = append(added, admin)
		} else {
			if !hasAdminPrivileges(keycloakUser) {
				// User in dedicated-admins group, in keycloak master realm, but does not have privledges
				promoted = append(promoted, *keycloakUser)
			}
		}
	}

	for i := range keycloakUsers {
		kcUser := keycloakUsers[i]
		osUser := getOpenShiftUser(kcUser, openshiftUsers)
		if osUser != nil && !kcUserInDedicatedAdmins(kcUser, dedicatedAdmins) && hasAdminPrivileges(&kcUser) {
			// User in OS and keycloak master realm, represented by a Keycloak CR, but not dedicated-admins group
			demoted = append(demoted, kcUser)
		} else if osUser == nil {
			// User not in OpenShift but is in Keycloak Master Realm, represented by a Keycloak CR
			deleted = append(deleted, kcUser)
		}
	}

	return added, deleted, promoted, demoted
}

func kcUserInDedicatedAdmins(kcUser keycloak.KeycloakAPIUser, admins []usersv1.User) bool {
	for _, admin := range admins {
		if len(kcUser.FederatedIdentities) >= 1 && kcUser.FederatedIdentities[0].UserID == string(admin.UID) {
			return true
		}
	}
	return false
}

func getOpenShiftUser(kcUser keycloak.KeycloakAPIUser, osUsers []usersv1.User) *usersv1.User {
	for _, osUser := range osUsers {
		if len(kcUser.FederatedIdentities) >= 1 && kcUser.FederatedIdentities[0].UserID == string(osUser.UID) {
			return &osUser
		}
	}
	return nil
}

// Look for 2 key privileges to determine if user has admin rights
func hasAdminPrivileges(kcUser *keycloak.KeycloakAPIUser) bool {
	if len(kcUser.ClientRoles["master-realm"]) >= 1 && contains(kcUser.ClientRoles["master-realm"], "manage-users") && contains(kcUser.RealmRoles, "create-realm") {
		return true
	}
	return false
}

func contains(items []string, find string) bool {
	for _, item := range items {
		if item == find {
			return true
		}
	}
	return false
}

func getKeyCloakUser(admin usersv1.User, kcUsers []keycloak.KeycloakAPIUser) *keycloak.KeycloakAPIUser {
	for _, kcUser := range kcUsers {
		if len(kcUser.FederatedIdentities) >= 1 && kcUser.FederatedIdentities[0].UserID == string(admin.UID) {
			return &kcUser
		}
	}
	return nil
}

func OsUserInDedicatedAdmins(dedicatedAdmins []string, kcUser keycloak.KeycloakAPIUser) bool {
	for _, user := range dedicatedAdmins {
		if kcUser.UserName == user {
			return true
		}
	}
	return false
}

// Add authenticator config to the master realm. Because it is the master realm we need to make direct calls
// with the Keycloak client. This config allows for the automatic redirect to openshift-v4 as the IDP for Keycloak,
// as apposed to presenting the user with multiple login options.
func (r *Reconciler) reconcileBrowserAuthFlow(ctx context.Context, kc *keycloak.Keycloak, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	kcClient, err := r.KeycloakClientFactory.AuthenticatedClient(*kc)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	executions, err := kcClient.ListAuthenticationExecutionsForFlow("browser", masterRealmName)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Failed to retrieve execution flows on master realm: %w", err)
	}

	executionID := ""
	for _, execution := range executions {
		if execution.ProviderID == "identity-provider-redirector" {
			if execution.AuthenticationConfig != "" {
				r.Log.Info("Authenticator Config exists on master realm, rhsso-user")
				return integreatlyv1alpha1.PhaseCompleted, nil
			}
			executionID = execution.ID
			break
		}
	}
	if executionID == "" {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Failed to find relevant ProviderID in Authentication Executions: %w", err)
	}

	config := keycloak.AuthenticatorConfig{Config: map[string]string{"defaultProvider": "openshift-v4"}, Alias: "openshift-v4"}
	_, err = kcClient.CreateAuthenticatorConfig(&config, masterRealmName, executionID)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Failed to create Authenticator Config: %w", err)
	}

	r.Log.Info("Successfully created Authenticator Config")

	return integreatlyv1alpha1.PhaseCompleted, nil
}

// Create a default group called `rhmi-developers` with the "view-realm" client role and
// the "create-realm" realm role
func (r *Reconciler) reconcileDevelopersGroup(kc *keycloak.Keycloak) (integreatlyv1alpha1.StatusPhase, error) {
	// Get Keycloak client
	kcClient, err := r.KeycloakClientFactory.AuthenticatedClient(*kc)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	groupSpec := &keycloakGroupSpec{
		Name:        developersGroupName,
		RealmName:   masterRealmName,
		IsDefault:   true,
		ChildGroups: []*keycloakGroupSpec{},
		RealmRoles: []string{
			createRealmRoleName,
		},
		ClientRoles: []*keycloakClientRole{
			&keycloakClientRole{
				ClientName: masterRealmClientName,
				RoleName:   viewRealmRoleName,
			},
		},
	}

	_, err = reconcileGroup(kcClient, groupSpec)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileDedicatedAdminsGroup(kc *keycloak.Keycloak) (integreatlyv1alpha1.StatusPhase, error) {
	// Get Keycloak client
	kcClient, err := r.KeycloakClientFactory.AuthenticatedClient(*kc)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	dedicatedAdminsSpec := &keycloakGroupSpec{
		Name:       dedicatedAdminsGroupName,
		RealmName:  masterRealmName,
		RealmRoles: []string{},
		ClientRoles: []*keycloakClientRole{
			&keycloakClientRole{
				ClientName: masterRealmClientName,
				RoleName:   manageUsersRoleName,
			},
			&keycloakClientRole{
				ClientName: masterRealmClientName,
				RoleName:   viewRealmRoleName,
			},
		},
		ChildGroups: []*keycloakGroupSpec{
			&keycloakGroupSpec{
				Name:        realmManagersGroupName,
				RealmName:   masterRealmName,
				ClientRoles: []*keycloakClientRole{},
				RealmRoles:  []string{},
				ChildGroups: []*keycloakGroupSpec{},
			},
		},
	}

	// Reconcile the dedicated-admins group hierarchy
	_, err = reconcileGroup(kcClient, dedicatedAdminsSpec)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Get all the realms
	realms, err := kcClient.ListRealms()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Create a "manage-realm" role for each realm
	clientRoles := []*keycloakClientRole{}
	for _, realm := range realms {
		// Skip the master realm
		if realm.Realm == masterRealmName {
			continue
		}

		clientName := fmt.Sprintf("%s-realm", realm.Realm)

		for _, roleName := range realmManagersClientRoles {
			clientRole := &keycloakClientRole{
				ClientName: clientName,
				RoleName:   roleName,
			}

			clientRoles = append(clientRoles, clientRole)
		}
	}

	// Reconcile the realm-managers group with the manage-realm roles
	realmManagersSpec := &keycloakGroupSpec{
		Name:        realmManagersGroupName,
		RealmName:   masterRealmName,
		ClientRoles: clientRoles,
		ChildGroups: []*keycloakGroupSpec{},
		RealmRoles:  []string{},
	}

	_, err = reconcileGroup(kcClient, realmManagersSpec)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, err
}

func mapClientRoleToGroup(kcClient keycloakCommon.KeycloakInterface, realmName, groupID, clientID, roleName string) error {
	return mapRoleToGroupByName(roleName,
		func() ([]*keycloak.KeycloakUserRole, error) {
			return kcClient.ListGroupClientRoles(realmName, clientID, groupID)
		},
		func() ([]*keycloak.KeycloakUserRole, error) {
			return kcClient.ListAvailableGroupClientRoles(realmName, clientID, groupID)
		},
		func(role *keycloak.KeycloakUserRole) error {
			_, err := kcClient.CreateGroupClientRole(role, realmName, clientID, groupID)
			return err
		},
	)
}

func mapRealmRoleToGroup(kcClient keycloakCommon.KeycloakInterface, realmName, groupID, roleName string) error {
	return mapRoleToGroupByName(roleName,
		func() ([]*keycloak.KeycloakUserRole, error) {
			return kcClient.ListGroupRealmRoles(realmName, groupID)
		},
		func() ([]*keycloak.KeycloakUserRole, error) {
			return kcClient.ListAvailableGroupRealmRoles(realmName, groupID)
		},
		func(role *keycloak.KeycloakUserRole) error {
			_, err := kcClient.CreateGroupRealmRole(role, realmName, groupID)
			return err
		},
	)
}

// Map a role to a group given the name of the role to map and the logic to:
// list the roles already mapped, list the roles available to the group, and
// to map a role to the group
func mapRoleToGroupByName(
	roleName string,
	listMappedRoles func() ([]*keycloak.KeycloakUserRole, error),
	listAvailableRoles func() ([]*keycloak.KeycloakUserRole, error),
	mapRoleToGroup func(*keycloak.KeycloakUserRole) error,
) error {
	// Utility local function to look for the role in a list
	findRole := func(roles []*keycloak.KeycloakUserRole) *keycloak.KeycloakUserRole {
		for _, role := range roles {
			if role.Name == roleName {
				return role
			}
		}

		return nil
	}

	// Get the existing roles mapped to the group
	existingRoles, err := listMappedRoles()
	if err != nil {
		return err
	}

	// Look for the role among them, if it's already there, return
	existingRole := findRole(existingRoles)
	if existingRole != nil {
		return nil
	}

	// Get the available roles for the group
	availableRoles, err := listAvailableRoles()
	if err != nil {
		return err
	}

	// Look for the role among them. If it's not found, return an error
	role := findRole(availableRoles)
	if role == nil {
		return fmt.Errorf("%s role not found as available role for group", roleName)
	}

	// Map the role to the group
	return mapRoleToGroup(role)
}

func (r *Reconciler) reconcileFirstLoginAuthFlow(kc *keycloak.Keycloak) (integreatlyv1alpha1.StatusPhase, error) {
	// Get Keycloak client
	kcClient, err := r.KeycloakClientFactory.AuthenticatedClient(*kc)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Find the "review profile" execution for the first broker login
	// authentication flow
	authenticationExecution, err := kcClient.FindAuthenticationExecutionForFlow(firstBrokerLoginFlowAlias, masterRealmName, func(execution *keycloak.AuthenticationExecutionInfo) bool {
		return execution.Alias == reviewProfileExecutionAlias
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// If the execution is not found, nothing needs to be done
	if authenticationExecution == nil {
		return integreatlyv1alpha1.PhaseCompleted, nil
	}
	// If the execution is already disabled, nothing needs to be done
	if strings.ToUpper(authenticationExecution.Requirement) == "DISABLED" {
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	r.Log.Info("Disabling \"review profile\" execution from first broker login authentication flow")

	// Update the execution to "DISABLED"
	authenticationExecution.Requirement = "DISABLED"
	err = kcClient.UpdateAuthenticationExecutionForFlow(firstBrokerLoginFlowAlias, masterRealmName, authenticationExecution)

	// Return the phase status depending on whether the update operation
	// succeeded or failed
	if err == nil {
		return integreatlyv1alpha1.PhaseCompleted, nil
	}
	return integreatlyv1alpha1.PhaseFailed, err
}

// Struct to define the desired status of a Keycloak group
type keycloakGroupSpec struct {
	Name      string
	RealmName string
	IsDefault bool

	RealmRoles  []string
	ClientRoles []*keycloakClientRole

	ChildGroups []*keycloakGroupSpec
}

type keycloakClientRole struct {
	ClientName string
	RoleName   string
}

func reconcileGroup(kcClient keycloakCommon.KeycloakInterface, group *keycloakGroupSpec) (string, error) {
	// Look for the group in case it already exists
	existingGroup, err := kcClient.FindGroupByName(group.Name, group.RealmName)
	if err != nil {
		return "", fmt.Errorf("Error querying groups in realm %s: %v", group.RealmName, err)
	}

	// Store the group ID based on the existing group ID (in case it already
	// exists), or the newly created group ID if
	var groupID string
	if existingGroup != nil {
		groupID = existingGroup.ID
	} else {
		groupID, err = kcClient.CreateGroup(group.Name, group.RealmName)
		if err != nil {
			return "", fmt.Errorf("Error creating new group %s: %v", group.Name, err)
		}
	}

	// If the group is meant to be default make it default
	if group.IsDefault {
		err = kcClient.MakeGroupDefault(groupID, group.RealmName)
		if err != nil {
			return "", fmt.Errorf("Error making group %s default: %v", group.Name, err)
		}
	}

	// Create its client roles
	err = reconcileGroupClientRoles(kcClient, groupID, group)
	if err != nil {
		return "", err
	}
	// Create its realm roles
	err = reconcileGroupRealmRoles(kcClient, groupID, group)
	if err != nil {
		return "", err
	}

	// Create its child groups
	for _, childGroup := range group.ChildGroups {
		childGroupID, err := reconcileGroup(kcClient, childGroup)
		if err != nil {
			return "", fmt.Errorf("Error creating child group %s for group %s: %v",
				childGroup.Name, group.Name, err)
		}

		err = kcClient.SetGroupChild(groupID, group.RealmName, &keycloakCommon.Group{
			ID:   childGroupID,
			Name: childGroup.Name,
		})
		if err != nil {
			return "", fmt.Errorf("Error assigning child group %s to parent group %s: %v",
				childGroup.Name, group.Name, err)
		}
	}

	return groupID, nil
}

func reconcileGroupClientRoles(kcClient keycloakCommon.KeycloakInterface, groupID string, group *keycloakGroupSpec) error {
	if len(group.ClientRoles) == 0 {
		return nil
	}

	realmClients, err := listClientsByName(kcClient, group.RealmName)
	if err != nil {
		return err
	}

	for _, role := range group.ClientRoles {
		client, ok := realmClients[role.ClientName]

		if !ok {
			return fmt.Errorf("Client %s required for role %s not found in realm %s",
				role.ClientName, role.RoleName, group.RealmName)
		}

		err = mapClientRoleToGroup(kcClient, group.RealmName, groupID, client.ID, role.RoleName)
		if err != nil {
			return err
		}
	}

	return nil
}

func reconcileGroupRealmRoles(kcClient keycloakCommon.KeycloakInterface, groupID string, group *keycloakGroupSpec) error {
	for _, role := range group.RealmRoles {
		err := mapRealmRoleToGroup(kcClient, group.RealmName, groupID, role)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) reconcileGroupMembership(users []keycloak.KeycloakAPIUser, kcClient keycloakCommon.KeycloakInterface, realm string) error {
	// Keep a reference of all the used groups to avoid unnecesary requests
	groupPool := map[string]string{}

	for _, user := range users {
		for _, group := range user.Groups {
			if _, ok := groupPool[group]; !ok {
				foundGroup, err := kcClient.FindGroupByPath(group, realm)
				if err != nil {
					return err
				}
				if foundGroup == nil {
					r.Log.Warningf("skipping addition of user to group (group not found)", map[string]interface{}{
						"user":  user.UserName,
						"group": group,
					})
					continue
				}

				groupPool[group] = foundGroup.ID
			}

			if err := kcClient.AddUserToGroup(realm, user.ID, groupPool[group]); err != nil {
				return err
			}
		}
	}

	return nil
}

// Query the clients in a realm and return a map where the key is the client name
// (`ClientID` field) and the value is the struct with the client information
func listClientsByName(kcClient keycloakCommon.KeycloakInterface, realmName string) (map[string]*keycloak.KeycloakAPIClient, error) {
	clients, err := kcClient.ListClients(realmName)
	if err != nil {
		return nil, err
	}

	clientsByID := map[string]*keycloak.KeycloakAPIClient{}

	for _, client := range clients {
		clientsByID[client.ClientID] = client
	}

	return clientsByID, nil
}

func (r *Reconciler) reconcileConsoleLink(ctx context.Context, serverClient k8sclient.Client) error {

	cl := &consolev1.ConsoleLink{
		ObjectMeta: metav1.ObjectMeta{
			Name: userSsoConsoleLink,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, cl, func() error {
		cl.Spec = consolev1.ConsoleLinkSpec{
			ApplicationMenu: &consolev1.ApplicationMenuSpec{
				ImageURL: userSSOIcon,
				Section:  "OpenShift Managed Services",
			},
			Location: consolev1.ApplicationMenu,
			Link: consolev1.Link{
				Href: r.Config.GetHost(),
				Text: "API Management SSO",
			},
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error reconciling console link: %v", err)
	}

	return nil
}

func (r *Reconciler) deleteConsoleLink(ctx context.Context, serverClient k8sclient.Client, name string) error {
	cl := &consolev1.ConsoleLink{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	err := serverClient.Delete(ctx, cl)
	if err != nil && !k8serr.IsNotFound(err) {
		return fmt.Errorf("error removing console link: %v", err)
	}

	return nil
}
