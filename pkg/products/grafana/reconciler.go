package grafana

import (
	"context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/quota"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Reconciler struct {
	*resources.Reconciler
	ConfigManager config.ConfigReadWriter
	Config        *config.Grafana
	installation  *integreatlyv1alpha1.RHMI
	mpm           marketplace.MarketplaceInterface
	log           l.Logger
	recorder      record.EventRecorder
}

func (r *Reconciler) GetPreflightObject(_ string) k8sclient.Object {
	return nil
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return true
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, logger l.Logger, productDeclaration *marketplace.ProductDeclaration) (*Reconciler, error) {
	productConfig, err := configManager.ReadGrafana()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve grafana config: %w", err)
	}

	if err := configManager.WriteConfig(productConfig); err != nil {
		return nil, fmt.Errorf("error writing grafana config : %w", err)
	}

	return &Reconciler{
		ConfigManager: configManager,
		Config:        productConfig,
		installation:  installation,
		mpm:           mpm,
		log:           logger,
		Reconciler:    resources.NewReconciler(mpm),
		recorder:      recorder,
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, productStatus *integreatlyv1alpha1.RHMIProductStatus, client k8sclient.Client, productConfig quota.ProductConfig, uninstall bool) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Start Grafana reconcile")

	requestsPerUnitStr := fmt.Sprint(productConfig.GetRateLimitConfig().RequestsPerUnit)
	activeQuota := productConfig.GetActiveQuota()

	if !resources.IsInProw(installation) {
		// Creates Grafana RateLimit ConfigMap
		phase, err := ReconcileGrafanaRateLimmitDashboardConfigMap(ctx, client, r.installation, requestsPerUnitStr, activeQuota)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			events.HandleError(r.recorder, installation, phase, "Failed to reconcile Grafana Ratelimit Dashboard ConfigMap", err)
			return phase, err
		}
	}

	r.log.Info("Reconciled successfully")
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func ReconcileGrafanaRateLimmitDashboardConfigMap(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI, requestsPerUnit string, activeQuota string) (integreatlyv1alpha1.StatusPhase, error) {
	log := l.NewLogger()
	log.Info("reconciling Grafana RateLimit Dashboard ConfigMap")
	nsPrefix := installation.Spec.NamespacePrefix

	rateLimitConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ratelimit-grafana-dashboard",
			Namespace: nsPrefix + "customer-monitoring",
		},
		Data: map[string]string{
			"ratelimit.json": getCustomerMonitoringGrafanaRateLimitJSON(requestsPerUnit, activeQuota),
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, rateLimitConfigMap, func() error {
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
