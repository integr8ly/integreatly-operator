package e2e

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/common"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var installType string
var err error
var retryInterval = time.Second * 30
var bootStrapStageTimeout = time.Minute * 5
var cloudResourcesStageTimeout = time.Minute * 10
var monitoringStageTimeout = time.Minute * 10
var authenticationStageTimeout = time.Minute * 30
var productsStageTimout = time.Minute * 30
var solutionExplorerStageTimeout = time.Minute * 10
var deploymentTimeout = time.Minute * 25
var failed = false
var artifactsDirEnv = "ARTIFACT_DIR"

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	// start test env
	By("bootstrapping test environment")

	useCluster := true
	testEnv = &envtest.Environment{
		UseExistingCluster:       &useCluster,
		AttachControlPlaneOutput: true,
	}

	cfg, err = testEnv.Start()
	if err != nil {
		t.Fatalf("could not get start test environment %s", err)
	}

	_, found := os.LookupEnv("INSTALLATION_PREFIX")
	if !found {
		t.Fatal("INSTALLATION_PREFIX env var is not set")
	}

	// get install type
	installType, err = common.GetInstallType(cfg)
	if err != nil {
		t.Fatalf("could not get install type %s", err)
	}

	RunSpecsWithDefaultAndCustomReporters(t,
		"E2E Test Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	err = rhmiv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	ctx, err := common.NewTestingContext(cfg)

	// wait for operator deployment to deploy
	err = waitForProductDeployment(ctx.KubeClient, "", "rhmi-operator")
	Expect(err).NotTo(HaveOccurred())

	// wait for cloud resource to deploy
	err = waitForProductDeployment(ctx.KubeClient, string(rhmiv1alpha1.ProductCloudResources), "cloud-resource-operator")
	Expect(err).NotTo(HaveOccurred())

	// wait for cloud resource phase to complete (10 minutes timeout)
	err = waitForInstallationStageCompletion(ctx.Client, retryInterval, cloudResourcesStageTimeout, string(rhmiv1alpha1.CloudResourcesStage))
	Expect(err).NotTo(HaveOccurred())

	// wait for bootstrap phase to complete (5 minutes timeout)
	err = waitForInstallationStageCompletion(ctx.Client, retryInterval, bootStrapStageTimeout, string(rhmiv1alpha1.BootstrapStage))
	Expect(err).NotTo(HaveOccurred())

	// wait for middleware-monitoring to deploy
	err = waitForProductDeployment(ctx.KubeClient, string(rhmiv1alpha1.ProductMonitoring), "application-monitoring-operator")
	Expect(err).NotTo(HaveOccurred())

	// wait for middleware-monitoring to deploy
	err = waitForInstallationStageCompletion(ctx.Client, retryInterval, monitoringStageTimeout, string(rhmiv1alpha1.MonitoringStage))
	Expect(err).NotTo(HaveOccurred())

	// wait for keycloak-operator to deploy
	err = waitForProductDeployment(ctx.KubeClient, string(rhmiv1alpha1.ProductRHSSO), "keycloak-operator")
	Expect(err).NotTo(HaveOccurred())

	// wait for authentication phase to complete (10 minutes timeout)
	err = waitForInstallationStageCompletion(ctx.Client, retryInterval, authenticationStageTimeout, string(rhmiv1alpha1.AuthenticationStage))
	Expect(err).NotTo(HaveOccurred())

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

	if rhmiv1alpha1.IsRHOAM(rhmiv1alpha1.InstallationType(installType)) {
		products = map[string]string{
			"3scale":   "3scale-operator",
			"user-sso": "keycloak-operator",
		}
	}

	for product, deploymentName := range products {
		err = waitForProductDeployment(ctx.KubeClient, product, deploymentName)
		Expect(err).NotTo(HaveOccurred())
	}

	// wait for products phase to complete (30 minutes timeout)
	err = waitForInstallationStageCompletion(ctx.Client, retryInterval, productsStageTimout, string(rhmiv1alpha1.ProductsStage))
	Expect(err).NotTo(HaveOccurred())

	// wait for solution-explorer operator to deploy
	if installType == string(rhmiv1alpha1.InstallationTypeManaged) {
		err = waitForInstallationStageCompletion(ctx.Client, retryInterval, solutionExplorerStageTimeout, string(rhmiv1alpha1.SolutionExplorerStage))
		Expect(err).NotTo(HaveOccurred())
	}
	// +kubebuilder:scaffold:scheme

	close(done)
}, 5400)

var _ = AfterEach(func() {
	failed = failed || CurrentGinkgoTestDescription().Failed
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")

	ctx, err := common.NewTestingContext(cfg)

	artifactsDir := os.Getenv(artifactsDirEnv)
	if failed && artifactsDir != "" {
		if _, err := os.Stat(artifactsDir); !os.IsNotExist(err) {
			out := path.Join(artifactsDir, "rhmi.yaml")
			By(fmt.Sprintf("Writing rhmi.yaml file to %s", out))
			err = common.WriteRHMICRToFile(ctx.Client, out)
			Expect(err).NotTo(HaveOccurred())
		}
	}

	err = testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func waitForInstallationStageCompletion(k8sClient client.Client, retryInterval, timeout time.Duration, phase string) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		logrus.Info("Checking installation stage completion")

		installation, err := common.GetRHMI(k8sClient, false)
		if installation == nil {
			return false, fmt.Errorf("Waiting for availability of rhmi installation %s", err)
		}

		phaseStatus := fmt.Sprintf("%#v", installation.Status.Stages[rhmiv1alpha1.StageName(phase)].Phase)
		if strings.Contains(phaseStatus, "completed") {
			return true, nil
		}

		if installation.Status.LastError != "" {
			logrus.Infof("Last Error: %s\n", installation.Status.LastError)
		}
		return false, err
	})
	if err != nil {
		return err
	}
	return nil
}

func waitForProductDeployment(kubeclient kubernetes.Interface, product, deploymentName string) error {
	namespace := common.NamespacePrefix + product + "-operator"
	if deploymentName == "enmasse-operator" {
		namespace = common.NamespacePrefix + product
	}
	if product == "" {
		namespace = common.NamespacePrefix + "operator"
	}

	logrus.Infof("namespace %s", namespace)
	err := wait.Poll(retryInterval, deploymentTimeout, func() (done bool, err error) {
		deployment, err := kubeclient.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
		if err != nil {
			if k8serr.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		if int(deployment.Status.AvailableReplicas) >= 1 {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return err
	}
	return nil
}
