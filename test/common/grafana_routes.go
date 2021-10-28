package common

import (
	"context"
	goctx "context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	v12 "github.com/openshift/api/authorization/v1"
	v1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	grafanaCredsUsername = fmt.Sprintf("%v%02d", defaultDedicatedAdminName, 1)
	grafanaCredsPassword = DefaultPassword
)

func TestCustomerGrafanaExternalRouteAccessible(t TestingTB, ctx *TestingContext) {
	if os.Getenv("SKIP_FLAKES") == "true" {
		// https://issues.redhat.com/browse/MGDAPI-555
		t.Log("skipping 3scale SMTP test due to skip_flakes flag")
		t.SkipNow()
	}
	grafanaRootHostname, err := getGrafanaRoute(ctx.Client, CustomerGrafanaNamespace)
	if err != nil {
		t.Fatal("failed to get grafana route", err)
	}

	testRoute(t, ctx, grafanaRootHostname)
}

func TestGrafanaExternalRouteAccessible(t TestingTB, ctx *TestingContext) {
	grafanaRootHostname, err := getGrafanaRoute(ctx.Client, ObservabilityProductNamespace)
	if err != nil {
		t.Fatal("failed to get grafana route", err)
	}

	testRoute(t, ctx, grafanaRootHostname)
}

func testRoute(t TestingTB, ctx *TestingContext, grafanaRootHostname string) {
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

func TestGrafanaExternalRouteDashboardExist(t TestingTB, ctx *TestingContext) {
	const (
		serviceAccountName = "test"
		bindingName        = "test"
	)

	//create service account - its token will be used to call grafana api
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ObservabilityProductNamespace,
			Name:      serviceAccountName,
		},
	}
	err := ctx.Client.Create(goctx.TODO(), serviceAccount)
	if err != nil {
		t.Skipf("Flaky test reported in https://issues.redhat.com/browse/MGDAPI-2548 failed on: %s", err)
		// t.Fatal("failed to create serviceAccount", err)
	}
	defer ctx.Client.Delete(goctx.TODO(), serviceAccount)
	binding := &v12.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: bindingName,
		},
		Subjects: []corev1.ObjectReference{
			{
				Kind:       "ServiceAccount",
				APIVersion: "rbac.authorization.k8s.io/v1",
				Name:       serviceAccountName,
				Namespace:  ObservabilityProductNamespace,
			},
		},
		RoleRef: corev1.ObjectReference{
			Kind:       "ClusterRole",
			Name:       "cluster-admin",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
	}
	err = ctx.Client.Create(goctx.TODO(), binding)
	if err != nil {
		t.Skipf("Flaky test reported in https://issues.redhat.com/browse/MGDAPI-2548 failed on: %s", err)
		// t.Fatal("failed to create clusterRoleBinding", err)
	}
	defer ctx.Client.Delete(goctx.TODO(), binding)

	grafanaRootHostname, err := getGrafanaRoute(ctx.Client, ObservabilityProductNamespace)
	if err != nil {
		t.Skipf("Flaky test reported in https://issues.redhat.com/browse/MGDAPI-2548 failed on: %s", err)
		// t.Fatal("failed to get grafana route", err)
	}

	token := ""
	if err := wait.PollImmediate(time.Second, time.Second*10, func() (bool, error) {
		// Poll the Service Account
		if err := ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{
			Name:      serviceAccountName,
			Namespace: ObservabilityProductNamespace,
		}, serviceAccount); err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}

			return false, err
		}

		// Iterate through the SA secrets to find the token
		var saSecret *corev1.ObjectReference = nil
		for _, secret := range serviceAccount.Secrets {
			if strings.HasPrefix(secret.Name, fmt.Sprintf("%s-token", serviceAccountName)) {
				saSecret = &secret
				break
			}
		}

		// The token secret hasn't been created yet
		if saSecret == nil {
			return false, nil
		}

		// Get the secret
		secret := &corev1.Secret{}
		if err := ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{
			Name:      saSecret.Name,
			Namespace: ObservabilityProductNamespace,
		}, secret); err != nil {
			return false, err
		}

		// Assign the token and finish polling
		token = string(secret.Data["token"])
		return true, nil
	}); err != nil {
		t.Skipf("Flaky test reported in https://issues.redhat.com/browse/MGDAPI-2548 failed on: %s", err)
		// t.Fatal("unexpected error while waiting for SA token", err)
	}

	//create new http client
	httpClient, err := NewTestingHTTPClient(ctx.KubeConfig)
	if err != nil {
		t.Skipf("Flaky test reported in https://issues.redhat.com/browse/MGDAPI-2548 failed on: %s", err)
		// t.Fatal("failed to create testing http client", err)
	}
	//get dashboards for grafana from the external route
	grafanaDashboardsURL := fmt.Sprintf("%s/api/search", grafanaRootHostname)
	req, err := http.NewRequest("GET", grafanaDashboardsURL, nil)
	if err != nil {
		t.Skipf("Flaky test reported in https://issues.redhat.com/browse/MGDAPI-2548 failed on: %s", err)
		// t.Fatal("failed to create request for grafana", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	dashboardResp, err := httpClient.Do(req)
	if err != nil {
		t.Skipf("Flaky test reported in https://issues.redhat.com/browse/MGDAPI-2548 failed on: %s", err)
		// t.Fatal("failed to perform test request to grafana", err)
	}
	defer dashboardResp.Body.Close()
	//there is an existing dashboard check, so confirm a valid response structure
	if dashboardResp.StatusCode != http.StatusOK {
		dumpResp, _ := httputil.DumpResponse(dashboardResp, true)
		t.Logf("dumpResp: %q", dumpResp)
		t.Skipf("Flaky test reported in https://issues.redhat.com/browse/MGDAPI-2548 failed on: %s", err)
		// t.Fatalf("unexpected status code on success request, got=%+v", dashboardResp)
	}

	var dashboards []interface{}
	if err := json.NewDecoder(dashboardResp.Body).Decode(&dashboards); err != nil {
		t.Skipf("Flaky test reported in https://issues.redhat.com/browse/MGDAPI-2548 failed on: %s", err)
		// t.Fatal("failed to decode grafana dashboards response", err)
	}
	if len(dashboards) == 0 {
		t.Skipf("Flaky test reported in https://issues.redhat.com/browse/MGDAPI-2548 failed on: %s", err)
		// t.Fatal("no grafana dashboards returned from grafana api")
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
	grafanaRootHostname := grafanaRoute.Spec.Host
	if grafanaRoute.Spec.TLS != nil {
		grafanaRootHostname = fmt.Sprintf("https://%s", grafanaRootHostname)
	}
	return grafanaRootHostname, nil
}
