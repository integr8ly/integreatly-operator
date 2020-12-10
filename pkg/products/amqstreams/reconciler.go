package amqstreams

import (
	"context"
	"fmt"

	"github.com/integr8ly/integreatly-operator/pkg/resources/constants"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	kafkav1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis-products/kafka.strimzi.io/v1alpha1"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/backup"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	defaultInstallationNamespace = "amq-streams"
	// ClusterName is the Kafka cluster name
	ClusterName                          = "rhmi-cluster"
	manifestPackage                      = "integreatly-amq-streams"
	kafkaReplicas                        = 3
	zookeeperReplicas                    = 3
	entityOperatorReplicas               = 1
	offsetsTopicReplicationFactor        = "3"
	transactionStateLogReplicationFactor = "3"
)

type Reconciler struct {
	Config        *config.AMQStreams
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	log           l.Logger
	*resources.Reconciler
	recorder record.EventRecorder
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, log l.Logger) (*Reconciler, error) {
	config, err := configManager.ReadAMQStreams()
	if err != nil {
		return nil, fmt.Errorf("could not read amq streams config: %w", err)
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

	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
		mpm:           mpm,
		log:           log,
		Reconciler:    resources.NewReconciler(mpm),
		recorder:      recorder,
	}, nil
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "amq-streams-cluster-operator",
			Namespace: ns,
		},
	}
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return true
}

// Reconcile reads that state of the cluster for amq streams and makes changes based on the state read
// and what is required
func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	operatorNamespace := r.Config.GetOperatorNamespace()
	productNamespace := r.Config.GetNamespace()
	phase, err := r.ReconcileFinalizer(ctx, serverClient, installation, string(r.Config.GetProductName()), func() (integreatlyv1alpha1.StatusPhase, error) {
		phase, err := resources.RemoveNamespace(ctx, installation, serverClient, productNamespace, r.log)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}
		phase, err = resources.RemoveNamespace(ctx, installation, serverClient, operatorNamespace, r.log)
		if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
			return phase, err
		}
		return integreatlyv1alpha1.PhaseCompleted, nil
	}, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile finalizer", err)
		return phase, err
	}

	phase, err = r.ReconcileNamespace(ctx, operatorNamespace, installation, serverClient, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", operatorNamespace), err)
		return phase, err
	}

	ns := r.Config.GetNamespace()
	phase, err = r.ReconcileNamespace(ctx, ns, installation, serverClient, r.log)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s namespace", ns), err)
		return phase, err
	}

	phase, err = r.reconcileSubscription(ctx, serverClient, installation, productNamespace, operatorNamespace)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, fmt.Sprintf("Failed to reconcile %s subscription", constants.AMQStreamsSubscriptionName), err)
		return phase, err
	}

	phase, err = r.handleCreatingComponents(ctx, serverClient, installation)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to create components", err)
		return phase, err
	}

	phase, err = r.handleProgressPhase(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to handle in progress phase", err)
		return phase, err
	}

	phase, err = r.setHostInConfig(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to share host", err)
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	r.log.Infof("Reconciled successfully", l.Fields{"productName": r.Config.GetProductName()})
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) handleCreatingComponents(ctx context.Context, client k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Debug("reconciling amq streams custom resource")

	kafka := &kafkav1alpha1.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ClusterName,
			Namespace: r.Config.GetNamespace(),
		},
	}

	// attempt to create or update the custom resource
	_, err := controllerutil.CreateOrUpdate(ctx, client, kafka, func() error {
		kafka.APIVersion = fmt.Sprintf("%s/%s", kafkav1alpha1.SchemeGroupVersion.Group, kafkav1alpha1.SchemeGroupVersion.Version)
		kafka.Kind = kafkav1alpha1.KafkaKind

		kafka.Name = ClusterName
		kafka.Namespace = r.Config.GetNamespace()

		kafka.Spec.Kafka.Version = "2.1.1"
		if kafka.Spec.Kafka.Replicas < kafkaReplicas {
			kafka.Spec.Kafka.Replicas = kafkaReplicas
		}
		kafka.Spec.Kafka.Listeners = map[string]kafkav1alpha1.KafkaListener{
			"plain": {},
			"tls":   {},
		}
		kafka.Spec.Kafka.Config.OffsetsTopicReplicationFactor = offsetsTopicReplicationFactor
		kafka.Spec.Kafka.Config.TransactionStateLogReplicationFactor = transactionStateLogReplicationFactor
		kafka.Spec.Kafka.Config.TransactionStateLogMinIsr = "2"
		kafka.Spec.Kafka.Config.LogMessageFormatVersion = "2.1"
		kafka.Spec.Kafka.Storage.Type = "persistent-claim"
		kafka.Spec.Kafka.Storage.Size = "10Gi"
		kafka.Spec.Kafka.Storage.DeleteClaim = false

		if kafka.Spec.Zookeeper.Replicas < zookeeperReplicas {
			kafka.Spec.Zookeeper.Replicas = zookeeperReplicas
		}
		kafka.Spec.Zookeeper.Storage.Type = "persistent-claim"
		kafka.Spec.Zookeeper.Storage.Size = "10Gi"
		kafka.Spec.Zookeeper.Storage.DeleteClaim = false

		kafka.Spec.EntityOperator.TopicOperator = kafkav1alpha1.KafkaTopicOperator{}
		kafka.Spec.EntityOperator.UserOperator = kafkav1alpha1.KafkaUserOperator{}

		owner.AddIntegreatlyOwnerAnnotations(kafka, installation)
		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get or create a kafka custom resource: %w", err)
	}

	// if there are no errors, the phase is complete
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) handleProgressPhase(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Debug("checking amq streams pods are running")

	pods := &corev1.PodList{}
	listOpts := []k8sclient.ListOption{
		k8sclient.InNamespace(r.Config.GetNamespace()),
	}
	err := client.List(ctx, pods, listOpts...)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to check amq streams installation: %w", err)
	}

	minExpectedPods := kafkaReplicas + zookeeperReplicas + entityOperatorReplicas
	if len(pods.Items) < minExpectedPods {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	//and they should all be ready
	if !r.allPodsReady(pods) {
		return integreatlyv1alpha1.PhaseInProgress, nil
	}

	r.log.Info("all pods ready, returning complete")
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) allPodsReady(pods *corev1.PodList) bool {
	for _, pod := range pods.Items {
		for _, cnd := range pod.Status.Conditions {
			if cnd.Type == corev1.ContainersReady {
				if cnd.Status != corev1.ConditionStatus("True") {
					return false
				}
			}
		}
	}
	return true
}

// setHostInConfig sets the Kafka Bootstrap service address into the config object
func (r *Reconciler) setHostInConfig(ctx context.Context, client k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	service := &corev1.Service{}

	err := client.Get(ctx, k8sclient.ObjectKey{Name: "rhmi-cluster-kafka-bootstrap", Namespace: r.Config.GetNamespace()}, service)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to read amq streams bootstrap server service: %w", err)
	}

	r.Config.SetHost(fmt.Sprintf("%s.%s.svc:9092", service.GetName(), service.GetNamespace()))
	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("could not persist config: %w", err)
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileSubscription(ctx context.Context, serverClient k8sclient.Client, inst *integreatlyv1alpha1.RHMI, productNamespace string, operatorNamespace string) (integreatlyv1alpha1.StatusPhase, error) {
	target := marketplace.Target{
		Pkg:       constants.AMQStreamsSubscriptionName,
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
		r.log,
	)
}
