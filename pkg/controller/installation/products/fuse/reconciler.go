package fuse

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	syn "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1alpha1"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/monitoring"
	"github.com/integr8ly/integreatly-operator/pkg/resources"

	appsv1 "github.com/openshift/api/apps/v1"
	v13 "github.com/openshift/api/image/v1"
	v1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "fuse"
	defaultSubscriptionName      = "integreatly-syndesis"
	defaultFusePullSecret        = "syndesis-pull-secret"
	developersGroupName          = "rhmi-developers"
	clusterViewRoleName          = "view"
	manifestPackage              = "integreatly-fuse"
)

type Reconciler struct {
	*resources.Reconciler
	coreClient    kubernetes.Interface
	Config        *config.Fuse
	ConfigManager config.ConfigReadWriter
	extraParams   map[string]string
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	fuseConfig, err := configManager.ReadFuse()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve fuse config: %w", err)
	}

	if fuseConfig.GetNamespace() == "" {
		fuseConfig.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}
	if err = fuseConfig.Validate(); err != nil {
		return nil, fmt.Errorf("fuse config is not valid: %w", err)
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

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetNamespace(), inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.ReconcilePullSecret(ctx, r.Config.GetNamespace(), defaultFusePullSecret, inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileViewFusePerms(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, r.Config.GetNamespace(), serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: defaultSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetNamespace(), ManifestPackage: manifestPackage}, r.Config.GetNamespace(), serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileImageVersion(ctx, inst, serverClient)
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

	phase, err = r.reconcileTemplates(ctx, inst, serverClient)
	logrus.Infof("Phase: %s reconcileTemplates", phase)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		logrus.Infof("Error: %s", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	logrus.Infof("%s has reconciled successfully", r.Config.GetProductName())
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

func (r *Reconciler) reconcileTemplates(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	// Interate over template_list
	for _, template := range r.Config.GetTemplateList() {
		// create it
		_, err := r.createResource(ctx, inst, template, serverClient)
		if err != nil {
			return v1alpha1.PhaseFailed, fmt.Errorf("failed to create/update monitoring template %s: %w", template, err)
		}
		logrus.Infof("Reconciling the monitoring template %s was successful", template)
	}
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileImageVersion(ctx context.Context, install *v1alpha1.Installation, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Info("FUSE POSTGRES: reconciling postgres version")
	dc := &appsv1.DeploymentConfig{}
	err := client.Get(ctx, pkgclient.ObjectKey{
		Namespace: r.Config.GetNamespace(),
		Name:      "syndesis-db",
	}, dc)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return v1alpha1.PhaseCompleted, nil
		}
		r.logger.Info("FUSE POSTGRES: error getting DC: " + err.Error())
		return v1alpha1.PhaseFailed, fmt.Errorf("error retrieving syndesis-db deployment config: %w", err)
	}

	for i, trigger := range dc.Spec.Triggers {
		if trigger.ImageChangeParams != nil {
			if trigger.ImageChangeParams.From.Name == "postgresql:9.5" {
				//found old image, update DC
				_, err = controllerutil.CreateOrUpdate(ctx, client, dc, func() error {
					dc.Spec.Triggers[i].ImageChangeParams.From.Name = "postgresql:9.6"
					return nil
				})
				if err != nil {
					return v1alpha1.PhaseFailed, fmt.Errorf("error updating postgres image to 9.6: %w", err)
				}
			}
		}
	}

	is := &v13.ImageStream{}
	err = client.Get(ctx, pkgclient.ObjectKey{Name: "fuse-komodo-server", Namespace: r.Config.GetNamespace()}, is)

	for i, tag := range is.Spec.Tags {
		if tag.Name == "latest" && tag.From.Name != "registry.redhat.io/fuse7-tech-preview/data-virtualization-server-rhel7:1.4" {
			_, err = controllerutil.CreateOrUpdate(ctx, client, is, func() error {
				is.Spec.Tags[i].From.Name = "registry.redhat.io/fuse7-tech-preview/data-virtualization-server-rhel7:1.4"
				return nil
			})
			if err != nil {
				return v1alpha1.PhaseFailed, fmt.Errorf("error updating komodo server image to 1.4: %w", err)
			}
			return v1alpha1.PhaseCompleted, nil
		}
		return v1alpha1.PhaseCompleted, nil
	}

	return v1alpha1.PhaseFailed, errors.New("Could not find trigger for postgres:9.5 in deploymentconfig")
}

// Ensures all users in rhmi-developers group have view Fuse permissions
func (r *Reconciler) reconcileViewFusePerms(ctx context.Context, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Infof("Reconciling view Fuse permissions for %s group on %s namespace", developersGroupName, r.Config.GetNamespace())

	openshiftUsers := &usersv1.UserList{}
	err := client.List(ctx, openshiftUsers)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	rhmiDevelopersGroup := &usersv1.Group{}
	err = client.Get(ctx, pkgclient.ObjectKey{Name: developersGroupName}, rhmiDevelopersGroup)
	if err != nil && !k8serr.IsNotFound(err) {
		return v1alpha1.PhaseFailed, err
	}

	viewFuseRoleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      developersGroupName + "-fuse-view",
			Namespace: r.Config.GetNamespace(),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterViewRoleName,
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, client, viewFuseRoleBinding, func() error {
		subjects := []rbacv1.Subject{}
		for _, osUser := range openshiftUsers.Items {
			if groupContainsUser(osUser, rhmiDevelopersGroup) {
				subjects = append(subjects, rbacv1.Subject{
					APIGroup: "rbac.authorization.k8s.io",
					Name:     osUser.Name,
					Kind:     "User",
				})
			}
		}

		viewFuseRoleBinding.Subjects = subjects
		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	r.logger.Infof("The %s subjects were: %s", viewFuseRoleBinding.Name, or)
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
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to get dc for the oauth proxy %v: %w", dcName, err)
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
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to update oauth proxy for fuse: %w", err)
	}

	return v1alpha1.PhaseCompleted, nil
}

// reconcileCustomResource ensures that the fuse custom resource exists
func (r *Reconciler) reconcileCustomResource(ctx context.Context, install *v1alpha1.Installation, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	st := &corev1.Secret{}
	// if this errors, it can be ignored
	err := client.Get(ctx, pkgclient.ObjectKey{Name: "syndesis-global-config", Namespace: r.Config.GetNamespace()}, st)
	if err == nil && string(r.Config.GetProductVersion()) != string(st.Data["syndesis"]) {
		r.Config.SetProductVersion(string(st.Data["syndesis"]))
		r.ConfigManager.WriteConfig(r.Config)
	}
	if err == nil && string(r.Config.GetOperatorVersion()) != string(v1alpha1.OperatorVersionFuse) {
		r.Config.SetOperatorVersion(string(v1alpha1.OperatorVersionFuse))
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

	// attempt to create the custom resource
	if err := client.Get(ctx, pkgclient.ObjectKey{Name: cr.Name, Namespace: cr.Namespace}, cr); err != nil {
		if k8serr.IsNotFound(err) {
			if err := client.Create(ctx, cr); err != nil && !k8serr.IsAlreadyExists(err) {
				return v1alpha1.PhaseFailed, fmt.Errorf("failed to create a syndesis cr when reconciling custom resource: %w", err)
			}
			return v1alpha1.PhaseInProgress, nil
		}
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to get a syndesis cr when reconciling custom resource: %w", err)
	}

	if cr.Status.Phase == syn.SyndesisPhaseStartupFailed {
		return v1alpha1.PhaseFailed, fmt.Errorf("failed to install fuse custom resource: %s", cr.Status.Reason)
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
		return v1alpha1.PhaseFailed, fmt.Errorf("could not read syndesis route for fuse: %w", err)
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

func groupContainsUser(user usersv1.User, group *usersv1.Group) bool {
	for _, userName := range group.Users {
		if user.Name == userName {
			return true
		}
	}

	return false
}
