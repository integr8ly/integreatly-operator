package threescale

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	marin3rv1alpha "github.com/3scale/marin3r/pkg/apis/marin3r/v1alpha1"
	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoycore "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_v2_endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	v2route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	yaml "github.com/ghodss/yaml"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	ptypes "github.com/golang/protobuf/ptypes"
	wrappers "github.com/golang/protobuf/ptypes/wrappers"
	consolev1 "github.com/openshift/api/console/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	structpb "google.golang.org/protobuf/types/known/structpb"

	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/ratelimit"

	"github.com/integr8ly/integreatly-operator/pkg/products/monitoring"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	"github.com/integr8ly/integreatly-operator/version"
	"github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

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

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	appsv1 "github.com/openshift/api/apps/v1"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	appsv1Client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"
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

	numberOfReplicas              int64 = 2
	apicastStagingName                  = "apicast-staging"
	apicastProductionName               = "apicast-production"
	systemSeedSecretName                = "system-seed"
	systemMasterApiCastSecretName       = "system-master-apicast"

	apicastRatelimiting = "apicast-ratelimit"
	registrySecretName  = "threescale-registry-auth"

	threeScaleIcon = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAxMDAgMTAwIj48ZGVmcz48c3R5bGU+LmNscy0xe2ZpbGw6I2Q3MWUwMDt9LmNscy0ye2ZpbGw6I2MyMWEwMDt9LmNscy0ze2ZpbGw6I2ZmZjt9PC9zdHlsZT48L2RlZnM+PHRpdGxlPnByb2R1Y3RpY29uc18xMDE3X1JHQl9BUEkgZmluYWwgY29sb3I8L3RpdGxlPjxnIGlkPSJMYXllcl8xIiBkYXRhLW5hbWU9IkxheWVyIDEiPjxjaXJjbGUgY2xhc3M9ImNscy0xIiBjeD0iNTAiIGN5PSI1MCIgcj0iNTAiIHRyYW5zZm9ybT0idHJhbnNsYXRlKC0yMC43MSA1MCkgcm90YXRlKC00NSkiLz48cGF0aCBjbGFzcz0iY2xzLTIiIGQ9Ik04NS4zNiwxNC42NEE1MCw1MCwwLDAsMSwxNC42NCw4NS4zNloiLz48cGF0aCBjbGFzcz0iY2xzLTMiIGQ9Ik01MC4yNSwzMC44M2EyLjY5LDIuNjksMCwxLDAtMi42OC0yLjY5QTIuNjUsMi42NSwwLDAsMCw1MC4yNSwzMC44M1pNNDMuMzYsMzkuNGEzLjM1LDMuMzUsMCwwLDAsMy4zMiwzLjM0LDMuMzQsMy4zNCwwLDAsMCwwLTYuNjdBMy4zNSwzLjM1LDAsMCwwLDQzLjM2LDM5LjRabTMuOTIsOS44OUEyLjY4LDIuNjgsMCwxLDAsNDQuNiw1MiwyLjcsMi43LDAsMCwwLDQ3LjI4LDQ5LjI5Wk0zMi42MywyOS42NWEzLjI2LDMuMjYsMCwxLDAtMy4yNC0zLjI2QTMuMjYsMy4yNiwwLDAsMCwzMi42MywyOS42NVpNNDAuNTMsMzRhMi43NywyLjc3LDAsMCwwLDAtNS41MywyLjc5LDIuNzksMCwwLDAtMi43NiwyLjc3QTIuODUsMi44NSwwLDAsMCw0MC41MywzNFptMS43Ni05LjMxYTQuNCw0LjQsMCwxLDAtNC4zOC00LjRBNC4zNyw0LjM3LDAsMCwwLDQyLjI5LDI0LjcxWk0zMi43OCw0OWE3LDcsMCwxLDAtNy03QTcsNywwLDAsMCwzMi43OCw0OVptMzIuMTMtNy43YTQuMjMsNC4yMywwLDAsMCw0LjMsNC4zMSw0LjMxLDQuMzEsMCwxLDAtNC4zLTQuMzFabTYuOSwxMC4wNmEzLjA4LDMuMDgsMCwxLDAsMy4wOC0zLjA5QTMuMDksMy4wOSwwLDAsMCw3MS44MSw1MS4zOFpNNzMuOSwzNC43N2E0LjMxLDQuMzEsMCwxLDAtNC4zLTQuMzFBNC4yOCw0LjI4LDAsMCwwLDczLjksMzQuNzdaTTUyLjE2LDQ1LjA2YTMuNjUsMy42NSwwLDEsMCwzLjY1LTMuNjZBMy42NCwzLjY0LDAsMCwwLDUyLjE2LDQ1LjA2Wk01NSwyMmEzLjE3LDMuMTcsMCwwLDAsMy4xNi0zLjE3QTMuMjMsMy4yMywwLDAsMCw1NSwxNS42MywzLjE3LDMuMTcsMCwwLDAsNTUsMjJabS0uNDcsMTAuMDlBNS4zNyw1LjM3LDAsMCwwLDYwLDM3LjU0YTUuNDgsNS40OCwwLDEsMC01LjQ1LTUuNDhaTTY2LjI1LDI1LjVhMi42OSwyLjY5LDAsMSwwLTIuNjgtMi42OUEyLjY1LDIuNjUsMCwwLDAsNjYuMjUsMjUuNVpNNDUuNyw2My4xYTMuNDIsMy40MiwwLDEsMC0zLjQxLTMuNDJBMy40MywzLjQzLDAsMCwwLDQ1LjcsNjMuMVptMTQsMTEuMTlhNC40LDQuNCwwLDEsMCw0LjM4LDQuNEE0LjM3LDQuMzcsMCwwLDAsNTkuNzMsNzQuMjlaTTYyLjMsNTAuNTFhOS4yLDkuMiwwLDEsMCw5LjE2LDkuMkE5LjIyLDkuMjIsMCwwLDAsNjIuMyw1MC41MVpNNTAuMSw2Ni43N2EyLjY5LDIuNjksMCwxLDAsMi42OCwyLjY5QTIuNywyLjcsMCwwLDAsNTAuMSw2Ni43N1pNODEuMjUsNDEuMTJhMi43LDIuNywwLDAsMC0yLjY4LDIuNjksMi42NSwyLjY1LDAsMCwwLDIuNjgsMi42OSwyLjY5LDIuNjksMCwwLDAsMC01LjM3Wk00NC40OSw3Ni40N2EzLjczLDMuNzMsMCwwLDAtMy43MywzLjc0LDMuNzcsMy43NywwLDEsMCwzLjczLTMuNzRaTTc5LjA2LDU2LjcyYTQsNCwwLDEsMCw0LDRBNCw0LDAsMCwwLDc5LjA2LDU2LjcyWm0tNiwxMS43OEEzLjA5LDMuMDksMCwwLDAsNzAsNzEuNmEzLDMsMCwwLDAsMy4wOCwzLjA5LDMuMDksMy4wOSwwLDAsMCwwLTYuMTlaTTI4LjMsNjhhNC4xNiw0LjE2LDAsMCwwLTQuMTQsNC4xNUE0LjIxLDQuMjEsMCwwLDAsMjguMyw3Ni4zYTQuMTUsNC4xNSwwLDAsMCwwLTguM1ptLTguMjItOWEzLDMsMCwxLDAsMywzQTMuMDUsMy4wNSwwLDAsMCwyMC4wOCw1OVptMS44NC05Ljc0YTMsMywwLDEsMCwzLDNBMy4wNSwzLjA1LDAsMCwwLDIxLjkxLDQ5LjIyWk0yMi4zNyw0MmEzLjI0LDMuMjQsMCwxLDAtMy4yNCwzLjI2QTMuMjYsMy4yNiwwLDAsMCwyMi4zNyw0MlpNNDMuMTEsNzAuMmEzLjgsMy44LDAsMCwwLTMuODEtMy43NCwzLjczLDMuNzMsMCwwLDAtMy43MywzLjc0QTMuOCwzLjgsMCwwLDAsMzkuMyw3NCwzLjg3LDMuODcsMCwwLDAsNDMuMTEsNzAuMlpNMzcuNTYsNTguNDNhNC42OCw0LjY4LDAsMCwwLTQuNjItNC42NCw0LjYzLDQuNjMsMCwwLDAtNC42Miw0LjY0LDQuNTgsNC41OCwwLDAsMCw0LjYyLDQuNjRBNC42Myw0LjYzLDAsMCwwLDM3LjU2LDU4LjQzWk0yMy4xMSwzMy44MmEyLjUyLDIuNTIsMCwxLDAtMi41MS0yLjUyQTIuNTMsMi41MywwLDAsMCwyMy4xMSwzMy44MloiLz48L2c+PC9zdmc+"
)

var (
	threeScaleDeploymentConfigs = []string{
		"apicast-production",
		"apicast-staging",
		"backend-cron",
		"backend-listener",
		"backend-worker",
		"system-app",
		"system-memcache",
		"system-sidekiq",
		"system-sphinx",
		"zync",
		"zync-database",
		"zync-que",
	}
)

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, appsv1Client appsv1Client.AppsV1Interface, oauthv1Client oauthClient.OauthV1Interface, tsClient ThreeScaleInterface, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder) (*Reconciler, error) {
	ns := installation.Spec.NamespacePrefix + defaultInstallationNamespace
	threescaleConfig, err := configManager.ReadThreeScale()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve threescale config: %w", err)
	}
	if threescaleConfig.GetNamespace() == "" {
		threescaleConfig.SetNamespace(ns)
		configManager.WriteConfig(threescaleConfig)
	}
	if threescaleConfig.GetOperatorNamespace() == "" {
		if installation.Spec.OperatorsInProductNamespace {
			threescaleConfig.SetOperatorNamespace(threescaleConfig.GetNamespace())
		} else {
			threescaleConfig.SetOperatorNamespace(threescaleConfig.GetNamespace() + "-operator")
		}
	}
	threescaleConfig.SetBlackboxTargetPathForAdminUI("/p/login/")

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		ConfigManager: configManager,
		Config:        threescaleConfig,
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

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return version.VerifyProductAndOperatorVersion(
		installation.Status.Stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.Product3Scale],
		string(integreatlyv1alpha1.Version3Scale),
		string(integreatlyv1alpha1.OperatorVersion3Scale),
	)
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	logrus.Infof("Reconciling %s", r.Config.GetProductName())

	operatorNamespace := r.Config.GetOperatorNamespace()
	productNamespace := r.Config.GetNamespace()

	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		if installation.Spec.Type == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
			phase, err := ratelimit.DeleteEnvoyConfigsInNamespaces(ctx, serverClient, productNamespace)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
		}

		phase, err := resources.RemoveNamespace(ctx, installation, serverClient, productNamespace)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		phase, err = resources.RemoveNamespace(ctx, installation, serverClient, operatorNamespace)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		err = resources.RemoveOauthClient(r.oauthv1Client, r.getOAuthClientName())
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		phase, err = r.deleteConsoleLink(ctx, serverClient)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		return integreatlyv1alpha1.PhaseCompleted, nil
	})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, operatorNamespace, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s ns", operatorNamespace), err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, productNamespace, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s ns", productNamespace), err)
		return phase, err
	}

	phase, err = r.restoreSystemSecrets(ctx, serverClient, installation)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s ns", productNamespace), err)
		return phase, err
	}

	err = resources.CopyPullSecretToNameSpace(ctx, installation.GetPullSecretSpec(), productNamespace, registrySecretName, serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile pull secret", err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	phase, err = r.reconcileSubscription(ctx, serverClient, installation, productNamespace, operatorNamespace)
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
	logrus.Infof("Phase: %s reconcileComponents", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile components", err)
		return phase, err
	}

	logrus.Infof("%s is successfully deployed", r.Config.GetProductName())

	phase, err = r.reconcileRHSSOIntegration(ctx, serverClient)
	logrus.Infof("Phase: %s reconcileRHSSOIntegration", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile rhsso integration", err)
		return phase, err
	}

	phase, err = r.reconcileBlackboxTargets(ctx, installation, serverClient)
	logrus.Infof("Phase: %s reconcileBlackboxTargets", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile blackbox targets", err)
		return phase, err
	}

	phase, err = r.reconcileOpenshiftUsers(ctx, installation, serverClient)
	logrus.Infof("Phase: %s reconcileOpenshiftUsers", phase)
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
	logrus.Infof("Phase: %s ReconcileOauthClient", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile oauth client", err)
		return phase, err
	}

	phase, err = r.reconcileServiceDiscovery(ctx, serverClient)
	logrus.Infof("Phase: %s reconcileServiceDiscovery", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile service discovery", err)
		return phase, err
	}

	phase, err = r.backupSystemSecrets(ctx, serverClient, installation)
	logrus.Infof("Phase: %s backupSystemSecrets", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile templates", err)
		return phase, err
	}

	phase, err = r.reconcileRouteEditRole(ctx, serverClient)
	logrus.Infof("Phase: %s reconcileRouteEditRole", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile roles", err)
		return phase, err
	}

	if installation.Spec.Type == string(integreatlyv1alpha1.InstallationTypeManagedApi) {

		apicasts := []string{apicastStagingName, apicastProductionName}
		for _, apicast := range apicasts {
			deploymentConfig, phase, err := r.getDeploymentConfig(context.TODO(), serverClient, apicast)
			if err != nil {
				return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get deployment config for %s: %w", apicast, err)
			}
			if phase == integreatlyv1alpha1.PhaseAwaitingComponents {
				return phase, nil
			}

			service, phase, err := r.getService(context.TODO(), serverClient, apicast)
			if err != nil {
				return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get service for %s: %w", apicast, err)
			}
			if phase == integreatlyv1alpha1.PhaseAwaitingComponents {
				return phase, nil
			}

			err = r.patchDeploymentConfig(ctx, serverClient, deploymentConfig, service)
			if err != nil {
				return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to update the deployment config: %w", err)
			}
		}

		err = r.createEnvoyRateLimitingConfig(ctx, serverClient)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create envoy config: %w", err)
		}

		alertsReconciler := r.newEnvoyAlertReconciler()
		if phase, err := alertsReconciler.ReconcileAlerts(ctx, serverClient); err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			events.HandleError(r.recorder, installation, phase, "Failed to reconcile threescale alerts", err)
			return phase, err
		}
	}

	alertsReconciler := r.newAlertReconciler()
	if phase, err := alertsReconciler.ReconcileAlerts(ctx, serverClient); err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile threescale alerts", err)
		return phase, err
	}

	phase, err = r.reconcileConsoleLink(ctx, serverClient)
	logrus.Infof("Phase: %s reconcileConsoleLink", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile console link", err)
		return phase, err
	}

	phase, err = r.reconcileDeploymentConfigs(ctx, serverClient, productNamespace)
	logrus.Infof("Phase: %s reconcileDeploymentConfigs", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile deployment configs", err)
		return phase, err
	}

	// Ensure deployment configs are ready before returning phase complete
	phase, err = r.ensureDeploymentConfigsReady(ctx, serverClient, productNamespace)
	logrus.Infof("Phase: %s ensureDeploymentConfigsReady", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to ensure deployment configs are ready", err)
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

func (r *Reconciler) getDeploymentConfig(ctx context.Context, client k8sclient.Client, dcName string) (*appsv1.DeploymentConfig, integreatlyv1alpha1.StatusPhase, error) {
	apiCastDeploymentConfig := &appsv1.DeploymentConfig{}

	err := client.Get(ctx, k8sTypes.NamespacedName{Name: dcName, Namespace: r.Config.GetNamespace()}, apiCastDeploymentConfig)

	if err != nil {
		if k8serr.IsNotFound(err) {
			return nil, integreatlyv1alpha1.PhaseAwaitingComponents, nil
		}
		return nil, integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error getting the deployment config %v", err)
	}
	return apiCastDeploymentConfig, integreatlyv1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) getService(ctx context.Context, client k8sclient.Client, dcName string) (*corev1.Service, integreatlyv1alpha1.StatusPhase, error) {
	service := &corev1.Service{}

	err := client.Get(ctx, k8sTypes.NamespacedName{Name: dcName, Namespace: r.Config.GetNamespace()}, service)

	if err != nil {
		if k8serr.IsNotFound(err) {
			return nil, integreatlyv1alpha1.PhaseAwaitingComponents, nil
		}
		return nil, integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error getting the deployment config %v", err)
	}
	return service, integreatlyv1alpha1.PhaseInProgress, nil

}

// Patching the deployment configuration of both apicasts, this is required in order to enable rate limiting on the deployment. "Enabled" means that the Discovery Service
// will attach a sidecar container to the apicasts, annotations are set to label the apicasts so that appropriate envoy config(sidecar container used configuration) is
// pulled and attached to sidecar containers.
func (r *Reconciler) patchDeploymentConfig(ctx context.Context, client k8sclient.Client, deploymentConfig *appsv1.DeploymentConfig, service *corev1.Service) error {

	ports := service.Spec.Ports
	for i, port := range ports {
		if port.Name == "gateway" {
			service.Spec.Ports[i].Port = 8443
			service.Spec.Ports[i].TargetPort = intstr.FromInt(8443)
			break
		}
	}
	if err := client.Update(ctx, service); err != nil {
		return fmt.Errorf("failed to update service: %v", err)
	}

	deploymentConfig.Spec.Template.Labels["marin3r.3scale.net/status"] = "enabled"
	deploymentConfig.Spec.Template.Annotations["marin3r.3scale.net/node-id"] = apicastRatelimiting
	deploymentConfig.Spec.Template.Annotations["marin3r.3scale.net/ports"] = "envoy-https:8443"
	deploymentConfig.Spec.Template.Annotations["marin3r.3scale.net/envoy-image"] = "registry.redhat.io/openshift-service-mesh/proxyv2-rhel8:2.0"

	if err := client.Update(ctx, deploymentConfig); err != nil {
		return fmt.Errorf("failed to update deployment config: %v", err)
	}

	return nil
}

// This function creates envoy configuration that is used by sidecar containers. It contains information about clusters, which is information about where and how to find rate limiting
// pods, how to proxy the traffic and what traffic to listen for. Our data needs to be converted to JSON in order for the envoyapi omit empty to filter through fields that we are only interested in,
// then we need to convert it to yaml and finally push.
func (r *Reconciler) createEnvoyRateLimitingConfig(ctx context.Context, client k8sclient.Client) error {
	rateLimitService := &corev1.Service{}
	marin3rConfig, err := r.ConfigManager.ReadMarin3r()
	if err != nil {
		return fmt.Errorf("failed to load marin3r config in 3scale reconciler: %v", err)
	}
	err = client.Get(ctx, k8sclient.ObjectKey{
		Namespace: marin3rConfig.GetNamespace(),
		Name:      "ratelimit",
	}, rateLimitService)

	if err != nil {
		return fmt.Errorf("failed to rate limiting service: %v", err)
	}

	// Setting up cluster endpoints for rate limit and apicast
	apicastEndpoint := &envoycore.Address{Address: &envoycore.Address_SocketAddress{
		SocketAddress: &envoycore.SocketAddress{
			Address:  "127.0.0.1",
			Protocol: envoycore.SocketAddress_TCP,
			PortSpecifier: &envoycore.SocketAddress_PortValue{
				PortValue: uint32(8080),
			},
		},
	}}

	rateLimitEndpoint := &envoycore.Address{Address: &envoycore.Address_SocketAddress{
		SocketAddress: &envoycore.SocketAddress{
			Address:  rateLimitService.Spec.ClusterIP,
			Protocol: envoycore.SocketAddress_TCP,
			PortSpecifier: &envoycore.SocketAddress_PortValue{
				PortValue: uint32(8081),
			},
		},
	}}

	cluster := envoyapi.Cluster{
		Name:                 apicastRatelimiting,
		ConnectTimeout:       ptypes.DurationProto(2 * time.Second),
		ClusterDiscoveryType: &envoyapi.Cluster_Type{Type: envoyapi.Cluster_STRICT_DNS},
		LbPolicy:             envoyapi.Cluster_ROUND_ROBIN,
		LoadAssignment: &envoyapi.ClusterLoadAssignment{
			ClusterName: apicastRatelimiting,
			Endpoints: []*envoy_api_v2_endpoint.LocalityLbEndpoints{{
				LbEndpoints: []*envoy_api_v2_endpoint.LbEndpoint{
					{
						HostIdentifier: &envoy_api_v2_endpoint.LbEndpoint_Endpoint{
							Endpoint: &envoy_api_v2_endpoint.Endpoint{
								Address: apicastEndpoint,
							}},
					},
				},
			}},
		},
	}

	rateLimitCluster := envoyapi.Cluster{
		Name:                 "ratelimit",
		ConnectTimeout:       ptypes.DurationProto(2 * time.Second),
		ClusterDiscoveryType: &envoyapi.Cluster_Type{Type: envoyapi.Cluster_STRICT_DNS},
		LbPolicy:             envoyapi.Cluster_ROUND_ROBIN,
		Http2ProtocolOptions: &envoycore.Http2ProtocolOptions{},
		LoadAssignment: &envoyapi.ClusterLoadAssignment{
			ClusterName: "ratelimit",
			Endpoints: []*envoy_api_v2_endpoint.LocalityLbEndpoints{{
				LbEndpoints: []*envoy_api_v2_endpoint.LbEndpoint{
					{
						HostIdentifier: &envoy_api_v2_endpoint.LbEndpoint_Endpoint{
							Endpoint: &envoy_api_v2_endpoint.Endpoint{
								Address: rateLimitEndpoint,
							}},
					},
				},
			}},
		},
	}

	// Setting up envoyListener
	virtualHost := v2route.VirtualHost{
		Name:    apicastRatelimiting,
		Domains: []string{"*"},

		Routes: []*v2route.Route{
			{
				Match: &v2route.RouteMatch{
					PathSpecifier: &v2route.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &v2route.Route_Route{
					Route: &v2route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: apicastRatelimiting,
						},
						RateLimits: []*route.RateLimit{{
							Stage: &wrappers.UInt32Value{Value: 0},
							Actions: []*route.RateLimit_Action{{
								ActionSpecifier: &route.RateLimit_Action_GenericKey_{
									GenericKey: &route.RateLimit_Action_GenericKey{
										DescriptorValue: "slowpath",
									},
								},
							}},
						}},
					},
				},
			},
		},
	}

	// Setting GRPC
	clusterName := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"cluster_name": {
				Kind: &structpb.Value_StringValue{
					StringValue: "ratelimit",
				},
			},
		},
	}

	grpcService := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"timeout": {
				Kind: &structpb.Value_StringValue{
					StringValue: "2s",
				},
			},
			"envoy_grpc": {
				Kind: &structpb.Value_StructValue{
					StructValue: clusterName,
				},
			},
		},
	}
	grpcInnerService := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"grpc_service": {
				Kind: &structpb.Value_StructValue{
					StructValue: grpcService,
				},
			},
		},
	}
	httpFilterGrpc := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"domain": {
				Kind: &structpb.Value_StringValue{
					StringValue: apicastRatelimiting,
				},
			},
			"stage": {
				Kind: &structpb.Value_NumberValue{
					NumberValue: 0,
				},
			},
			"rate_limit_service": {
				Kind: &structpb.Value_StructValue{
					StructValue: grpcInnerService,
				},
			},
		},
	}

	// Setting up connection manager
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "ingress_http",
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: &envoyapi.RouteConfiguration{
				Name:         "local_route",
				VirtualHosts: []*v2route.VirtualHost{&virtualHost},
			},
		},
		HttpFilters: []*hcm.HttpFilter{
			{
				Name:       "envoy.rate_limit",
				ConfigType: &hcm.HttpFilter_Config{Config: httpFilterGrpc},
			},
			{
				Name: "envoy.router",
			},
		},
	}

	pbst, err := ptypes.MarshalAny(manager)
	if err != nil {
		return fmt.Errorf("failed to convert HttpConnectionManager for rate limiting: %v", err)
	}

	envoyListener := &envoyapi.Listener{
		Name: "http",
		Address: &envoycore.Address{
			Address: &envoycore.Address_SocketAddress{
				SocketAddress: &envoycore.SocketAddress{
					Protocol: envoycore.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &envoycore.SocketAddress_PortValue{
						PortValue: uint32(8443),
					},
				},
			},
		},
		FilterChains: []*listener.FilterChain{{
			Filters: []*listener.Filter{{
				Name:       "envoy.http_connection_manager",
				ConfigType: &listener.Filter_TypedConfig{TypedConfig: pbst},
			}},
		}},
	}

	// Converting to Json and then to Yaml before creating the CR
	rateLimitClusterJson, err := ResourcesToJSON(&rateLimitCluster)
	if err != nil {
		return fmt.Errorf("Failed to convert envoy rate limiting cluster configuration to JSON %v", err)
	}

	yamlRateLimitCluster, err := yaml.JSONToYAML(rateLimitClusterJson)
	if err != nil {
		return fmt.Errorf("Failed to convert envoy rate limiting cluster JSON configuration to YAML %v", err)
	}

	clusterJson, err := ResourcesToJSON(&cluster)
	if err != nil {
		return fmt.Errorf("Failed to convert envoy cluster configuration to JSON %v", err)
	}

	yamlCluster, err := yaml.JSONToYAML(clusterJson)
	if err != nil {
		return fmt.Errorf("Failed to convert envoy cluster JSON configuration to YAML %v", err)
	}

	listenerJson, err := ResourcesToJSON(envoyListener)
	if err != nil {
		return fmt.Errorf("Failed to convert envoy envoyListener configuration to JSON %v", err)
	}

	yamlListener, err := yaml.JSONToYAML(listenerJson)
	if err != nil {
		return fmt.Errorf("Failed to convert envoy envoyListener JSON configuration to YAML %v", err)
	}

	envoyconfig := &marin3rv1alpha.EnvoyConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apicastRatelimiting,
			Namespace: r.Config.GetNamespace(),
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, client, envoyconfig, func() error {
		owner.AddIntegreatlyOwnerAnnotations(envoyconfig, r.installation)
		envoyconfig.Spec.NodeID = apicastRatelimiting
		envoyconfig.Spec.Serialization = "yaml"
		envoyconfig.Spec.EnvoyResources = &marin3rv1alpha.EnvoyResources{
			Clusters: []marin3rv1alpha.EnvoyResource{
				{
					Name:  apicastRatelimiting,
					Value: string(yamlCluster),
				},
				{
					Name:  "ratelimit",
					Value: string(yamlRateLimitCluster),
				},
			},
			Listeners: []marin3rv1alpha.EnvoyResource{
				{
					Name:  "http",
					Value: string(yamlListener),
				},
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Failed to create envoy config CR %v", err)
	}

	return nil
}

func ResourcesToJSON(pb proto.Message) ([]byte, error) {
	m := jsonpb.Marshaler{}

	json := bytes.NewBuffer([]byte{})
	err := m.Marshal(json, pb)
	if err != nil {
		return []byte{}, err
	}
	return json.Bytes(), nil
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
			Monitoring:          &threescalev1.MonitoringSpec{},
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

	antiAffinityRequired, err := resources.IsAntiAffinityRequired(ctx, serverClient)
	if err != nil {
		r.logger.Errorf("error when deciding if pod anti affinity is required. Defaulted to false: %v", err)
		antiAffinityRequired = false
	}

	status, err := controllerutil.CreateOrUpdate(ctx, serverClient, apim, func() error {

		apim.Spec.HighAvailability = &threescalev1.HighAvailabilitySpec{Enabled: true}
		apim.Spec.APIManagerCommonSpec.ResourceRequirementsEnabled = &resourceRequirements
		apim.Spec.APIManagerCommonSpec.WildcardDomain = r.installation.Spec.RoutingSubdomain
		apim.Spec.System.FileStorageSpec = fss
		apim.Spec.PodDisruptionBudget = &threescalev1.PodDisruptionBudgetSpec{Enabled: true}
		apim.Spec.Monitoring = &threescalev1.MonitoringSpec{Enabled: false}

		replicas := r.Config.GetReplicasConfig(r.installation)

		if *apim.Spec.System.AppSpec.Replicas < replicas["systemApp"] {
			*apim.Spec.System.AppSpec.Replicas = replicas["systemApp"]
		}
		if *apim.Spec.System.SidekiqSpec.Replicas < replicas["systemSidekiq"] {
			*apim.Spec.System.SidekiqSpec.Replicas = replicas["systemSidekiq"]
		}
		if *apim.Spec.Apicast.ProductionSpec.Replicas < replicas["apicastProd"] {
			*apim.Spec.Apicast.ProductionSpec.Replicas = replicas["apicastProd"]
		}
		if *apim.Spec.Apicast.StagingSpec.Replicas < replicas["apicastStage"] {
			*apim.Spec.Apicast.StagingSpec.Replicas = replicas["apicastStage"]
		}
		if *apim.Spec.Backend.ListenerSpec.Replicas < replicas["backendListener"] {
			*apim.Spec.Backend.ListenerSpec.Replicas = replicas["backendListener"]
		}
		if *apim.Spec.Backend.WorkerSpec.Replicas < replicas["backendWorker"] {
			*apim.Spec.Backend.WorkerSpec.Replicas = replicas["backendWorker"]
		}
		if *apim.Spec.Backend.CronSpec.Replicas < replicas["backendCron"] {
			*apim.Spec.Backend.CronSpec.Replicas = replicas["backendCron"]
		}
		if *apim.Spec.Zync.AppSpec.Replicas < replicas["zyncApp"] {
			*apim.Spec.Zync.AppSpec.Replicas = replicas["zyncApp"]
		}
		if *apim.Spec.Zync.QueSpec.Replicas < replicas["zyncQue"] {
			*apim.Spec.Zync.QueSpec.Replicas = replicas["zyncQue"]
		}

		apim.Spec.System.AppSpec.Affinity = resources.SelectAntiAffinityForCluster(antiAffinityRequired, map[string]string{
			"threescale_component":         "system",
			"threescale_component_element": "app",
		})
		apim.Spec.System.SidekiqSpec.Affinity = resources.SelectAntiAffinityForCluster(antiAffinityRequired, map[string]string{
			"threescale_component":         "system",
			"threescale_component_element": "sidekiq",
		})
		apim.Spec.Apicast.ProductionSpec.Affinity = resources.SelectAntiAffinityForCluster(antiAffinityRequired, map[string]string{
			"threescale_component":         "apicast",
			"threescale_component_element": "production",
		})
		apim.Spec.Apicast.StagingSpec.Affinity = resources.SelectAntiAffinityForCluster(antiAffinityRequired, map[string]string{
			"threescale_component":         "apicast",
			"threescale_component_element": "staging",
		})

		apim.Spec.Backend.ListenerSpec.Affinity = resources.SelectAntiAffinityForCluster(antiAffinityRequired, map[string]string{
			"threescale_component":         "backend",
			"threescale_component_element": "listener",
		})
		apim.Spec.Backend.WorkerSpec.Affinity = resources.SelectAntiAffinityForCluster(antiAffinityRequired, map[string]string{
			"threescale_component":         "backend",
			"threescale_component_element": "worker",
		})
		apim.Spec.Backend.CronSpec.Affinity = resources.SelectAntiAffinityForCluster(antiAffinityRequired, map[string]string{
			"threescale_component":         "backend",
			"threescale_component_element": "cron",
		})
		apim.Spec.Zync.AppSpec.Affinity = resources.SelectAntiAffinityForCluster(antiAffinityRequired, map[string]string{
			"threescale_component":         "zync",
			"threescale_component_element": "zync",
		})
		apim.Spec.Zync.QueSpec.Affinity = resources.SelectAntiAffinityForCluster(antiAffinityRequired, map[string]string{
			"threescale_component":         "zync",
			"threescale_component_element": "zync-que",
		})

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
		} else if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
		// Its not enough to just check if the system-provider route exists. This can exist but system-master, for example, may not
		exist, err := r.routesExist(ctx, serverClient)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
		if exist {
			return integreatlyv1alpha1.PhaseCompleted, nil
		} else {
			// If the system-provider route does not exist at this point (i.e. when Deployments are ready)
			// we can force a resync of routes. see below for more details on why this is required:
			// https://access.redhat.com/documentation/en-us/red_hat_3scale_api_management/2.7/html/operating_3scale/backup-restore#creating_equivalent_zync_routes
			// This scenario will manifest during a backup and restore and also if the product ns was accidentially deleted.
			return r.resyncRoutes(ctx, serverClient)
		}
	}

	return integreatlyv1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) routesExist(ctx context.Context, serverClient k8sclient.Client) (bool, error) {
	expectedRoutes := 6
	opts := k8sclient.ListOptions{
		Namespace: r.Config.GetNamespace(),
	}

	routes := routev1.RouteList{}
	err := serverClient.List(ctx, &routes, &opts)
	if err != nil {
		return false, err
	}

	if len(routes.Items) >= expectedRoutes {
		return true, nil
	}
	return false, nil
}

func (r *Reconciler) resyncRoutes(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	ns := r.Config.GetNamespace()
	podname := ""

	pods := &corev1.PodList{}
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(ns),
		k8sclient.MatchingLabels(map[string]string{"deploymentConfig": "system-sidekiq"}),
	}
	err := client.List(ctx, pods, listOpts...)

	for _, pod := range pods.Items {
		if pod.Status.Phase == "Running" {
			podname = pod.ObjectMeta.Name
			break
		}
	}

	if podname == "" {
		logrus.Info("Waiting on system-sidekiq pod to start, 3Scale install in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	stdout, stderr, err := resources.ExecuteRemoteCommand(ns, podname, "bundle exec rake zync:resync:domains")
	if err != nil {
		logrus.Errorf("Failed to resync 3Scale routes %v", err)
		return integreatlyv1alpha1.PhaseFailed, nil
	} else if stderr != "" {
		logrus.Errorf("Error attempting to resync 3Scale routes %s", stderr)
		return integreatlyv1alpha1.PhaseFailed, nil
	} else {
		logrus.Infof("Resync 3Scale routes result: %s", stdout)
		return integreatlyv1alpha1.PhaseInProgress, nil
	}
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

	phase, err := resources.ReconcileRedisAlerts(ctx, serverClient, r.installation, backendRedis)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile redis alerts: %w", err)
	}
	if phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, nil
	}

	// create Redis Cpu Usage High alert
	err = resources.CreateRedisCpuUsageAlerts(ctx, serverClient, r.installation, backendRedis)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create backend redis prometheus Cpu usage high alerts for threescale: %s", err)
	}
	// wait for the backend redis cr to reconcile
	if backendRedis.Status.Phase != types.PhaseComplete {
		return integreatlyv1alpha1.PhaseAwaitingComponents, nil
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

	phase, err = resources.ReconcileRedisAlerts(ctx, serverClient, r.installation, systemRedis)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile redis alerts: %w", err)
	}
	if phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, nil
	}
	// wait for the system redis cr to reconcile
	if systemRedis.Status.Phase != types.PhaseComplete {
		return integreatlyv1alpha1.PhaseAwaitingComponents, nil
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

	// reconcile postgres alerts
	phase, err = resources.ReconcilePostgresAlerts(ctx, serverClient, r.installation, postgres)
	productName := postgres.Labels["productName"]
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile postgres alerts for %s: %w", productName, err)
	}
	if phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, nil
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
		logrus.Warnf("Cannot configure SSO integration without SSO ns: %v and SSO realm: %v", rhssoNamespace, rhssoRealm)
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	kcClient := &keycloak.KeycloakClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clientID,
			Namespace: rhssoNamespace,
		},
	}

	// keycloak-operator sets the spec.client.id, we need to preserve that value
	apiClientID := ""
	err = serverClient.Get(ctx, k8sclient.ObjectKey{
		Namespace: rhssoNamespace,
		Name:      clientID,
	}, kcClient)
	if err == nil {
		apiClientID = kcClient.Spec.Client.ID
	}

	clientSecret, err := r.getOauthClientSecret(ctx, serverClient)
	if err != nil {
		logrus.Errorf("Error retrieving client secret: %v", err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	opRes, err := controllerutil.CreateOrUpdate(ctx, serverClient, kcClient, func() error {
		kcClient.Spec = r.getKeycloakClientSpec(apiClientID, clientSecret)
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not create/update 3scale keycloak client: %w operation: %v", err, opRes)
	}

	accessToken, err := r.GetAdminToken(ctx, serverClient)
	if err != nil {
		logrus.Errorf("Failed to get admin token %v", err)
		return integreatlyv1alpha1.PhaseInProgress, err
	}
	_, err = r.tsClient.GetAuthenticationProviderByName(rhssoIntegrationName, *accessToken)
	if err != nil && !tsIsNotFoundError(err) {
		logrus.Errorf("Failed to get authentication provider: %v", err)
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
			logrus.Errorf("Failed to add authentication provider: %v", err)
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
		logrus.Errorf("Failed to retrieve admin name and password from secret: %v", err)
		return integreatlyv1alpha1.PhaseInProgress, err
	}

	kcu, err := rhsso.GetKeycloakUsers(ctx, serverClient, rhssoConfig.GetNamespace())
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	tsUsers, err := r.tsClient.GetUsers(*accessToken)
	if err != nil {
		logrus.Errorf("Failed to get users: %v", err)
		return integreatlyv1alpha1.PhaseInProgress, err
	}

	added, deleted := r.getUserDiff(kcu, tsUsers.Users)
	for _, kcUser := range added {
		res, err := r.tsClient.AddUser(strings.ToLower(kcUser.UserName), strings.ToLower(kcUser.Email), "", *accessToken)
		if err != nil || res.StatusCode != http.StatusCreated {
			logrus.Errorf("Failed to add user: %v", err)
			return integreatlyv1alpha1.PhaseInProgress, err
		}
	}
	for _, tsUser := range deleted {
		if tsUser.UserDetails.Username != *systemAdminUsername {
			res, err := r.tsClient.DeleteUser(tsUser.UserDetails.Id, *accessToken)
			if err != nil || res.StatusCode != http.StatusOK {
				logrus.Errorf("Failed to delete user: %v", err)
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
		logrus.Errorf("Failed to retrieve dedicated admins: %v", err)
		return integreatlyv1alpha1.PhaseInProgress, err
	}
	newTsUsers, err := r.tsClient.GetUsers(*accessToken)
	if err != nil {
		logrus.Errorf("Failed to get users: %v", err)
		return integreatlyv1alpha1.PhaseInProgress, err
	}

	isWorkshop := installation.Spec.Type == string(integreatlyv1alpha1.InstallationTypeWorkshop)

	err = syncOpenshiftAdminMembership(openshiftAdminGroup, newTsUsers, *systemAdminUsername, isWorkshop, r.tsClient, *accessToken)
	if err != nil {
		logrus.Errorf("Failed to sync openshift admin membership: %v", err)
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
		logrus.Errorf("Failed to get oauth client secret: %v", err)
		return integreatlyv1alpha1.PhaseInProgress, err
	}

	if status != controllerutil.OperationResultNone {
		err = r.RolloutDeployment("system-app")
		if err != nil {
			logrus.Errorf("Failed to rollout deployment (system-app): %v", err)
			return integreatlyv1alpha1.PhaseInProgress, err
		}

		err = r.RolloutDeployment("system-sidekiq")
		if err != nil {
			logrus.Errorf("Failed to rollout deployment (system-sidekiq): %v", err)
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
		logrus.Errorf("Error creating threescale blackbox target: %v", err)
		return integreatlyv1alpha1.PhaseInProgress, fmt.Errorf("error creating threescale blackbox target: %w", err)
	}

	// Create a blackbox target for the developer console ui
	threescaleRoute, err := r.getThreescaleRoute(ctx, client, "system-developer", func(r routev1.Route) bool {
		return strings.HasPrefix(r.Spec.Host, "3scale.")
	})
	if err != nil {
		logrus.Errorf("Error retrieving threescale threescaleRoute: %v", err)
		return integreatlyv1alpha1.PhaseInProgress, fmt.Errorf("error getting threescale system-developer threescaleRoute: %w", err)
	}
	err = monitoring.CreateBlackboxTarget(ctx, "integreatly-3scale-system-developer", monitoringv1alpha1.BlackboxtargetData{
		Url:     "https://" + threescaleRoute.Spec.Host,
		Service: "3scale-developer-console-ui",
	}, cfg, installation, client)
	if err != nil {
		logrus.Errorf("Error creating blackbox target (system-developer): %v", err)
		return integreatlyv1alpha1.PhaseInProgress, fmt.Errorf("error creating threescale blackbox target (system-developer): %w", err)
	}

	// Create a blackbox target for the master console ui
	threescaleRoute, err = r.getThreescaleRoute(ctx, client, "system-master", nil)
	if err != nil {
		return integreatlyv1alpha1.PhaseInProgress, fmt.Errorf("error getting threescale system-master threescaleRoute: %w", err)
	}
	err = monitoring.CreateBlackboxTarget(ctx, "integreatly-3scale-system-master", monitoringv1alpha1.BlackboxtargetData{
		Url:     "https://" + threescaleRoute.Spec.Host,
		Service: "3scale-system-admin-ui",
	}, cfg, installation, client)
	if err != nil {
		logrus.Errorf("Error creating blackbox target (system-master): %v", err)
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
	for _, rt := range routes.Items {
		if filterFn(rt) {
			foundRoute = &rt
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
		if strings.EqualFold(kcu.UserName, tsUser.UserDetails.Username) {
			return true
		}
	}

	return false
}

func tsContainsKc(tsusers []*User, kcUser keycloak.KeycloakAPIUser) bool {
	for _, tsu := range tsusers {
		if strings.EqualFold(tsu.UserDetails.Username, kcUser.UserName) {
			return true
		}
	}

	return false
}

func userIsOpenshiftAdmin(tsUser *User, adminGroup *usersv1.Group) bool {
	for _, userName := range adminGroup.Users {
		if strings.EqualFold(tsUser.UserDetails.Username, userName) {
			return true
		}
	}

	return false
}

func (r *Reconciler) getKeycloakClientSpec(id, clientSecret string) keycloak.KeycloakClientSpec {
	return keycloak.KeycloakClientSpec{
		RealmSelector: &metav1.LabelSelector{
			MatchLabels: rhsso.GetInstanceLabels(),
		},
		Client: &keycloak.KeycloakAPIClient{
			ID:                      id,
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

func (r *Reconciler) reconcileSubscription(ctx context.Context, serverClient k8sclient.Client, inst *integreatlyv1alpha1.RHMI, productNamespace string, operatorNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	target := marketplace.Target{
		Pkg:       constants.ThreeScaleSubscriptionName,
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
	)
}

func (r *Reconciler) reconcileConsoleLink(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	cl := &consolev1.ConsoleLink{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rhmi-3scale-console-link",
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, cl, func() error {
		cl.Spec = consolev1.ConsoleLinkSpec{
			ApplicationMenu: &consolev1.ApplicationMenuSpec{
				ImageURL: threeScaleIcon,
				Section:  "OpenShift Managed Services",
			},
			Link: consolev1.Link{
				Href: fmt.Sprintf("%v/auth/rhsso/bounce", r.Config.GetHost()),
				Text: "API Management",
			},
			Location: consolev1.ApplicationMenu,
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating or updating 3Scale console link, %s", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) deleteConsoleLink(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	cl := &consolev1.ConsoleLink{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rhmi-3scale-console-link",
		},
	}

	err := serverClient.Delete(ctx, cl)
	if err != nil && !k8serr.IsNotFound(err) {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error removing 3Scale console link, %s", err)
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileDeploymentConfigs(ctx context.Context, serverClient k8sclient.Client, productNamespace string) (integreatlyv1alpha1.StatusPhase, error) {

	for _, name := range threeScaleDeploymentConfigs {
		deploymentConfig := &appsv1.DeploymentConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: productNamespace,
			},
		}

		podPriorityMutation := resources.NoopMutate
		if r.installation.Spec.Type == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
			podPriorityMutation = resources.MutatePodPriority(r.installation.Spec.PriorityClassName)
		}

		phase, err := resources.UpdatePodTemplateIfExists(
			ctx,
			serverClient,
			resources.SelectFromDeploymentConfig,
			resources.AllMutationsOf(
				resources.MutateZoneTopologySpreadConstraints("app"),
				podPriorityMutation,
			),
			deploymentConfig,
		)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

// Deployment configs are rescaled when adding topologySpreadConstraints, PodTopology etc
// Should check that these deployment configs are ready before returning phase complete in CR
func (r *Reconciler) ensureDeploymentConfigsReady(ctx context.Context, serverClient k8sclient.Client, productNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	for _, name := range threeScaleDeploymentConfigs {
		deploymentConfig := &appsv1.DeploymentConfig{}

		err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: name, Namespace: productNamespace}, deploymentConfig)

		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		for _, condition := range deploymentConfig.Status.Conditions {
			if condition.Status != corev1.ConditionTrue || (deploymentConfig.Status.Replicas != deploymentConfig.Status.AvailableReplicas ||
				deploymentConfig.Status.ReadyReplicas != deploymentConfig.Status.UpdatedReplicas) {
				logrus.Infof("waiting for 3scale deployment config %s to become ready", name)
				return integreatlyv1alpha1.PhaseInProgress, fmt.Errorf("waiting for 3scale deployment config %s to become available", name)
			}
		}
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
