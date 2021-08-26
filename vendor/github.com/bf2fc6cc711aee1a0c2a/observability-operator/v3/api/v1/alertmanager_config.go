package v1

type AlertmanagerConfigGlobal struct {
	ResolveTimeout string `json:"resolve_timeout"`
}

type AlertmanagerConfigRoute struct {
	Receiver       string                    `json:"receiver,omitempty"`
	Match          map[string]string         `json:"match,omitempty"`
	RepeatInterval string                    `json:"repeat_interval,omitempty"`
	Routes         []AlertmanagerConfigRoute `json:"routes,omitempty"`
}

type PagerDutyConfig struct {
	ServiceKey string `json:"service_key"`
}

type WebhookConfig struct {
	Url string `json:"url"`
}

type AlertmanagerConfigReceiver struct {
	Name             string            `json:"name"`
	PagerDutyConfigs []PagerDutyConfig `json:"pagerduty_configs,omitempty"`
	WebhookConfigs   []WebhookConfig   `json:"webhook_configs,omitempty"`
}

type AlertmanagerConfigRoot struct {
	Global    *AlertmanagerConfigGlobal    `json:"global,omitempty"`
	Route     *AlertmanagerConfigRoute     `json:"route,omitempty"`
	Receivers []AlertmanagerConfigReceiver `json:"receivers,omitempty"`
}
