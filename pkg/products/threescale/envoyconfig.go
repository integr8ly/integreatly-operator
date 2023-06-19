package threescale

import (
	"fmt"
	envoycorev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoylistenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoyratelimitconfigv3 "github.com/envoyproxy/go-control-plane/envoy/config/ratelimit/v3"
	envoyroutev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	lua "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/lua/v3"
	envoyratelimitv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ratelimit/v3"
	router "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/wrappers"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/ratelimit"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	ApicastContainerAddress  = "127.0.0.1"
	ApicastContainerPort     = 8444
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
	headerName               = "host"
	tenantHeaderName         = "tenant"
	safeRegex                = ".*apicast.*"
	multitenantDescriptorKey = "per-mt-limit"
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
		Defines actions for multitenantcy
	        - actions:
	            - header_value_match:
	                descriptor_value: per-mt-limit
	                headers:
	                - name: tenant
	                safe_regex_match:
	                    google_re2: {}
	                    regex: ".*apicast.*"
	                - request_headers:
	                    header_name: tenant
	                    descriptor_key: tenant
*/
var multiTenantRatelimitDescriptor = envoyroutev3.RateLimit{
	Stage: &wrappers.UInt32Value{Value: 0},
	Actions: []*envoyroutev3.RateLimit_Action{
		{
			ActionSpecifier: &envoyroutev3.RateLimit_Action_HeaderValueMatch_{
				HeaderValueMatch: &envoyroutev3.RateLimit_Action_HeaderValueMatch{
					DescriptorValue: multitenantDescriptorKey,
					Headers: []*envoyroutev3.HeaderMatcher{
						{
							Name: headerName,
							HeaderMatchSpecifier: &envoyroutev3.HeaderMatcher_SafeRegexMatch{
								SafeRegexMatch: &matcher.RegexMatcher{
									EngineType: &matcher.RegexMatcher_GoogleRe2{},
									Regex:      safeRegex,
								},
							},
						},
					},
				},
			},
		},
		{
			ActionSpecifier: &envoyroutev3.RateLimit_Action_RequestHeaders_{
				RequestHeaders: &envoyroutev3.RateLimit_Action_RequestHeaders{
					HeaderName:    tenantHeaderName,
					DescriptorKey: tenantHeaderName,
				},
			},
		},
	},
}

/*
*

	 httpFilters:
		- &tsHTTPRateLimitFilter
		- name: envoy.filters.http.grpc_http1_reverse_bridge
		- name: envoy.filters.http.router

*
*/
func getAPICastHTTPFilters() ([]*hcm.HttpFilter, error) {
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
		     name: envoy.envoy.filters.http.ratelimit
	*/
	ratelimitSerial, err := anypb.New(
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

	if err != nil {
		return nil, fmt.Errorf("failed to convert rate limit filter for rate limiting")
	}

	var tsHTTPRateLimitFilter = hcm.HttpFilter{
		Name:       "envoy.filters.http.ratelimit",
		ConfigType: &hcm.HttpFilter_TypedConfig{TypedConfig: ratelimitSerial},
	}

	routerSerial, err := anypb.New(
		&router.Router{},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to convert router filter for Apicast ratelimit envoy configuration")
	}

	var routerFiler = hcm.HttpFilter{
		Name:       "envoy.filters.http.router",
		ConfigType: &hcm.HttpFilter_TypedConfig{TypedConfig: routerSerial},
	}

	httpFilters := []*hcm.HttpFilter{
		&tsHTTPRateLimitFilter,
		&routerFiler,
	}

	return httpFilters, nil
}

/*
function envoy_on_request(request_handle)
host = request_handle:headers():get('Host')
local headers = request_handle:headers()
split_string = Split(host, "-apicast")
headers:add('tenant',split_string[1])
end
function Split(s, delimiter)
result = {};
for match in (s..delimiter):gmatch("(.-)"..delimiter) do
table.insert(result, match);
end
return result;
end
*/
func getMultitenantAPICastHTTPFilters() ([]*hcm.HttpFilter, error) {

	luaFunctionToAddTSHeaders := "function envoy_on_request(request_handle) host = request_handle:headers():get('Host') local headers = request_handle:headers() split_string = Split(host, '-apicast') headers:add('tenant', split_string[1]) end function Split(s, delimiter) result = {}; for match in (s..delimiter):gmatch('(.-)'..delimiter) do table.insert(result, match); end return result; end"

	luaFilter := &lua.Lua{
		InlineCode: luaFunctionToAddTSHeaders,
	}
	pbst, err := anypb.New(luaFilter)
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

	filters, err := getAPICastHTTPFilters()
	if err != nil {
		return nil, err
	}

	httpFilters = append(httpFilters, filters...)

	return httpFilters, nil
}

/*
*

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

*
*/
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
	pbst, err := anypb.New(luaFilter)
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
	filters, err := getAPICastHTTPFilters()
	if err != nil {
		return nil, err
	}

	httpFilters = append(httpFilters, filters...)
	return httpFilters, nil
}

/*
*
virtualHosts:
  - domains:
  - '*'
    name: apicast-ratelimit
    routes:
  - match:
    prefix: /
    route:
    cluster: apicast-ratelimit
    timeout: 75s
    rateLimits:
  - actions:
  - genericKey:
    descriptorValue: slowpath
    stage: 0
*/
func getAPICastVirtualHosts(installation *integreatlyv1alpha1.RHMI, clusterName string) []*envoyroutev3.VirtualHost {
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
						Timeout: &duration.Duration{
							Seconds: 75,
						},
						RateLimits: getRateLimitsPerInstallType(installation),
					},
				},
			},
		},
	}
	return []*envoyroutev3.VirtualHost{&virtualHost}
}

func getRateLimitsPerInstallType(installation *integreatlyv1alpha1.RHMI) []*envoyroutev3.RateLimit {
	var routes []*envoyroutev3.RateLimit

	if !integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installation.Spec.Type)) {
		routes = []*envoyroutev3.RateLimit{&tsRatelimitDescriptor}
	} else {
		routes = []*envoyroutev3.RateLimit{&tsRatelimitDescriptor, &multiTenantRatelimitDescriptor}
	}

	return routes
}

/*
*
virtual_hosts:
  - name: backend-listener-ratelimit
    domains: ["*"]
    routes:
  - match:
    prefix: "/"
    route:
    cluster: backend-listener-ratelimit
    timeout: 75s
    rate_limits:

*
*/
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
							Timeout: &duration.Duration{
								Seconds: 75,
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

/*
*
  - name: envoy.filters.network.http_connection_manager
    typedConfig:
    '@type': type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
    httpFilters:
    &httpFilters
    routeConfig:
    name: local_route
    virtualHosts: virtualHosts
    statPrefix: ingress_http
    httpProtocolOptions:
    enableTrailers: true

*
*/
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
		HttpProtocolOptions: &envoycorev3.Http1ProtocolOptions{
			EnableTrailers: true,
		},
		HttpFilters: httpFilters,
	}

	pbst, err := anypb.New(manager)
	if err != nil {
		return nil, fmt.Errorf("failed to convert HttpConnectionManager for rate limiting: %v", err)
	}

	filters := []*envoylistenerv3.Filter{{
		Name:       "envoy.filters.network.http_connection_manager",
		ConfigType: &envoylistenerv3.Filter_TypedConfig{TypedConfig: pbst},
	}}

	return filters, nil
}
