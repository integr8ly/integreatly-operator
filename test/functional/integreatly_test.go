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
		if err := common.WaitForRHMIStageToComplete(t, restConfig); err != nil {
			t.Error(err)
		}
	})

	shouldRunFunctionalTests := func() bool {
		context, err := common.NewTestingContext(cfg)
		if err != nil {
			t.Fatalf("\"failed to create testing context: %s", err)
		}

		rhmi, err := common.GetRHMI(context.Client, true)
		if err != nil {
			t.Fatalf("error getting RHMI CR: %v", err)
		}

		return rhmi.Spec.UseClusterStorage == "false" || os.Getenv("BYPASS_STORAGE_TYPE_CHECK") == "true"
	}

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
				Type:      fmt.Sprintf("%s SCALABILITY TESTS", installType),
				TestCases: common.GetScalabilityTestCases(installType),
			},
			{
				Type:      "FAILURE TESTS",
				TestCases: common.FAILURE_TESTS,
			},
		}

		// Run functional (AWS) tests only in case of AWS storage type installation (useClusterStorage: false)
		// or if overriden by BYPASS_STORAGE_TYPE_CHECK=true env var
		if shouldRunFunctionalTests() {
			tests = append(tests, common.Tests{
				Type:      fmt.Sprintf("%s Functional", installType),
				TestCases: FUNCTIONAL_TESTS,
			})
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
