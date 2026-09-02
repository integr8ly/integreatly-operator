package resources

import (
	"fmt"
	"regexp"
)

// NoReadyServiceEndpointsExpr is true when a Service has no ready backends
// according to either kube-state-metrics family.
//
// OCP 4.18/4.19 export kube_endpoint_address (v1 Endpoints). OCP 4.20+ may
// export only kube_endpointslice_endpoints. Using absent() on a single metric
// false-fires on clusters that do not export that family. Count both; fire
// only when both are empty.
func NoReadyServiceEndpointsExpr(namespace, serviceName string) string {
	slice := regexp.QuoteMeta(serviceName)
	return fmt.Sprintf(
		"(count(kube_endpoint_address{endpoint='%s', namespace='%s'}) or vector(0)) + (count(kube_endpointslice_endpoints{namespace='%s', endpointslice=~'^%s-[a-z0-9]+$', ready='true'}) or vector(0)) == 0",
		serviceName, namespace, namespace, slice,
	)
}
