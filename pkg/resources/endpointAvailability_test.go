package resources

import (
	"strings"
	"testing"
)

func TestNoReadyServiceEndpointsExpr(t *testing.T) {
	got := NoReadyServiceEndpointsExpr("redhat-rhoam-3scale", "apicast-production")
	if !strings.Contains(got, "kube_endpoint_address{endpoint='apicast-production', namespace='redhat-rhoam-3scale'}") {
		t.Errorf("missing endpoints metric selector: %s", got)
	}
	if !strings.Contains(got, "endpointslice=~'^apicast-production-[a-z0-9]+$'") {
		t.Errorf("missing endpointslice selector: %s", got)
	}
	if !strings.Contains(got, "or vector(0)") {
		t.Errorf("expected or vector(0) so a missing metric family does not fire: %s", got)
	}

	zync := NoReadyServiceEndpointsExpr("ns", "zync")
	zyncDB := NoReadyServiceEndpointsExpr("ns", "zync-database")
	if !strings.Contains(zync, "endpointslice=~'^zync-[a-z0-9]+$'") {
		t.Errorf("zync slice selector = %s", zync)
	}
	if !strings.Contains(zyncDB, "endpointslice=~'^zync-database-[a-z0-9]+$'") {
		t.Errorf("zync-database slice selector = %s", zyncDB)
	}
	if strings.Contains(zync, "zync-database") {
		t.Errorf("zync selector must not match zync-database: %s", zync)
	}
}
