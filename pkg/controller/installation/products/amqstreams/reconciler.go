package amqstreams

import (
	"context"
	"errors"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	kafkav1 "github.com/integr8ly/integreatly-operator/pkg/apis/kafka.strimzi.io/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	errors2 "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	defaultInstallationNamespace = "amq-streams"
	defaultSubscriptionName      = "amq-streams"
)

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	config, err := configManager.ReadAMQStreams()
	if err != nil {
		return nil, err
	}
	if config.GetNamespace() == "" {
		config.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}
	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
		mpm:           mpm,
		Reconciler:    resources.NewReconciler(mpm),
	}, nil
}

type Reconciler struct {
	Config        *config.AMQStreams
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	*resources.Reconciler
}

func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	phase := inst.Status.ProductStatus[r.Config.GetProductName()]
	switch v1alpha1.StatusPhase(phase) {
	case v1alpha1.PhaseNone:
		return r.reconcileNamespace(ctx, r.Config.GetNamespace(), inst, serverClient)
	case v1alpha1.PhaseCreatingSubscription, v1alpha1.PhaseAwaitingOperator:
		return r.handleCreatingSubscription(ctx, inst, r.Config.GetNamespace(), serverClient)
	case v1alpha1.PhaseCreatingComponents:
		return r.handleCreatingComponents(ctx, serverClient, inst)
	case v1alpha1.PhaseInProgress:
		return r.handleProgressPhase(ctx, serverClient)
	case v1alpha1.PhaseCompleted:
		return v1alpha1.PhaseCompleted, nil
	case v1alpha1.PhaseFailed:
		//potentially do a dump and recover in the future
		return v1alpha1.PhaseFailed, errors.New("installation of AMQ Streams failed")
	default:
		return v1alpha1.PhaseFailed, errors.New("Unknown phase for AMQ Streams: " + string(phase))
	}
}

// TODO this reconciler needs a refactor working around the problem for now (we shouldn't need to wrap the reconcile namespace call)
func (r *Reconciler) reconcileNamespace(ctx context.Context, ns string, inst *v1alpha1.Installation, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileNamespace(ctx, ns, inst, client)
	if err != nil {
		return v1alpha1.PhaseFailed, errors2.Wrap(err, "failed to reconcile namespace for amq streams")
	}
	if phase == v1alpha1.PhaseCompleted {
		return v1alpha1.PhaseCreatingSubscription, nil
	}
	return phase, err
}

// TODO this reconciler needs a refactor working around the problem for now (we shouldn't need to wrap the ReconcileSubscription namespace call)
func (r *Reconciler) handleCreatingSubscription(ctx context.Context, inst *v1alpha1.Installation, ns string, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileSubscription(ctx, inst, defaultSubscriptionName, ns, client)
	if err != nil {
		return v1alpha1.PhaseFailed, errors2.Wrap(err, "failed to reconcile subscription for amq streams")
	}
	if phase == v1alpha1.PhaseCompleted {
		return v1alpha1.PhaseCreatingComponents, nil
	}
	return phase, nil
}

func (r *Reconciler) handleCreatingComponents(ctx context.Context, serverClient pkgclient.Client, inst *v1alpha1.Installation) (v1alpha1.StatusPhase, error) {
	kafka := &kafkav1.Kafka{
		TypeMeta: metav1.TypeMeta{
			APIVersion: fmt.Sprintf(
				"%s/%s",
				kafkav1.SchemeGroupVersion.Group,
				kafkav1.SchemeGroupVersion.Version),
			Kind: kafkav1.KafkaKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "integreatly-cluster",
			Namespace: r.Config.GetNamespace(),
		},
		Spec: kafkav1.KafkaSpec{
			Kafka: kafkav1.KafkaSpecKafka{
				Version:  "2.1.1",
				Replicas: 3,
				Listeners: map[string]kafkav1.KafkaListener{
					"plain": {},
					"tls":   {},
				},
				Config: kafkav1.KafkaSpecKafkaConfig{
					OffsetsTopicReplicationFactor:        "3",
					TransactionStateLogReplicationFactor: "3",
					TransactionStateLogMinIsr:            "2",
					LogMessageFormatVersion:              "2.1",
				},
				Storage: kafkav1.KafkaStorage{
					Type:        "persistent-claim",
					Size:        "10Gi",
					DeleteClaim: false,
				},
			},
			Zookeeper: kafkav1.KafkaSpecZookeeper{
				Replicas: 3,
				Storage: kafkav1.KafkaStorage{
					Type:        "persistent-claim",
					Size:        "10Gi",
					DeleteClaim: false,
				},
			},
			EntityOperator: kafkav1.KafkaSpecEntityOperator{
				TopicOperator: kafkav1.KafkaTopicOperator{},
				UserOperator:  kafkav1.KafkaUserOperator{},
			},
		},
	}
	ownerutil.EnsureOwner(kafka, inst)
	err := serverClient.Create(ctx, kafka)
	if err != nil {
		return v1alpha1.PhaseCreatingComponents, errors2.Wrap(err, "error creating kafka CR")
	}
	return v1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) handleProgressPhase(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	// check AMQ Streams is in ready state
	pods := &v1.PodList{}
	err := serverClient.List(ctx, &pkgclient.ListOptions{Namespace: r.Config.GetNamespace()}, pods)
	if err != nil {
		return v1alpha1.PhaseFailed, errors2.Wrap(err, "Failed to check AMQ Streams installation")
	}

	//expecting 8 pods in total
	if len(pods.Items) < 8 {
		return v1alpha1.PhaseInProgress, nil
	}

	//and they should all be ready
checkPodStatus:
	for _, pod := range pods.Items {
		for _, cnd := range pod.Status.Conditions {
			if cnd.Type == v1.ContainersReady {
				if cnd.Status != v1.ConditionStatus("True") {
					logrus.Infof("pod not ready, returning in progress: %+v", cnd.Status)
					return v1alpha1.PhaseInProgress, nil
				}
				break checkPodStatus
			}
		}
	}
	logrus.Infof("All pods ready, returning complete")
	return v1alpha1.PhaseCompleted, nil
}
