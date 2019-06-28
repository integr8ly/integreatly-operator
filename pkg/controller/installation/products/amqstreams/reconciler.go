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
	coreosv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	errors2 "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	defaultInstallationNamespace = "amq-streams"
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
	}, nil
}

type Reconciler struct {
	Config        *config.AMQStreams
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
}

func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	phase := inst.Status.ProductStatus[r.Config.GetProductName()]
	switch v1alpha1.StatusPhase(phase) {
	case v1alpha1.PhaseNone:
		return r.handleNoPhase(ctx, serverClient, inst)
	case v1alpha1.PhaseAwaitingNS:
		return r.handleAwaitingNSPhase(ctx, serverClient)
	case v1alpha1.PhaseCreatingSubscription:
		return r.handleCreatingSubscription(ctx)
	case v1alpha1.PhaseCreatingComponents:
		return r.handleCreatingComponents(ctx, serverClient, inst)
	case v1alpha1.PhaseAwaitingOperator:
		return r.handleAwaitingOperator(ctx)
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

func (r *Reconciler) handleNoPhase(ctx context.Context, serverClient pkgclient.Client, inst *v1alpha1.Installation) (v1alpha1.StatusPhase, error) {
	nsr := resources.NewNamespaceReconciler(serverClient)
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      r.Config.GetNamespace(),
		},
	}
	ns, err := nsr.Reconcile(ctx, ns, inst)
	if err != nil {
		return v1alpha1.PhaseFailed, errors2.Wrapf(err, "error reconciling namespace for amq streams")
	}
	return v1alpha1.PhaseAwaitingNS, nil
}

func (r *Reconciler) handleAwaitingNSPhase(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Config.GetNamespace(),
		},
	}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: r.Config.GetNamespace()}, ns)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}
	if ns.Status.Phase == v1.NamespaceActive {
		logrus.Infof("Creating subscription")
		return v1alpha1.PhaseCreatingSubscription, nil
	}

	return v1alpha1.PhaseAwaitingNS, nil
}

func (r *Reconciler) handleCreatingSubscription(ctx context.Context) (v1alpha1.StatusPhase, error) {
	err := r.mpm.CreateSubscription(
		ctx,
		marketplace.GetOperatorSources().Integreatly,
		r.Config.GetNamespace(),
		"amq-streams",
		"integreatly",
		[]string{r.Config.GetNamespace()},
		coreosv1alpha1.ApprovalAutomatic)
	if err != nil && !k8serr.IsAlreadyExists(err) {
		return v1alpha1.PhaseFailed, err
	}

	return v1alpha1.PhaseAwaitingOperator, nil
}

func (r *Reconciler) handleAwaitingOperator(ctx context.Context) (v1alpha1.StatusPhase, error) {
	ip, _, err := r.mpm.GetSubscriptionInstallPlan(ctx, "amq-streams", r.Config.GetNamespace())
	if err != nil {
		if k8serr.IsNotFound(err) {
			logrus.Infof("No installplan created yet")
			return v1alpha1.PhaseAwaitingOperator, nil
		}

		logrus.Infof("Error getting amq-streams subscription installplan")
		return v1alpha1.PhaseFailed, err
	}

	if ip.Status.Phase != coreosv1alpha1.InstallPlanPhaseComplete {
		logrus.Infof("amq-streams installplan phase is %s", ip.Status.Phase)
		return v1alpha1.PhaseAwaitingOperator, nil
	}

	logrus.Infof("amq-streams installplan phase is %s", coreosv1alpha1.InstallPlanPhaseComplete)

	return v1alpha1.PhaseCreatingComponents, nil
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
	pods := &corev1.PodList{}
	err = serverClient.List(ctx, &pkgclient.ListOptions{Namespace: r.Config.GetNamespace()}, pods)
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
