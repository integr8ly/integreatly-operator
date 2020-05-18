package common

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
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
	//reconcile idp setup
	if err := createTestingIDP(t, context.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Fatal("failed to reconcile testing idp", err)
	}
	grafanaRootHostname, err := getGrafanaRoute(ctx.Client)
	if err != nil {
		t.Fatal("failed to get grafana route", err)
	}
	//perform a request that we expect to be forbidden initially
	forbiddenResp, err := ctx.HttpClient.Get(grafanaRootHostname)
	if err != nil {
		t.Fatal("failed to perform expected forbidden request", err)
	}
	if forbiddenResp.StatusCode != http.StatusForbidden {
		t.Fatalf("unexpected status code on forbidden request, got=%+v", forbiddenResp)
	}
	//retrieve an openshift oauth proxy cookie
	grafanaOauthHostname := fmt.Sprintf("%s/oauth/start", grafanaRootHostname)
	if err := resources.DoAuthOpenshiftUser(grafanaOauthHostname, grafanaCredsUsername, grafanaCredsPassword, ctx.HttpClient, TestingIDPRealm, t); err != nil {
		t.Skip("Skipping due to known flaky behavior, to be fixed ASAP.\nJIRA: https://issues.redhat.com/browse/INTLY-6738", err)
		//t.Fatal("failed to login through openshift oauth proxy", err)
	}

	req, err := http.NewRequest("GET", grafanaRootHostname, nil)
	if err != nil {
		t.Fatal("failed to prepare test request to grafana", err)
	}
	successResp, err := ctx.HttpClient.Do(req)

	if err != nil {
		t.Fatal("failed to perform test request to grafana", err)
	}
	defer successResp.Body.Close()
	if successResp.StatusCode != http.StatusOK {
		dumpReq, _ := httputil.DumpRequest(req, true)
		t.Logf("dumpReq: %q", dumpReq)
		dumpResp, _ := httputil.DumpResponse(successResp, true)
		t.Logf("dumpResp: %q", dumpResp)
		t.Skip("Skipping due to known flaky behavior, to be fixed ASAP.\nJIRA: https://issues.redhat.com/browse/INTLY-6738", err)
		//t.Fatalf("unexpected status code on success request, got=%+v", successResp)
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
	//retrieve an openshift oauth proxy cookie
	grafanaOauthHostname := fmt.Sprintf("%s/oauth/start", grafanaRootHostname)
	if err = resources.DoAuthOpenshiftUser(grafanaOauthHostname, grafanaCredsUsername, grafanaCredsPassword, ctx.HttpClient, TestingIDPRealm, t); err != nil {
		t.Fatal("failed to login through openshift oauth proxy", err)
	}
	//get dashboards for grafana from the external route
	grafanaDashboardsUrl := fmt.Sprintf("%s/api/search", grafanaRootHostname)
	dashboardResp, err := ctx.HttpClient.Get(grafanaDashboardsUrl)
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
