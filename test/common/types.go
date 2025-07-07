package common

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/integr8ly/integreatly-operator/pkg/resources/k8s"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	NamespacePrefix                = GetNamespacePrefix()
	OpenShiftConsoleRoute          = "console"
	OpenShiftConsoleNamespace      = "openshift-console"
	RHOAMOperatorNamespace         = NamespacePrefix + "operator"
	ObservabilityProductNamespace  = NamespacePrefix + "operator-observability"
	ObservabilityNamespacePrefix   = ObservabilityProductNamespace + "-"
	ObservabilityPrometheusPodName = "prometheus-rhoam-0"
	SMTPSecretName                 = NamespacePrefix + "smtp"
	DMSSecretName                  = NamespacePrefix + "deadmanssnitch"
	CloudResourceOperatorNamespace = NamespacePrefix + "cloud-resources-operator"
	RHSSOUserProductNamespace      = NamespacePrefix + "user-sso"
	RHSSOUserOperatorNamespace     = RHSSOUserProductNamespace + "-operator"
	RHSSOProductNamespace          = NamespacePrefix + "rhsso"
	RHSSOOperatorNamespace         = RHSSOProductNamespace + "-operator"
	ThreeScaleProductNamespace     = NamespacePrefix + "3scale"
	ThreeScaleOperatorNamespace    = ThreeScaleProductNamespace + "-operator"
	Marin3rOperatorNamespace       = NamespacePrefix + "marin3r-operator"
	Marin3rProductNamespace        = NamespacePrefix + "marin3r"
	CustomerGrafanaNamespace       = NamespacePrefix + "customer-monitoring"
	CustomerGrafanaNamespacePrefix = CustomerGrafanaNamespace + "-"
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
	ns, err := k8s.GetWatchNamespace()
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

type CustomResource struct {
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

type keycloakUser struct {
	Access                     *map[string]bool     `json:"access,omitempty"`
	Attributes                 *map[string][]string `json:"attributes,omitempty"`
	ClientRoles                *map[string][]string `json:"clientRoles,omitempty"`
	CreatedTimestamp           *int64               `json:"createdTimestamp,omitempty"`
	DisableableCredentialTypes *[]interface{}       `json:"disableableCredentialTypes,omitempty"`
	Email                      *string              `json:"email,omitempty"`
	EmailVerified              *bool                `json:"emailVerified,omitempty"`
	Enabled                    *bool                `json:"enabled,omitempty"`
	FederationLink             *string              `json:"federationLink,omitempty"`
	FirstName                  *string              `json:"firstName,omitempty"`
	Groups                     *[]string            `json:"groups,omitempty"`
	ID                         *string              `json:"id,omitempty"`
	LastName                   *string              `json:"lastName,omitempty"`
	RealmRoles                 *[]string            `json:"realmRoles,omitempty"`
	RequiredActions            *[]string            `json:"requiredActions,omitempty"`
	ServiceAccountClientID     *string              `json:"serviceAccountClientId,omitempty"`
	UserName                   *string              `json:"username,omitempty"`
}

type keycloakUserOptions struct {
	BriefRepresentation *bool
	Email               *string
	EmailVerified       *bool
	Enabled             *bool
	Exact               *bool
	First               *int32
	FirstName           *string
	IDPAlias            *string
	IDPUserID           *string
	LastName            *string
	Max                 *int32
	Q                   *string
	RealmName           string
	Search              *string
	Username            *string
}

type keycloakUserGroup struct {
	ID   *string `json:"id,omitempty"`
	Name *string `json:"name,omitempty"`
	Path *string `json:"path,omitempty"`
}

type keycloakUserGroupOptions struct {
	BriefRepresentation *bool
	First               *int32
	Max                 *int32
	RealmName           string
	Search              *string
	UserID              string
}

type keycloakTokenOptions struct {
	ClientID     *string
	GrantType    *string
	RealmName    string
	RefreshToken *string
	Username     *string
	Password     *string
}

type keycloakOpenIDTokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int32  `json:"expires_in"`
	RefreshExpiresIn int32  `json:"refresh_expires_in"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	Scope            string `json:"scope"`
}
