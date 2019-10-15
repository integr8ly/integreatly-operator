package config

import "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"

type MobileSecurityService struct {
	config ProductConfig
}

func NewMobileSecurityService(config ProductConfig) *MobileSecurityService {
	return &MobileSecurityService{config: config}
}

func (a *MobileSecurityService) GetHost() string {
	return a.config["HOST"]
}

func (a *MobileSecurityService) SetHost(newHost string) {
	a.config["HOST"] = newHost
}

func (a *MobileSecurityService) GetNamespace() string {
	return a.config["NAMESPACE"]
}

func (a *MobileSecurityService) SetNamespace(newNamespace string) {
	a.config["NAMESPACE"] = newNamespace
}

func (a *MobileSecurityService) Read() ProductConfig {
	return a.config
}

func (a *MobileSecurityService) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductMobileSecurityService
}

func (c *MobileSecurityService) GetProductVersion() v1alpha1.ProductVersion {
	return v1alpha1.VersionMobileSecurityService
}
