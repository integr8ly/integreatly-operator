package config

import "time"

// A little bit arbitary but it is to work around an issue that subscriptions can take a long time to be picked up and an install plan created
// deleting and recreating the subscription can speed up this process
var SubscriptionTimeout = time.Minute * 5

const (
	// SMTPSecretLabelKey is a label key to match on the SMTP Secret
	SMTPSecretLabelKey = "owner"
	// SMTPSecretLabelValue is the value of the SMTPSecretLabelKey to match on the SMTP Secret
	SMTPSecretLabelValue = "integreatly"
)
