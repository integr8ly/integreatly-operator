package functional

import (
	"fmt"
	"os"
	"testing"

	"github.com/integr8ly/integreatly-operator/test/utils"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/common"
)

const (
	testResultsDirectory = "/test-run-results"
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
	cfg.Impersonate = rest.ImpersonationConfig{
		UserName: "system:admin",
		Groups:   []string{"system:authenticated"},
	}

	context, err := common.NewTestingContext(cfg)
	if err != nil {
		t.Fatalf("\"failed to create testing context: %s", err)
	}
	k8sClient = context.Client

	//Allow overriding via environment variable
	if os.Getenv("BYPASS_STORAGE_TYPE_CHECK") != "true" {
		rhmi, err := common.GetRHMI(k8sClient, true)
		if err != nil {
			t.Fatalf("error getting RHMI CR: %v", err)
		}
		// For now, we want to allow RHOAM multitenant to be tested on "on-cluster" storage
		if !rhmiv1alpha1.IsRHOAMMultitenant(rhmiv1alpha1.InstallationType(rhmi.Spec.Type)) {
			if rhmi.Spec.UseClusterStorage == "true" {
				t.Skip("Aborting functional tests: \"UseClusterStorage\" is set to true. \nPlease, run another testing suite or reinstall operator with \"UseClusterStorage\" set to false")
			}
		}
	}

	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("%s/%s", testResultsDirectory, utils.JUnitFileName(testSuiteName)))

	RunSpecsWithDefaultAndCustomReporters(t,
		utils.SpecDescription("Functional Test Suite"),
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
