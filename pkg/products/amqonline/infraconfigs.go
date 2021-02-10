package amqonline

import (
	"github.com/integr8ly/integreatly-operator/apis-products/enmasse/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetDefaultBrokeredInfraConfigs(ns string) []*v1beta1.BrokeredInfraConfig {
	return []*v1beta1.BrokeredInfraConfig{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default",
				Namespace: ns,
			},
			Spec: v1beta1.BrokeredInfraConfigSpec{
				Admin: v1beta1.InfraConfigAdmin{
					Resources: v1beta1.InfraConfigResources{
						Memory: "512Mi",
					},
				},
				Broker: v1beta1.InfraConfigBroker{
					Resources: v1beta1.InfraConfigResources{
						Memory:  "512Mi",
						Storage: "5Gi",
					},
					AddressFullPolicy: "FAIL",
				},
			},
		},
	}
}

func GetDefaultStandardInfraConfigs(ns string) []*v1beta1.StandardInfraConfig {
	return []*v1beta1.StandardInfraConfig{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default-minimal",
			},
			Spec: v1beta1.StandardInfraConfigSpec{
				Admin: v1beta1.InfraConfigAdmin{
					Resources: v1beta1.InfraConfigResources{
						Memory: "512Mi",
					},
				},
				Broker: v1beta1.InfraConfigBroker{
					Resources: v1beta1.InfraConfigResources{
						Memory:  "512Mi",
						Storage: "2Gi",
					},
					AddressFullPolicy: "FAIL",
				},
				Router: v1beta1.InfraConfigRouter{
					MinReplicas: 1,
					Resources: v1beta1.InfraConfigResources{
						Memory: "256Mi",
					},
					LinkCapacity: 250,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default-with-mqtt",
				Annotations: map[string]string{
					"enmasse.io/with-mqtt": "true",
				},
			},
			Spec: v1beta1.StandardInfraConfigSpec{
				Admin: v1beta1.InfraConfigAdmin{
					Resources: v1beta1.InfraConfigResources{
						Memory: "512Mi",
					},
				},
				Broker: v1beta1.InfraConfigBroker{
					Resources: v1beta1.InfraConfigResources{
						Memory:  "512Mi",
						Storage: "2Gi",
					},
					AddressFullPolicy: "FAIL",
					MaxUnavailable:    1,
				},
				Router: v1beta1.InfraConfigRouter{
					MinReplicas: 2,
					Resources: v1beta1.InfraConfigResources{
						Memory: "512Mi",
					},
					LinkCapacity:   250,
					MaxUnavailable: 1,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
			Spec: v1beta1.StandardInfraConfigSpec{
				Admin: v1beta1.InfraConfigAdmin{
					Resources: v1beta1.InfraConfigResources{
						Memory: "512Mi",
					},
				},
				Broker: v1beta1.InfraConfigBroker{
					Resources: v1beta1.InfraConfigResources{
						Memory:  "512Mi",
						Storage: "2Gi",
					},
					AddressFullPolicy: "FAIL",
					MaxUnavailable:    1,
				},
				Router: v1beta1.InfraConfigRouter{
					MinReplicas: 2,
					Resources: v1beta1.InfraConfigResources{
						Memory: "512Mi",
					},
					LinkCapacity:   250,
					MaxUnavailable: 1,
				},
			},
		},
	}
}
