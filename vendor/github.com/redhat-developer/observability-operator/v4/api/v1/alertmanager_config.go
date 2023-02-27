package v1

type AlertmanagerConfigGlobal struct {
	ResolveTimeout   string `json:"resolve_timeout,omitempty"`
	SmtpSmartHost    string `json:"smtp_smarthost,omitempty"`
	SmtpFrom         string `json:"smtp_from,omitempty"`
	SmtpAuthUserName string `json:"smtp_auth_username,omitempty"`
	SmtpAuthPassword string `json:"smtp_auth_password,omitempty"`
	SmtpRequireTls   bool   `json:"smtp_require_tls,omitempty"`
}

type AlertmanagerConfigRoute struct {
	Receiver       string                    `json:"receiver,omitempty"`
	RepeatInterval string                    `json:"repeat_interval,omitempty"`
	Match          map[string]string         `json:"match,omitempty"`
	Routes         []AlertmanagerConfigRoute `json:"routes,omitempty"`
}

type EmailSubject struct {
	Subject string `json:"Subject,omitempty"`
}

type EmailConfig struct {
	SendResolved bool         `json:"send_resolved,omitempty"`
	To           string       `json:"to,omitempty"`
	EmailHeader  EmailSubject `json:"headers,omitempty"`
	Html         string       `json:"html,omitempty"`
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
	EmailConfig      []EmailConfig     `json:"email_configs,omitempty"`
}

type AlertmanagerConfigRoot struct {
	Global    *AlertmanagerConfigGlobal    `json:"global,omitempty"`
	Route     *AlertmanagerConfigRoute     `json:"route,omitempty"`
	Receivers []AlertmanagerConfigReceiver `json:"receivers,omitempty"`
}
