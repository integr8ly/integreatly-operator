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
	NamespacePrefix                   = "redhat-rhmi-"
	RHMIOperatorNamespace             = NamespacePrefix + "operator"
	MonitoringOperatorNamespace       = NamespacePrefix + "middleware-monitoring-operator"
	AMQOnlineOperatorNamespace        = NamespacePrefix + "amq-online"
	ApicuritoOperatorNamespace        = NamespacePrefix + "apicurito-operator"
	CloudResourceOperatorNamespace    = NamespacePrefix + "cloud-resources-operator"
	CodeReadyOperatorNamespace        = NamespacePrefix + "codeready-workspaces-operator"
	FuseOperatorNamespace             = NamespacePrefix + "fuse-operator"
	RHSSOUserOperatorNamespace        = NamespacePrefix + "user-sso-operator"
	RHSSOOperatorNamespace            = NamespacePrefix + "rhsso-operator"
	SolutionExplorerOperatorNamespace = NamespacePrefix + "solution-explorer-operator"
	ThreeScaleOperatorNamespace       = NamespacePrefix + "3scale-operator"
	UPSOperatorNamespace              = NamespacePrefix + "ups-operator"
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
