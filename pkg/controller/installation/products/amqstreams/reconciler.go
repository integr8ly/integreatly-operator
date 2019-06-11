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
	"k8s.io/client-go/kubernetes"

	errors2 "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

var (
	defaultInstallationNamespace = "amq-streams"
)

func NewReconciler(client pkgclient.Client, coreClient *kubernetes.Clientset, configManager config.ConfigReadWriter, instance *v1alpha1.Installation, clusterHasOLM bool) (*Reconciler, error) {
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
		coreClient:    coreClient,
		ConfigManager: configManager,
		Config:        config,
		mpm:           mpm,
	}, nil
}

type Reconciler struct {
	client        pkgclient.Client
	coreClient    *kubernetes.Clientset
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
			logrus.Infof("Creating subscription")
			return v1alpha1.PhaseCreatingSubscription, nil
		} else {
			logrus.Infof("jumping to component creation")
			return v1alpha1.PhaseCreatingComponents, nil
		}
	}

	return v1alpha1.PhaseAwaitingNS, nil
}

func (r *Reconciler) handleCreatingSubscription() (v1alpha1.StatusPhase, error) {
	err := r.mpm.CreateSubscription(
		marketplace.GetOperatorSources().Integreatly,
		r.Config.GetNamespace(),
		"amq-streams",
		"integreatly",
		[]string{r.Config.GetNamespace()},
		coreosv1alpha1.ApprovalAutomatic)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, err
	}

	return v1alpha1.PhaseCreatingComponents, nil
}

func (r *Reconciler) handleCreatingComponents() (v1alpha1.StatusPhase, error) {
	// commented out properties are for 1.1.0
	//Keep trying to create the kafka resource until the CRD exists or some other error
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
	err := r.client.Create(context.TODO(), kafka)

	//if this fails due to no matches for kind, keep trying until OLM creates the CRDs
	if err != nil && strings.Contains(err.Error(), "no matches for kind") {
		logrus.Infof("no match for kind error")
		return v1alpha1.PhaseCreatingComponents, nil
	}

	if err != nil && !k8serr.IsAlreadyExists(err) {
		logrus.Infof("kafka create error")
		return v1alpha1.PhaseFailed, err
	}

	return v1alpha1.PhaseInProgress, nil
}

func (r *Reconciler) handleProgressPhase() (v1alpha1.StatusPhase, error) {
	// check AMQ Streams is in ready state
	pods, err := r.coreClient.CoreV1().Pods(r.Config.GetNamespace()).List(metav1.ListOptions{})
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
