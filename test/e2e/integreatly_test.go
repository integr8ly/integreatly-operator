package e2e

import (
	goctx "context"
	"fmt"

	//"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"time"

	"github.com/integr8ly/integreatly-operator/pkg/apis"
	operator "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	retryInterval                    = time.Second * 5
	timeout                          = time.Second * 60
	deploymentRetryInterval          = time.Second * 30
	deploymentTimeout                = time.Minute * 20
	cleanupRetryInterval             = time.Second * 1
	cleanupTimeout                   = time.Second * 5
	installationCleanupRetryInterval = time.Second * 20
	installationCleanupTimeout       = time.Minute * 4 //Longer timeout required to allow for finalizers to execute
	intlyNamespacePrefix             = "intly-"
)

func TestIntegreatly(t *testing.T) {

	logf.SetLogger(logf.ZapLogger(true))

	installationList := &operator.InstallationList{}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, installationList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}
	// run subtests
	t.Run("integreatly", func(t *testing.T) {
		t.Run("Cluster", IntegreatlyCluster)
	})

}

func waitForProductDeployment(t *testing.T, f *framework.Framework, ctx *framework.TestCtx, product, deploymentName string) error {
	namespace := intlyNamespacePrefix + product
	t.Logf("Checking %s:%s", namespace, deploymentName)

	start := time.Now()
	err := e2eutil.WaitForDeployment(t, f.KubeClient, namespace, deploymentName, 1, deploymentRetryInterval, deploymentTimeout)
	if err != nil {
		return err
	}

	end := time.Now()
	elapsed := end.Sub(start)

	t.Logf("%s:%s up, waited %d", namespace, deploymentName, elapsed)
	return nil
}

func integreatlyWorkshopTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %deploymentName", err)
	}

	// create installation custom resource
	workshopInstallation := &operator.Installation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e-workshop-installation",
			Namespace: namespace,
		},
		Spec: operator.InstallationSpec{
			Type:             "workshop",
			NamespacePrefix:  intlyNamespacePrefix,
			RoutingSubdomain: "apps.example.com",
			MasterURL:        "http://console.apps.example.com",
			SelfSignedCerts:  true,
		},
	}
	// use TestCtx's create helper to create the object and add a cleanup function for the new object
	err = f.Client.Create(goctx.TODO(), workshopInstallation, &framework.CleanupOptions{TestContext: ctx, Timeout: installationCleanupTimeout, RetryInterval: installationCleanupRetryInterval})
	if err != nil {
		return err
	}

	//Auth Stage - verify operators deploy
	err = waitForProductDeployment(t, f, ctx, "rhsso", "keycloak-operator")
	if err != nil {
		return err
	}
	//Product Stage - verify operators deploy
	products := map[string]string{
		"3scale":                "3scale-operator",
		"amq-online":            "enmasse-operator",
		"amq-streams":           "amq-streams-cluster-operator",
		"codeready-workspaces":  "codeready-operator",
		"fuse":                  "syndesis-operator",
		"launcher":              "launcher-operator",
		"middleware-monitoring": "application-monitoring-operator",
		"nexus":                 "nexus-operator",
		"user-sso":              "keycloak-operator",
		"ups":                   "unifiedpush-operator",
	}
	for product, deploymentName := range products {
		err = waitForProductDeployment(t, f, ctx, product, deploymentName)
		if err != nil {
			break
		}
	}
	//SolutionExplorer Stage - verify operators deploy
	products = map[string]string{
		"solution-explorer": "tutorial-web-app-operator",
	}
	for product, deploymentName := range products {
		err = waitForProductDeployment(t, f, ctx, product, deploymentName)
		if err != nil {
			break
		}
	}
	//These test only that the operators themselves have came up. Further testing should be done to verify the individual parts of each product come up also (What the operator is creating).

	return err
}

func IntegreatlyCluster(t *testing.T) {
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	t.Log("Initialized cluster resources")
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	// get global framework variables
	f := framework.Global
	// wait for integreatly-operator to be ready
	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "integreatly-operator", 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	if err = integreatlyWorkshopTest(t, f, ctx); err != nil {
		t.Fatal(err)
	}
}
