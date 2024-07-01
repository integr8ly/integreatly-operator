package common

import (
	goctx "context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	threescalev1 "github.com/3scale/3scale-operator/apis/capabilities/v1alpha1"
	portaClient "github.com/3scale/3scale-porta-go-client/client"
	projectv1 "github.com/openshift/api/project/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
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
	// poll to make sure the project is deleted from H30 and H29 before attempting to create again
	project := &projectv1.Project{}
	err := wait.PollUntilContextTimeout(goctx.TODO(), tenantCreatedLoopTimeout, tenantCreatedTimeout, false, func(ctx2 goctx.Context) (done bool, err error) {
		err = ctx.Client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: projectNamespace, Namespace: projectNamespace}, project)
		if err != nil {
			return true, nil
		}
		return false, err

	})
	if err != nil {
		t.Logf("failed to check project %v", err)
	}
	// make project
	project, err = makeProject(ctx, projectNamespace)
	if err != nil {
		t.Fatalf("failed to create project %v", err)
	}

	//make secret
	secret, err := createSecret(ctx, map[string][]byte{
		"admin_password": []byte("admin"),
	}, projectAdminSecret, projectNamespace)
	if err != nil {
		t.Fatalf("failed to create secret %v", err)
	}

	//get system url
	route, err := getRoutes(ctx, masterRoute, ThreeScaleProductNamespace)
	if err != nil {
		t.Fatalf("failed to get route %v", err)
	}
	masterURL := fmt.Sprintf("https://%v", route.Spec.Host)

	//create tenant cr under same namespace
	tenantCR := &threescalev1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tenantCrName,
			Namespace: tenantCrNamespace,
			Annotations: map[string]string{
				"insecure_skip_verify": "true",
			},
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

	err = wait.PollUntilContextTimeout(goctx.TODO(), tenantCreatedLoopTimeout, tenantCreatedTimeout, false, func(ctx2 goctx.Context) (done bool, err error) {
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
	return getToken(ctx, namespace, "MASTER_ACCESS_TOKEN", "system-seed")
}

func getAdminToken(ctx *TestingContext, namespace string) (*string, error) {
	return getToken(ctx, namespace, "ADMIN_ACCESS_TOKEN", "system-seed")
}

func setupPortaClient(accessToken *string, host string) (*portaClient.ThreeScaleClient, error) {
	adminPortal, err := portaClient.NewAdminPortal("https", host, 443)
	if err != nil {
		return nil, fmt.Errorf("could not create admin portal %v", err)
	}
	// #nosec G402
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

	err := wait.PollUntilContextTimeout(goctx.TODO(), time.Second*5, time.Minute*2, false, func(ctx2 goctx.Context) (done bool, err error) {
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
