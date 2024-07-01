package e2e

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	packageOperatorv1alpha1 "package-operator.run/apis/core/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	threescalev1 "github.com/3scale/3scale-operator/apis/capabilities/v1alpha1"
	threescaleBv1 "github.com/3scale/3scale-operator/apis/capabilities/v1beta1"
	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/common"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var testEnv *envtest.Environment
var installType string
var err error
var retryInterval = time.Second * 30
var deploymentTimeout = time.Minute * 25
var installStageTimeout = time.Minute * 40
var failed = false
var artifactsDirEnv = "ARTIFACT_DIR"

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	// start test env
	useCluster := true
	testEnv = &envtest.Environment{
		UseExistingCluster:       &useCluster,
		AttachControlPlaneOutput: true,
	}

	var err error
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

	// Fetch the current config
	suiteConfig, _ := GinkgoConfiguration()
	suiteConfig.Timeout = time.Minute * 90

	RunSpecs(t, "E2E Test Suite", suiteConfig)
}

var _ = BeforeSuite(func() {
	done := make(chan interface{})
	go func() {
		logf.SetLogger(zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter)))
		By("bootstrapping test environment")
		err = rhmiv1alpha1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		err = threescalev1.SchemeBuilder.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		err = threescaleBv1.SchemeBuilder.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		err = operatorsv1alpha1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		err = packageOperatorv1alpha1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		ctx, err := common.NewTestingContext(cfg)
		Expect(err).NotTo(HaveOccurred())

		// wait for operator deployment to deploy
		err = waitForProductDeployment(ctx.KubeClient, "", "rhmi-operator")
		Expect(err).NotTo(HaveOccurred())

		// wait for cloud resource to deploy
		err = waitForProductDeployment(ctx.KubeClient, string(rhmiv1alpha1.ProductCloudResources), "cloud-resource-operator")
		Expect(err).NotTo(HaveOccurred())

		// wait for keycloak-operator to deploy
		err = waitForProductDeployment(ctx.KubeClient, string(rhmiv1alpha1.ProductRHSSO), "rhsso-operator")
		Expect(err).NotTo(HaveOccurred())

		if !rhmiv1alpha1.IsRHOAMMultitenant(rhmiv1alpha1.InstallationType(installType)) {
			// wait for keycloak-operator (user sso) to deploy
			err = waitForProductDeployment(ctx.KubeClient, "user-sso", "rhsso-operator")
			Expect(err).NotTo(HaveOccurred())
		}

		// wait for 3scale to deploy
		err = waitForProductDeployment(ctx.KubeClient, string(rhmiv1alpha1.Product3Scale), "threescale-operator-controller-manager-v2")
		Expect(err).NotTo(HaveOccurred())

		// wait for installation phase to complete (40 minutes timeout)
		err = waitForInstallationStageCompletion(ctx.Client, retryInterval, installStageTimeout, string(rhmiv1alpha1.InstallStage))
		Expect(err).NotTo(HaveOccurred())
		// +kubebuilder:scaffold:scheme

		close(done)
	}()
	Eventually(done, 5400).Should(BeClosed())
})

var _ = AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")

	ctx, err := common.NewTestingContext(cfg)
	Expect(err).NotTo(HaveOccurred())

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
	err := wait.PollUntilContextTimeout(context.TODO(), retryInterval, timeout, false, func(ctx context.Context) (done bool, err error) {
		logrus.Info("Checking installation stage completion")

		installation, err := common.GetRHMI(k8sClient, false)
		if installation == nil {
			return false, fmt.Errorf("waiting for availability of rhmi installation %s", err)
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
	if product == "" {
		namespace = common.NamespacePrefix + "operator"
	}

	logrus.Infof("namespace %s", namespace)
	err := wait.PollUntilContextTimeout(context.TODO(), retryInterval, deploymentTimeout, false, func(ctx context.Context) (done bool, err error) {
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
