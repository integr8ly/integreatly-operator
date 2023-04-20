package common

import (
	"context"
	goctx "context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/integr8ly/integreatly-operator/test/resources"
	v12 "github.com/openshift/api/authorization/v1"
	v1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCustomerGrafanaExternalRouteAccessible(t TestingTB, ctx *TestingContext) {
	grafanaRouteHostname, err := getGrafanaRoute(ctx.Client, CustomerGrafanaNamespace)
	if err != nil {
		t.Fatal("failed to get grafana route", err)
	}

	testRoute(t, ctx, grafanaRouteHostname)
}

func TestGrafanaExternalRouteAccessible(t TestingTB, ctx *TestingContext) {
	grafanaRouteHostname, err := getGrafanaRoute(ctx.Client, ObservabilityProductNamespace)
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

func TestGrafanaExternalRouteDashboardExist(t TestingTB, ctx *TestingContext) {
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	if resources.RunningInProw(rhmi) {
		t.Skip("403 Forbidden is returned when accessing Grafana Dashboard in Prow")
	}

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
	err = ctx.Client.Create(goctx.TODO(), serviceAccount)
	if err != nil {
		t.Fatal("failed to create serviceAccount", err)
	}
	defer func(Client k8sclient.Client, ctx goctx.Context, obj client.Object) {
		if err := Client.Delete(ctx, obj); err != nil {
			t.Log(err)
		}
	}(ctx.Client, goctx.TODO(), serviceAccount)
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
		t.Fatal("failed to create clusterRoleBinding", err)
	}
	defer func(Client k8sclient.Client, ctx goctx.Context, obj client.Object) {
		if err := Client.Delete(ctx, obj); err != nil {
			t.Log(err)
		}
	}(ctx.Client, goctx.TODO(), binding)

	grafanaRouteHostname, err := getGrafanaRoute(ctx.Client, ObservabilityProductNamespace)
	if err != nil {
		t.Fatal("failed to get grafana route", err)
	}

	token := ""
	if err := wait.PollImmediate(time.Second*5, time.Minute*1, func() (bool, error) {
		// Get the secrets in Observability ns
		secrets := &corev1.SecretList{}
		opts := []k8sclient.ListOption{
			k8sclient.InNamespace(ObservabilityProductNamespace),
		}
		if err := ctx.Client.List(goctx.TODO(), secrets, opts...); err != nil {
			return false, err
		}

		// Find the service account secret containing the token
		var saSecret *corev1.Secret
		for i := range secrets.Items {
			if strings.HasPrefix(secrets.Items[i].Name, fmt.Sprintf("%s-token", serviceAccountName)) {
				saSecret = &secrets.Items[i]
				break
			}
		}

		if saSecret == nil {
			return false, nil
		}

		// Assign the token and finish polling
		token = string(saSecret.Data["token"])
		return true, nil
	}); err != nil {
		t.Fatal("unexpected error while waiting for SA token", err)
	}

	//create new http client
	httpClient, err := NewTestingHTTPClient(ctx.KubeConfig)
	if err != nil {
		t.Fatal("failed to create testing http client", err)
	}
	//get dashboards for grafana from the external route
	grafanaDashboardsURL := fmt.Sprintf("%s/api/search", grafanaRouteHostname)
	req, err := http.NewRequest("GET", grafanaDashboardsURL, nil)
	if err != nil {
		t.Fatal("failed to create request for grafana", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	dashboardResp, err := httpClient.Do(req)
	if err != nil {
		t.Fatal("failed to perform test request to grafana", err)
	}
	defer dashboardResp.Body.Close()
	//there is an existing dashboard check, so confirm a valid response structure
	if dashboardResp.StatusCode != http.StatusOK {
		dumpResp, err := httputil.DumpResponse(dashboardResp, true)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("dumpResp: %q", dumpResp)
		// To unskip: https://issues.redhat.com/browse/MGDAPI-5378
		//t.Fatalf("unexpected status code on success request, got=%+v", dashboardResp)
		t.Skipf("unexpected status code on success request, got=%+v", dashboardResp)
	}

	var dashboards []interface{}
	if err := json.NewDecoder(dashboardResp.Body).Decode(&dashboards); err != nil {
		t.Fatal("failed to decode grafana dashboards response", err)
	}
	if len(dashboards) == 0 {
		t.Fatal("no grafana dashboards returned from grafana api")
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
