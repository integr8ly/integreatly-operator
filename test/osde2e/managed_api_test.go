package osde2e

import (
	"fmt"
	"os"

	"github.com/integr8ly/integreatly-operator/test/common"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("integreatly", func() {

	var (
		restConfig = cfg
		t          = GinkgoT()
	)

	BeforeEach(func() {
		restConfig = cfg
		t = GinkgoT()
	})

	RunTests := func() {

		os.Setenv("WATCH_NAMESPACE", "redhat-rhoam-operator")
		os.Setenv("SKIP_FLAKES", "true")
		common.NamespacePrefix = "redhat-rhoam-"
		common.RHMIOperatorNamespace = common.NamespacePrefix + "operator"
		common.MonitoringOperatorNamespace = common.NamespacePrefix + "middleware-monitoring-operator"
		common.MonitoringFederateNamespace = common.NamespacePrefix + "middleware-monitoring-federate"
		common.AMQOnlineOperatorNamespace = common.NamespacePrefix + "amq-online"
		common.ApicurioRegistryProductNamespace = common.NamespacePrefix + "apicurio-registry"
		common.ApicurioRegistryOperatorNamespace = common.ApicurioRegistryProductNamespace + "-operator"
		common.ApicuritoProductNamespace = common.NamespacePrefix + "apicurito"
		common.ApicuritoOperatorNamespace = common.ApicuritoProductNamespace + "-operator"
		common.CloudResourceOperatorNamespace = common.NamespacePrefix + "cloud-resources-operator"
		common.CodeReadyProductNamespace = common.NamespacePrefix + "codeready-workspaces"
		common.CodeReadyOperatorNamespace = common.CodeReadyProductNamespace + "-operator"
		common.FuseProductNamespace = common.NamespacePrefix + "fuse"
		common.FuseOperatorNamespace = common.FuseProductNamespace + "-operator"
		common.RHSSOUserProductOperatorNamespace = common.NamespacePrefix + "user-sso"
		common.RHSSOUserOperatorNamespace = common.RHSSOUserProductOperatorNamespace + "-operator"
		common.RHSSOProductNamespace = common.NamespacePrefix + "rhsso"
		common.RHSSOOperatorNamespace = common.RHSSOProductNamespace + "-operator"
		common.SolutionExplorerProductNamespace = common.NamespacePrefix + "solution-explorer"
		common.SolutionExplorerOperatorNamespace = common.SolutionExplorerProductNamespace + "-operator"
		common.ThreeScaleProductNamespace = common.NamespacePrefix + "3scale"
		common.ThreeScaleOperatorNamespace = common.ThreeScaleProductNamespace + "-operator"
		common.UPSProductNamespace = common.NamespacePrefix + "ups"
		common.UPSOperatorNamespace = common.UPSProductNamespace + "-operator"
		common.MonitoringSpecNamespace = common.NamespacePrefix + "monitoring"
		common.Marin3rOperatorNamespace = common.NamespacePrefix + "marin3r-operator"
		common.Marin3rProductNamespace = common.NamespacePrefix + "marin3r"
		common.CustomerGrafanaNamespace = common.NamespacePrefix + "customer-monitoring-operator"

		// get all automated tests
		tests := []common.Tests{
			{
				Type:      fmt.Sprintf("%s HAPPY PATH", installType),
				TestCases: common.GetHappyPathTestCases(installType),
			},
			{
				Type:      fmt.Sprintf("%s pre-test", installType),
				TestCases: OSD_E2E_PRE_TESTS,
			},
		}

		for _, test := range tests {
			Context(test.Type, func() {
				for _, testCase := range test.TestCases {
					currentTest := testCase
					It(currentTest.Description, func() {
						testingContext, err := common.NewTestingContext(restConfig)
						if err != nil {
							t.Fatal("failed to create testing context", err)
						}
						currentTest.Test(t, testingContext)
					})
				}
			})
		}

	}

	RunTests()

})
