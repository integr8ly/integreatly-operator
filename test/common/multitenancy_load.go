package common

import (
	goctx "context"
	"fmt"
	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
	testResources "github.com/integr8ly/integreatly-operator/test/resources"
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	"os"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
	"time"
)

const (
	pollingTime = time.Second * 5
	realmName   = "rhd"
)

var multitenantUsers int

func TestMultitenancyLoad(t TestingTB, ctx *TestingContext) {
	var testUser string
	var timeOfLogin time.Time

	// get tenant creation time limit
	tenantCreationTime, err := getTenantCreationTime(t)
	if err != nil {
		t.Fatalf("error while getting TENANTS_CREATION_TIMEOUT: %v", err)
	}
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

	// Either setup IDP and create users if in PROW, or just create users if in pipeline
	err = setupIDP(t, ctx, rhmi)
	if err != nil {
		t.Fatal("error while setting up IDP or users %s", err)
	}

	// Creation of tenants
	postfix := 1
	for postfix <= multitenantUsers {
		isClusterLoggedIn := false
		routeFound := false
		isThreeScaleLoggedIn := false
		var testTimeout time.Duration

		// First user needs more time due to the time IDP takes to be created and available
		if postfix == 1 {
			testTimeout = time.Minute * 10
		} else {
			testTimeout = tenantCreationTime
		}

		// Create client for current tenant
		tenantClient, err := createTenantClient(t, ctx)
		if err != nil {
			t.Fatalf("error while creating client for tenant: %v", err)
		}

		// Build username string eg test-user01, test-user02, test-user100
		testUser = fmt.Sprintf("%v%02v", DefaultTestUserName, postfix)

		err = wait.Poll(pollingTime, testTimeout, func() (done bool, err error) {

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
			if !routeFound && isClusterLoggedIn {
				err = getTenant3scaleRoute(t, ctx, testUser)
				if err != nil {
					return false, nil
				} else {
					routeFound = true
				}
			}

			//Login tenant to 3scale
			if !isThreeScaleLoggedIn && routeFound && isClusterLoggedIn {
				err = loginTenantToThreeScale(t, ctx, testUser, rhmi, tenantClient)
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
			t.Errorf("User %s login and creation of tenant failed with: %v", testUser, err)
		}
		postfix++
	}
}

func loginToCluster(t TestingTB, tenantClient *http.Client, masterURL, testUser string) error {
	if err := resources.DoAuthOpenshiftUser(fmt.Sprintf("%s/auth/login", masterURL), testUser, DefaultPassword, tenantClient, realmName, t); err != nil {
		return err
	}
	t.Log(fmt.Sprintf("%s has logged in successfully", testUser))
	return nil
}

func loginTenantToThreeScale(t TestingTB, ctx *TestingContext, testUser string, rhmi *rhmiv1alpha1.RHMI, tenantClient *http.Client) error {
	host := fmt.Sprintf("https://%v-admin.%v", testUser, rhmi.Spec.RoutingSubdomain)
	err := loginToThreeScale(t, host, threescaleLoginUser, DefaultPassword, realmName, tenantClient)
	if err != nil {
		return err
	}
	return nil
}

func getRegisteredTenantsNumber(t TestingTB) (int, error) {
	var multitenantUsersEnvar string
	var ok bool
	multitenantUsersEnvar, ok = os.LookupEnv("NUMBER_OF_TENANTS")
	if ok != true {
		t.Log("NUMBER_OF_TENANTS envvar not found, setting to default 10")
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
		t.Log("TENANTS_CREATION_TIMEOUT not found, setting to default value of 3 minutes")
		tenantCreationTime, err = time.ParseDuration("10m")
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
		generatedName := route.Spec.Host

		if strings.Contains(generatedName, testUser) {
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
		// settign TestingIDPRealm to realmName required by Multitenant
		TestingIDPRealm = realmName
		err := createTestingIDP(t, goctx.TODO(), ctx.Client, ctx.KubeConfig, true)
		if err != nil {
			return fmt.Errorf("error while creating rhd testing IDP: %s", err)
		}
	}

	return nil
}
