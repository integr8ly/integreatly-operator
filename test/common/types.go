package common

import (
	"encoding/json"
	"net/http"
	"strings"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	NamespacePrefix                   = GetNamespacePrefix()
	OpenShiftConsoleRoute             = "console"
	OpenShiftConsoleNamespace         = "openshift-console"
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
	Test        func(t TestingTB, ctx *TestingContext)
}

type TestSuite struct {
	TestCases   []TestCase
	InstallType []rhmiv1alpha1.InstallationType
}

type Tests struct {
	Type      string
	TestCases []TestCase
}

type prometheusAPIResponse struct {
	Status    string                 `json:"status"`
	Data      json.RawMessage        `json:"data"`
	ErrorType prometheusv1.ErrorType `json:"errorType"`
	Error     string                 `json:"error"`
	Warnings  []string               `json:"warnings,omitempty"`
}

type TestingTB interface {
	Fail()
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	FailNow()
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Log(args ...interface{})
	Logf(format string, args ...interface{})
	Failed() bool
	Parallel()
	Skip(args ...interface{})
	Skipf(format string, args ...interface{})
	SkipNow()
	Skipped() bool
}

type SubscriptionCheck struct {
	Name      string
	Namespace string
}

func GetNamespacePrefix() string {
	ns, err := resources.GetWatchNamespace()
	if err != nil {
		return ""
	}
	return strings.Join(strings.Split(ns, "-")[0:2], "-") + "-"

}

func GetPrefixedNamespace(subNS string) string {
	return NamespacePrefix + subNS
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

type PersistentVolumeClaim struct {
	Namespace                  string
	PersistentVolumeClaimNames []string
}

type StatefulSets struct {
	Namespace string
	Name      string
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

type DeploymentConfigs struct {
	Namespace string
	Name      string
}
