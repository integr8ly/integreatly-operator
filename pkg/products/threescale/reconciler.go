package threescale

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	configv1 "github.com/openshift/api/config/v1"

	portaClient "github.com/3scale/3scale-porta-go-client/client"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/integr8ly/integreatly-operator/pkg/addon"
	"github.com/integr8ly/integreatly-operator/pkg/resources/k8s"
	"github.com/integr8ly/integreatly-operator/pkg/resources/sts"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"

	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	customDomain "github.com/integr8ly/integreatly-operator/pkg/resources/custom-domain"
	cs "github.com/integr8ly/integreatly-operator/pkg/resources/custom-smtp"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"github.com/integr8ly/integreatly-operator/version"
	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/integr8ly/integreatly-operator/pkg/metrics"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	envoyclusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoylistenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	consolev1 "github.com/openshift/api/console/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"

	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/ratelimit"

	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	crov1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/client"
	userHelper "github.com/integr8ly/integreatly-operator/pkg/resources/user"

	apps "github.com/3scale/3scale-operator/apis/apps"
	threescalev1 "github.com/3scale/3scale-operator/apis/apps/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	keycloak "github.com/integr8ly/keycloak-client/apis/keycloak/v1alpha1"

	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/products/rhsso"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	appsv1Client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	oauthClient "github.com/openshift/client-go/oauth/clientset/versioned/typed/oauth/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	apiMachineryTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "3scale"
	apiManagerName               = "3scale"
	clientID                     = "3scale"
	rhssoIntegrationName         = "rhsso"
	apicastHTTPsPort             = int32(8444)

	s3CredentialsSecretName         = "s3-credentials"
	externalRedisSecretName         = "system-redis"
	externalBackendRedisSecretName  = "backend-redis"
	externalPostgresSecretName      = "system-database"
	apicastStagingDeploymentName    = "apicast-staging"
	apicastProductionDeploymentName = "apicast-production"
	backendListenerDeploymentName   = "backend-listener"
	systemSeedSecretName            = "system-seed"
	systemMasterApiCastSecretName   = "system-master-apicast"
	systemAppDeploymentName         = "system-app"
	multitenantID                   = "rhoam-mt"
	registrySecretName              = "threescale-registry-auth"
	threeScaleIcon                  = "data:image/svg+xml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0idXRmLTgiPz4KPCEtLSBHZW5lcmF0b3I6IEFkb2JlIElsbHVzdHJhdG9yIDI1LjIuMCwgU1ZHIEV4cG9ydCBQbHVnLUluIC4gU1ZHIFZlcnNpb246IDYuMDAgQnVpbGQgMCkgIC0tPgo8c3ZnIHZlcnNpb249IjEuMSIgaWQ9IkxheWVyXzEiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgeG1sbnM6eGxpbms9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkveGxpbmsiIHg9IjBweCIgeT0iMHB4IgoJIHZpZXdCb3g9IjAgMCAzNyAzNyIgc3R5bGU9ImVuYWJsZS1iYWNrZ3JvdW5kOm5ldyAwIDAgMzcgMzc7IiB4bWw6c3BhY2U9InByZXNlcnZlIj4KPHN0eWxlIHR5cGU9InRleHQvY3NzIj4KCS5zdDB7ZmlsbDojRUUwMDAwO30KCS5zdDF7ZmlsbDojRkZGRkZGO30KPC9zdHlsZT4KPGc+Cgk8cGF0aCBkPSJNMjcuNSwwLjVoLTE4Yy00Ljk3LDAtOSw0LjAzLTksOXYxOGMwLDQuOTcsNC4wMyw5LDksOWgxOGM0Ljk3LDAsOS00LjAzLDktOXYtMThDMzYuNSw0LjUzLDMyLjQ3LDAuNSwyNy41LDAuNUwyNy41LDAuNXoiCgkJLz4KCTxnPgoJCTxwYXRoIGNsYXNzPSJzdDAiIGQ9Ik0yNSwyMi4zN2MtMC45NSwwLTEuNzUsMC42My0yLjAyLDEuNWgtMS44NVYyMS41YzAtMC4zNS0wLjI4LTAuNjItMC42Mi0wLjYycy0wLjYyLDAuMjgtMC42MiwwLjYydjMKCQkJYzAsMC4zNSwwLjI4LDAuNjIsMC42MiwwLjYyaDIuNDhjMC4yNywwLjg3LDEuMDcsMS41LDIuMDIsMS41YzEuMTcsMCwyLjEyLTAuOTUsMi4xMi0yLjEyUzI2LjE3LDIyLjM3LDI1LDIyLjM3eiBNMjUsMjUuMzcKCQkJYy0wLjQ4LDAtMC44OC0wLjM5LTAuODgtMC44OHMwLjM5LTAuODgsMC44OC0wLjg4czAuODgsMC4zOSwwLjg4LDAuODhTMjUuNDgsMjUuMzcsMjUsMjUuMzd6Ii8+CgkJPHBhdGggY2xhc3M9InN0MCIgZD0iTTIwLjUsMTYuMTJjMC4zNCwwLDAuNjItMC4yOCwwLjYyLTAuNjJ2LTIuMzhoMS45MWMwLjMyLDAuNzcsMS4wOCwxLjMxLDEuOTYsMS4zMQoJCQljMS4xNywwLDIuMTItMC45NSwyLjEyLTIuMTJzLTAuOTUtMi4xMi0yLjEyLTIuMTJjLTEuMDIsMC0xLjg4LDAuNzMtMi4wOCwxLjY5SDIwLjVjLTAuMzQsMC0wLjYyLDAuMjgtMC42MiwwLjYydjMKCQkJQzE5Ljg3LDE1Ljg1LDIwLjE2LDE2LjEyLDIwLjUsMTYuMTJ6IE0yNSwxMS40M2MwLjQ4LDAsMC44OCwwLjM5LDAuODgsMC44OHMtMC4zOSwwLjg4LTAuODgsMC44OHMtMC44OC0wLjM5LTAuODgtMC44OAoJCQlTMjQuNTIsMTEuNDMsMjUsMTEuNDN6Ii8+CgkJPHBhdGggY2xhc3M9InN0MCIgZD0iTTEyLjEyLDE5Ljk2di0wLjg0aDIuMzhjMC4zNCwwLDAuNjItMC4yOCwwLjYyLTAuNjJzLTAuMjgtMC42Mi0wLjYyLTAuNjJoLTIuMzh2LTAuOTEKCQkJYzAtMC4zNS0wLjI4LTAuNjItMC42Mi0wLjYyaC0zYy0wLjM0LDAtMC42MiwwLjI4LTAuNjIsMC42MnYzYzAsMC4zNSwwLjI4LDAuNjIsMC42MiwwLjYyaDNDMTEuODQsMjAuNTksMTIuMTIsMjAuMzEsMTIuMTIsMTkuOTYKCQkJeiBNMTAuODcsMTkuMzRIOS4xMnYtMS43NWgxLjc1VjE5LjM0eiIvPgoJCTxwYXRoIGNsYXNzPSJzdDAiIGQ9Ik0yOC41LDE2LjM0aC0zYy0wLjM0LDAtMC42MiwwLjI4LTAuNjIsMC42MnYwLjkxSDIyLjVjLTAuMzQsMC0wLjYyLDAuMjgtMC42MiwwLjYyczAuMjgsMC42MiwwLjYyLDAuNjJoMi4zOAoJCQl2MC44NGMwLDAuMzUsMC4yOCwwLjYyLDAuNjIsMC42MmgzYzAuMzQsMCwwLjYyLTAuMjgsMC42Mi0wLjYydi0zQzI5LjEyLDE2LjYyLDI4Ljg0LDE2LjM0LDI4LjUsMTYuMzR6IE0yNy44NywxOS4zNGgtMS43NXYtMS43NQoJCQloMS43NVYxOS4zNHoiLz4KCQk8cGF0aCBjbGFzcz0ic3QwIiBkPSJNMTYuNSwyMC44N2MtMC4zNCwwLTAuNjMsMC4yOC0wLjYzLDAuNjJ2Mi4zOGgtMS44NWMtMC4yNy0wLjg3LTEuMDctMS41LTIuMDItMS41CgkJCWMtMS4xNywwLTIuMTIsMC45NS0yLjEyLDIuMTJzMC45NSwyLjEyLDIuMTIsMi4xMmMwLjk1LDAsMS43NS0wLjYzLDIuMDItMS41aDIuNDhjMC4zNCwwLDAuNjItMC4yOCwwLjYyLTAuNjJ2LTMKCQkJQzE3LjEyLDIxLjE1LDE2Ljg0LDIwLjg3LDE2LjUsMjAuODd6IE0xMiwyNS4zN2MtMC40OCwwLTAuODgtMC4zOS0wLjg4LTAuODhzMC4zOS0wLjg4LDAuODgtMC44OHMwLjg4LDAuMzksMC44OCwwLjg4CgkJCVMxMi40OCwyNS4zNywxMiwyNS4zN3oiLz4KCQk8cGF0aCBjbGFzcz0ic3QwIiBkPSJNMTYuNSwxMS44N2gtMi40MmMtMC4yLTAuOTctMS4wNi0xLjY5LTIuMDgtMS42OWMtMS4xNywwLTIuMTIsMC45NS0yLjEyLDIuMTJzMC45NSwyLjEyLDIuMTIsMi4xMgoJCQljMC44OCwwLDEuNjQtMC41NCwxLjk2LTEuMzFoMS45MXYyLjM4YzAsMC4zNSwwLjI4LDAuNjIsMC42MywwLjYyczAuNjItMC4yOCwwLjYyLTAuNjJ2LTNDMTcuMTIsMTIuMTUsMTYuODQsMTEuODcsMTYuNSwxMS44N3oKCQkJIE0xMiwxMy4xOGMtMC40OCwwLTAuODgtMC4zOS0wLjg4LTAuODhzMC4zOS0wLjg4LDAuODgtMC44OHMwLjg4LDAuMzksMC44OCwwLjg4UzEyLjQ4LDEzLjE4LDEyLDEzLjE4eiIvPgoJPC9nPgoJPHBhdGggY2xhc3M9InN0MSIgZD0iTTE4LjUsMjIuNjJjLTIuMjcsMC00LjEzLTEuODUtNC4xMy00LjEyczEuODUtNC4xMiw0LjEzLTQuMTJzNC4xMiwxLjg1LDQuMTIsNC4xMlMyMC43NywyMi42MiwxOC41LDIyLjYyegoJCSBNMTguNSwxNS42MmMtMS41OCwwLTIuODgsMS4yOS0yLjg4LDIuODhzMS4yOSwyLjg4LDIuODgsMi44OHMyLjg4LTEuMjksMi44OC0yLjg4UzIwLjA4LDE1LjYyLDE4LjUsMTUuNjJ6Ii8+CjwvZz4KPC9zdmc+Cg=="
	user3ScaleID                    = "3scale_user_id"

	labelRouteToSystemMaster    = "system-master"
	labelRouteToSystemDeveloper = "system-developer"
	labelRouteToSystemProvider  = "system-provider"

	// STS
	stsS3CredentialsSecretName  = "sts-s3-credentials"                              // #nosec G101 -- This is a false positive
	stsWebIdentityTokenFilePath = "/var/run/secrets/openshift/serviceaccount/token" // #nosec G101 -- This is a false positive
	stsTokenAudience            = "openshift"
)

var (
	threeScaleDeployments = []string{
		"apicast-production",
		"apicast-staging",
		"backend-cron",
		"backend-listener",
		"backend-worker",
		"system-app",
		"system-memcache",
		"system-sidekiq",
		"system-searchd",
		"zync",
		"zync-database",
		"zync-que",
	}
)

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, appsv1Client appsv1Client.AppsV1Interface, oauthv1Client oauthClient.OauthV1Interface, tsClient ThreeScaleInterface, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, logger l.Logger, productDeclaration *marketplace.ProductDeclaration) (*Reconciler, error) {
	if productDeclaration == nil {
		return nil, fmt.Errorf("no product declaration found for 3scale")
	}

	ns := installation.Spec.NamespacePrefix + defaultInstallationNamespace
	threescaleConfig, err := configManager.ReadThreeScale()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve threescale config: %w", err)
	}

	threescaleConfig.SetNamespace(ns)
	if installation.Spec.OperatorsInProductNamespace {
		threescaleConfig.SetOperatorNamespace(threescaleConfig.GetNamespace())
	} else {
		threescaleConfig.SetOperatorNamespace(threescaleConfig.GetNamespace() + "-operator")
	}
	threescaleConfig.SetBlackboxTargetPathForAdminUI("/p/login/")

	if err := configManager.WriteConfig(threescaleConfig); err != nil {
		return nil, fmt.Errorf("error writing threescale config : %w", err)
	}

	return &Reconciler{
		ConfigManager: configManager,
		Config:        threescaleConfig,
		mpm:           mpm,
		installation:  installation,
		tsClient:      tsClient,
		oauthv1Client: oauthv1Client,
		Reconciler:    resources.NewReconciler(mpm).WithProductDeclaration(*productDeclaration),
		recorder:      recorder,
		log:           logger,
		podExecutor:   resources.NewPodExecutor(logger),
	}, nil
}

type Reconciler struct {
	ConfigManager config.ConfigReadWriter
	Config        *config.ThreeScale
	mpm           marketplace.MarketplaceInterface
	installation  *integreatlyv1alpha1.RHMI
	tsClient      ThreeScaleInterface
	oauthv1Client oauthClient.OauthV1Interface
	*resources.Reconciler
	extraParams map[string]string
	recorder    record.EventRecorder
	log         l.Logger
	podExecutor resources.PodExecutorInterface
}

func (r *Reconciler) GetPreflightObject(ns string) k8sclient.Object {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "system-app",
			Namespace: ns,
		},
	}
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return version.VerifyProductAndOperatorVersion(
		installation.Status.Stages[integreatlyv1alpha1.InstallStage].Products[integreatlyv1alpha1.Product3Scale],
		string(integreatlyv1alpha1.Version3Scale),
		string(integreatlyv1alpha1.OperatorVersion3Scale),
	)
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, productStatus *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client, productConfig quota.ProductConfig, uninstall bool) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Start Reconciling")
	operatorNamespace := r.Config.GetOperatorNamespace()
	productNamespace := r.Config.GetNamespace()
	customDomainActive := customDomain.IsCustomDomain(installation)
	platformType := configv1.AWSPlatformType

	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), uninstall, func() (integreatlyv1alpha1.StatusPhase, error) {
		phase, err := ratelimit.DeleteEnvoyConfigsInNamespaces(ctx, serverClient, productNamespace)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}
		phase, err = k8s.EnsureObjectDeleted(ctx, serverClient, &threescalev1.APIManager{
			ObjectMeta: metav1.ObjectMeta{
				Name:      apiManagerName,
				Namespace: productNamespace,
			},
		})
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}
		phase, err = k8s.EnsureObjectDeleted(ctx, serverClient, &operatorsv1alpha1.Subscription{
			ObjectMeta: metav1.ObjectMeta{
				Name:      constants.ThreeScaleSubscriptionName,
				Namespace: operatorNamespace,
			},
		})
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}
		phase, err = k8s.EnsureObjectDeleted(ctx, serverClient, &operatorsv1alpha1.ClusterServiceVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("3scale-operator.v%s", integreatlyv1alpha1.OperatorVersion3Scale),
				Namespace: operatorNamespace,
			},
		})
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}
		isHiveManaged, err := addon.OperatorIsHiveManaged(ctx, serverClient, installation)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
		if !isHiveManaged {
			phase, err = resources.RemoveNamespace(ctx, installation, serverClient, productNamespace, r.log)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
		}
		phase, err = resources.RemoveNamespace(ctx, installation, serverClient, operatorNamespace, r.log)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		err = resources.RemoveOauthClient(r.oauthv1Client, r.getOAuthClientName(), r.log)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		phase, err = r.deleteConsoleLink(ctx, serverClient)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		return integreatlyv1alpha1.PhaseCompleted, nil
	}, r.log)
	if err != nil || phase == integreatlyv1alpha1.PhaseFailed {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	if uninstall {
		return phase, nil
	}

	alertsReconciler, err := r.newAlertReconciler(r.log, r.installation.Spec.Type, ctx, serverClient, config.GetOboNamespace(r.installation.Namespace))
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if phase, err = alertsReconciler.ReconcileAlerts(ctx, serverClient); err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile threescale alerts", err)
		return phase, err
	}

	if customDomainActive {

		customDomainPhase, customDomainName, customDomainErr := r.findCustomDomainCr(ctx, serverClient)

		ingressControllerPhase, ingressControllerErr := r.findIngressControllerCr(ctx, customDomainName, serverClient)

		// If both custom domain and ingress controller have errors, prioritize ingress controller error
		if customDomainErr != nil && ingressControllerErr != nil {
			errorMessage := "Both CustomDomain and IngressController CRs failed to be found or are in unexpected state"
			r.log.Error("msg", nil, errors.New(errorMessage))
			events.HandleError(r.recorder, installation, ingressControllerPhase, errorMessage, ingressControllerErr)
			customDomain.UpdateErrorAndCustomDomainMetric(r.installation, customDomainActive, ingressControllerErr)
			return ingressControllerPhase, ingressControllerErr
		}

		// If both are found but neither are in the completed phase, report an error
		if customDomainPhase != integreatlyv1alpha1.PhaseCompleted && ingressControllerPhase != integreatlyv1alpha1.PhaseCompleted {
			errorMessage := "CustomDomain or IngressController CR is not in a completed phase"
			//nolint:staticcheck // SA1006: Error is not printf-style, so this is fine
			err := fmt.Errorf("customDomain Phase and ingressController Phase failed : %s", errorMessage)
			r.log.Error("msg", nil, err)
			events.HandleError(r.recorder, installation, phase, errorMessage, err)
			customDomain.UpdateErrorAndCustomDomainMetric(r.installation, customDomainActive, err)
			return phase, err
		}

		// If no errors occurred, proceed with updating the metrics and let the reconciler continue
		customDomain.UpdateErrorAndCustomDomainMetric(r.installation, customDomainActive, nil)
	}

	phase, err = r.ReconcileNamespace(ctx, operatorNamespace, installation, serverClient, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s ns", operatorNamespace), err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, productNamespace, installation, serverClient, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s ns", productNamespace), err)
		return phase, err
	}

	phase, err = resources.ReconcileLimitRange(ctx, serverClient, operatorNamespace, resources.DefaultLimitRangeParams)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile LimitRange for Namespace %s", operatorNamespace), err)
		return phase, err
	}

	phase, err = resources.ReconcileLimitRange(ctx, serverClient, productNamespace, resources.DefaultLimitRangeParams)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile LimitRange for Namespace %s", productNamespace), err)
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

	phase, err = r.ReconcileCsvDeploymentsPriority(
		ctx,
		serverClient,
		fmt.Sprintf("3scale-operator.v%s", integreatlyv1alpha1.OperatorVersion3Scale),
		r.Config.GetOperatorNamespace(),
		r.installation.Spec.PriorityClassName,
	)
	if err != nil || phase == integreatlyv1alpha1.PhaseFailed {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile 3scale-operator csv deployments priority class name", err)
		return phase, err
	}

	if r.installation.GetDeletionTimestamp() == nil {
		phase, err = r.reconcileSMTPCredentials(ctx, serverClient)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			events.HandleError(r.recorder, installation, phase, "Failed to reconcile smtp credentials", err)
			return phase, err
		}

		// Wait for RHSSO postgres to be completed
		phase, err = resources.WaitForRHSSOPostgresToBeComplete(serverClient, installation.Name, r.ConfigManager.GetOperatorNamespace())
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			events.HandleError(r.recorder, installation, phase, "Waiting for RHSSO postgres to be completed", err)
			return phase, err
		}

		phase, err = r.reconcileExternalDatasources(ctx, serverClient, productConfig.GetActiveQuota(), platformType)
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

	phase, err = r.reconcileComponents(ctx, serverClient, productConfig, platformType)
	r.log.Infof("reconcileComponents", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile components", err)
		return phase, err
	}

	phase, err = r.ping3scalePortals(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		errorMessage := "failed pinging 3scale portals through the ingress cluster router"
		r.log.Error("msg", nil, errors.New(errorMessage))
		events.HandleError(r.recorder, installation, phase, errorMessage, err)
		return phase, err
	}

	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installation.Spec.Type)) {
		phase, err = r.reconcile3scaleMultiTenancy(ctx, serverClient)
		if err != nil {
			r.log.Error("reconcile3scaleMultiTenancy", nil, err)
			return phase, err
		}
	}

	r.log.Info("Successfully deployed")

	phase, err = r.reconcileOutgoingEmailAddress(ctx, serverClient)
	r.log.Infof("reconcileOutgoingEmailAddress", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		if err != nil {
			r.log.Warning("Failed to reconcileOutgoingEmailAddress: " + err.Error())
			events.HandleError(r.recorder, installation, phase, "Failed to reconcileOutgoingEmailAddress", err)
		}
		return phase, err
	}

	phase, err = r.reconcileDeploymentEnvarEmailAddress(ctx, serverClient, systemAppDeploymentName, updateSystemAppAddresses)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		if err != nil {
			r.log.Warning("Failed to reconcileDeploymentEnvarEmailAddress: " + err.Error())
			events.HandleError(r.recorder, installation, phase, "Failed to reconcileDeploymentEnvarEmailAddress", err)
		}
		return phase, err
	}

	phase, err = r.reconcileDeploymentEnvarEmailAddress(ctx, serverClient, "system-sidekiq", updateSystemSidekiqAddresses)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		if err != nil {
			r.log.Warning("Failed to reconcileDeploymentEnvarEmailAddress: " + err.Error())
			events.HandleError(r.recorder, installation, phase, "Failed to reconcileDeploymentEnvarEmailAddress", err)
		}
		return phase, err
	}

	phase, err = r.reconcileRHSSOIntegration(ctx, serverClient)
	r.log.Infof("reconcileRHSSOIntegration", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile rhsso integration", err)
		return phase, err
	}

	if !integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installation.Spec.Type)) {
		phase, err = r.reconcileOpenshiftUsers(ctx, installation, serverClient)
		r.log.Infof("reconcileOpenshiftUsers", l.Fields{"phase": phase})
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			events.HandleError(r.recorder, installation, phase, "Failed to reconcile openshift users", err)
			return phase, err
		}
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
	r.log.Infof("ReconcileOauthClient", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile oauth client", err)
		return phase, err
	}

	phase, err = r.reconcileServiceDiscovery(ctx, serverClient)
	r.log.Infof("reconcileServiceDiscovery", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile service discovery", err)
		return phase, err
	}

	phase, err = r.backupSystemSecrets(ctx, serverClient, installation)
	r.log.Infof("backupSystemSecrets", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile templates", err)
		return phase, err
	}

	phase, err = r.reconcileRouteEditRole(ctx, serverClient)
	r.log.Infof("reconcileRouteEditRole", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile roles", err)
		return phase, err
	}

	// Ensure ratelimit annotation is ready before returning phase complete
	phase, err = r.reconcileRatelimitPortAnnotation(ctx, serverClient)
	r.log.Infof("reconcileRatelimitPortAnnotation", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile ratelimit service port annotation", err)
		return phase, err
	}

	phase, err = r.reconcileRatelimitingTo3scaleComponents(ctx, serverClient, r.installation)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile rate limiting to 3scale components", err)
		return phase, err
	}

	alertsReconciler = r.newEnvoyAlertReconciler(r.log, r.installation.Spec.Type, config.GetOboNamespace(installation.Namespace))
	if phase, err := alertsReconciler.ReconcileAlerts(ctx, serverClient); err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile threescale alerts", err)
		return phase, err
	}

	phase, err = r.reconcileServiceMonitor(ctx, serverClient, productNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile 3scale service monitor", err)
		return phase, err
	}

	if !integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installation.Spec.Type)) {
		phase, err = r.reconcileConsoleLink(ctx, serverClient)
		r.log.Infof("reconcileConsoleLink", l.Fields{"phase": phase})
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			events.HandleError(r.recorder, installation, phase, "Failed to reconcile console link", err)
			return phase, err
		}
	}

	phase, err = r.syncInvitationEmail(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		if err != nil {
			r.log.Warning("Failed to syncInvitationEmail: " + err.Error())
			events.HandleError(r.recorder, installation, phase, "Failed to syncInvitationEmail", err)
		}
		return phase, err
	}

	// Ensure deployments are ready before returning phase complete
	phase, err = r.ensureDeploymentsReady(ctx, serverClient, productNamespace)
	r.log.Infof("ensureDeploymentsReady", l.Fields{"phase": phase})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to ensure deployments are ready", err)
		return phase, err
	}

	productStatus.Host = r.Config.GetHost()
	productStatus.Version = r.Config.GetProductVersion()
	productStatus.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	r.log.Infof("Installation reconciled successfully", l.Fields{"productStatus": r.Config.GetProductName()})
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
			r.log.Info(fmt.Sprintf("no backed up secret %v found in %v", secretName, installation.Namespace))
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

func (r *Reconciler) getOauthClientSecret(ctx context.Context, serverClient k8sclient.Client) (string, error) {
	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.ConfigManager.GetOauthClientsSecretName(),
		},
	}

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: oauthClientSecrets.Name, Namespace: r.ConfigManager.GetOperatorNamespace()}, oauthClientSecrets)
	if err != nil {
		return "", fmt.Errorf("could not find %s Secret: %w", oauthClientSecrets.Name, err)
	}

	clientSecretBytes, ok := oauthClientSecrets.Data[string(r.Config.GetProductName())]
	if !ok {
		return "", fmt.Errorf("could not find %s key in %s Secret", string(r.Config.GetProductName()), oauthClientSecrets.Name)
	}
	return string(clientSecretBytes), nil
}

func (r *Reconciler) reconcileSMTPCredentials(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Reconciling smtp credentials")

	// get the secret containing smtp credentials
	credSec := &corev1.Secret{}
	secretName := r.installation.Spec.SMTPSecret

	if r.installation.Status.CustomSmtp != nil && r.installation.Status.CustomSmtp.Enabled {
		r.log.Info("configuring user smtp for 3scale notifications")
		secretName = cs.CustomSecret
	}

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: secretName, Namespace: r.installation.Namespace}, credSec)
	if err != nil {
		r.log.Warningf("could not obtain smtp credentials secret", l.Fields{"error": err})
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

		// There is an issue with setting smtp values and creating Tenants. CreateTenant fails when SMTP values are set.
		if !integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(r.installation.Spec.Type)) {

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
				err = r.RolloutDeployment(ctx, serverClient, "system-app")
				if err != nil {
					r.log.Error("Rollout system-app deployment", nil, err)
				}

				err = r.RolloutDeployment(ctx, serverClient, "system-sidekiq")
				if err != nil {
					r.log.Error("Rollout system-sidekiq deployment", nil, err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create or update 3scale smtp configmap: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, serverClient k8sclient.Client, productConfig quota.ProductConfig, platformType configv1.PlatformType) (integreatlyv1alpha1.StatusPhase, error) {
	fss, err := r.getBlobStorageFileStorageSpec(ctx, serverClient, platformType)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	apim := &threescalev1.APIManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiManagerName,
			Namespace: r.Config.GetNamespace(),
		},
	}

	key := k8sclient.ObjectKeyFromObject(apim)
	err = serverClient.Get(ctx, key, apim)
	if err != nil && !k8serr.IsNotFound(err) {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	antiAffinityRequired, err := resources.IsAntiAffinityRequired(ctx, serverClient)
	if err != nil {
		r.log.Warning("Failure when deciding if pod anti affinity is required. Defaulted to false: " + err.Error())
		antiAffinityRequired = false
	}

	ExternalComponentsTrue := true
	resourceRequirements := true
	replicas := r.Config.GetReplicasConfig(r.installation)
	systemAppReplicas := replicas["systemApp"]
	systemSidekiqReplicas := replicas["systemSidekiq"]
	apicastStageReplicas := replicas["apicastStage"]
	backendCronReplicas := replicas["backendCron"]
	zyncReplicas := replicas["zyncApp"]
	zyncQueReplicas := replicas["zyncQue"]
	apicastport := apicastHTTPsPort

	status, err := controllerutil.CreateOrUpdate(ctx, serverClient, apim, func() error {
		// Check nested "optional" fields
		*apim = prepareNestedOptionalFields(*apim)
		topologySpreadConstraints := []corev1.TopologySpreadConstraint{
			{
				MaxSkew:           1,
				TopologyKey:       resources.ZoneLabel,
				WhenUnsatisfiable: corev1.ScheduleAnyway,
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "3scale-api-management",
					},
				},
			},
		}
		// Set TopologySpreadConstraints in APIManager for deployments
		apim.Spec.Apicast.StagingSpec.TopologySpreadConstraints = topologySpreadConstraints
		apim.Spec.Apicast.ProductionSpec.TopologySpreadConstraints = topologySpreadConstraints
		apim.Spec.Backend.CronSpec.TopologySpreadConstraints = topologySpreadConstraints
		apim.Spec.Backend.ListenerSpec.TopologySpreadConstraints = topologySpreadConstraints
		apim.Spec.Backend.WorkerSpec.TopologySpreadConstraints = topologySpreadConstraints
		apim.Spec.System.AppSpec.TopologySpreadConstraints = topologySpreadConstraints
		apim.Spec.System.MemcachedTopologySpreadConstraints = topologySpreadConstraints
		apim.Spec.System.SearchdSpec.TopologySpreadConstraints = topologySpreadConstraints
		apim.Spec.System.SidekiqSpec.TopologySpreadConstraints = topologySpreadConstraints
		apim.Spec.Zync.AppSpec.TopologySpreadConstraints = topologySpreadConstraints
		apim.Spec.Zync.QueSpec.TopologySpreadConstraints = topologySpreadConstraints
		apim.Spec.Zync.DatabaseTopologySpreadConstraints = topologySpreadConstraints

		// General config
		apim.Spec.HighAvailability = &threescalev1.HighAvailabilitySpec{Enabled: true}
		apim.Spec.APIManagerCommonSpec.ResourceRequirementsEnabled = &resourceRequirements
		if apim.Spec.WildcardDomain == "" {
			apim.Spec.APIManagerCommonSpec.WildcardDomain = r.installation.Spec.RoutingSubdomain
		}

		apim.Spec.System.FileStorageSpec = fss
		apim.Spec.PodDisruptionBudget = &threescalev1.PodDisruptionBudgetSpec{Enabled: true}
		apim.Spec.Monitoring = &threescalev1.MonitoringSpec{Enabled: false}
		apim.Spec.ExternalComponents.System.Redis = &ExternalComponentsTrue
		apim.Spec.ExternalComponents.System.Database = &ExternalComponentsTrue
		apim.Spec.ExternalComponents.Backend.Redis = &ExternalComponentsTrue

		// Configure https ports to enable gRPC support
		apim.Spec.Apicast.StagingSpec.HTTPSPort = &apicastport
		apim.Spec.Apicast.ProductionSpec.HTTPSPort = &apicastport

		// Set priority class names
		apim.Spec.System.AppSpec.PriorityClassName = &r.installation.Spec.PriorityClassName
		apim.Spec.System.SidekiqSpec.PriorityClassName = &r.installation.Spec.PriorityClassName
		apim.Spec.System.MemcachedPriorityClassName = &r.installation.Spec.PriorityClassName
		apim.Spec.System.SearchdSpec.PriorityClassName = &r.installation.Spec.PriorityClassName
		apim.Spec.Apicast.StagingSpec.PriorityClassName = &r.installation.Spec.PriorityClassName
		apim.Spec.Apicast.ProductionSpec.PriorityClassName = &r.installation.Spec.PriorityClassName
		apim.Spec.Backend.CronSpec.PriorityClassName = &r.installation.Spec.PriorityClassName
		apim.Spec.Backend.ListenerSpec.PriorityClassName = &r.installation.Spec.PriorityClassName
		apim.Spec.Backend.WorkerSpec.PriorityClassName = &r.installation.Spec.PriorityClassName
		apim.Spec.Zync.AppSpec.PriorityClassName = &r.installation.Spec.PriorityClassName
		apim.Spec.Zync.QueSpec.PriorityClassName = &r.installation.Spec.PriorityClassName
		apim.Spec.Zync.DatabasePriorityClassName = &r.installation.Spec.PriorityClassName

		// Reconcile replicas
		if apim.Spec.System.AppSpec.Replicas == nil || *apim.Spec.System.AppSpec.Replicas < replicas["systemApp"] {
			apim.Spec.System.AppSpec.Replicas = &systemAppReplicas
		}

		if apim.Spec.System.SidekiqSpec.Replicas == nil || *apim.Spec.System.SidekiqSpec.Replicas < replicas["systemSidekiq"] {
			apim.Spec.System.SidekiqSpec.Replicas = &systemSidekiqReplicas
		}

		if apim.Spec.Apicast.StagingSpec.Replicas == nil || *apim.Spec.Apicast.StagingSpec.Replicas < replicas["apicastStage"] {
			apim.Spec.Apicast.StagingSpec.Replicas = &apicastStageReplicas
		}

		if apim.Spec.Backend.CronSpec.Replicas == nil || *apim.Spec.Backend.CronSpec.Replicas < replicas["backendCron"] {
			apim.Spec.Backend.CronSpec.Replicas = &backendCronReplicas
		}

		if apim.Spec.Zync.AppSpec.Replicas == nil || *apim.Spec.Zync.AppSpec.Replicas < replicas["zyncApp"] {
			apim.Spec.Zync.AppSpec.Replicas = &zyncReplicas
		}

		if apim.Spec.Zync.QueSpec.Replicas == nil || *apim.Spec.Zync.QueSpec.Replicas < replicas["zyncQue"] {
			apim.Spec.Zync.QueSpec.Replicas = &zyncQueReplicas
		}

		// Reconcile pod affinity
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

		err = productConfig.Configure(apim)

		if err != nil {
			return err
		}

		owner.AddIntegreatlyOwnerAnnotations(apim, r.installation)

		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	r.log.Infof("API Manager: ", l.Fields{"status": status})

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
			r.log.Error("Error getting system-provider route", nil, err)
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
			// This scenario will manifest during a backup and restore and also if the product ns was accidentally deleted.
			return r.resyncRoutes(ctx, serverClient)
		}
	}
	r.log.Infof("3Scale Deployments in progress",
		l.Fields{"starting": len(apim.Status.Deployments.Starting), "stopped": len(apim.Status.Deployments.Stopped), "ready": len(apim.Status.Deployments.Ready)})

	return integreatlyv1alpha1.PhaseInProgress, nil
}

func prepareNestedOptionalFields(apim threescalev1.APIManager) threescalev1.APIManager {
	if apim.Spec.System == nil {
		apim.Spec.System = &threescalev1.SystemSpec{
			FileStorageSpec: &threescalev1.SystemFileStorageSpec{},
			AppSpec:         &threescalev1.SystemAppSpec{},
			SidekiqSpec:     &threescalev1.SystemSidekiqSpec{},
			SearchdSpec:     &threescalev1.SystemSearchdSpec{},
			SphinxSpec:      &threescalev1.SystemSphinxSpec{},
			DatabaseSpec:    &threescalev1.SystemDatabaseSpec{},
		}
	}
	if apim.Spec.Apicast == nil {
		apim.Spec.Apicast = &threescalev1.ApicastSpec{
			ProductionSpec: &threescalev1.ApicastProductionSpec{},
			StagingSpec:    &threescalev1.ApicastStagingSpec{},
		}
	}
	if apim.Spec.Backend == nil {
		apim.Spec.Backend = &threescalev1.BackendSpec{
			ListenerSpec: &threescalev1.BackendListenerSpec{},
			WorkerSpec:   &threescalev1.BackendWorkerSpec{},
			CronSpec:     &threescalev1.BackendCronSpec{},
		}
	}
	if apim.Spec.Zync == nil {
		apim.Spec.Zync = &threescalev1.ZyncSpec{
			AppSpec: &threescalev1.ZyncAppSpec{},
			QueSpec: &threescalev1.ZyncQueSpec{},
		}
	}
	if apim.Spec.ExternalComponents == nil {
		apim.Spec.ExternalComponents = &threescalev1.ExternalComponentsSpec{
			System:  &threescalev1.ExternalSystemComponents{},
			Backend: &threescalev1.ExternalBackendComponents{},
		}
	}

	return apim
}

func (r *Reconciler) routesExist(ctx context.Context, serverClient k8sclient.Client) (bool, error) {
	expectedRoutes := 4
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
	r.log.Warningf("Required number of routes do not exist", l.Fields{"found": len(routes.Items), "required": expectedRoutes})
	return false, nil
}

func (r *Reconciler) resyncRoutes(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	ns := r.Config.GetNamespace()
	podname := ""

	pods := &corev1.PodList{}
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(ns),
		k8sclient.MatchingLabels(map[string]string{"deployment": "system-sidekiq"}),
	}
	err := client.List(ctx, pods, listOpts...)
	if err != nil {
		r.log.Error("Error getting list of pods", nil, err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase == "Running" {
			podname = pod.ObjectMeta.Name
			break
		}
	}

	if podname == "" {
		r.log.Info("Waiting on system-sidekiq pod to start, 3Scale install in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	stdout, stderr, err := r.podExecutor.ExecuteRemoteCommand(ns, podname, []string{"/bin/bash",
		"-c", "bundle exec rake zync:resync:domains"})
	if err != nil {
		r.log.Error("Failed to resync 3Scale routes", nil, err)
		return integreatlyv1alpha1.PhaseFailed, nil
	} else if stderr != "" {
		err := errors.New(stderr)
		r.log.Error("Error attempting to resync 3Scale routes", nil, err)
		return integreatlyv1alpha1.PhaseFailed, err
	} else {
		r.log.Infof("Resync 3Scale routes result", l.Fields{"stdout": stdout})
		return integreatlyv1alpha1.PhaseInProgress, nil
	}
}

func (r *Reconciler) reconcileBlobStorage(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Reconciling blob storage")
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

func (r *Reconciler) getBlobStorageFileStorageSpec(ctx context.Context, serverClient k8sclient.Client, platformType configv1.PlatformType) (*threescalev1.SystemFileStorageSpec, error) {
	// create s3 credentials secret
	credSec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s3CredentialsSecretName,
			Namespace: r.Config.GetNamespace(),
		},
		Data: map[string][]byte{},
	}

	var err error
	var isSTS bool
	switch platformType {
	case configv1.AWSPlatformType:
		blobStorage := &crov1.BlobStorage{}
		// get blob storage cr
		err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: fmt.Sprintf("%s%s", constants.ThreeScaleBlobStoragePrefix, r.installation.Name), Namespace: r.installation.Namespace}, blobStorage)
		if err != nil {
			return nil, fmt.Errorf("failed to get blob storage custom resource: %w", err)
		}

		// get blob storage connection secret
		blobStorageSec := &corev1.Secret{}
		err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: blobStorage.Status.SecretRef.Name, Namespace: blobStorage.Status.SecretRef.Namespace}, blobStorageSec)
		if err != nil {
			return nil, fmt.Errorf("failed to get blob storage connection secret: %w", err)
		}
		isSTS, err = sts.IsClusterSTS(ctx, serverClient, r.log)
		if err != nil {
			return nil, fmt.Errorf("error checking STS mode: %w", err)
		}
		if isSTS {
			err = r.createStsS3Secret(ctx, serverClient, credSec, blobStorageSec)
		} else {
			_, err = controllerutil.CreateOrUpdate(ctx, serverClient, credSec, func() error {
				// Map known key names from CRO, and append any additional values that may be used for Minio
				for key, value := range blobStorageSec.Data {
					switch key {
					case "credentialKeyID":
						credSec.Data[apps.AwsAccessKeyID] = blobStorageSec.Data["credentialKeyID"]
					case "credentialSecretKey":
						credSec.Data[apps.AwsSecretAccessKey] = blobStorageSec.Data["credentialSecretKey"]
					case "bucketName":
						credSec.Data[apps.AwsBucket] = blobStorageSec.Data["bucketName"]
					case "bucketRegion":
						credSec.Data[apps.AwsRegion] = blobStorageSec.Data["bucketRegion"]
					default:
						credSec.Data[key] = value
					}
				}
				return nil
			})
		}
	default:
		err = fmt.Errorf("unsupported cluster type: %s", platformType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create or update blob storage aws credentials secret: %w", err)
	}

	systemFileSpec := &threescalev1.SystemFileStorageSpec{
		S3: &threescalev1.SystemS3Spec{
			ConfigurationSecretRef: corev1.LocalObjectReference{
				Name: s3CredentialsSecretName,
			},
		},
	}

	if isSTS {
		audience := stsTokenAudience
		systemFileSpec.S3.STS = &threescalev1.STSSpec{
			Enabled:  &isSTS,
			Audience: &audience,
		}
	}

	return systemFileSpec, nil
}

func (r *Reconciler) createStsS3Secret(ctx context.Context, serverClient k8sclient.Client, credSec *corev1.Secret, blobStorageSec *corev1.Secret) error {
	stsSecret := &corev1.Secret{}
	if err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: stsS3CredentialsSecretName, Namespace: r.Config.GetNamespace()}, stsSecret); err != nil {
		return fmt.Errorf("failed to get 3scale sts secret resource: %w", err)
	}

	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, credSec, func() error {
		for key := range blobStorageSec.Data {
			switch key {
			case "bucketName":
				credSec.Data[apps.AwsBucket] = blobStorageSec.Data["bucketName"]
			case "bucketRegion":
				credSec.Data[apps.AwsRegion] = blobStorageSec.Data["bucketRegion"]
			}
		}

		credSec.Data[apps.AwsRoleArn] = stsSecret.Data["role_arn"]
		credSec.Data[apps.AwsWebIdentityTokenFile] = []byte(stsWebIdentityTokenFilePath)

		return nil
	})

	return err
}

// reconcileExternalDatasources provisions 2 redis caches and a postgres instance
// which are used when 3scale HighAvailability mode is enabled
func (r *Reconciler) reconcileExternalDatasources(ctx context.Context, serverClient k8sclient.Client, activeQuota string, platformType configv1.PlatformType) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Reconciling external datastores")
	ns := r.installation.Namespace

	// setup backend redis custom resource
	// this will be used by the cloud resources operator to provision a redis instance
	r.log.Info("Creating backend redis instance")
	backendRedisName := fmt.Sprintf("%s%s", constants.ThreeScaleBackendRedisPrefix, r.installation.Name)

	// If there is a quota change, the quota on the installation spec would not be set to the active quota yet
	quotaChange := isQuotaChanged(r.installation.Status.Quota, activeQuota)

	r.log.Infof("Backend redis config", map[string]interface{}{"quotaChange": quotaChange, "activeQuota": activeQuota})
	backendRedis, err := croUtil.ReconcileRedis(ctx, serverClient, defaultInstallationNamespace, r.installation.Spec.Type, croUtil.TierProduction, backendRedisName, ns, backendRedisName, ns, r.Config.GetBackendRedisNodeSize(activeQuota, platformType), quotaChange, quotaChange, func(cr metav1.Object) error {
		owner.AddIntegreatlyOwnerAnnotations(cr, r.installation)
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile backend redis request: %w", err)
	}

	// setup system redis custom resource
	// this will be used by the cloud resources operator to provision a redis instance
	r.log.Info("Creating system redis instance")
	systemRedisName := fmt.Sprintf("%s%s", constants.ThreeScaleSystemRedisPrefix, r.installation.Name)
	systemRedis, err := croUtil.ReconcileRedis(ctx, serverClient, defaultInstallationNamespace, r.installation.Spec.Type, croUtil.TierProduction, systemRedisName, ns, systemRedisName, ns, "", false, false, func(cr metav1.Object) error {
		owner.AddIntegreatlyOwnerAnnotations(cr, r.installation)
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile system redis request: %w", err)
	}

	// setup postgres cr for the cloud resource operator
	// this will be used by the cloud resources operator to provision a postgres instance
	r.log.Info("Creating postgres instance")
	postgresName := fmt.Sprintf("%s%s", constants.ThreeScalePostgresPrefix, r.installation.Name)
	postgres, err := croUtil.ReconcilePostgres(ctx, serverClient, defaultInstallationNamespace, r.installation.Spec.Type, croUtil.TierProduction, postgresName, ns, postgresName, ns, constants.PostgresApplyImmediately, "", "", func(cr metav1.Object) error {
		owner.AddIntegreatlyOwnerAnnotations(cr, r.installation)
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile postgres request: %w", err)
	}
	if postgres.Status.Phase != types.PhaseComplete {
		return integreatlyv1alpha1.PhaseAwaitingCloudResources, nil
	}
	phase, err := resources.ReconcileRedisAlerts(ctx, serverClient, r.installation, backendRedis, r.log)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile redis alerts: %w", err)
	}
	if phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, nil
	}

	// create Redis Cpu Usage High alert
	err = resources.CreateRedisCpuUsageAlerts(ctx, serverClient, r.installation, backendRedis, r.log)
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

	phase, err = resources.ReconcileRedisAlerts(ctx, serverClient, r.installation, systemRedis, r.log)
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

	messageBusKeys := []string{"MESSAGE_BUS_URL", "MESSAGE_BUS_NAMESPACE", "MESSAGE_BUS_SENTINEL_HOSTS", "MESSAGE_BUS_SENTINEL_ROLE"}

	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, redisSecret, func() error {
		uri := systemCredSec.Data["uri"]
		port := systemCredSec.Data["port"]
		conn := fmt.Sprintf("redis://%s:%s/1", uri, port)
		redisSecret.Data["URL"] = []byte(conn)
		for _, key := range messageBusKeys {
			if redisSecret.Data[key] != nil {
				delete(redisSecret.Data, key)
			}
		}

		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create or update 3scale %s connection secret: %w", externalRedisSecretName, err)
	}

	// reconcile postgres alerts
	phase, err = resources.ReconcilePostgresAlerts(ctx, serverClient, r.installation, postgres, r.log)
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

func isQuotaChanged(newQuota string, activeQuota string) bool {
	// During fresh installation, quota is not set until installation completes
	if newQuota == "" {
		return false
	}

	return newQuota != activeQuota
}

func (r *Reconciler) reconcileOutgoingEmailAddress(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	existingSMTPFromAddress, err := resources.GetSMTPFromAddress(ctx, serverClient, r.log, r.installation)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	accessToken, err := r.GetAdminToken(ctx, serverClient)
	if err != nil {
		r.log.Info("Failed to get admin token in reconcileOutgoingEmailAddresss: " + err.Error())
		return integreatlyv1alpha1.PhaseInProgress, err
	}

	err = r.reconcileTenantOutgoingEmailAddress(ctx, serverClient, existingSMTPFromAddress)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	_, err = r.tsClient.SetFromEmailAddress(existingSMTPFromAddress, *accessToken)
	if err != nil {
		r.log.Error("Failed to set email from address:", nil, err)
		return integreatlyv1alpha1.PhaseFailed, err
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
		r.log.Warningf("Cannot configure SSO integration without SSO", l.Fields{"ns": rhssoNamespace, "realm": rhssoRealm})
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
		r.log.Error("Error retrieving client secret", nil, err)
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
		r.log.Info("Failed to get admin token: " + err.Error())
		return integreatlyv1alpha1.PhaseInProgress, err
	}
	_, err = r.tsClient.GetAuthenticationProviderByName(rhssoIntegrationName, *accessToken)
	if err != nil && !tsIsNotFoundError(err) {
		r.log.Info("Failed to get authentication provider:" + err.Error())
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
			if err != nil {
				r.log.Info("Failed to add authentication provider:" + err.Error())
			}
			return integreatlyv1alpha1.PhaseInProgress, err
		}
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) getOAuthClientName() string {
	return r.installation.Spec.NamespacePrefix + string(r.Config.GetProductName())
}

func (r *Reconciler) reconcileOpenshiftUsers(ctx context.Context, _ *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Reconciling openshift users to 3scale")

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
		r.log.Info("Failed to retrieve admin name and password from secret: " + err.Error())
		return integreatlyv1alpha1.PhaseInProgress, err
	}

	kcu, err := rhsso.GetKeycloakUsers(ctx, serverClient, rhssoConfig.GetNamespace())
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	tsUsers, err := r.tsClient.GetUsers(*accessToken)
	if err != nil {
		r.log.Info("Failed to get users:" + err.Error())
		return integreatlyv1alpha1.PhaseInProgress, err
	}

	added, deleted, updated := r.getUserDiff(ctx, serverClient, kcu, tsUsers.Users)
	// reset the user action metric before we re-reconcile
	// in order to get up to date metrics on user creation
	metrics.ResetThreeScaleUserAction()
	// the deleted entries are addressed first
	// a common use case is where one idp is added to give early access to the cluster
	// later that idp is removed and a more permanent one is added
	// if there are any duplicate emails across the set of users the user from the first idp
	// should be removed first and that allows for the new one which had a potential conflict
	// can now be added.
	for _, tsUser := range deleted {
		if tsUser.UserDetails.Username != *systemAdminUsername {
			statusCode := http.StatusServiceUnavailable

			res, err := r.tsClient.DeleteUser(tsUser.UserDetails.Id, *accessToken)
			if err != nil {
				r.log.Error("msg", nil, err)
			} else {
				statusCode = res.StatusCode
			}

			metrics.SetThreeScaleUserAction(statusCode, strconv.Itoa(tsUser.UserDetails.Id), http.MethodDelete)

			if statusCode != http.StatusOK {
				r.log.Error("msg", nil, errors.New("error on http request"))
			}
		}
	}

	for _, tsUser := range updated {
		if tsUser.UserDetails.Username != *systemAdminUsername {
			genKcUser, err := getGeneratedKeycloakUser(ctx, serverClient, rhssoConfig.GetNamespace(), tsUser)

			if err != nil {
				r.log.Warning("Failed to get generate keycloak user: " + err.Error())
				continue
			}

			_, err = r.tsClient.UpdateUser(tsUser.UserDetails.Id, strings.ToLower(genKcUser.Spec.User.UserName), tsUser.UserDetails.Email, *accessToken)
			if err != nil {
				r.log.Warning("Failed to updating 3scale user details: " + err.Error())
			}
		}
	}

	for _, kcUser := range added {
		user, err := r.tsClient.GetUser(strings.ToLower(kcUser.UserName), *accessToken)
		if err != nil {
			r.log.Error("Failed to get user", nil, err)
		}

		// recheck the user is new.
		// 3scale user may being update during the update phase
		if user == nil {
			statusCode := http.StatusServiceUnavailable
			res, err := r.tsClient.AddUser(strings.ToLower(kcUser.UserName), strings.ToLower(kcUser.Email), "", *accessToken)

			if err != nil {
				r.log.Error("msg", nil, err)
			} else {
				statusCode = res.StatusCode
			}

			// when the failure of user happens we don't want to block the reconciler.
			// failure to create a user can happen in the case of the username being too long
			// the max allowed user length is 40 characters in 3scale.
			// The reconciler will continue to allow the installation to happen and a metric
			// will be exposed and alert fire to alert to the creation failure
			metrics.SetThreeScaleUserAction(statusCode, kcUser.UserName, http.MethodPost)

			if statusCode != http.StatusCreated {
				r.log.Error("msg", nil, errors.New("error on http request"))
			}
		}
	}

	// update KeycloakUser attribute after user is created in 3scale
	phase, err := r.updateKeycloakUsersAttributeWith3ScaleUserId(ctx, serverClient, kcu, accessToken)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, err
	}

	openshiftAdminGroup := &usersv1.Group{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: "dedicated-admins"}, openshiftAdminGroup)
	if err != nil && !k8serr.IsNotFound(err) {
		r.log.Info("Failed to retrieve dedicated admins: " + err.Error())
		return integreatlyv1alpha1.PhaseInProgress, err
	}
	newTsUsers, err := r.tsClient.GetUsers(*accessToken)
	if err != nil {
		r.log.Info("Failed to get users: " + err.Error())
		return integreatlyv1alpha1.PhaseInProgress, err
	}

	err = syncOpenshiftAdminMembership(openshiftAdminGroup, newTsUsers, *systemAdminUsername, r.tsClient, *accessToken)
	if err != nil {
		r.log.Info("Failed to sync openshift admin membership: " + err.Error())
		return integreatlyv1alpha1.PhaseInProgress, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) updateKeycloakUsersAttributeWith3ScaleUserId(ctx context.Context, serverClient k8sclient.Client, kcu []keycloak.KeycloakAPIUser, accessToken *string) (integreatlyv1alpha1.StatusPhase, error) {
	rhssoConfig, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	userCreated3ScaleName := "3scale_user_created"
	for _, user := range kcu {
		tsUser, err := r.tsClient.GetUser(strings.ToLower(user.UserName), *accessToken)
		if err != nil {
			// Continue installation to not block for when users could not be created in 3scale (i.e. too many characters in username)
			continue
		}

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
		if kcUser.Name == "" {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get valid generated username")
		}

		_, err = controllerutil.CreateOrUpdate(ctx, serverClient, kcUser, func() error {
			user.Attributes[userCreated3ScaleName] = []string{"true"}
			user.Attributes[user3ScaleID] = []string{fmt.Sprint(tsUser.UserDetails.Id)}
			kcUser.Spec.User = user
			return nil
		})
		if err != nil {
			return integreatlyv1alpha1.PhaseInProgress,
				fmt.Errorf("failed to update KeycloakUser CR with %s attribute: %w", userCreated3ScaleName, err)
		}
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcile3scaleMultiTenancy(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	mtUserIdentities, err := userHelper.GetMultiTenantUsers(ctx, serverClient)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	totalIdentities := len(mtUserIdentities)
	r.log.Infof("Found user identities from MT accounts",
		l.Fields{"totalIdentities": totalIdentities},
	)

	// get 3scale master access token
	accessToken, err := r.GetMasterToken(ctx, serverClient)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	defaultPageSize := 500
	totalPages := 1
	if totalIdentities > defaultPageSize {
		totalPages = totalIdentities / defaultPageSize
	}

	// adding an extra page in case of accounts set to be deleted
	totalPages++

	var allAccounts []AccountDetail
	for page := 1; page <= totalPages; page++ {
		// list 3scale tenant accounts

		r.log.Infof("Retrieving list of MT accounts available ",
			l.Fields{"Page": page},
		)
		accounts, err := r.tsClient.ListTenantAccounts(*accessToken, page, func(ac AccountDetail) bool {
			return ac.Id != 1 && ac.Id != 2
		})
		if err != nil {
			r.log.Error("failed to get accounts from 3scale API:", nil, err)
			return integreatlyv1alpha1.PhaseFailed, err
		}
		allAccounts = append(allAccounts, accounts...)
	}

	r.log.Infof("Total of accounts available",
		l.Fields{
			"totalPages":                totalPages,
			"totalOpenshiftUsers":       totalIdentities,
			"total3scaleTenantAccounts": len(allAccounts),
		},
	)

	setTenantMetrics(mtUserIdentities, allAccounts)

	r.log.Info("getAccessTokenSecret")
	signUpAccountsSecret, err := getAccessTokenSecret(ctx, serverClient, r.Config.GetNamespace())
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	r.log.Info("getAccessTokenSecret length: " + strconv.Itoa(len(signUpAccountsSecret.Data)))

	tenantsCreated, err := getAccountsCreatedCM(ctx, serverClient, r.Config.GetNamespace())
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// looping through the accounts to reconcile default config back
	for index, account := range allAccounts {
		r.log.Infof("Checking 3scale account", l.Fields{"tenantAccountName": account.OrgName})

		state, created := tenantsCreated.Data[account.OrgName]
		if created && state == "true" {
			continue
		}

		if account.State == "approved" {
			r.log.Infof("3scale account is approved", l.Fields{"tenantAccountName": account.OrgName})
			breakout := false
			for _, user := range account.Users.User {
				if user.State == "pending" {
					r.log.Infof("Activating user access to new tenant account",
						l.Fields{
							"userName":           user.Username,
							"tenantAccountName":  account.OrgName,
							"tenantAccountState": account.State,
						},
					)

					err = r.tsClient.ActivateUser(*accessToken, account.Id, user.Id)
					if err != nil {
						r.log.Error("Error activating user access to new tenant account",
							l.Fields{
								"userName":          user.Username,
								"tenantAccountName": account.OrgName,
							},
							err,
						)
						breakout = true
					}
				}
			}
			if breakout {
				continue
			}

			val, ok := signUpAccountsSecret.Data[string(account.OrgName)]
			if !ok || string(val) == "" {
				r.log.Infof("Tenant account does not have access token created",
					l.Fields{
						"tenantAccountId":    account.Id,
						"tenantAccountName":  account.OrgName,
						"tenantAccountState": account.State,
					},
					//TODO: delete account?
				)
				continue
			}

			signUpAccount := SignUpAccount{
				AccountDetail: account,
				AccountAccessToken: AccountAccessToken{
					Value: string(val),
				},
			}

			r.log.Infof("Adding authentication provider to tenant account",
				l.Fields{
					"tenantAccountId":    account.Id,
					"tenantAccountName":  account.OrgName,
					"tenantAccountState": account.State,
				},
			)

			// verify if the account have the auth provider already
			err = r.addAuthProviderToMTAccount(ctx, serverClient, signUpAccount)
			if err != nil {
				r.log.Error("Error adding authentication provider to tenant account",
					l.Fields{
						"tenantAccountId":    account.Id,
						"tenantAccountName":  account.OrgName,
						"tenantAccountState": account.State,
					},
					err,
				)
				continue
			}

			// Get the account's corresponding KeycloakUser for later verification
			kcUser, err := r.getKeycloakUserFromAccount(serverClient, account.OrgName)
			if err != nil {
				r.log.Error("Failed to get KeycloakUser for tenant account",
					l.Fields{
						"tenantAccountId":    account.Id,
						"tenantAccountName":  account.OrgName,
						"tenantAccountState": account.State,
					},
					err,
				)
				continue
			}

			// Get the account's corresponding KeycloakClient for later verification
			kcClient, err := r.getKeycloakClientFromAccount(serverClient, account.OrgName)
			if err != nil {
				r.log.Error("Failed to get KeycloakClient for tenant account",
					l.Fields{
						"tenantAccountId":    account.Id,
						"tenantAccountName":  account.OrgName,
						"tenantAccountState": account.State,
					},
					err,
				)
				continue
			}

			r.log.Infof("Checking Keycloak state...", l.Fields{"tenantAccountName": account.OrgName, "kcUser.Status.Phase": kcUser.Status.Phase, "kcClient.Status.Ready": kcClient.Status.Ready})

			// Only add the ssoReady annotation if the tenant account's corresponding KeycloakUser and KeycloakClient CR's are ready.
			// If not, continue to next account.
			if kcUser.Status.Phase == keycloak.UserPhaseReconciled && kcClient.Status.Ready {

				r.log.Infof("Adding SSO on 3scale account ", l.Fields{"tenantAccountName": account.OrgName})

				// Add ssoReady annotation to the user CR associated with the tenantAccount's OrgName
				// This is required by the apimanagementtenant_controller so it can finish reconciling the APIManagementTenant CR
				err = r.addSSOReadyAnnotationToUser(ctx, serverClient, account.OrgName)
				if err != nil {
					r.log.Error("Error adding ssoReady annotation for the user associated with the tenant account org",
						l.Fields{
							"tenantAccountOrgName": account.OrgName,
						},
						err,
					)
					continue
				}

				r.log.Infof("Reconciling Dashboard link for ", l.Fields{"tenantAccountName": account.OrgName})

				// Only add the dashboard link when account fully ready
				err = r.reconcileDashboardLink(ctx, serverClient, account.OrgName, account.AdminBaseURL)
				if err != nil {
					r.log.Error("Error reconciling console link for the tenant account",
						l.Fields{
							"tenantAccountId":   account.Id,
							"tenantAccountName": account.OrgName,
						},
						err,
					)
					continue
				}

				r.log.Infof("Setting account created in config map to true", l.Fields{"tenantAccountName": account.OrgName})
				if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, tenantsCreated, func() error {
					tenantsCreated.Data[account.OrgName] = "true"
					tenantsCreated.ObjectMeta.ResourceVersion = ""
					return nil
				}); err != nil {
					r.log.Error("Error setting account created in config map to true", l.Fields{"tenantAccountName": account.OrgName}, err)
					return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating/updating tenant created CM: %w", err)
				}
			}
		} else if account.State != "scheduled_for_deletion" {
			r.log.Infof("Deleting broke account for recreation",
				l.Fields{
					"tenantAccountId":    account.Id,
					"tenantAccountName":  account.OrgName,
					"tenantAccountState": account.State,
				},
			)

			err = r.tsClient.DeleteTenant(*accessToken, account.Id)
			if err != nil {
				r.log.Error("Error deleting broken account",
					l.Fields{
						"tenantAccountId":    account.Id,
						"tenantAccountName":  account.OrgName,
						"tenantAccountState": account.State,
					},
					err,
				)
			}

			//remove account from the list of accounts so it can be recreated
			r.log.Infof("Account removed to be recreated",
				l.Fields{"tenantAccountRemoved": allAccounts[index]},
			)
			allAccounts = append(allAccounts[:index], allAccounts[index+1:]...)
		}
	}

	r.log.Info("creating new MT accounts in 3scale")

	// creating new MT accounts in 3scale
	accountsToBeCreated, emailAddrs, err := getMTAccountsToBeCreated(mtUserIdentities, allAccounts)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	r.log.Infof("Retrieving tenant accounts to be created",
		l.Fields{
			"accountsToBeCreated": accountsToBeCreated,
			"totalAccounts":       len(accountsToBeCreated),
		},
	)

	for idx, account := range accountsToBeCreated {

		r.log.Info("Accounts to be created loop")

		pw, err := r.getTenantAccountPassword(ctx, serverClient, account)
		if err != nil {
			r.log.Error("Failed to get account tenant password:", nil, err)
			return integreatlyv1alpha1.PhaseFailed, err
		}

		// Create 3scale account
		newSignupAccount, err := r.tsClient.CreateTenant(*accessToken, account, pw, emailAddrs[idx])
		if err != nil {
			r.log.Error("Error creating tenant account",
				l.Fields{"tenantAccountName": account.OrgName},
				err,
			)

			// Attempt a delete of Tenant to force re-entry !!!

			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating tenant account: %s, Error=[%v]", account.OrgName, err)
		}

		r.log.Infof("New tenant account created",
			l.Fields{
				"tenantAccountId":    newSignupAccount.AccountDetail.Id,
				"tenantAccountName":  newSignupAccount.AccountDetail.OrgName,
				"tenantAccountState": newSignupAccount.AccountDetail.State,
			},
		)

		if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, signUpAccountsSecret, func() error {
			r.log.Info("Creating/updating signUpAccountsSecret " + signUpAccountsSecret.Name + " " + signUpAccountsSecret.Namespace)
			signUpAccountsSecret.Data[account.OrgName] = []byte(newSignupAccount.AccountAccessToken.Value)
			signUpAccountsSecret.ObjectMeta.ResourceVersion = ""
			return nil
		}); err != nil {
			r.log.Error("Error creating access token secret ", nil, err)
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating access token secret: %w", err)
		}
		r.log.Info("After signUpAccountsSecret " + signUpAccountsSecret.Name + " " + signUpAccountsSecret.Namespace)
	}

	// deleting MT accounts in 3scale
	accountsToBeDeleted := getMTAccountsToBeDeleted(mtUserIdentities, allAccounts)
	r.log.Infof(
		"Deleting unused tenant accounts",
		l.Fields{
			"accountsToBeDeleted": accountsToBeDeleted,
			"totalAccounts":       len(accountsToBeDeleted),
		},
	)
	err = r.tsClient.DeleteTenants(*accessToken, accountsToBeDeleted)
	if err != nil {
		r.log.Error("error deleting tenant accounts:", nil, err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Remove redundant access token secrets
	for _, account := range accountsToBeDeleted {
		_, ok := signUpAccountsSecret.Data[string(account.OrgName)]
		if ok {
			delete(signUpAccountsSecret.Data, string(account.OrgName))
		}
		err := r.removeTenantAccountPassword(ctx, serverClient, account)
		if err != nil {
			r.log.Error("error deleting tenant account password",
				l.Fields{
					"tenantAccount": account,
				},
				err,
			)
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error deleting tenant account password: %s, Error=[%v]", account.OrgName, err)
		}
	}

	if len(accountsToBeCreated) > 0 {
		r.log.Infof("Returning in progress as there were accounts created and users need to be activated",
			l.Fields{"totalAccountsCreated": len(accountsToBeCreated)},
		)
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func setTenantMetrics(users []userHelper.MultiTenantUser, accounts []AccountDetail) {
	metrics.ResetNoActivated3ScaleTenantAccount()

	for _, user := range users {
		if !accountExists(user.TenantName, accounts) {
			metrics.SetNoActivated3ScaleTenantAccount(user.Username)
		}
	}
}

func accountExists(tenant string, accounts []AccountDetail) bool {
	for _, acc := range accounts {
		if tenant == acc.OrgName && acc.State == "approved" {
			return true
		}
	}
	return false
}

func getAccessTokenSecret(ctx context.Context, serverClient k8sclient.Client, namespace string) (*corev1.Secret, error) {
	signUpAccountsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mt-signupaccount-3scale-access-token",
			Namespace: namespace,
		},
	}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: signUpAccountsSecret.Name, Namespace: signUpAccountsSecret.Namespace}, signUpAccountsSecret)
	if !k8serr.IsNotFound(err) && err != nil {
		return nil, fmt.Errorf("error getting access token secret: %w", err)
	} else if k8serr.IsNotFound(err) {
		signUpAccountsSecret.Data = map[string][]byte{}
	}

	return signUpAccountsSecret, nil
}

func getAccountsCreatedCM(ctx context.Context, serverClient k8sclient.Client, namespace string) (*corev1.ConfigMap, error) {
	accountsCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tenants-created",
			Namespace: namespace,
		},
	}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: accountsCM.Name, Namespace: accountsCM.Namespace}, accountsCM)
	if !k8serr.IsNotFound(err) && err != nil {
		return nil, fmt.Errorf("error getting accounts created configmap: %w", err)
	} else if k8serr.IsNotFound(err) {
		accountsCM.Data = map[string]string{}
	}

	return accountsCM, nil
}

func (r *Reconciler) removeTenantAccountPassword(ctx context.Context, serverClient k8sclient.Client, account AccountDetail) error {

	r.log.Infof("Remove Tenant Account Password", l.Fields{"tenant": account.OrgName})

	tenantAccountSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      "tenant-account-passwords",
		},
	}

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: tenantAccountSecret.Name, Namespace: tenantAccountSecret.Namespace}, tenantAccountSecret)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			r.log.Error("Failed to get tenantAccountPasswords secret", nil, err)
			return err
		}
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, tenantAccountSecret, func() error {
		if tenantAccountSecret.Data == nil || tenantAccountSecret.Data[account.OrgName] == nil {
			r.log.Infof("Tenant Account Password not found", l.Fields{"tenant": account.OrgName})
			return nil
		} else {
			delete(tenantAccountSecret.Data, account.OrgName)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("error occurred while removing tenant Account password: %w", err)
	}

	return nil
}

func (r *Reconciler) getTenantAccountPassword(ctx context.Context, serverClient k8sclient.Client, account AccountDetail) (string, error) {
	var pw = ""
	tenantAccountSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      "tenant-account-passwords",
		},
	}

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: tenantAccountSecret.Name, Namespace: tenantAccountSecret.Namespace}, tenantAccountSecret)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			r.log.Error("Failed to get tenantAccountPasswords secret", nil, err)
			return "", err
		}
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, tenantAccountSecret, func() error {
		if tenantAccountSecret.Data == nil {
			tenantAccountSecret.Data = map[string][]byte{}
		}
		if tenantAccountSecret.Data[account.Name] == nil {
			pw = resources.GenerateRandomPassword(20, 2, 2, 2)
			tenantAccountSecret.Data[account.Name] = []byte(pw)
		} else {
			pw = string(tenantAccountSecret.Data[account.Name])
		}
		if pw == "" {
			return fmt.Errorf("failed to generate password")
		}
		return nil
	}); err != nil {
		return "", fmt.Errorf("error occurred while creating or updating tenant Account Secret: %w", err)
	}

	return pw, nil
}

func (r *Reconciler) reconcileDashboardLink(ctx context.Context, serverClient k8sclient.Client, username string, tenantLink string) error {
	cl := &consolev1.ConsoleLink{
		ObjectMeta: metav1.ObjectMeta{
			Name: username + "-3scale",
		},
	}

	tenantNamespaces := []string{fmt.Sprintf("%s-stage", username), fmt.Sprintf("%s-dev", username)}
	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, cl, func() error {
		cl.Spec = consolev1.ConsoleLinkSpec{
			Location: consolev1.NamespaceDashboard,
			Link: consolev1.Link{
				Href: tenantLink,
				Text: "API Management",
			},
			NamespaceDashboard: &consolev1.NamespaceDashboardSpec{
				Namespaces: tenantNamespaces,
			},
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error reconciling console link: %v", err)
	}

	return nil
}

func (r *Reconciler) addAuthProviderToMTAccount(ctx context.Context, serverClient k8sclient.Client, account SignUpAccount) error {

	tenantID := string(account.AccountDetail.OrgName)
	clientID := fmt.Sprintf("%s-%s", multitenantID, tenantID)
	integration := fmt.Sprintf("%s-%s", rhssoIntegrationName, clientID)

	isAdded, err := r.tsClient.IsAuthProviderAdded(account.AccountAccessToken.Value,
		integration, account.AccountDetail)
	if err != nil {
		return err
	}
	if isAdded {
		return nil
	}

	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tenant-oauth-client-secrets",
			Namespace: r.installation.GetNamespace(),
		},
	}

	err = serverClient.Get(ctx, k8sclient.ObjectKey{
		Name:      oauthClientSecrets.Name,
		Namespace: oauthClientSecrets.Namespace},
		oauthClientSecrets,
	)
	if err != nil {
		r.log.Error("could not find secret", l.Fields{"secret": oauthClientSecrets.Name, "operatorNamespace": oauthClientSecrets.Namespace}, err)
		return fmt.Errorf("could not find %s Secret: %w", oauthClientSecrets.Name, err)
	}

	secret, ok := oauthClientSecrets.Data[tenantID]
	if !ok {
		r.log.Error("could not find tenant key in secret", l.Fields{"tenant": tenantID, "secret": oauthClientSecrets.Name}, err)
		return fmt.Errorf("could not find %s key in %s Secret: %w", tenantID, oauthClientSecrets.Name, err)
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, oauthClientSecrets, func() error {
		oauthClientSecrets.ObjectMeta.ResourceVersion = ""
		return nil
	}); err != nil {
		return fmt.Errorf("error creating or updating RHSSO secret clients : %w", err)
	}

	rhssoConfig, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		return fmt.Errorf("error getting RHSSO config: %w", err)
	}

	r.log.Infof("Add auth provider to new account", l.Fields{"SignUpAccount": account})

	site := rhssoConfig.GetHost() + "/auth/realms/" + rhssoConfig.GetRealm()
	_, err = resources.CreateRHSSOClient(
		clientID,
		string(secret),
		account.AccountDetail.AdminBaseURL,
		serverClient,
		r.ConfigManager,
		ctx,
		*r.installation,
		r.log,
	)
	if err != nil {
		r.log.Error("failed to create RHSSO client", l.Fields{"tenant": tenantID, "clientID": clientID}, err)
		return fmt.Errorf("failed to create RHSSO client: %w", err)
	}
	authProviderDetails := AuthProviderDetails{
		Kind:                           "keycloak",
		Name:                           integration,
		ClientId:                       clientID,
		ClientSecret:                   string(secret),
		Site:                           site,
		SkipSSLCertificateVerification: true,
		Published:                      true, // This field does the test?
		SystemName:                     clientID,
	}
	r.log.Infof("auth provider", l.Fields{"authProviderDetails": authProviderDetails})

	err = r.tsClient.AddAuthProviderToAccount(account.AccountAccessToken.Value,
		account.AccountDetail, authProviderDetails,
	)
	if err != nil {
		r.log.Error("failed to add auth provider to tenant account", l.Fields{"tenant": tenantID, "authProviderDetails": authProviderDetails}, err)
		return fmt.Errorf("failed to add auth provider to account %w", err)
	}

	return nil
}

func getMTAccountsToBeCreated(usersIdentity []userHelper.MultiTenantUser, accounts []AccountDetail) (accountsToBeCreated []AccountDetail, emailAddrs []string, err error) {
	accountsToBeCreated = []AccountDetail{}
	email := ""
	for _, identity := range usersIdentity {
		foundAccount := false
		for _, account := range accounts {
			if account.OrgName == identity.TenantName {
				foundAccount = true
			}
		}
		if !foundAccount {
			accountsToBeCreated = append(accountsToBeCreated, AccountDetail{
				Name:    identity.TenantName,
				OrgName: identity.TenantName,
			})
			if identity.Email != "" {
				email = identity.Email
			} else {
				email, err = userHelper.SetUserNameAsEmail(identity.TenantName)
				if err != nil {
					return nil, nil, err
				}
			}
			emailAddrs = append(emailAddrs, email)
		}
	}
	return accountsToBeCreated, emailAddrs, nil
}

func getMTAccountsToBeDeleted(usersIdentity []userHelper.MultiTenantUser, accounts []AccountDetail) []AccountDetail {
	accountsToBeDeleted := []AccountDetail{}
	for _, account := range accounts {
		foundUser := false
		for _, identity := range usersIdentity {
			if account.OrgName == identity.TenantName {
				foundUser = true
			}
		}
		if !foundUser {
			accountsToBeDeleted = append(accountsToBeDeleted, account)
		}
	}
	return accountsToBeDeleted
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

func syncOpenshiftAdminMembership(openshiftAdminGroup *usersv1.Group, newTsUsers *Users, systemAdminUsername string, tsClient ThreeScaleInterface, accessToken string) error {
	for _, tsUser := range newTsUsers.Users {
		// skip if ts user is the system user admin
		if tsUser.UserDetails.Username == systemAdminUsername {
			continue
		}

		// In workshop mode, developer users also get admin permissions in 3scale
		if (userIsOpenshiftAdmin(tsUser, openshiftAdminGroup)) && tsUser.UserDetails.Role != adminRole {
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
		if err := r.ConfigManager.WriteConfig(r.Config); err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error writing threescale config : %w", err)
		}
	}

	if string(r.Config.GetOperatorVersion()) != string(integreatlyv1alpha1.OperatorVersion3Scale) {
		r.Config.SetOperatorVersion(string(integreatlyv1alpha1.OperatorVersion3Scale))
		if err := r.ConfigManager.WriteConfig(r.Config); err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error writing threescale config : %w", err)
		}
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
		r.log.Info("Failed to get oauth client secret:" + err.Error())
		return integreatlyv1alpha1.PhaseInProgress, err
	}

	if status != controllerutil.OperationResultNone {
		err = r.RolloutDeployment(ctx, serverClient, "system-app")
		if err != nil {
			r.log.Info("Failed to rollout deployment (system-app):" + err.Error())
			return integreatlyv1alpha1.PhaseInProgress, err
		}

		err = r.RolloutDeployment(ctx, serverClient, "system-sidekiq")
		if err != nil {
			r.log.Info("Failed to rollout deployment (system-sidekiq)" + err.Error())
			return integreatlyv1alpha1.PhaseInProgress, err
		}
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
	for i := range routes.Items {
		rt := routes.Items[i]
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
	return getToken(ctx, serverClient, r.Config.GetNamespace(), "ADMIN_ACCESS_TOKEN")
}

func (r *Reconciler) GetMasterToken(ctx context.Context, serverClient k8sclient.Client) (*string, error) {
	return getToken(ctx, serverClient, r.Config.GetNamespace(), "MASTER_ACCESS_TOKEN")
}

func getToken(ctx context.Context, serverClient k8sclient.Client, namespace, tokenType string) (*string, error) {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "system-seed",
		},
	}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: s.Name, Namespace: namespace}, s)
	if err != nil {
		return nil, err
	}

	accessToken := string(s.Data[tokenType])
	return &accessToken, nil
}

func (r *Reconciler) RolloutDeployment(ctx context.Context, client k8sclient.Client, name string) error {
	deployment := &appsv1.Deployment{}
	err := client.Get(ctx, apiMachineryTypes.NamespacedName{Namespace: r.Config.GetNamespace(), Name: name}, deployment)
	if err != nil {
		return err
	}

	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}

	deployment.Spec.Template.Annotations["kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	err = client.Update(ctx, deployment)
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	return err
}

func (r *Reconciler) getUserDiff(ctx context.Context, serverClient k8sclient.Client, kcUsers []keycloak.KeycloakAPIUser, tsUsers []*User) ([]keycloak.KeycloakAPIUser, []*User, []*User) {
	var added []keycloak.KeycloakAPIUser
	var deleted []*User
	var updated []*User

	rhssoConfig, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		r.log.Warning("Failed to get rhsso config: " + err.Error())
		return added, deleted, updated
	}

	for _, kcUser := range kcUsers {
		if !tsContainsKc(tsUsers, kcUser) {
			added = append(added, kcUser)
		}
	}

	var expectedDeleted []*User
	for _, tsUser := range tsUsers {
		if !kcContainsTs(kcUsers, tsUser) {
			expectedDeleted = append(expectedDeleted, tsUser)
		}
	}

	// compare the id fields in the generated user to that of the expected deleted user
	for _, user := range expectedDeleted {
		toDelete := true
		for _, kuUser := range kcUsers {
			genKcUser := &keycloak.KeycloakUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      userHelper.GetValidGeneratedUserName(kuUser),
					Namespace: rhssoConfig.GetNamespace(),
				},
			}
			if genKcUser.Name == "" {
				r.log.Warning("failed to get valid generated username")
				return added, deleted, updated
			}
			objectKey := k8sclient.ObjectKeyFromObject(genKcUser)
			err = serverClient.Get(ctx, objectKey, genKcUser)
			if err != nil {
				r.log.Warning("Failed get generated Keycloak User: " + err.Error())
				continue
			}

			if tsUserIDInKc(user, genKcUser) {
				updated = append(updated, user)
				toDelete = false
				break
			}
		}
		if toDelete {
			deleted = append(deleted, user)
		}
	}

	return added, deleted, updated
}

// getGeneratedKeycloakUser returns a keycloakUser CR for a matching 3scale user ID
func getGeneratedKeycloakUser(ctx context.Context, serverClient k8sclient.Client, ns string, tsUser *User) (*keycloak.KeycloakUser, error) {

	var users keycloak.KeycloakUserList

	listOptions := []k8sclient.ListOption{
		k8sclient.MatchingLabels(rhsso.GetInstanceLabels()),
		k8sclient.InNamespace(ns),
	}
	err := serverClient.List(ctx, &users, listOptions...)
	if err != nil {
		return nil, err
	}

	for i := range users.Items {
		kcUser := users.Items[i]
		if tsUserIDInKc(tsUser, &kcUser) {
			return &kcUser, nil
		}
	}

	return nil, fmt.Errorf("genrated Keycloak user was not found")
}

// tsUserIDInKc checks if a 3scale user ID is listed in the keycloak user attributes
func tsUserIDInKc(tsUser *User, kcUser *keycloak.KeycloakUser) bool {
	if len(kcUser.Spec.User.Attributes[user3ScaleID]) == 0 {
		return false
	}

	if strings.EqualFold(fmt.Sprint(tsUser.UserDetails.Id), kcUser.Spec.User.Attributes[user3ScaleID][0]) {
		return true
	}
	return false
}

func kcContainsTs(kcUsers []keycloak.KeycloakAPIUser, tsUser *User) bool {
	for _, kcu := range kcUsers {
		if strings.ToLower(kcu.UserName) == tsUser.UserDetails.Username {
			return true
		}
	}

	return false
}

func tsContainsKc(tsusers []*User, kcUser keycloak.KeycloakAPIUser) bool {
	for _, tsu := range tsusers {
		if tsu.UserDetails.Username == strings.ToLower(kcUser.UserName) {
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
	fullScopeAllowed := true

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
			FullScopeAllowed:    &fullScopeAllowed,
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

	r.log.Info("reconciling edit routes role to the dedicated admins group")

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
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed reconciling edit routes role %v: %w", editRoutesRole, err)
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
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed reconciling service admin role binding %v: %w", editRoutesRoleBinding, err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileSubscription(ctx context.Context, serverClient k8sclient.Client, rhmi *integreatlyv1alpha1.RHMI, productNamespace string, operatorNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	target := marketplace.Target{
		SubscriptionName: constants.ThreeScaleSubscriptionName,
		Namespace:        operatorNamespace,
	}

	catalogSourceReconciler, err := r.GetProductDeclaration().PrepareTarget(
		r.log,
		serverClient,
		marketplace.CatalogSourceName,
		&target,
	)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(rhmi.Spec.Type)) {
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
	return r.Reconciler.ReconcileSubscription(
		ctx,
		target,
		[]string{},
		r.preUpgradeBackupExecutor(),
		serverClient,
		catalogSourceReconciler,
		r.log,
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

// Deployments are rescaled when adding topologySpreadConstraints, PodTopology etc
// Should check that these deployments are ready before returning phase complete in CR
func (r *Reconciler) ensureDeploymentsReady(ctx context.Context, serverClient k8sclient.Client, productNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	for _, name := range threeScaleDeployments {
		deployment := &appsv1.Deployment{}

		err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: name, Namespace: productNamespace}, deployment)

		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		// Rollout new deployment if there is a failed condition
		for _, condition := range deployment.Status.Conditions {
			if condition.Status == corev1.ConditionFalse {
				r.log.Warningf("3scale deployment in a failed condition, rolling out new deployment", l.Fields{"deployment": name})
				err = r.RolloutDeployment(ctx, serverClient, name)
				if err != nil {
					return integreatlyv1alpha1.PhaseFailed, err
				}

				return integreatlyv1alpha1.PhaseCreatingComponents, nil
			}
		}

		//  Check that replicas are fully rolled out
		for _, condition := range deployment.Status.Conditions {
			if condition.Status != corev1.ConditionTrue || (deployment.Status.Replicas != deployment.Status.AvailableReplicas ||
				deployment.Status.ReadyReplicas != deployment.Status.UpdatedReplicas) {
				r.log.Infof("waiting for 3scale deployment to become ready", l.Fields{"deployment": name})
				return integreatlyv1alpha1.PhaseInProgress, fmt.Errorf("waiting for 3scale deployments %s to become available", name)
			}
		}
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileRatelimitingTo3scaleComponents(ctx context.Context, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {

	r.log.Info("Reconciling rate limiting settings to 3scale components")

	proxyServer := ratelimit.NewEnvoyProxyServer(ctx, serverClient, r.log)

	err := r.createBackendListenerProxyService(ctx, serverClient)
	if err != nil {
		return integreatlyv1alpha1.PhaseInProgress, err
	}

	// creates envoy proxy sidecar container for apicast staging
	phase, err := proxyServer.CreateEnvoyProxyContainer(
		apicastStagingDeploymentName,
		r.Config.GetNamespace(),
		ApicastNodeID,
		apicastStagingDeploymentName,
		"gateway",
		ApicastEnvoyProxyPort,
	)
	if phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, err
	}

	// creates envoy proxy sidecar container for apicast production
	phase, err = proxyServer.CreateEnvoyProxyContainer(
		apicastProductionDeploymentName,
		r.Config.GetNamespace(),
		ApicastNodeID,
		apicastProductionDeploymentName,
		"gateway",
		ApicastEnvoyProxyPort,
	)
	if phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, err
	}

	// creates envoy proxy sidecar container for backend listener
	phase, err = proxyServer.CreateEnvoyProxyContainer(
		backendListenerDeploymentName,
		r.Config.GetNamespace(),
		BackendNodeID,
		BackendServiceName,
		"http",
		BackendEnvoyProxyPort,
	)
	if phase != integreatlyv1alpha1.PhaseCompleted {
		return phase, err
	}

	r.log.Info("Finished creating envoy sidecar containers for 3scale components")

	// setting up envoy config
	ratelimitServiceCR, err := r.getRateLimitServiceCR(ctx, serverClient)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// rate limit cluster
	ratelimitClusterResource := ratelimit.CreateClusterResource(
		ratelimitServiceCR.Spec.ClusterIP,
		ratelimit.RateLimitClusterName,
		getRatelimitServicePort(ratelimitServiceCR),
	)

	extensionProtocol, err := ratelimit.CreateTypedExtensionProtocol()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	ratelimitClusterResource.TypedExtensionProtocolOptions = map[string]*any.Any{
		"envoy.extensions.upstreams.http.v3.HttpProtocolOptions": extensionProtocol,
	}

	// apicast cluster
	apiCastClusterResource := ratelimit.CreateClusterResource(
		ApicastContainerAddress,
		ApicastClusterName,
		ApicastContainerPort,
	)

	// http2 upstream
	apiCastClusterResource.TypedExtensionProtocolOptions = map[string]*any.Any{
		"envoy.extensions.upstreams.http.v3.HttpProtocolOptions": extensionProtocol,
	}

	// transport socket configuration
	apicastTLSContext, err := ratelimit.CreateApicastTransportSocketConfig()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	apiCastClusterResource.TransportSocket = &corev3.TransportSocket{
		Name: ratelimit.TransportSocketName,
		ConfigType: &corev3.TransportSocket_TypedConfig{
			TypedConfig: apicastTLSContext,
		},
	}

	var apicastHTTPFilters []*hcm.HttpFilter
	// apicast filters based on installation type
	if !integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(r.installation.Spec.Type)) {
		apicastHTTPFilters, err = getAPICastHTTPFilters()
		if err != nil {
			r.log.Error("Failed to create envoyconfig filters for multitenant RHOAM", l.Fields{"APICast": ApicastClusterName}, err)
			return integreatlyv1alpha1.PhaseFailed, err
		}
	} else {
		apicastHTTPFilters, err = getMultitenantAPICastHTTPFilters()
		if err != nil {
			r.log.Error("Failed to create envoyconfig filters for multitenant RHOAM", l.Fields{"APICast": ApicastClusterName}, err)
			return integreatlyv1alpha1.PhaseFailed, err
		}
	}

	// apicast listener
	apiCastFilters, err := getListenerResourceFilters(
		getAPICastVirtualHosts(installation, ApicastClusterName),
		apicastHTTPFilters,
	)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	apiCastListenerResource := ratelimit.CreateListenerResource(
		ApicastListenerName,
		ApicastEnvoyProxyAddress,
		ApicastEnvoyProxyPort,
		apiCastFilters,
	)

	// apicast runtime
	apiCastRuntimes := ratelimit.CreateRuntimesResource()

	// create envoy config for apicast
	apiCastProxyConfig := ratelimit.NewEnvoyConfig(ApicastClusterName, r.Config.GetNamespace(), ApicastNodeID)
	err = apiCastProxyConfig.CreateEnvoyConfig(ctx, serverClient, []*envoyclusterv3.Cluster{apiCastClusterResource, ratelimitClusterResource}, []*envoylistenerv3.Listener{apiCastListenerResource}, apiCastRuntimes, installation)
	if err != nil {
		r.log.Error("Failed to create envoyconfig for apicast", l.Fields{"APICast": ApicastClusterName}, err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// backend-listener cluster
	backendClusterResource := ratelimit.CreateClusterResource(
		BackendContainerAddress,
		BackendClusterName,
		BackendContainerPort,
	)

	backendHTTPFilters, err := getBackendListenerHTTPFilters()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	// backend listener listener
	backendFilters, err := getListenerResourceFilters(
		getBackendListenerVitualHosts(BackendClusterName),
		backendHTTPFilters,
	)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	backendListenerResource := ratelimit.CreateListenerResource(
		BackendListenerName,
		BackendEnvoyProxyAddress,
		BackendEnvoyProxyPort,
		backendFilters,
	)

	// backend runtimes
	backendRuntimes := ratelimit.CreateRuntimesResource()

	// create envoy config for backend listener
	backendProxyConfig := ratelimit.NewEnvoyConfig(BackendClusterName, r.Config.GetNamespace(), BackendNodeID)
	err = backendProxyConfig.CreateEnvoyConfig(ctx, serverClient, []*envoyclusterv3.Cluster{backendClusterResource, ratelimitClusterResource}, []*envoylistenerv3.Listener{backendListenerResource}, backendRuntimes, installation)
	if err != nil {
		r.log.Error("Failed to create envoyconfig for backend-listener", l.Fields{"BackendListener": BackendClusterName}, err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) getRateLimitServiceCR(ctx context.Context, serverClient k8sclient.Client) (*corev1.Service, error) {
	rateLimitService := &corev1.Service{}
	marin3rConfig, err := r.ConfigManager.ReadMarin3r()
	if err != nil {
		return nil, fmt.Errorf("failed to load marin3r config in 3scale reconciler: %v", err)
	}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{
		Namespace: marin3rConfig.GetNamespace(),
		Name:      "ratelimit",
	}, rateLimitService)

	if err != nil {
		return nil, fmt.Errorf("failed to rate limiting service: %v", err)
	}

	return rateLimitService, nil
}

func getRatelimitServicePort(rateLimitService *corev1.Service) int {
	for _, port := range rateLimitService.Spec.Ports {
		if port.Name == "grpc" {
			return port.TargetPort.IntValue()
		}
	}
	return 0
}

func (r *Reconciler) createBackendListenerProxyService(ctx context.Context, serverClient k8sclient.Client) error {

	backendListenerService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BackendServiceName,
			Namespace: r.Config.GetNamespace(),
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, backendListenerService, func() error {
		owner.AddIntegreatlyOwnerAnnotations(backendListenerService, r.installation)
		backendListenerService.Spec.Ports = []corev1.ServicePort{
			{
				Name:       "http",
				Protocol:   "TCP",
				Port:       BackendEnvoyProxyPort,
				TargetPort: intstr.FromInt(BackendEnvoyProxyPort),
			},
		}
		backendListenerService.Spec.Selector = map[string]string{
			"deployment": backendListenerDeploymentName,
		}
		return nil
	}); err != nil {
		return err
	}

	// links the backend listener proxy service to the external backend listener route
	backendRoute, err := r.getBackendListenerRoute(ctx, serverClient)
	if err != nil {
		return err
	}

	backendRoute.Spec.To.Name = backendListenerService.Name
	err = serverClient.Update(ctx, backendRoute)
	if err != nil {
		return fmt.Errorf("error updating the backend-listener external route to the backend-listener proxy server: %v", err)
	}

	r.log.Infof("Created service to rate limit external backend-listener route",
		l.Fields{"ServiceName": backendListenerService.Name},
	)
	return nil
}

func (r *Reconciler) getBackendListenerRoute(ctx context.Context, serverClient k8sclient.Client) (*routev1.Route, error) {
	backendRoute := &routev1.Route{}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{
		Namespace: r.Config.GetNamespace(),
		Name:      "backend",
	}, backendRoute)
	if err != nil {
		return nil, fmt.Errorf("error getting the backend-listener external route: %v", err)
	}
	return backendRoute, nil
}

func (r *Reconciler) addSSOReadyAnnotationToUser(_ context.Context, client k8sclient.Client, name string) error {
	// Get the User CR to annotate
	userToAnnotate := &usersv1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	key := k8sclient.ObjectKeyFromObject(userToAnnotate)
	err := client.Get(context.TODO(), key, userToAnnotate)
	if err != nil {
		return fmt.Errorf("error getting user %s: %v", name, err)
	}

	// Add the annotation `ssoReady: 'yes'` to the User CR
	_, err = controllerutil.CreateOrUpdate(context.TODO(), client, userToAnnotate, func() error {
		if userToAnnotate.Annotations == nil {
			userToAnnotate.Annotations = map[string]string{}
		}
		userToAnnotate.Annotations["ssoReady"] = "yes"
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to add ssoReady annotation to user %s: %v", userToAnnotate.Name, err)
	}

	return nil
}

func (r *Reconciler) reconcileRatelimitPortAnnotation(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	apim := &threescalev1.APIManager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiManagerName,
			Namespace: r.Config.GetNamespace(),
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, client, apim, func() error {
		annotations := apim.ObjectMeta.GetAnnotations()
		if annotations == nil {
			annotations = map[string]string{}
		}
		annotations["apps.3scale.net/disable-apicast-service-reconciler"] = "true"
		return nil
	}); err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) findCustomDomainCr(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, string, error) {
	ok, customDomainName, err := customDomain.HasValidCustomDomainCR(ctx, serverClient, r.installation.Spec.RoutingSubdomain)
	if ok {
		return integreatlyv1alpha1.PhaseCompleted, customDomainName, nil
	}
	return integreatlyv1alpha1.PhaseFailed, customDomainName, fmt.Errorf("finding CustomDomain CR failed: %v", err)
}

func (r *Reconciler) findIngressControllerCr(ctx context.Context, customDomainName string, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	ok, err := customDomain.HasValidIngressControllerCR(ctx, serverClient, customDomainName, r.installation.Spec.RoutingSubdomain)
	if ok {
		return integreatlyv1alpha1.PhaseCompleted, nil
	}
	return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("finding IngressController CR failed: %v", err)
}

func (r *Reconciler) ping3scalePortals(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	format := "failed to retrieve %s 3scale route"
	portals := map[string]metrics.PortalInfo{}

	accessToken, err := r.GetMasterToken(ctx, serverClient)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	accounts, err := r.tsClient.ListTenantAccounts(*accessToken, 1, nil)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	filter := NewTenantAccountsFilter(accounts)

	systemMasterRoute, err := r.getThreescaleRoute(ctx, serverClient, labelRouteToSystemMaster, nil)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf(format, labelRouteToSystemMaster)
	}
	systemDeveloperRoute, err := r.getThreescaleRoute(ctx, serverClient, labelRouteToSystemDeveloper, filter.Developer)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf(format, labelRouteToSystemDeveloper)
	}
	systemProviderRoute, err := r.getThreescaleRoute(ctx, serverClient, labelRouteToSystemProvider, filter.Provider)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf(format, labelRouteToSystemProvider)
	}

	if systemMasterRoute != nil && systemMasterRoute.Status.Ingress != nil && len(systemMasterRoute.Status.Ingress) > 0 {
		portals[metrics.LabelSystemMaster] = metrics.PortalInfo{
			Host:       systemMasterRoute.Status.Ingress[0].Host,
			PortalName: labelRouteToSystemMaster,
			Ingress:    strings.Split(systemMasterRoute.Status.Ingress[0].RouterCanonicalHostname, ".")[0],
		}
	}
	if systemDeveloperRoute != nil && systemDeveloperRoute.Status.Ingress != nil && len(systemDeveloperRoute.Status.Ingress) > 0 {
		portals[metrics.LabelSystemDeveloper] = metrics.PortalInfo{
			Host:       systemDeveloperRoute.Status.Ingress[0].Host,
			PortalName: labelRouteToSystemDeveloper,
			Ingress:    strings.Split(systemDeveloperRoute.Status.Ingress[0].RouterCanonicalHostname, ".")[0],
		}
	}
	if systemProviderRoute != nil && systemProviderRoute.Status.Ingress != nil && len(systemProviderRoute.Status.Ingress) > 0 {
		portals[metrics.LabelSystemProvider] = metrics.PortalInfo{
			Host:       systemProviderRoute.Status.Ingress[0].Host,
			PortalName: labelRouteToSystemProvider,
			Ingress:    strings.Split(systemProviderRoute.Status.Ingress[0].RouterCanonicalHostname, ".")[0],
		}
	}

	var hasUnavailablePortal float64

	for key, portal := range portals {

		// GETIP for portal
		// we GET IP for portal here because 3scales routes are not always exposed on ingress router-default,
		// Example for such case is Jira: https://issues.redhat.com/browse/OHSS-24580
		service, err := customDomain.GetIngressRouterService(ctx, serverClient, portal.Ingress)
		if err != nil || len(service.Status.LoadBalancer.Ingress) == 0 {
			errorMessage := "failed to retrieve ingress router service"
			r.log.Error("msg", nil, errors.New(errorMessage))
			phase := integreatlyv1alpha1.PhaseFailed
			events.HandleError(r.recorder, r.installation, phase, errorMessage, err)
			return phase, fmt.Errorf("%s: %v", errorMessage, err)
		}

		ips, err := customDomain.GetIngressRouterIPs(service.Status.LoadBalancer.Ingress)
		if err != nil {
			errorMessage := "failed to retrieve ingress router ips"
			r.log.Error("msg", nil, errors.New(errorMessage))
			phase := integreatlyv1alpha1.PhaseFailed
			events.HandleError(r.recorder, r.installation, phase, errorMessage, err)
			return phase, fmt.Errorf("%s: %v", errorMessage, err)
		}

		// #nosec G402 -- intentionally allowed
		customHTTPClient := &http.Client{
			Transport: &http.Transport{
				DisableKeepAlives: true,
				IdleConnTimeout:   10 * time.Second,
				TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					addressFormat := "%s:443"
					if addr == fmt.Sprintf(addressFormat, portal.Host) {
						addr = fmt.Sprintf(addressFormat, ips[0].String())
					}
					dialer := &net.Dialer{Timeout: 10 * time.Second}
					return dialer.DialContext(ctx, network, addr)
				},
			},
			Timeout: 10 * time.Second,
		}
		url := fmt.Sprintf("https://%s", portal.Host)
		res, err := customHTTPClient.Get(url)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to ping %v 3scale portal (%v): %v", portal.PortalName, portal.Host, err)
		}

		r.log.Infof("Got initial status code", l.Fields{"url": url, "code": res.StatusCode})

		ok := true
		statusCode := http.StatusOK
		if res.StatusCode != http.StatusOK {
			ok, statusCode = checkRedirects(portal.Host, "/p/login", res, http.StatusFound)
			if !ok {
				hasUnavailablePortal = 1
			}
		}
		portal.IsAvailable = ok
		portals[key] = portal
		r.log.Infof("pinged 3scale portal", map[string]interface{}{
			"Portal":           portal.PortalName,
			"Host":             portal.Host,
			"Status Code":      statusCode,
			"Portal available": portal.IsAvailable,
		})
	}
	metrics.SetThreeScalePortals(portals, hasUnavailablePortal)
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func checkRedirects(host string, path string, res *http.Response, statusCode int) (bool, int) {

	if res == nil {
		return false, 000
	}

	if res.StatusCode == statusCode {
		if res.Request.URL.Host == host && res.Request.URL.Path == path {
			return true, res.StatusCode
		}
	}

	if res.Request.Response != nil {
		return checkRedirects(host, path, res.Request.Response, statusCode)
	}

	return false, 000
}

func (r *Reconciler) getKeycloakUserFromAccount(client k8sclient.Client, accountName string) (*keycloak.KeycloakUser, error) {
	kcUserList := &keycloak.KeycloakUserList{}
	if err := client.List(context.TODO(), kcUserList, k8sclient.InNamespace(fmt.Sprintf("%srhsso", r.installation.Spec.NamespacePrefix))); err != nil {
		return nil, fmt.Errorf("failed to get list of KeycloakUsers, err: %v", err)
	}
	for _, kcUser := range kcUserList.Items {
		if kcUser.Spec.User.UserName == accountName {
			return &kcUser, nil
		}
	}

	// If the KeycloakUser wasn't found return an error
	return nil, fmt.Errorf("failed to find the KeycloakUser for %v", accountName)
}

func (r *Reconciler) getKeycloakClientFromAccount(client k8sclient.Client, accountName string) (*keycloak.KeycloakClient, error) {
	kcClientList := &keycloak.KeycloakClientList{}
	if err := client.List(context.TODO(), kcClientList, k8sclient.InNamespace(fmt.Sprintf("%srhsso", r.installation.Spec.NamespacePrefix))); err != nil {
		return nil, fmt.Errorf("failed to get list of KeycloakClients, err: %v", err)
	}
	for _, kcClient := range kcClientList.Items {
		if strings.Contains(kcClient.Name, accountName) {
			return &kcClient, nil
		}
	}

	// If the KeycloakClient wasn't found return an error
	return nil, fmt.Errorf("failed to find the KeycloakClient for %v", accountName)
}

func (r *Reconciler) reconcileServiceMonitor(ctx context.Context, client k8sclient.Client, namespace string) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Start reconcileServiceMonitor for 3scale")

	serviceMonitor := &prometheus.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "3scale-service-monitor",
			Namespace: namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, serviceMonitor, func() error {
		serviceMonitor.Labels = map[string]string{
			"monitoring-key": "middleware",
		}
		serviceMonitor.Spec = prometheus.ServiceMonitorSpec{
			Endpoints: []prometheus.Endpoint{
				{
					Path: "metrics",
					Port: "metrics",
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "3scale-api-management",
				},
			},
		}
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileDeploymentEnvarEmailAddress(ctx context.Context, serverClient k8sclient.Client, deploymentName string, updateFn func(deployment *appsv1.Deployment, value string) bool) (integreatlyv1alpha1.StatusPhase, error) {
	existingSMTPFromAddress, err := resources.GetSMTPFromAddress(ctx, serverClient, r.log, r.installation)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	deployment := &appsv1.Deployment{}

	err = serverClient.Get(ctx, apiMachineryTypes.NamespacedName{Namespace: r.Config.GetNamespace(), Name: deploymentName}, deployment)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	updated := updateFn(deployment, existingSMTPFromAddress)

	if updated {
		err = serverClient.Update(ctx, deployment)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}
		err = r.RolloutDeployment(ctx, serverClient, deploymentName)
		if err != nil {
			r.log.Error(fmt.Sprintf("Rollout %v deployment", deploymentName), nil, err)
			return integreatlyv1alpha1.PhaseFailed, err
		}

	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) syncInvitationEmail(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	fromAddress, err := resources.GetSMTPFromAddress(ctx, serverClient, r.log, r.installation)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	ns := r.Config.GetNamespace()
	podname := ""

	pods := &corev1.PodList{}
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(ns),
		k8sclient.MatchingLabels(map[string]string{"deployment": systemAppDeploymentName}),
	}
	err = serverClient.List(ctx, pods, listOpts...)
	if err != nil {
		r.log.Error("Error getting list of pods", nil, err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			podname = pod.ObjectMeta.Name
			break
		}
	}

	if podname == "" {
		r.log.Info("Waiting on system-app pod to start, 3Scale install in progress")
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	stdout, stderr, err := r.podExecutor.ExecuteRemoteContainerCommand(ns, podname, "system-master", []string{"/bin/bash",
		"-c", fmt.Sprintf("bundle exec rails runner \"a=Account.find_by_master true; a.from_email = '%v'; a.save;\"", fromAddress)})
	if err != nil {
		r.log.Error("Failed to set 3Scale invitation email", nil, err)
		return integreatlyv1alpha1.PhaseFailed, nil
	} else if stderr != "" {
		err := errors.New(stderr)
		r.log.Error("Error attempting to set 3Scale invitation email", nil, err)
		return integreatlyv1alpha1.PhaseFailed, err
	} else {
		r.log.Infof("Set 3Scale invitation email", l.Fields{"stdout": stdout})
		return integreatlyv1alpha1.PhaseCompleted, nil
	}
}

func updateContainerSupportEmail(deployment *appsv1.Deployment, existingSMTPFromAddress string, envar string) bool {
	updated := false
	for index, container := range deployment.Spec.Template.Spec.Containers {
		found := false
		for i, envVar := range container.Env {
			if envVar.Name == envar {
				found = true
				if envVar.Value != existingSMTPFromAddress {
					deployment.Spec.Template.Spec.Containers[index].Env[i].Value = existingSMTPFromAddress
					updated = true
				}
			}
		}
		if !found {

			deployment.Spec.Template.Spec.Containers[index].Env = append(deployment.Spec.Template.Spec.Containers[index].Env, corev1.EnvVar{
				Name:  envar,
				Value: existingSMTPFromAddress,
			})
			updated = true
		}
	}
	return updated
}

func updateSystemAppAddresses(deployment *appsv1.Deployment, value string) bool {
	return updateContainerSupportEmail(deployment, value, "SUPPORT_EMAIL")

}

func updateSystemSidekiqAddresses(deployment *appsv1.Deployment, value string) bool {
	support := updateContainerSupportEmail(deployment, value, "SUPPORT_EMAIL")
	notification := updateContainerSupportEmail(deployment, value, "NOTIFICATION_EMAIL")

	if support || notification {
		return true
	}
	return false
}

func (r *Reconciler) reconcileTenantOutgoingEmailAddress(ctx context.Context, serverClient k8sclient.Client, address string) error {

	masterAccessToken, err := r.GetMasterToken(ctx, serverClient)
	if err != nil {
		return err
	}

	routes := &routev1.RouteList{}
	admRoute := routev1.Route{}
	found := false
	err = serverClient.List(ctx, routes, &k8sclient.ListOptions{
		Namespace: r.Config.GetNamespace(),
	})
	if err != nil {
		return fmt.Errorf("failed to get 3scale route list during portaClient creation, error: %v", err)
	}
	for _, route := range routes.Items {
		routeLabels := route.GetLabels()
		value, exists := routeLabels["zync.3scale.net/route-to"]
		if exists && value == "system-master" {
			admRoute = route
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("failed to get 3scale route during portaClient creation")
	}

	adminPortal, err := portaClient.NewAdminPortal("https", admRoute.Spec.Host, 443)
	if err != nil {
		return fmt.Errorf("could not create admin portal during portaClient creation, error: %v", err)
	}

	/* #nosec */
	httpc := &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			IdleConnTimeout:   time.Second * 10,
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: r.installation.Spec.SelfSignedCerts}, //#nosec G402 -- value is read from CR config
		},
	}

	pc := portaClient.NewThreeScale(adminPortal, *masterAccessToken, httpc)
	var accountList []AccountDetail
	for page := 1; page <= 100; page++ {
		accounts, err := r.tsClient.ListTenantAccounts(*masterAccessToken, page, func(ac AccountDetail) bool {
			return ac.Id != 1 && ac.Id != 2
		})
		if err != nil {
			return err
		}
		if accounts == nil {
			break
		}
		accountList = append(accountList, accounts...)
	}

	for _, account := range accountList {
		err = r.tsClient.UpdateTenant(int64(account.Id), portaClient.Params{"from_email": address}, pc)
		if err != nil {
			return err
		}

	}
	return nil
}
