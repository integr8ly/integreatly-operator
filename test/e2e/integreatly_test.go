package e2e

import (
	goctx "context"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	//"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"time"

	"github.com/integr8ly/integreatly-operator/pkg/apis"
	operator "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
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
	installationName                 = "e2e-managed-installation"
	bootstrapStage                   = "bootstrap"
	monitoringStage                  = "monitoring"
	authenticationStage              = "authentication"
	productsStage                    = "products"
	solutionExplorerStage            = "solution-explorer"
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

func integreatlyManagedTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %deploymentName", err)
	}

	consoleRoute := &routev1.Route{}
	err = f.Client.Get(goctx.TODO(), client.ObjectKey{Name: "console", Namespace: "openshift-console"}, consoleRoute)
	if err != nil {
		return fmt.Errorf("could not get console route: %deploymentName", err)
	}
	masterUrl := consoleRoute.Spec.Host

	clusterIngress := &configv1.Ingress{}
	err = f.Client.Get(goctx.TODO(), client.ObjectKey{Name: "cluster", Namespace: ""}, clusterIngress)
	if err != nil {
		return fmt.Errorf("could not get cluster ingress: %deploymentName", err)
	}
	routingSubdomain := clusterIngress.Spec.Domain

	t.Logf("Creating installation CR with routingSubdomain:%s, masterUrl:%s\n", routingSubdomain, masterUrl)

	// create installation custom resource
	managedInstallation := &operator.Installation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      installationName,
			Namespace: namespace,
		},
		Spec: operator.InstallationSpec{
			Type:             "managed",
			NamespacePrefix:  intlyNamespacePrefix,
			RoutingSubdomain: routingSubdomain,
			MasterURL:        masterUrl,
			SelfSignedCerts:  true,
		},
	}
	// use TestCtx's create helper to create the object and add a cleanup function for the new object
	err = f.Client.Create(goctx.TODO(), managedInstallation, &framework.CleanupOptions{TestContext: ctx, Timeout: installationCleanupTimeout, RetryInterval: installationCleanupRetryInterval})
	if err != nil {
		return err
	}

	// wait for bootstrap phase to complete (5 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, deploymentTimeout, bootstrapStage)
	if err != nil {
		return err
	}

	// wait for middleware-monitoring to deploy
	err = waitForProductDeployment(t, f, ctx, "middleware-monitoring", "application-monitoring-operator")
	if err != nil {
		return err
	}

	// wait for authentication phase to complete (15 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, deploymentTimeout, monitoringStage)
	if err != nil {
		return err
	}

	// wait for keycloak-operator to deploy
	err = waitForProductDeployment(t, f, ctx, "rhsso", "keycloak-operator")
	if err != nil {
		return err
	}

	// wait for authentication phase to complete (15 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, deploymentTimeout, authenticationStage)
	if err != nil {
		return err
	}

	//Product Stage - verify operators deploy
	products := map[string]string{
		"3scale":                  "3scale-operator",
		"amq-online":              "enmasse-operator",
		"codeready-workspaces":    "codeready-operator",
		"fuse":                    "syndesis-operator",
		"launcher":                "launcher-operator",
		"mdc":                     "mobile-developer-console-operator",
		"mobile-security-service": "mobile-security-service-operator",
		"user-sso":                "keycloak-operator",
		"ups":                     "unifiedpush-operator",
	}
	for product, deploymentName := range products {
		err = waitForProductDeployment(t, f, ctx, product, deploymentName)
		if err != nil {
			return err
		}
	}

	// wait for products phase to complete (5 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, deploymentTimeout*2, productsStage)
	if err != nil {
		return err
	}

	// wait for solution-explorer operator to deploy
	err = waitForProductDeployment(t, f, ctx, "solution-explorer", "tutorial-web-app-operator")
	if err != nil {
		return err
	}

	// wait for solution-explorer phase to complete (10 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, deploymentTimeout, solutionExplorerStage)
	if err != nil {
		return err
	}
	return err
}

func waitForInstallationStageCompletion(t *testing.T, f *framework.Framework, namespace string, retryInterval, timeout time.Duration, phase string) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		installation := &operator.Installation{}
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: installationName, Namespace: namespace}, installation)
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s installation\n", installationName)
				return false, nil
			}
			return false, err
		}

		phaseStatus := fmt.Sprintf("%#v", installation.Status.Stages[operator.StageName(phase)])
		if strings.Contains(phaseStatus, "completed") {
			return true, nil
		}

		t.Logf("Waiting for completion of %s\n", phase)
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("%s phase completed \n", phase)
	return nil
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
	// check that all of the operators deploy and all of the installation phases complete
	if err = integreatlyManagedTest(t, f, ctx); err != nil {
		t.Fatal(err)
	}
}
