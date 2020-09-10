package e2e

import (
	"context"
	goctx "context"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/integr8ly/integreatly-operator/test/common"

	"github.com/integr8ly/integreatly-operator/pkg/apis"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	retryInterval                = time.Second * 5
	timeout                      = time.Second * 75
	deploymentRetryInterval      = time.Second * 30
	deploymentTimeout            = time.Minute * 25
	cleanupRetryInterval         = time.Second * 1
	cleanupTimeout               = time.Second * 5
	namespaceLabel               = "integreatly"
	bootStrapStageTimeout        = time.Minute * 5
	cloudResourcesStageTimeout   = time.Minute * 10
	monitoringStageTimeout       = time.Minute * 10
	authenticationStageTimeout   = time.Minute * 30
	productsStageTimout          = time.Minute * 30
	solutionExplorerStageTimeout = time.Minute * 10
	artifactsDirEnv              = "ARTIFACT_DIR"
)

func TestIntegreatly(t *testing.T) {
	err := framework.AddToFrameworkScheme(apis.AddToScheme, &integreatlyv1alpha1.RHMIList{})
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}
	ctx := framework.NewTestCtx(t)
	err = ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	t.Log("Initialized cluster resources")
	if err != nil {
		t.Fatal(err)
	}
	// get global framework variables
	f := framework.Global

	// run subtests
	t.Run("integreatly", func(t *testing.T) {

		// running ALL_TESTS test cases
		common.RunTestCases(common.ALL_TESTS, t, f.KubeConfig)

		// create the cluster
		var err error
		t.Run("Cluster", func(t *testing.T) {
			err = IntegreatlyCluster(t, f, ctx)
			if err != nil {
				t.Log(err)
				t.Fail()
			}
		})
		if err != nil {
			t.Log("cluster not in a testable state, quiting test run")
			t.FailNow()
		}

		installType, err := common.GetInstallType(f.KubeConfig)
		if err != nil {
			t.Fatalf("failed to get install type, err: %s", installType)
		}

		// get happy path test cases according to the install type
		happyPathTestCases := common.GetHappyPathTestCases(installType)

		// running HAPPY_PATH_TESTS tests cases
		common.RunTestCases(happyPathTestCases, t, f.KubeConfig)

		t.Run("Integreatly Destructive Tests", func(t *testing.T) {
			// Do not execute these tests unless DESTRUCTIVE is set to true
			if os.Getenv("DESTRUCTIVE") != "true" {
				t.Skip("Skipping Destructive tests as DESTRUCTIVE env var is not set to true")
			}

			common.RunTestCases(common.DESTRUCTIVE_TESTS, t, f.KubeConfig)
		})

	})

	artifactsDir := os.Getenv(artifactsDirEnv)
	if t.Failed() && artifactsDir != "" {
		if _, err := os.Stat(artifactsDir); !os.IsNotExist(err) {
			out := path.Join(artifactsDir, "rhmi.yaml")
			t.Logf("Writing rhmi.yaml file to %s", out)
			err = common.WriteRHMICRToFile(f.Client.Client, out)
			if err != nil {
				t.Error("Failed to write RHMI cr due to error", err)
			}
		}
	} else {
		ctx.Cleanup()
	}
}

func waitForProductDeployment(t *testing.T, f *framework.Framework, ctx *framework.TestCtx, product, deploymentName string) error {
	namespace := ""
	if deploymentName != "enmasse-operator" {
		namespace = common.NamespacePrefix + product + "-operator"
	} else {
		namespace = common.NamespacePrefix + product
	}
	t.Logf("Checking %s:%s", namespace, deploymentName)

	start := time.Now()
	err := e2eutil.WaitForDeployment(t, f.KubeClient, namespace, deploymentName, 1, deploymentRetryInterval, deploymentTimeout)
	if err != nil {
		end := time.Now()
		elapsed := end.Sub(start)
		t.Logf("%s:%s down , Timed out after %d :", namespace, deploymentName, elapsed)
		return err
	}

	end := time.Now()
	elapsed := end.Sub(start)

	t.Logf("%s:%s up, waited %d", namespace, deploymentName, elapsed)
	return nil
}

func integreatlyManagedTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {

	installType, err := common.GetInstallType(f.KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to get install type, err: %s", installType)
	}

	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %deploymentName", err)
	}

	// wait for cloud resource phase to complete (10 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, cloudResourcesStageTimeout, string(integreatlyv1alpha1.CloudResourcesStage))
	if err != nil {
		return err
	}

	// wait for cloud resource to deploy
	err = waitForProductDeployment(t, f, ctx, string(integreatlyv1alpha1.ProductCloudResources), "cloud-resource-operator")
	if err != nil {
		return err
	}

	// wait for bootstrap phase to complete (5 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, bootStrapStageTimeout, string(integreatlyv1alpha1.BootstrapStage))
	if err != nil {
		return err
	}

	// wait for middleware-monitoring to deploy
	err = waitForProductDeployment(t, f, ctx, string(integreatlyv1alpha1.ProductMonitoring), "application-monitoring-operator")
	if err != nil {
		return err
	}

	// wait for middleware-monitoring phase to complete (10 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, monitoringStageTimeout, string(integreatlyv1alpha1.MonitoringStage))
	if err != nil {
		return err
	}

	// wait for keycloak-operator to deploy
	err = waitForProductDeployment(t, f, ctx, string(integreatlyv1alpha1.ProductRHSSO), "keycloak-operator")
	if err != nil {
		return err
	}

	// wait for authentication phase to complete (10 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, authenticationStageTimeout, string(integreatlyv1alpha1.AuthenticationStage))
	if err != nil {
		return err
	}

	//Product Stage - verify operators deploy
	products := map[string]string{
		"3scale":               "3scale-operator",
		"amq-online":           "enmasse-operator",
		"codeready-workspaces": "codeready-operator",
		"fuse":                 "syndesis-operator",
		"user-sso":             "keycloak-operator",
		"ups":                  "unifiedpush-operator",
		"apicurito":            "apicurito-operator",
	}

	if installType == string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		products = map[string]string{
			"3scale": "3scale-operator",
		}
	}

	for product, deploymentName := range products {
		err = waitForProductDeployment(t, f, ctx, product, deploymentName)
		if err != nil {
			return err
		}
	}

	// wait for products phase to complete (30 minutes timeout)
	err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, productsStageTimout, string(integreatlyv1alpha1.ProductsStage))
	if err != nil {
		return err
	}

	if installType != string(integreatlyv1alpha1.InstallationTypeManagedApi) {
		// wait for solution-explorer operator to deploy
		err = waitForProductDeployment(t, f, ctx, string(integreatlyv1alpha1.ProductSolutionExplorer), "tutorial-web-app-operator")
		if err != nil {
			return err
		}

		// wait for solution-explorer phase to complete (10 minutes timeout)
		err = waitForInstallationStageCompletion(t, f, namespace, deploymentRetryInterval, solutionExplorerStageTimeout, string(integreatlyv1alpha1.SolutionExplorerStage))
		if err != nil {
			return err
		}
	}

	return nil
}

func waitForInstallationStageCompletion(t *testing.T, f *framework.Framework, namespace string, retryInterval, timeout time.Duration, phase string) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		installation := &integreatlyv1alpha1.RHMI{}
		err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: common.InstallationName, Namespace: namespace}, installation)
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s installation in namespace: %s, phase: %s\n", common.InstallationName, namespace, phase)
				return false, nil
			}
			return false, err
		}

		phaseStatus := fmt.Sprintf("%#v", installation.Status.Stages[integreatlyv1alpha1.StageName(phase)].Phase)
		if strings.Contains(phaseStatus, "completed") {
			return true, nil
		}

		t.Logf("Waiting for completion of %s\n", phase)
		if installation.Status.LastError != "" {
			t.Logf("Last Error: %s\n", installation.Status.LastError)
		}
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("%s phase completed \n", phase)
	return nil
}

func IntegreatlyCluster(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	// Create SMTP Secret
	installationPrefix, found := os.LookupEnv("INSTALLATION_PREFIX")
	if !found {
		err := fmt.Errorf("INSTALLATION_PREFIX env var is not set")
		return err
	}

	var smtpSec = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprint(installationPrefix, "-smtp"),
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"host":     []byte("test"),
			"password": []byte("test"),
			"port":     []byte("test"),
			"tls":      []byte("test"),
			"username": []byte("test"),
		},
	}
	err = f.Client.Create(context.TODO(), smtpSec, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create SMTP Secret: %w", err)
	}

	// create pagerduty secret
	pagerdutySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprint(installationPrefix, "-pagerduty"),
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"secretKey": []byte("test"),
		},
		Type: corev1.SecretTypeOpaque,
	}
	err = f.Client.Create(context.TODO(), pagerdutySecret, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create pagerduty secret: %w", err)
	}

	// create dead mans snitch secret
	dmsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprint(installationPrefix, "-deadmanssnitch"),
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"url": []byte("test"),
		},
		Type: corev1.SecretTypeOpaque,
	}
	err = f.Client.Create(context.TODO(), dmsSecret, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create dead mans snitch secret : %w", err)
	}

	// wait for integreatly-operator to be ready
	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "rhmi-operator", 1, retryInterval, timeout)
	if err != nil {
		return fmt.Errorf("wait for integreatly-operator to be ready: %w", err)

	}
	//TODO: split them into their own test cases
	// check that all of the operators deploy and all of the installation phases complete
	if err = integreatlyManagedTest(t, f, ctx); err != nil {
		return fmt.Errorf("error in fuction integreatlyManagedTest: %w", err)
	}

	return nil
}
