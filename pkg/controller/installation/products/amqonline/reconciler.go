package amqonline

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
)

const (
	defaultInstallationNamespace = "amq-online"
	defaultSubscriptionName      = "amq-online"
	defaultConsoleSvcName        = "console"
)

type Reconciler struct {
	client        pkgclient.Client
	Config        *config.AMQOnline
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	restConfig    *rest.Config
	nsReconciler  resources.NamespaceReconciler
}

type resourceSet struct {
	AddrPlans            []*v1beta2.AddressPlan           `json:"addressPlans"`
	AddrSpacePlans       []*v1beta2.AddressSpacePlan      `json:"addressSpacePlans"`
	AuthServices         []*v1beta1.AuthenticationService `json:"authServices"`
	StdInfraConfigs      []*v1beta12.StandardInfraConfig  `json:"standardInfraConfigs"`
	BrokeredInfraConfigs []*v1beta12.BrokeredInfraConfig  `json:"brokeredInfraConfigs"`
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface, nsr resources.NamespaceReconciler) (*Reconciler, error) {
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
		nsReconciler:  nsr,
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
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
	defResourceSet, err := getResourceSetFromURLList(inst.Spec.AMQOnlineConfig.ResourceURLs, http.DefaultClient)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, " failed to retrieve default resource set for amq online")
	}
	reconciledPhase, err = r.reconcileAuthServices(ctx, serverClient, defResourceSet.AuthServices)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, " failed to reconcile amq online auth services ")
	}
	reconciledPhase, err = r.reconcileBrokerConfigs(ctx, serverClient, defResourceSet.BrokeredInfraConfigs, defResourceSet.StdInfraConfigs)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, " failed to reconcile amq online broker configs ")
	}
	reconciledPhase, err = r.reconcileAddressPlans(ctx, serverClient, defResourceSet.AddrPlans)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, " failed to reconcile amq online address plans ")
	}
	reconciledPhase, err = r.reconcileAddressSpacePlans(ctx, serverClient, defResourceSet.AddrSpacePlans)
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
		ctx,
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
	ip, sub, err := r.mpm.GetSubscriptionInstallPlan(ctx, defaultSubscriptionName, r.Config.GetNamespace())
	if err != nil {
		logrus.Info("error in handleAwaitingOperator ", err.Error())
		if k8serr.IsNotFound(err) {
			logrus.Infof("error in handleAwaitingOperator is not found error ")
			if sub != nil {
				logrus.Infof("time since created %v", time.Now().Sub(sub.CreationTimestamp.Time).Seconds())
			}
			if sub != nil && time.Now().Sub(sub.CreationTimestamp.Time) > config.SubscriptionTimeout {
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
	logrus.Infof("reconciling default auth services (%d)", len(authSvcs))
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
	logrus.Infof("reconciling default infra configs (%d brokered, %d standard)", len(brokeredCfgs), len(stdCfgs))
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
	logrus.Infof("reconciling default address plans (%d)", len(addrPlans))
	for _, ap := range addrPlans {
		ap.Namespace = r.Config.GetNamespace()
		err := serverClient.Create(ctx, ap)
		if err != nil && !k8serr.IsAlreadyExists(err) {
			return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not create address plan %v", ap))
		}
	}
	return v1alpha1.PhaseNone, nil
}

func (r *Reconciler) reconcileAddressSpacePlans(ctx context.Context, serverClient pkgclient.Client, addrSpacePlans []*v1beta2.AddressSpacePlan) (v1alpha1.StatusPhase, error) {
	logrus.Infof("reconciling default address space plans (%d)", len(addrSpacePlans))
	for _, asp := range addrSpacePlans {
		asp.Namespace = r.Config.GetNamespace()
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
		ObjectMeta: v12.ObjectMeta{
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

func getResourceSetFromURLList(urls []string, client *http.Client) (*resourceSet, error) {
	defaultResources := &resourceSet{}

	for _, url := range urls {
		defaultResourcesForURL, err := getResourceSetFromURL(url, client)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("could not retrieve resource set from url %s", url))
		}
		defaultResources.AddrPlans = append(defaultResources.AddrPlans, defaultResourcesForURL.AddrPlans...)
		defaultResources.AddrSpacePlans = append(defaultResources.AddrSpacePlans, defaultResourcesForURL.AddrSpacePlans...)
		defaultResources.AuthServices = append(defaultResources.AuthServices, defaultResourcesForURL.AuthServices...)
		defaultResources.StdInfraConfigs = append(defaultResources.StdInfraConfigs, defaultResourcesForURL.StdInfraConfigs...)
		defaultResources.BrokeredInfraConfigs = append(defaultResources.BrokeredInfraConfigs, defaultResourcesForURL.BrokeredInfraConfigs...)
	}
	return defaultResources, nil
}

func getResourceSetFromURL(url string, client *http.Client) (*resourceSet, error) {
	defaultResources := &resourceSet{}

	res, err := client.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("could not complete default resource request to URL %s", url))
	}
	defer res.Body.Close()
	if err = json.NewDecoder(res.Body).Decode(defaultResources); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("could not decode response for default resources from URL %s", url))
	}
	return defaultResources, nil
}
