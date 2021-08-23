package functional

import (
	"fmt"
	"os"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
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

	rhmi, err := common.GetRHMI(k8sClient, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	RunTests := func() {

		// get all automated tests
		tests := getTestsBasedOnInstallType(*rhmi)

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

func getTestsBasedOnInstallType(rhmi rhmiv1alpha1.RHMI) []common.Tests {
	if rhmiv1alpha1.IsRHOAMMultitenant(rhmiv1alpha1.InstallationType(rhmi.Spec.Type)) {
		return []common.Tests{
			{
				Type:      "MULTITENANCY LOAD TESTS",
				TestCases: common.MULTITENANCY_LOAD_TESTS,
			},
		}
	} else {
		return []common.Tests{
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
	}
}
