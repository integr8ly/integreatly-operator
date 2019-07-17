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
	v1alpha12 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	defaultInstallationNamespace = "amq-online"
	defaultSubscriptionName      = "amq-online"
)

type Reconciler struct {
	client        pkgclient.Client
	Config        *config.AMQOnline
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	restConfig    *rest.Config
	nsReconciler  resources.NamespaceReconciler
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface, nsr resources.NamespaceReconciler) (*Reconciler, error) {
	amqOnlineConfig, err := configManager.ReadAMQOnline()
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve keycloak codeReadyConfig")
	}

	if amqOnlineConfig.GetNamespace() == "" {
		amqOnlineConfig.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}

	return &Reconciler{
		ConfigManager: configManager,
		Config:        amqOnlineConfig,
		mpm:           mpm,
		nsReconciler:  nsr,
	}, nil
}

func (r *Reconciler) Reconcile(inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	ctx := context.TODO()
	phase := inst.Status.ProductStatus[r.Config.GetProductName()]
	logrus.Info("phase status ", phase)

	reconciledPhase, err := r.reconcileNamespace(ctx, inst)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, " failed to reconcile namespace for amq online ")
	}

	reconciledPhase, err = r.reconcileSubscription(ctx)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, " failed to reconcile subscription for amq online ")
	}

	reconciledPhase, err = r.handleAwaitingOperator(ctx)
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

	logrus.Info("End of reconcile Phase : ", reconciledPhase)

	return v1alpha1.StatusPhase(v1alpha1.PhaseCompleted), nil
}

func (r *Reconciler) reconcileNamespace(ctx context.Context, inst *v1alpha1.Installation) (v1alpha1.StatusPhase, error) {
	ns := &v1.Namespace{
		ObjectMeta: v12.ObjectMeta{
			Name: r.Config.GetNamespace(),
		},
	}
	ns, err := r.nsReconciler.Reconcile(ctx, ns, inst)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to reconcile amq online namespace "+r.Config.GetNamespace())
	}
	if ns.Status.Phase == v1.NamespaceTerminating {
		logrus.Debugf("namespace %s is terminating, maintaining phase to try again on next reconcile", r.Config.GetNamespace())
		return v1alpha1.PhaseAwaitingNS, nil
	}
	if ns.Status.Phase != v1.NamespaceActive {
		return v1alpha1.PhaseAwaitingNS, nil
	}
	// all good return no status if already
	return v1alpha1.PhaseNone, nil
}

func (r *Reconciler) reconcileSubscription(ctx context.Context) (v1alpha1.StatusPhase, error) {
	// NEED to ensure a subscription is created if not exists
	// need to make sure there is only one operator source
	logrus.Infof("reconciling subscription %s from channel %s in namespace: %s", defaultSubscriptionName, "integreatly", r.Config.GetNamespace())
	err := r.mpm.CreateSubscription(
		marketplace.GetOperatorSources().Integreatly,
		r.Config.GetNamespace(),
		defaultSubscriptionName,
		marketplace.IntegreatlyChannel,
		[]string{r.Config.GetNamespace()},
		v1alpha12.ApprovalAutomatic)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not create subscription in namespace: %s", r.Config.GetNamespace()))
	}
	return r.handleAwaitingOperator(ctx)
}

func (r *Reconciler) handleAwaitingOperator(ctx context.Context) (v1alpha1.StatusPhase, error) {
	logrus.Infof("checking installplan is created for subscription %s in namespace: %s", defaultSubscriptionName, r.Config.GetNamespace())
	ip, sub, err := r.mpm.GetSubscriptionInstallPlan(defaultSubscriptionName, r.Config.GetNamespace())
	if err != nil {
		logrus.Info("error in handleAwaitingOperator ", err.Error())
		if k8serr.IsNotFound(err) {
			logrus.Infof("error in handleAwaitingOperator is not found error ")
			if sub != nil {
				logrus.Infof("time since created %v", time.Now().Sub(sub.CreationTimestamp.Time).Seconds())
			}
			if sub != nil && time.Now().Sub(sub.CreationTimestamp.Time) > time.Second*60 {
				// delete subscription so it is recreated
				logrus.Info("removing subscription as no install plan ready yet will recreate")
				if err := r.client.Delete(ctx, sub, func(options *pkgclient.DeleteOptions) {
					gp := int64(0)
					options.GracePeriodSeconds = &gp

				}); err != nil {
					// not going to fail here will retry
					logrus.Error("failed to delete sub after install plan was not available for more than 20 seconds . Ignoring will retry ", err)
				}
			}
			logrus.Debugf(fmt.Sprintf("installplan resource is not found in namespace: %s", r.Config.GetNamespace()))
			return v1alpha1.PhaseAwaitingOperator, nil
		}
		return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not retrieve installplan in namespace: %s", r.Config.GetNamespace()))
	}

	logrus.Infof("installplan phase is %s", ip.Status.Phase)
	if ip.Status.Phase != v1alpha12.InstallPlanPhaseComplete {
		logrus.Infof("amq online online install plan is not complete yet")
		return v1alpha1.PhaseAwaitingOperator, nil
	}
	logrus.Infof("amq online online install plan is complete. Installation ready ")
	return v1alpha1.PhaseNone, nil
}

func (r *Reconciler) reconcileAuthServices(ctx context.Context, serverClient pkgclient.Client, authSvcs []*v1beta1.AuthenticationService) (v1alpha1.StatusPhase, error) {
	logrus.Infof("reconciling default auth services")
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
	logrus.Infof("reconciling default infra configs")
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
	logrus.Infof("reconciling default address plans")
	for _, ap := range addrPlans {
		err := serverClient.Create(ctx, ap)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not create address plan %v", ap))
		}
	}
	return v1alpha1.PhaseNone, nil
}

func (r *Reconciler) reconcileAddressSpacePlans(ctx context.Context, serverClient pkgclient.Client, addrSpacePlans []*v1beta2.AddressSpacePlan) (v1alpha1.StatusPhase, error) {
	logrus.Infof("reconciling default address space plans")
	for _, asp := range addrSpacePlans {
		err := serverClient.Create(ctx, asp)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not create address plan %v", asp))
		}
	}
	return v1alpha1.PhaseNone, nil
}
