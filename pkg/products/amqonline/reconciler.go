package amqonline

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"

	enmasseadminv1beta1 "github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/admin/v1beta1"
	enmassev1beta1 "github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta1"
	enmassev1beta2 "github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta2"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	monitoringv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/products/monitoring"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"

	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/resources"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "amq-online"
	defaultSubscriptionName      = "integreatly-amq-online"
	defaultConsoleSvcName        = "console"
	manifestPackage              = "integreatly-amq-online"
)

type Reconciler struct {
	Config        *config.AMQOnline
	ConfigManager config.ConfigReadWriter
	extraParams   map[string]string
	mpm           marketplace.MarketplaceInterface
	restConfig    *rest.Config
	logger        *logrus.Entry
	inst          *integreatlyv1alpha1.RHMI
	*resources.Reconciler
	recorder record.EventRecorder
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder) (*Reconciler, error) {
	config, err := configManager.ReadAMQOnline()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve amq online config: %w", err)
	}

	if config.GetNamespace() == "" {
		config.SetNamespace(installation.Spec.NamespacePrefix + defaultInstallationNamespace)
	}
	if config.GetOperatorNamespace() == "" {
		config.SetOperatorNamespace(config.GetNamespace())
	}

	config.SetBlackboxTargetPath("/oauth/healthz")

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
		mpm:           mpm,
		logger:        logger,
		Reconciler:    resources.NewReconciler(mpm),
		inst:          installation,
		recorder:      recorder,
	}, nil
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-server",
			Namespace: ns,
		},
	}
}

// Reconcile reads that state of the cluster for amq online and makes changes based on the state read
// and what is required
func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		phase, err := resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetNamespace())
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}
		phase, err = resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetOperatorNamespace())
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}
		return integreatlyv1alpha1.PhaseCompleted, nil
	})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetOperatorNamespace(), installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetOperatorNamespace()), err)
		return phase, err
	}

	ns := r.Config.GetNamespace()
	phase, err = r.ReconcileNamespace(ctx, ns, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", ns), err)
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, ns, serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, fmt.Sprintf("Failed to retrieve %s namespace", ns), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: defaultSubscriptionName, Namespace: r.Config.GetOperatorNamespace(), Channel: marketplace.IntegreatlyChannel, ManifestPackage: manifestPackage}, []string{ns}, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", defaultSubscriptionName), err)
		return phase, err
	}

	phase, err = r.reconcileNoneAuthenticationService(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile 'none' auth service", err)
		return phase, err
	}

	phase, err = r.reconcileStandardAuthenticationService(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile 'standard' auth service", err)
		return phase, err
	}

	phase, err = r.reconcileBrokerConfigs(ctx, serverClient, GetDefaultBrokeredInfraConfigs(ns), GetDefaultStandardInfraConfigs(ns))
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile broker configs", err)
		return phase, err
	}

	phase, err = r.reconcileAddressPlans(ctx, serverClient, GetDefaultAddressPlans(ns))
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile address plans", err)
		return phase, err
	}

	phase, err = r.reconcileAddressSpacePlans(ctx, serverClient, GetDefaultAddressSpacePlans(ns))
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile address space plans", err)
		return phase, err
	}

	phase, err = r.reconcileConfig(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile console service config", err)
		return phase, err
	}

	phase, err = r.reconcileBackup(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile backup", err)
		return phase, err
	}

	phase, err = r.reconcileTemplates(ctx, serverClient)
	logrus.Infof("Phase: %s reconcileTemplates", phase)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile templates", err)
		return phase, err
	}

	phase, err = r.reconcileBlackboxTargets(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile blackbox targets", err)
		return phase, err
	}

	phase, err = r.reconcilePrometheusRule(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile prometheus rules", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
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

func (r *Reconciler) reconcileTemplates(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// Interate over template_list
	for _, template := range r.Config.GetTemplateList() {
		// create it
		_, err := r.createResource(ctx, template, serverClient)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update monitoring template %s: %w", template, err)
		}
		logrus.Infof("Reconciling the monitoring template %s was successful", template)
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileNoneAuthenticationService(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.logger.Info("reconciling default auth services")

	noneAuthService := &enmasseadminv1beta1.AuthenticationService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "none-authservice",
			Namespace: r.Config.GetNamespace(),
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, noneAuthService, func() error {
		owner.AddIntegreatlyOwnerAnnotations(noneAuthService, r.inst)
		noneAuthService.Spec.Type = "none"
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update 'none' AuthenticationService: %w", err)
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileStandardAuthenticationService(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.logger.Info("reconciling standard AuthenticationService")

	const postgresqlName string = "standard-authservice-postgresql"

	// Get CRO to create a Postgresql in the integreatly operator namespace. The
	// CRO operator SA only has permissions to create the secret in the intly
	// operator namespace, so it will be first created there, then copied into
	// the enmasse namespace.
	_, err := croUtil.ReconcilePostgres(
		ctx,
		serverClient,
		defaultInstallationNamespace,
		"workshop", // workshop here so that it creates in-cluster postgresql
		"production",
		postgresqlName,
		r.inst.Namespace,
		postgresqlName,
		r.inst.Namespace,
		func(cr metav1.Object) error {
			owner.AddIntegreatlyOwnerAnnotations(cr, r.inst)
			return nil
		},
	)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not create postgresql for standard auth service: %w", err)
	}

	// Read the CRO secret, to get values to copy to enmasse namespace.
	croSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.inst.Namespace,
			Name:      postgresqlName,
		},
	}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: croSecret.Name, Namespace: croSecret.Namespace}, croSecret)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseAwaitingComponents, nil
		}
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Copy secret to enmasse namespace (adjust keys for keycloak)
	keycloakSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      postgresqlName,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, keycloakSecret, func() error {
		// I think it would be better to set the owner here to be the
		// AuthService, but I can't immediately figure out how to get a
		// reference to the scheme.
		owner.AddIntegreatlyOwnerAnnotations(keycloakSecret, r.inst)

		if keycloakSecret.Data == nil {
			keycloakSecret.Data = make(map[string][]byte, 2)
		}

		keycloakSecret.Data["database-user"] = croSecret.Data["username"]
		keycloakSecret.Data["database-password"] = croSecret.Data["password"]
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed reconciling enmasse standard auth service keycloak secret: %w", err)
	}

	standardAuthSvc := &enmasseadminv1beta1.AuthenticationService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "standard-authservice",
			Namespace: r.Config.GetNamespace(),
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, standardAuthSvc, func() error {
		owner.AddIntegreatlyOwnerAnnotations(standardAuthSvc, r.inst)
		standardAuthSvc.Spec.Type = "standard"

		if standardAuthSvc.Spec.Standard == nil {
			standardAuthSvc.Spec.Standard = &enmasseadminv1beta1.AuthenticationServiceSpecStandard{}
		}
		if standardAuthSvc.Spec.Standard.Datasource == nil {
			standardAuthSvc.Spec.Standard.Datasource = &enmasseadminv1beta1.AuthenticationServiceSpecStandardDatasource{}
		}

		standardAuthSvc.Spec.Standard.Datasource.CredentialsSecret = corev1.SecretReference{
			Name:      keycloakSecret.Name,
			Namespace: keycloakSecret.Namespace,
		}
		standardAuthSvc.Spec.Standard.Datasource.Type = enmasseadminv1beta1.PostgresqlDatasource
		standardAuthSvc.Spec.Standard.Datasource.Database = string(croSecret.Data["database"])
		standardAuthSvc.Spec.Standard.Datasource.Host = string(croSecret.Data["host"])
		standardAuthSvc.Spec.Standard.Datasource.Port, err = strconv.Atoi(string(croSecret.Data["port"]))
		return err // either nil or failed to convert port to an int
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed reconciling standard AuthenticationService: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileBrokerConfigs(ctx context.Context, serverClient k8sclient.Client, brokeredCfgs []*enmassev1beta1.BrokeredInfraConfig, stdCfgs []*enmassev1beta1.StandardInfraConfig) (integreatlyv1alpha1.StatusPhase, error) {
	r.logger.Info("reconciling default infra configs")

	for _, bic := range brokeredCfgs {
		bic.Namespace = r.Config.GetNamespace()
		owner.AddIntegreatlyOwnerAnnotations(bic, r.inst)
		err := serverClient.Create(ctx, bic)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not create brokered infra config %v: %w", bic, err)
		}
	}
	for _, sic := range stdCfgs {
		sic.Namespace = r.Config.GetNamespace()
		owner.AddIntegreatlyOwnerAnnotations(sic, r.inst)
		err := serverClient.Create(ctx, sic)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not create standard infra config %v: %w", sic, err)
		}
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileAddressPlans(ctx context.Context, serverClient k8sclient.Client, addrPlans []*enmassev1beta2.AddressPlan) (integreatlyv1alpha1.StatusPhase, error) {
	r.logger.Info("reconciling default address plans")

	for _, ap := range addrPlans {
		owner.AddIntegreatlyOwnerAnnotations(ap, r.inst)
		err := serverClient.Create(ctx, ap)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not create address plan %v: %w", ap, err)
		}
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileAddressSpacePlans(ctx context.Context, serverClient k8sclient.Client, addrSpacePlans []*enmassev1beta2.AddressSpacePlan) (integreatlyv1alpha1.StatusPhase, error) {
	r.logger.Info("reconciling default address space plans")

	for _, asp := range addrSpacePlans {
		owner.AddIntegreatlyOwnerAnnotations(asp, r.inst)
		err := serverClient.Create(ctx, asp)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not create address space plan %v: %w", asp, err)
		}
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileConfig(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.logger.Infof("reconciling config")

	consoleSvc := &enmasseadminv1beta1.ConsoleService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultConsoleSvcName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: defaultConsoleSvcName, Namespace: r.Config.GetNamespace()}, consoleSvc)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not find consoleservice %s: %w", defaultConsoleSvcName, err)
		}
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not retrieve consoleservice %s: %w", defaultConsoleSvcName, err)
	}

	if consoleSvc.Status.Host != "" && consoleSvc.Status.Port == 443 {
		r.Config.SetHost(fmt.Sprintf("https://%s", consoleSvc.Status.Host))
		if err := r.ConfigManager.WriteConfig(r.Config); err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not persist config: %w", err)
		}
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileBackup(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	backupConfig := resources.BackupConfig{
		Namespace: r.Config.GetNamespace(),
		Name:      string(r.Config.GetProductName()),
		BackendSecret: resources.BackupSecretLocation{
			Name:      r.Config.GetBackupsSecretName(),
			Namespace: r.Config.GetNamespace(),
		},
		Components: []resources.BackupComponent{
			{
				Name:     "enmasse-pv-backup",
				Type:     "enmasse_pv",
				Schedule: r.Config.GetBackupSchedule(),
			},
		},
	}

	err := resources.ReconcileBackup(ctx, serverClient, backupConfig, r.ConfigManager)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create backups for amq-online: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileBlackboxTargets(ctx context.Context, installation *integreatlyv1alpha1.RHMI, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	cfg, err := r.ConfigManager.ReadMonitoring()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error reading monitoring config: %w", err)
	}

	err = monitoring.CreateBlackboxTarget(ctx, "integreatly-amqonline", monitoringv1alpha1.BlackboxtargetData{
		Url:     r.Config.GetHost() + r.Config.GetBlackboxTargetPath(),
		Service: "amq-service-broker",
	}, cfg, installation, client)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating enmasse blackbox target: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcilePrometheusRule(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	monitoringConfig := config.NewMonitoring(config.ProductConfig{})
	keycloakServicePortCount := 2
	rule := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rhmi-amq-online-slo",
			Namespace: r.Config.GetNamespace(),
		},
	}

	rules := []monitoringv1.Rule{
		{
			Alert: fmt.Sprintf("AMQOnlineConsoleAvailable"),
			Annotations: map[string]string{
				"sop_url": "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/alerts_and_troubleshooting.md",
				"message": fmt.Sprintf("AMQ-SLO-1.1: AMQ Online console is not available in namespace '%s'", r.Config.GetNamespace()),
			},
			Expr:   intstr.FromString(fmt.Sprintf("absent(kube_endpoint_address_available{endpoint='console',namespace='%s'}==1)", r.Config.GetNamespace())),
			For:    "300s",
			Labels: map[string]string{"severity": "critical"},
		},
		{
			Alert: fmt.Sprintf("AMQOnlineKeycloakAvailable"),
			Annotations: map[string]string{
				"sop_url": "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/alerts_and_troubleshooting.md",
				"message": fmt.Sprintf("AMQ-SLO-1.4: Keycloak is not available in namespace %s", r.Config.GetNamespace()),
			},
			Expr:   intstr.FromString(fmt.Sprintf("absent(kube_endpoint_address_available{endpoint='standard-authservice',namespace='%s'}==%v)", r.Config.GetNamespace(), keycloakServicePortCount)),
			For:    "300s",
			Labels: map[string]string{"severity": "critical"},
		},
		{
			Alert: fmt.Sprintf("AMQOnlineOperatorAvailable"),
			Annotations: map[string]string{
				"sop_url": "https://github.com/RHCloudServices/integreatly-help/blob/master/sops/alerts_and_troubleshooting.md",
				"message": fmt.Sprintf("AMQ-SLO-1.5: amq-online(enmasse) operator is not available in namespace %s", r.Config.GetNamespace()),
			},
			Expr:   intstr.FromString(fmt.Sprintf("absent(kube_pod_status_ready{condition='true',namespace='%s',pod=~'enmasse-operator-.*'}==1)", r.Config.GetNamespace())),
			For:    "300s",
			Labels: map[string]string{"severity": "critical"},
		}}

	_, err := controllerutil.CreateOrUpdate(ctx, client, rule, func() error {
		rule.ObjectMeta.Labels = map[string]string{"integreatly": "yes", monitoringConfig.GetLabelSelectorKey(): monitoringConfig.GetLabelSelector()}
		rule.Spec = monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name:  "amqonline.rules",
					Rules: rules,
				},
			},
		}
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating enmasse PrometheusRule: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
