package monitoring

import (
	"context"
	"fmt"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	v1alpha12 "github.com/integr8ly/integreatly-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
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

	phase, err = r.reconcileAlertManagerExtraResources(ctx, inst, serverClient)
	logrus.Infof("Phase: %s reconcileAlertManagerExtraResources", phase)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		logrus.Infof("Error: %s", err)
		return phase, err
	}

	phase, err = r.reconcilePrometheusExtraResources(ctx, inst, serverClient)
	logrus.Infof("Phase: %s reconcilePrometheusExtraResources", phase)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		logrus.Infof("Error: %s", err)
		return phase, err
	}

	phase, err = r.reconcilePrometheusOperatorExtraResources(ctx, inst, serverClient)
	logrus.Infof("Phase: %s reconcilePrometheusOperatorExtraResources", phase)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		logrus.Infof("Error: %s", err)
		return phase, err
	}

	phase, err = r.reconcileGrafanaExtraResources(ctx, inst, serverClient)
	logrus.Infof("Phase: %s reconcileGrafanaExtraResources", phase)
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

func (r *Reconciler) reconcileGrafanaExtraResources(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	ns := r.Config.GetNamespace()

	err := r.createClusterRole(ctx, inst, serverClient, "grafana-operator", []rbacv1.PolicyRule{
		{
			APIGroups: []string{"integreatly.org"},
			Resources: []string{"grafanadashboards"},
			Verbs:     []string{"get", "list", "update", "watch"},
		},
		{
			APIGroups: []string{"authentication.k8s.io"},
			Resources: []string{"tokenreviews"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups: []string{"authorization.k8s.io"},
			Resources: []string{"subjectaccessreviews"},
			Verbs:     []string{"create"},
		},
	})
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	err = r.createClusterRole(ctx, inst, serverClient, "grafana-proxy", []rbacv1.PolicyRule{
		{
			APIGroups: []string{"authentication.k8s.io"},
			Resources: []string{"tokenreviews"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups: []string{"authorization.k8s.io"},
			Resources: []string{"subjectaccessreviews"},
			Verbs:     []string{"create"},
		},
	})
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	err = r.createClusterRoleBinding(ctx, inst, serverClient, fmt.Sprintf("grafana-operator-%s", ns), rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     "grafana-operator",
	}, []rbacv1.Subject{
		{
			Name:      "grafana-operator",
			Namespace: ns,
			Kind:      "ServiceAccount",
		},
	})
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	err = r.createClusterRoleBinding(ctx, inst, serverClient, fmt.Sprintf("grafana-proxy-%s", ns), rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     "grafana-proxy",
	}, []rbacv1.Subject{
		{
			Name:      "grafana-serviceaccount",
			Namespace: ns,
			Kind:      "ServiceAccount",
		},
	})
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcilePrometheusExtraResources(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	ns := r.Config.GetNamespace()

	err := r.createClusterRole(ctx, inst, serverClient, "prometheus-application-monitoring", []rbacv1.PolicyRule{
		{
			APIGroups: []string{"authentication.k8s.io"},
			Resources: []string{"tokenreviews"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups: []string{"authorization.k8s.io"},
			Resources: []string{"subjectaccessreviews"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"nodes", "services", "endpoints", "pods"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"configmaps", "namespaces"},
			Verbs:     []string{"get"},
		},
		{
			NonResourceURLs: []string{"/metrics"},
			Verbs:           []string{"get"},
		},
	})
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	err = r.createClusterRoleBinding(ctx, inst, serverClient, fmt.Sprintf("prometheus-application-monitoring-%s", ns), rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     "prometheus-application-monitoring",
	}, []rbacv1.Subject{
		{
			Name:      "prometheus-application-monitoring",
			Namespace: ns,
			Kind:      "ServiceAccount",
		},
	})
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcilePrometheusOperatorExtraResources(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	ns := r.Config.GetNamespace()

	err := r.createClusterRole(ctx, inst, serverClient, "prometheus-application-monitoring-operator", []rbacv1.PolicyRule{
		{
			APIGroups: []string{"apiextensions.k8s.io"},
			Resources: []string{"customresourcedefinitions"},
			Verbs:     []string{"*"},
		},
		{
			APIGroups: []string{"monitoring.coreos.com"},
			Resources: []string{"alertmanagers", "prometheuses", "prometheuses/finalizers", "alertmanagers/finalizers", "servicemonitors", "prometheusrules", "podmonitors"},
			Verbs:     []string{"*"},
		},
		{
			APIGroups: []string{"apps"},
			Resources: []string{"statefulsets"},
			Verbs:     []string{"*"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"configmaps", "secrets"},
			Verbs:     []string{"*"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"list", "delete"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"services", "endpoints", "services/finalizers"},
			Verbs:     []string{"*"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"nodes"},
			Verbs:     []string{"list", "watch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"namespaces"},
			Verbs:     []string{"get", "list", "watch"},
		},
	})
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	err = r.createClusterRoleBinding(ctx, inst, serverClient, fmt.Sprintf("prometheus-application-monitoring-operator-%s", ns), rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     "prometheus-application-monitoring-operator",
	}, []rbacv1.Subject{
		{
			Name:      "prometheus-operator",
			Namespace: ns,
			Kind:      "ServiceAccount",
		},
	})
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileAlertManagerExtraResources(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	ns := r.Config.GetNamespace()

	err := r.createClusterRole(ctx, inst, serverClient, "alertmanager-application-monitoring", []rbacv1.PolicyRule{
		{
			APIGroups: []string{"authentication.k8s.io"},
			Resources: []string{"tokenreviews"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups: []string{"authorization.k8s.io"},
			Resources: []string{"subjectaccessreviews"},
			Verbs:     []string{"create"},
		},
	})
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	err = r.createClusterRoleBinding(ctx, inst, serverClient, fmt.Sprintf("alertmanager-application-monitoring-%s", ns), rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     "alertmanager-application-monitoring",
	}, []rbacv1.Subject{
		{
			Name:      "alertmanager",
			Namespace: ns,
			Kind:      "ServiceAccount",
		},
	})
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) createClusterRole(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client, name string, rules []rbacv1.PolicyRule) error {
	cr := &rbacv1.ClusterRole{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
	}
	ownerutil.EnsureOwner(cr, inst)
	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, cr, func(existing runtime.Object) error {
		role := existing.(*rbacv1.ClusterRole)
		role.Rules = rules
		return nil
	})
	return err
}

func (r *Reconciler) createClusterRoleBinding(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client, name string, roleRef rbacv1.RoleRef, subjects []rbacv1.Subject) error {
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
	}

	ownerutil.EnsureOwner(crb, inst)
	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, crb, func(existing runtime.Object) error {
		role := existing.(*rbacv1.ClusterRoleBinding)
		role.RoleRef = roleRef
		role.Subjects = subjects
		return nil
	})
	return err
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
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to create/update monitoring custom resource")
	}

	r.Logger.Infof("The operation result for monitoring %s was %s", m.Name, or)

	return v1alpha1.PhaseCompleted, nil
}
