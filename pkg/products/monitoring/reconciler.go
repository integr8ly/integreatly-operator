package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	prometheus "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	rbac "k8s.io/api/rbac/v1"

	v1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"

	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/sirupsen/logrus"

	monitoring "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "middleware-monitoring"
	defaultMonitoringName        = "middleware-monitoring"
	packageName                  = "monitoring"
	openshiftMonitoringNamespace = "openshift-monitoring"
	grafanaDataSourceSecretName  = "grafana-datasources"
	grafanaDataSourceSecretKey   = "prometheus.yaml"
	defaultBlackboxModule        = "http_2xx"
	manifestPackagae             = "integreatly-monitoring"

	// alert manager configuration
	alertManagerRouteName            = "alertmanager-route"
	alertManagerConfigSecretName     = "alertmanager-application-monitoring"
	alertManagerConfigSecretFileName = "alertmanager.yaml"
	alertmanagerAlertAddressEnv      = "ALERTING_EMAIL_ADDRESS"
	alertManagerConfigTemplatePath   = "alertmanager/alertmanager-application-monitoring.yaml"

	// cluster monitorint federation
	federationServiceMonitorName              = "rhmi-alerts-federate"
	federationRoleBindingName                 = "federation-view"
	clusterMonitoringPrometheusServiceAccount = "prometheus-k8s"
	clusterMonitoringNamespace                = "openshift-monitoring"
)

type Reconciler struct {
	Config        *config.Monitoring
	extraParams   map[string]string
	ConfigManager config.ConfigReadWriter
	Logger        *logrus.Entry
	mpm           marketplace.MarketplaceInterface
	installation  *integreatlyv1alpha1.RHMI
	monitoring    *monitoring.ApplicationMonitoring
	*resources.Reconciler
	recorder record.EventRecorder
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return nil
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder) (*Reconciler, error) {
	logger := logrus.NewEntry(logrus.StandardLogger())
	config, err := configManager.ReadMonitoring()

	if err != nil {
		return nil, err
	}

	config.SetNamespacePrefix(installation.Spec.NamespacePrefix)
	if config.GetNamespace() == "" {
		config.SetNamespace(installation.Spec.NamespacePrefix + defaultInstallationNamespace)
	}

	if config.GetOperatorNamespace() == "" {
		if installation.Spec.OperatorsInProductNamespace {
			config.SetOperatorNamespace(config.GetNamespace())
		} else {
			config.SetOperatorNamespace(config.GetNamespace() + "-operator")
		}
	}

	if config.GetFederationNamespace() == "" {
		config.SetFederationNamespace(config.GetNamespace() + "-federate")
	}

	return &Reconciler{
		Config:        config,
		extraParams:   make(map[string]string),
		ConfigManager: configManager,
		Logger:        logger,
		installation:  installation,
		mpm:           mpm,
		Reconciler:    resources.NewReconciler(mpm),
		recorder:      recorder,
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		logrus.Infof("Phase: Monitoring ReconcileFinalizer")

		// Check if namespace is still present before trying to delete it resources
		_, err := resources.GetNS(ctx, r.Config.GetOperatorNamespace(), serverClient)
		if k8serr.IsNotFound(err) {
			//namespace is gone, return complete
			return integreatlyv1alpha1.PhaseCompleted, nil
		}

		logrus.Infof("Phase: Monitoring ReconcileFinalizer list blackboxtargets")
		blackboxtargets := &monitoring.BlackboxTargetList{}
		blackboxtargetsListOpts := []k8sclient.ListOption{
			k8sclient.MatchingLabels(map[string]string{r.Config.GetLabelSelectorKey(): r.Config.GetLabelSelector()}),
		}
		err = serverClient.List(ctx, blackboxtargets, blackboxtargetsListOpts...)
		if err != nil {
			logrus.Infof("Phase: Monitoring ReconcileFinalizer blackboxtargets error")
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to list blackbox targets: %w", err)
		}
		if len(blackboxtargets.Items) > 0 {
			logrus.Infof("Phase: Monitoring ReconcileFinalizer blackboxtargets list > 0")
			// do something to delete these dashboards
			for _, bbt := range blackboxtargets.Items {
				logrus.Infof("Phase: Monitoring ReconcileFinalizer try delete blackboxtarget %s", bbt.Name)
				b := &monitoring.BlackboxTarget{}
				err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: bbt.Name, Namespace: r.Config.GetOperatorNamespace()}, b)
				if k8serr.IsNotFound(err) {
					continue
				}
				if err != nil {
					return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Failed to get %s blackbox target: %w", bbt.Name, err)
				}

				err = serverClient.Delete(ctx, b)
				if err != nil && !k8serr.IsNotFound(err) {
					return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to delete %s blackbox target: %w", b.Name, err)
				}
			}
			return integreatlyv1alpha1.PhaseInProgress, nil
		}

		m := &monitoring.ApplicationMonitoring{}
		err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: defaultMonitoringName, Namespace: r.Config.GetOperatorNamespace()}, m)
		if err != nil && !k8serr.IsNotFound(err) {
			logrus.Infof("Phase: Monitoring ReconcileFinalizer error fetch ApplicationMonitoring CR")
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get %s application monitoring custom resource: %w", defaultMonitoringName, err)
		}
		if !k8serr.IsNotFound(err) {
			if m.DeletionTimestamp == nil {
				logrus.Infof("Phase: Monitoring ReconcileFinalizer delete ApplicationMonitoring CR")
				err = serverClient.Delete(ctx, m)
				if err != nil && !k8serr.IsNotFound(err) {
					return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to delete %s application monitoring custom resource: %w", defaultMonitoringName, err)
				}
			}
			return integreatlyv1alpha1.PhaseInProgress, nil
		}

		phase, err := resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetFederationNamespace())
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		phase, err = resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetOperatorNamespace())
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}

		return integreatlyv1alpha1.PhaseInProgress, nil
	})

	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetOperatorNamespace(), installation, serverClient)
	logrus.Infof("Phase: %s ReconcileNamespace", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetOperatorNamespace()), err)
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, r.Config.GetOperatorNamespace(), serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, fmt.Sprintf("Failed to retrieve %s namespace", r.Config.GetOperatorNamespace()), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: constants.MonitoringSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetOperatorNamespace(), ManifestPackage: manifestPackagae}, []string{r.Config.GetOperatorNamespace()}, backup.NewNoopBackupExecutor(), serverClient)
	logrus.Infof("Phase: %s ReconcileSubscription", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.MonitoringSubscriptionName), err)
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, serverClient)
	logrus.Infof("Phase: %s reconcileComponents", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile components", err)
		return phase, err
	}

	phase, err = r.reconcileAlertManagerConfigSecret(ctx, serverClient)
	logrus.Infof("Phase %s reconcileAlertManagerConfigSecret", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile alert manager config secret", err)
		logrus.Errorf("failed to reconcile alert manager config secret: %v", err)
		return phase, err
	}

	phase, err = r.populateParams(ctx, serverClient)
	logrus.Infof("Phase: %s populateParams", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to populate parameters", err)
		return phase, err
	}

	phase, err = r.reconcileTemplates(ctx, serverClient)
	logrus.Infof("Phase: %s reconcileTemplates", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile templates", err)
		return phase, err
	}

	phase, err = r.reconcileScrapeConfigs(ctx, serverClient)
	logrus.Infof("Phase: %s reconcileScrapeConfigs", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile scrape configs", err)
		return phase, err
	}

	phase, err = r.createFederationNamespace(ctx, serverClient, installation)
	logrus.Infof("Phase: %s labelFederationNamespace", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to label federation namespace", err)
		return phase, err
	}

	phase, err = r.reconcileFederation(ctx, serverClient)
	logrus.Infof("Phase: %s reconcileFederation", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile federation", err)
		return phase, err
	}

	phase, err = r.reconcileBackupMonitoringAlerts(ctx, serverClient)
	logrus.Infof("Phase: %s reconcilePrometheusRule", phase)

	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile backup monitroing alerts", err)
		return phase, err
	}

	phase, err = r.reconcileKubeStateMetricsAlerts(ctx, serverClient)
	logrus.Infof("Phase: %s reconcileKubeStateMetricsAlerts", err)

	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile reconcileKubeStateMetricsAlerts", err)
		return phase, err
	}
	phase, err = r.reconcileKubeStateMetricsMonitoringAlerts(ctx, serverClient)
	logrus.Infof("Phase: %s reconcileKubeStateMonitoringMetricsAlerts", err)

	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile reconcileKubeStateMonitoringMetricsAlerts", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, "Failed to update monitoring config", err)
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not update monitoring config: %w", err)
	}

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.MonitoringStage, r.Config.GetProductName())
	logrus.Infof("%s installation is reconciled successfully", packageName)
	return integreatlyv1alpha1.PhaseCompleted, nil
}

// make the federation namespace discoverable by cluster monitoring
func (r *Reconciler) createFederationNamespace(ctx context.Context, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {
	namespace, err := resources.GetNS(ctx, r.Config.GetFederationNamespace(), serverClient)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		_, err := resources.CreateNSWithProjectRequest(ctx, r.Config.GetFederationNamespace(), serverClient, installation, false, true)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	resources.PrepareObject(namespace, installation, false, true)
	err = serverClient.Update(ctx, namespace)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

// Creates a service monitor that federates metrics about alerts to the cluster
// monitoring stack
func (r *Reconciler) reconcileFederation(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	serviceMonitor := &prometheus.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      federationServiceMonitorName,
			Namespace: r.Config.GetFederationNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, serviceMonitor, func() error {
		serviceMonitor.Labels = map[string]string{
			"k8s-app": federationServiceMonitorName,
			"name":    federationServiceMonitorName,
		}
		serviceMonitor.Spec = prometheus.ServiceMonitorSpec{
			Endpoints: []prometheus.Endpoint{
				{
					Port:   "upstream",
					Path:   "/federate",
					Scheme: "http",
					Params: map[string][]string{
						"match[]": []string{"{__name__=\"ALERTS\",alertstate=\"firing\"}"},
					},
					Interval:      "30s",
					ScrapeTimeout: "30s",
					HonorLabels:   true,
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"application-monitoring": "true",
				},
			},
			NamespaceSelector: prometheus.NamespaceSelector{
				MatchNames: []string{r.Config.GetOperatorNamespace()},
			},
		}
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	r.Logger.Infof("operation result of %v was %v", federationServiceMonitorName, or)

	roleBinding := &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      federationRoleBindingName,
			Namespace: r.Config.GetOperatorNamespace(),
		},
	}

	or, err = controllerutil.CreateOrUpdate(ctx, serverClient, roleBinding, func() error {
		roleBinding.Subjects = []rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Name:      clusterMonitoringPrometheusServiceAccount,
				Namespace: clusterMonitoringNamespace,
			},
		}
		roleBinding.RoleRef = rbac.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     bundle.ClusterRoleKind,
			Name:     "view",
		}
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	r.Logger.Infof("operation result of %v was %v", federationRoleBindingName, or)

	return integreatlyv1alpha1.PhaseCompleted, nil
}

// Create the integreatly additional scrape config secret which is reconciled
// by the application monitoring operator and passed to prometheus
func (r *Reconciler) reconcileScrapeConfigs(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	templateHelper := NewTemplateHelper(r.extraParams)
	threeScaleConfig, err := r.ConfigManager.ReadThreeScale()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error reading config: %w", err)
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
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error loading template: %w", err)
		}

		jobs.Write(bytes)
		jobs.WriteByte('\n')
	}

	scrapeConfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Config.GetAdditionalScrapeConfigSecretName(),
			Namespace: r.Config.GetOperatorNamespace(),
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
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating additional scrape config secret: %w", err)
	}

	r.Logger.Info(fmt.Sprintf("operation result of creating additional scrape config secret was %v", or))

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileTemplates(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// Iterate over template_list
	for _, template := range r.Config.GetTemplateList() {
		// create it
		_, err := r.createResource(ctx, template, serverClient)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update monitoring template %s: %w", template, err)
		}
		r.Logger.Infof("Reconciling the monitoring template %s was successful", template)
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.Logger.Info("Reconciling Monitoring Components")
	m := &monitoring.ApplicationMonitoring{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultMonitoringName,
			Namespace: r.Config.GetOperatorNamespace(),
		},
	}
	owner.AddIntegreatlyOwnerAnnotations(m, r.installation)
	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, m, func() error {
		m.Spec = monitoring.ApplicationMonitoringSpec{
			LabelSelector:                    r.Config.GetLabelSelector(),
			AdditionalScrapeConfigSecretName: r.Config.GetAdditionalScrapeConfigSecretName(),
			AdditionalScrapeConfigSecretKey:  r.Config.GetAdditionalScrapeConfigSecretKey(),
			PrometheusRetention:              r.Config.GetPrometheusRetention(),
			PrometheusStorageRequest:         r.Config.GetPrometheusStorageRequest(),
			AlertmanagerInstanceNamespaces:   r.Config.GetOperatorNamespace(),
			PrometheusInstanceNamespaces:     r.Config.GetOperatorNamespace(),
			SelfSignedCerts:                  r.installation.Spec.SelfSignedCerts,
		}
		r.monitoring = m
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update applicationmonitoring custom resource: %w", err)
	}

	r.Logger.Infof("The operation result for monitoring %s was %s", m.Name, or)
	return integreatlyv1alpha1.PhaseCompleted, nil
}

// CreateResource Creates a generic kubernetes resource from a template
func (r *Reconciler) createResource(ctx context.Context, resourceName string, serverClient k8sclient.Client) (runtime.Object, error) {
	if r.extraParams == nil {
		r.extraParams = map[string]string{}
	}
	r.extraParams["MonitoringKey"] = r.Config.GetLabelSelector()
	r.extraParams["Namespace"] = r.Config.GetOperatorNamespace()

	templateHelper := NewTemplateHelper(r.extraParams)
	resource, err := templateHelper.CreateResource(resourceName)

	if err != nil {
		return nil, fmt.Errorf("createResource failed: %w", err)
	}

	metaObj, err := meta.Accessor(resource)
	if err == nil {
		owner.AddIntegreatlyOwnerAnnotations(metaObj, r.installation)
	}

	err = serverClient.Create(ctx, resource)
	if err != nil {
		if !k8serr.IsAlreadyExists(err) {
			return nil, fmt.Errorf("error creating resource: %w", err)
		}
	}

	return resource, nil
}

// Read the credentials of the Prometheus instance in the openshift-monitoring
// namespace from the grafana datasource secret
func (r *Reconciler) readFederatedPrometheusCredentials(ctx context.Context, serverClient k8sclient.Client) (*monitoring.GrafanaDataSourceSecret, error) {
	secret := &corev1.Secret{}

	selector := k8sclient.ObjectKey{
		Namespace: openshiftMonitoringNamespace,
		Name:      grafanaDataSourceSecretName,
	}

	err := serverClient.Get(ctx, selector, secret)
	if err != nil {
		return nil, err
	}

	prometheusConfig := secret.Data[grafanaDataSourceSecretKey]
	datasources := monitoring.GrafanaDataSourceSecret{}

	err = json.Unmarshal(prometheusConfig, &datasources)
	if err != nil {
		return nil, err
	}

	return &datasources, err
}

// Populate the extra params for templating
func (r *Reconciler) populateParams(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// Obtain the prometheus credentials from openshift-monitoring
	datasources, err := r.readFederatedPrometheusCredentials(ctx, serverClient)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	if len(datasources.DataSources) < 1 {
		return integreatlyv1alpha1.PhaseFailed, errors.New("cannot obtain prometheus credentials")
	}

	// Obtain the 3scale config and namespace
	threeScaleConfig, err := r.ConfigManager.ReadThreeScale()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	r.extraParams["threescale_namespace"] = threeScaleConfig.GetNamespace()
	r.extraParams["namespace-prefix"] = r.installation.Spec.NamespacePrefix
	r.extraParams["openshift_monitoring_namespace"] = openshiftMonitoringNamespace
	r.extraParams["openshift_monitoring_prometheus_username"] = datasources.DataSources[0].BasicAuthUser
	r.extraParams["openshift_monitoring_prometheus_password"] = datasources.DataSources[0].BasicAuthPassword

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileAlertManagerConfigSecret(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.Logger.Infof("reconciling alertmanager configuration secret")
	rhmiOperatorNs := r.installation.Namespace

	// handle alert manager route
	alertmanagerRoute := &v1.Route{}
	if err := serverClient.Get(ctx, types.NamespacedName{Name: alertManagerRouteName, Namespace: r.Config.GetOperatorNamespace()}, alertmanagerRoute); err != nil {
		if k8serr.IsNotFound(err) {
			r.Logger.Infof("alert manager route %s is not available, cannot create alert manager config secret", alertManagerRouteName)
			return integreatlyv1alpha1.PhaseAwaitingComponents, nil
		}
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not obtain alert manager route: %w", err)
	}

	// handle smtp credentials
	smtpSecret := &corev1.Secret{}
	if err := serverClient.Get(ctx, types.NamespacedName{Name: r.installation.Spec.SMTPSecret, Namespace: rhmiOperatorNs}, smtpSecret); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not obtain smtp credentials secret: %w", err)
	}

	// handle pagerduty credentials
	pagerdutySecret := &corev1.Secret{}
	if err := serverClient.Get(ctx, types.NamespacedName{Name: r.installation.Spec.PagerDutySecret, Namespace: rhmiOperatorNs}, pagerdutySecret); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not obtain pagerduty credentials secret: %w", err)
	}
	if len(pagerdutySecret.Data["serviceKey"]) == 0 {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("serviceKey is undefined in pager duty secret")
	}

	// handle dead mans snitch credentials
	dmsSecret := &corev1.Secret{}
	if err := serverClient.Get(ctx, types.NamespacedName{Name: r.installation.Spec.DeadMansSnitchSecret, Namespace: rhmiOperatorNs}, dmsSecret); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not obtain dead mans snitch credentials secret: %w", err)
	}
	if len(dmsSecret.Data["url"]) == 0 {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("url is undefined in dead mans switch secret")
	}

	// only set the to address to a real value for managed deployments
	smtpToAddress := fmt.Sprintf("noreply@%s", alertmanagerRoute.Spec.Host)
	smtpToAddressEnvVal := os.Getenv(alertmanagerAlertAddressEnv)
	if smtpToAddressEnvVal != "" {
		smtpToAddress = smtpToAddressEnvVal
	}

	// parse the config template into a secret object
	templateUtil := NewTemplateHelper(map[string]string{
		"SMTPHost":            string(smtpSecret.Data["host"]),
		"SMTPPort":            string(smtpSecret.Data["port"]),
		"AlertManagerRoute":   alertmanagerRoute.Spec.Host,
		"SMTPUsername":        string(smtpSecret.Data["username"]),
		"SMTPPassword":        string(smtpSecret.Data["password"]),
		"SMTPToAddress":       smtpToAddress,
		"PagerDutyServiceKey": string(pagerdutySecret.Data["serviceKey"]),
		"DeadMansSnitchURL":   string(dmsSecret.Data["url"]),
	})
	configSecretData, err := templateUtil.loadTemplate(alertManagerConfigTemplatePath)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not parse alert manager configuration template: %w", err)
	}
	configSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      alertManagerConfigSecretName,
			Namespace: r.Config.GetOperatorNamespace(),
		},
		Type: corev1.SecretTypeOpaque,
	}

	// create the config secret
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, configSecret, func() error {
		owner.AddIntegreatlyOwnerAnnotations(configSecret, r.installation)
		configSecret.Data = map[string][]byte{
			alertManagerConfigSecretFileName: configSecretData,
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not create or update alert manager secret: %w", err)
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func CreateBlackboxTarget(ctx context.Context, name string, target monitoring.BlackboxtargetData, cfg *config.Monitoring, installation *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client) error {
	if cfg.GetOperatorNamespace() == "" {
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
		"Namespace":     cfg.GetOperatorNamespace(),
		"MonitoringKey": cfg.GetLabelSelector(),
		"name":          name,
		"url":           target.Url,
		"service":       target.Service,
		"module":        module,
	}

	templateHelper := NewTemplateHelper(extraParams)
	obj, err := templateHelper.CreateResource("blackbox/target.yaml")
	if err != nil {
		return fmt.Errorf("error creating resource from template: %w", err)
	}

	metaObj, err := meta.Accessor(obj)
	if err == nil {
		owner.AddIntegreatlyOwnerAnnotations(metaObj, installation)
	}
	// try to create the blackbox target. If if fails with already exist do nothing
	err = serverClient.Create(ctx, obj)
	if err != nil {
		if k8serr.IsAlreadyExists(err) {
			// The target already exists. Nothing else to do
			return nil
		}
		return fmt.Errorf("error creating blackbox target: %w", err)
	}

	return nil
}
