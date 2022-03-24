package common

import (
	"context"
	goctx "context"
	"fmt"
	"github.com/headzoo/surf"
	brow "github.com/headzoo/surf/browser"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
	routev1 "github.com/openshift/api/route/v1"
	usersv1 "github.com/openshift/api/user/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	"net/url"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"

	"os"
	"strings"
	"time"
)

const (
	pollingTime        = time.Second * 5
	tenantReadyTimeout = time.Minute * 10
	userReadyTimeout   = time.Minute * 5
)

var (
	testUserForDeletion = fmt.Sprintf("%v%02v", DefaultTestUserName, 1)
)

func TestMultitenancy(t TestingTB, ctx *TestingContext) {
	// Get master URL
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	// Testing IDP with 2 regular users and 2 admins gets created
	err = createTestingIDP(t, goctx.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts)
	if err != nil {
		t.Errorf("error while creating testing IDP: %s", err)
	}

	// Verify that the regular users can log in to their 3scale account
	err = loginUsersTo3scale(t, ctx, rhmi)
	if err != nil {
		t.Errorf("User login to 3scale failed: %v", err)
	}

	// Delete user CR for one of the users
	err = deleteTenantUserCR(t, ctx)
	if err != nil {
		t.Error(err)
	}

	// Confirm that the 3scale is no longer available for a tenant
	err = confirmTenantAccountNotAvailable(t, ctx, rhmi.Spec.RoutingSubdomain)
	if err != nil {
		t.Error(err)
	}
}

func loginUsersTo3scale(t TestingTB, ctx *TestingContext, rhmi *integreatlyv1alpha1.RHMI) error {
	postfix := 1
	numberOfTestUsers := getNumberOfTestUsersFromEnv()
	for postfix <= numberOfTestUsers {
		isClusterLoggedIn := false
		isTenantCRCreated := false
		routeFound := false
		isThreeScaleLoggedIn := false

		// Create client for current tenant
		tenantClient, err := NewTestingHTTPClient(ctx.KubeConfig)
		if err != nil {
			return fmt.Errorf("error while creating client for tenant: %v", err)
		}

		testUser := fmt.Sprintf("%v%02v", DefaultTestUserName, postfix)

		host := fmt.Sprintf("https://%v-admin.%v", testUser, rhmi.Spec.RoutingSubdomain)

		// Create new namespace for APIManagementTenant CR
		err, testUserNamespace := createTestingUserNamespace(t, testUser, ctx)
		if err != nil {
			return fmt.Errorf("error while creating namespace for testing user: %v", err)
		}

		err = wait.Poll(pollingTime, tenantReadyTimeout, func() (done bool, err error) {
			// login to cluster
			if !isClusterLoggedIn {
				err = loginToCluster(t, tenantClient, rhmi.Spec.MasterURL, testUser)
				if err != nil {
					return false, nil
				} else {
					isClusterLoggedIn = true
				}
			}

			if !isTenantCRCreated {
				// Create testing user CR an APIManagementTenant CR
				err = createTestingUserApiManagementTenantCR(t, testUser, testUserNamespace, ctx)
				if err != nil {
					return false, nil
				} else {
					isTenantCRCreated = true
				}
			}

			// check if 3scale route is available
			if !routeFound {
				err = getTenant3scaleRoute(t, ctx, testUser)
				if err != nil {
					return false, nil
				} else {
					routeFound = true
				}
			}

			// login tenant to 3scale
			if !isThreeScaleLoggedIn {
				err := loginToThreeScale(t, host, testUser, DefaultPassword, TestingIDPRealm, tenantClient)
				if err != nil {
					t.Log(fmt.Sprintf("User failed to login to 3scale %s", testUser))
					return false, nil
				} else {
					isThreeScaleLoggedIn = true
				}
			}

			return true, nil
		})

		if err != nil {
			return fmt.Errorf("user %s login and creation of tenant failed with: %v", testUser, err)
		}

		postfix++
	}

	return nil
}

func getTenant3scaleRoute(t TestingTB, ctx *TestingContext, testUser string) error {
	routeFound := false
	routes := &routev1.RouteList{}

	err := ctx.Client.List(goctx.TODO(), routes, &k8sclient.ListOptions{
		Namespace: ThreeScaleProductNamespace,
	})

	if err != nil {
		return err
	}

	for _, route := range routes.Items {
		if route.Spec.To.Kind != "Service" {
			continue
		}
		tenantHostRoute := route.Spec.Host
		if strings.Contains(tenantHostRoute, testUser) {
			routeFound = true
		}
	}

	if !routeFound {
		return fmt.Errorf("route for %s has not yet been found", testUser)
	} else {
		return nil
	}
}

func deleteTenantUserCR(t TestingTB, ctx *TestingContext) error {
	usersList := &usersv1.UserList{}
	err := ctx.Client.List(context.TODO(), usersList)
	if err != nil {
		return fmt.Errorf("failed at finding users list, error: %v", err)
	}

	for _, user := range usersList.Items {
		if user.Name == testUserForDeletion {
			err := ctx.Client.Delete(context.TODO(), &user)
			if err != nil {
				return fmt.Errorf("failed to remove user CR, error: %v", err)
			}
		}
	}

	return nil
}

func confirmTenantAccountNotAvailable(t TestingTB, ctx *TestingContext, routingDomain string) error {
	tenantClient, err := NewTestingHTTPClient(ctx.KubeConfig)
	if err != nil {
		return fmt.Errorf("error while creating client for tenant: %v", err)
	}

	host := fmt.Sprintf("https://%v-admin.%v", testUserForDeletion, routingDomain)

	err = wait.Poll(pollingTime, tenantReadyTimeout, func() (done bool, err error) {
		err = is3scaleLoginFailed(t, host, tenantClient)
		if err != nil {
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		return fmt.Errorf("3scale did not return 404 after tenant deletion: %v", err)
	}

	return nil
}

func loginToCluster(t TestingTB, tenantClient *http.Client, masterURL, testUser string) error {
	if err := resources.DoAuthOpenshiftUser(fmt.Sprintf("%s/auth/login", masterURL), testUser, DefaultPassword, tenantClient, TestingIDPRealm, t); err != nil {
		return err
	}
	return nil
}

func is3scaleLoginFailed(t TestingTB, tsHost string, client *http.Client) error {
	parsedURL, err := url.Parse(tsHost)
	if err != nil {
		return fmt.Errorf("failed to parse three scale url %s: %s", parsedURL, err)
	}

	if parsedURL.Scheme == "" {
		tsHost = fmt.Sprintf("https://%s", tsHost)
	}

	tsLoginURL := fmt.Sprintf("%v/p/login", tsHost)
	browser := surf.NewBrowser()
	browser.SetCookieJar(client.Jar)
	browser.SetTransport(client.Transport)
	browser.SetAttribute(brow.FollowRedirects, true)

	_ = browser.Open(tsLoginURL)
	statusCode := browser.StatusCode()

	if statusCode != 404 {
		return fmt.Errorf("unsuccessful 3scale login failed, response code recived is %v", statusCode)
	}

	return nil
}

func createTestingUserNamespace(t TestingTB, user string, ctx *TestingContext) (error, string) {
	testUserNamespaceName := user + "-dev"
	testUserNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testUserNamespaceName,
		},
	}

	_, err := controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, testUserNamespace, func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("error while creating namespace for testing user %v", err)
	}

	return nil, testUserNamespaceName
}

func createTestingUserApiManagementTenantCR(t TestingTB, testUserName string, testUserNamespace string, ctx *TestingContext) error {
	// Wait until User has finished being created before attempting to create an APIManagementTenant CR
	err := wait.Poll(pollingTime, userReadyTimeout, func() (done bool, err error) {
		// Get User successfully to exit the poll
		user := &usersv1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: testUserName,
			},
		}
		key, err := k8sclient.ObjectKeyFromObject(user)
		if err != nil {
			return false, nil
		}
		err = ctx.Client.Get(context.TODO(), key, user)
		if err != nil {
			return false, nil
		}
		return true, nil
	})

	tenantCR := &integreatlyv1alpha1.APIManagementTenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testUserName,
			Namespace: testUserNamespace,
		},
	}
	_, err = controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, tenantCR, func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("error while creating APIManagementTenant CR for testing user %v", err)
	}
	return nil
}

func getNumberOfTestUsersFromEnv() int {
	strNum := os.Getenv("TENANTS_NUMBER")
	if strNum == "" {
		return defaultNumberOfTestUsers
	}
	num, err := strconv.Atoi(strNum)
	if err != nil {
		fmt.Println("error converting env var TENANTS_NUMBER to integer, using default number of test users")
		return defaultNumberOfTestUsers
	}
	return num
}
