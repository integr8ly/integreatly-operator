package common

import (
	"encoding/json"
	"net/http"
	"testing"

	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	InstallationName                  = "rhmi"
	NamespacePrefix                   = "redhat-integration"
	RHMIOperatorNamespace             = NamespacePrefix + "operator"
	MonitoringOperatorNamespace       = NamespacePrefix + "middleware-monitoring-operator"
	MonitoringFederateNamespace       = NamespacePrefix + "middleware-monitoring-federate"
	AMQOnlineOperatorNamespace        = NamespacePrefix + "amq-online"
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
