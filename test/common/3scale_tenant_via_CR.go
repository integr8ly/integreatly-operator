package common

import (
	goctx "context"
	"fmt"

	"strings"
	"time"

	"crypto/tls"
	threescalev1 "github.com/3scale/3scale-operator/pkg/apis/capabilities/v1alpha1"
	portaClient "github.com/3scale/3scale-porta-go-client/client"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var ()

const (
	tenantCreatedLoopTimeout = time.Second * 5
	tenantCreatedTimeout     = time.Minute * 10
	masterRoute              = "master.apps"
	tenantUsername           = "tenant-test-username"
	tenantOrg                = "tenant-org"
	tenantOutputSecret       = "tenant-output"
	tenantEmail              = "awesomeTenant@email.com"
	tenantCrName             = "tenant-cr"
	projectNamespace         = "test-project"
	tenantCrNamespace        = projectNamespace
	projectAdminSecret       = "project-secret"
)

// Tests that a user in group dedicated-admins can create an integration
func Test3scaleTenantViaCr(t TestingTB, ctx *TestingContext) {
	// make project
	project, err := makeProject(ctx)
	if err != nil {
		t.Fatalf("failed to create project %v", err)
	}

	//make secret
	secret, err := genSecret(ctx, map[string][]byte{
		"admin_password": []byte("admin"),
	})
	if err != nil {
		t.Fatalf("failed to create secret %v", err)
	}

	//get system url
	route, err := getRoutes(ctx, masterRoute)
	if err != nil {
		t.Fatalf("failed to get route %v", err)
	}
	masterURL := fmt.Sprintf("https://%v", route.Spec.Host)

	//create tenant cr under same namespace
	tenantCR := &threescalev1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tenantCrName,
			Namespace: tenantCrNamespace,
		},
		Spec: threescalev1.TenantSpec{
			Email: tenantEmail,
			MasterCredentialsRef: corev1.SecretReference{
				Name:      "system-seed",
				Namespace: ThreeScaleProductNamespace,
			},
			OrganizationName: tenantOrg,
			PasswordCredentialsRef: corev1.SecretReference{
				Name:      projectAdminSecret,
				Namespace: projectNamespace,
			},
			SystemMasterUrl: masterURL,
			TenantSecretRef: corev1.SecretReference{
				Name:      tenantOutputSecret,
				Namespace: projectNamespace,
			},
			Username: tenantUsername,
		},
	}
	if err := ctx.Client.Create(goctx.TODO(), tenantCR); err != nil {
		t.Fatalf("failed to create tenant cr with error: %v", err)
	}

	// get access token
	accessToken, err := getMasterToken(ctx, ThreeScaleProductNamespace)
	if err != nil {
		t.Fatalf("failed to get master token:%v", err)
	}

	// setup portaClient
	threescaleClient, err := setupPortaClient(accessToken, route.Spec.Host)
	if err != nil {
		t.Fatalf("failed to setup portaClient: %v", err)
	}

	// get tenantID
	tenantId, err := getTenantID(ctx)
	if err != nil {
		t.Fatalf("failed to get tenantID: %v", err)
	}

	err = wait.Poll(tenantCreatedLoopTimeout, tenantCreatedTimeout, func() (done bool, err error) {
		state, err := getTenantState(threescaleClient, tenantId)

		if err != nil {
			return true, err
		}

		if state != "approved" {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("Failed to get tenant state: %v", err)
	}

	// delete tenantCR
	if err := ctx.Client.Delete(goctx.TODO(), tenantCR); err != nil {
		t.Fatalf("failed to delete tenant cr with error: %v", err)
	}

	// delete secret
	if err := ctx.Client.Delete(goctx.TODO(), secret); err != nil {
		t.Fatalf("Failed to delete secret: %v", err)
	}
	// delete namespace
	if err := ctx.Client.Delete(goctx.TODO(), project); err != nil {
		t.Fatalf("Failed to delete testing namespace with error: %v", err)
	}
}

func getTenantState(threescaleClient *portaClient.ThreeScaleClient, tenantID int64) (string, error) {
	tenant, err := threescaleClient.ShowTenant(tenantID)
	if err != nil {
		return "", fmt.Errorf("failed to get tenant from 3scale API:%v", err)
	}

	return tenant.Signup.Account.State, nil
}

func getMasterToken(ctx *TestingContext, namespace string) (*string, error) {
	return getToken(ctx, namespace, "MASTER_ACCESS_TOKEN")
}

func getAdminToken(ctx *TestingContext, namespace string) (*string, error) {
	return getToken(ctx, namespace, "ADMIN_ACCESS_TOKEN")
}

func getToken(ctx *TestingContext, namespace, tokenType string) (*string, error) {
	token := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "system-seed",
		},
	}
	err := ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: token.Name, Namespace: namespace}, token)
	if err != nil {
		return nil, err
	}
	accessToken := string(token.Data[tokenType])
	return &accessToken, nil
}

func makeProject(ctx *TestingContext) (*projectv1.Project, error) {
	project := &projectv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: projectNamespace,
		},
	}
	if err := ctx.Client.Create(goctx.TODO(), project); err != nil {
		return project, fmt.Errorf("failed to create testing namespace with error: %v", err)
	}

	return project, nil
}

func getRoutes(ctx *TestingContext, routeName string) (routev1.Route, error) {
	routes := &routev1.RouteList{}

	routeFound := routev1.Route{}
	err := ctx.Client.List(goctx.TODO(), routes, &k8sclient.ListOptions{
		Namespace: ThreeScaleProductNamespace,
	})

	if err != nil {
		return routeFound, fmt.Errorf("failed to get 3scale routes with error: %v", err)
	}

	for _, route := range routes.Items {
		if strings.Contains(route.Spec.Host, routeName) {
			routeFound = route
		}
	}

	return routeFound, nil
}

func genSecret(ctx *TestingContext, datamap map[string][]byte) (*corev1.Secret, error) {
	secretRef := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      projectAdminSecret,
			Namespace: projectNamespace,
		},
		Data: datamap,
	}
	if err := ctx.Client.Create(goctx.TODO(), secretRef); err != nil {
		return secretRef, fmt.Errorf("failed to create secret with error: %v", err)
	}

	return secretRef, nil
}

func setupPortaClient(accessToken *string, host string) (*portaClient.ThreeScaleClient, error) {
	adminPortal, err := portaClient.NewAdminPortal("https", host, 443)
	if err != nil {
		return nil, fmt.Errorf("could not create admin portal %v", err)
	}

	insecureClient := &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // gosec G402 override exclued
		},
	}

	threescaleClient := portaClient.NewThreeScale(adminPortal, *accessToken, insecureClient)

	return threescaleClient, nil
}

func getTenantID(ctx *TestingContext) (int64, error) {
	var tenantID int64
	tenantCR := &threescalev1.Tenant{}

	err := wait.Poll(time.Second*5, time.Minute*2, func() (done bool, err error) {
		err = ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: tenantCrName, Namespace: tenantCrNamespace}, tenantCR)
		if err != nil {
			return true, err
		}

		if tenantCR.Status.TenantId == 0 {
			return false, nil
		}

		tenantID = tenantCR.Status.TenantId
		return true, nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get tenant state: %v", err)
	}

	return tenantID, nil
}
