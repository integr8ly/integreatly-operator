package models


type ConfigService struct{
	MobileSecurityServiceURL string `json:"mobile-security-server-url"`
}

func NewConfigService(url string) *ConfigService {
	service := new (ConfigService)
	service.MobileSecurityServiceURL = url
	return service
}