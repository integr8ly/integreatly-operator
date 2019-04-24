package amqstreams

import (
	"context"
	"errors"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	kafkav1 "github.com/integr8ly/integreatly-operator/pkg/apis/kafka.strimzi.io/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	defaultInstallationNamespace = "amq-streams"
)

func NewReconciler(client pkgclient.Client, configManager config.ConfigReadWriter, instance *v1alpha1.Installation, clusterHasOLM bool) (*Reconciler, error) {
	config, err := configManager.ReadAMQStreams()
	if err != nil {
		return nil, err
	}
	if config.GetNamespace() == "" {
		config.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}
	var mpm marketplace.MarketplaceInterface
	if clusterHasOLM {
		mpm = marketplace.NewManager(client)
	}
	return &Reconciler{client: client,
		ConfigManager: configManager,
		Config:        config,
		mpm:           mpm,
	}, nil
}

type Reconciler struct {
	client        pkgclient.Client
	Config        *config.AMQStreams
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
}

func (r *Reconciler) Reconcile(phase v1alpha1.StatusPhase) (v1alpha1.StatusPhase, error) {
	switch phase {
	case v1alpha1.PhaseNone:
		return r.handleNoPhase()
	case v1alpha1.PhaseAwaitingNS:
		return r.handleAwaitingNSPhase()
	case v1alpha1.PhaseCreatingSubscription:
		return r.handleCreatingSubscription()
	case v1alpha1.PhaseAwaitingSubscription:
		return r.handleAwaitingSubscription()
	case v1alpha1.PhaseCreatingComponents:
		return r.handleCreatingComponents()
	case v1alpha1.PhaseInProgress:
		return r.handleProgressPhase()
	case v1alpha1.PhaseCompleted:
		return v1alpha1.PhaseCompleted, nil
	case v1alpha1.PhaseFailed:
		//potentially do a dump and recover in the future
		return v1alpha1.PhaseFailed, errors.New("installation of AMQ Streams failed")
	default:
		return v1alpha1.PhaseFailed, errors.New("Unknown phase for AMQ Streams: " + string(phase))
	}
}

func (r *Reconciler) handleNoPhase() (v1alpha1.StatusPhase, error) {
	logrus.Infof("amq streams no phase")

	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      r.Config.GetNamespace(),
		},
	}
	err := r.client.Create(context.TODO(), ns)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, err
	}
	return v1alpha1.PhaseAwaitingNS, nil
}

func (r *Reconciler) handleAwaitingNSPhase() (v1alpha1.StatusPhase, error) {
	logrus.Infof("waiting for namespace to be active")
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Config.GetNamespace(),
		},
	}
	err := r.client.Get(context.TODO(), pkgclient.ObjectKey{Name: r.Config.GetNamespace()}, ns)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	if ns.Status.Phase == v1.NamespaceActive {
		// 23/04/19 pbrookes: if mpm is nil we are not in an OLM environment, so do not create a subscription
		//instead skip to creating components and assume operator is set up already
		if r.mpm != nil {
			return v1alpha1.PhaseCreatingSubscription, nil
		} else {
			return v1alpha1.PhaseCreatingComponents, nil
		}
	}

	return v1alpha1.PhaseAwaitingNS, nil
}

func (r *Reconciler) handleCreatingSubscription() (v1alpha1.StatusPhase, error) {
	logrus.Infof("amq streams accepted phase")

	err := r.mpm.CreateSubscription(
		marketplace.GetOperatorSources().Redhat,
		r.Config.GetNamespace(),
		"amq-streams",
		"final",
		[]string{r.Config.GetNamespace()},
		coreosv1alpha1.ApprovalAutomatic)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, err
	}

	return v1alpha1.PhaseAwaitingSubscription, nil
}

func (r *Reconciler) handleAwaitingSubscription() (v1alpha1.StatusPhase, error) {
	//wait to see kafka CRD exists

	return v1alpha1.PhaseAwaitingSubscription, nil
}

func (r *Reconciler) handleCreatingComponents() (v1alpha1.StatusPhase, error) {
	// commented out properties are for 1.1.0
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
				//Version: "2.1.1",
				Replicas: 3,
				Listeners: map[string]kafkav1.KafkaListener{
					"plain": kafkav1.KafkaListener{},
					"tls":   kafkav1.KafkaListener{},
				},
				Config: kafkav1.KafkaSpecKafkaConfig{
					OffsetsTopicReplicationFactor: "3",
					//LogMessageFormatVersion: "2.1",
					TransactionStateLogMinIsr:            "2",
					TransactionStateLogReplicationFactor: "3",
				},
				Storage: kafkav1.KafkaStorage{
					Type: "ephemeral",
				},
			},
			Zookeeper: kafkav1.KafkaSpecZookeeper{
				Replicas: 3,
				Storage: kafkav1.KafkaStorage{
					Type: "ephemeral",
				},
			},
			EntityOperator: kafkav1.KafkaSpecEntityOperator{
				TopicOperator: kafkav1.KafkaTopicOperator{},
				UserOperator:  kafkav1.KafkaUserOperator{},
			},
		},
	}
	err := r.client.Create(context.TODO(), kafka)

	if err != nil && !k8serr.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, err
	}

	return v1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) handleProgressPhase() (v1alpha1.StatusPhase, error) {
	// check AMQ Streams pods are correct counts
	// no status on kafka object until 1.2

	return v1alpha1.PhaseInProgress, nil
}
