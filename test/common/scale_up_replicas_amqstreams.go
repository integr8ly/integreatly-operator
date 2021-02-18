package common

import (
	goctx "context"
	"fmt"
	"strconv"
	"time"

	kafkav1alpha1 "github.com/integr8ly/integreatly-operator/apis-products/kafka.strimzi.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	numberOfAmqstreamsReplicas  = 2 //size in reconciler
	scaleUpAmqstreamsReplicas   = 1
	scaleDownAmqstreamsReplicas = 1
	amqstreamsName              = "rhmi-cluster"
	amqstreamsNamespace         = NamespacePrefix + "amq-streams"
	retryIntervalAmqstreams     = time.Second * 20
	timeoutAmqstreams           = time.Minute * 7
)

func TestReplicasInAmqstreams(t TestingTB, ctx *TestingContext) {

	amqstreams, err := getAmqstreams(ctx.Client)
	if err != nil {
		t.Fatalf("failed to get amqstreams : %v", err)
	}

	t.Log("Checking correct number of replicas are set")
	if err := checkNumberOfReplicasAgainstValueAmqstreams(amqstreams, ctx, numberOfAmqstreamsReplicas, retryIntervalAmqstreams, timeoutAmqstreams, t); err != nil {
		t.Fatalf("Incorrect number of replicas to start : %v", err)
	}

	amqstreams, err = updateAmqstreams(ctx.Client, scaleUpAmqstreamsReplicas, t)
	if err != nil {
		t.Fatalf("Unable to update : %v", err)
	}

	t.Log("Checking correct number of updated replicas are set")
	if err := checkNumberOfReplicasAgainstValueAmqstreams(amqstreams, ctx, scaleUpAmqstreamsReplicas, retryIntervalAmqstreams, timeoutAmqstreams, t); err != nil {
		t.Fatalf("Incorrect number of replicas : %v", err)
	}

	amqstreams, err = updateAmqstreams(ctx.Client, scaleDownAmqstreamsReplicas, t)
	if err != nil {
		t.Fatalf("Unable to update : %v", err)
	}

	t.Log("Checking correct number of updated replicas are set")
	if err := checkNumberOfReplicasAgainstValueAmqstreams(amqstreams, ctx, numberOfAmqstreamsReplicas, retryIntervalAmqstreams, timeoutAmqstreams, t); err != nil {
		t.Fatalf("Incorrect number of replicas : %v", err)
	}

}

func getAmqstreams(dynClient k8sclient.Client) (kafkav1alpha1.Kafka, error) {
	amqstreams := &kafkav1alpha1.Kafka{}

	if err := dynClient.Get(goctx.TODO(), types.NamespacedName{Name: amqstreamsName, Namespace: amqstreamsNamespace}, amqstreams); err != nil {
		return *amqstreams, err
	}

	return *amqstreams, nil
}

func updateAmqstreams(dynClient k8sclient.Client, replicas int, t TestingTB) (kafkav1alpha1.Kafka, error) {

	amqstreams, err := getAmqstreams(dynClient)

	if err != nil {
		return amqstreams, err
	}
	amqstreams = kafkav1alpha1.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Name:            amqstreamsName,
			Namespace:       amqstreamsNamespace,
			ResourceVersion: amqstreams.GetResourceVersion(),
		},
	}

	amqstreams.APIVersion = fmt.Sprintf("%s/%s", kafkav1alpha1.SchemeGroupVersion.Group, kafkav1alpha1.SchemeGroupVersion.Version)
	amqstreams.Kind = kafkav1alpha1.KafkaKind

	amqstreams.Name = amqstreamsName
	amqstreams.Namespace = amqstreamsNamespace

	amqstreams.Spec.Kafka.Version = "2.1.1"

	amqstreams.Spec.Kafka.Replicas = replicas
	amqstreams.Spec.Zookeeper.Replicas = replicas

	amqstreams.Spec.Kafka.Listeners = map[string]kafkav1alpha1.KafkaListener{
		"plain": {},
		"tls":   {},
	}
	amqstreams.Spec.Kafka.Config.OffsetsTopicReplicationFactor = strconv.FormatInt(int64(numberOfAmqstreamsReplicas), 1)
	amqstreams.Spec.Kafka.Config.TransactionStateLogReplicationFactor = strconv.FormatInt(int64(numberOfAmqstreamsReplicas), 1)
	amqstreams.Spec.Kafka.Config.TransactionStateLogMinIsr = "2"
	amqstreams.Spec.Kafka.Config.LogMessageFormatVersion = "2.1"
	amqstreams.Spec.Kafka.Storage.Type = "persistent-claim"
	amqstreams.Spec.Kafka.Storage.Size = "8Gi"
	amqstreams.Spec.Kafka.Storage.DeleteClaim = false

	amqstreams.Spec.Zookeeper.Storage.Type = "persistent-claim"
	amqstreams.Spec.Zookeeper.Storage.Size = "8Gi"
	amqstreams.Spec.Zookeeper.Storage.DeleteClaim = false

	amqstreams.Spec.EntityOperator.TopicOperator = kafkav1alpha1.KafkaTopicOperator{}
	amqstreams.Spec.EntityOperator.UserOperator = kafkav1alpha1.KafkaUserOperator{}

	if err := dynClient.Update(goctx.TODO(), amqstreams.DeepCopy(), &k8sclient.UpdateOptions{}); err != nil {
		return amqstreams, err
	}

	return amqstreams, nil
}

func checkNumberOfReplicasAgainstValueAmqstreams(amqstreams kafkav1alpha1.Kafka, ctx *TestingContext, numberOfRequiredReplicas int, retryInterval, timeout time.Duration, t TestingTB) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		amqstreams, err = getAmqstreams(ctx.Client)
		if err != nil {
			t.Fatalf("failed to get Apicurito : %v", err)
		}
		if *&amqstreams.Spec.Kafka.Replicas == numberOfRequiredReplicas {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("Number of replicas for apicurito.Spec.Size is not correct : Replicas - %v, Expected - %v", *&amqstreams.Spec.Kafka.Replicas, numberOfRequiredReplicas)
	}
	return nil
}
