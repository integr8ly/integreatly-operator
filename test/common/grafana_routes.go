package common

import (
	"context"
	"fmt"
	v1 "github.com/openshift/api/route/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCustomerGrafanaExternalRouteAccessible(t TestingTB, ctx *TestingContext) {
	grafanaRouteHostname, err := getGrafanaRoute(ctx.Client, CustomerGrafanaNamespace)
	if err != nil {
		t.Fatal("failed to get grafana route", err)
	}

	testRoute(t, ctx, grafanaRouteHostname)
}

func testRoute(t TestingTB, ctx *TestingContext, grafanaRouteHostname string) {
	// create new http client
	httpClient, err := NewTestingHTTPClient(ctx.KubeConfig)
	if err != nil {
		t.Fatal("failed to create testing http client", err)
	}

	grafanaMetricsEndpoint := fmt.Sprintf("%s/metrics", grafanaRouteHostname)

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

func getGrafanaRoute(c client.Client, namespace string) (string, error) {
	const (
		routeGrafanaName = "grafana-route"
	)
	Context := context.TODO()
	//get grafana openshift route
	grafanaRoute := &v1.Route{}
	if err := c.Get(Context, client.ObjectKey{Name: routeGrafanaName, Namespace: namespace}, grafanaRoute); err != nil {
		return "", fmt.Errorf("failed to get grafana route: %w", err)
	}
	//evaluate the grafana route hostname
	grafanaRouteHostname := grafanaRoute.Spec.Host
	if grafanaRoute.Spec.TLS != nil {
		grafanaRouteHostname = fmt.Sprintf("https://%s", grafanaRouteHostname)
	}
	return grafanaRouteHostname, nil
}
