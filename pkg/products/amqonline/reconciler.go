package amqonline

import (
	"context"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"strconv"

	"github.com/integr8ly/integreatly-operator/pkg/resources/events"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/version"

	monitoringv1alpha1 "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"
	enmasseadminv1beta1 "github.com/integr8ly/integreatly-operator/pkg/apis-products/enmasse/admin/v1beta1"
	enmassev1beta1 "github.com/integr8ly/integreatly-operator/pkg/apis-products/enmasse/v1beta1"
	enmassev1beta2 "github.com/integr8ly/integreatly-operator/pkg/apis-products/enmasse/v1beta2"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/products/monitoring"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"

	cro1types "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "amq-online"
	defaultConsoleSvcName        = "console"
	manifestPackage              = "integreatly-amq-online"
)

type Reconciler struct {
	Config        *config.AMQOnline
	ConfigManager config.ConfigReadWriter
	extraParams   map[string]string
	mpm           marketplace.MarketplaceInterface
	restConfig    *rest.Config
	log           l.Logger
	inst          *integreatlyv1alpha1.RHMI
	*resources.Reconciler
	recorder record.EventRecorder
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, logger l.Logger) (*Reconciler, error) {
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

	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
		mpm:           mpm,
		log:           logger,
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

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	product := installation.Status.Stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductAMQOnline]
	return version.VerifyProductAndOperatorVersion(
		product,
		string(integreatlyv1alpha1.VersionAMQOnline),
		string(integreatlyv1alpha1.OperatorVersionAMQOnline),
	)
}

// Reconcile reads that state of the cluster for amq online and makes changes based on the state read
// and what is required
func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	operatorNamespace := r.Config.GetOperatorNamespace()
	productNamespace := r.Config.GetNamespace()

	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		phase, err := resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetNamespace(), r.log)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}
		phase, err = resources.RemoveNamespace(ctx, installation, serverClient, operatorNamespace, r.log)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}
		return integreatlyv1alpha1.PhaseCompleted, nil
	}, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, operatorNamespace, installation, serverClient, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", operatorNamespace), err)
		return phase, err
	}

	ns := r.Config.GetNamespace()
	phase, err = r.ReconcileNamespace(ctx, productNamespace, installation, serverClient, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", productNamespace), err)
		return phase, err
	}

	phase, err = r.reconcileSubscription(ctx, serverClient, installation, productNamespace, operatorNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.AMQOnlineSubscriptionName), err)
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

	phase, err = r.reconcileInfraConfigs(ctx, serverClient, GetDefaultBrokeredInfraConfigs(ns), GetDefaultStandardInfraConfigs(ns))
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

	phase, err = r.reconcileServiceAdmin(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile service admin role to dedicated admins group", err)
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

	phase, err = r.reconcileBlackboxTargets(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile blackbox targets", err)
		return phase, err
	}

	phase, err = r.newAlertsReconciler(r.log).ReconcileAlerts(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile amqonline alerts", err)
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

func (r *Reconciler) preUpgradeBackupExecutor() backup.BackupExecutor {
	return backup.NewConcurrentBackupExecutor(
		backup.NewCronJobBackupExecutor(
			"enmasse-postgres-backup",
			r.Config.GetNamespace(),
			"enmasse-preupgrade-postgres-backup",
		),
		backup.NewCronJobBackupExecutor(
			"enmasse-pv-backup",
			r.Config.GetNamespace(),
			"enmasse-preupgrade-pv-backup",
		),
	)
}

func (r *Reconciler) reconcileNoneAuthenticationService(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("reconciling default auth services")

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
		r.log.Error("failed to create/update 'none' AuthenticationService", err)
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update 'none' AuthenticationService: %w", err)
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileStandardAuthenticationService(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("reconciling standard AuthenticationService")

	const postgresqlName string = constants.AMQAuthServicePostgres

	// Get CRO to create a Postgresql in the integreatly operator namespace. The
	// CRO operator SA only has permissions to create the secret in the intly
	// operator namespace, so it will be first created there, then copied into
	// the enmasse namespace.
	postgres, err := croUtil.ReconcilePostgres(
		ctx,
		serverClient,
		defaultInstallationNamespace,
		"workshop", // workshop here so that it creates in-cluster postgresql
		croUtil.TierProduction,
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

	if postgres.Status.Phase != cro1types.PhaseComplete {
		return integreatlyv1alpha1.PhaseAwaitingComponents, nil
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

	// create backup secret
	r.log.Info("Reconciling amq-online postgres backup secret")
	amqOnlneBackUpSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Config.GetPostgresBackupSecretName(),
			Namespace: r.Config.GetNamespace(),
		},
		Data: map[string][]byte{},
	}

	// create or update backup secret
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, amqOnlneBackUpSecret, func() error {
		amqOnlneBackUpSecret.Data["POSTGRES_HOST"] = croSecret.Data["host"]
		amqOnlneBackUpSecret.Data["POSTGRES_USERNAME"] = croSecret.Data["username"]
		amqOnlneBackUpSecret.Data["POSTGRES_PASSWORD"] = croSecret.Data["password"]
		amqOnlneBackUpSecret.Data["POSTGRES_DATABASE"] = croSecret.Data["database"]
		amqOnlneBackUpSecret.Data["POSTGRES_PORT"] = croSecret.Data["port"]
		amqOnlneBackUpSecret.Data["POSTGRES_VERSION"] = []byte("10")
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create or update %s connection secret: %w", r.Config.GetPostgresBackupSecretName(), err)
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

func (r *Reconciler) reconcileInfraConfigs(ctx context.Context, serverClient k8sclient.Client, brokeredCfgs []*enmassev1beta1.BrokeredInfraConfig, stdCfgs []*enmassev1beta1.StandardInfraConfig) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("reconciling default infra configs")

	for _, bic := range brokeredCfgs {

		_, err := controllerutil.CreateOrUpdate(ctx, serverClient, bic, func() error {
			bic.Namespace = r.Config.GetNamespace()
			owner.AddIntegreatlyOwnerAnnotations(bic, r.inst)
			return nil
		})
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not create brokered infra config %v: %w", bic, err)
		}
	}
	for _, sic := range stdCfgs {
		sic.Namespace = r.Config.GetNamespace()
		_, err := controllerutil.CreateOrUpdate(ctx, serverClient, sic, func() error {
			sic.Namespace = r.Config.GetNamespace()
			owner.AddIntegreatlyOwnerAnnotations(sic, r.inst)
			return nil
		})
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not create standard infra config %v: %w", sic, err)
		}
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileAddressPlans(ctx context.Context, serverClient k8sclient.Client, addrPlans []*enmassev1beta2.AddressPlan) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("reconciling default address plans")

	for _, ap := range addrPlans {
		_, err := controllerutil.CreateOrUpdate(ctx, serverClient, ap, func() error {
			owner.AddIntegreatlyOwnerAnnotations(ap, r.inst)
			return nil
		})
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not create address plan %v: %w", ap, err)
		}
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileAddressSpacePlans(ctx context.Context, serverClient k8sclient.Client, addrSpacePlans []*enmassev1beta2.AddressSpacePlan) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("reconciling default address space plans")

	for _, asp := range addrSpacePlans {
		_, err := controllerutil.CreateOrUpdate(ctx, serverClient, asp, func() error {
			owner.AddIntegreatlyOwnerAnnotations(asp, r.inst)
			return nil
		})
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not create address space plan %v: %w", asp, err)
		}
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileServiceAdmin(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("reconciling service admin role to the dedicated admins group")

	serviceAdminRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "enmasse.io:service-admin",
			Namespace: r.Config.GetNamespace(),
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, serviceAdminRole, func() error {
		owner.AddIntegreatlyOwnerAnnotations(serviceAdminRole, r.inst)

		serviceAdminRole.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{"admin.enmasse.io"},
				Resources: []string{"addressplans", "addressspaceplans", "brokeredinfraconfigs", "standardinfraconfigs", "authenticationservices"},
				Verbs:     []string{"create", "get", "update", "delete", "list", "watch", "patch"},
			},
		}

		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Failed reconciling service admin role %v: %w", serviceAdminRole, err)
	}

	// Bind the amq online service admin role to the dedicated-admins group
	serviceAdminRoleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dedicated-admins-service-admin",
			Namespace: r.Config.GetNamespace(),
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, serviceAdminRoleBinding, func() error {
		owner.AddIntegreatlyOwnerAnnotations(serviceAdminRoleBinding, r.inst)

		serviceAdminRoleBinding.RoleRef = rbacv1.RoleRef{
			Name: serviceAdminRole.GetName(),
			Kind: "Role",
		}
		serviceAdminRoleBinding.Subjects = []rbacv1.Subject{
			{
				Name: "dedicated-admins",
				Kind: "Group",
			},
		}

		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Failed reconciling service admin role binding %v: %w", serviceAdminRoleBinding, err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileConfig(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("reconciling config")

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
				Name:     "enmasse-postgres-backup",
				Type:     "postgres",
				Secret:   resources.BackupSecretLocation{Name: r.Config.GetPostgresBackupSecretName(), Namespace: r.Config.GetNamespace()},
				Schedule: r.Config.GetBackupSchedule(),
			},
			{
				Name:     "enmasse-pv-backup",
				Type:     "enmasse_pv",
				Schedule: r.Config.GetBackupSchedule(),
			},
			{
				Name:     "resources-backup",
				Type:     "amq_online_resources",
				Schedule: r.Config.GetBackupSchedule(),
			},
		},
	}

	err := resources.ReconcileBackup(ctx, serverClient, backupConfig, r.ConfigManager, r.log)
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

func (r *Reconciler) reconcileSubscription(ctx context.Context, serverClient k8sclient.Client, inst *integreatlyv1alpha1.RHMI, productNamespace string, operatorNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	target := marketplace.Target{
		Pkg:       constants.AMQOnlineSubscriptionName,
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
		r.log,
	)
}
