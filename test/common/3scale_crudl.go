package common

import (
	goctx "context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"time"

	"github.com/integr8ly/integreatly-operator/pkg/products/threescale"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
)

var (
	threescaleLoginUser = fmt.Sprintf("%v%02d", defaultDedicatedAdminName, 1)
)

// Tests that a user in group dedicated-admins can create an integration
func Test3ScaleCrudlPermissions(t TestingTB, ctx *TestingContext) {
	if err := createTestingIDP(t, goctx.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Skip("Skipping due to known flaky behavior due to, reported in Jira: https://issues.redhat.com/browse/MGDAPI-1806")
		//t.Fatalf("error while creating testing idp: %v", err)
	}

	// get console master url
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Skip("Skipping due to known flaky behavior due to, reported in Jira: https://issues.redhat.com/browse/MGDAPI-1806")
		//t.Fatalf("error getting RHMI CR: %v", err)
	}

	// Get the fuse host url from the rhmi status
	host := rhmi.Status.Stages[rhmiv1alpha1.ProductsStage].Products[rhmiv1alpha1.Product3Scale].Host
	if host == "" {
		host = fmt.Sprintf("https://3scale-admin.%v", rhmi.Spec.RoutingSubdomain)
	}
	keycloakHost := rhmi.Status.Stages[rhmiv1alpha1.AuthenticationStage].Products[rhmiv1alpha1.ProductRHSSO].Host
	redirectUrl := fmt.Sprintf("%v/p/admin/dashboard", host)

	tsClient := resources.NewThreeScaleAPIClient(host, keycloakHost, redirectUrl, ctx.HttpClient, ctx.Client, t)

	// Login to 3Scale
	err = loginToThreeScale(t, host, threescaleLoginUser, DefaultPassword, "testing-idp", ctx.HttpClient)
	if err != nil {
		t.Skip("Skipping due to known flaky behavior due to, reported in Jira: https://issues.redhat.com/browse/MGDAPI-1806")
		//t.Fatalf("Failed to log into 3Scale: %v", err)
	}

	waitForUserToBecome3ScaleAdmin(t, ctx, host, threescaleLoginUser)

	// Make sure 3Scale is available
	err = tsClient.Ping()
	if err != nil {
		t.Log("Error during making sure 3Scale is available")
	}

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	// Create a product
	productId, err := tsClient.CreateProduct(fmt.Sprintf("dummy-product-%v", r1.Intn(100000)))
	if err != nil {
		t.Log("Error during create the product")
		t.Skip("Skipping due to known flaky behavior due to, reported in Jira: https://issues.redhat.com/browse/MGDAPI-1806")
		//t.Fatal(err)
	}

	// Delete the product
	err = tsClient.DeleteProduct(productId)
	if err != nil {
		t.Log("Error during deleting the product")
		t.Skip("Skipping due to known flaky behavior due to, reported in Jira: https://issues.redhat.com/browse/MGDAPI-1806")
		//t.Fatal(err)
	}
}

func waitForUserToBecome3ScaleAdmin(t TestingTB, ctx *TestingContext, host, userName string) {
	err := wait.PollImmediate(time.Second*10, time.Minute*5, func() (done bool, err error) {
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
		t.Skip("Skipping due to known flaky behavior due to, reported in Jira: https://issues.redhat.com/browse/MGDAPI-1806")
		//t.Fatalf("timout asserting 3scale user, %s, is admin for performing test: %s", userName, err)
	}
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
	defer resp.Body.Close()

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
