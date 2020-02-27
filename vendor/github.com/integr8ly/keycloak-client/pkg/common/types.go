package common

// Group representation
// https://www.keycloak.org/docs-api/9.0/rest-api/index.html#_grouprepresentation
type Group struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}
