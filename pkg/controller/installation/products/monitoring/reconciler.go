package monitoring

import (
	"context"
	"fmt"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	monitoring_v1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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
	extraParams  map[string]string
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
		extraParams:  make(map[string]string),
		Logger:       logger,
		installation: instance,
		mpm:          mpm,
		Reconciler:   resources.NewReconciler(mpm),
	}, nil
}

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
	version, err := resources.NewVersion(v1alpha1.OperatorVersionMonitoring)

	phase, err = r.ReconcileNamespace(ctx, ns, inst, serverClient)
	logrus.Infof("Phase: %s ReconcileNamespace", phase)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, ns, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: defaultSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: ns}, serverClient, version)
	logrus.Infof("Phase: %s ReconcileSubscription", phase)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, inst, serverClient)
	logrus.Infof("Phase: %s reconcileComponents", phase)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileTemplates(ctx, inst, serverClient)
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

func (r *Reconciler) reconcileTemplates(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	// Interate over template_list
	for _, template := range r.Config.GetTemplateList() {
		// create it
		_, err := r.createResource(inst, template, serverClient)
		if err != nil {
			return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("failed to create/update monitoring template %s", template))
		}
		r.Logger.Infof("Reconciling the monitoring template %s was successful", template)
	}
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.Logger.Info("Reconciling Monitoring Components")
	m := &monitoring_v1alpha1.ApplicationMonitoring{
		ObjectMeta: v1.ObjectMeta{
			Name:      defaultMonitoringName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, m, func(existing runtime.Object) error {
		monitoring := existing.(*monitoring_v1alpha1.ApplicationMonitoring)
		monitoring.Spec = monitoring_v1alpha1.ApplicationMonitoringSpec{
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

// CreateResource Creates a generic kubernetes resource from a templates
func (r *Reconciler) createResource(inst *v1alpha1.Installation, resourceName string, serverClient pkgclient.Client) (runtime.Object, error) {
	templateHelper := newTemplateHelper(inst, r.extraParams, r.Config)
	resourceHelper := newResourceHelper(inst, templateHelper)
	resource, err := resourceHelper.createResource(resourceName)

	if err != nil {
		return nil, errors.Wrap(err, "createResource failed")
	}

	// Set the CR as the owner of this resource so that when
	// the CR is deleted this resource also gets removed
	ownerutil.EnsureOwner(resource.(v1.Object), inst)

	err = serverClient.Create(context.TODO(), resource)
	if err != nil {
		if !kerrors.IsAlreadyExists(err) {
			return nil, errors.Wrap(err, "error creating resource")
		}
	}

	return resource, nil
}
