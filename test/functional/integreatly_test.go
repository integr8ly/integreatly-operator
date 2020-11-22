package functional

import (
	"os"
	"testing"

	"k8s.io/client-go/rest"

	"github.com/integr8ly/integreatly-operator/test/common"
	runtimeConfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

func TestIntegreatly(t *testing.T) {
	config, err := runtimeConfig.GetConfig()
	config.Impersonate = rest.ImpersonationConfig{
		UserName: "system:admin",
		Groups:   []string{"system:authenticated"},
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Run("Integreatly Happy Path Tests", func(t *testing.T) {

		// running ALL_TESTS test cases
		common.RunTestCases(common.ALL_TESTS, t, config)

		installType, err := common.GetInstallType(config)
		if err != nil {
			t.Fatalf("failed to get install type, err: %s, %v", installType, err)
		}

		// get happy path test cases according to the install type
		happyPathTestCases := common.GetHappyPathTestCases(installType)

		// running HAPPY_PATH_TESTS tests cases
		common.RunTestCases(happyPathTestCases, t, config)

		// running functional tests
		common.RunTestCases(FUNCTIONAL_TESTS, t, config)
	})

	t.Run("Integreatly IDP Based Tests", func(t *testing.T) {
		installType, err := common.GetInstallType(config)
		if err != nil {
			t.Fatalf("failed to get install type, err: %s, %v", installType, err)
		}

		// get IDP test cases according to the install type
		idpTestCases := common.GetIDPBasedTestCases(installType)

		// running IDP Based test cases
		common.RunTestCases(idpTestCases, t, config)
	})

	t.Run("API Managed Multi-AZ Tests", func(t *testing.T) {
		// Do not execute these tests unless MULTIAZ is set to true
		if os.Getenv("MULTIAZ") != "true" {
			t.Skip("Skipping Multi-AZ tests as MULTIAZ env var is not set to true")
		}

		common.RunTestCases(MULTIAZ_TESTS, t, config)
	})

	t.Run("Integreatly Destructive Tests", func(t *testing.T) {
		// Do not execute these tests unless DESTRUCTIVE is set to true
		if os.Getenv("DESTRUCTIVE") != "true" {
			t.Skip("Skipping Destructive tests as DESTRUCTIVE env var is not set to true")
		}

		common.RunTestCases(common.DESTRUCTIVE_TESTS, t, config)
	})
}
