package common

import (
	goctx "context"
	"fmt"
	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources/user"
	"github.com/integr8ly/integreatly-operator/test/resources"
	testResources "github.com/integr8ly/integreatly-operator/test/resources"
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	"os"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	pollingTime = time.Second * 5
)

var (
	testUser               string
	multitenantUsers       int
	err                    error
	waitgroup              sync.WaitGroup
	clusterLoginSuccessful = true
	realmName, _           = user.GetIdpName()
)

func TestMultitenancyLoad(t TestingTB, ctx *TestingContext) {
	// get amount of tenants to be created
	multitenantUsers, err = getRegisteredTenantsNumber(t)
	if err != nil {
		t.Fatalf("error while getting NUMBER_OF_TENANTS")
	}

	// Get master URL
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}
	masterURL := rhmi.Spec.MasterURL

	//// Either setup IDP and create users if in PROW, or create users if in pipeline
	//err = setupIDP(t, ctx, rhmi)
	//if err != nil {
	//	t.Fatal("error while setting up IDP or users %s", err)
	//}

	// Skip mass login to cluster if running in PROW, instead login as 1st and 2nd user only.
	if testResources.RunningInProw(rhmi) {
		t.Logf("Mass login")
		// Cluster login all users apart from last one
		clusterLoginSuccess := loginUsersToCluster(t, ctx, masterURL)
		if !clusterLoginSuccess {
			t.Errorf("Loggin in of users has failed")
		}
	}

	// Verify that last user created can login to 3scale
	err = loginTo3scaleAsCertainUser(t, ctx, masterURL, rhmi.Spec.RoutingSubdomain, multitenantUsers-1)
	if err != nil {
		t.Errorf("User login to 3scale failed: %v", err)
	}

	// Meassure the time it takes for a newly logged in tenant to be created + logged in to 3scale
	err = loginTo3scaleAsCertainUser(t, ctx, masterURL, rhmi.Spec.RoutingSubdomain, multitenantUsers)
	if err != nil {
		t.Errorf("User login to 3scale failed: %v", err)
	}
}

func loginTo3scaleAsCertainUser(t TestingTB, ctx *TestingContext, masterURL, routingDomain string, userID int) error {
	var timeOfLogin time.Time
	var tenantCreationTime time.Duration
	isClusterLoggedIn := false
	routeFound := false
	isThreeScaleLoggedIn := false

	tenantCreationTime, err = getTenantCreationTime(t)
	if err != nil {
		return fmt.Errorf("error while setting tenantCreationTime timeout: %v", err)
	}

	// Create client for current tenant
	tenantClient, err := createTenantClient(t, ctx)
	if err != nil {
		return fmt.Errorf("error while creating client for tenant: %v", err)
	}

	// Build username for final user
	testUser := fmt.Sprintf("%v%02v", DefaultTestUserName, userID)

	// 3scale route for tenant
	host := fmt.Sprintf("https://%v-admin.%v", testUser, routingDomain)

	err = wait.Poll(pollingTime, tenantCreationTime, func() (done bool, err error) {

		// Let user login to the cluster
		if !isClusterLoggedIn {
			err = loginToCluster(t, tenantClient, masterURL, testUser)
			if err != nil {
				return false, nil
			} else {
				timeOfLogin = time.Now()
				isClusterLoggedIn = true
			}
		}

		// Check if 3scale route is available
		if !routeFound {
			err = getTenant3scaleRoute(t, ctx, testUser)
			if err != nil {
				return false, nil
			} else {
				routeFound = true
			}
		}

		//Login tenant to 3scale
		if !isThreeScaleLoggedIn {
			err := loginToThreeScale(t, host, threescaleLoginUser, DefaultPassword, realmName, tenantClient)
			if err != nil {
				t.Log(fmt.Sprintf("User failed to login to 3scale %s", testUser))
				return false, nil
			} else {
				timeSinceLoginToRoutesFound := time.Since(timeOfLogin).Seconds()
				t.Log(fmt.Sprintf("User logged in to 3scale %s after %v", testUser, timeSinceLoginToRoutesFound))
				isThreeScaleLoggedIn = true
			}
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("User %s login and creation of tenant failed with: %v", testUser, err)
	} else {
		return nil
	}
}

func loginUsersToCluster(t TestingTB, ctx *TestingContext, masterURL string) bool {
	postfix := 1
	for postfix < multitenantUsers {
		waitgroup.Add(1)
		testUser := fmt.Sprintf("%v%02v", DefaultTestUserName, postfix)

		// Add sleep in between starting new go routines to avoid spamming cluster sso instantly
		time.Sleep(100 * time.Millisecond)

		go func() {
			var tenantClient *http.Client
			isClusterLoggedIn := false
			isClientReady := false
			timeoutForInitialUsersClusterLogin := time.Minute * 10

			err := wait.Poll(pollingTime, timeoutForInitialUsersClusterLogin, func() (done bool, err error) {
				if !isClientReady {
					tenantClient, err = createTenantClient(t, ctx)
					if err != nil {
						return false, nil
					} else {
						isClientReady = true
					}
				}
				if !isClusterLoggedIn {
					err = loginToCluster(t, tenantClient, masterURL, testUser)
					if err != nil {
						return false, nil
					} else {
						isClusterLoggedIn = true
					}
				}
				return true, nil
			})

			if err != nil {
				// Log the error but continue
				t.Logf("User %s login to the cluster failed: %v", testUser, err)
				clusterLoginSuccessful = false
			}
			waitgroup.Done()
		}()
		postfix++
	}
	waitgroup.Wait()
	return clusterLoginSuccessful
}

func loginToCluster(t TestingTB, tenantClient *http.Client, masterURL, testUser string) error {
	if err := resources.DoAuthOpenshiftUser(fmt.Sprintf("%s/auth/login", masterURL), testUser, DefaultPassword, tenantClient, realmName, t); err != nil {
		return err
	}
	t.Log(fmt.Sprintf("%s has logged in successfully", testUser))
	return nil
}

func getRegisteredTenantsNumber(t TestingTB) (int, error) {
	var multitenantUsersEnvar string
	var ok bool
	multitenantUsersEnvar, ok = os.LookupEnv("NUMBER_OF_TENANTS")
	if ok != true {
		t.Log("NUMBER_OF_TENANTS envvar not found, setting to default 2")
		multitenantUsersEnvar = "2"
	}
	multitenantUsers, err := strconv.ParseInt(multitenantUsersEnvar, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error while converting NUMBER_OF_TENANTS to int64")
	}
	return int(multitenantUsers), nil
}

func getTenantCreationTime(t TestingTB) (time.Duration, error) {
	var tenantCreationTime time.Duration
	var err error
	duration, ok := os.LookupEnv("TENANTS_CREATION_TIMEOUT")
	if ok != true {
		t.Log("TENANTS_CREATION_TIMEOUT not found, setting to default value of 20 minutes")
		tenantCreationTime, err = time.ParseDuration("20m")
	} else {
		tenantCreationTime, err = time.ParseDuration(fmt.Sprintf("%sm", duration))
	}
	if err != nil {
		return 0, err
	}
	return tenantCreationTime, nil
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
		return fmt.Errorf("Route for %s has not yet been found", testUser)
	} else {
		return nil
	}
}

func createTenantClient(t TestingTB, ctx *TestingContext) (*http.Client, error) {
	tenantClient, err := NewTestingHTTPClient(ctx.KubeConfig)
	if err != nil {
		return nil, err
	}

	return tenantClient, nil
}

func setupIDP(t TestingTB, ctx *TestingContext, rhmi *rhmiv1alpha1.RHMI) error {

	// If not running in PROW we need to skip IDP creation as it will be added manually
	if !testResources.RunningInProw(rhmi) {
		if !hasIDPCreated(goctx.TODO(), ctx.Client, t, realmName) {
			return fmt.Errorf("IDP is not present on the cluster")
		}

		err := createKeycloakUsers(goctx.TODO(), ctx.Client, rhmi, multitenantUsers, realmName)
		if err != nil {
			return fmt.Errorf("error while creating keycloak users: %s", err)
		}
	} else {
		// setting TestingIDPRealm to realmName required by Multitenant
		TestingIDPRealm = realmName
		err := createTestingIDP(t, goctx.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts)
		if err != nil {
			return fmt.Errorf("error while creating DevSandbox testing IDP: %s", err)
		}
	}

	return nil
}
