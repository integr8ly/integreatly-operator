package osde2e

import (
	"fmt"

	"github.com/integr8ly/integreatly-operator/test/common"
	. "github.com/onsi/ginkgo/v2"
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
		// get all automated tests
		tests := []common.Tests{
			{
				Type:      "Integreatly Operator pre-test",
				TestCases: OSD_E2E_PRE_TESTS,
			},
			{
				Type:      fmt.Sprintf("%s HAPPY PATH", installType),
				TestCases: common.GetHappyPathTestCases(installType),
			},
			{
				Type:      fmt.Sprintf("%s OBSERVABILITY TESTS", installType),
				TestCases: common.GetObservabilityTestCases(installType),
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
