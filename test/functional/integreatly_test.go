package functional

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

	JustBeforeEach(func() {
		// If we're running the post uninstall tests, don't wait for the
		// RHMI to be marked as completed
		if os.Getenv("POST_UNINSTALL") == "true" {
			return
		}

		if err := common.WaitForRHMIStageToComplete(t, restConfig); err != nil {
			t.Error(err)
		}
	})

	RunTests := func() {

		// get all automated tests
		tests := []common.Tests{
			{
				Type:      fmt.Sprintf("%s ALL TESTS", installType),
				TestCases: common.GetAllTestCases(installType),
			},
			{
				Type:      fmt.Sprintf("%s HAPPY PATH", installType),
				TestCases: common.GetHappyPathTestCases(installType),
			},
			{
				Type:      fmt.Sprintf("%s IDP BASED", installType),
				TestCases: common.GetIDPBasedTestCases(installType),
			},
			{
				Type:      fmt.Sprintf("%s Functional", installType),
				TestCases: FUNCTIONAL_TESTS,
			},
			{
				Type:      fmt.Sprintf("%s SCALABILITY TESTS", installType),
				TestCases: common.GetScalabilityTestCases(installType),
			},
			{
				Type:      "FAILURE TESTS",
				TestCases: common.FAILURE_TESTS,
			},
		}

		if os.Getenv("MULTIAZ") == "true" {
			tests = append(tests, common.Tests{
				Type:      fmt.Sprintf("%s Multi AZ", installType),
				TestCases: MULTIAZ_TESTS,
			})
		}

		if os.Getenv("DESTRUCTIVE") == "true" {
			tests = append(tests, common.Tests{
				Type:      "Destructive Tests",
				TestCases: common.DESTRUCTIVE_TESTS,
			})
		}

		if os.Getenv("POST_UNINSTALL") == "true" {
			// Note: assign instead of append in order to run **exclusively**
			// the post uninstall tests
			tests = []common.Tests{
				{
					Type:      "Post Uninstall Tests",
					TestCases: POST_UNINSTALL_TESTS,
				},
			}
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
