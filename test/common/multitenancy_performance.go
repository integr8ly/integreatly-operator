package common

import (
	"context"
	"fmt"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
)

const (
	numberOfPerformanceTestUsers = 200
)

// M02 test
// The test is required IDP to be already created with number of users as defined in const: numberOfPerformanceTestUsers
// Test could be extended to add Products creation and Promotion as in script:
// test/scripts/performance/m02_check_tenants_creation_performance.bash

// To allow users login and creation: export PERF_TEST_USERS_LOGIN=true

func TestMultitenancyPerformance(t TestingTB, ctx *TestingContext) {
	numberOfTestUsers := getNumberOfPerformanceTestUsersFromEnv()
	// Get master URL
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	// Login Users to Cluster to create OC users for test
	if os.Getenv("PERF_TEST_USERS_LOGIN") == "true" {
		err = loginUsersToCluster(t, ctx, rhmi, numberOfTestUsers)
		if err != nil {
			t.Errorf("User login to Cluster failed: %v", err)
		}
	}

	err = createNamespaces(t, ctx, numberOfTestUsers)
	if err != nil {
		t.Errorf("error while createNamespaces for test users: %v", err)
	}

	err = createApiManagementTenantCRs(t, ctx, numberOfTestUsers)
	if err != nil {
		t.Errorf("error while createNamespaces for test users: %v", err)
	}
}

func loginUsersToCluster(t TestingTB, ctx *TestingContext, rhmi *integreatlyv1alpha1.RHMI, numberOfTestUsers int) error {
	postfix := 1
	for postfix <= numberOfTestUsers {
		isClusterLoggedIn := false
		// Create client for current tenant
		tenantClient, err := NewTestingHTTPClient(ctx.KubeConfig)
		if err != nil {
			return fmt.Errorf("error while creating client for tenant: %v", err)
		}
		testUser := fmt.Sprintf("%v%02v", DefaultTestUserName, postfix)
		err = wait.PollUntilContextTimeout(context.TODO(), pollingTime, tenantReadyTimeout, false, func(ctx2 context.Context) (done bool, err error) {
			// login to cluster
			if !isClusterLoggedIn {
				err = resources.DoAuthOpenshiftUser(fmt.Sprintf("%s/auth/login", rhmi.Spec.MasterURL),
					testUser, TestingIdpPassword, tenantClient, TestingIDPRealm, t)
				if err != nil {
					return false, nil
				} else {
					isClusterLoggedIn = true
				}
			}
			return true, nil
		})
		if err != nil {
			return fmt.Errorf("user %s login failed with: %v", testUser, err)
		}
		postfix++
	}
	return nil
}

func getNumberOfPerformanceTestUsersFromEnv() int {
	strNum := os.Getenv("PERF_TEST_TENANTS_NUMBER")
	if strNum == "" {
		return numberOfPerformanceTestUsers
	}
	num, err := strconv.Atoi(strNum)
	if err != nil {
		fmt.Println("error converting env var TENANTS_NUMBER to integer, using default number of test users")
		return numberOfPerformanceTestUsers
	}
	return num
}

func createNamespaces(t TestingTB, ctx *TestingContext, numberOfTestUsers int) error {
	postfix := 1
	for postfix <= numberOfTestUsers {
		testUser := fmt.Sprintf("%v%02v", DefaultTestUserName, postfix)
		testUserNamespaceName := testUser + "-dev"
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
		postfix++
	}
	return nil
}

func createApiManagementTenantCRs(t TestingTB, ctx *TestingContext, numberOfTestUsers int) error {
	postfix := 1
	for postfix <= numberOfTestUsers {
		testUser := fmt.Sprintf("%v%02v", DefaultTestUserName, postfix)
		testUserNamespace := testUser + "-dev"
		tenantCR := &integreatlyv1alpha1.APIManagementTenant{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "example",
				Namespace: testUserNamespace,
			},
		}
		_, err := controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, tenantCR, func() error {
			return nil
		})
		if err != nil {
			t.Fatalf("error while creating APIManagementTenant CR for testing user %v", err)
		}
		postfix++
	}
	return nil
}
