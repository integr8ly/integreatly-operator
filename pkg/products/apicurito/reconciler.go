package apicurito

import (
	"context"
	"fmt"
	"strings"

	"github.com/integr8ly/integreatly-operator/pkg/products/monitoring"
	"github.com/integr8ly/integreatly-operator/version"

	k8serr "k8s.io/apimachinery/pkg/api/errors"

	monitoringv1alpha1 "github.com/integr8ly/application-monitoring-operator/pkg/apis/applicationmonitoring/v1alpha1"

	v1 "k8s.io/api/apps/v1"

	apicuritov1alpha1 "github.com/apicurio/apicurio-operators/apicurito/pkg/apis/apicur/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	appsv1 "github.com/openshift/api/apps/v1"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "apicurito"
	manifestPackage              = "integreatly-apicurito"
	apicuritoName                = "apicurito"
	defaultApicuritoPullSecret   = "apicurito-pull-secret"
	size                         = 2
)

type Reconciler struct {
	Config        *config.Apicurito
	extraParams   map[string]string
	ConfigManager config.ConfigReadWriter
	logger        *logrus.Entry
	mpm           marketplace.MarketplaceInterface
	installation  *integreatlyv1alpha1.RHMI
	*resources.Reconciler
	recorder record.EventRecorder
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder) (*Reconciler, error) {
	apicuritoConfig, err := configManager.ReadApicurito()
	if err != nil {
		return nil, fmt.Errorf("Could not retrieve apicurito config : %w", err)
	}

	if apicuritoConfig.GetNamespace() == "" {
		apicuritoConfig.SetNamespace(installation.Spec.NamespacePrefix + defaultInstallationNamespace)
		configManager.WriteConfig(apicuritoConfig)
	}
	if apicuritoConfig.GetOperatorNamespace() == "" {
		if installation.Spec.OperatorsInProductNamespace {
			apicuritoConfig.SetOperatorNamespace(apicuritoConfig.GetOperatorNamespace())
		}
		configManager.WriteConfig(apicuritoConfig)
	}
	apicuritoConfig.SetBlackboxTargetPath("/oauth/healthz")

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		Config:        apicuritoConfig,
		extraParams:   make(map[string]string),
		ConfigManager: configManager,
		logger:        logger,
		installation:  installation,
		mpm:           mpm,
		Reconciler:    resources.NewReconciler(mpm),
		recorder:      recorder,
	}, nil
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return nil
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return version.VerifyProductAndOperatorVersion(
		installation.Status.Stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductApicurito],
		string(integreatlyv1alpha1.VersionApicurito),
		string(integreatlyv1alpha1.OperatorVersionApicurito),
	)
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	operatorNamespace := r.Config.GetOperatorNamespace()
	productNamespace := r.Config.GetNamespace()
	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {

		// Check if namespace is still present before trying to delete it resources
		_, err := resources.GetNS(ctx, productNamespace, serverClient)
		if !k8serr.IsNotFound(err) {
			phase, err := resources.RemoveNamespace(ctx, installation, serverClient, productNamespace)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
		}
		_, err = resources.GetNS(ctx, operatorNamespace, serverClient)
		if !k8serr.IsNotFound(err) {
			phase, err := resources.RemoveNamespace(ctx, installation, serverClient, operatorNamespace)
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
		}

		//if both namespaces are deleted, return complete
		_, operatorNSErr := resources.GetNS(ctx, operatorNamespace, serverClient)
		_, nsErr := resources.GetNS(ctx, productNamespace, serverClient)
		if k8serr.IsNotFound(operatorNSErr) && k8serr.IsNotFound(nsErr) {
			return integreatlyv1alpha1.PhaseCompleted, nil
		}
		return integreatlyv1alpha1.PhaseInProgress, nil
	})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, "Failed to write config in apicurito reconciler", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetNamespace(), installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", productNamespace), err)
		return phase, err
	}
	phase, err = r.ReconcileNamespace(ctx, r.Config.GetOperatorNamespace(), installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", operatorNamespace), err)
		return phase, err
	}

	err = resources.CopyPullSecretToNameSpace(ctx, installation.GetPullSecretSpec(), productNamespace, defaultApicuritoPullSecret, serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s pull secret", defaultApicuritoPullSecret), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	err = resources.CopyPullSecretToNameSpace(ctx, installation.GetPullSecretSpec(), operatorNamespace, defaultApicuritoPullSecret, serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s pull secret", defaultApicuritoPullSecret), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	phase, err = r.reconcileSubscription(ctx, serverClient, installation, productNamespace, operatorNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.ApicuritoSubscriptionName), err)
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile components", err)
		return phase, err
	}

	phase, err = r.newAlertsReconciler().ReconcileAlerts(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile alerts", err)
		return phase, err
	}

	phase, err = r.reconcileBlackboxTargets(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile blackbox targets", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	phase, err = r.handleProgressPhase(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to handle in progress phase", err)
		return phase, err
	}

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	logrus.Infof("%s is successfully reconciled", apicuritoName)

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, installation *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	r.logger.Info("Reconciling Apicurito components")
	apicuritoCR := &apicuritov1alpha1.Apicurito{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apicuritoName,
			Namespace: r.Config.GetNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, apicuritoCR, func() error {
		// Ideally the operator would set the Image field but it currently (operator v1.6) does not - review on upgrades
		apicuritoCR.Spec.Image = "registry.redhat.io/fuse7/fuse-apicurito:1.6"
		// Specify a minimum of 2 pods to provide HA
		if apicuritoCR.Spec.Size < size {
			apicuritoCR.Spec.Size = size
		}

		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create/update apicurito custom resource: %w", err)
	}
	r.logger.Infof("The operation result for apicurito %s was %s", apicuritoCR.Name, or)

	phase, err := r.reconcileHost(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile apicurito host"), err)
		return phase, err
	}

	// Create DC for Generated
	err = r.createDeployConfigForGenerator(ctx, serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to create deployment config for generator service"), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Create Service for apicurito generator
	err = r.createServiceForGenerator(ctx, serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to create apicurito generator service"), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Create Route for apicurito generator
	err = r.createRouteForGenerator(ctx, serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to create apicurito generator route"), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Create ConfigMap for UI
	err = r.createConfigMapForUI(ctx, serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to create apicurito config map"), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Modify Route for Generator (add TLS)
	err = r.addTLSToApicuritoRoute(ctx, serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to apicurito route to have tls"), err)
		return phase, err
	}

	// Modify Deployment for Generator (mount Config Map)
	err = r.updateDeploymentWithConfigMapVolume(ctx, serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to update apicurito deployment config with config map"), err)
		return phase, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) createDeployConfigForGenerator(ctx context.Context, client k8sclient.Client) error {

	var dc = &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fuse-apicurito-generator",
			Namespace: r.Config.GetNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, client, dc, func() error {
		dc.Spec.Selector = map[string]string{
			"app":       "apicurito",
			"component": "fuse-apicurito-generator",
		}
		dc.Spec.Replicas = 1
		dc.Spec.Template = &corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Image: "registry.redhat.io/fuse7/fuse-apicurito-generator:1.6",
					Name:  "fuse-apicurito-generator",
				},
				},
			},
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app":       "apicurito",
					"component": "fuse-apicurito-generator",
				},
			},
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update apicurito generator dc: %w", err)
	}
	r.logger.Infof("The operation result for apicurito %s was %s", dc.Name, or)

	return nil
}

func (r *Reconciler) handleProgressPhase(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.logger.Debug("checking amq streams pods are running")

	pods := &corev1.PodList{}
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(r.Config.GetNamespace()),
	}
	err := client.List(ctx, pods, listOpts...)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to check apicurito installation: %w", err)
	}

	//expecting 2 pods in total
	if len(pods.Items) < 2 {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	//and they should all be ready
checkPodStatus:
	for _, pod := range pods.Items {
		for _, cnd := range pod.Status.Conditions {
			if cnd.Type == corev1.ContainersReady {
				if cnd.Status != corev1.ConditionStatus("True") {
					return integreatlyv1alpha1.PhaseInProgress, nil
				}
				break checkPodStatus
			}
		}
	}

	r.logger.Infof("all apicurito pods ready, returning complete")
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileHost(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// Setting host on config to exposed route
	logrus.Info("Getting apicurito host")
	apicuritoRoute := &routev1.Route{}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: "apicurito", Namespace: r.Config.GetNamespace()}, apicuritoRoute)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get route for apicurito: %w", err)
	}

	r.Config.SetHost("https://" + apicuritoRoute.Spec.Host)
	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not update apicurito config: %w", err)
	}

	logrus.Info("Successfully set apircurito host")

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) createServiceForGenerator(ctx context.Context, client k8sclient.Client) error {
	var service = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fuse-apicurito-generator",
			Namespace: r.Config.GetNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, client, service, func() error {
		service.Spec.Selector = map[string]string{
			"app":       "apicurito",
			"component": "fuse-apicurito-generator",
		}
		service.Spec.Ports = []corev1.ServicePort{
			{
				Protocol: "TCP",
				Port:     80,
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 8080,
				},
			},
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update apicurito generator service: %w", err)
	}
	r.logger.Infof("The operation result for apicurito %s was %s", service.Name, or)

	return nil
}

func (r *Reconciler) createRouteForGenerator(ctx context.Context, client k8sclient.Client) error {
	var route = &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fuse-apicurito-generator",
			Namespace: r.Config.GetNamespace(),
		},
	}

	host := strings.Replace(r.Config.GetHost(), "https://", "", 1)
	host = strings.Replace(host, "http://", "", 1)

	or, err := controllerutil.CreateOrUpdate(ctx, client, route, func() error {
		route.Spec.Host = host
		route.Spec.Path = "/api/v1"
		route.Spec.To = routev1.RouteTargetReference{
			Kind: "Service",
			Name: "fuse-apicurito-generator",
		}
		route.Spec.TLS = &routev1.TLSConfig{
			Termination: "edge",
		}
		route.Spec.WildcardPolicy = "None"
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update apicurito generator route: %w", err)
	}
	r.logger.Infof("The operation result for apicurito route %s was %s", route.Name, or)

	return nil
}

func (r *Reconciler) createConfigMapForUI(ctx context.Context, client k8sclient.Client) error {
	cfgMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "apicurito-ui-config",
			Namespace: r.Config.GetNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, client, cfgMap, func() error {
		cfgMap.Data = map[string]string{
			"config.js": `var ApicuritoConfig = {
					"generators": [
					{
						"name":"Fuse 7.1 Camel Project",
						"url":"/api/v1/generate/camel-project.zip"
					}
				]
			}`,
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create/update apicurito generator route: %w", err)
	}
	r.logger.Infof("The operation result for apicurito config map %s was %s", cfgMap.Name, or)

	return nil
}

func (r *Reconciler) addTLSToApicuritoRoute(ctx context.Context, client k8sclient.Client) error {
	apicuritoRoute := &routev1.Route{}
	err := client.Get(ctx, k8sclient.ObjectKey{Name: "apicurito", Namespace: r.Config.GetNamespace()}, apicuritoRoute)
	if err != nil {
		return fmt.Errorf("Failed to update apicurito route: %w", err)
	}

	or, err := controllerutil.CreateOrUpdate(ctx, client, apicuritoRoute, func() error {
		apicuritoRoute.Spec.TLS = &routev1.TLSConfig{
			Termination: "edge",
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create/update apicurio route: %w", err)
	}
	r.logger.Infof("The operation result for apicurito route %s was %s", apicuritoRoute.Name, or)

	return nil
}

func (r *Reconciler) updateDeploymentWithConfigMapVolume(ctx context.Context, client k8sclient.Client) error {

	var deployment = &v1.Deployment{}
	err := client.Get(ctx, k8sclient.ObjectKey{Name: "apicurito", Namespace: r.Config.GetNamespace()}, deployment)
	if err != nil {
		return fmt.Errorf("Failed to update apicurito deployment config in namespace: %v, %w", r.Config.GetNamespace(), err)
	}

	or, err := controllerutil.CreateOrUpdate(ctx, client, deployment, func() error {
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "config-volume",
				MountPath: "/html/config",
			},
		}
		deployment.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: "config-volume",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "apicurito-ui-config",
						},
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create/update apicurito deployment config: %w", err)
	}
	r.logger.Infof("The operation result for apicurito deployment config %s was %s", deployment.Name, or)

	return nil
}

func (r *Reconciler) reconcileBlackboxTargets(ctx context.Context, installation *integreatlyv1alpha1.RHMI, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	cfg, err := r.ConfigManager.ReadMonitoring()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error reading monitoring config : %w", err)
	}

	err = monitoring.CreateBlackboxTarget(ctx, "integreatly-apicurito", monitoringv1alpha1.BlackboxtargetData{
		Url:     r.Config.GetHost() + r.Config.GetBlackboxTargetPath(),
		Service: "apicurito-ui",
	}, cfg, installation, client)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("error creating apicurito blackbox target: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileSubscription(ctx context.Context, serverClient k8sclient.Client, inst *integreatlyv1alpha1.RHMI, productNamespace string, operatorNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	target := marketplace.Target{
		Pkg:       constants.ApicuritoSubscriptionName,
		Namespace: operatorNamespace,
		Channel:   marketplace.IntegreatlyChannel,
	}
	catalogSourceReconciler := marketplace.NewConfigMapCatalogSourceReconciler(
		manifestPackage,
		serverClient,
		operatorNamespace,
		marketplace.CatalogSourceName,
	)
	return r.Reconciler.ReconcileSubscription(
		ctx,
		target,
		[]string{productNamespace},
		backup.NewNoopBackupExecutor(),
		serverClient,
		catalogSourceReconciler,
	)
}
