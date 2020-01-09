package config

import (
	"errors"

	aerogearv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
)

type RHSSO struct {
	Config ProductConfig
}

func NewRHSSO(config ProductConfig) *RHSSO {
	return &RHSSO{Config: config}
}

//GetWatchableCRDs to trigger a reconcile of the integreatly installation when these are updated
func (r *RHSSO) GetWatchableCRDs() []runtime.Object {
	return []runtime.Object{
		&aerogearv1.Keycloak{
			TypeMeta: metav1.TypeMeta{
				Kind:       aerogearv1.KeycloakKind,
				APIVersion: aerogearv1.SchemeGroupVersion.String(),
			},
		},
		&aerogearv1.KeycloakRealm{
			TypeMeta: metav1.TypeMeta{
				Kind:       aerogearv1.KeycloakRealmKind,
				APIVersion: aerogearv1.SchemeGroupVersion.String(),
			},
		},
	}
}

func (r *RHSSO) GetNamespace() string {
	return r.Config["NAMESPACE"]
}

func (r *RHSSO) SetNamespace(newNamespace string) {
	r.Config["NAMESPACE"] = newNamespace
}

func (r *RHSSO) GetRealm() string {
	return r.Config["REALM"]
}

func (r *RHSSO) SetRealm(newRealm string) {
	r.Config["REALM"] = newRealm
}

func (r *RHSSO) GetHost() string {
	return r.Config["HOST"]
}

func (r *RHSSO) SetHost(newHost string) {
	r.Config["HOST"] = newHost
}

func (r *RHSSO) Read() ProductConfig {
	return r.Config
}

func (r *RHSSO) GetLabelSelector() string {
	return "middleware"
}

func (r *RHSSO) GetTemplateList() []string {
	template_list := []string{
		"kube_state_metrics_rhsso_alerts.yaml",
	}
	return template_list
}

func (r *RHSSO) GetProductName() integreatlyv1alpha1.ProductName {
	return integreatlyv1alpha1.ProductRHSSO
}

func (r *RHSSO) GetProductVersion() integreatlyv1alpha1.ProductVersion {
	return integreatlyv1alpha1.ProductVersion(r.Config["VERSION"])
}

func (r *RHSSO) GetOperatorVersion() integreatlyv1alpha1.OperatorVersion {
	return integreatlyv1alpha1.OperatorVersionRHSSO
}

func (r *RHSSO) SetProductVersion(version string) {
	r.Config["VERSION"] = version
}

func (r *RHSSO) SetOperatorVersion(operator string) {
	r.Config["OPERATOR"] = operator
}

func (r *RHSSO) Validate() error {
	if r.GetProductName() == "" {
		return errors.New("config product name is not defined")
	}
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
