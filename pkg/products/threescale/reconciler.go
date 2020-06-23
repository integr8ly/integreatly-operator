package threescale

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	oauthv1 "github.com/openshift/api/oauth/v1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/events"

	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"

	"github.com/sirupsen/logrus"

	"github.com/integr8ly/integreatly-operator/pkg/products/monitoring"

	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/client"
	userHelper "github.com/integr8ly/integreatly-operator/pkg/resources/user"

	threescalev1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	monitoringv1alpha1 "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"

	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/products/rhsso"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	appsv1 "github.com/openshift/api/apps/v1"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	appsv1Client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "3scale"
	manifestPackage              = "integreatly-3scale"
	apiManagerName               = "3scale"
	clientID                     = "3scale"
	rhssoIntegrationName         = "rhsso"

	s3CredentialsSecretName        = "s3-credentials"
	externalRedisSecretName        = "system-redis"
	externalBackendRedisSecretName = "backend-redis"
	externalPostgresSecretName     = "system-database"

	numberOfReplicas int64 = 2

	systemSeedSecretName          = "system-seed"
	systemMasterApiCastSecretName = "system-master-apicast"

	registrySecretName = "threescale-registry-auth"
)

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, appsv1Client appsv1Client.AppsV1Interface, oauthv1Client oauthClient.OauthV1Interface, tsClient ThreeScaleInterface, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder) (*Reconciler, error) {
	ns := installation.Spec.NamespacePrefix + defaultInstallationNamespace
	config, err := configManager.ReadThreeScale()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve threescale config: %w", err)
	}
	if config.GetNamespace() == "" {
		config.SetNamespace(ns)
		configManager.WriteConfig(config)
	}
	if config.GetOperatorNamespace() == "" {
		if installation.Spec.OperatorsInProductNamespace {
			config.SetOperatorNamespace(config.GetNamespace())
		} else {
			config.SetOperatorNamespace(config.GetNamespace() + "-operator")
		}
	}
	config.SetBlackboxTargetPathForAdminUI("/p/login/")

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
		mpm:           mpm,
		installation:  installation,
		tsClient:      tsClient,
		appsv1Client:  appsv1Client,
		oauthv1Client: oauthv1Client,
		Reconciler:    resources.NewReconciler(mpm),
		recorder:      recorder,
		logger:        logger,
	}, nil
}

type Reconciler struct {
	ConfigManager config.ConfigReadWriter
	Config        *config.ThreeScale
	mpm           marketplace.MarketplaceInterface
	installation  *integreatlyv1alpha1.RHMI
	tsClient      ThreeScaleInterface
	appsv1Client  appsv1Client.AppsV1Interface
	oauthv1Client oauthClient.OauthV1Interface
	*resources.Reconciler
	extraParams map[string]string
	recorder    record.EventRecorder
	logger      *logrus.Entry
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "system-app",
			Namespace: ns,
		},
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	logrus.Infof("Reconciling %s", r.Config.GetProductName())

	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		phase, err := resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetNamespace())
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		phase, err = resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetOperatorNamespace())
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		err = resources.RemoveOauthClient(r.oauthv1Client, r.getOAuthClientName())
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
		return integreatlyv1alpha1.PhaseCompleted, nil
	})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetOperatorNamespace(), installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s ns", r.Config.GetOperatorNamespace()), err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetNamespace(), installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s ns", r.Config.GetNamespace()), err)
		return phase, err
	}

	phase, err = r.restoreSystemSecrets(ctx, serverClient, installation)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s ns", r.Config.GetNamespace()), err)
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, r.Config.GetNamespace(), serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, fmt.Sprintf("Failed to retrieve %s ns", r.Config.GetNamespace()), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	err = resources.CopyPullSecretToNameSpace(ctx, installation.GetPullSecretSpec(), r.Config.GetNamespace(), registrySecretName, serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile pull secret", err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	preUpgradeBackups := r.preUpgradeBackupExecutor()
	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: constants.ThreeScaleSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetOperatorNamespace(), ManifestPackage: manifestPackage}, []string{r.Config.GetNamespace()}, preUpgradeBackups, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.ThreeScaleSubscriptionName), err)
		return phase, err
	}

	if r.installation.GetDeletionTimestamp() == nil {
		phase, err = r.reconcileSMTPCredentials(ctx, serverClient)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			events.HandleError(r.recorder, installation, phase, "Failed to reconcile smtp credentials", err)
			return phase, err
		}

		phase, err = r.reconcileExternalDatasources(ctx, serverClient)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			events.HandleError(r.recorder, installation, phase, "Failed to reconcile external data sources", err)
			return phase, err
		}

		phase, err = r.reconcileBlobStorage(ctx, serverClient)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			events.HandleError(r.recorder, installation, phase, "Failed to reconcile blob storage", err)
			return phase, err
		}
	}

	phase, err = r.reconcileComponents(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile components", err)
		return phase, err
	}

	logrus.Infof("%s is successfully deployed", r.Config.GetProductName())

	phase, err = r.reconcileRHSSOIntegration(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile rhsso integration", err)
		return phase, err
	}

	phase, err = r.reconcileBlackboxTargets(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile blackbox targets", err)
		return phase, err
	}

	phase, err = r.reconcileOpenshiftUsers(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile openshift users", err)
		return phase, err
	}

	clientSecret, err := r.getOauthClientSecret(ctx, serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, "Failed to get oauth client secret", err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	threescaleMasterRoute, err := r.getThreescaleRoute(ctx, serverClient, "system-master", nil)
	if err != nil || threescaleMasterRoute == nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	phase, err = r.ReconcileOauthClient(ctx, installation, &oauthv1.OAuthClient{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.getOAuthClientName(),
		},
		Secret: clientSecret,
		RedirectURIs: []string{
			"https://" + threescaleMasterRoute.Spec.Host,
		},
		GrantMethod: oauthv1.GrantHandlerAuto,
	}, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile oauth client", err)
		return phase, err
	}

	phase, err = r.reconcileServiceDiscovery(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile service discovery", err)
		return phase, err
	}

	phase, err = r.backupSystemSecrets(ctx, serverClient, installation)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile templates", err)
		return phase, err
	}

	phase, err = r.reconcileRouteEditRole(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile roles", err)
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
	phase, err = r.reconcileKubeStateMetrics3scaleAlerts(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile 3scale alerts", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	logrus.Infof("%s installation is reconciled successfully", r.Config.GetProductName())
	return integreatlyv1alpha1.PhaseCompleted, nil
}

// restores seed and master api cast secrets if available
func (r *Reconciler) restoreSystemSecrets(ctx context.Context, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {
	for _, secretName := range []string{systemSeedSecretName, systemMasterApiCastSecretName} {
		err := resources.CopySecret(ctx, serverClient, secretName, installation.Namespace, secretName, r.Config.GetNamespace())
		if err != nil {
			if !k8serr.IsNotFound(err) && !k8serr.IsConflict(err) {
				return integreatlyv1alpha1.PhaseFailed, err
			}
			logrus.Info(fmt.Sprintf("no backed up secret %v found in %v", secretName, installation.Namespace))
		}
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

// Copies the seed and master api cast secrets for later restoration
func (r *Reconciler) backupSystemSecrets(ctx context.Context, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {
	for _, secretName := range []string{systemSeedSecretName, systemMasterApiCastSecretName} {
		err := resources.CopySecret(ctx, serverClient, secretName, r.Config.GetNamespace(), secretName, installation.Namespace)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

// CreateResource Creates a generic kubernetes resource from a template
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

func (r *Reconciler) getOauthClientSecret(ctx context.Context, serverClient k8sclient.Client) (string, error) {
	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.ConfigManager.GetOauthClientsSecretName(),
		},
	}

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: oauthClientSecrets.Name, Namespace: r.ConfigManager.GetOperatorNamespace()}, oauthClientSecrets)
	if err != nil {
		return "", fmt.Errorf("Could not find %s Secret: %w", oauthClientSecrets.Name, err)
	}

	clientSecretBytes, ok := oauthClientSecrets.Data[string(r.Config.GetProductName())]
	if !ok {
		return "", fmt.Errorf("Could not find %s key in %s Secret", string(r.Config.GetProductName()), oauthClientSecrets.Name)
	}
	return string(clientSecretBytes), nil
}

func (r *Reconciler) reconcileSMTPCredentials(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	logrus.Info("Reconciling smtp credentials")

	// get the secret containing smtp credentials
	credSec := &corev1.Secret{}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: r.installation.Spec.SMTPSecret, Namespace: r.installation.Namespace}, credSec)
	if err != nil {
		logrus.Warnf("could not obtain smtp credentials secret: %v", err)
	}
	smtpConfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "system-smtp",
			Namespace: r.Config.GetNamespace(),
		},
		Data: map[string][]byte{},
	}

	// reconcile the smtp configmap for 3scale
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, smtpConfigSecret, func() error {
		owner.AddIntegreatlyOwnerAnnotations(smtpConfigSecret, r.installation)
		if smtpConfigSecret.Data == nil {
			smtpConfigSecret.Data = map[string][]byte{}
		}

		smtpUpdated := false

		if string(credSec.Data["host"]) != string(smtpConfigSecret.Data["address"]) {
			smtpConfigSecret.Data["address"] = credSec.Data["host"]
			smtpUpdated = true
		}
		if string(credSec.Data["authentication"]) != string(smtpConfigSecret.Data["authentication"]) {
			smtpConfigSecret.Data["authentication"] = credSec.Data["authentication"]
			smtpUpdated = true
		}
		if string(credSec.Data["domain"]) != string(smtpConfigSecret.Data["domain"]) {
			smtpConfigSecret.Data["domain"] = credSec.Data["domain"]
			smtpUpdated = true
		}
		if string(credSec.Data["openssl.verify.mode"]) != string(smtpConfigSecret.Data["openssl.verify.mode"]) {
			smtpConfigSecret.Data["openssl.verify.mode"] = credSec.Data["openssl.verify.mode"]
			smtpUpdated = true
		}
		if string(credSec.Data["password"]) != string(smtpConfigSecret.Data["password"]) {
			smtpConfigSecret.Data["password"] = credSec.Data["password"]
			smtpUpdated = true
		}
		if string(credSec.Data["port"]) != string(smtpConfigSecret.Data["port"]) {
			smtpConfigSecret.Data["port"] = credSec.Data["port"]
			smtpUpdated = true
		}
		if string(credSec.Data["username"]) != string(smtpConfigSecret.Data["username"]) {
			smtpConfigSecret.Data["username"] = credSec.Data["username"]
			smtpUpdated = true
		}

		if smtpUpdated {
			err = r.RolloutDeployment("system-app")
			if err != nil {
				logrus.Error(err)
			}

			err = r.RolloutDeployment("system-sidekiq")
			if err != nil {
				logrus.Error(err)
			}
		}

		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create or update 3scale smtp configmap: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	fss, err := r.getBlobStorageFileStorageSpec(ctx, serverClient)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// create the 3scale api manager
	resourceRequirements := r.installation.Spec.Type != string(integreatlyv1alpha1.InstallationTypeWorkshop)
	apim := &threescalev1.APIManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiManagerName,
			Namespace: r.Config.GetNamespace(),
		},
		Spec: threescalev1.APIManagerSpec{
			HighAvailability:    &threescalev1.HighAvailabilitySpec{},
			PodDisruptionBudget: &threescalev1.PodDisruptionBudgetSpec{},
			APIManagerCommonSpec: threescalev1.APIManagerCommonSpec{
				ResourceRequirementsEnabled: &resourceRequirements,
			},
			System: &threescalev1.SystemSpec{
				DatabaseSpec: &threescalev1.SystemDatabaseSpec{
					PostgreSQL: &threescalev1.SystemPostgreSQLSpec{},
				},
				FileStorageSpec: &threescalev1.SystemFileStorageSpec{
					S3: &threescalev1.SystemS3Spec{},
				},
				AppSpec:     &threescalev1.SystemAppSpec{Replicas: &[]int64{0}[0]},
				SidekiqSpec: &threescalev1.SystemSidekiqSpec{Replicas: &[]int64{0}[0]},
			},
			Apicast: &threescalev1.ApicastSpec{
				ProductionSpec: &threescalev1.ApicastProductionSpec{Replicas: &[]int64{0}[0]},
				StagingSpec:    &threescalev1.ApicastStagingSpec{Replicas: &[]int64{0}[0]},
			},
			Backend: &threescalev1.BackendSpec{
				ListenerSpec: &threescalev1.BackendListenerSpec{Replicas: &[]int64{0}[0]},
				WorkerSpec:   &threescalev1.BackendWorkerSpec{Replicas: &[]int64{0}[0]},
				CronSpec:     &threescalev1.BackendCronSpec{Replicas: &[]int64{0}[0]},
			},
			Zync: &threescalev1.ZyncSpec{
				AppSpec: &threescalev1.ZyncAppSpec{Replicas: &[]int64{0}[0]},
				QueSpec: &threescalev1.ZyncQueSpec{Replicas: &[]int64{0}[0]},
			},
		},
	}

	status, err := controllerutil.CreateOrUpdate(ctx, serverClient, apim, func() error {

		apim.Spec.HighAvailability = &threescalev1.HighAvailabilitySpec{Enabled: true}
		apim.Spec.APIManagerCommonSpec.ResourceRequirementsEnabled = &resourceRequirements
		apim.Spec.APIManagerCommonSpec.WildcardDomain = r.installation.Spec.RoutingSubdomain
		apim.Spec.System.FileStorageSpec = fss
		apim.Spec.PodDisruptionBudget = &threescalev1.PodDisruptionBudgetSpec{Enabled: true}

		if *apim.Spec.System.AppSpec.Replicas < numberOfReplicas {
			*apim.Spec.System.AppSpec.Replicas = numberOfReplicas
		}
		if *apim.Spec.System.SidekiqSpec.Replicas < numberOfReplicas {
			*apim.Spec.System.SidekiqSpec.Replicas = numberOfReplicas
		}
		if *apim.Spec.Apicast.ProductionSpec.Replicas < numberOfReplicas {
			*apim.Spec.Apicast.ProductionSpec.Replicas = numberOfReplicas
		}
		if *apim.Spec.Apicast.StagingSpec.Replicas < numberOfReplicas {
			*apim.Spec.Apicast.StagingSpec.Replicas = numberOfReplicas
		}
		if *apim.Spec.Backend.ListenerSpec.Replicas < numberOfReplicas {
			*apim.Spec.Backend.ListenerSpec.Replicas = numberOfReplicas
		}
		if *apim.Spec.Backend.WorkerSpec.Replicas < numberOfReplicas {
			*apim.Spec.Backend.WorkerSpec.Replicas = numberOfReplicas
		}
		if *apim.Spec.Backend.CronSpec.Replicas < numberOfReplicas {
			*apim.Spec.Backend.CronSpec.Replicas = numberOfReplicas
		}
		if *apim.Spec.Zync.AppSpec.Replicas < numberOfReplicas {
			*apim.Spec.Zync.AppSpec.Replicas = numberOfReplicas
		}
		if *apim.Spec.Zync.QueSpec.Replicas < numberOfReplicas {
			*apim.Spec.Zync.QueSpec.Replicas = numberOfReplicas
		}

		owner.AddIntegreatlyOwnerAnnotations(apim, r.installation)

		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	logrus.Info("API Manager: ", status)

	if len(apim.Status.Deployments.Starting) == 0 && len(apim.Status.Deployments.Stopped) == 0 && len(apim.Status.Deployments.Ready) > 0 {

		threescaleRoute, err := r.getThreescaleRoute(ctx, serverClient, "system-provider", func(r routev1.Route) bool {
			return strings.HasPrefix(r.Spec.Host, "3scale-admin.")
		})
		if threescaleRoute != nil {
			r.Config.SetHost("https://" + threescaleRoute.Spec.Host)
			err = r.ConfigManager.WriteConfig(r.Config)
			if err != nil {
				return integreatlyv1alpha1.PhaseFailed, err
			}
			return integreatlyv1alpha1.PhaseCompleted, nil
		} else if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	return integreatlyv1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) reconcileBlobStorage(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	logrus.Info("Reconciling blob storage")
	ns := r.installation.Namespace

	// setup blob storage cr for the cloud resource operator
	blobStorageName := fmt.Sprintf("%s%s", constants.ThreeScaleBlobStoragePrefix, r.installation.Name)
	blobStorage, err := croUtil.ReconcileBlobStorage(ctx, serverClient, defaultInstallationNamespace, r.installation.Spec.Type, croUtil.TierProduction, blobStorageName, ns, blobStorageName, ns, func(cr metav1.Object) error {
		owner.AddIntegreatlyOwnerAnnotations(cr, r.installation)
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile blob storage request: %w", err)
	}

	// wait for the blob storage cr to reconcile
	if blobStorage.Status.Phase != types.PhaseComplete {
		return integreatlyv1alpha1.PhaseAwaitingComponents, nil
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) getBlobStorageFileStorageSpec(ctx context.Context, serverClient k8sclient.Client) (*threescalev1.SystemFileStorageSpec, error) {
	// create blob storage cr
	blobStorage := &crov1.BlobStorage{}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: fmt.Sprintf("%s%s", constants.ThreeScaleBlobStoragePrefix, r.installation.Name), Namespace: r.installation.Namespace}, blobStorage)
	if err != nil {
		return nil, fmt.Errorf("failed to get blob storage custom resource: %w", err)
	}

	// get blob storage connection secret
	blobStorageSec := &corev1.Secret{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: blobStorage.Status.SecretRef.Name, Namespace: blobStorage.Status.SecretRef.Namespace}, blobStorageSec)
	if err != nil {
		return nil, fmt.Errorf("failed to get blob storage connection secret: %w", err)
	}

	// create s3 credentials secret
	credSec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s3CredentialsSecretName,
			Namespace: r.Config.GetNamespace(),
		},
		Data: map[string][]byte{},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, credSec, func() error {
		// Map known key names from CRO, and append any additional values that may be used for Minio
		for key, value := range blobStorageSec.Data {
			switch key {
			case "credentialKeyID":
				credSec.Data["AWS_ACCESS_KEY_ID"] = blobStorageSec.Data["credentialKeyID"]
			case "credentialSecretKey":
				credSec.Data["AWS_SECRET_ACCESS_KEY"] = blobStorageSec.Data["credentialSecretKey"]
			case "bucketName":
				credSec.Data["AWS_BUCKET"] = blobStorageSec.Data["bucketName"]
			case "bucketRegion":
				credSec.Data["AWS_REGION"] = blobStorageSec.Data["bucketRegion"]
			default:
				credSec.Data[key] = value
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create or update blob storage aws credentials secret: %w", err)
	}
	// return the file storage spec
	return &threescalev1.SystemFileStorageSpec{
		S3: &threescalev1.SystemS3Spec{
			ConfigurationSecretRef: corev1.LocalObjectReference{
				Name: s3CredentialsSecretName,
			},
		},
	}, nil
}

// reconcileExternalDatasources provisions 2 redis caches and a postgres instance
// which are used when 3scale HighAvailability mode is enabled
func (r *Reconciler) reconcileExternalDatasources(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	logrus.Info("Reconciling external datastores")
	ns := r.installation.Namespace

	// setup backend redis custom resource
	// this will be used by the cloud resources operator to provision a redis instance
	logrus.Info("Creating backend redis instance")
	backendRedisName := fmt.Sprintf("%s%s", constants.ThreeScaleBackendRedisPrefix, r.installation.Name)
	backendRedis, err := croUtil.ReconcileRedis(ctx, serverClient, defaultInstallationNamespace, r.installation.Spec.Type, croUtil.TierProduction, backendRedisName, ns, backendRedisName, ns, func(cr metav1.Object) error {
		owner.AddIntegreatlyOwnerAnnotations(cr, r.installation)
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile backend redis request: %w", err)
	}

	// setup system redis custom resource
	// this will be used by the cloud resources operator to provision a redis instance
	logrus.Info("Creating system redis instance")
	systemRedisName := fmt.Sprintf("%s%s", constants.ThreeScaleSystemRedisPrefix, r.installation.Name)
	systemRedis, err := croUtil.ReconcileRedis(ctx, serverClient, defaultInstallationNamespace, r.installation.Spec.Type, croUtil.TierProduction, systemRedisName, ns, systemRedisName, ns, func(cr metav1.Object) error {
		owner.AddIntegreatlyOwnerAnnotations(cr, r.installation)
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile system redis request: %w", err)
	}

	// setup postgres cr for the cloud resource operator
	// this will be used by the cloud resources operator to provision a postgres instance
	logrus.Info("Creating postgres instance")
	postgresName := fmt.Sprintf("%s%s", constants.ThreeScalePostgresPrefix, r.installation.Name)
	postgres, err := croUtil.ReconcilePostgres(ctx, serverClient, defaultInstallationNamespace, r.installation.Spec.Type, croUtil.TierProduction, postgresName, ns, postgresName, ns, func(cr metav1.Object) error {
		owner.AddIntegreatlyOwnerAnnotations(cr, r.installation)
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile postgres request: %w", err)
	}

	// redis cr returning a failed state
	_, err = resources.CreateRedisResourceStatusPhaseFailedAlert(ctx, serverClient, r.installation, backendRedis)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create redis failure alert: %w", err)
	}

	// wait for the backend redis cr to reconcile
	if backendRedis.Status.Phase != types.PhaseComplete {
		return integreatlyv1alpha1.PhaseAwaitingComponents, nil
	}

	// create prometheus pending rule
	_, err = resources.CreateRedisResourceStatusPhasePendingAlert(ctx, serverClient, r.installation, backendRedis)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create redis pending alert: %w", err)
	}

	// create the prometheus availability rule
	if _, err = resources.CreateRedisAvailabilityAlert(ctx, serverClient, r.installation, backendRedis); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create backend redis prometheus alert for threescale: %w", err)
	}
	// create backend connectivity alert
	if _, err = resources.CreateRedisConnectivityAlert(ctx, serverClient, r.installation, backendRedis); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create backend redis prometheus connectivity alert for threescale: %s", err)
	}

	// get the secret created by the cloud resources operator
	// containing backend redis connection details
	credSec := &corev1.Secret{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: backendRedis.Status.SecretRef.Name, Namespace: backendRedis.Status.SecretRef.Namespace}, credSec)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get backend redis credential secret: %w", err)
	}

	// create backend redis external connection secret needed for the 3scale apimanager
	backendRedisSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      externalBackendRedisSecretName,
			Namespace: r.Config.GetNamespace(),
		},
		Data: map[string][]byte{},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, backendRedisSecret, func() error {
		uri := credSec.Data["uri"]
		port := credSec.Data["port"]
		backendRedisSecret.Data["REDIS_STORAGE_URL"] = []byte(fmt.Sprintf("redis://%s:%s/0", uri, port))
		backendRedisSecret.Data["REDIS_QUEUES_URL"] = []byte(fmt.Sprintf("redis://%s:%s/1", uri, port))
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create or update 3scale %s connection secret: %w", externalBackendRedisSecretName, err)
	}

	// create prometheus failure rule
	_, err = resources.CreateRedisResourceStatusPhaseFailedAlert(ctx, serverClient, r.installation, systemRedis)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create system redis failure alert: %w", err)
	}

	// wait for the system redis cr to reconcile
	if systemRedis.Status.Phase != types.PhaseComplete {
		return integreatlyv1alpha1.PhaseAwaitingComponents, nil
	}

	// create prometheus pending rule
	_, err = resources.CreateRedisResourceStatusPhasePendingAlert(ctx, serverClient, r.installation, systemRedis)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create system redis pending alert: %w", err)
	}

	// create the prometheus availability rule
	_, err = resources.CreateRedisAvailabilityAlert(ctx, serverClient, r.installation, systemRedis)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create system redis prometheus alert for threescale: %w", err)
	}
	// create system redis connectivity alert
	_, err = resources.CreateRedisConnectivityAlert(ctx, serverClient, r.installation, systemRedis)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create system redis prometheus connectivity alert for threescale: %s", err)
	}

	// get the secret created by the cloud resources operator
	// containing system redis connection details
	systemCredSec := &corev1.Secret{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: systemRedis.Status.SecretRef.Name, Namespace: systemRedis.Status.SecretRef.Namespace}, systemCredSec)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get system redis credential secret: %w", err)
	}

	// create system redis external connection secret needed for the 3scale apimanager
	redisSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      externalRedisSecretName,
			Namespace: r.Config.GetNamespace(),
		},
		Data: map[string][]byte{},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, redisSecret, func() error {
		uri := systemCredSec.Data["uri"]
		port := systemCredSec.Data["port"]
		conn := fmt.Sprintf("redis://%s:%s/1", uri, port)
		redisSecret.Data["URL"] = []byte(conn)
		redisSecret.Data["MESSAGE_BUS_URL"] = []byte(conn)
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create or update 3scale %s connection secret: %w", externalRedisSecretName, err)
	}

	// cr returning a failed state
	_, err = resources.CreatePostgresResourceStatusPhaseFailedAlert(ctx, serverClient, r.installation, postgres)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres failure alert: %w", err)
	}

	// wait for the postgres cr to reconcile
	if postgres.Status.Phase != types.PhaseComplete {
		return integreatlyv1alpha1.PhaseAwaitingComponents, nil
	}

	// create prometheus pending rule
	_, err = resources.CreatePostgresResourceStatusPhasePendingAlert(ctx, serverClient, r.installation, postgres)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres pending alert: %w", err)
	}

	// create the prometheus availability rule
	_, err = resources.CreatePostgresAvailabilityAlert(ctx, serverClient, r.installation, postgres)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres prometheus alert for threescale: %w", err)
	}
	// create postgres connectivity alert
	_, err = resources.CreatePostgresConnectivityAlert(ctx, serverClient, r.installation, postgres)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create postgres prometheus connectivity alert for threescale: %s", err)
	}

	// get the secret containing redis credentials
	postgresCredSec := &corev1.Secret{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: postgres.Status.SecretRef.Name, Namespace: postgres.Status.SecretRef.Namespace}, postgresCredSec)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get postgres credential secret: %w", err)
	}

	// create postgres external connection secret
	postgresSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      externalPostgresSecretName,
			Namespace: r.Config.GetNamespace(),
		},
		Data: map[string][]byte{},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, postgresSecret, func() error {
		username := postgresCredSec.Data["username"]
		password := postgresCredSec.Data["password"]
		url := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", username, password, postgresCredSec.Data["host"], postgresCredSec.Data["port"], postgresCredSec.Data["database"])

		postgresSecret.Data["URL"] = []byte(url)
		postgresSecret.Data["DB_USER"] = username
		postgresSecret.Data["DB_PASSWORD"] = password
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create or update 3scale %s connection secret: %w", externalPostgresSecretName, err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileRHSSOIntegration(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	rhssoConfig, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	rhssoNamespace := rhssoConfig.GetNamespace()
	rhssoRealm := rhssoConfig.GetRealm()
	if rhssoNamespace == "" || rhssoRealm == "" {
		logrus.Info("Cannot configure SSO integration without SSO ns and SSO realm")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	kcClient := &keycloak.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clientID,
			Namespace: rhssoNamespace,
		},
	}

	clientSecret, err := r.getOauthClientSecret(ctx, serverClient)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, kcClient, func() error {
		kcClient.Spec = r.getKeycloakClientSpec(clientSecret)
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not create/update 3scale keycloak client: %w", err)
	}

	accessToken, err := r.GetAdminToken(ctx, serverClient)
	if err != nil {
		return integreatlyv1alpha1.PhaseInProgress, err
	}
	_, err = r.tsClient.GetAuthenticationProviderByName(rhssoIntegrationName, *accessToken)
	if err != nil && !tsIsNotFoundError(err) {
		return integreatlyv1alpha1.PhaseInProgress, err
	}
	if tsIsNotFoundError(err) {
		site := rhssoConfig.GetHost() + "/auth/realms/" + rhssoRealm
		res, err := r.tsClient.AddAuthenticationProvider(map[string]string{
			"kind":                              "keycloak",
			"name":                              rhssoIntegrationName,
			"client_id":                         clientID,
			"client_secret":                     clientSecret,
			"site":                              site,
			"skip_ssl_certificate_verification": "true",
			"published":                         "true",
		}, *accessToken)
		if err != nil || res.StatusCode != http.StatusCreated {
			return integreatlyv1alpha1.PhaseInProgress, err
		}
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) getOAuthClientName() string {
	return r.installation.Spec.NamespacePrefix + string(r.Config.GetProductName())
}

func (r *Reconciler) reconcileOpenshiftUsers(ctx context.Context, installation *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	logrus.Info("Reconciling openshift users to 3scale")

	rhssoConfig, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	accessToken, err := r.GetAdminToken(ctx, serverClient)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	systemAdminUsername, _, err := r.GetAdminNameAndPassFromSecret(ctx, serverClient)
	if err != nil {
		return integreatlyv1alpha1.PhaseInProgress, err
	}

	kcu, err := rhsso.GetKeycloakUsers(ctx, serverClient, rhssoConfig.GetNamespace())
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	tsUsers, err := r.tsClient.GetUsers(*accessToken)
	if err != nil {
		return integreatlyv1alpha1.PhaseInProgress, err
	}

	added, deleted := r.getUserDiff(kcu, tsUsers.Users)
	for _, kcUser := range added {
		res, err := r.tsClient.AddUser(kcUser.UserName, kcUser.Email, "", *accessToken)
		if err != nil || res.StatusCode != http.StatusCreated {
			return integreatlyv1alpha1.PhaseInProgress, err
		}
	}
	for _, tsUser := range deleted {
		if tsUser.UserDetails.Username != *systemAdminUsername {
			res, err := r.tsClient.DeleteUser(tsUser.UserDetails.Id, *accessToken)
			if err != nil || res.StatusCode != http.StatusOK {
				return integreatlyv1alpha1.PhaseInProgress, err
			}
		}
	}

	// update KeycloakUser attribute after user is created in 3scale
	userCreated3ScaleName := "3scale_user_created"
	for _, user := range kcu {
		if user.Attributes == nil {
			user.Attributes = map[string][]string{
				userCreated3ScaleName: {"true"},
			}
		}

		kcUser := &keycloak.KeycloakUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      userHelper.GetValidGeneratedUserName(user),
				Namespace: rhssoConfig.GetNamespace(),
			},
		}

		_, err := controllerutil.CreateOrUpdate(ctx, serverClient, kcUser, func() error {
			user.Attributes[userCreated3ScaleName] = []string{"true"}
			kcUser.Spec.User = user
			return nil
		})
		if err != nil {
			return integreatlyv1alpha1.PhaseInProgress,
				fmt.Errorf("failed to update KeycloakUser CR with %s attribute: %w", userCreated3ScaleName, err)
		}
	}

	openshiftAdminGroup := &usersv1.Group{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: "dedicated-admins"}, openshiftAdminGroup)
	if err != nil && !k8serr.IsNotFound(err) {
		return integreatlyv1alpha1.PhaseInProgress, err
	}
	newTsUsers, err := r.tsClient.GetUsers(*accessToken)
	if err != nil {
		return integreatlyv1alpha1.PhaseInProgress, err
	}

	isWorkshop := installation.Spec.Type == string(integreatlyv1alpha1.InstallationTypeWorkshop)

	err = syncOpenshiftAdminMembership(openshiftAdminGroup, newTsUsers, *systemAdminUsername, isWorkshop, r.tsClient, *accessToken)
	if err != nil {
		return integreatlyv1alpha1.PhaseInProgress, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) preUpgradeBackupExecutor() backup.BackupExecutor {
	if r.installation.Spec.UseClusterStorage != "false" {
		return backup.NewNoopBackupExecutor()
	}

	return backup.NewConcurrentBackupExecutor(
		backup.NewAWSBackupExecutor(
			r.installation.Namespace,
			"threescale-postgres-rhmi",
			backup.PostgresSnapshotType,
		),
		backup.NewAWSBackupExecutor(
			r.installation.Namespace,
			"threescale-backend-redis-rhmi",
			backup.RedisSnapshotType,
		),
		backup.NewAWSBackupExecutor(
			r.installation.Namespace,
			"threescale-redis-rhmi",
			backup.RedisSnapshotType,
		),
	)
}

func syncOpenshiftAdminMembership(openshiftAdminGroup *usersv1.Group, newTsUsers *Users, systemAdminUsername string, isWorkshop bool, tsClient ThreeScaleInterface, accessToken string) error {
	for _, tsUser := range newTsUsers.Users {
		// skip if ts user is the system user admin
		if tsUser.UserDetails.Username == systemAdminUsername {
			continue
		}

		// In workshop mode, developer users also get admin permissions in 3scale
		if (userIsOpenshiftAdmin(tsUser, openshiftAdminGroup) || isWorkshop) && tsUser.UserDetails.Role != adminRole {
			res, err := tsClient.SetUserAsAdmin(tsUser.UserDetails.Id, accessToken)
			if err != nil || res.StatusCode != http.StatusOK {
				return err
			}
		}
	}

	return nil
}

func (r *Reconciler) reconcileServiceDiscovery(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	if string(r.Config.GetProductVersion()) != string(integreatlyv1alpha1.Version3Scale) {
		r.Config.SetProductVersion(string(integreatlyv1alpha1.Version3Scale))
		r.ConfigManager.WriteConfig(r.Config)
	}

	if string(r.Config.GetOperatorVersion()) != string(integreatlyv1alpha1.OperatorVersion3Scale) {
		r.Config.SetOperatorVersion(string(integreatlyv1alpha1.OperatorVersion3Scale))
		r.ConfigManager.WriteConfig(r.Config)
	}

	system := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "system",
			Namespace: r.Config.GetNamespace(),
		},
	}

	status, err := controllerutil.CreateOrUpdate(ctx, serverClient, system, func() error {
		clientSecret, err := r.getOauthClientSecret(ctx, serverClient)
		if err != nil {
			return err
		}
		sdConfig := fmt.Sprintf("production:\n  enabled: true\n  authentication_method: oauth\n  oauth_server_type: builtin\n  client_id: '%s'\n  client_secret: '%s'\n", r.getOAuthClientName(), clientSecret)

		system.Data["service_discovery.yml"] = sdConfig
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseInProgress, err
	}

	if status != controllerutil.OperationResultNone {
		err = r.RolloutDeployment("system-app")
		if err != nil {
			return integreatlyv1alpha1.PhaseInProgress, err
		}

		err = r.RolloutDeployment("system-sidekiq")
		if err != nil {
			return integreatlyv1alpha1.PhaseInProgress, err
		}
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileBlackboxTargets(ctx context.Context, installation *integreatlyv1alpha1.RHMI, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	cfg, err := r.ConfigManager.ReadMonitoring()
	if err != nil {
		return integreatlyv1alpha1.PhaseInProgress, fmt.Errorf("error reading monitoring config: %w", err)
	}

	err = monitoring.CreateBlackboxTarget(ctx, "integreatly-3scale-admin-ui", monitoringv1alpha1.BlackboxtargetData{
		Url:     r.Config.GetHost() + "/" + r.Config.GetBlackboxTargetPathForAdminUI(),
		Service: "3scale-admin-ui",
	}, cfg, installation, client)
	if err != nil {
		return integreatlyv1alpha1.PhaseInProgress, fmt.Errorf("error creating threescale blackbox target: %w", err)
	}

	// Create a blackbox target for the developer console ui
	route, err := r.getThreescaleRoute(ctx, client, "system-developer", func(r routev1.Route) bool {
		return strings.HasPrefix(r.Spec.Host, "3scale.")
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseInProgress, fmt.Errorf("error getting threescale system-developer route: %w", err)
	}
	err = monitoring.CreateBlackboxTarget(ctx, "integreatly-3scale-system-developer", monitoringv1alpha1.BlackboxtargetData{
		Url:     "https://" + route.Spec.Host,
		Service: "3scale-developer-console-ui",
	}, cfg, installation, client)
	if err != nil {
		return integreatlyv1alpha1.PhaseInProgress, fmt.Errorf("error creating threescale blackbox target (system-developer): %w", err)
	}

	// Create a blackbox target for the master console ui
	route, err = r.getThreescaleRoute(ctx, client, "system-master", nil)
	if err != nil {
		return integreatlyv1alpha1.PhaseInProgress, fmt.Errorf("error getting threescale system-master route: %w", err)
	}
	err = monitoring.CreateBlackboxTarget(ctx, "integreatly-3scale-system-master", monitoringv1alpha1.BlackboxtargetData{
		Url:     "https://" + route.Spec.Host,
		Service: "3scale-system-admin-ui",
	}, cfg, installation, client)
	if err != nil {
		return integreatlyv1alpha1.PhaseInProgress, fmt.Errorf("error creating threescale blackbox target (system-master): %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) getThreescaleRoute(ctx context.Context, serverClient k8sclient.Client, label string, filterFn func(r routev1.Route) bool) (*routev1.Route, error) {
	// Add backwards compatible filter function, first element will do
	if filterFn == nil {
		filterFn = func(r routev1.Route) bool { return true }
	}

	selector, err := labels.Parse(fmt.Sprintf("zync.3scale.net/route-to=%v", label))
	if err != nil {
		return nil, err
	}

	opts := k8sclient.ListOptions{
		LabelSelector: selector,
		Namespace:     r.Config.GetNamespace(),
	}

	routes := routev1.RouteList{}
	err = serverClient.List(ctx, &routes, &opts)
	if err != nil {
		return nil, err
	}

	if len(routes.Items) == 0 {
		return nil, nil
	}

	var foundRoute *routev1.Route
	for _, route := range routes.Items {
		if filterFn(route) {
			foundRoute = &route
			break
		}
	}
	return foundRoute, nil
}

func (r *Reconciler) GetAdminNameAndPassFromSecret(ctx context.Context, serverClient k8sclient.Client) (*string, *string, error) {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      "system-seed",
		},
	}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: s.Name, Namespace: r.Config.GetNamespace()}, s)
	if err != nil {
		return nil, nil, err
	}

	username := string(s.Data["ADMIN_USER"])
	email := string(s.Data["ADMIN_EMAIL"])
	return &username, &email, nil
}

func (r *Reconciler) SetAdminDetailsOnSecret(ctx context.Context, serverClient k8sclient.Client, username string, email string) error {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      "system-seed",
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, s, func() error {
		s.Data["ADMIN_USER"] = []byte(username)
		s.Data["ADMIN_EMAIL"] = []byte(email)
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) GetAdminToken(ctx context.Context, serverClient k8sclient.Client) (*string, error) {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "system-seed",
		},
	}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: s.Name, Namespace: r.Config.GetNamespace()}, s)
	if err != nil {
		return nil, err
	}

	accessToken := string(s.Data["ADMIN_ACCESS_TOKEN"])
	return &accessToken, nil
}

func (r *Reconciler) RolloutDeployment(name string) error {
	_, err := r.appsv1Client.DeploymentConfigs(r.Config.GetNamespace()).Instantiate(name, &appsv1.DeploymentRequest{
		Name:   name,
		Force:  true,
		Latest: true,
	})

	return err
}

func (r *Reconciler) getUserDiff(kcUsers []keycloak.KeycloakAPIUser, tsUsers []*User) ([]keycloak.KeycloakAPIUser, []*User) {
	var added []keycloak.KeycloakAPIUser
	for _, kcUser := range kcUsers {
		if !tsContainsKc(tsUsers, kcUser) {
			added = append(added, kcUser)
		}
	}

	var deleted []*User
	for _, tsUser := range tsUsers {
		if !kcContainsTs(kcUsers, tsUser) {
			deleted = append(deleted, tsUser)
		}
	}

	return added, deleted
}

func kcContainsTs(kcUsers []keycloak.KeycloakAPIUser, tsUser *User) bool {
	for _, kcu := range kcUsers {
		if kcu.UserName == tsUser.UserDetails.Username {
			return true
		}
	}

	return false
}

func tsContainsKc(tsusers []*User, kcUser keycloak.KeycloakAPIUser) bool {
	for _, tsu := range tsusers {
		if tsu.UserDetails.Username == kcUser.UserName {
			return true
		}
	}

	return false
}

func userIsOpenshiftAdmin(tsUser *User, adminGroup *usersv1.Group) bool {
	for _, userName := range adminGroup.Users {
		if tsUser.UserDetails.Username == userName {
			return true
		}
	}

	return false
}

func (r *Reconciler) getKeycloakClientSpec(clientSecret string) keycloak.KeycloakClientSpec {
	return keycloak.KeycloakClientSpec{
		RealmSelector: &metav1.LabelSelector{
			MatchLabels: rhsso.GetInstanceLabels(),
		},
		Client: &keycloak.KeycloakAPIClient{
			ID:                      clientID,
			ClientID:                clientID,
			Enabled:                 true,
			Secret:                  clientSecret,
			ClientAuthenticatorType: "client-secret",
			RedirectUris: []string{
				fmt.Sprintf("https://3scale-admin.%s/*", r.installation.Spec.RoutingSubdomain),
			},
			StandardFlowEnabled: true,
			RootURL:             fmt.Sprintf("https://3scale-admin.%s", r.installation.Spec.RoutingSubdomain),
			FullScopeAllowed:    true,
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
					Name:            "email verified",
					Protocol:        "openid-connect",
					ProtocolMapper:  "oidc-usermodel-property-mapper",
					ConsentRequired: true,
					ConsentText:     "${emailVerified}",
					Config: map[string]string{
						"userinfo.token.claim": "true",
						"user.attribute":       "emailVerified",
						"id.token.claim":       "true",
						"access.token.claim":   "true",
						"claim.name":           "email_verified",
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
						"id.token.claim":     "true",
						"access.token.claim": "true",
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
					Name:            "org_name",
					Protocol:        "openid-connect",
					ProtocolMapper:  "oidc-usermodel-property-mapper",
					ConsentRequired: false,
					ConsentText:     "n.a.",
					Config: map[string]string{
						"userinfo.token.claim": "true",
						"user.attribute":       "org_name",
						"id.token.claim":       "true",
						"access.token.claim":   "true",
						"claim.name":           "org_name",
						"jsonType.label":       "String",
					},
				},
			},
		},
	}
}

func (r *Reconciler) reconcileRouteEditRole(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	// Allow dedicated-admin group to edit routes. This is enabled to allow the public API in 3Scale, on private clusters, to be exposed.
	// This is achieved by labelling the route to match the additional router created by SRE for private clusters. INTLY-7398.

	logrus.Infof("reconciling edit routes role to the dedicated admins group")

	editRoutesRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "edit-3scale-routes",
			Namespace: r.Config.GetNamespace(),
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, editRoutesRole, func() error {
		owner.AddIntegreatlyOwnerAnnotations(editRoutesRole, r.installation)

		editRoutesRole.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{"route.openshift.io"},
				Resources: []string{"routes"},
				Verbs:     []string{"get", "update", "list", "watch", "patch"},
			},
		}

		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Failed reconciling edit routes role %v: %w", editRoutesRole, err)
	}

	// Bind the amq online service admin role to the dedicated-admins group
	editRoutesRoleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dedicated-admins-edit-routes",
			Namespace: r.Config.GetNamespace(),
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, client, editRoutesRoleBinding, func() error {
		owner.AddIntegreatlyOwnerAnnotations(editRoutesRoleBinding, r.installation)

		editRoutesRoleBinding.RoleRef = rbacv1.RoleRef{
			Name: editRoutesRole.GetName(),
			Kind: "Role",
		}
		editRoutesRoleBinding.Subjects = []rbacv1.Subject{
			{
				Name: "dedicated-admins",
				Kind: "Group",
			},
		}

		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Failed reconciling service admin role binding %v: %w", editRoutesRoleBinding, err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
