package functional

import (
	"fmt"
	"os"
	"testing"
	"time"

	threescaleBv1 "github.com/3scale/3scale-operator/apis/capabilities/v1beta1"
	"github.com/integr8ly/integreatly-operator/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	threescalev1 "github.com/3scale/3scale-operator/apis/capabilities/v1alpha1"
	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/common"
	configv1 "github.com/openshift/api/config/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	packageOperatorv1alpha1 "package-operator.run/apis/core/v1alpha1"
)

const (
	testSuiteName = "integreatly-operator"
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
	cfg.Impersonate = rest.ImpersonationConfig{
		UserName: "system:admin",
		Groups:   []string{"system:authenticated"},
	}

	// get install type
	installType, err = common.GetInstallType(cfg)
	if err != nil {
		t.Fatalf("could not get install type %s", err)
	}
	testResultsDirectory := os.Getenv("OUTPUT_DIR")
	if len(testResultsDirectory) == 0 {
		testResultsDirectory = "/test-run-results"
	}
	jUnitReportLocation := fmt.Sprintf("%s/%s", testResultsDirectory, utils.JUnitFileName(testSuiteName))

	// Fetch the current config
	suiteConfig, reporterConfig := GinkgoConfiguration()
	// This timeout is slightly less that 3h because that is the default timeout on ci-cd level
	suiteConfig.Timeout = time.Minute * 170
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

		err = threescalev1.SchemeBuilder.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		err = threescaleBv1.SchemeBuilder.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		err = operatorsv1alpha1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())
		// +kubebuilder:scaffold:scheme

		err = configv1.Install(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		err = packageOperatorv1alpha1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		close(done)
	}()
	Eventually(done, 120).Should(BeClosed())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")

	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
