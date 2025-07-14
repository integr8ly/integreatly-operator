package ratelimit

import (
	"context"
	"fmt"
	"time"

	// "github.com/3scale-ops/marin3r/pkg/envoy"  // Temporarily disabled
	// envoyserializer "github.com/3scale-ops/marin3r/pkg/envoy/serializer"  // Temporarily disabled
	envoyclusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoycorev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoyendpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoylistenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	transport_sockets "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	envoyextentionv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	envoy_runtime "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"  // Temporarily disabled
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
	TransportSocketName      = "envoy.transport_sockets.tls"
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
	// Temporarily disabled - marin3r functionality stubbed out
	// TODO: Re-enable when marin3r compatibility is resolved
	return integreatlyv1alpha1.PhaseCompleted, nil
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

/*
*

	  envoyAPI: v3
	  envoyResources:
		clusters:
		listeners:
		nodeID:
		serialization: yaml

*
*/
func (ec *EnvoyConfig) CreateEnvoyConfig(ctx context.Context, client k8sclient.Client, clusterResources []*envoyclusterv3.Cluster, listenerResources []*envoylistenerv3.Listener, runtimes *envoy_runtime.Runtime, installation *integreatlyv1alpha1.RHMI) error {
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
			return fmt.Errorf("failed to convert envoy rate limiting cluster configuration to JSON %v", err)
		}

		yamlClusterResource, err := yaml.JSONToYAML(jsonClusterResource)
		if err != nil {
			return fmt.Errorf("failed to convert envoy rate limiting cluster JSON configuration to YAML %v", err)
		}
		envoyClusterResource = append(envoyClusterResource,
			marin3rv1alpha1.EnvoyResource{
				Name:  &cluster.Name,
				Value: string(yamlClusterResource),
			},
		)
	}

	envoyListenerResource := []marin3rv1alpha1.EnvoyResource{}
	for _, listener := range listenerResources {
		jsonListenerResource, err := ResourcesToJSON(listener)
		if err != nil {
			return fmt.Errorf("failed to convert envoy rate limiting listeners configuration to JSON %v", err)
		}

		yamlListenerResource, err := yaml.JSONToYAML(jsonListenerResource)
		if err != nil {
			return fmt.Errorf("failed to convert envoy rate limiting listener JSON configuration to YAML %v", err)
		}
		envoyListenerResource = append(envoyListenerResource,
			marin3rv1alpha1.EnvoyResource{
				Name:  &listener.Name,
				Value: string(yamlListenerResource),
			},
		)
	}

	envoyRuntimeResource := []marin3rv1alpha1.EnvoyResource{}
	jsonRuntimeResource, err := ResourcesToJSON(runtimes)
	if err != nil {
		return fmt.Errorf("failed to convert envoy rate limiting runtimes configuration to JSON %v", err)
	}

	yamlRuntimeResource, err := yaml.JSONToYAML(jsonRuntimeResource)
	if err != nil {
		return fmt.Errorf("failed to convert envoy rate limiting runtimes JSON configuration to YAML %v", err)
	}

	envoyRuntimeResource = append(envoyRuntimeResource, marin3rv1alpha1.EnvoyResource{
		Name:  &runtimes.Name,
		Value: string(yamlRuntimeResource),
	})

	_, err = controllerutil.CreateOrUpdate(ctx, client, envoyconfig, func() error {
		owner.AddIntegreatlyOwnerAnnotations(envoyconfig, installation)
		serialization := envoyserializer.YAML
		envoyAPIVersion := envoy.APIv3
		envoyconfig.Spec.NodeID = ec.nodeID
		envoyconfig.Spec.EnvoyAPI = &envoyAPIVersion
		envoyconfig.Spec.Serialization = &serialization
		envoyconfig.Spec.EnvoyResources = &marin3rv1alpha1.EnvoyResources{
			Clusters:  envoyClusterResource,
			Listeners: envoyListenerResource,
			Runtimes:  envoyRuntimeResource,
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create envoy config CR %v", err)
	}
	return nil
}

/*
*
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

*
*/
func CreateClusterResource(containerAddress, clusterName string, containerPort int) *envoyclusterv3.Cluster {

	// Setting up cluster endpoints
	clusterEndpoint := &envoycorev3.Address{
		Address: &envoycorev3.Address_SocketAddress{
			SocketAddress: &envoycorev3.SocketAddress{
				Address:  containerAddress,
				Protocol: envoycorev3.SocketAddress_TCP,
				PortSpecifier: &envoycorev3.SocketAddress_PortValue{
					PortValue: uint32(containerPort),
				},
			},
		},
	}

	cluster := envoyclusterv3.Cluster{
		Name:                 clusterName,
		ConnectTimeout:       durationpb.New(2 * time.Second),
		ClusterDiscoveryType: &envoyclusterv3.Cluster_Type{Type: envoyclusterv3.Cluster_STRICT_DNS},
		LbPolicy:             envoyclusterv3.Cluster_ROUND_ROBIN,
		LoadAssignment: &envoyendpointv3.ClusterLoadAssignment{
			ClusterName: clusterName,
			Endpoints: []*envoyendpointv3.LocalityLbEndpoints{{
				LbEndpoints: []*envoyendpointv3.LbEndpoint{
					{
						HostIdentifier: &envoyendpointv3.LbEndpoint_Endpoint{
							Endpoint: &envoyendpointv3.Endpoint{
								Address: clusterEndpoint,
							}},
					},
				},
			}},
		},
	}
	return &cluster
}

/*
*
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
func CreateListenerResource(listenerName, listenerAddress string, listenerPort int, filters []*envoylistenerv3.Filter) *envoylistenerv3.Listener {
	envoyListener := envoylistenerv3.Listener{
		Name: listenerName,
		Address: &envoycorev3.Address{
			Address: &envoycorev3.Address_SocketAddress{
				SocketAddress: &envoycorev3.SocketAddress{
					Protocol: envoycorev3.SocketAddress_TCP,
					Address:  listenerAddress,
					PortSpecifier: &envoycorev3.SocketAddress_PortValue{
						PortValue: uint32(listenerPort),
					},
				},
			},
		},
		FilterChains: []*envoylistenerv3.FilterChain{{
			Filters: filters,
		}},
	}

	return &envoyListener
}

/*
*

	runtimes:
	  - name: runtime
	    value: >-
	      {"name":
	      "runtime","layer":{"envoy.reloadable_features.sanitize_http_header_referer":
	      "false"}}

*
*/
func CreateRuntimesResource() *envoy_runtime.Runtime {
	layer, err := structpb.NewStruct(map[string]interface{}{
		"envoy.reloadable_features.sanitize_http_header_referer": false,
	})
	if err != nil {
		fmt.Printf("failed to create runtimes resource with error %v", err)
		return nil
	}

	envoyRuntime := &envoy_runtime.Runtime{
		Name:  "runtime",
		Layer: layer,
	}

	return envoyRuntime
}

/*
*

	explicitHttpConfig:
	    http2ProtocolOptions: {}

*
*/
func CreateTypedExtensionProtocol() (*anypb.Any, error) {
	serial, err := anypb.New(&envoyextentionv3.HttpProtocolOptions{
		UpstreamProtocolOptions: &envoyextentionv3.HttpProtocolOptions_ExplicitHttpConfig_{
			ExplicitHttpConfig: &envoyextentionv3.HttpProtocolOptions_ExplicitHttpConfig{
				ProtocolConfig: &envoyextentionv3.HttpProtocolOptions_ExplicitHttpConfig_Http2ProtocolOptions{},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return serial, nil
}

/*
*

	"@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext
	common_tls_context:
	        validation_context:
	            trust_chain_verification: ACCEPT_UNTRUSTED

*
*/
func CreateApicastTransportSocketConfig() (*anypb.Any, error) {
	serial, err := anypb.New(&transport_sockets.UpstreamTlsContext{
		CommonTlsContext: &transport_sockets.CommonTlsContext{
			ValidationContextType: &transport_sockets.CommonTlsContext_ValidationContext{
				ValidationContext: &transport_sockets.CertificateValidationContext{
					TrustChainVerification: transport_sockets.CertificateValidationContext_ACCEPT_UNTRUSTED,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return serial, nil
}

func ResourcesToJSON(pb proto.Message) ([]byte, error) {
	json, err := protojson.Marshal(pb)
	if err != nil {
		return []byte{}, err
	}

	return json, nil
}
