package common

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/integr8ly/integreatly-operator/test/resources"
	v1 "github.com/openshift/api/route/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	grafanaCredsUsername = "customer-admin-1"
	grafanaCredsPassword = "Password1"
)

func TestGrafanaExternalRouteAccessible(t *testing.T, ctx *TestingContext) {

	grafanaRootHostname, err := getGrafanaRoute(ctx.Client)
	if err != nil {
		t.Fatal("failed to get grafana route", err)
	}

	// create new http client
	httpClient, err := NewTestingHTTPClient(ctx.KubeConfig)
	if err != nil {
		t.Fatal("failed to create testing http client", err)
	}

	grafanaMetricsEndpoint := fmt.Sprintf("%s/metrics", grafanaRootHostname)

	req, err := http.NewRequest("GET", grafanaMetricsEndpoint, nil)
	if err != nil {
		t.Fatal("failed to prepare test request to grafana", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatal("failed to perform test request to grafana", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code on request, got=%+v", resp.StatusCode)
	}
}

func TestGrafanaExternalRouteDashboardExist(t *testing.T, ctx *TestingContext) {
	//reconcile idp setup
	if err := createTestingIDP(t, context.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Fatal("failed to reconcile testing idp", err)
	}
	grafanaRootHostname, err := getGrafanaRoute(ctx.Client)
	if err != nil {
		t.Fatal("failed to get grafana route", err)
	}
	//create new http client
	httpClient, err := NewTestingHTTPClient(ctx.KubeConfig)
	if err != nil {
		t.Fatal("failed to create testing http client", err)
	}
	//retrieve an openshift oauth proxy cookie
	grafanaOauthHostname := fmt.Sprintf("%s/oauth/start", grafanaRootHostname)
	if err = resources.DoAuthOpenshiftUser(grafanaOauthHostname, grafanaCredsUsername, grafanaCredsPassword, httpClient, TestingIDPRealm, t); err != nil {
		t.Fatal("failed to login through openshift oauth proxy", err)
	}
	//get dashboards for grafana from the external route
	grafanaDashboardsUrl := fmt.Sprintf("%s/api/search", grafanaRootHostname)
	dashboardResp, err := httpClient.Get(grafanaDashboardsUrl)
	if err != nil {
		t.Fatal("failed to perform test request to grafana", err)
	}
	defer dashboardResp.Body.Close()
	//there is an existing dashboard check, so confirm a valid response structure
	if dashboardResp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code on success request, got=%+v", dashboardResp)
	}
	var dashboards []interface{}
	if err := json.NewDecoder(dashboardResp.Body).Decode(&dashboards); err != nil {
		t.Fatal("failed to decode grafana dashboards response", err)
	}
	if len(dashboards) == 0 {
		t.Fatal("no grafana dashboards returned from grafana api")
	}
}

func getGrafanaRoute(c client.Client) (string, error) {
	const (
		routeGrafanaName      = "grafana-route"
		routeGrafanaNamespace = "redhat-rhmi-middleware-monitoring-operator"
	)
	testCtx := context.TODO()
	//get grafana openshift route
	grafanaRoute := &v1.Route{}
	if err := c.Get(testCtx, client.ObjectKey{Name: routeGrafanaName, Namespace: routeGrafanaNamespace}, grafanaRoute); err != nil {
		return "", fmt.Errorf("failed to get grafana route: %w", err)
	}
	//evaluate the grafana route hostname
	grafanaRootHostname := grafanaRoute.Spec.Host
	if grafanaRoute.Spec.TLS != nil {
		grafanaRootHostname = fmt.Sprintf("https://%s", grafanaRootHostname)
	}
	return grafanaRootHostname, nil
}
