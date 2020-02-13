package apicurio

import (
	"context"
	"fmt"
	apicurio "github.com/integr8ly/integreatly-operator/pkg/apis/apicur/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	appsv1 "github.com/openshift/api/apps/v1"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"

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
	defaultInstallationNamespace = "apicurio-2"
	defaultSubscriptionName      = "integreatly-apicurio"
	manifestPackage              = "integreatly-apicurio"
	apicurioName                 = "apicurio"
	defaultApicurioPullSecret    = "apicurio-pull-secret"
)

type Reconciler struct {
	Config        *config.Apicurio
	extraParams   map[string]string
	ConfigManager config.ConfigReadWriter
	logger        *logrus.Entry
	mpm           marketplace.MarketplaceInterface
	installation  *integreatlyv1alpha1.Installation
	*resources.Reconciler
	recorder record.EventRecorder
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.Installation, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder) (*Reconciler, error) {
	logger := logrus.NewEntry(logrus.StandardLogger())
	apicurioConfig, err := configManager.ReadApicurio()

	if err != nil {
		return nil, err
	}

	if apicurioConfig.GetNamespace() == "" {
		apicurioConfig.SetNamespace(installation.Spec.NamespacePrefix + defaultInstallationNamespace)
	}

	return &Reconciler{
		Config:        apicurioConfig,
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

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.Installation, product *integreatlyv1alpha1.InstallationProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {

		// Check if namespace is still present before trying to delete it resources
		_, err := resources.GetNS(ctx, r.Config.GetNamespace(), serverClient)
		if !k8serr.IsNotFound(err) {
			phase, err := resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetNamespace())
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
		}
		_, err = resources.GetNS(ctx, r.Config.GetOperatorNamespace(), serverClient)
		if !k8serr.IsNotFound(err) {
			phase, err := resources.RemoveNamespace(ctx, installation, serverClient, r.Config.GetOperatorNamespace())
			if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
				return phase, err
			}
		}

		//if both namespaces are deleted, return complete
		_, operatorNSErr := resources.GetNS(ctx, r.Config.GetOperatorNamespace(), serverClient)
		_, nsErr := resources.GetNS(ctx, r.Config.GetNamespace(), serverClient)
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
		events.HandleError(r.recorder, installation, phase, "Failed to write config in apicurio reconciler", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetNamespace(), installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetNamespace()), err)
		return phase, err
	}
	phase, err = r.ReconcileNamespace(ctx, r.Config.GetOperatorNamespace(), installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetOperatorNamespace()), err)
		return phase, err
	}

	phase, err = r.ReconcilePullSecret(ctx, r.Config.GetNamespace(), defaultApicurioPullSecret, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s pull secret", defaultApicurioPullSecret), err)
		return phase, err
	}

	phase, err = r.ReconcilePullSecret(ctx, r.Config.GetOperatorNamespace(), defaultApicurioPullSecret, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s pull secret", defaultApicurioPullSecret), err)
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, r.Config.GetNamespace(), serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, fmt.Sprintf("Failed to retrieve %s namespace", r.Config.GetNamespace()), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: defaultSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetOperatorNamespace(), ManifestPackage: manifestPackage}, []string{r.Config.GetNamespace()}, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", defaultSubscriptionName), err)
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, installation, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile components", err)
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
	logrus.Infof("%s is successfully reconciled", apicurioName)

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, installation *integreatlyv1alpha1.Installation, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {

	r.logger.Info("Reconciling Apicurio components")
	apicurioCR := &apicurio.Apicurito{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apicurioName,
			Namespace: r.Config.GetNamespace(),
		},
	}

	or, err := controllerutil.CreateOrUpdate(ctx, serverClient, apicurioCR, func() error {
		// Specify 2 pods to provide HA
		apicurioCR.Spec.Size = 2
		apicurioCR.Spec.Image = "registry.redhat.io/fuse7/fuse-apicurito:1.5"
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, errors.Wrap(err, "failed to create/update apicurio custom resource")
	}
	r.logger.Infof("The operation result for apicurio %s was %s", apicurioCR.Name, or)

	phase, err := r.reconcileHost(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile apicurio host"), err)
		return phase, err
	}

	// Create DC for Generated
	err = r.createDeployConfigForGenerator(ctx, serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to create deployment config for generator service"), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Create Service for apicurio generator
	err = r.createServiceForGenerator(ctx, serverClient)
	if err != nil {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to create apicurito generator service"), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	// Create Route for apicurio generator
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
	err = r.addTlsToApicurioRoute(ctx, serverClient)
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
		dc.Spec.Template = &corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Image: "registry.redhat.io/fuse7/fuse-apicurito-generator:1.5",
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
	//
	//DeploymentConfig.apps.openshift.io "fuse-apicurito-generator" is invalid: spec.template.metadata.labels: Invalid value: map[string]string(nil): `selector` does not match template `labels`

	if err != nil {
		return errors.Wrap(err, "failed to create/update apicurio generator dc")
	}
	r.logger.Infof("The operation result for apicurio %s was %s", dc.Name, or)

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
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to check apicurio installation: %w", err)
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

	r.logger.Infof("all apicurio pods ready, returning complete")
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileHost(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	// Setting host on config to exposed route
	logrus.Info("Getting apicurito host")
	apicuritoRoute := &routev1.Route{}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: "apicurio", Namespace: r.Config.GetNamespace()}, apicuritoRoute)
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
		return errors.Wrap(err, "failed to create/update apicurio generator service")
	}
	r.logger.Infof("The operation result for apicurio %s was %s", service.Name, or)

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
		return errors.Wrap(err, "failed to create/update apicurio generator route")
	}
	r.logger.Infof("The operation result for apicurio route %s was %s", route.Name, or)

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
		return errors.Wrap(err, "failed to create/update apicurio generator route")
	}
	r.logger.Infof("The operation result for apicurio config map %s was %s", cfgMap.Name, or)

	return nil
}

func (r *Reconciler) addTlsToApicurioRoute(ctx context.Context, client k8sclient.Client) error {
	apicuritoRoute := &routev1.Route{}
	err := client.Get(ctx, k8sclient.ObjectKey{Name: "apicurio", Namespace: r.Config.GetNamespace()}, apicuritoRoute)
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
		return errors.Wrap(err, "failed to create/update apicurio route")
	}
	r.logger.Infof("The operation result for apicurito route %s was %s", apicuritoRoute.Name, or)

	return nil
}

func (r *Reconciler) updateDeploymentWithConfigMapVolume(ctx context.Context, client k8sclient.Client) error {

	var dc = &appsv1.DeploymentConfig{}
	err := client.Get(ctx, k8sclient.ObjectKey{Name: "apicurio", Namespace: r.Config.GetNamespace()}, dc)
	if err != nil {
		return fmt.Errorf("Failed to update apicurito deployment config: %w", err)
	}

	or, err := controllerutil.CreateOrUpdate(ctx, client, dc, func() error {
		dc.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "config-volume",
				MountPath: "/html/config",
			},
		}
		dc.Spec.Template.Spec.Volumes = []corev1.Volume{
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
		return errors.Wrap(err, "failed to create/update apicurio deployment config")
	}
	r.logger.Infof("The operation result for apicurito deployment config %s was %s", dc.Name, or)

	return nil
}
