package monitoring

import (
	"context"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	v1alpha12 "github.com/integr8ly/integreatly-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace            = "middleware-monitoring"
	defaultSubscriptionName                 = "integreatly-monitoring"
	defaultMonitoringName                   = "middleware-monitoring"
	defaultLabelSelector                    = "middleware"
	defaultAdditionalScrapeConfigSecretName = "integreatly-additional-scrape-configs"
	defaultAdditionalScrapeConfigSecretKey  = "integreatly-additional.yaml"
	defaultPrometheusRetention              = "15d"
	defaultPrometheusStorageRequest         = "10Gi"
	packageName                             = "monitoring"
)

type Reconciler struct {
	Config       *config.Monitoring
	Logger       *logrus.Entry
	mpm          marketplace.MarketplaceInterface
	installation *v1alpha1.Installation
	*resources.Reconciler
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return nil
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	monitoringConfig, err := configManager.ReadMonitoring()

	if err != nil {
		return nil, err
	}

	if monitoringConfig.GetNamespace() == "" {
		monitoringConfig.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		Config:       monitoringConfig,
		Logger:       logger,
		installation: instance,
		mpm:          mpm,
		Reconciler:   resources.NewReconciler(mpm),
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, product *v1alpha1.InstallationProductStatus, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	ns := r.Config.GetNamespace()
	version, err := resources.NewVersion(v1alpha1.OperatorVersionMonitoring)

	phase, err := r.ReconcileNamespace(ctx, ns, inst, serverClient)
	logrus.Infof("Phase: %s ReconcileNamespace", phase)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		logrus.Infof("Error: %s", err)
		return phase, err
	}

	phase, err = r.ReconcileSubscription(ctx, inst, marketplace.Target{Pkg: defaultSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: ns}, serverClient, version)
	logrus.Infof("Phase: %s ReconcileSubscription", phase)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		logrus.Infof("Error: %s", err)
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, inst, serverClient)
	logrus.Infof("Phase: %s reconcileComponents", phase)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		logrus.Infof("Error: %s", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()

	logrus.Infof("%s installation is reconciled successfully", packageName)
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.Logger.Info("Reconciling Monitoring Components")
	m := &v1alpha12.ApplicationMonitoring{
		ObjectMeta: v1.ObjectMeta{
			Name:      defaultMonitoringName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	ownerutil.EnsureOwner(m, inst)
	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, m, func(existing runtime.Object) error {
		monitoring := existing.(*v1alpha12.ApplicationMonitoring)
		monitoring.Spec = v1alpha12.ApplicationMonitoringSpec{
			LabelSelector:                    defaultLabelSelector,
			AdditionalScrapeConfigSecretName: defaultAdditionalScrapeConfigSecretName,
			AdditionalScrapeConfigSecretKey:  defaultAdditionalScrapeConfigSecretKey,
			PrometheusRetention:              defaultPrometheusRetention,
			PrometheusStorageRequest:         defaultPrometheusStorageRequest,
		}
		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to create/update applicationmonitoring custom resource")
	}

	r.Logger.Infof("The operation result for monitoring %s was %s", m.Name, or)

	return v1alpha1.PhaseCompleted, nil
}
