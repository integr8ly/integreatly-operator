package threescale

import (
	"fmt"
	envoycorev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoylistenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoyratelimitconfigv3 "github.com/envoyproxy/go-control-plane/envoy/config/ratelimit/v3"
	envoyroutev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	lua "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/lua/v3"
	envoyratelimitv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ratelimit/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	ptypes "github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/duration"
	wrappers "github.com/golang/protobuf/ptypes/wrappers"
	"github.com/integr8ly/integreatly-operator/pkg/resources/ratelimit"
)

const (
	ApicastContainerAddress  = "127.0.0.1"
	ApicastContainerPort     = 8080
	ApicastClusterName       = "apicast-ratelimit"
	ApicastNodeID            = "apicast-ratelimit"
	ApicastEnvoyProxyAddress = "0.0.0.0"
	ApicastEnvoyProxyPort    = 8443
	ApicastListenerName      = "http"
	BackendContainerAddress  = "127.0.0.1"
	BackendContainerPort     = 3000
	BackendClusterName       = "backend-listener-ratelimit"
	BackendEnvoyProxyAddress = "0.0.0.0"
	BackendEnvoyProxyPort    = 3001
	BackendListenerName      = "http"
	BackendNodeID            = "backend-listener-ratelimit"
	BackendServiceName       = "backend-listener-proxy"
)

/*
	Defines 3scale rate limit descriptor
	rateLimits:
	- actions:
	- genericKey:
		descriptorValue: slowpath
*/
var tsRatelimitDescriptor = envoyroutev3.RateLimit{
	Stage: &wrappers.UInt32Value{Value: 0},
	Actions: []*envoyroutev3.RateLimit_Action{{
		ActionSpecifier: &envoyroutev3.RateLimit_Action_GenericKey_{
			GenericKey: &envoyroutev3.RateLimit_Action_GenericKey{
				DescriptorValue: ratelimit.RateLimitDescriptorValue,
			},
		},
	}},
}

/*
	Defines http filters for the rate limit service
	   httpFilters:
	   - config:
	       domain: apicast-ratelimit
	       rate_limit_service:
	         grpc_service:
	           envoy_grpc:
	             cluster_name: ratelimit
	           timeout: 2s
	       stage: 0
	     name: envoy.rate_limit
*/

/**
 httpFilters:
	- &tsHTTPRateLimitFilter
	- name: envoy.router
**/
func getAPICastHTTPFilters() []*hcm.HttpFilter {
	serial, _ := ptypes.MarshalAny(
		&envoyratelimitv3.RateLimit{
			Domain: ratelimit.RateLimitDomain,
			Stage:  0,
			RateLimitService: &envoyratelimitconfigv3.RateLimitServiceConfig{
				GrpcService: &envoycorev3.GrpcService{
					TargetSpecifier: &envoycorev3.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &envoycorev3.GrpcService_EnvoyGrpc{
							ClusterName: ratelimit.RateLimitClusterName,
						},
					},
					Timeout: &duration.Duration{
						Seconds: 2,
					},
				},
				TransportApiVersion: envoycorev3.ApiVersion_V3,
			},
		},
	)

	var tsHTTPRateLimitFilter = hcm.HttpFilter{
		Name:       "envoy.filters.http.ratelimit",
		ConfigType: &hcm.HttpFilter_TypedConfig{TypedConfig: serial},
	}

	httpFilters := []*hcm.HttpFilter{
		&tsHTTPRateLimitFilter,
		{
			Name: "envoy.filters.http.router",
		},
	}
	return httpFilters
}

/**
	httpFilters:
	- name: envoy.filters.http.lua
	typed_config:
		"@type": type.googleapis.com/envoy.extensions.filters.http.lua.v3.Lua
		inline_code: |
		function envoy_on_response(response_handle)
			rate_limit = response_handle:headers():get("x-envoy-ratelimited")
			if rate_limit ~= nil then
			response_handle:headers():add("3scale-rejection-reason", "limits_exceeded")
			end
		end
	- config:
		domain: apicast-ratelimit
		rate_limit_service:
		grpc_service:
			envoy_grpc:
			cluster_name: ratelimit
			timeout: 2s
		stage: 0
	  name: envoy.filters.http.ratelimit
	- name: envoy.filters.http.router
**/
func getBackendListenerHTTPFilters() ([]*hcm.HttpFilter, error) {

	// function envoy_on_response(response_handle)
	// 	rate_limit = response_handle:headers():get("x-envoy-ratelimited")
	// 	if rate_limit ~= nil then
	// 		response_handle:headers():add("3scale-rejection-reason", "limits_exceeded")
	// 	end
	// end
	luaFunctionToAddTSHeaders := "function envoy_on_response(response_handle) rate_limit = response_handle:headers():get('x-envoy-ratelimited') if rate_limit ~= nil then response_handle:headers():add('3scale-rejection-reason', 'limits_exceeded') end end"

	luaFilter := &lua.Lua{
		InlineCode: luaFunctionToAddTSHeaders,
	}
	pbst, err := ptypes.MarshalAny(luaFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to convert HttpConnectionManager for rate limiting: %v", err)
	}

	httpFilters := []*hcm.HttpFilter{
		{
			Name: "envoy.filters.http.lua",
			ConfigType: &hcm.HttpFilter_TypedConfig{
				TypedConfig: pbst,
			},
		},
	}
	httpFilters = append(httpFilters, getAPICastHTTPFilters()...)
	return httpFilters, nil
}

/**
virtualHosts:
	- domains:
	- '*'
	name: apicast-ratelimit
	routes:
	- match:
		prefix: /
		route:
		cluster: apicast-ratelimit
		rateLimits:
		- actions:
			- genericKey:
				descriptorValue: slowpath
			stage: 0
*/
func getAPICastVirtualHosts(clusterName string) []*envoyroutev3.VirtualHost {
	virtualHost := envoyroutev3.VirtualHost{
		Name:    clusterName,
		Domains: []string{"*"},

		Routes: []*envoyroutev3.Route{
			{
				Match: &envoyroutev3.RouteMatch{
					PathSpecifier: &envoyroutev3.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &envoyroutev3.Route_Route{
					Route: &envoyroutev3.RouteAction{
						ClusterSpecifier: &envoyroutev3.RouteAction_Cluster{
							Cluster: clusterName,
						},
						RateLimits: []*envoyroutev3.RateLimit{&tsRatelimitDescriptor},
					},
				},
			},
		},
	}
	return []*envoyroutev3.VirtualHost{&virtualHost}
}

/**
virtual_hosts:
	- name: backend-listener-ratelimit
	domains: ["*"]
	routes:
		- match:
			prefix: "/"
		route:
			cluster: backend-listener-ratelimit
			rate_limits:
**/
func getBackendListenerVitualHosts(clusterName string) []*envoyroutev3.VirtualHost {
	virtualHosts := []*envoyroutev3.VirtualHost{
		{
			Name:    clusterName,
			Domains: []string{"*"},

			Routes: []*envoyroutev3.Route{
				{
					Match: &envoyroutev3.RouteMatch{
						PathSpecifier: &envoyroutev3.RouteMatch_Prefix{
							Prefix: "/",
						},
					},
					Action: &envoyroutev3.Route_Route{
						Route: &envoyroutev3.RouteAction{
							ClusterSpecifier: &envoyroutev3.RouteAction_Cluster{
								Cluster: clusterName,
							},
							RateLimits: []*envoyroutev3.RateLimit{&tsRatelimitDescriptor},
						},
					},
				},
			},
		},
	}

	return virtualHosts
}

/**
        - name: envoy.http_connection_manager
          typedConfig:
            '@type': type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager
			httpFilters:
				&httpFilters
            routeConfig:
               name: local_route
			   virtualHosts: virtualHosts
			statPrefix: ingress_http
**/
func getListenerResourceFilters(virtualHosts []*envoyroutev3.VirtualHost, httpFilters []*hcm.HttpFilter) ([]*envoylistenerv3.Filter, error) {
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "ingress_http",
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: &envoyroutev3.RouteConfiguration{
				Name:         "local_route",
				VirtualHosts: virtualHosts,
			},
		},
		HttpFilters: httpFilters,
	}

	pbst, err := ptypes.MarshalAny(manager)
	if err != nil {
		return nil, fmt.Errorf("failed to convert HttpConnectionManager for rate limiting: %v", err)
	}

	filters := []*envoylistenerv3.Filter{{
		Name:       "envoy.filters.network.http_connection_manager",
		ConfigType: &envoylistenerv3.Filter_TypedConfig{TypedConfig: pbst},
	}}

	return filters, nil
}
