package models

import (
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/aerogear/mobile-security-service-operator/pkg/utils"
)

const (
	ID = "security"
	Name = "security"
	Type = "security"
)

type SDKConfigService struct{
	ID					  string     `json:"id"`
	Name                  string     `json:"name"`
	Type                  string     `json:"type"`
	URL             	  string     `json:"url"`
	ConfigService         ConfigService `json:"config,omitempty"`
}

func NewSDKConfigServices(m *mobilesecurityservicev1alpha1.MobileSecurityServiceApp) *SDKConfigService {
	service := new(SDKConfigService)
	service.ID = ID
	service.Name = Name
	service.Type = Type
	service.URL = utils.GetAppIngressURL(m.Spec.Protocol, m.Spec.ClusterHost, m.Spec.HostSufix)
	service.ConfigService = *NewConfigService(service.URL)
	return service
}





