package functional

import (
	"fmt"
	"testing"

	threescaleBv1 "github.com/3scale/3scale-operator/apis/capabilities/v1beta1"
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

	threescalev1 "github.com/3scale/3scale-operator/apis/capabilities/v1alpha1"
	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/test/common"
	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
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

	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("%s/%s", testResultsDirectory, utils.JUnitFileName(testSuiteName)))

	RunSpecsWithDefaultAndCustomReporters(t,
		utils.SpecDescription("Functional Test Suite"),
		[]Reporter{junitReporter})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	err = rhmiv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = threescalev1.SchemeBuilder.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = threescaleBv1.SchemeBuilder.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = operatorsv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	// +kubebuilder:scaffold:scheme

	close(done)
}, 120)

var _ = AfterSuite(func() {
	By("tearing down the test environment")

	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
