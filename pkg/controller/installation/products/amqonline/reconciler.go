package amqonline

import (
	"context"
	"fmt"
	"github.com/enmasseproject/enmasse/pkg/apis/admin/v1beta1"
	v1beta12 "github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta2"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "amq-online"
	defaultSubscriptionName      = "amq-online"
	defaultConsoleSvcName        = "console"
)

type Reconciler struct {
	Config        *config.AMQOnline
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	restConfig    *rest.Config
	*resources.Reconciler
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	amqOnlineConfig, err := configManager.ReadAMQOnline()
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve keycloak amq online config")
	}

	if amqOnlineConfig.GetNamespace() == "" {
		amqOnlineConfig.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}

	return &Reconciler{
		ConfigManager: configManager,
		Config:        amqOnlineConfig,
		mpm:           mpm,
		Reconciler:    resources.NewReconciler(mpm),
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	phase := inst.Status.ProductStatus[r.Config.GetProductName()]
	logrus.Info("phase status ", phase)

	reconciledPhase, err := r.ReconcileNamespace(ctx, r.Config.GetNamespace(), inst, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, " failed to reconcile namespace for amq online ")
	}

	reconciledPhase, err = r.ReconcileSubscription(ctx, inst, defaultSubscriptionName, r.Config.GetNamespace(), serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, " failed to reconcile subscription for amq online ")
	}
	reconciledPhase, err = r.reconcileAuthServices(ctx, serverClient, GetDefaultAuthServices(r.Config.GetNamespace()))
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, " failed to reconcile amq online auth services ")
	}
	reconciledPhase, err = r.reconcileBrokerConfigs(ctx, serverClient, GetDefaultBrokeredInfraConfigs(r.Config.GetNamespace()), GetDefaultStandardInfraConfigs(r.Config.GetNamespace()))
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, " failed to reconcile amq online broker configs ")
	}
	reconciledPhase, err = r.reconcileAddressPlans(ctx, serverClient, GetDefaultAddressPlans(r.Config.GetNamespace()))
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, " failed to reconcile amq online address plans ")
	}
	reconciledPhase, err = r.reconcileAddressSpacePlans(ctx, serverClient, GetDefaultAddressSpacePlans(r.Config.GetNamespace()))
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, " failed to reconcile amq online address space plans ")
	}
	reconciledPhase, err = r.reconcileConfig(ctx, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, " failed to reconcile amq online config")
	}

	logrus.Info("End of reconcile Phase : ", reconciledPhase)

	return v1alpha1.StatusPhase(v1alpha1.PhaseCompleted), nil
}

func (r *Reconciler) reconcileAuthServices(ctx context.Context, serverClient pkgclient.Client, authSvcs []*v1beta1.AuthenticationService) (v1alpha1.StatusPhase, error) {
	logrus.Info("reconciling default auth services")
	for _, as := range authSvcs {
		as.Namespace = r.Config.GetNamespace()
		err := serverClient.Create(ctx, as)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not create auth service %v", as))
		}
	}
	return v1alpha1.PhaseNone, nil
}

func (r *Reconciler) reconcileBrokerConfigs(ctx context.Context, serverClient pkgclient.Client, brokeredCfgs []*v1beta12.BrokeredInfraConfig, stdCfgs []*v1beta12.StandardInfraConfig) (v1alpha1.StatusPhase, error) {
	logrus.Info("reconciling default infra configs")
	for _, bic := range brokeredCfgs {
		bic.Namespace = r.Config.GetNamespace()
		err := serverClient.Create(ctx, bic)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not create brokered infra config %v", bic))
		}
	}
	for _, sic := range stdCfgs {
		sic.Namespace = r.Config.GetNamespace()
		err := serverClient.Create(ctx, sic)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not create standard infra config %v", sic))
		}
	}
	return v1alpha1.PhaseNone, nil
}

func (r *Reconciler) reconcileAddressPlans(ctx context.Context, serverClient pkgclient.Client, addrPlans []*v1beta2.AddressPlan) (v1alpha1.StatusPhase, error) {
	logrus.Info("reconciling default address plans")
	for _, ap := range addrPlans {
		err := serverClient.Create(ctx, ap)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not create address plan %v", ap))
		}
	}
	return v1alpha1.PhaseNone, nil
}

func (r *Reconciler) reconcileAddressSpacePlans(ctx context.Context, serverClient pkgclient.Client, addrSpacePlans []*v1beta2.AddressSpacePlan) (v1alpha1.StatusPhase, error) {
	logrus.Info("reconciling default address space plans")
	for _, asp := range addrSpacePlans {
		err := serverClient.Create(ctx, asp)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not create address plan %v", asp))
		}
	}
	return v1alpha1.PhaseNone, nil
}

func (r *Reconciler) reconcileConfig(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Infof("reconciling config")
	consoleSvc := &v1beta1.ConsoleService{
		ObjectMeta: v1.ObjectMeta{
			Name:      defaultConsoleSvcName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: defaultConsoleSvcName, Namespace: r.Config.GetNamespace()}, consoleSvc)
	if err != nil {
		if k8serr.IsNotFound(err) {
			logrus.Debugf("could not find consoleservice %s, trying again on next reconcile", defaultConsoleSvcName)
			return v1alpha1.PhaseNone, nil
		}
		return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not retrieve consoleservice %s", defaultConsoleSvcName))
	}
	if consoleSvc.Status.Host != "" && consoleSvc.Status.Port == 443 {
		r.Config.SetHost(fmt.Sprintf("https://%s", consoleSvc.Status.Host))
		if err := r.ConfigManager.WriteConfig(r.Config); err != nil {
			return v1alpha1.PhaseFailed, errors.Wrap(err, "could not persist config")
		}
	}
	return v1alpha1.PhaseNone, nil
}
