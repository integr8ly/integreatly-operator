package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	v12 "k8s.io/api/core/v1"

	grafanav1alpha1 "github.com/integr8ly/grafana-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	monitoring_v1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "middleware-monitoring"
	defaultSubscriptionName      = "integreatly-monitoring"
	defaultMonitoringName        = "middleware-monitoring"
	packageName                  = "monitoring"
	openshiftMonitoringNamespace = "openshift-monitoring"
	grafanaDataSourceSecretName  = "grafana-datasources"
	grafanaDataSourceSecretKey   = "prometheus.yaml"
	defaultBlackboxModule        = "http_2xx"
	manifestPackagae             = "integreatly-monitoring"
	alertManagerConfigSecretName = "alertmanager-application-monitoring"
	alertManagerConfigSecretKey  = "alertmanager.yaml"
	alertManagerRoute            = "alertmanager-route"
	alertManagerTemplateName     = "alertmanager"
)

type Reconciler struct {
	Config        *config.Monitoring
	extraParams   map[string]string
	ConfigManager config.ConfigReadWriter
	Logger        *logrus.Entry
	mpm           marketplace.MarketplaceInterface
	installation  *v1alpha1.Installation
	monitoring    *monitoring_v1alpha1.ApplicationMonitoring
	*resources.Reconciler
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return nil
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	logger := logrus.NewEntry(logrus.StandardLogger())
	monitoringConfig, err := configManager.ReadMonitoring()

	if err != nil {
		return nil, err
	}

	monitoringConfig.SetNamespacePrefix(instance.Spec.NamespacePrefix)
	monitoringConfig.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)

	return &Reconciler{
		Config:        monitoringConfig,
		extraParams:   make(map[string]string),
		ConfigManager: configManager,
		Logger:        logger,
		installation:  instance,
		mpm:           mpm,
		Reconciler:    resources.NewReconciler(mpm),
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, product *v1alpha1.InstallationProductStatus, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileFinalizer(ctx, serverClient, inst, string(r.Config.GetProductName()), func() (v1alpha1.StatusPhase, error) {
		dashboards := &grafanav1alpha1.GrafanaDashboardList{}
		dashboardListOpts := []pkgclient.ListOption{
			pkgclient.MatchingLabels(map[string]string{r.Config.GetLabelSelectorKey(): r.Config.GetLabelSelector()}),
		}
		err := serverClient.List(ctx, dashboards, dashboardListOpts...)
		if err != nil {
			return v1alpha1.PhaseFailed, err
		}
		if len(dashboards.Items) > 0 {
			// do something to delete these dashboards
			for _, gdb := range dashboards.Items {
				g := &grafanav1alpha1.GrafanaDashboard{}
				err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: gdb.Name, Namespace: r.Config.GetNamespace()}, g)
				if err != nil {
					return v1alpha1.PhaseFailed, err
				}

				err = serverClient.Delete(ctx, g)
				if err != nil {
					return v1alpha1.PhaseFailed, err
				}
			}
			return v1alpha1.PhaseInProgress, nil
		}

		fuseConfig, err := r.ConfigManager.ReadFuse()
		if err != nil {
			return v1alpha1.PhaseFailed, err
		}

		fuseDashboardlistOpts := []pkgclient.ListOption{
			pkgclient.InNamespace(fuseConfig.GetNamespace()),
		}
		err = serverClient.List(ctx, dashboards, fuseDashboardlistOpts...)
		if err != nil {
			return v1alpha1.PhaseFailed, err
		}
		if len(dashboards.Items) > 0 {
			return v1alpha1.PhaseInProgress, nil
		}

		m := &monitoring_v1alpha1.ApplicationMonitoring{}
		err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: defaultMonitoringName, Namespace: r.Config.GetNamespace()}, m)
		if err != nil && !kerrors.IsNotFound(err) {
			return v1alpha1.PhaseFailed, err
		}
		if !kerrors.IsNotFound(err) {
			if m.DeletionTimestamp == nil {
				err = serverClient.Delete(ctx, m)
				if err != nil {
					return v1alpha1.PhaseFailed, err
				}
			}
			return v1alpha1.PhaseInProgress, nil
		}

		phase, err := resources.RemoveNamespace(ctx, inst, serverClient, r.Config.GetNamespace())
		if err != nil || phase != v1alpha1.PhaseCompleted {
			return phase, err
		}
		return v1alpha1.PhaseCompleted, nil
	})
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	ns := r.Config.GetNamespace()

	phase, err = r.ReconcileNamespace(ctx, ns, inst, serverClient)
	logrus.Infof("Phase: %s ReconcileNamespace", phase)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, ns, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: defaultSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: ns, ManifestPackage: manifestPackagae}, ns, serverClient)
	logrus.Infof("Phase: %s ReconcileSubscription", phase)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, serverClient)
	logrus.Infof("Phase: %s reconcileComponents", phase)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.populateParams(ctx, serverClient)
	logrus.Infof("Phase: %s populateParams", phase)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		logrus.Infof("Error: %s", err)
		return phase, err
	}

	phase, err = r.reconcileTemplates(ctx, serverClient)
	logrus.Infof("Phase: %s reconcileComponents", phase)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		logrus.Infof("Error: %s", err)
		return phase, err
	}

	phase, err = r.reconcileScrapeConfigs(ctx, serverClient)
	logrus.Infof("Phase: %s reconcileScrapeConfigs", phase)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		logrus.Infof("Error: %s", err)
		return phase, err
	}

	phase, err = r.reconcileAlertManagerConfig(ctx, inst, serverClient)
	logrus.Infof("Phase: %s reconcileAlertManagerConfig", phase)
	if err != nil || phase == v1alpha1.PhaseFailed {
		logrus.Infof("Error: %s", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "could not update monitoring config")
	}

	logrus.Infof("%s installation is reconciled successfully", packageName)
	return v1alpha1.PhaseCompleted, nil
}

// Create the integreatly additional scrape config secret which is reconciled
// by the application monitoring operator and passed to prometheus
func (r *Reconciler) reconcileScrapeConfigs(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	templateHelper := NewTemplateHelper(r.extraParams)
	threeScaleConfig, err := r.ConfigManager.ReadThreeScale()
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "error reading config")
	}

	jobs := strings.Builder{}
	for _, job := range r.Config.GetJobTemplates() {
		// Don't include the 3scale extra scrape config if the product is not installed
		if strings.Contains(job, "3scale") && threeScaleConfig.GetNamespace() == "" {
			r.Logger.Info("skipping 3scale additional scrape config")
			continue
		}

		bytes, err := templateHelper.loadTemplate(job)
		if err != nil {
			return v1alpha1.PhaseFailed, errors.Wrap(err, "error loading template")
		}

		jobs.Write(bytes)
		jobs.WriteByte('\n')
	}

	scrapeConfigSecret := &v12.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      r.Config.GetAdditionalScrapeConfigSecretName(),
			Namespace: r.Config.GetNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, scrapeConfigSecret, func() error {
		scrapeConfigSecret.Data = map[string][]byte{
			r.Config.GetAdditionalScrapeConfigSecretKey(): []byte(jobs.String()),
		}
		scrapeConfigSecret.Type = "Opaque"
		scrapeConfigSecret.Labels = map[string]string{
			r.Config.GetLabelSelectorKey(): r.Config.GetLabelSelector(),
		}
		return nil
	})

	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "error creating additional scrape config secret")
	}

	r.Logger.Info(fmt.Sprintf("operation result of creating additional scrape config secret was %v", or))

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileTemplates(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	// Interate over template_list
	for _, template := range r.Config.GetTemplateList() {
		// create it
		_, err := r.createResource(ctx, template, serverClient)
		if err != nil {
			return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("failed to create/update monitoring template %s", template))
		}
		r.Logger.Infof("Reconciling the monitoring template %s was successful", template)
	}
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.Logger.Info("Reconciling Monitoring Components")
	m := &monitoring_v1alpha1.ApplicationMonitoring{
		ObjectMeta: v1.ObjectMeta{
			Name:      defaultMonitoringName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, m, func() error {
		m.Spec = monitoring_v1alpha1.ApplicationMonitoringSpec{
			LabelSelector:                    r.Config.GetLabelSelector(),
			AdditionalScrapeConfigSecretName: r.Config.GetAdditionalScrapeConfigSecretName(),
			AdditionalScrapeConfigSecretKey:  r.Config.GetAdditionalScrapeConfigSecretKey(),
			PrometheusRetention:              r.Config.GetPrometheusRetention(),
			PrometheusStorageRequest:         r.Config.GetPrometheusStorageRequest(),
		}
		r.monitoring = m
		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to create/update applicationmonitoring custom resource")
	}

	r.Logger.Infof("The operation result for monitoring %s was %s", m.Name, or)
	return v1alpha1.PhaseCompleted, nil
}

// CreateResource Creates a generic kubernetes resource from a template
func (r *Reconciler) createResource(ctx context.Context, resourceName string, serverClient pkgclient.Client) (runtime.Object, error) {
	if r.extraParams == nil {
		r.extraParams = map[string]string{}
	}
	r.extraParams["MonitoringKey"] = r.Config.GetLabelSelector()
	r.extraParams["Namespace"] = r.Config.GetNamespace()

	templateHelper := NewTemplateHelper(r.extraParams)
	resource, err := templateHelper.CreateResource(resourceName)

	if err != nil {
		return nil, errors.Wrap(err, "createResource failed")
	}

	err = serverClient.Create(ctx, resource)
	if err != nil {
		if !kerrors.IsAlreadyExists(err) {
			return nil, errors.Wrap(err, "error creating resource")
		}
	}

	return resource, nil
}

// Read the credentials of the Prometheus instance in the openshift-monitoring
// namespace from the grafana datasource secret
func (r *Reconciler) readFederatedPrometheusCredentials(ctx context.Context, serverClient pkgclient.Client) (*monitoring_v1alpha1.GrafanaDataSourceSecret, error) {
	secret := &v12.Secret{}

	selector := pkgclient.ObjectKey{
		Namespace: openshiftMonitoringNamespace,
		Name:      grafanaDataSourceSecretName,
	}

	err := serverClient.Get(ctx, selector, secret)
	if err != nil {
		return nil, err
	}

	prometheusConfig := secret.Data[grafanaDataSourceSecretKey]
	datasources := monitoring_v1alpha1.GrafanaDataSourceSecret{}

	err = json.Unmarshal(prometheusConfig, &datasources)
	if err != nil {
		return nil, err
	}

	return &datasources, err
}

// Populate the extra params for templating
func (r *Reconciler) populateParams(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	// Obtain the prometheus credentials from openshift-monitoring
	datasources, err := r.readFederatedPrometheusCredentials(ctx, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	if len(datasources.DataSources) < 1 {
		return v1alpha1.PhaseFailed, errors.New("cannot obtain prometheus credentials")
	}

	// Obtain the 3scale config and namespace
	threeScaleConfig, err := r.ConfigManager.ReadThreeScale()
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	r.extraParams["threescale_namespace"] = threeScaleConfig.GetNamespace()
	r.extraParams["openshift_monitoring_namespace"] = openshiftMonitoringNamespace
	r.extraParams["openshift_monitoring_prometheus_username"] = datasources.DataSources[0].BasicAuthUser
	r.extraParams["openshift_monitoring_prometheus_password"] = datasources.DataSources[0].BasicAuthPassword

	return v1alpha1.PhaseCompleted, nil
}

func getMonitoringCr(ctx context.Context, cfg *config.Monitoring, serverClient pkgclient.Client) (*monitoring_v1alpha1.ApplicationMonitoring, error) {
	monitoring := monitoring_v1alpha1.ApplicationMonitoring{}

	selector := pkgclient.ObjectKey{
		Namespace: cfg.GetNamespace(),
		Name:      defaultMonitoringName,
	}

	err := serverClient.Get(ctx, selector, &monitoring)
	if err != nil {
		return nil, err
	}

	return &monitoring, nil
}

func CreateBlackboxTarget(name string, target monitoring_v1alpha1.BlackboxtargetData, ctx context.Context, cfg *config.Monitoring, inst *v1alpha1.Installation, serverClient pkgclient.Client) error {
	if cfg.GetNamespace() == "" {
		// Retry later
		return nil
	}

	if target.Url == "" {
		// Retry later if the URL is not yet known
		return nil
	}

	// default policy is to require a 2xx http return code
	module := target.Module
	if module == "" {
		module = defaultBlackboxModule
	}

	// prepare the template
	extraParams := map[string]string{
		"Namespace":     cfg.GetNamespace(),
		"MonitoringKey": cfg.GetLabelSelector(),
		"name":          name,
		"url":           target.Url,
		"service":       target.Service,
		"module":        module,
	}

	cr, err := getMonitoringCr(ctx, cfg, serverClient)
	if err != nil {
		// Retry later
		if kerrors.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "error getting monitoring cr")
	}

	templateHelper := NewTemplateHelper(extraParams)
	obj, err := templateHelper.CreateResource("blackbox/target.yaml")
	if err != nil {
		return errors.Wrap(err, "error creating resource from template")
	}

	cr.TypeMeta = v1.TypeMeta{
		Kind:       monitoring_v1alpha1.ApplicationMonitoringKind,
		APIVersion: monitoring_v1alpha1.SchemeGroupVersion.Version,
	}
	ownerutil.EnsureOwner(obj.(v1.Object), cr)

	// try to create the blackbox target. If if fails with already exist do nothing
	err = serverClient.Create(ctx, obj)
	if err != nil {
		if kerrors.IsAlreadyExists(err) {
			// The target already exists. Nothing else to do
			return nil
		}
		return errors.Wrap(err, "error creating blackbox target")
	}

	return nil
}

func (r *Reconciler) reconcileAlertManagerConfig(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	// Obtain the alertmanager-route
	route := &routev1.Route{}

	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: alertManagerRoute, Namespace: r.Config.GetNamespace()}, route)

	// Wait until the route is available before progressing
	if err != nil {
		if k8serr.IsNotFound(err) {
			return v1alpha1.PhaseInProgress, nil
		}
		return v1alpha1.PhaseFailed, errors.Wrap(err, "error creating alertmanager config")
	}

	// Params required by the alertmanager template
	r.extraParams["alertmanager_route"] = route.Spec.Host
	r.extraParams["alertmanager_to_email"] = inst.Spec.AlertManager.AlertmanagerToEmail
	r.extraParams["dms_webhook_url"] = inst.Spec.AlertManager.DMSWebHookURL
	r.extraParams["pd_service_key"] = inst.Spec.AlertManager.PDServiceKey
	r.extraParams["smtp_smarthost"] = inst.Spec.AlertManager.SMTPSmarthost
	r.extraParams["smtp_auth_username"] = inst.Spec.AlertManager.SMTPauthUsername
	r.extraParams["smtp_auth_password"] = inst.Spec.AlertManager.SMTPauthPassword

	templateHelper := NewTemplateHelper(r.extraParams)
	job := r.Config.GetConfigTemplate()

	bytes, err := templateHelper.loadTemplate(job)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "error loading template")
	}

	alertManagerConfigSecret := &v12.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      alertManagerConfigSecretName,
			Namespace: r.Config.GetNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, alertManagerConfigSecret, func() error {
		alertManagerConfigSecret.Data = map[string][]byte{
			alertManagerConfigSecretKey: bytes,
		}

		return nil
	})

	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "error creating alertmanager config secret")
	}

	r.Logger.Info(fmt.Sprintf("The operation result of creating alertmanager config secret was %v", or))

	return v1alpha1.PhaseCompleted, nil
}
