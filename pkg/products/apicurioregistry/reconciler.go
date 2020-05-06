package apicurioregistry

import (
	"context"
	"fmt"

	apicurioregistry "github.com/Apicurio/apicurio-registry-operator/pkg/apis/apicur/v1alpha1"
	kafkav1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis-products/kafka.strimzi.io/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/products/amqstreams"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	appsv1 "github.com/openshift/api/apps/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultInstallationNamespace = "apicurio-registry"
	manifestPackage              = "integreatly-apicurio-registry"
	replicas                     = 1
	amqStreamsTopicPartitions    = 3
	amqStreamsTopicReplicas      = 3
	amqStreamsTopicCleanupPolicy = "compact"
)

// Reconciler reconciles everything needed to install Apicurio Registry. The resources that it works
// with are considered secondary resources in the context of the installation controller.
type Reconciler struct {
	Config        *config.ApicurioRegistry
	ConfigManager config.ConfigReadWriter
	logger        *logrus.Entry
	mpm           marketplace.MarketplaceInterface
	*resources.Reconciler
	recorder record.EventRecorder
}

// NewReconciler creates a new Reconciler.
func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder) (*Reconciler, error) {
	config, err := configManager.ReadApicurioRegistry()
	if err != nil {
		return nil, fmt.Errorf("could not read apicurio registry config: %w", err)
	}

	if config.GetNamespace() == "" {
		config.SetNamespace(installation.Spec.NamespacePrefix + defaultInstallationNamespace)
	}
	if config.GetOperatorNamespace() == "" {
		if installation.Spec.OperatorsInProductNamespace {
			config.SetOperatorNamespace(config.GetNamespace())
		} else {
			config.SetOperatorNamespace(config.GetNamespace() + "-operator")
		}
	}

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		Config:        config,
		ConfigManager: configManager,
		logger:        logger,
		mpm:           mpm,
		Reconciler:    resources.NewReconciler(mpm),
		recorder:      recorder,
	}, nil
}

// GetPreflightObject returns an object that will be checked in the preflight checks in the main
// Installation controller to ensure there isn't a conflicting Camel K installation.
func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "apicurio-registry-operator",
			Namespace: ns,
		},
	}
}

// Reconcile changes the current state to match the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileFinalizer(ctx, client, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		phase, err := resources.RemoveNamespace(ctx, installation, client, r.Config.GetNamespace())
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}
		phase, err = resources.RemoveNamespace(ctx, installation, client, r.Config.GetOperatorNamespace())
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}
		return integreatlyv1alpha1.PhaseCompleted, nil
	})
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetOperatorNamespace(), installation, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetOperatorNamespace()), err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, r.Config.GetNamespace(), installation, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", r.Config.GetNamespace()), err)
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, r.Config.GetNamespace(), client)
	if err != nil {
		events.HandleError(r.recorder, installation, integreatlyv1alpha1.PhaseFailed, fmt.Sprintf("Failed to retrieve %s namespace", r.Config.GetNamespace()), err)
		return integreatlyv1alpha1.PhaseFailed, err
	}

	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: constants.ApicurioRegistrySubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetOperatorNamespace(), ManifestPackage: manifestPackage}, []string{r.Config.GetNamespace()}, backup.NewNoopBackupExecutor(), client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.ApicurioRegistrySubscriptionName), err)
		return phase, err
	}

	phase, err = r.reconcileStorage(ctx, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile storage", err)
		return phase, err
	}

	phase, err = r.reconcileCustomResource(ctx, installation, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile custom resource", err)
		return phase, err
	}

	phase, err = r.handleProgressPhase(ctx, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to handle in progress phase", err)
		return phase, err
	}

	phase, err = r.reconcileConfig(ctx, client)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile config", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	logrus.Infof("%s is successfully reconciled", r.Config.GetProductName())

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileStorage(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	amqStreams, err := r.ConfigManager.ReadAMQStreams()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to read AMQ Streams config: %s", err)
	}

	err = createKafkaTopic(ctx, client, "storage-topic", amqStreams.GetNamespace())
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create storage topic: %w", err)
	}

	err = createKafkaTopic(ctx, client, "global-id-topic", amqStreams.GetNamespace())
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create global id topic: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func createKafkaTopic(ctx context.Context, client k8sclient.Client, name string, namespace string) error {
	kafkaTopic := &kafkav1alpha1.KafkaTopic{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"strimzi.io/cluster": amqstreams.ClusterName,
			},
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, kafkaTopic, func() error {
		kafkaTopic.Spec.Partitions = amqStreamsTopicPartitions
		kafkaTopic.Spec.Replicas = amqStreamsTopicReplicas
		kafkaTopic.Spec.Config = map[string]string{
			"cleanup.policy": amqStreamsTopicCleanupPolicy,
		}
		return nil
	})

	return err
}

// ReconcileCustomResource creates/updates the ApicurioRegistry custom resource
func (r *Reconciler) reconcileCustomResource(ctx context.Context, installation *integreatlyv1alpha1.RHMI, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	amqStreams, err := r.ConfigManager.ReadAMQStreams()
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	apicurioRegistry := &apicurioregistry.ApicurioRegistry{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      string(r.Config.GetProductName()),
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, client, apicurioRegistry, func() error {
		apicurioRegistry.Spec.Configuration.Persistence = "streams"
		apicurioRegistry.Spec.Configuration.Streams.ApplicationId = string(r.Config.GetProductName())
		apicurioRegistry.Spec.Configuration.Streams.BootstrapServers = amqStreams.GetHost()
		apicurioRegistry.Spec.Image.Name = fmt.Sprintf("apicurio/apicurio-registry-streams:%v", integreatlyv1alpha1.VersionApicurioRegistry)

		if apicurioRegistry.Spec.Deployment.Replicas < replicas {
			apicurioRegistry.Spec.Deployment.Replicas = replicas
		}

		owner.AddIntegreatlyOwnerAnnotations(apicurioRegistry, installation)
		return nil
	})
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to reconcile ApicurioRegistry resource: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) handleProgressPhase(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.logger.Debug("checking service registry replicas")

	cr := &apicurioregistry.ApicurioRegistry{}
	err := client.Get(ctx, k8sclient.ObjectKey{Name: string(r.Config.GetProductName()), Namespace: r.Config.GetNamespace()}, cr)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get ApicurioRegistry CR: %w", err)
	}

	if cr.Status.ReplicaCount < replicas {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	r.logger.Infof("service registry replicas ready, returning complete")
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileConfig(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.logger.Infof("reconciling config")

	cr := &apicurioregistry.ApicurioRegistry{}
	err := client.Get(ctx, k8sclient.ObjectKey{Name: string(r.Config.GetProductName()), Namespace: r.Config.GetNamespace()}, cr)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get ApicurioRegistry CR: %w", err)
	}

	r.Config.SetHost(cr.Status.Host)

	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to persist config: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}
