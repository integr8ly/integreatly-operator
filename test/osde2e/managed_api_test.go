package osde2e

import (
	"os"
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

	os.Setenv("WATCH_NAMESPACE", "redhat-rhoam-operator")
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
