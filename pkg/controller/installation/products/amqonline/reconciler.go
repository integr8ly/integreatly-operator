package amqonline

import (
	"context"
	"fmt"

	v1alpha12 "github.com/integr8ly/integreatly-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/monitoring"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/admin/v1beta1"
	v1beta12 "github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta2"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
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
	*resources.Reconciler
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	amqOnlineConfig, err := configManager.ReadAMQOnline()
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve amq online config")
	}

	if amqOnlineConfig.GetNamespace() == "" {
		amqOnlineConfig.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}

	amqOnlineConfig.SetBlackboxTargetPath("/oauth/healthz")

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		ConfigManager: configManager,
		Config:        amqOnlineConfig,
		mpm:           mpm,
		logger:        logger,
		Reconciler:    resources.NewReconciler(mpm),
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
func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, product *v1alpha1.InstallationProductStatus, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileFinalizer(ctx, serverClient, inst, string(r.Config.GetProductName()), func() (v1alpha1.StatusPhase, error) {
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
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, ns, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: defaultSubscriptionName, Namespace: ns, Channel: marketplace.IntegreatlyChannel, ManifestPackage: manifestPackage}, ns, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileAuthServices(ctx, serverClient, GetDefaultAuthServices(ns))
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileBrokerConfigs(ctx, serverClient, GetDefaultBrokeredInfraConfigs(ns), GetDefaultStandardInfraConfigs(ns))
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileAddressPlans(ctx, serverClient, GetDefaultAddressPlans(ns))
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileAddressSpacePlans(ctx, serverClient, GetDefaultAddressSpacePlans(ns))
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileConfig(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileBackup(ctx, inst, serverClient, namespace)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileTemplates(ctx, inst, serverClient)
	logrus.Infof("Phase: %s reconcileTemplates", phase)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		logrus.Infof("Error: %s", err)
		return phase, err
	}

	phase, err = r.reconcileBlackboxTargets(ctx, inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	return v1alpha1.PhaseCompleted, nil
}

// CreateResource Creates a generic kubernetes resource from a template
func (r *Reconciler) createResource(ctx context.Context, inst *v1alpha1.Installation, resourceName string, serverClient pkgclient.Client) (runtime.Object, error) {
	if r.extraParams == nil {
		r.extraParams = map[string]string{}
	}
	r.extraParams["MonitoringKey"] = r.Config.GetLabelSelector()
	r.extraParams["Namespace"] = r.Config.GetNamespace()

	templateHelper := monitoring.NewTemplateHelper(r.extraParams)
	resource, err := templateHelper.CreateResource(resourceName)

	if err != nil {
		return nil, errors.Wrap(err, "createResource failed")
	}

	err = serverClient.Create(ctx, resource)
	if err != nil {
		if !k8serr.IsAlreadyExists(err) {
			return nil, errors.Wrap(err, "error creating resource")
		}
	}

	return resource, nil
}

func (r *Reconciler) reconcileTemplates(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	// Interate over template_list
	for _, template := range r.Config.GetTemplateList() {
		// create it
		_, err := r.createResource(ctx, inst, template, serverClient)
		if err != nil {
			return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("failed to create/update monitoring template %s", template))
		}
		logrus.Infof("Reconciling the monitoring template %s was successful", template)
	}
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileAuthServices(ctx context.Context, serverClient pkgclient.Client, authSvcs []*v1beta1.AuthenticationService) (v1alpha1.StatusPhase, error) {
	r.logger.Info("reconciling default auth services")

	for _, as := range authSvcs {
		as.Namespace = r.Config.GetNamespace()
		err := serverClient.Create(ctx, as)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			return v1alpha1.PhaseFailed, errors.Wrapf(err, "could not create auth service %v", as)
		}
	}
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileBrokerConfigs(ctx context.Context, serverClient pkgclient.Client, brokeredCfgs []*v1beta12.BrokeredInfraConfig, stdCfgs []*v1beta12.StandardInfraConfig) (v1alpha1.StatusPhase, error) {
	r.logger.Info("reconciling default infra configs")

	for _, bic := range brokeredCfgs {
		bic.Namespace = r.Config.GetNamespace()
		err := serverClient.Create(ctx, bic)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			return v1alpha1.PhaseFailed, errors.Wrapf(err, "could not create brokered infra config %v", bic)
		}
	}
	for _, sic := range stdCfgs {
		sic.Namespace = r.Config.GetNamespace()
		err := serverClient.Create(ctx, sic)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			return v1alpha1.PhaseFailed, errors.Wrapf(err, "could not create standard infra config %v", sic)
		}
	}
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileAddressPlans(ctx context.Context, serverClient pkgclient.Client, addrPlans []*v1beta2.AddressPlan) (v1alpha1.StatusPhase, error) {
	r.logger.Info("reconciling default address plans")

	for _, ap := range addrPlans {
		err := serverClient.Create(ctx, ap)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			return v1alpha1.PhaseFailed, errors.Wrapf(err, "could not create address plan %v", ap)
		}
	}
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileAddressSpacePlans(ctx context.Context, serverClient pkgclient.Client, addrSpacePlans []*v1beta2.AddressSpacePlan) (v1alpha1.StatusPhase, error) {
	r.logger.Info("reconciling default address space plans")

	for _, asp := range addrSpacePlans {
		err := serverClient.Create(ctx, asp)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not create address space plan %v", asp))
		}
	}
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileConfig(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Infof("reconciling config")

	consoleSvc := &v1beta1.ConsoleService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultConsoleSvcName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: defaultConsoleSvcName, Namespace: r.Config.GetNamespace()}, consoleSvc)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return v1alpha1.PhaseFailed, errors.Wrapf(err, "could not find consoleservice %s", defaultConsoleSvcName)
		}
		return v1alpha1.PhaseFailed, errors.Wrapf(err, "could not retrieve consoleservice %s", defaultConsoleSvcName)
	}

	if consoleSvc.Status.Host != "" && consoleSvc.Status.Port == 443 {
		r.Config.SetHost(fmt.Sprintf("https://%s", consoleSvc.Status.Host))
		if err := r.ConfigManager.WriteConfig(r.Config); err != nil {
			return v1alpha1.PhaseFailed, errors.Wrap(err, "could not persist config")
		}
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileBackup(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client, owner ownerutil.Owner) (v1alpha1.StatusPhase, error) {
	backupConfig := resources.BackupConfig{
		Namespace: r.Config.GetNamespace(),
		Name:      string(r.Config.GetProductName()),
		BackendSecret: resources.BackupSecretLocation{
			Name:      r.Config.GetBackendSecretName(),
			Namespace: r.ConfigManager.GetOperatorNamespace(),
		},
		Components: []resources.BackupComponent{
			{
				Name:     "enmasse-pv-backup",
				Type:     "enmasse_pv",
				Schedule: r.Config.GetBackupSchedule(),
			},
		},
	}

	err := resources.ReconcileBackup(ctx, serverClient, backupConfig, owner)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrapf(err, "failed to create backups for amq-online")
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileBlackboxTargets(ctx context.Context, inst *v1alpha1.Installation, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	cfg, err := r.ConfigManager.ReadMonitoring()
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "error reading monitoring config")
	}

	err = monitoring.CreateBlackboxTarget("integreatly-amqonline", v1alpha12.BlackboxtargetData{
		Url:     r.Config.GetHost() + "/" + r.Config.GetBlackboxTargetPath(),
		Service: "amq-service-broker",
	}, ctx, cfg, inst, client)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "error creating enmasse blackbox target")
	}

	return v1alpha1.PhaseCompleted, nil
}
