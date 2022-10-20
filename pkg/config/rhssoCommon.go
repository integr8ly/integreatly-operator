package config

import (
	"errors"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	dr "github.com/integr8ly/integreatly-operator/pkg/resources/dynamic-resources"
	keycloak "github.com/integr8ly/keycloak-client/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
)

// A common rhsso interface
type RHSSOInterface interface {
	GetProductName() integreatlyv1alpha1.ProductName
	GetOperatorVersion() integreatlyv1alpha1.OperatorVersion
	Validate() error
}

type RHSSOCommon struct {
	Config ProductConfig
}

func NewRHSSOCommon(config ProductConfig) *RHSSOCommon {
	return &RHSSOCommon{Config: config}
}

//GetWatchableCRDs to trigger a reconcile of the integreatly installation when these are updated
func (r *RHSSOCommon) GetWatchableCRDs() []runtime.Object {
	keycloakUnstructured := dr.CreateUnstructuredWithGVK(keycloak.KeycloakGroup, keycloak.KeycloakKind, keycloak.KeycloakVersion, "", "")
	keycloakRealmUnstructured := dr.CreateUnstructuredWithGVK(keycloak.KeycloakRealmGroup, keycloak.KeycloakRealmKind, keycloak.KeycloakRealmVersion, "", "")
	keycloakUserUnstructured := dr.CreateUnstructuredWithGVK(keycloak.KeycloakUserGroup, keycloak.KeycloakUserKind, keycloak.KeycloakUserVersion, "", "")
	keycloakClientUnstructured := dr.CreateUnstructuredWithGVK(keycloak.KeycloakClientGroup, keycloak.KeycloakClientKind, keycloak.KeycloakClientVersion, "", "")
	keycloakBackupUnstructured := dr.CreateUnstructuredWithGVK(keycloak.KeycloakBackupGroup, keycloak.KeycloakBackupKind, keycloak.KeycloakBackupVersion, "", "")
	return []runtime.Object{
		keycloakUnstructured,
		keycloakRealmUnstructured,
		keycloakUserUnstructured,
		keycloakClientUnstructured,
		keycloakBackupUnstructured,
	}
}

func (r *RHSSOCommon) GetNamespace() string {
	return r.Config["NAMESPACE"]
}

func (r *RHSSOCommon) SetNamespace(newNamespace string) {
	r.Config["NAMESPACE"] = newNamespace
}

func (r *RHSSOCommon) GetOperatorNamespace() string {
	return r.Config["OPERATOR_NAMESPACE"]
}

func (r *RHSSOCommon) SetOperatorNamespace(newNamespace string) {
	r.Config["OPERATOR_NAMESPACE"] = newNamespace
}

func (r *RHSSOCommon) GetRealm() string {
	return r.Config["REALM"]
}

func (r *RHSSOCommon) SetRealm(newRealm string) {
	r.Config["REALM"] = newRealm
}

func (r *RHSSOCommon) GetHost() string {
	return r.Config["HOST"]
}

func (r *RHSSOCommon) SetHost(newHost string) {
	r.Config["HOST"] = newHost
}

func (r *RHSSOCommon) Read() ProductConfig {
	return r.Config
}

func (r *RHSSOCommon) GetLabelSelector() string {
	return "middleware"
}

func (r *RHSSOCommon) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.ProductVersion(r.Config["VERSION"])
}

func (r *RHSSOCommon) SetProductVersion(version string) {
	r.Config["VERSION"] = version
}

func (r *RHSSOCommon) SetOperatorVersion(operator string) {
	r.Config["OPERATOR"] = operator
}

func (r *RHSSOCommon) ValidateCommon() error {
	if r.GetRealm() == "" {
		return errors.New("config realm is not defined")
	}
	if r.GetNamespace() == "" {
		return errors.New("config namespace is not defined")
	}
	if r.GetHost() == "" {
		return errors.New("config url is not defined")
	}
	return nil
}
