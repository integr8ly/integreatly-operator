package common

import (
	"context"
	goctx "context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"testing"

	v12 "github.com/openshift/api/authorization/v1"
	v1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
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
	var (
		serviceAccountName = "test"
		bindingName        = "test"
		grafanaNamespace   = NamespacePrefix + "middleware-monitoring-operator"
	)

	//create service account - its token will be used to call grafana api
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: grafanaNamespace,
			Name:      serviceAccountName,
		},
	}
	err := ctx.Client.Create(goctx.TODO(), serviceAccount)
	if err != nil {
		t.Fatal("failed to create serviceAccount", err)
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
				Namespace:  grafanaNamespace,
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
	defer ctx.Client.Delete(goctx.TODO(), binding)

	grafanaRootHostname, err := getGrafanaRoute(ctx.Client)
	if err != nil {
		t.Fatal("failed to get grafana route", err)
	}

	token := ""
	secrets, err := ctx.KubeClient.CoreV1().Secrets(grafanaNamespace).List(metav1.ListOptions{})
	if err != nil {
		t.Fatal("failed to get secrets", err)
	}
	for _, secretsItem := range secrets.Items {
		if strings.HasPrefix(secretsItem.Name, "test-token-") {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretsItem.Name,
					Namespace: grafanaNamespace,
				},
			}
			err = ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: secretsItem.Name, Namespace: grafanaNamespace}, secret)
			if err != nil {
				t.Fatal("failed to get secret", err)
			}

			if _, ok := secret.Annotations["kubernetes.io/created-by"]; !ok {
				token = string(secret.Data["token"])
				break
			}
		}
	}
	if token == "" {
		t.Fatal("failed to find token for serviceAccount")
	}

	//create new http client
	httpClient, err := NewTestingHTTPClient(ctx.KubeConfig)
	if err != nil {
		t.Fatal("failed to create testing http client", err)
	}
	//get dashboards for grafana from the external route
	grafanaDashboardsURL := fmt.Sprintf("%s/api/search", grafanaRootHostname)
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
		dumpResp, _ := httputil.DumpResponse(dashboardResp, true)
		t.Logf("dumpResp: %q", dumpResp)
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
	var (
		routeGrafanaName      = "grafana-route"
		routeGrafanaNamespace = NamespacePrefix + "middleware-monitoring-operator"
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
