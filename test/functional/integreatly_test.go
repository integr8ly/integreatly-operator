package functional

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	v1 "github.com/openshift/api/config/v1"
	"os"

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

		testingContext, err := common.NewTestingContext(restConfig)
		platform, err := resources.GetPlatformType(context.TODO(), testingContext.Client)
		if err != nil {
			t.Fatal("failed to determine platform type", err)
		}
		var functionalTests common.Tests
		switch platform {
		case v1.AWSPlatformType:
			functionalTests = common.Tests{
				Type:      fmt.Sprintf("%s Functional", installType),
				TestCases: FUNCTIONAL_TESTS_AWS,
			}
			tests = append(tests, common.Tests{
				Type:      "AWS Specific Tests",
				TestCases: common.GetAWSSpecificTestCases(installType),
			})
		case v1.GCPPlatformType:
			functionalTests = common.Tests{
				Type:      fmt.Sprintf("%s Functional", installType),
				TestCases: FUNCTIONAL_TESTS_GCP,
			}
			tests = append(tests, common.Tests{
				Type:      "GCP Tests",
				TestCases: common.GetGCPTestCases(installType),
			})
		}
		// Run functional (AWS or GCP) tests only in case of cloud provider storage type installation (useClusterStorage: false)
		// or if overriden by BYPASS_STORAGE_TYPE_CHECK=true env var
		if shouldRunFunctionalTests() {
			tests = append(tests, functionalTests)
		}

		if os.Getenv("MULTIAZ") == "true" {
			tests = append(tests, common.Tests{
				Type:      fmt.Sprintf("%s Multi AZ", installType),
				TestCases: MULTIAZ_TESTS,
			})
		}
		//Function to be used when Threescale becomes cluster scoped.
		// clusterScoped, err := common.IsClusterScoped(restConfig)
		// if err != nil {
		// 	t.Error(err)
		// }
		// if clusterScoped {
		// 	tests = append(tests, common.Tests{
		// 		Type:      fmt.Sprintf("%s Threescale Cluster Scoped", installType),
		// 		TestCases: common.GetClusterScopedTestCases(installType),
		// 	})
		// }

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
