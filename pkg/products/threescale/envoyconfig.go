package threescale

import (
	"fmt"

	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	v2route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	lua "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/lua/v3"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"
	ptypes "github.com/golang/protobuf/ptypes"
	wrappers "github.com/golang/protobuf/ptypes/wrappers"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/ratelimit"
	structpb "google.golang.org/protobuf/types/known/structpb"
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
var tsRatelimitDescriptor = route.RateLimit{
	Stage: &wrappers.UInt32Value{Value: 0},
	Actions: []*route.RateLimit_Action{{
		ActionSpecifier: &route.RateLimit_Action_GenericKey_{
			GenericKey: &route.RateLimit_Action_GenericKey{
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
var multiTenantRatelimitDescriptor = route.RateLimit{
	Stage: &wrappers.UInt32Value{Value: 0},
	Actions: []*route.RateLimit_Action{
		{
			ActionSpecifier: &route.RateLimit_Action_HeaderValueMatch_{
				HeaderValueMatch: &v2route.RateLimit_Action_HeaderValueMatch{
					DescriptorValue: multitenantDescriptorKey,
					Headers: []*v2route.HeaderMatcher{
						{
							Name: headerName,
							HeaderMatchSpecifier: &v2route.HeaderMatcher_SafeRegexMatch{
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
			ActionSpecifier: &route.RateLimit_Action_RequestHeaders_{
				RequestHeaders: &v2route.RateLimit_Action_RequestHeaders{
					HeaderName:    tenantHeaderName,
					DescriptorKey: tenantHeaderName,
				},
			},
		},
	},
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
var tsHTTPRateLimitFilter = hcm.HttpFilter{
	Name: "envoy.rate_limit",
	ConfigType: &hcm.HttpFilter_Config{
		Config: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"domain": {
					Kind: &structpb.Value_StringValue{
						StringValue: ratelimit.RateLimitDomain,
					},
				},
				"stage": {
					Kind: &structpb.Value_NumberValue{
						NumberValue: 0,
					},
				},
				"rate_limit_service": {
					Kind: &structpb.Value_StructValue{
						StructValue: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"grpc_service": {
									Kind: &structpb.Value_StructValue{
										StructValue: &structpb.Struct{
											Fields: map[string]*structpb.Value{
												"timeout": {
													Kind: &structpb.Value_StringValue{
														StringValue: "2s",
													},
												},
												"envoy_grpc": {
													Kind: &structpb.Value_StructValue{
														StructValue: &structpb.Struct{
															Fields: map[string]*structpb.Value{
																"cluster_name": {
																	Kind: &structpb.Value_StringValue{
																		StringValue: ratelimit.RateLimitClusterName,
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	},
}

/**
 httpFilters:
	- &tsHTTPRateLimitFilter
	- name: envoy.router
**/
func getAPICastHTTPFilters(installation *integreatlyv1alpha1.RHMI) ([]*hcm.HttpFilter, error) {
	var httpFilters []*hcm.HttpFilter

	if !integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installation.Spec.Type)) {
		httpFilters = []*hcm.HttpFilter{
			&tsHTTPRateLimitFilter,
			{
				Name: "envoy.router",
			},
		}
	} else {
		// function envoy_on_request(request_handle)
		// host = request_handle:headers():get('Host')
		// local headers = request_handle:headers()
		// split_string = Split(host, "-apicast")
		// headers:add('tenant',split_string[1])
		// end
		// function Split(s, delimiter)
		// result = {};
		// for match in (s..delimiter):gmatch("(.-)"..delimiter) do
		// table.insert(result, match);
		// end
		// return result;
		// end
		luaFunctionToAddTSHeaders := "function envoy_on_request(request_handle) host = request_handle:headers():get('Host') local headers = request_handle:headers() split_string = Split(host, '-apicast') headers:add('tenant', split_string[1]) end function Split(s, delimiter) result = {}; for match in (s..delimiter):gmatch('(.-)'..delimiter) do table.insert(result, match); end return result; end"

		luaFilter := &lua.Lua{
			InlineCode: luaFunctionToAddTSHeaders,
		}
		pbst, err := ptypes.MarshalAny(luaFilter)
		if err != nil {
			return nil, fmt.Errorf("failed to convert HttpConnectionManager for rate limiting: %v", err)
		}

		httpFilters = []*hcm.HttpFilter{
			{
				Name: "envoy.filters.http.lua",
				ConfigType: &hcm.HttpFilter_TypedConfig{
					TypedConfig: pbst,
				},
			},
			&tsHTTPRateLimitFilter,
			{
				Name: "envoy.router",
			},
		}
	}

	return httpFilters, nil
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
		&tsHTTPRateLimitFilter,
		{
			Name: "envoy.filters.http.lua",
			ConfigType: &hcm.HttpFilter_TypedConfig{
				TypedConfig: pbst,
			},
		},
		{
			Name: "envoy.router",
		},
	}
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
func getAPICastVirtualHosts(installation *integreatlyv1alpha1.RHMI, clusterName string) []*v2route.VirtualHost {
	virtualHost := v2route.VirtualHost{
		Name:    clusterName,
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
							Cluster: clusterName,
						},
						RateLimits: getRateLimitsPerInstallType(installation),
					},
				},
			},
		},
	}
	return []*v2route.VirtualHost{&virtualHost}
}

func getRateLimitsPerInstallType(installation *integreatlyv1alpha1.RHMI) []*route.RateLimit {
	var routes []*route.RateLimit

	if !integreatlyv1alpha1.IsRHOAMMultitenant(integreatlyv1alpha1.InstallationType(installation.Spec.Type)) {
		routes = []*route.RateLimit{&tsRatelimitDescriptor}
	} else {
		routes = []*route.RateLimit{&tsRatelimitDescriptor, &multiTenantRatelimitDescriptor}
	}

	return routes
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
func getBackendListenerVitualHosts(clusterName string) []*v2route.VirtualHost {
	virtualHosts := []*v2route.VirtualHost{
		{
			Name:    clusterName,
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
								Cluster: clusterName,
							},
							RateLimits: []*route.RateLimit{&tsRatelimitDescriptor},
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
func getListenerResourceFilters(virtualHosts []*v2route.VirtualHost, httpFilters []*hcm.HttpFilter) ([]*listener.Filter, error) {
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "ingress_http",
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: &envoyapi.RouteConfiguration{
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

	filters := []*listener.Filter{{
		Name:       "envoy.http_connection_manager",
		ConfigType: &listener.Filter_TypedConfig{TypedConfig: pbst},
	}}

	return filters, nil
}
