package config

import "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

type RHSSO struct {
	config ProductConfig
}

func newRHSSO(config ProductConfig) *RHSSO {
	return &RHSSO{config: config}
}

func (r *RHSSO) GetNamespace() string {
	return r.config["NAMESPACE"]
}

func (r *RHSSO) SetNamespace(newNamespace string) {
	r.config["NAMESPACE"] = newNamespace
}

func (r *RHSSO) GetRealm() string {
	return r.config["REALM"]
}

func (r *RHSSO) SetRealm(newRealm string) {
	r.config["REALM"] = newRealm
}

func (r *RHSSO) GetURL() string {
	return r.config["URL"]
}

func (r *RHSSO) SetURL(newURL string) {
	r.config["URL"] = newURL
}

func (r *RHSSO) Read() ProductConfig {
	return r.config
}

func (r *RHSSO) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductRHSSO
}
