package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	customDomain "github.com/integr8ly/integreatly-operator/pkg/resources/custom-domain"
	"github.com/integr8ly/integreatly-operator/test/common"
	v1 "github.com/openshift/api/config/v1"
	"math/rand"
	"os"
	"strings"
	"time"

	userHelper "github.com/integr8ly/integreatly-operator/pkg/resources/user"

	"github.com/integr8ly/integreatly-operator/pkg/addon"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pkg/errors"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	prometheusv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/metrics"
	"github.com/integr8ly/integreatly-operator/pkg/products/observability"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"

	cs "github.com/integr8ly/integreatly-operator/pkg/resources/custom-smtp"

	oauthv1 "github.com/openshift/api/oauth/v1"
	routev1 "github.com/openshift/api/route/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	res "github.com/integr8ly/integreatly-operator/pkg/resources"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client, installationQuota *quota.Quota, request ctrl.Request) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Reconciling bootstrap stage")

	if integreatlyv1alpha1.IsRHOAM(rhmiv1alpha1.InstallationType(r.installation.Spec.Type)) {
		observabilityConfig, err := r.ConfigManager.ReadObservability()
		if err != nil {
			return rhmiv1alpha1.PhaseFailed, err
		}

		uninstall := false
		if installation.DeletionTimestamp != nil {
			uninstall = true
		}

		phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(observabilityConfig.GetProductName()), uninstall, func() (integreatlyv1alpha1.StatusPhase, error) {
			phase, err := resources.RemoveNamespace(ctx, installation, serverClient, observabilityConfig.GetNamespace(), r.log)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
			return integreatlyv1alpha1.PhaseCompleted, nil
		}, r.log)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
			return phase, err
		}
	}

	phase, err := r.reconcileOauthSecrets(ctx, serverClient)
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

	phase, err = r.reconcilerRHMIConfigCR(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile customer config", err)
		return phase, errors.Wrap(err, "failed to reconcile customer config")
	}

	phase, err = r.reconcileRHMIConfigPermissions(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile customer config dedicated admin permissions", err)
		return phase, errors.Wrap(err, "failed to reconcile customer config dedicated admin permissions")
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

	if integreatlyv1alpha1.IsRHOAM(rhmiv1alpha1.InstallationType(r.installation.Spec.Type)) {
		observabilityConfig, err := r.ConfigManager.ReadObservability()
		if err != nil {
			return rhmiv1alpha1.PhaseFailed, err
		}
		ns := observability.GetDefaultNamespace(r.installation.Spec.NamespacePrefix)
		if observabilityConfig.GetNamespace() == "" {
			observabilityConfig.SetNamespace(ns)
			err := r.ConfigManager.WriteConfig(observabilityConfig)
			if err != nil {
				return rhmiv1alpha1.PhaseFailed, err
			}
		}
		phase, err = r.ReconcileNamespace(ctx, observabilityConfig.GetNamespace(), installation, serverClient, log)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			events.HandleError(r.recorder, installation, phase, "Failed to create observability operand namespace", err)
			return phase, errors.Wrap(err, "failed to create observability operand namespace")
		}
	}

	// temp code for rhmi 2.8 to 2.9.0 upgrades, remove this when all clusters upgraded to 2.9.0
	r.deleteObsoleteService(ctx, serverClient)

	if integreatlyv1alpha1.IsRHOAM(rhmiv1alpha1.InstallationType(installation.Spec.Type)) {
		if err = r.processQuota(installation, request.Namespace, installationQuota, serverClient); err != nil {
			events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, "Error while processing the Quota", err)
			installation.Status.LastError = err.Error()
			return integreatlyv1alpha1.PhaseFailed, err
		}
		metrics.SetQuota(installation.Status.Quota, installation.Status.ToQuota)

		// temp code for RHOAM, remove once all clusters are upgraded to 1.14
		// Remove all prometheus rules under redhat/sandbox-rhoam/rhoami-operator
		phase, err = r.removePrometheusRules(ctx, serverClient, installation.Spec.NamespacePrefix)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			events.HandleError(r.recorder, installation, phase, "Failed to remove existing prometheus rules from rhoam-operator namespace", err)
			return phase, errors.Wrap(err, "Failed to remove existing prometheus rules from rhoam-operator namespace")
		}

	}

	phase, err = r.reconcileCustomSMTP(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Reconciling custom SMTP has failed ", err)
		return phase, errors.Wrap(err, "reconciling custom SMTP has failed ")
	}

	events.HandleStageComplete(r.recorder, installation, integreatlyv1alpha1.BootstrapStage)

	metrics.SetRHMIInfo(installation)
	r.log.Info("Metric rhmi_info exposed")

	r.log.Info("Bootstrap stage reconciled successfully")
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) removePrometheusRules(ctx context.Context, serverClient k8sclient.Client, nsPrefix string) (integreatlyv1alpha1.StatusPhase, error) {
	rhoamProductNamespaces, err := getRHOAMNamespaces(ctx, serverClient, nsPrefix)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	for _, namespace := range rhoamProductNamespaces {
		namespaceRules := &prometheusv1.PrometheusRuleList{}

		err := serverClient.List(ctx, namespaceRules, k8sclient.InNamespace(namespace))
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		} else if k8serr.IsNotFound(err) || len(namespaceRules.Items) == 0 {
			continue
		}

		// Exclude keycloak rule from the deletion as it gets recreated by keycloak operator
		for _, rule := range namespaceRules.Items {
			if rule.Name != "keycloak" {
				err := serverClient.Delete(ctx, rule)
				if err != nil {
					return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Failed to remove %s rule: %s", rule.ObjectMeta.Name, err)
				}
			}
		}
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func getRHOAMNamespaces(ctx context.Context, serverClient k8sclient.Client, nsPrefix string) ([]string, error) {
	var namespaces []string
	namespaceList := &corev1.NamespaceList{}
	err := serverClient.List(ctx, namespaceList)
	if err != nil {
		return nil, err
	}
	// Only return namespaces that have the integreatly label and nsPrefix, but also return rhoam operator ns (it does not have the integreatly label on)
	for _, namespace := range namespaceList.Items {
		if !strings.Contains(namespace.Name, "observability") && strings.Contains(namespace.Name, nsPrefix) || namespace.Name == fmt.Sprintf("%soperator", nsPrefix) {
			namespaces = append(namespaces, namespace.Name)
		}
	}

	return namespaces, nil
}

// temp code for rhmi 2.8 to 2.9.0 upgrades, remove this when all clusters upgraded to 2.9.0
func (r *Reconciler) deleteObsoleteService(ctx context.Context, serverClient k8sclient.Client) {
	if r.installation.Spec.Type == string(integreatlyv1alpha1.InstallationTypeManaged) {
		service := &corev1.Service{}
		err := serverClient.Get(ctx, k8sclient.ObjectKey{
			Name:      "rhmi-operator-metrics",
			Namespace: "redhat-rhmi-operator",
		}, service)
		if err == nil {
			if err := serverClient.Delete(ctx, service); err != nil {
				r.log.Info("Service \"rhmi-operator-metrics\" was deleted from redhat-rhmi-operator")
			}
		}
	}
}

func (r *Reconciler) setTenantMetrics(ctx context.Context, serverClient k8sclient.Client) error {
	total, err := userHelper.GetTotalAPIManagementTenantsCount(ctx, serverClient)
	if err != nil {
		return err
	}
	reconciled, err := userHelper.GetReconciledAPIManagementTenantsCount(ctx, serverClient)
	if err != nil {
		return err
	}
	failed, err := userHelper.GetFailedAPIManagementTenantsCount(ctx, serverClient)
	if err != nil {
		return err
	}
	r.log.Info("Setting tenant metrics")
	metrics.SetTotalNumTenants(total)
	metrics.SetNumReconciledTenants(reconciled)
	metrics.SetNumFailedTenants(failed)
	return nil
}

func (r *Reconciler) reconcilePriorityClass(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	if integreatlyv1alpha1.IsRHOAM(rhmiv1alpha1.InstallationType(r.installation.Spec.Type)) {
		priorityClass := &schedulingv1.PriorityClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: r.installation.Spec.PriorityClassName,
			},
		}
		if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, priorityClass, func() error {
			if integreatlyv1alpha1.IsRHOAMMultitenant(rhmiv1alpha1.InstallationType(r.installation.Spec.Type)) {
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

		if res.Contains(cloudConfig.Finalizers, previousDeletionFinalizer) {
			res.Replace(cloudConfig.Finalizers, previousDeletionFinalizer, deletionFinalizer)
		}

		if strings.ToLower(r.installation.Spec.UseClusterStorage) == "true" {
			cloudConfig.Data["managed"] = `{"blobstorage":"openshift", "smtpcredentials":"openshift", "redis":"openshift", "postgres":"openshift"}`
			cloudConfig.Data["workshop"] = `{"blobstorage":"openshift", "smtpcredentials":"openshift", "redis":"openshift", "postgres":"openshift"}`
			cloudConfig.Data["self-managed"] = `{"blobstorage":"openshift", "smtpcredentials":"openshift", "redis":"openshift", "postgres":"openshift"}`
			cloudConfig.Data["managed-api"] = `{"blobstorage":"openshift", "smtpcredentials":"openshift", "redis":"openshift", "postgres":"openshift"}`
			cloudConfig.Data["multitenant-managed-api"] = `{"blobstorage":"openshift", "smtpcredentials":"openshift", "redis":"openshift", "postgres":"openshift"}`
		} else {
			cloudConfig.Data["managed"] = `{"blobstorage":"aws", "smtpcredentials":"aws", "redis":"aws", "postgres":"aws"}`
			cloudConfig.Data["workshop"] = `{"blobstorage":"openshift", "smtpcredentials":"openshift", "redis":"openshift", "postgres":"openshift"}`
			cloudConfig.Data["self-managed"] = `{"blobstorage":"aws", "smtpcredentials":"aws", "redis":"aws", "postgres":"aws"}`
			cloudConfig.Data["managed-api"] = `{"blobstorage":"aws", "smtpcredentials":"aws", "redis":"aws", "postgres":"aws"}`
			cloudConfig.Data["multitenant-managed-api"] = `{"blobstorage":"aws", "smtpcredentials":"aws", "redis":"aws", "postgres":"aws"}`
		}
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

func (r *Reconciler) reconcilerRHMIConfigCR(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	rhmiConfig := &integreatlyv1alpha1.RHMIConfig{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhmi-config",
			Namespace: r.ConfigManager.GetOperatorNamespace(),
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, rhmiConfig, func() error {
		return nil
	}); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error reconciling the Customer Config CR: %v", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileTenantOauthSecrets(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	allTenants, err := userHelper.GetMultiTenantUsers(ctx, serverClient, r.installation)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Error getting teants for OAuth clients secrets: %w", err)
	}

	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tenantOauthclientSecretsName,
			Namespace: r.ConfigManager.GetOperatorNamespace(),
		},
	}

	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: oauthClientSecrets.Name, Namespace: oauthClientSecrets.Namespace}, oauthClientSecrets)
	if !k8serr.IsNotFound(err) && err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	} else if k8serr.IsNotFound(err) {
		oauthClientSecrets.Data = map[string][]byte{}
	} else if oauthClientSecrets.Data == nil {
		oauthClientSecrets.Data = map[string][]byte{}
	}

	for _, tenant := range allTenants {
		r.reconcileOauthSecretData(ctx, serverClient, oauthClientSecrets, tenant.TenantName)
	}

	// Remove redundant tenant secrets
	for key, _ := range oauthClientSecrets.Data {
		if !tenantExists(key, allTenants) {
			delete(oauthClientSecrets.Data, key)
		}
	}

	oauthClientSecrets.ObjectMeta.ResourceVersion = ""
	err = resources.CreateOrUpdate(ctx, serverClient, oauthClientSecrets)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Error reconciling OAuth clients secrets: %w", err)
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
			oauthClientSecret.Data[key] = []byte(generateSecret(32))
		} else {
			// recover secret from existing OAuthClient object in case Secret object was deleted
			oauthClientSecret.Data[key] = []byte(oauthClient.Secret)
			r.log.Warningf("OAuth client secret recovered from OAutchClient object", l.Fields{"key": key})
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

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: oauthClientSecrets.Name, Namespace: oauthClientSecrets.Namespace}, oauthClientSecrets)
	if !k8serr.IsNotFound(err) && err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	} else if k8serr.IsNotFound(err) {
		oauthClientSecrets.Data = map[string][]byte{}
	}

	for _, product := range productsList {
		if _, ok := oauthClientSecrets.Data[string(product)]; !ok {
			oauthClient := &oauthv1.OAuthClient{
				ObjectMeta: metav1.ObjectMeta{
					Name: r.installation.Spec.NamespacePrefix + string(product),
				},
			}
			err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: oauthClientSecrets.Name}, oauthClient)
			if !k8serr.IsNotFound(err) && err != nil {
				return integreatlyv1alpha1.PhaseFailed, err
			} else if k8serr.IsNotFound(err) {
				oauthClientSecrets.Data[string(product)] = []byte(generateSecret(32))
			} else {
				// recover secret from existing OAuthClient object in case Secret object was deleted
				oauthClientSecrets.Data[string(product)] = []byte(oauthClient.Secret)
				r.log.Warningf("OAuth client secret recovered from OAutchClient object", l.Fields{"product": string(product)})
			}
		}
	}

	oauthClientSecrets.ObjectMeta.ResourceVersion = ""
	err = resources.CreateOrUpdate(ctx, serverClient, oauthClientSecrets)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Error reconciling OAuth clients secrets: %w", err)
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
	consoleRouteCR, err := getConsoleRouteCR(ctx, serverClient)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not find CR route: %w", err)
		}
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not retrieve CR route: %w", err)
	}
	r.installation.Spec.MasterURL = consoleRouteCR.Status.Ingress[0].Host
	threeScaleOperatorNs := &corev1.Namespace{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: common.ThreeScaleOperatorNamespace}, threeScaleOperatorNs)
	if err != nil {
		if k8serr.IsNotFound(err) {
			found, domain, err := customDomain.GetDomain(ctx, serverClient, r.installation)
			if err != nil && !found {
				log.Warning(err.Error())
				return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("customDomain.GetDomain() failure: %w", err)
			}
			if found {
				r.installation.Spec.RoutingSubdomain = domain
				if r.installation.Status.CustomDomain == nil {
					r.installation.Status.CustomDomain = &integreatlyv1alpha1.CustomDomainStatus{}
				}
				r.installation.Status.CustomDomain.Enabled = true
				if err != nil {
					r.installation.Status.CustomDomain.Error = err.Error()
					r.installation.Status.LastError = err.Error()
				}
			}
			if !found {
				r.installation.Spec.RoutingSubdomain = strings.TrimPrefix(consoleRouteCR.Status.Ingress[0].RouterCanonicalHostname, "router-default.")
			}
			return integreatlyv1alpha1.PhaseCompleted, nil
		}
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("unexpected error when retrieving 3scale operator namespace: %w", err)
	}
	// skip custom domain checks if 3scale is installed
	r.installation.Spec.RoutingSubdomain = strings.TrimPrefix(consoleRouteCR.Status.Ingress[0].RouterCanonicalHostname, "router-default.")
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func getConsoleRouteCR(ctx context.Context, serverClient k8sclient.Client) (*routev1.Route, error) {
	// discover and set master url and routing subdomain
	consoleRouteCR := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "console",
			Namespace: "openshift-console",
		},
	}
	key := k8sclient.ObjectKey{
		Name:      consoleRouteCR.GetName(),
		Namespace: consoleRouteCR.GetNamespace(),
	}

	err := serverClient.Get(ctx, key, consoleRouteCR)
	if err != nil {
		return nil, err
	}
	return consoleRouteCR, nil
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
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Error reconciling Github OAuth secrets: %w", err)
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
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Error creating Github OAuth secrets role: %w", err)
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
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Error creating Github OAuth secrets role binding: %w", err)
	}
	r.log.Info("Bootstrap Github OAuth secrets Role and Role Binding successfully reconciled")

	return integreatlyv1alpha1.PhaseCompleted, nil

}

func (r *Reconciler) reconcileRHMIConfigPermissions(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	configRoleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhmiconfig-dedicated-admins-role-binding",
			Namespace: r.ConfigManager.GetOperatorNamespace(),
		},
	}

	// Get the dedicated admins role binding. If it's not found, return
	if err := serverClient.Get(ctx, k8sclient.ObjectKey{
		Name:      "rhmiconfig-dedicated-admins-role-binding",
		Namespace: r.ConfigManager.GetOperatorNamespace(),
	}, configRoleBinding); err != nil {
		if k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseCompleted, nil
		}

		return integreatlyv1alpha1.PhaseFailed, err
	}

	r.log.Info("Found and deleted rhmiconfig-dedicated-admins-role-binding")

	// Delete the role binding
	if err := serverClient.Delete(ctx, configRoleBinding); err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func generateSecret(length int) string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
	buf := make([]rune, length)
	for i := range buf {
		buf[i] = chars[rand.Intn(len(chars))]
	}
	return string(buf)
}

func (r *Reconciler) processQuota(installation *rhmiv1alpha1.RHMI, namespace string,
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
	err = quota.GetQuota(quotaParam, configMap, installationQuota)
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

func (r *Reconciler) reconcileCustomSMTP(ctx context.Context, serverClient k8sclient.Client) (rhmiv1alpha1.StatusPhase, error) {

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
			r.installation.Status.CustomSmtp = &rhmiv1alpha1.CustomSmtpStatus{}
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
			r.installation.Status.CustomSmtp = &rhmiv1alpha1.CustomSmtpStatus{}
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

func (r *Reconciler) retrieveAPIServerURL(ctx context.Context, serverClient k8sclient.Client) (rhmiv1alpha1.StatusPhase, error) {

	cr := &v1.Infrastructure{
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

func getSecretQuotaParam(installation *rhmiv1alpha1.RHMI, serverClient k8sclient.Client, namespace string) (string, error) {
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
			log.Info(fmt.Sprintf("no secret param found after one minute so falling back to env var '%s' for sku value", rhmiv1alpha1.EnvKeyQuota))
			quotaValue, exists := os.LookupEnv(rhmiv1alpha1.EnvKeyQuota)
			if !exists || quotaValue == "" {
				return "", fmt.Errorf("no quota value provided by add on parameter '%s' or by env var '%s'", addon.QuotaParamName, rhmiv1alpha1.EnvKeyQuota)
			}
			return quotaValue, nil
		}
	}

	return "", fmt.Errorf("waiting for quota parameter for 1 minute after creation of cr")
}
