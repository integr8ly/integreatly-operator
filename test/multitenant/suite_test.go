package multitenant

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
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
var k8sClient client.Client
var testEnv *envtest.Environment
var installType string
var err error

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

	// Set envvars
	setVars("redhat-rhoam-operator", "redhat-rhoam-", t)
	installType, err = common.GetInstallType(cfg)
	if err != nil {
		t.Fatalf("could not get install type %s", err)
	}

	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("%s/%s", testResultsDirectory, jUnitOutputFilename))

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{junitReporter})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	err = rhmiv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	close(done)
}, 120)

var _ = AfterSuite(func() {
	By("tearing down the test environment")

	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func setVars(watchNamespace, ns string, t *testing.T) {
	err := os.Setenv("WATCH_NAMESPACE", watchNamespace)
	if err != nil {
		t.Fatalf("Failed to set WATCH_NAMESPACE env var %s", err)
	}
	err = os.Setenv("SKIP_FLAKES", "true")
	if err != nil {
		t.Logf("Failed to set SKIP_FLAKES env var %s", err)
	}
	common.NamespacePrefix = ns
	common.RHMIOperatorNamespace = common.NamespacePrefix + "operator"
	common.MonitoringOperatorNamespace = common.NamespacePrefix + "middleware-monitoring-operator"
	common.MonitoringFederateNamespace = common.NamespacePrefix + "middleware-monitoring-federate"
	common.CloudResourceOperatorNamespace = common.NamespacePrefix + "cloud-resources-operator"
	common.RHSSOUserProductNamespace = common.NamespacePrefix + "user-sso"
	common.RHSSOUserOperatorNamespace = common.RHSSOUserProductNamespace + "-operator"
	common.RHSSOProductNamespace = common.NamespacePrefix + "rhsso"
	common.RHSSOOperatorNamespace = common.RHSSOProductNamespace + "-operator"
	common.ThreeScaleProductNamespace = common.NamespacePrefix + "3scale"
	common.ThreeScaleOperatorNamespace = common.ThreeScaleProductNamespace + "-operator"
	common.MonitoringSpecNamespace = common.NamespacePrefix + "monitoring"
	common.Marin3rOperatorNamespace = common.NamespacePrefix + "marin3r-operator"
	common.Marin3rProductNamespace = common.NamespacePrefix + "marin3r"
	common.CustomerGrafanaNamespace = common.NamespacePrefix + "customer-monitoring-operator"
}
