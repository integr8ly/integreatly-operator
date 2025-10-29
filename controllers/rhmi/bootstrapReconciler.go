package controllers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/integr8ly/integreatly-operator/pkg/resources/cluster"
	customDomain "github.com/integr8ly/integreatly-operator/pkg/resources/custom-domain"
	userHelper "github.com/integr8ly/integreatly-operator/pkg/resources/user"
	"github.com/integr8ly/integreatly-operator/utils"
	configv1 "github.com/openshift/api/config/v1"

	"github.com/integr8ly/integreatly-operator/pkg/addon"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pkg/errors"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/metrics"
	"github.com/integr8ly/integreatly-operator/pkg/products/obo"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"

	cs "github.com/integr8ly/integreatly-operator/pkg/resources/custom-smtp"

	oauthv1 "github.com/openshift/api/oauth/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	marin3rconfig "github.com/integr8ly/integreatly-operator/pkg/products/marin3r/config"
)

var tenantOauthclientSecretsName = "tenant-oauth-client-secrets" // #nosec G101 -- This is a false positive

func NewBootstrapReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, logger l.Logger) (*Reconciler, error) {
	return &Reconciler{
		ConfigManager: configManager,
		mpm:           mpm,
		installation:  installation,
		Reconciler:    resources.NewReconciler(mpm),
		recorder:      recorder,
		log:           logger,
	}, nil
}

type Reconciler struct {
	ConfigManager config.ConfigReadWriter
	Config        *config.ThreeScale
	mpm           marketplace.MarketplaceInterface
	installation  *integreatlyv1alpha1.RHMI
	*resources.Reconciler
	recorder record.EventRecorder
	log      l.Logger
}

func (r *Reconciler) GetPreflightObject(_ string) k8sclient.Object {
	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client, installationQuota *quota.Quota, request ctrl.Request) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Reconciling bootstrap stage")

	phase, err := resources.ReconcileLimitRange(ctx, serverClient, r.installation.Namespace, resources.DefaultLimitRangeParams)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile LimitRange for Namespace %s", r.installation.Namespace), err)
		return phase, err
	}

	phase, err = r.reconcileOauthSecrets(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile oauth secrets", err)
		return phase, errors.Wrap(err, "failed to reconcile oauth secrets")
	}

	if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installation.Spec.Type)) {
		phase, err := r.reconcileTenantOauthSecrets(ctx, serverClient)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			events.HandleError(r.recorder, installation, phase, "Failed to reconcile tenant oauth secrets", err)
			return phase, errors.Wrap(err, "failed to reconcile tenant oauth secrets")
		}
		err = r.setTenantMetrics(ctx, serverClient)
		if err != nil {
			events.HandleError(r.recorder, installation, phase, "Error setting tenant metrics", err)
			return phase, errors.Wrap(err, "failed to set tenant metrics")
		}
	}

	phase, err = r.reconcilerGithubOauthSecret(ctx, serverClient, installation)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile github oauth secrets", err)
		return phase, errors.Wrap(err, "failed to reconcile github oauth secrets")
	}

	phase, err = r.reconcileAddonManagedApiServiceParameters(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile addon parameters", err)
		return phase, errors.Wrap(err, "failed to reconcile addon parameters secret")
	}

	phase, err = r.retrieveConsoleURLAndSubdomain(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to retrieve console url and subdomain", err)
		return phase, errors.Wrap(err, "failed to retrieve console url and subdomain")
	}

	phase, err = r.retrieveAPIServerURL(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to retrieve API server URL", err)
		return phase, errors.Wrap(err, "failed to retrieve API server URL")
	}

	phase, err = r.checkCloudResourcesConfig(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to check cloud resources config settings", err)
		return phase, errors.Wrap(err, "failed to check cloud resources config settings")
	}

	phase, err = r.reconcilePriorityClass(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile priority class", err)
		return phase, errors.Wrap(err, "failed to reconcile priority class")
	}

	phase, err = r.checkRateLimitAlertsConfig(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to check rate limit alert config settings", err)
		return phase, errors.Wrap(err, "failed to check rate limit alert config settings")
	}

	if err = r.processQuota(installation, request.Namespace, installationQuota, serverClient); err != nil {
		events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, "Error while processing the Quota", err)
		installation.Status.LastError = err.Error()
		return integreatlyv1alpha1.PhaseFailed, err
	}
	metrics.SetQuota(installation.Status.Quota, installation.Status.ToQuota)

	phase, err = r.reconcileCustomSMTP(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Reconciling custom SMTP has failed ", err)
		return phase, errors.Wrap(err, "reconciling custom SMTP has failed ")
	}

	if !resources.IsInProw(installation) {
		// Creates the Alertmanager config secret
		phase, err = obo.ReconcileAlertManagerSecrets(ctx, serverClient, r.installation)
		r.log.Infof("ReconcileAlertManagerConfigSecret", l.Fields{"phase": phase})
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			if err != nil {
				r.log.Warning("failed to reconcile alert manager config secret " + err.Error())
			}
			events.HandleError(r.recorder, installation, phase, "Failed to reconcile alert manager config secret", err)
			return phase, err
		}

		// Creates an alert to check for the presence of sendgrid smtp secret
		phase, err = resources.CreateSmtpSecretExists(ctx, serverClient, installation)
		r.log.Infof("Reconcile SendgridSmtpSecretExists alert", l.Fields{"phase": phase})
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			events.HandleError(r.recorder, installation, phase, "Failed to reconcile SendgridSmtpSecretExists alert", err)
			return phase, err
		}

		// Creates an alert to check for the presence of DeadMansSnitch secret
		phase, err = resources.CreateDeadMansSnitchSecretExists(ctx, serverClient, installation)
		r.log.Infof("Reconcile DeadMansSnitchSecretExists alert", l.Fields{"phase": phase})
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			events.HandleError(r.recorder, installation, phase, "Failed to reconcile DeadMansSnitchSecretExists alert", err)
			return phase, err
		}

		// Creates an alert to check for the presence of addon-managed-api-service-parameters secret
		phase, err = resources.CreateAddonManagedApiServiceParametersExists(ctx, serverClient, installation)
		r.log.Infof("Reconcile AddonManagedApiServiceParametersExists alert", l.Fields{"phase": phase})
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			events.HandleError(r.recorder, installation, phase, "Failed to reconcile AddonManagedApiServiceParametersExists alert", err)
			return phase, err
		}

		// Creates remaining OBO alerts
		phase, err = obo.OboAlertsReconciler(r.log, r.installation).ReconcileAlerts(ctx, serverClient)
		r.log.Infof("Reconcile OBO alerts", l.Fields{"phase": phase})
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			events.HandleError(r.recorder, installation, phase, "Failed to reconcile OBO alerts", err)
			return phase, err
		}

	}

	events.HandleStageComplete(r.recorder, installation, integreatlyv1alpha1.BootstrapStage)

	metrics.SetInfo(installation)
	r.log.Info("Metric rhmi_info exposed")

	r.log.Info("Bootstrap stage reconciled successfully")
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) setTenantMetrics(ctx context.Context, serverClient k8sclient.Client) error {
	tenants := &integreatlyv1alpha1.APIManagementTenantList{}
	err := serverClient.List(ctx, tenants)
	if err != nil {
		return err
	}

	r.log.Info("Setting tenant metrics")
	metrics.SetTenantsSummary(tenants)
	return nil
}

func (r *Reconciler) reconcilePriorityClass(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	priorityClass := &schedulingv1.PriorityClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.installation.Spec.PriorityClassName,
		},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, priorityClass, func() error {
		if integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(r.installation.Spec.Type)) {
			priorityClass.Value = 0
		} else {
			priorityClass.Value = 1000000000
		}
		priorityClass.GlobalDefault = false
		priorityClass.Description = "Priority Class for managed-api"

		return nil
	}); err != nil {
		return integreatlyv1alpha1.PhaseInProgress, err
	}
	return integreatlyv1alpha1.PhaseCompleted, nil

}

func (r *Reconciler) checkCloudResourcesConfig(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	cloudConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultCloudResourceConfigName,
			Namespace: r.installation.Namespace,
		},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, cloudConfig, func() error {
		if cloudConfig.Data == nil {
			cloudConfig.Data = map[string]string{}
		}

		if resources.Contains(cloudConfig.Finalizers, previousDeletionFinalizer) {
			resources.Replace(cloudConfig.Finalizers, previousDeletionFinalizer, deletionFinalizer)
		}

		if strings.ToLower(r.installation.Spec.UseClusterStorage) == "true" {
			cloudConfig.Data["managed-api"] = `{"blobstorage":"openshift", "smtpcredentials":"openshift", "redis":"openshift", "postgres":"openshift"}`
			cloudConfig.Data["multitenant-managed-api"] = `{"blobstorage":"openshift", "smtpcredentials":"openshift", "redis":"openshift", "postgres":"openshift"}`
		} else {
			platformType, err := cluster.GetPlatformType(ctx, serverClient)
			if err != nil {
				return errors.Wrap(err, "failed to retrieve platform type")
			}
			switch platformType {
			case configv1.AWSPlatformType:
				cloudConfig.Data["managed-api"] = `{"blobstorage":"aws", "smtpcredentials":"aws", "redis":"aws", "postgres":"aws"}`
				cloudConfig.Data["multitenant-managed-api"] = `{"blobstorage":"aws", "smtpcredentials":"aws", "redis":"aws", "postgres":"aws"}`
			}
		}
		cloudConfig.Data["workshop"] = `{"blobstorage":"openshift", "smtpcredentials":"openshift", "redis":"openshift", "postgres":"openshift"}`
		return nil
	}); err != nil {
		return integreatlyv1alpha1.PhaseInProgress, err
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) checkRateLimitAlertsConfig(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	alertsConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      marin3rconfig.AlertConfigMapName,
			Namespace: r.installation.Namespace,
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, alertsConfig, func() error {
		owner.AddIntegreatlyOwnerAnnotations(alertsConfig, r.installation)

		if alertsConfig.Data == nil {
			alertsConfig.Data = map[string]string{}
		}

		if _, ok := alertsConfig.Data["alerts"]; ok {
			return nil
		}

		maxRate1 := "90%"
		maxRate2 := "95%"

		defaultConfig := map[string]*marin3rconfig.AlertConfig{
			"api-usage-alert-level1": {
				Type:     marin3rconfig.AlertTypeThreshold,
				RuleName: "RHOAMApiUsageLevel1ThresholdExceeded",
				Level:    "info",
				Threshold: &marin3rconfig.AlertThresholdConfig{
					MinRate: "80%",
					MaxRate: &maxRate1,
				},
				Period: "4h",
			},
			"api-usage-alert-level2": {
				Type:     marin3rconfig.AlertTypeThreshold,
				RuleName: "RHOAMApiUsageLevel2ThresholdExceeded",
				Level:    "info",
				Threshold: &marin3rconfig.AlertThresholdConfig{
					MinRate: "90%",
					MaxRate: &maxRate2,
				},
				Period: "2h",
			},
			"api-usage-alert-level3": {
				Type:     marin3rconfig.AlertTypeThreshold,
				RuleName: "RHOAMApiUsageLevel3ThresholdExceeded",
				Level:    "info",
				Threshold: &marin3rconfig.AlertThresholdConfig{
					MinRate: "95%",
					MaxRate: nil,
				},
				Period: "30m",
			},
			"rate-limit-spike": {
				Type:     marin3rconfig.AlertTypeSpike,
				RuleName: "RHOAMApiUsageOverLimit",
				Level:    "warning",
				Period:   "30m",
			},
		}

		defaultConfigJSON, err := json.MarshalIndent(defaultConfig, "", "  ")
		if err != nil {
			return err
		}

		alertsConfig.Data["alerts"] = string(defaultConfigJSON)

		return nil
	}); err != nil {
		return integreatlyv1alpha1.PhaseInProgress, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileTenantOauthSecrets(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	allTenants, err := userHelper.GetMultiTenantUsers(ctx, serverClient)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error getting teants for OAuth clients secrets: %w", err)
	}

	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tenantOauthclientSecretsName,
			Namespace: r.ConfigManager.GetOperatorNamespace(),
		},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, oauthClientSecrets, func() error {
		err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: oauthClientSecrets.Name, Namespace: oauthClientSecrets.Namespace}, oauthClientSecrets)
		if !k8serr.IsNotFound(err) && err != nil {
			return err
		} else if k8serr.IsNotFound(err) {
			oauthClientSecrets.Data = map[string][]byte{}
		} else if oauthClientSecrets.Data == nil {
			oauthClientSecrets.Data = map[string][]byte{}
		}
		for _, tenant := range allTenants {
			err = r.reconcileOauthSecretData(ctx, serverClient, oauthClientSecrets, tenant.TenantName)
			if err != nil {
				return fmt.Errorf("error reconciling OAuth secret data for tenant %v: %w", tenant.TenantName, err)
			}
		}
		// Remove redundant tenant secrets
		for key := range oauthClientSecrets.Data {
			if !tenantExists(key, allTenants) {
				delete(oauthClientSecrets.Data, key)
			}
		}
		oauthClientSecrets.ObjectMeta.ResourceVersion = ""
		return nil
	}); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error reconciling oauth clients secrets: %w", err)
	}
	r.log.Info("Tenant OAuth client secrets successfully reconciled")

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileOauthSecretData(ctx context.Context, serverClient k8sclient.Client, oauthClientSecret *corev1.Secret, key string) error {
	if _, ok := oauthClientSecret.Data[key]; !ok {
		oauthClient := &oauthv1.OAuthClient{
			ObjectMeta: metav1.ObjectMeta{
				Name: r.installation.Spec.NamespacePrefix + key,
			},
		}
		err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: oauthClientSecret.Name}, oauthClient)
		if !k8serr.IsNotFound(err) && err != nil {
			r.log.Error("Error getting oauth client secret", err)
			return err
		} else if k8serr.IsNotFound(err) {
			oauthClientSecret.Data[key] = []byte(r.generateSecret(32))
		} else {
			// recover secret from existing OAuthClient object in case Secret object was deleted
			oauthClientSecret.Data[key] = []byte(oauthClient.Secret)
			r.log.Warningf("OAuth client secret recovered from OAuthClient object", l.Fields{"key": key})
		}
	}
	return nil
}

func tenantExists(user string, tenants []userHelper.MultiTenantUser) bool {
	for _, tenant := range tenants {
		if tenant.TenantName == user {
			return true
		}
	}
	return false
}

func (r *Reconciler) reconcileOauthSecrets(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// List of products that require secret for OAuthClient
	productsList := []integreatlyv1alpha1.ProductName{
		integreatlyv1alpha1.ProductRHSSO,
		integreatlyv1alpha1.ProductRHSSOUser,
		integreatlyv1alpha1.Product3Scale,
	}

	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.ConfigManager.GetOauthClientsSecretName(),
			Namespace: r.ConfigManager.GetOperatorNamespace(),
		},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, oauthClientSecrets, func() error {
		err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: oauthClientSecrets.Name, Namespace: oauthClientSecrets.Namespace}, oauthClientSecrets)
		if !k8serr.IsNotFound(err) && err != nil {
			return err
		} else if k8serr.IsNotFound(err) {
			oauthClientSecrets.Data = map[string][]byte{}
		}
		for _, product := range productsList {
			// ugly check the first char is the same as the next three to remove duplicate string if it is, would like a better way to do this
			// this can be removed again after a successful upgrade using this version
			//buf := oauthClientSecrets.Data[string(product)]
			//if buf != nil && buf[0] == buf[1] && buf[0] == buf[2] && buf[0] == buf[3] {
			//	oauthClientSecrets.Data[string(product)] = []byte(r.generateSecret(32))
			//}

			if _, ok := oauthClientSecrets.Data[string(product)]; !ok {
				oauthClient := &oauthv1.OAuthClient{
					ObjectMeta: metav1.ObjectMeta{
						Name: r.installation.Spec.NamespacePrefix + string(product),
					},
				}
				err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: oauthClientSecrets.Name}, oauthClient)
				if !k8serr.IsNotFound(err) && err != nil {
					return err
				} else if k8serr.IsNotFound(err) {
					oauthClientSecrets.Data[string(product)] = []byte(r.generateSecret(32))
				} else {

					// recover secret from existing OAuthClient object in case Secret object was deleted
					oauthClientSecrets.Data[string(product)] = []byte(oauthClient.Secret)
					r.log.Warningf("OAuth client secret recovered from OAuthClient object", l.Fields{"product": string(product)})

				}
			}
		}
		oauthClientSecrets.ObjectMeta.ResourceVersion = ""
		return nil
	}); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error reconciling oauth clients secrets: %w", err)
	}
	r.log.Info("Bootstrap OAuth client secrets successfully reconciled")

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileAddonManagedApiServiceParameters(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	_, err := addon.GetAddonParametersSecret(ctx, serverClient, r.ConfigManager.GetOperatorNamespace())
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) retrieveConsoleURLAndSubdomain(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// Before editing, understand the effect changes to the RoutingSubdomain will have on the following SOP:
	//https://gitlab.cee.redhat.com/rhcloudservices/integreatly-help/-/blob/master/sops/rhoam/ChangeWildcardDomainRhoam/ChangeWildcardDomainRhoam.md

	consoleRouteCR, err := utils.GetConsoleRouteCR(ctx, serverClient)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not find CR route: %w", err)
		}
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not retrieve CR route: %w", err)
	}
	r.installation.Spec.MasterURL = consoleRouteCR.Status.Ingress[0].Host
	routerDefault := strings.TrimPrefix(consoleRouteCR.Status.Ingress[0].RouterCanonicalHostname, "router-default.")
	ok, domain, err := customDomain.GetDomain(ctx, serverClient, r.installation)
	// Only fail when unable to get custom domain parameter from the addon secret to allow for installation of monitoring stack
	if err != nil && !ok {
		log.Warning(err.Error())
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("customDomain.GetDomain() failure: %w", err)
	}
	if ok && domain != "" {
		if r.installation.Spec.RoutingSubdomain == "" { // DO NOT force reconcile of subdomain route, see SOP
			log.Info("setting routing domain to custom domain")
			r.installation.Spec.RoutingSubdomain = domain
		}
		if r.installation.Status.CustomDomain == nil {
			r.installation.Status.CustomDomain = &integreatlyv1alpha1.CustomDomainStatus{}
		}
		r.installation.Status.CustomDomain.Enabled = true
		if err != nil {
			r.installation.Status.CustomDomain.Error = err.Error()
			r.installation.Status.LastError = err.Error()
		}

		return integreatlyv1alpha1.PhaseCompleted, nil
	} else {
		if r.installation.Spec.RoutingSubdomain == "" { // DO NOT force reconcile of subdomain route, see SOP
			log.Info("setting routing domain to cluster default")
			r.installation.Spec.RoutingSubdomain = routerDefault
		}
	}

	if r.installation.Spec.RoutingSubdomain != routerDefault && r.installation.Status.CustomDomain == nil {
		r.installation.Status.CustomDomain = &integreatlyv1alpha1.CustomDomainStatus{Enabled: true}
	}

	if r.installation.Spec.RoutingSubdomain == routerDefault && r.installation.Status.CustomDomain != nil {
		r.installation.Status.CustomDomain = nil
		metrics.SetCustomDomain(false, 0)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcilerGithubOauthSecret(ctx context.Context, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {

	githubOauthSecretCR := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.ConfigManager.GetGHOauthClientsSecretName(),
			Namespace: r.ConfigManager.GetOperatorNamespace(),
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, githubOauthSecretCR, func() error {
		ownerutil.EnsureOwner(githubOauthSecretCR, installation)

		if len(githubOauthSecretCR.Data) == 0 {
			githubOauthSecretCR.Data = map[string][]byte{
				"clientId": []byte("dummy"),
				"secret":   []byte("dummy"),
			}
		}
		return nil
	}); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error reconciling Github OAuth secrets: %w", err)
	}

	r.log.Info("Bootstrap Github OAuth secrets successfully reconciled")

	secretRoleCR := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.ConfigManager.GetGHOauthClientsSecretName(),
			Namespace: r.ConfigManager.GetOperatorNamespace(),
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, secretRoleCR, func() error {
		secretRoleCR.Rules = []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"secrets"},
				Verbs:         []string{"update", "get"},
				ResourceNames: []string{r.ConfigManager.GetGHOauthClientsSecretName()},
			},
		}
		return nil
	}); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating Github OAuth secrets role: %w", err)
	}

	secretRoleBindingCR := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.ConfigManager.GetGHOauthClientsSecretName(),
			Namespace: r.ConfigManager.GetOperatorNamespace(),
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, secretRoleBindingCR, func() error {
		secretRoleBindingCR.RoleRef = rbacv1.RoleRef{
			Name: secretRoleCR.GetName(),
			Kind: "Role",
		}
		secretRoleBindingCR.Subjects = []rbacv1.Subject{
			{
				Name: "dedicated-admins",
				Kind: "Group",
			},
		}
		return nil
	}); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating Github OAuth secrets role binding: %w", err)
	}
	r.log.Info("Bootstrap Github OAuth secrets Role and Role Binding successfully reconciled")

	return integreatlyv1alpha1.PhaseCompleted, nil

}

func (r *Reconciler) generateSecret(length int) string {
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
	buf := make([]rune, length)
	for i := range buf {
		rnd, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			r.log.Error("error generating client secret: ", err)
		}
		buf[i] = chars[rnd.Int64()]
	}
	return string(buf)
}

func (r *Reconciler) processQuota(installation *integreatlyv1alpha1.RHMI, namespace string,
	installationQuota *quota.Quota, serverClient k8sclient.Client) error {
	isQuotaUpdated := false

	quotaParam, err := getSecretQuotaParam(installation, serverClient, namespace)
	if err != nil {
		return err
	}

	// get the quota config map from the cluster
	configMap := &corev1.ConfigMap{}
	err = serverClient.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: quota.ConfigMapName}, configMap)
	if err != nil {
		return fmt.Errorf("error getting quota config map %w", err)
	}

	// Updates the installation quota to the quota param if the quota is updated
	err = quota.GetQuota(context.TODO(), serverClient, quotaParam, configMap, installationQuota)
	if err != nil {
		return err
	}

	// if both are toQuota and Quota are empty this indicates that it's either
	// the first reconcile of an installation or it's the first reconcile of an upgrade to 1.6.0
	// if the secretname is not the same as status.Quota this indicates there has been a quota change
	// to an installation which is already using the Quota functionality.
	// if either case is true set toQuota in the rhmi cr and update the status object and set isQuotaUpdated to true
	if (installation.Status.ToQuota == "" && installation.Status.Quota == "") ||
		installationQuota.GetName() != installation.Status.Quota {
		installation.Status.ToQuota = installationQuota.GetName()
		isQuotaUpdated = true
	}

	installationQuota.SetIsUpdated(isQuotaUpdated)
	return nil
}

func (r *Reconciler) reconcileCustomSMTP(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	smtp, err := cs.GetCustomAddonValues(serverClient, r.installation.Namespace)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	validation := cs.ParameterValidation(smtp)

	switch validation {
	case cs.Valid:
		phase, err := cs.CreateOrUpdateCustomSMTPSecret(ctx, serverClient, smtp, r.installation.Namespace)
		if err != nil {
			return phase, err
		}

		if r.installation.Status.CustomSmtp == nil {
			r.installation.Status.CustomSmtp = &integreatlyv1alpha1.CustomSmtpStatus{}
		}
		r.installation.Status.CustomSmtp.Enabled = true
		r.installation.Status.CustomSmtp.Error = ""
	case cs.Partial:
		phase, err := cs.DeleteCustomSMTP(ctx, serverClient, r.installation.Namespace)
		if err != nil {
			return phase, err
		}

		errorString := cs.ParameterErrors(smtp)
		if r.installation.Status.CustomSmtp == nil {
			r.installation.Status.CustomSmtp = &integreatlyv1alpha1.CustomSmtpStatus{}
		}
		r.installation.Status.CustomSmtp.Enabled = false
		r.installation.Status.CustomSmtp.Error = fmt.Sprintf("Custom SMTP partially configured, missing fields: %s", errorString)
	case cs.Blank:
		phase, err := cs.DeleteCustomSMTP(ctx, serverClient, r.installation.Namespace)
		if err != nil {
			r.installation.Status.CustomSmtp.Enabled = false
			return phase, err
		}
		r.installation.Status.CustomSmtp = nil
	default:
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("unknown validation state found: %s", validation)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) retrieveAPIServerURL(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	cr := &configv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
	}

	key := k8sclient.ObjectKey{
		Name: cr.GetName(),
	}

	err := serverClient.Get(ctx, key, cr)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	r.installation.Spec.APIServer = cr.Status.APIServerURL
	if r.installation.Spec.APIServer != "" {
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("no Status.apiServerURL found in infrastricture CR")
}

func getSecretQuotaParam(installation *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client, namespace string) (string, error) {
	// Check for normal addon quota parameter
	quotaParam, found, err := addon.GetStringParameter(context.TODO(), serverClient, namespace, addon.QuotaParamName)
	if err != nil {
		return "", fmt.Errorf("error checking for quota secret %w", err)
	}

	if found && quotaParam != "" {
		return quotaParam, nil
	}

	// if the param is not found after the installation is 1 minute old it means that it wasn't provided to the installation
	// in this case check for the trial-quota parameter, and use it instead of quota if it is found
	// if trial-quota is not found then check for an Environment Variable QUOTA
	// if neither are found then return an error as there is no QUOTA value for the installation to use and it's required by the reconcilers.
	if isInstallationOlderThan1Minute(installation) {
		quotaParam, found, err = addon.GetStringParameter(context.TODO(), serverClient, namespace, addon.TrialQuotaParamName)
		if err != nil {
			return "", fmt.Errorf("error checking for quota secret %w", err)
		}
		if found && quotaParam != "" {
			return quotaParam, nil
		}

		if !found {
			log.Info(fmt.Sprintf("no secret param found after one minute so falling back to env var '%s' for sku value", integreatlyv1alpha1.EnvKeyQuota))
			quotaValue, exists := os.LookupEnv(integreatlyv1alpha1.EnvKeyQuota)
			if !exists || quotaValue == "" {
				return "", fmt.Errorf("no quota value provided by add on parameter '%s' or by env var '%s'", addon.QuotaParamName, integreatlyv1alpha1.EnvKeyQuota)
			}
			return quotaValue, nil
		}
	}

	return "", fmt.Errorf("waiting for quota parameter for 1 minute after creation of cr")
}
