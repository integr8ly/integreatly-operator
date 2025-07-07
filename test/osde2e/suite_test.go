package osde2e

import (
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/common"
)

const (
	testResultsDirectory = "/test-run-results"
	jUnitOutputFilename  = "junit-integreatly-operator.xml"
	addonMetadataName    = "addon-metadata.json"
	testOutputFileName   = "test-output.txt"
	testSuiteName        = "integreatly-operator"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var testEnv *envtest.Environment
var installType string
var err error

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
	//TODO: Trigger operator install

	// get install type
	setVars("redhat-rhoam-operator", "redhat-rhoam-", t)
	installType, err = common.GetInstallType(cfg)
	if err != nil {
		t.Fatalf("could not get install type %s", err)
	}

	jUnitReportLocation := fmt.Sprintf("%s/%s", testResultsDirectory, jUnitOutputFilename)

	// Fetch the current config
	suiteConfig, reporterConfig := GinkgoConfiguration()
	suiteConfig.Timeout = time.Minute * 90

	// Update the JUnitReport
	reporterConfig.JUnitReport = jUnitReportLocation
	// Pass the updated config to RunSpecs()
	RunSpecs(t, "Functional Test Suite", suiteConfig, reporterConfig)

}

var _ = BeforeSuite(func() {
	done := make(chan interface{})
	go func() {
		logf.SetLogger(zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter)))
		By("bootstrapping test environment")
		err = rhmiv1alpha1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		// +kubebuilder:scaffold:scheme

		close(done)
	}()
	Eventually(done, 120).Should(BeClosed())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")

	//TODO: remove operator

	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func setVars(possibleWN, possibleNS string, t *testing.T) {
	err := os.Setenv("WATCH_NAMESPACE", possibleWN)
	if err != nil {
		t.Fatalf("Failed to set WATCH_NAMESPACE env var %s", err)
	}
	err = os.Setenv("SKIP_FLAKES", "true")
	if err != nil {
		t.Logf("Failed to set SKIP_FLAKES env var %s", err)
	}
	common.NamespacePrefix = possibleNS
	common.RHOAMOperatorNamespace = common.NamespacePrefix + "operator"
	common.ObservabilityNamespacePrefix = common.NamespacePrefix + "observability-"
	common.ObservabilityProductNamespace = common.NamespacePrefix + "observability"
	common.CloudResourceOperatorNamespace = common.NamespacePrefix + "cloud-resources-operator"
	common.RHSSOUserProductNamespace = common.NamespacePrefix + "user-sso"
	common.RHSSOUserOperatorNamespace = common.RHSSOUserProductNamespace + "-operator"
	common.RHSSOProductNamespace = common.NamespacePrefix + "rhsso"
	common.RHSSOOperatorNamespace = common.RHSSOProductNamespace + "-operator"
	common.ThreeScaleProductNamespace = common.NamespacePrefix + "3scale"
	common.ThreeScaleOperatorNamespace = common.ThreeScaleProductNamespace + "-operator"
	common.Marin3rOperatorNamespace = common.NamespacePrefix + "marin3r-operator"
	common.Marin3rProductNamespace = common.NamespacePrefix + "marin3r"
	common.SMTPSecretName = common.NamespacePrefix + "smtp"
}
