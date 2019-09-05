package fuse

import (
	"context"
	"fmt"
	"strings"

	appsv1 "github.com/openshift/api/apps/v1"
	v1 "github.com/openshift/api/route/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	usersv1 "github.com/openshift/api/user/v1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	syn "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "fuse"
	defaultSubscriptionName      = "integreatly-syndesis"
	adminGroupName               = "dedicated-admins"
	defaultFusePullSecret        = "syndesis-pull-secret"
)

type Reconciler struct {
	*resources.Reconciler
	coreClient    kubernetes.Interface
	Config        *config.Fuse
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	fuseConfig, err := configManager.ReadFuse()
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve fuse config")
	}

	if fuseConfig.GetNamespace() == "" {
		fuseConfig.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}
	if err = fuseConfig.Validate(); err != nil {
		return nil, errors.Wrap(err, "fuse config is not valid")
	}

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		ConfigManager: configManager,
		Config:        fuseConfig,
		mpm:           mpm,
		logger:        logger,
		Reconciler:    resources.NewReconciler(mpm),
	}, nil
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "syndesis-server",
			Namespace: ns,
		},
	}
}

// Reconcile reads that state of the cluster for fuse and makes changes based on the state read
// and what is required
func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, product *v1alpha1.InstallationProductStatus, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileNamespace(ctx, r.Config.GetNamespace(), inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}
	phase, err = r.ReconcilePullSecret(ctx, r.Config.GetNamespace(), inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileAdminPerms(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.ReconcileSubscription(ctx, inst, marketplace.Target{Pkg: defaultSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetNamespace()}, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}
	phase, err = r.reconcileCustomResource(ctx, inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileOauthProxy(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()

	logrus.Infof("%s has reconciled successfully", r.Config.GetProductName())
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileAdminPerms(ctx context.Context, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Infof("Reconciling permissions for %s group on %s namespace", adminGroupName, r.Config.GetNamespace())

	roleName := adminGroupName + "-view-fuse"
	adminViewFuseRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: roleName,
		},
	}
	or, err := controllerutil.CreateOrUpdate(ctx, client, adminViewFuseRole, func(existing runtime.Object) error {
		cr := existing.(*rbacv1.ClusterRole)

		cr.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"namespaces"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{"*"},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list"},
			},
			{
				APIGroups: []string{"*"},
				Resources: []string{"pods/log"},
				Verbs:     []string{"get"},
			},
		}

		return nil
	})
	r.logger.Infof("The %s role perms were: %s", adminViewFuseRole.Name, or)

	openshiftUsers := &usersv1.UserList{}
	err = client.List(ctx, &pkgclient.ListOptions{}, openshiftUsers)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	openshiftAdminGroup := &usersv1.Group{}
	err = client.Get(ctx, pkgclient.ObjectKey{Name: adminGroupName}, openshiftAdminGroup)
	if err != nil && !k8serr.IsNotFound(err) {
		return v1alpha1.PhaseFailed, err
	}

	adminViewFuseRoleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: r.Config.GetNamespace(),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     roleName,
		},
	}
	or, err = controllerutil.CreateOrUpdate(ctx, client, adminViewFuseRoleBinding, func(existing runtime.Object) error {
		rb := existing.(*rbacv1.RoleBinding)

		subjects := []rbacv1.Subject{}
		for _, osUser := range openshiftUsers.Items {
			if userIsOpenshiftAdmin(osUser, openshiftAdminGroup) {
				subjects = append(subjects, rbacv1.Subject{
					APIGroup: "rbac.authorization.k8s.io",
					Name:     osUser.Name,
					Kind:     "User",
				})
			}
		}

		rb.Subjects = subjects
		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	r.logger.Infof("The %s subjects were: %s", adminViewFuseRoleBinding.Name, or)
	return v1alpha1.PhaseCompleted, nil
}

//TODO this should be removed once https://issues.jboss.org/browse/INTLY-2836 is implemented
// We want to avoid this kind of thing as really this is owned by the syndesis operator
func (r *Reconciler) reconcileOauthProxy(ctx context.Context, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	var dcName = "syndesis-oauthproxy"
	dc := &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      dcName,
		},
	}
	if err := client.Get(ctx, pkgclient.ObjectKey{Name: dcName, Namespace: r.Config.GetNamespace()}, dc); err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to get dc for the oauth proxy "+dcName)
	}

	for i, a := range dc.Spec.Template.Spec.Containers[0].Args {
		if strings.Contains(a, "--openshift-sar") {
			args := dc.Spec.Template.Spec.Containers[0].Args
			args[i] = args[0]
			dc.Spec.Template.Spec.Containers[0].Args = args[1:]
			break
		}
	}
	if err := client.Update(ctx, dc); err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to update oauth proxy for fuse")
	}

	return v1alpha1.PhaseCompleted, nil
}

// reconcileCustomResource ensures that the fuse custom resource exists
func (r *Reconciler) reconcileCustomResource(ctx context.Context, install *v1alpha1.Installation, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	st := &v12.Secret{}
	// if this errors, it can be ignored
	err := client.Get(ctx, pkgclient.ObjectKey{Name: "syndesis-global-config", Namespace: r.Config.GetNamespace()}, st)
	if err == nil && string(r.Config.GetProductVersion()) != string(st.Data["syndesis"]) {
		r.Config.SetProductVersion(string(st.Data["syndesis"]))
		r.ConfigManager.WriteConfig(r.Config)
	}

	r.logger.Info("Reconciling fuse custom resource")

	intLimit := 0
	cr := &syn.Syndesis{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      "integreatly",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Syndesis",
			APIVersion: syn.SchemeGroupVersion.String(),
		},
		Spec: syn.SyndesisSpec{
			Integration: syn.IntegrationSpec{
				Limit: &intLimit,
			},
			Components: syn.ComponentsSpec{
				Server: syn.ServerConfiguration{
					Features: syn.ServerFeatures{
						ExposeVia3Scale: true,
					},
				},
			},
		},
	}
	ownerutil.EnsureOwner(cr, install)

	// attempt to create the custom resource
	if err := client.Get(ctx, pkgclient.ObjectKey{Name: cr.Name, Namespace: cr.Namespace}, cr); err != nil {
		if k8serr.IsNotFound(err) {
			if err := client.Create(ctx, cr); err != nil && !k8serr.IsAlreadyExists(err) {
				return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to create a syndesis cr when reconciling custom resource")
			}
			return v1alpha1.PhaseInProgress, nil
		}
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to get a syndesis cr when reconciling custom resource")
	}

	if cr.Status.Phase == syn.SyndesisPhaseStartupFailed {
		return v1alpha1.PhaseFailed, errors.New(fmt.Sprintf("failed to install fuse custom resource: %s", cr.Status.Reason))
	}

	if cr.Status.Phase != syn.SyndesisPhaseInstalled {
		return v1alpha1.PhaseInProgress, nil
	}

	route := &v1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "syndesis",
			Namespace: r.Config.GetNamespace(),
		},
	}

	if err := client.Get(ctx, pkgclient.ObjectKey{Name: route.Name, Namespace: route.Namespace}, route); err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "could not read syndesis route for fuse")
	}

	var url string
	if route.Spec.TLS != nil {
		url = fmt.Sprintf("https://" + route.Spec.Host)
	} else {
		url = fmt.Sprintf("http://" + route.Spec.Host)
	}
	if r.Config.GetHost() != url {
		r.Config.SetHost(url)
		r.ConfigManager.WriteConfig(r.Config)
	}

	// if there are no errors, the phase is complete
	return v1alpha1.PhaseCompleted, nil
}

func userIsOpenshiftAdmin(user usersv1.User, adminGroup *usersv1.Group) bool {
	for _, userName := range adminGroup.Users {
		if user.Name == userName {
			return true
		}
	}

	return false
}
