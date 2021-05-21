package ratelimit

import (
	"bytes"
	"context"
	"fmt"
	"time"

	listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoycore "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_v2_endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	"github.com/golang/protobuf/ptypes"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

const (
	RateLimitClusterName     = "ratelimit"
	RateLimitDomain          = "apicast-ratelimit"
	RateLimitDescriptorValue = "slowpath"
)

func DeleteEnvoyConfigsInNamespaces(ctx context.Context, client k8sclient.Client, namespaces ...string) (integreatlyv1alpha1.StatusPhase, error) {
	phase := integreatlyv1alpha1.PhaseCompleted

	for _, namespace := range namespaces {
		nsPhase, err := DeleteEnvoyConfigsInNamespace(ctx, client, namespace)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, err
		}

		// Only change the status phase if it was Completed, to ensure that
		// as long as one of the namespaces returns InProgress, the phase is
		// set to InProgress
		if phase == integreatlyv1alpha1.PhaseCompleted {
			phase = nsPhase
		}
	}

	return phase, nil
}

func DeleteEnvoyConfigsInNamespace(ctx context.Context, client k8sclient.Client, namespace string) (integreatlyv1alpha1.StatusPhase, error) {
	envoyConfigs := &marin3rv1alpha1.EnvoyConfigList{}
	if err := client.List(ctx, envoyConfigs, k8sclient.InNamespace(namespace)); err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	if len(envoyConfigs.Items) == 0 {
		return integreatlyv1alpha1.PhaseCompleted, nil
	}

	for _, envoyConfig := range envoyConfigs.Items {
		if err := k8sclient.IgnoreNotFound(
			client.Delete(ctx, &envoyConfig),
		); err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to delete envoyconfig for namespace %s: %v",
				namespace, err)
		}
	}

	return integreatlyv1alpha1.PhaseInProgress, nil
}

type EnvoyConfig struct {
	name      string
	namespace string
	nodeID    string
}

func NewEnvoyConfig(name, namespace, nodeID string) *EnvoyConfig {
	return &EnvoyConfig{
		name:      name,
		namespace: namespace,
		nodeID:    nodeID,
	}
}

/**
  envoyAPI: v2
  envoyResources:
	clusters:
	listeners:
	nodeID:
	serialization: yaml
**/
func (ec *EnvoyConfig) CreateEnvoyConfig(ctx context.Context, client k8sclient.Client, clusterResources []*envoyapi.Cluster, listenerResources []*envoyapi.Listener, installation *integreatlyv1alpha1.RHMI) error {
	envoyconfig := &marin3rv1alpha1.EnvoyConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ec.name,
			Namespace: ec.namespace,
		},
	}

	envoyClusterResource := []marin3rv1alpha1.EnvoyResource{}
	for _, cluster := range clusterResources {
		jsonClusterResource, err := ResourcesToJSON(cluster)
		if err != nil {
			return fmt.Errorf("Failed to convert envoy rate limiting cluster configuration to JSON %v", err)
		}

		yamlClusterResource, err := yaml.JSONToYAML(jsonClusterResource)
		if err != nil {
			return fmt.Errorf("Failed to convert envoy rate limiting cluster JSON configuration to YAML %v", err)
		}
		envoyClusterResource = append(envoyClusterResource,
			marin3rv1alpha1.EnvoyResource{
				Name:  cluster.Name,
				Value: string(yamlClusterResource),
			},
		)
	}

	envoyListenerResource := []marin3rv1alpha1.EnvoyResource{}
	for _, listener := range listenerResources {
		jsonListenerResource, err := ResourcesToJSON(listener)
		if err != nil {
			return fmt.Errorf("Failed to convert envoy rate limiting cluster configuration to JSON %v", err)
		}

		yamlListenerResource, err := yaml.JSONToYAML(jsonListenerResource)
		if err != nil {
			return fmt.Errorf("Failed to convert envoy rate limiting listener JSON configuration to YAML %v", err)
		}
		envoyListenerResource = append(envoyListenerResource,
			marin3rv1alpha1.EnvoyResource{
				Name:  listener.Name,
				Value: string(yamlListenerResource),
			},
		)
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, envoyconfig, func() error {
		owner.AddIntegreatlyOwnerAnnotations(envoyconfig, installation)
		serialization := "yaml"
		envoyconfig.Spec.NodeID = ec.nodeID
		envoyconfig.Spec.Serialization = &serialization
		envoyconfig.Spec.EnvoyResources = &marin3rv1alpha1.EnvoyResources{
			Clusters:  envoyClusterResource,
			Listeners: envoyListenerResource,
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Failed to create envoy config CR %v", err)
	}
	return nil
}

/**
Creates envoy config cluster resource
- name: clusterName
	value: |
		connectTimeout: 2s
		loadAssignment:
		clusterName: clusterName
		endpoints:
		- lbEndpoints:
			- endpoint:
				address:
				socketAddress:
					address: containerAddress
					portValue: containerPort
		name: apicast-ratelimit
		type: STRICT_DNS
**/
func CreateClusterResource(containerAddress, clusterName string, containerPort int) *envoyapi.Cluster {

	// Setting up cluster endpoints
	clusterEndpoint := &envoycore.Address{
		Address: &envoycore.Address_SocketAddress{
			SocketAddress: &envoycore.SocketAddress{
				Address:  containerAddress,
				Protocol: envoycore.SocketAddress_TCP,
				PortSpecifier: &envoycore.SocketAddress_PortValue{
					PortValue: uint32(containerPort),
				},
			},
		},
	}

	cluster := envoyapi.Cluster{
		Name:                 clusterName,
		ConnectTimeout:       ptypes.DurationProto(2 * time.Second),
		ClusterDiscoveryType: &envoyapi.Cluster_Type{Type: envoyapi.Cluster_STRICT_DNS},
		LbPolicy:             envoyapi.Cluster_ROUND_ROBIN,
		LoadAssignment: &envoyapi.ClusterLoadAssignment{
			ClusterName: clusterName,
			Endpoints: []*envoy_api_v2_endpoint.LocalityLbEndpoints{{
				LbEndpoints: []*envoy_api_v2_endpoint.LbEndpoint{
					{
						HostIdentifier: &envoy_api_v2_endpoint.LbEndpoint_Endpoint{
							Endpoint: &envoy_api_v2_endpoint.Endpoint{
								Address: clusterEndpoint,
							}},
					},
				},
			}},
		},
	}
	return &cluster
}

/**
  - name: listenerName
    value: |
      address:
        socketAddress:
          address: listenerAddress
          portValue: listenerPort
      filterChains:
      - filters: &filters
	name: http
*/
func CreateListenerResource(listenerName, listenerAddress string, listenerPort int, filters []*listener.Filter) *envoyapi.Listener {

	envoyListener := envoyapi.Listener{
		Name: listenerName,
		Address: &envoycore.Address{
			Address: &envoycore.Address_SocketAddress{
				SocketAddress: &envoycore.SocketAddress{
					Protocol: envoycore.SocketAddress_TCP,
					Address:  listenerAddress,
					PortSpecifier: &envoycore.SocketAddress_PortValue{
						PortValue: uint32(listenerPort),
					},
				},
			},
		},
		FilterChains: []*listener.FilterChain{{
			Filters: filters,
		}},
	}

	return &envoyListener
}

func ResourcesToJSON(pb proto.Message) ([]byte, error) {
	m := jsonpb.Marshaler{}

	json := bytes.NewBuffer([]byte{})
	err := m.Marshal(json, pb)
	if err != nil {
		return []byte{}, err
	}
	return json.Bytes(), nil
}
