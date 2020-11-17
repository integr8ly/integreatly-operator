package common

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	NamespacePrefix                   = GetNamespacePrefix()
	RHMIOperatorNamespace             = NamespacePrefix + "operator"
	MonitoringOperatorNamespace       = NamespacePrefix + "middleware-monitoring-operator"
	MonitoringFederateNamespace       = NamespacePrefix + "middleware-monitoring-federate"
	AMQOnlineOperatorNamespace        = NamespacePrefix + "amq-online"
	ApicurioRegistryProductNamespace  = NamespacePrefix + "apicurio-registry"
	ApicurioRegistryOperatorNamespace = ApicurioRegistryProductNamespace + "-operator"
	ApicuritoProductNamespace         = NamespacePrefix + "apicurito"
	ApicuritoOperatorNamespace        = ApicuritoProductNamespace + "-operator"
	CloudResourceOperatorNamespace    = NamespacePrefix + "cloud-resources-operator"
	CodeReadyProductNamespace         = NamespacePrefix + "codeready-workspaces"
	CodeReadyOperatorNamespace        = CodeReadyProductNamespace + "-operator"
	FuseProductNamespace              = NamespacePrefix + "fuse"
	FuseOperatorNamespace             = FuseProductNamespace + "-operator"
	RHSSOUserProductOperatorNamespace = NamespacePrefix + "user-sso"
	RHSSOUserOperatorNamespace        = RHSSOUserProductOperatorNamespace + "-operator"
	RHSSOProductNamespace             = NamespacePrefix + "rhsso"
	RHSSOOperatorNamespace            = RHSSOProductNamespace + "-operator"
	SolutionExplorerProductNamespace  = NamespacePrefix + "solution-explorer"
	SolutionExplorerOperatorNamespace = SolutionExplorerProductNamespace + "-operator"
	ThreeScaleProductNamespace        = NamespacePrefix + "3scale"
	ThreeScaleOperatorNamespace       = ThreeScaleProductNamespace + "-operator"
	UPSProductNamespace               = NamespacePrefix + "ups"
	UPSOperatorNamespace              = UPSProductNamespace + "-operator"
	MonitoringSpecNamespace           = NamespacePrefix + "monitoring"
	Marin3rOperatorNamespace          = NamespacePrefix + "marin3r-operator"
	Marin3rProductNamespace           = NamespacePrefix + "marin3r"
	CustomerGrafanaNamespace          = NamespacePrefix + "customer-monitoring-operator"
	OpenShiftConsoleRoute             = "console"
	OpenShiftConsoleNamespace         = "openshift-console"
)

type TestingContext struct {
	Client          dynclient.Client
	KubeConfig      *rest.Config
	KubeClient      kubernetes.Interface
	ExtensionClient *clientset.Clientset
	HttpClient      *http.Client
	SelfSignedCerts bool
}

type TestCase struct {
	Description string
	Test        func(t *testing.T, ctx *TestingContext)
}

type TestSuite struct {
	TestCases   []TestCase
	InstallType []integreatlyv1alpha1.InstallationType
}

type prometheusAPIResponse struct {
	Status    string                 `json:"status"`
	Data      json.RawMessage        `json:"data"`
	ErrorType prometheusv1.ErrorType `json:"errorType"`
	Error     string                 `json:"error"`
	Warnings  []string               `json:"warnings,omitempty"`
}

type Namespace struct {
	Name                     string
	Products                 []Product
	PodDisruptionBudgetNames []string
}

type Product struct {
	Name             string
	ExpectedReplicas int32
}

// ExpectedRoute contains the data of a route that is expected to be found
type ExpectedRoute struct {
	// Name is either the name of the route or the generated name (if the
	// `IsGeneratedName` field is true)
	Name string

	isTLS bool

	// ServiceName is the name of the service that the route points to (used
	// when the name is generated as there can be multiple routes with the same
	// generated name)
	ServiceName string

	IsGeneratedName bool
}

type SubscriptionCheck struct {
	Name      string
	Namespace string
}

type PersistentVolumeClaim struct {
	Namespace                  string
	PersistentVolumeClaimNames []string
}

type StatefulSets struct {
	Namespace string
	Name      string
}

type DeploymentConfigs struct {
	Namespace string
	Name      string
}

func GetNamespacePrefix() string {
	ns, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return ""
	}
	return strings.Join(strings.Split(ns, "-")[0:2], "-") + "-"

}

func GetPrefixedNamespace(subNS string) string {
	return NamespacePrefix + subNS
}
