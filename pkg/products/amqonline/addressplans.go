package amqonline

import (
	"github.com/integr8ly/integreatly-operator/pkg/apis/enmasse/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetDefaultAddressPlans(ns string) []*v1beta2.AddressPlan {
	return []*v1beta2.AddressPlan{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "brokered-topic",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Brokered Topic",
				DisplayOrder:     0,
				ShortDescription: "Creates a topic on a broker.",
				LongDescription:  "Creates a topic on a broker.",
				AddressType:      "topic",
				Resources: v1beta2.AddressPlanResources{
					Broker: "0.0",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "brokered-queue",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Brokered Queue",
				DisplayOrder:     0,
				ShortDescription: "Creates a queue on a broker.",
				LongDescription:  "Creates a queue on a broker.",
				AddressType:      "queue",
				Resources: v1beta2.AddressPlanResources{
					Broker: "0.0",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standard-large-anycast",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Large Anycast",
				DisplayOrder:     0,
				ShortDescription: "Creates a large anycast address.",
				LongDescription:  "Creates a large anycast address where messages go via a router that does not take ownership of the messages.",
				AddressType:      "anycast",
				Resources: v1beta2.AddressPlanResources{
					Router: "0.1",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standard-large-multicast",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Large Multicast",
				DisplayOrder:     0,
				ShortDescription: "Creates a large multicast address.",
				LongDescription:  "Creates a large multicast address where messages go via a router that does not take ownership of the messages.",
				AddressType:      "multicast",
				Resources: v1beta2.AddressPlanResources{
					Router: "0.1",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standard-large-queue",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Large Queue",
				DisplayOrder:     0,
				ShortDescription: "Creates a large queue backed by a dedicated broker.",
				LongDescription:  "Creates a large queue backed by a dedicated broker.",
				AddressType:      "queue",
				Resources: v1beta2.AddressPlanResources{
					Router: "0.1",
					Broker: "1.0",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standard-large-partitioned-queue",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Large HA Queue",
				DisplayOrder:     6,
				ShortDescription: "Creates a large HA queue sharing underlying brokers with other queues.",
				LongDescription:  "Creates a large HA queue sharing underlying brokers with other queues. The queue is sharded accross multiple brokers for HA and improved performance. A sharded queue no longer guarantees message ordering.",
				AddressType:      "queue",
				Resources: v1beta2.AddressPlanResources{
					Router: "0.1",
					Broker: "1.0",
				},
				Partitions: 3,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standard-large-subscription",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Large Subscription",
				DisplayOrder:     0,
				ShortDescription: "Creates a large durable subscription on a topic.",
				LongDescription:  "Creates a large durable subscription on a topic, which is then accessed as a distinct address.",
				AddressType:      "subscription",
				Resources: v1beta2.AddressPlanResources{
					Router: "0.1",
					Broker: "1.0",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standard-large-topic",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Large Topic",
				DisplayOrder:     0,
				ShortDescription: "Creates a large topic backed by a dedicated broker.",
				LongDescription:  "Creates a large topic backed by a dedicated broker.",
				AddressType:      "topic",
				Resources: v1beta2.AddressPlanResources{
					Router: "0.1",
					Broker: "1.0",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standard-medium-anycast",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Medium Anycast",
				DisplayOrder:     0,
				ShortDescription: "Creates a medium anycast address.",
				LongDescription:  "Creates a medium anycast address where messages go via a router that does not take ownership of the messages.",
				AddressType:      "anycast",
				Resources: v1beta2.AddressPlanResources{
					Router: "0.01",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standard-medium-multicast",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Medium Multicast",
				DisplayOrder:     0,
				ShortDescription: "Creates a medium multicast address.",
				LongDescription:  "Creates a medium multicast address where messages go via a router that does not take ownership of the messages.",
				AddressType:      "multicast",
				Resources: v1beta2.AddressPlanResources{
					Router: "0.01",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standard-medium-queue",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Medium Queue",
				DisplayOrder:     1,
				ShortDescription: "Creates a medium sized queue sharing underlying broker with other queues.",
				LongDescription:  "Creates a medium sized queue sharing underlying broker with other queues.",
				AddressType:      "queue",
				Resources: v1beta2.AddressPlanResources{
					Router: "0.01",
					Broker: "0.1",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standard-medium-partitioned-queue",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Medium HA Queue",
				DisplayOrder:     5,
				ShortDescription: "Creates a medium sized HA queue sharing underlying broker with other queues.",
				LongDescription:  "Creates a medium sized HA queue sharing underlying broker with other queues. The queue is sharded accross multiple brokers for HA and improved performance. A sharded queue no longer guarantees message ordering.",
				AddressType:      "queue",
				Resources: v1beta2.AddressPlanResources{
					Router: "0.01",
					Broker: "0.1",
				},
				Partitions: 3,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standard-medium-subscription",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Medium Subscription",
				DisplayOrder:     1,
				ShortDescription: "Creates a medium durable subscription on a topic.",
				LongDescription:  "Creates a medium durable subscription on a topic, which is then accessed as a distinct address.",
				AddressType:      "subscription",
				Resources: v1beta2.AddressPlanResources{
					Router: "0.01",
					Broker: "0.1",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standard-medium-topic",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Medium Topic",
				DisplayOrder:     1,
				ShortDescription: "Creates a medium sized topic sharing underlying broker with other topics.",
				LongDescription:  "Creates a medium sized topic sharing underlying broker with other topics.",
				AddressType:      "topic",
				Resources: v1beta2.AddressPlanResources{
					Router: "0.001",
					Broker: "0.1",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standard-small-anycast",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Small Anycast",
				DisplayOrder:     0,
				ShortDescription: "Creates a small anycast address.",
				LongDescription:  "Creates a small anycast address where messages go via a router that does not take ownership of the messages.",
				AddressType:      "anycast",
				Resources: v1beta2.AddressPlanResources{
					Router: "0.001",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standard-small-multicast",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Small Multicast",
				DisplayOrder:     0,
				ShortDescription: "Creates a small multicast address.",
				LongDescription:  "Creates a small multicast address where messages go via a router that does not take ownership of the messages.",
				AddressType:      "multicast",
				Resources: v1beta2.AddressPlanResources{
					Router: "0.001",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standard-small-queue",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Small Queue",
				DisplayOrder:     0,
				ShortDescription: "Creates a small queue sharing underlying broker with other queues.",
				LongDescription:  "Creates a small queue sharing underlying broker with other queues.",
				AddressType:      "queue",
				Resources: v1beta2.AddressPlanResources{
					Router: "0.001",
					Broker: "0.01",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standard-small-partitioned-queue",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Small HA Queue",
				DisplayOrder:     4,
				ShortDescription: "Creates a small HA queue sharing underlying broker with other queues.",
				LongDescription:  "Creates a small HA queue sharing underlying broker with other queues. The queue is sharded accross multiple brokers for HA and improved performance. A sharded queue no longer guarantees message ordering.",
				AddressType:      "queue",
				Resources: v1beta2.AddressPlanResources{
					Router: "0.001",
					Broker: "0.01",
				},
				Partitions: 3,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standard-small-subscription",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Small Subscription",
				DisplayOrder:     0,
				ShortDescription: "Creates a small durable subscription on a topic.",
				LongDescription:  "Creates a small durable subscription on a topic, which is then accessed as a distinct address.",
				AddressType:      "subscription",
				Resources: v1beta2.AddressPlanResources{
					Router: "0.001",
					Broker: "0.01",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standard-small-topic",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Small Topic",
				DisplayOrder:     0,
				ShortDescription: "Creates a small topic sharing underlying broker with other topics.",
				LongDescription:  "Creates a small topic sharing underlying broker with other topics.",
				AddressType:      "topic",
				Resources: v1beta2.AddressPlanResources{
					Router: "0.001",
					Broker: "0.01",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standard-xlarge-queue",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Extra Large Queue",
				DisplayOrder:     3,
				ShortDescription: "Creates an extra large queue backed by 2 brokers.",
				LongDescription:  "Creates an extra large queue backed by 2 brokers.",
				AddressType:      "queue",
				Resources: v1beta2.AddressPlanResources{
					Router: "0.2",
					Broker: "2.0",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "standard-xlarge-topic",
				Namespace: ns,
			},
			Spec: v1beta2.AddressPlanSpec{
				DisplayName:      "Extra Large Topic",
				DisplayOrder:     3,
				ShortDescription: "Creates an extra large topic backed by 2 brokers.",
				LongDescription:  "Creates an extra large topic backed by 2 brokers.",
				AddressType:      "topic",
				Resources: v1beta2.AddressPlanResources{
					Router: "0.2",
					Broker: "2.0",
				},
			},
		},
	}
}
