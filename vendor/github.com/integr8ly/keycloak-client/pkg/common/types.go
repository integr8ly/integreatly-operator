package common

import (
	"github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
)

type Requirement string

var (
	Required    Requirement = "REQUIRED"
	Conditional Requirement = "CONDITIONAL"
	Alternative Requirement = "ALTERNATIVE"
	Disabled    Requirement = "DISABLED"
)

// Group representation
// https://www.keycloak.org/docs-api/9.0/rest-api/index.html#_grouprepresentation
type Group struct {
	Name      string   `json:"name,omitempty"`
	ID        string   `json:"id,omitempty"`
	SubGroups []*Group `json:"subGroups,omitempty"`
}

// AuthenticationFlow representation
// https://www.keycloak.org/docs-api/9.0/rest-api/index.html#_authenticationflowrepresentation
type AuthenticationFlow struct {
	Alias       string `json:"alias,omitempty"`
	BuiltIn     bool   `json:"builtIn,omitempty"`
	Description string `json:"description,omitempty"`
	ID          string `json:"id,omitempty"`
	ProviderID  string `json:"providerId,omitempty"`
	TopLevel    bool   `json:"topLevel,omitempty"`
}

type UserAttributes struct {
	User      v1alpha1.KeycloakAPIUser `json:"user,omitempty"`
	Attribute map[string]string        `json:"attribute,omitempty"`
}
