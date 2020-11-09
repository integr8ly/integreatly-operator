package osde2e

import (
	"testing"

	"github.com/integr8ly/integreatly-operator/test/common"
	"k8s.io/client-go/rest"
	runtimeConfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	InstallationName = "managed-api"
)

func TestManagedAPI(t *testing.T) {
	config, err := runtimeConfig.GetConfig()
	config.Impersonate = rest.ImpersonationConfig{
		UserName: "system:admin",
		Groups:   []string{"system:authenticated"},
	}
	if err != nil {
		t.Fatal(err)
	}

	InstallationName, err = common.GetInstallType(config)
	if err != nil {
		t.Fatal(err)
	}

	happyPathTestCases := common.GetHappyPathTestCases(InstallationName)

	t.Run("Managed-API-pre-e2e-test", func(t *testing.T) {
		common.RunTestCases(OSD_E2E_PRE_TESTS, t, config)
	})

	t.Run("Managed-API-e2e-test", func(t *testing.T) {
		common.RunTestCases(happyPathTestCases, t, config)
	})
}
