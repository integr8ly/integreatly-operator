package common

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/integr8ly/integreatly-operator/test/resources"
	v1 "github.com/openshift/api/route/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

const (
	grafanaCredsUsername = "customer-admin-1"
	grafanaCredsPassword = "Password1"
)

func TestGrafanaExternalRouteAccessible(t *testing.T, ctx *TestingContext) {
	httpClient, err := buildHTTPClientFromContext(ctx)
	if err != nil {
		t.Fatal("failed to create test http client", err)
	}
	//reconcile idp setup
	if err := createTestingIDP(context.TODO(), ctx.Client, httpClient, ctx.SelfSignedCerts); err != nil {
		t.Fatal("failed to reconcile testing idp", err)
	}
	grafanaRootHostname, err := getGrafanaRoute(ctx.Client)
	if err != nil {
		t.Fatal("failed to get grafana route", err)
	}
	//perform a request that we expect to be forbidden initially
	forbiddenReq, err := http.NewRequest(http.MethodGet, grafanaRootHostname, nil)
	if err != nil {
		t.Fatal("failed to build request for grafana", err)
	}
	forbiddenResp, err := http.DefaultClient.Do(forbiddenReq)
	if err != nil {
		t.Fatal("failed to perform expected forbidden request", err)
	}
	if forbiddenResp.StatusCode != http.StatusForbidden {
		t.Fatalf("unexpected status code on forbidden request, got=%+v", forbiddenResp)
	}
	//retrieve an openshift oauth proxy cookie
	grafanaOauthHostname := fmt.Sprintf("%s/oauth/start", grafanaRootHostname)
	if err := resources.DoAuthOpenshiftUser(grafanaOauthHostname, grafanaCredsUsername, grafanaCredsPassword, httpClient); err != nil {
		t.Fatal("failed to login through openshift oauth proxy", err)
	}
	successResp, err := httpClient.Get(grafanaRootHostname)
	if err != nil {
		t.Fatal("failed to perform test request to grafana", err)
	}
	defer successResp.Body.Close()
	if successResp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code on success request, got=%+v", successResp)
	}
}

func TestGrafanaExternalRouteDashboardExist(t *testing.T, ctx *TestingContext) {
	httpClient, err := buildHTTPClientFromContext(ctx)
	if err != nil {
		t.Fatal("failed to create test http client", err)
	}
	//reconcile idp setup
	if err := createTestingIDP(context.TODO(), ctx.Client, httpClient, ctx.SelfSignedCerts); err != nil {
		t.Fatal("failed to reconcile testing idp", err)
	}
	grafanaRootHostname, err := getGrafanaRoute(ctx.Client)
	if err != nil {
		t.Fatal("failed to get grafana route", err)
	}
	//retrieve an openshift oauth proxy cookie
	grafanaOauthHostname := fmt.Sprintf("%s/oauth/start", grafanaRootHostname)
	if err = resources.DoAuthOpenshiftUser(grafanaOauthHostname, grafanaCredsUsername, grafanaCredsPassword, httpClient); err != nil {
		t.Fatal("failed to login through openshift oauth proxy", err)
	}
	//get dashboards for grafana from the external route
	grafanaDashboardsUrl := fmt.Sprintf("%s/api/search", grafanaRootHostname)
	dashboardResp, err := httpClient.Get(grafanaDashboardsUrl)
	if err != nil {
		t.Fatal("failed to perform test request to grafana", err)
	}
	defer dashboardResp.Body.Close()
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
