package mobiledeveloperconsole

import (
	"context"
	"fmt"

	mdc "github.com/aerogear/mobile-developer-console-operator/pkg/apis/mdc/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	appsv1 "github.com/openshift/api/apps/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "mdc"
	defaultSubscriptionName      = "integreatly-mobile-developer-console"
	resourceName                 = "mobiledeveloperconsole"
	routeResourceName            = "mobiledeveloperconsole-mdc-proxy"
)

type Reconciler struct {
	Config        *config.MobileDeveloperConsole
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
	*resources.Reconciler
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	config, err := configManager.ReadMobileDeveloperConsole()
	if err != nil {
		return nil, errors.Wrap(err, "could not read mobile developer console")
	}

	if config.GetNamespace() == "" {
		config.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}

	err = config.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "mdc config is not valid")
	}

	logger := logrus.WithFields(logrus.Fields{"product": config.GetProductName()})

	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
		mpm:           mpm,
		logger:        logger,
		Reconciler:    resources.NewReconciler(mpm),
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, product *v1alpha1.InstallationProductStatus, client pkgclient.Client) (v1alpha1.StatusPhase, error) {

	phase, err := r.ReconcileNamespace(ctx, r.Config.GetNamespace(), inst, client)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	version, err := resources.NewVersion(v1alpha1.OperatorVersionMDC)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "invalid version number for mdc")
	}
	phase, err = r.ReconcileSubscription(
		ctx,
		inst,
		marketplace.Target{Pkg: defaultSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetNamespace()},
		client,
		version,
	)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, client, inst)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.handleProgress(ctx, client)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()

	r.logger.Infof("%s has reconciled successfully", r.Config.GetProductName())
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, client pkgclient.Client, inst *v1alpha1.Installation) (v1alpha1.StatusPhase, error) {
	r.logger.Info("reconciling mobile-developer-console custom resource")

	clientSecret, err := r.getOauthClientSecret(ctx, client)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	cr := &mdc.MobileDeveloperConsole{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      resourceName,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: mdc.SchemeGroupVersion.String(),
			Kind:       "MobileDeveloperConsole",
		},
		Spec: mdc.MobileDeveloperConsoleSpec{
			OAuthClientId:     inst.Spec.NamespacePrefix + string(r.Config.GetProductName()),
			OAuthClientSecret: string(clientSecret),
		},
	}

	//  creates the custom resource
	if _, err := controllerutil.CreateOrUpdate(ctx, client, cr, func(existing runtime.Object) error {
		return nil
	}); err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to get or create a mdc custom resource")
	}

	// change the OPENSHIFT_HOST addess to the cluster master URL
	dc := &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      resourceName,
		},
	}
	if err := client.Get(ctx, pkgclient.ObjectKey{Name: resourceName, Namespace: r.Config.GetNamespace()}, dc); err != nil {
		return v1alpha1.PhaseInProgress, errors.Wrap(err, "dc isn't available yet for "+resourceName)
	}
	for i, container := range dc.Spec.Template.Spec.Containers {
		if container.Name == defaultInstallationNamespace {
			for j, env := range container.Env {
				if env.Name == "OPENSHIFT_HOST" {
					dc.Spec.Template.Spec.Containers[i].Env[j].Value = inst.Spec.MasterURL
					break
				}
			}
		}
	}
	if err := client.Update(ctx, dc); err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to update dc for mdc")
	}

	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeResourceName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	if err := client.Get(ctx, pkgclient.ObjectKey{Name: route.Name, Namespace: route.Namespace}, route); err != nil {
		return v1alpha1.PhaseInProgress, errors.Wrap(err, "could not read mdc route")
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

	phase, err := r.ReconcileOauthClient(ctx, inst, &oauthv1.OAuthClient{
		ObjectMeta: metav1.ObjectMeta{
			Name: inst.Spec.NamespacePrefix + string(r.Config.GetProductName()),
		},
		Secret: clientSecret,
		RedirectURIs: []string{
			r.Config.GetHost(),
		},
		GrantMethod: oauthv1.GrantHandlerAuto,
	}, client)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	// if there are no errors, the phase is complete
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) getOauthClientSecret(ctx context.Context, serverClient pkgclient.Client) (string, error) {

	oauthClientSecrets := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.ConfigManager.GetOauthClientsSecretName(),
		},
	}

	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: oauthClientSecrets.Name, Namespace: r.ConfigManager.GetOperatorNamespace()}, oauthClientSecrets)
	if err != nil {
		return "", errors.Wrapf(err, "Could not find %s Secret", oauthClientSecrets.Name)
	}

	clientSecretBytes, ok := oauthClientSecrets.Data[string(r.Config.GetProductName())]
	if !ok {
		return "", errors.Wrapf(err, "Could not find %s key in %s Secret", string(r.Config.GetProductName()), oauthClientSecrets.Name)
	}
	return string(clientSecretBytes), nil
}

func (r *Reconciler) handleProgress(ctx context.Context, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Info("checking status of mdc cr")

	mdcCR := &mdc.MobileDeveloperConsole{}

	if err := client.Get(ctx, pkgclient.ObjectKey{Name: resourceName, Namespace: r.Config.GetNamespace()}, mdcCR); err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to get mdc cr while reconciling custom resource")
	}

	if mdcCR.Status.Phase != mdc.PhaseComplete {
		return v1alpha1.PhaseInProgress, nil
	}

	r.logger.Infof("all pods ready, mdc complete")
	return v1alpha1.PhaseCompleted, nil
}
