package common

import (
	"crypto/tls"
	"net/smtp"

	"github.com/integr8ly/integreatly-operator/test/resources"
)

func TestSendgridCredentialsAreValid(t TestingTB, ctx *TestingContext) {

	// Get SMTP secret from rhmi-operator namespace
	kc := ctx.KubeClient
	smtpSecret, err := resources.GetSMTPSecret(kc, RHMIOperatorNamespace, SMTPSecretName)
	if err != nil {
		t.Fatal("Failed to get an SMTP secret", err)
	}

	username, password, host, port :=
		string(smtpSecret["username"]),
		string(smtpSecret["password"]),
		string(smtpSecret["host"]),
		string(smtpSecret["port"])

	if host != "smtp.sendgrid.net" {
		if host != "smtp.example.com" {
			t.Fatal("Dummy host values have changed. Expected: smtp.example.com, Actual: ", host)
		}
		if port != "587" {
			t.Fatal("Dummy port values have changed. Expected: 587, Actual: ", port)
		}
		if username != "" && username != "dummy" {
			t.Fatal("Dummy uesrname values have changed. Expected: dummy, Actual: ", username)
		}
		if password != "" && password != "dummy" {
			t.Fatal("Dummy password values have changed. Expected: dummy, Actual: %s", password)
		}

	} else {

		// Test if SMTP credentials are valid using smtp.Auth method
		auth := smtp.PlainAuth("", username, password, host)
		client, err := smtp.Dial(host + ":" + port)
		if err != nil {
			t.Fatal("Failed to create an SMTP client", err)
		}
		err = client.StartTLS(&tls.Config{ServerName: host})
		if err != nil {
			t.Fatal("Failed to encrypt the communication between an SMTP client and a server", err)
		}
		err = client.Auth(auth)
		if err != nil {
			t.Fatal("Failed to authenticate an SMTP client with provided credentials", err)
		}
	}
}
