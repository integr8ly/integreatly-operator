package common

import (
	goctx "context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"time"

	"github.com/integr8ly/integreatly-operator/pkg/products/threescale"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
	. "github.com/onsi/ginkgo/v2"
)

var (
	threescaleLoginUser = fmt.Sprintf("%v%02d", defaultDedicatedAdminName, 1)
)

// Tests that a user in group dedicated-admins can create an integration
func Test3ScaleCrudlPermissions(t TestingTB, ctx *TestingContext) {
	By("Ensure testing IDP is configured")
	if err := createTestingIDP(t, goctx.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Fatalf("error while creating testing idp: %v", err)
	}

	// get console master url
	By("Get RHMI CR")
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	// Get the fuse host url from the rhmi status
	By("Get 3Scale URL and 3Scale clint")
	host := rhmi.Status.Stages[rhmiv1alpha1.InstallStage].Products[rhmiv1alpha1.Product3Scale].Host
	if host == "" {
		t.Log("No host route found. Creating route from `Spec.RoutingSubdomain`")
		host = fmt.Sprintf("https://3scale-admin.%v", rhmi.Spec.RoutingSubdomain)
	}
	keycloakHost := rhmi.Status.Stages[rhmiv1alpha1.InstallStage].Products[rhmiv1alpha1.ProductRHSSO].Host
	if keycloakHost == "" {
		t.Log("Keycloak host route not found")
	}
	redirectUrl := fmt.Sprintf("%v/p/admin/dashboard", host)

	tsClient := resources.NewThreeScaleAPIClient(host, keycloakHost, redirectUrl, ctx.HttpClient, ctx.Client, t)

	// Login to 3Scale
	By("Login to 3Scale and become admin")
	err = loginToThreeScale(t, host, threescaleLoginUser, TestingIdpPassword, "testing-idp", ctx.HttpClient)
	if err != nil {
		t.Fatalf("Failed to log into 3Scale: %v", err)
	}

	err = waitForUserToBecome3ScaleAdmin(t, ctx, host, threescaleLoginUser)
	if err != nil {
		t.Fatalf("timout asserting 3scale user, %s, is admin for performing test: %s", threescaleLoginUser, err)
	}

	// Make sure 3Scale is available
	By("Ensure 3Scale is available")
	err = tsClient.Ping()
	if err != nil {
		t.Log("Error during making sure 3Scale is available")
	}

	// Create a product
	By("Create a product")
	r1, err := rand.Int(rand.Reader, big.NewInt(100000))
	if err != nil {
		t.Fatal("Error generating rand int")
	}
	productId, err := tsClient.CreateProduct(fmt.Sprintf("dummy-product-%v", r1.Int64()))
	if err != nil {
		t.Log("Error during create the product")
		t.Fatal(err)
	}

	// Delete the product
	By("Delete the product")
	err = tsClient.DeleteProduct(productId)
	if err != nil {
		t.Log("Error during deleting the product")
		t.Fatal(err)
	}
}

func waitForUserToBecome3ScaleAdmin(t TestingTB, ctx *TestingContext, host, userName string) error {
	err := wait.PollUntilContextTimeout(goctx.TODO(), time.Second*10, time.Minute*5, true, func(ctx2 goctx.Context) (done bool, err error) {
		users, err := getUsersIn3scale(ctx, host)
		if err != nil {
			t.Logf("Error getting 3scale users: %s", err)
			return false, nil
		}

		for _, user := range users.Users {
			if user.UserDetails.Username == userName && user.UserDetails.Role == "member" {
				t.Logf("user, %s, is not an admin in 3scale", userName)
				return false, nil
			}

		}

		return true, nil
	})

	if err != nil {
		return err
	}

	return nil
}

func getUsersIn3scale(ctx *TestingContext, host string) (*threescale.Users, error) {
	systemSeedSecret := &corev1.Secret{}

	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: "system-seed", Namespace: ThreeScaleProductNamespace}, systemSeedSecret); err != nil {
		return nil, fmt.Errorf("unable to get system seed secret")
	}

	adminAccessToken := string(systemSeedSecret.Data["ADMIN_ACCESS_TOKEN"])

	resp, err := ctx.HttpClient.Get(fmt.Sprintf("%s/admin/api/users.json?access_token=%s", host, adminAccessToken))
	if err != nil {
		return nil, fmt.Errorf("unable to get list of users via api: %s", err)
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			fmt.Println("request body close error: ", err)
		}
	}()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read api response: %s", err)
	}

	users := threescale.Users{}
	err = json.Unmarshal(bytes, &users)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal json response to struct: %s", err)
	}

	return &users, nil
}
