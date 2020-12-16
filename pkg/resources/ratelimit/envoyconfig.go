package ratelimit

import (
	"bytes"
	"context"
	"fmt"

	"time"

	marin3rv1alpha "github.com/3scale/marin3r/pkg/apis/marin3r/v1alpha1"
	marin3rv1alpha1 "github.com/3scale/marin3r/pkg/apis/marin3r/v1alpha1"
	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoycore "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	envoy_api_v2_endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	v2route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	yaml "github.com/ghodss/yaml"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	ptypes "github.com/golang/protobuf/ptypes"
	wrappers "github.com/golang/protobuf/ptypes/wrappers"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	structpb "google.golang.org/protobuf/types/known/structpb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

func CreateEnvoyConfigurationCR(ctx context.Context, client k8sclient.Client, configTarget string, configManager config.ConfigReadWriter, config config.ConfigReadable, installation integreatlyv1alpha1.RHMI) error {

	rateLimitService := &corev1.Service{}
	marin3rConfig, err := configManager.ReadMarin3r()
	if err != nil {
		return fmt.Errorf("failed to load marin3r config in 3scale reconciler: %v", err)
	}
	err = client.Get(ctx, k8sclient.ObjectKey{
		Namespace: marin3rConfig.GetNamespace(),
		Name:      "ratelimit",
	}, rateLimitService)

	if err != nil {
		return fmt.Errorf("failed to rate limiting service: %v", err)
	}

	// Setting up cluster endpoints for rate limit and apicast
	apicastEndpoint := &envoycore.Address{Address: &envoycore.Address_SocketAddress{
		SocketAddress: &envoycore.SocketAddress{
			Address:  "127.0.0.1",
			Protocol: envoycore.SocketAddress_TCP,
			PortSpecifier: &envoycore.SocketAddress_PortValue{
				PortValue: uint32(8080),
			},
		},
	}}

	rateLimitEndpoint := &envoycore.Address{Address: &envoycore.Address_SocketAddress{
		SocketAddress: &envoycore.SocketAddress{
			Address:  rateLimitService.Spec.ClusterIP,
			Protocol: envoycore.SocketAddress_TCP,
			PortSpecifier: &envoycore.SocketAddress_PortValue{
				PortValue: uint32(8081),
			},
		},
	}}

	cluster := envoyapi.Cluster{
		Name:                 configTarget,
		ConnectTimeout:       ptypes.DurationProto(2 * time.Second),
		ClusterDiscoveryType: &envoyapi.Cluster_Type{Type: envoyapi.Cluster_STRICT_DNS},
		LbPolicy:             envoyapi.Cluster_ROUND_ROBIN,
		LoadAssignment: &envoyapi.ClusterLoadAssignment{
			ClusterName: configTarget,
			Endpoints: []*envoy_api_v2_endpoint.LocalityLbEndpoints{{
				LbEndpoints: []*envoy_api_v2_endpoint.LbEndpoint{
					{
						HostIdentifier: &envoy_api_v2_endpoint.LbEndpoint_Endpoint{
							Endpoint: &envoy_api_v2_endpoint.Endpoint{
								Address: apicastEndpoint,
							}},
					},
				},
			}},
		},
	}

	rateLimitCluster := envoyapi.Cluster{
		Name:                 "ratelimit",
		ConnectTimeout:       ptypes.DurationProto(2 * time.Second),
		ClusterDiscoveryType: &envoyapi.Cluster_Type{Type: envoyapi.Cluster_STRICT_DNS},
		LbPolicy:             envoyapi.Cluster_ROUND_ROBIN,
		Http2ProtocolOptions: &envoycore.Http2ProtocolOptions{},
		LoadAssignment: &envoyapi.ClusterLoadAssignment{
			ClusterName: "ratelimit",
			Endpoints: []*envoy_api_v2_endpoint.LocalityLbEndpoints{{
				LbEndpoints: []*envoy_api_v2_endpoint.LbEndpoint{
					{
						HostIdentifier: &envoy_api_v2_endpoint.LbEndpoint_Endpoint{
							Endpoint: &envoy_api_v2_endpoint.Endpoint{
								Address: rateLimitEndpoint,
							}},
					},
				},
			}},
		},
	}

	// Setting up envoyListener
	virtualHost := v2route.VirtualHost{
		Name:    configTarget,
		Domains: []string{"*"},

		Routes: []*v2route.Route{
			{
				Match: &v2route.RouteMatch{
					PathSpecifier: &v2route.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &v2route.Route_Route{
					Route: &v2route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: configTarget,
						},
						RateLimits: []*route.RateLimit{{
							Stage: &wrappers.UInt32Value{Value: 0},
							Actions: []*route.RateLimit_Action{{
								ActionSpecifier: &route.RateLimit_Action_GenericKey_{
									GenericKey: &route.RateLimit_Action_GenericKey{
										DescriptorValue: "slowpath",
									},
								},
							}},
						}},
					},
				},
			},
		},
	}

	// Setting GRPC
	clusterName := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"cluster_name": {
				Kind: &structpb.Value_StringValue{
					StringValue: "ratelimit",
				},
			},
		},
	}

	grpcService := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"timeout": {
				Kind: &structpb.Value_StringValue{
					StringValue: "2s",
				},
			},
			"envoy_grpc": {
				Kind: &structpb.Value_StructValue{
					StructValue: clusterName,
				},
			},
		},
	}
	grpcInnerService := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"grpc_service": {
				Kind: &structpb.Value_StructValue{
					StructValue: grpcService,
				},
			},
		},
	}
	httpFilterGrpc := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"domain": {
				Kind: &structpb.Value_StringValue{
					StringValue: "apicast-ratelimit",
				},
			},
			"stage": {
				Kind: &structpb.Value_NumberValue{
					NumberValue: 0,
				},
			},
			"rate_limit_service": {
				Kind: &structpb.Value_StructValue{
					StructValue: grpcInnerService,
				},
			},
		},
	}

	// Setting up connection manager
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "ingress_http",
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: &envoyapi.RouteConfiguration{
				Name:         "local_route",
				VirtualHosts: []*v2route.VirtualHost{&virtualHost},
			},
		},
		HttpFilters: []*hcm.HttpFilter{
			{
				Name:       "envoy.rate_limit",
				ConfigType: &hcm.HttpFilter_Config{Config: httpFilterGrpc},
			},
			{
				Name: "envoy.router",
			},
		},
	}

	pbst, err := ptypes.MarshalAny(manager)
	if err != nil {
		return fmt.Errorf("failed to convert HttpConnectionManager for rate limiting: %v", err)
	}

	envoyListener := &envoyapi.Listener{
		Name: "http",
		Address: &envoycore.Address{
			Address: &envoycore.Address_SocketAddress{
				SocketAddress: &envoycore.SocketAddress{
					Protocol: envoycore.SocketAddress_TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &envoycore.SocketAddress_PortValue{
						PortValue: uint32(8443),
					},
				},
			},
		},
		FilterChains: []*listener.FilterChain{{
			Filters: []*listener.Filter{{
				Name:       "envoy.http_connection_manager",
				ConfigType: &listener.Filter_TypedConfig{TypedConfig: pbst},
			}},
		}},
	}

	// Converting to Json and then to Yaml before creating the CR
	rateLimitClusterJson, err := ResourcesToJSON(&rateLimitCluster)
	if err != nil {
		return fmt.Errorf("Failed to convert envoy rate limiting cluster configuration to JSON %v", err)
	}

	yamlRateLimitCluster, err := yaml.JSONToYAML(rateLimitClusterJson)
	if err != nil {
		return fmt.Errorf("Failed to convert envoy rate limiting cluster JSON configuration to YAML %v", err)
	}

	clusterJson, err := ResourcesToJSON(&cluster)
	if err != nil {
		return fmt.Errorf("Failed to convert envoy cluster configuration to JSON %v", err)
	}

	yamlCluster, err := yaml.JSONToYAML(clusterJson)
	if err != nil {
		return fmt.Errorf("Failed to convert envoy cluster JSON configuration to YAML %v", err)
	}

	listenerJson, err := ResourcesToJSON(envoyListener)
	if err != nil {
		return fmt.Errorf("Failed to convert envoy envoyListener configuration to JSON %v", err)
	}

	yamlListener, err := yaml.JSONToYAML(listenerJson)
	if err != nil {
		return fmt.Errorf("Failed to convert envoy envoyListener JSON configuration to YAML %v", err)
	}

	envoyconfig := &marin3rv1alpha.EnvoyConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configTarget,
			Namespace: config.GetNamespace(),
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, client, envoyconfig, func() error {
		owner.AddIntegreatlyOwnerAnnotations(envoyconfig, &installation)
		envoyconfig.Spec.NodeID = configTarget
		envoyconfig.Spec.Serialization = "yaml"
		envoyconfig.Spec.EnvoyResources = &marin3rv1alpha.EnvoyResources{
			Clusters: []marin3rv1alpha.EnvoyResource{
				{
					Name:  configTarget,
					Value: string(yamlCluster),
				},
				{
					Name:  "ratelimit",
					Value: string(yamlRateLimitCluster),
				},
			},
			Listeners: []marin3rv1alpha.EnvoyResource{
				{
					Name:  "http",
					Value: string(yamlListener),
				},
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Failed to create envoy config CR %v", err)
	}

	return nil
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
